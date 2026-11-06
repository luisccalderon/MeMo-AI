// Package doctor performs read-only health checks for a MeMo-AI installation.
package doctor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/luis/memo-ai/internal/config"
	"github.com/luis/memo-ai/internal/detect"
	"github.com/luis/memo-ai/internal/opencode"
	"github.com/luis/memo-ai/internal/state"
)

// Status is the outcome of one check.
type Status string

// Check outcomes.
const (
	OK   Status = "ok"
	Warn Status = "warn"
	Fail Status = "fail"
	Skip Status = "skip"
)

// Check is one diagnostic result.
type Check struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
	Hint   string `json:"hint,omitempty"`
}

// Report is the full diagnostic result.
type Report struct {
	Checks []Check      `json:"checks"`
	State  *state.State `json:"state,omitempty"`
}

// Failed reports whether any check failed.
func (r Report) Failed() bool {
	for _, c := range r.Checks {
		if c.Status == Fail {
			return true
		}
	}
	return false
}

// Counts returns ok, warn, fail, skip totals.
func (r Report) Counts() (int, int, int, int) {
	var ok, warn, fail, skip int
	for _, c := range r.Checks {
		switch c.Status {
		case OK:
			ok++
		case Warn:
			warn++
		case Fail:
			fail++
		case Skip:
			skip++
		}
	}
	return ok, warn, fail, skip
}

