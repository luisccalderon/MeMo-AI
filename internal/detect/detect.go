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

	"github.com/luisccalderon/MeMo-AI/internal/omp"
)

// LocalModel is a model served by a local OpenAI-compatible runtime.
type LocalModel struct {
	Provider string
	ID       string
	Name     string
	Detail   string
	Endpoint string
}

// Ref returns the provider/model reference used by omp.
func (m LocalModel) Ref() string {
	return m.Provider + "/" + m.ID
}

// Agent is a discovered omp task agent.
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
	OmpPath       string
	OmpVersion    string
	EngramPath    string
	EngramVersion string
	Models        []LocalModel
	Agents        []Agent
}

// Detect inspects the machine with a short, bounded timeout.
func Detect(ctx context.Context) Environment {
	env := Environment{}
	if path := FindOmp(); path != "" {
		env.OmpPath = path
		env.OmpVersion = commandVersion(ctx, path, "--version")
		env.OmpVersion = strings.TrimPrefix(env.OmpVersion, "omp ")
	}
	env.EngramPath, env.EngramVersion = FindEngram(ctx)
	env.Models = LocalModels(ctx)
	return env
}

// FindOmp locates the oh-my-pi binary. It checks OMP_BIN and PATH first, then
// the common install locations that are not always present in a
// non-interactive shell's PATH.
func FindOmp() string {
	if override := os.Getenv("OMP_BIN"); override != "" {
		if info, err := os.Stat(override); err == nil && !info.IsDir() {
			return override
		}
	}
	if path, err := exec.LookPath("omp"); err == nil {
		return path
	}
	name := "omp"
	if runtime.GOOS == "windows" {
		name = "omp.exe"
	}
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".bun", "bin", name),
			filepath.Join(home, ".omp", "bin", name),
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, "bin", name),
		)
	}
	candidates = append(candidates, "/usr/local/bin/omp", "/opt/homebrew/bin/omp")
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
		{provider: "lm-studio", url: envOr("LMSTUDIO_HOST", "http://127.0.0.1:1234") + "/api/v1/models", kind: "openai"},
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

// Agents returns omp's bundled task agents plus the user-defined ones.
func Agents(paths omp.Paths) []Agent {
	agents := []Agent{
		{ID: "task", Name: "task", Description: "General-purpose worker", Mode: "bundled", Builtin: true},
		{ID: "scout", Name: "scout", Description: "Fast read-only codebase research", Mode: "bundled", Builtin: true},
		{ID: "reviewer", Name: "reviewer", Description: "Code review specialist", Mode: "bundled", Builtin: true},
		{ID: "security-reviewer", Name: "security-reviewer", Description: "Security-focused review", Mode: "bundled", Builtin: true},
		{ID: "sonic", Name: "sonic", Description: "Fast worker for small tasks", Mode: "bundled", Builtin: true},
	}
	seen := map[string]bool{}
	for _, agent := range agents {
		seen[agent.ID] = true
	}
	entries, _ := os.ReadDir(paths.AgentsDir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		agent := parseAgentFile(paths.AgentsDir, filepath.Join(paths.AgentsDir, entry.Name()))
		if agent.ID == "" {
			continue
		}
		if seen[agent.ID] {
			// A user agent overrides the bundled one with the same name.
			for i := range agents {
				if agents[i].ID == agent.ID {
					agents[i] = agent
				}
			}
			continue
		}
		seen[agent.ID] = true
		agents = append(agents, agent)
	}
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
	agent := Agent{ID: id, Name: id, Mode: "user", Path: path}

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
		case "model":
			agent.Model = value
		case "name":
			// omp identifies an agent by its frontmatter name, not its file.
			if value != "" {
				agent.ID = value
				agent.Name = value
			}
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
