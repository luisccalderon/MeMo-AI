// Package detect discovers the local agent, model and memory environment.
package detect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/luis/memo-ai/internal/opencode"
)

// LocalModel is a model served by a local OpenAI-compatible runtime.
type LocalModel struct {
	Provider string
	ID       string
	Name     string
	Detail   string
	Endpoint string
}

// Ref returns the provider/model reference used by OpenCode.
func (m LocalModel) Ref() string {
	return m.Provider + "/" + m.ID
}

// Agent is a discovered OpenCode agent.
type Agent struct {
	ID          string
	Name        string
	Description string
	Mode        string
	Model       string
	Path        string
	Builtin     bool
}

// Environment is the result of detection.
type Environment struct {
	OpenCodePath    string
	OpenCodeVersion string
	EngramPath      string
	EngramVersion   string
	Models          []LocalModel
	Agents          []Agent
}

// Detect inspects the machine with a short, bounded timeout.
func Detect(ctx context.Context) Environment {
	env := Environment{}
	if path := FindOpenCode(); path != "" {
		env.OpenCodePath = path
		env.OpenCodeVersion = commandVersion(ctx, path, "--version")
		env.OpenCodeVersion = strings.TrimPrefix(env.OpenCodeVersion, "opencode ")
	}
	env.EngramPath, env.EngramVersion = FindEngram(ctx)
	env.Models = LocalModels(ctx)
	return env
}

// FindOpenCode locates the OpenCode binary. It checks PATH first, then the
// common install locations that are not always present in a non-interactive
// shell's PATH.
func FindOpenCode() string {
	if override := os.Getenv("OPENCODE_BIN"); override != "" {
		if info, err := os.Stat(override); err == nil && !info.IsDir() {
			return override
		}
	}
	if path, err := exec.LookPath("opencode"); err == nil {
		return path
	}
	name := "opencode"
	if runtime.GOOS == "windows" {
		name = "opencode.exe"
	}
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".opencode", "bin", name),
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, "bin", name),
		)
	}
	candidates = append(candidates, "/usr/local/bin/opencode", "/opt/homebrew/bin/opencode")
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func commandVersion(ctx context.Context, path string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line)
}

// FindEngram locates the Engram binary and reads its version.
func FindEngram(ctx context.Context) (string, string) {
	var candidates []string
	if path, err := exec.LookPath("engram"); err == nil {
		candidates = append(candidates, path)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "engram"),
			filepath.Join(home, "go", "bin", "engram"),
			filepath.Join(home, "bin", "engram"),
		)
	}
	candidates = append(candidates, "/usr/local/bin/engram", "/opt/homebrew/bin/engram")
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			v := commandVersion(ctx, path, "--version")
			if v == "" {
				v = commandVersion(ctx, path, "version")
			}
			return path, strings.TrimSpace(strings.TrimPrefix(v, "engram "))
		}
	}
	return "", ""
}

type endpoint struct {
	provider string
	url      string
	kind     string
}

// LocalModels probes Ollama, LM Studio and vLLM concurrently.
func LocalModels(ctx context.Context) []LocalModel {
	endpoints := []endpoint{
		{provider: "ollama", url: envOr("OLLAMA_HOST", "http://127.0.0.1:11434") + "/api/tags", kind: "ollama"},
		{provider: "lmstudio", url: envOr("LMSTUDIO_HOST", "http://127.0.0.1:1234") + "/api/v1/models", kind: "openai"},
		{provider: "vllm", url: envOr("VLLM_HOST", "http://127.0.0.1:8000") + "/v1/models", kind: "openai"},
	}

	client := &http.Client{Timeout: 1500 * time.Millisecond}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []LocalModel
	for _, ep := range endpoints {
		wg.Add(1)
		go func(ep endpoint) {
			defer wg.Done()
			models := probe(ctx, client, ep)
			mu.Lock()
			out = append(out, models...)
			mu.Unlock()
		}(ep)
	}
	wg.Wait()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Provider == out[j].Provider {
			return out[i].ID < out[j].ID
		}
		return out[i].Provider < out[j].Provider
	})
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fallback
}