// Run executes every check.
func Run(paths opencode.Paths, env detect.Environment, lang string) Report {
	st, stateErr := state.Load(paths.StateFile)
	report := Report{State: st}

	add := func(name string, status Status, detail, hint string) {
		report.Checks = append(report.Checks, Check{Name: name, Status: status, Detail: detail, Hint: hint})
	}

	// Installation state.
	switch {
	case stateErr != nil:
		add("State", Fail, fmt.Sprintf("cannot read %s: %v", paths.StateFile, stateErr), "Run `memo-ai install`.")
	case st.InstalledAt == "":
		add("State", Warn, "no installation recorded", "Run `memo-ai install`.")
	default:
		add("State", OK, fmt.Sprintf("installed %s", st.InstalledAt), "")
	}

	// OpenCode.
	if env.OpenCodePath != "" {
		add("OpenCode", OK, fmt.Sprintf("%s (%s)", env.OpenCodeVersion, env.OpenCodePath), "")
	} else {
		add("OpenCode", Fail, "binary not found on PATH", "Install OpenCode: https://opencode.ai")
	}

	// Configuration.
	doc, err := config.Load(paths.ConfigFile)
	switch {
	case err != nil:
		add("Config", Fail, err.Error(), "Fix the JSON/JSONC or restore a backup.")
	case !opencode.Exists(paths.ConfigFile):
		add("Config", Warn, "no configuration file yet", "Run `memo-ai install`.")
	default:
		add("Config", OK, paths.ConfigFile, "")
	}

	if doc != nil {
		primary := config.GetString(doc.Data, "default_agent")
		if primary == "" {
			add("Primary agent", Warn, "default_agent is not set", "Run `memo-ai configure`.")
		} else {
			add("Primary agent", OK, primary, "")
		}
		model := config.GetString(doc.Data, "model")
		if model == "" {
			add("Model", Warn, "no root model configured", "Choose one in /models or run `memo-ai configure`.")
		} else {
			add("Model", OK, model, "")
		}
	}

	// Files.
	agentFiles := []string{
		"memo-orchestrator.md", "memo-explorer.md", "memo-writer.md",
		"memo-reviewer.md", "memo-curator.md",
	}
	add("Agents", filesStatus(paths.AgentsDir, agentFiles), joinExisting(paths.AgentsDir, agentFiles), "Run `memo-ai install`.")

	skills := []string{"memo-memory", "memo-delegation", "memo-handoff", "memo-local-models"}
	var skillPaths []string
	for _, skill := range skills {
		skillPaths = append(skillPaths, filepath.Join(skill, "SKILL.md"))
	}
	add("Skills", filesStatus(paths.SkillsDir, skillPaths), joinExisting(paths.SkillsDir, skillPaths), "Run `memo-ai install`.")

	commands := []string{"memo.md", "memo-recall.md", "memo-remember.md", "memo-handoff.md", "memo-review.md", "memo-agents.md"}
	add("Commands", filesStatus(paths.CommandsDir, commands), joinExisting(paths.CommandsDir, commands), "Run `memo-ai install`.")

	pluginFiles := []string{"index.ts", "tui.tsx", "package.json"}
	if st.Plugin {
		add("TUI plugin", filesStatus(paths.PluginDir, pluginFiles), paths.PluginDir, "Run `memo-ai install --plugin`.")
	} else {
		add("TUI plugin", Skip, "disabled in state", "Run `memo-ai configure` to enable it.")
	}

	// Engram.
	if env.EngramPath != "" {
		add("Engram", OK, fmt.Sprintf("%s (%s)", env.EngramVersion, env.EngramPath), "")
	} else if st.EngramMCP {
		add("Engram", Fail, "MCP configured but binary not found", "Install it with `memo-ai engram install`.")
	} else {
		add("Engram", Warn, "binary not found", "Optional: `memo-ai engram install`.")
	}

	if doc != nil {
		if st.EngramMCP {
			command, ok := config.Get(doc.Data, "mcp", "servers", "engram", "command")
			if !ok {
				add("Engram MCP", Fail, "missing mcp.servers.engram", "Run `memo-ai configure`.")
			} else {
				add("Engram MCP", OK, fmt.Sprint(command), "")
			}
		} else {
			add("Engram MCP", Skip, "not configured", "")
		}
	}

	// Local runtime.
	if len(env.Models) > 0 {
		var refs []string
		seen := map[string]bool{}
		for _, model := range env.Models {
			if !seen[model.Ref()] {
				refs = append(refs, model.Ref())
				seen[model.Ref()] = true
			}
		}
		sort.Strings(refs)
		add("Local models", OK, strings.Join(refs, ", "), "")
	} else {
		add("Local models", Warn, "no local runtime answered", "Optional: start Ollama, LM Studio or vLLM.")
	}

	// Recorded files.
	if len(st.Files) > 0 {
		missing, modified := 0, 0
		for _, record := range st.Files {
			hash, err := state.HashFile(record.Path)
			if err != nil {
				missing++
				continue
			}
			if hash != record.SHA256 {
				modified++
			}
		}
		switch {
		case missing == 0 && modified == 0:
			add("Installed files", OK, fmt.Sprintf("%d files verified", len(st.Files)), "")
		case missing > 0:
			add("Installed files", Fail, fmt.Sprintf("%d missing, %d modified", missing, modified), "Run `memo-ai install`.")
		default:
			add("Installed files", Warn, fmt.Sprintf("%d modified by the user", modified), "Reinstall only if you want to overwrite them.")
		}
	}

	// Backup.
	if st.Config.Backup != "" {
		if opencode.Exists(st.Config.Backup) {
			add("Config backup", OK, st.Config.Backup, "")
		} else {
			add("Config backup", Warn, "backup missing: "+st.Config.Backup, "")
		}
	}

	// Subagent permissions.
	if doc != nil {
		primary := st.PrimaryAgent
		if primary == "" {
			primary = "memo-orchestrator"
		}
		perms, _ := config.Get(doc.Data, "agents", primary, "permissions")
		allowed := allowedSubagents(perms)
		switch {
		case len(st.Subagents) == 0:
			if allowed["*"] == "deny" {
				add("Subagent policy", OK, "disabled (deny *)", "")
			} else {
				add("Subagent policy", Warn, "state says disabled but permission deny is missing", "Run `memo-ai configure`.")
			}
		default:
			var missing []string
			for _, name := range st.Subagents {
				if allowed[name] != "allow" {
					missing = append(missing, name)
				}
			}
			if len(missing) == 0 {
				add("Subagent policy", OK, strings.Join(st.Subagents, ", "), "")
			} else {
				add("Subagent policy", Warn, "missing allow rule: "+strings.Join(missing, ", "), "Run `memo-ai configure`.")
			}
		}
	}

	_ = lang
	return report
}

func filesStatus(root string, files []string) Status {
	missing := 0
	for _, file := range files {
		if !opencode.Exists(filepath.Join(root, file)) {
			missing++
		}
	}
	switch {
	case missing == 0:
		return OK
	case missing == len(files):
		return Fail
	default:
		return Warn
	}
}

func joinExisting(root string, files []string) string {
	var present []string
	for _, file := range files {
		if opencode.Exists(filepath.Join(root, file)) {
			present = append(present, file)
		}
	}
	if len(present) == 0 {
		return "none found"
	}
	return fmt.Sprintf("%d/%d present", len(present), len(files))
}

func allowedSubagents(value any) map[string]string {
	out := map[string]string{}
	rules, ok := value.([]any)
	if !ok {
		return out
	}
	for _, rule := range rules {
		m, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		action, _ := m["action"].(string)
		resource, _ := m["resource"].(string)
		effect, _ := m["effect"].(string)
		if action == "subagent" {
			out[resource] = effect
		}
	}
	return out
}
