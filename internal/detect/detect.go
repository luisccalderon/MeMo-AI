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

	"github.com/luisccalderon/MeMo-AI/internal/pi"
)

// LocalModel is a model served by a local OpenAI-compatible runtime.
type LocalModel struct {
	Provider string
	ID       string
	Name     string
	Detail   string
	Endpoint string
}

// Ref returns the provider/model reference used by Pi.
func (m LocalModel) Ref() string {
	return m.Provider + "/" + m.ID
}

// Environment is the result of detection.
type Environment struct {
	PiPath        string
	PiVersion     string
	EngramPath    string
	EngramVersion string
	Models        []LocalModel
}

// Detect inspects the machine with a short, bounded timeout.
func Detect(ctx context.Context) Environment {
	env := Environment{}
	if path := FindPi(); path != "" {
		env.PiPath = path
		env.PiVersion = commandVersion(ctx, path, "--version")
		env.PiVersion = strings.TrimPrefix(env.PiVersion, "pi ")
	}
	env.EngramPath, env.EngramVersion = FindEngram(ctx)
	env.Models = LocalModels(ctx)
	return env
}

// FindPi locates the Pi binary. It checks PI_BIN, then PATH, then a managed
// install (the pi.dev installer publishes its launcher inside the agent
// directory), then the other common install locations that are not always
// present in a non-interactive shell's PATH.
//
// A managed install keeps its launchers in `<agent dir>/bin` and does not add
// that directory to PATH, so PATH lookup alone reports Pi as missing.
func FindPi() string {
	if override := os.Getenv("PI_BIN"); override != "" {
		if isFile(override) {
			return override
		}
	}
	if path, err := exec.LookPath("pi"); err == nil {
		return path
	}

	if paths, err := pi.Discover(); err == nil {
		if path := managedEntrypoint(paths.ConfigDir); path != "" {
			return path
		}
		for _, name := range launcherNames() {
			if candidate := filepath.Join(paths.ConfigDir, "bin", name); isFile(candidate) {
				return candidate
			}
		}
	}

	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range launcherNames() {
			candidates = append(candidates,
				filepath.Join(home, ".pi", "bin", name),
				filepath.Join(home, ".local", "bin", name),
				filepath.Join(home, "bin", name),
			)
		}
	}
	candidates = append(candidates, "/usr/local/bin/pi", "/opt/homebrew/bin/pi")
	for _, path := range candidates {
		if isFile(path) {
			return path
		}
	}
	return ""
}

// launcherNames lists the Pi launcher file names for this platform, in the
// order a managed install prefers them.
func launcherNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"pi.exe", "pi.cmd", "pi.bat"}
	}
	return []string{"pi"}
}

// managedEntrypoint reads the entrypoint a managed Pi install recorded. The
// file is the installer's own source of truth, so it wins over guessing names.
func managedEntrypoint(configDir string) string {
	raw, err := os.ReadFile(filepath.Join(configDir, "install", "managed-install.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		Entrypoint struct {
			Path string `json:"path"`
		} `json:"entrypoint"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	if !isFile(doc.Entrypoint.Path) {
		return ""
	}
	return doc.Entrypoint.Path
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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

// DescribeModel formats a model for display.
func DescribeModel(m LocalModel) string {
	if m.Detail != "" {
		return fmt.Sprintf("%s (%s)", m.Ref(), m.Detail)
	}
	return m.Ref()
}
