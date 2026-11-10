// Package doctor performs read-only health checks for a MeMo-AI installation.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
	"github.com/luisccalderon/MeMo-AI/internal/promptblock"
	"github.com/luisccalderon/MeMo-AI/internal/state"
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
func Run(paths pi.Paths, env detect.Environment, lang string) Report {
	st, stateErr := state.Load(paths.StateFile)
	report := Report{State: st}

	add := func(name string, status Status, detail, hint string) {
		if status == OK {
			hint = "" // a hint is a next step; a passing check has none
		}
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

	// Pi.
	if env.PiPath != "" {
		add("Pi", OK, fmt.Sprintf("%s (%s)", env.PiVersion, env.PiPath), "")
	} else {
		add("Pi", Fail, "pi binary not found on PATH", "Install it: https://pi.dev (or set PI_BIN).")
	}
	add("Agent directory", OK, paths.ConfigDir, "")

	// MCP configuration.
	doc, err := config.Load(paths.MCPFile)
	switch {
	case err != nil:
		add("MCP config", Fail, err.Error(), "Fix the JSON in mcp.json or restore a backup.")
	case !pi.Exists(paths.MCPFile):
		add("MCP config", Skip, "no mcp.json yet", "")
	default:
		add("MCP config", OK, paths.MCPFile, "")
	}

	// Orchestrator prompt.
	prompt, _ := os.ReadFile(paths.PromptFile)
	switch {
	case st.Orchestrator && promptblock.Has(string(prompt)):
		add("Orchestrator prompt", OK, paths.PromptFile, "")
	case st.Orchestrator:
		add("Orchestrator prompt", Fail, "managed block missing from "+paths.PromptFile, "Run `memo-ai install`.")
	case promptblock.Has(string(prompt)):
		add("Orchestrator prompt", Warn, "block present but disabled in state", "Run `memo-ai configure`.")
	default:
		add("Orchestrator prompt", Skip, "disabled in state", "Run `memo-ai configure` to enable it.")
	}

	if st.PrimaryModel != "" && st.ModelApplied {
		if provider := localProvider(st.PrimaryModel); provider != "" && !declaresProvider(paths.ModelsFile, provider) {
			add("Model", Warn, st.PrimaryModel+" is set, but models.json does not declare the "+provider+" provider",
				"Add it to "+paths.ModelsFile+" (see docs/local-models.md); Pi only lists models of declared providers.")
		} else {
			add("Model", OK, st.PrimaryModel+" (settings.json defaultProvider/defaultModel)", "")
		}
	} else if st.PrimaryModel != "" {
		add("Model", Warn, st.PrimaryModel+" was not applied to Pi", "Choose it with `/model "+st.PrimaryModel+"` or run `memo-ai configure --model "+st.PrimaryModel+"`.")
	} else {
		add("Model", Skip, "managed by Pi", "Choose one with /model or `memo-ai configure --model <provider/model>`.")
	}

	// Optional free model providers. Pi owns their settings entries, so the
	// check reads settings.json instead of MeMo-AI state.
	var missingFree []string
	for _, ext := range pi.FreeModelExtensions {
		if !pi.HasExtension(paths.SettingsFile, ext.Source) {
			missingFree = append(missingFree, ext.Source)
		}
	}
	switch len(missingFree) {
	case 0:
		add("Free models", OK, "pi-free and opencode-pi installed", "")
	case len(pi.FreeModelExtensions):
		add("Free models", Warn, "no free model provider installed", "Run `memo-ai install --free-models`.")
	default:
		add("Free models", Warn, "missing "+strings.Join(missingFree, ", "), "Run `memo-ai install --free-models`.")
	}

	// opencode-pi starts the OpenCode CLI without a shell, so a script-only
	// launcher on Windows lists the models but fails at the first request.
	if pi.HasExtension(paths.SettingsFile, pi.OpenCodePiSource) {
		name, detail, status, hint := openCodeCLICheck(pi.DetectOpenCodeLauncher())
		add(name, status, detail, hint)
	}

	// Files. Pi has no built-in subagents, so the orchestrator always works inline.
	add("Subagents", Skip, "not supported on Pi yet: the orchestrator works inline", "")

	skills := []string{"memo-memory", "memo-delegation", "memo-handoff", "memo-local-models"}
	var skillPaths []string
	for _, skill := range skills {
		skillPaths = append(skillPaths, filepath.Join(skill, "SKILL.md"))
	}
	add("Skills", filesStatus(paths.SkillsDir, skillPaths), joinExisting(paths.SkillsDir, skillPaths), "Run `memo-ai install`.")

	commands := []string{
		"memo.md", "memo-changes.md", "memo-tasks.md", "memo-recall.md", "memo-remember.md",
		"memo-handoff.md", "memo-review.md",
	}
	add("Commands", filesStatus(paths.PromptsDir, commands), joinExisting(paths.PromptsDir, commands), "Run `memo-ai install`.")

	if st.Extension {
		add("Extension", filesStatus(paths.ExtensionDir, []string{"index.ts"}), paths.ExtensionDir, "Run `memo-ai install`.")
	} else {
		add("Extension", Skip, "disabled in state", "Run `memo-ai configure` to enable it.")
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
			command := config.GetString(doc.Data, "mcpServers", "engram", "command")
			if command == "" {
				add("Engram MCP", Fail, "missing mcpServers.engram in mcp.json", "Run `memo-ai configure`.")
			} else {
				add("Engram MCP", OK, command, "")
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
		if pi.Exists(st.Config.Backup) {
			add("Config backup", OK, st.Config.Backup, "")
		} else {
			add("Config backup", Warn, "backup missing: "+st.Config.Backup, "")
		}
	}

	_ = lang
	return report
}

// openCodeCLICheck turns the launcher inspection into a doctor row.
func openCodeCLICheck(launcher pi.OpenCodeLauncher) (name, detail string, status Status, hint string) {
	name = "OpenCode CLI"
	fix := "Run `memo-ai install --free-models`"
	if launcher.Candidate != "" {
		fix += " or set " + pi.OpenCodeBinEnv + " to " + launcher.Candidate
	} else {
		fix = "Set " + pi.OpenCodeBinEnv + " to the full path of opencode.exe"
	}
	switch launcher.Status {
	case pi.LauncherOK:
		if launcher.Configured != "" {
			return name, pi.OpenCodeBinEnv + "=" + launcher.Configured, OK, ""
		}
		return name, "opencode starts from PATH", OK, ""
	case pi.LauncherMissing:
		return name, "opencode-pi is installed but the OpenCode CLI was not found", Warn,
			"Install it: https://opencode.ai (or set " + pi.OpenCodeBinEnv + ")."
	case pi.LauncherBrokenBin:
		return name, pi.OpenCodeBinEnv + " points to a missing file: " + launcher.Configured, Warn, fix + "."
	default:
		return name, "opencode is only a .cmd/.ps1 launcher on PATH; opencode-pi cannot start it", Warn, fix + "."
	}
}

// localProvider returns the provider of a model served by a local runtime that
// Pi does not ship a provider for, or "" for any other model. Hosted providers
// authenticate through /login or the environment, which doctor cannot see.
func localProvider(ref string) string {
	provider, _, _ := strings.Cut(ref, "/")
	switch provider {
	case "ollama", "lm-studio", "vllm":
		return provider
	}
	return ""
}

// declaresProvider reports whether models.json declares the provider.
func declaresProvider(modelsFile, provider string) bool {
	doc, err := config.Load(modelsFile)
	if err != nil {
		return false
	}
	_, ok := config.Get(doc.Data, "providers", provider)
	return ok
}

func filesStatus(root string, files []string) Status {
	missing := 0
	for _, file := range files {
		if !pi.Exists(filepath.Join(root, file)) {
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
		if pi.Exists(filepath.Join(root, file)) {
			present = append(present, file)
		}
	}
	if len(present) == 0 {
		return "none found"
	}
	return fmt.Sprintf("%d/%d present", len(present), len(files))
}