func probe(ctx context.Context, client *http.Client, ep endpoint) []LocalModel {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.url, nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	if ep.kind == "ollama" {
		var payload struct {
			Models []struct {
				Name    string `json:"name"`
				Model   string `json:"model"`
				Details struct {
					ParameterSize string `json:"parameter_size"`
					Family        string `json:"family"`
				} `json:"details"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil
		}
		out := make([]LocalModel, 0, len(payload.Models))
		for _, m := range payload.Models {
			id := m.Model
			if id == "" {
				id = m.Name
			}
			name := m.Name
			if name == "" {
				name = id
			}
			detail := strings.TrimSpace(strings.Join(nonEmpty(m.Details.ParameterSize, m.Details.Family), " · "))
			out = append(out, LocalModel{Provider: ep.provider, ID: id, Name: name, Detail: detail, Endpoint: ep.url})
		}
		return out
	}

	var payload struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Models []struct {
			Key         string `json:"key"`
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Name        string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}
	var out []LocalModel
	for _, m := range payload.Data {
		if m.ID == "" {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.ID
		}
		out = append(out, LocalModel{Provider: ep.provider, ID: m.ID, Name: name, Endpoint: ep.url})
	}
	for _, m := range payload.Models {
		id := m.Key
		if id == "" {
			id = m.ID
		}
		if id == "" {
			continue
		}
		name := m.DisplayName
		if name == "" {
			name = m.Name
		}
		if name == "" {
			name = id
		}
		out = append(out, LocalModel{Provider: ep.provider, ID: id, Name: name, Endpoint: ep.url})
	}
	return out
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

// Agents returns built-in and user-defined OpenCode agents.
func Agents(paths opencode.Paths) []Agent {
	agents := []Agent{
		{ID: "build", Name: "Build", Description: "Default coding agent", Mode: "primary", Builtin: true},
		{ID: "plan", Name: "Plan", Description: "Explores and plans without editing", Mode: "primary", Builtin: true},
		{ID: "general", Name: "General", Description: "General read/write subagent", Mode: "subagent", Builtin: true},
		{ID: "explore", Name: "Explore", Description: "Read-only code explorer", Mode: "subagent", Builtin: true},
	}
	seen := map[string]bool{"build": true, "plan": true, "general": true, "explore": true}
	_ = filepath.WalkDir(paths.AgentsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		agent := parseAgentFile(paths.AgentsDir, path)
		if agent.ID == "" || seen[agent.ID] {
			return nil
		}
		seen[agent.ID] = true
		agents = append(agents, agent)
		return nil
	})
	sort.SliceStable(agents, func(i, j int) bool {
		if agents[i].Builtin != agents[j].Builtin {
			return !agents[i].Builtin
		}
		return agents[i].ID < agents[j].ID
	})
	return agents
}

func parseAgentFile(root, path string) Agent {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Agent{}
	}
	id := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
	agent := Agent{ID: id, Name: id, Mode: "primary", Path: path}

	raw, err := os.ReadFile(path)
	if err != nil {
		return agent
	}
	text := string(raw)
	if !strings.HasPrefix(strings.TrimSpace(text), "---") {
		return agent
	}
	lines := strings.Split(text, "\n")
	inFront := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFront {
				break
			}
			inFront = true
			continue
		}
		if !inFront {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch key {
		case "description":
			agent.Description = value
		case "mode":
			agent.Mode = value
		case "model":
			agent.Model = value
		case "name":
			agent.Name = value
		}
	}
	return agent
}

// DescribeModel formats a model for display.
func DescribeModel(m LocalModel) string {
	if m.Detail != "" {
		return fmt.Sprintf("%s (%s)", m.Ref(), m.Detail)
	}
	return m.Ref()
}
