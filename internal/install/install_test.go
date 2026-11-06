package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luis/memo-ai/internal/config"
	"github.com/luis/memo-ai/internal/opencode"
	"github.com/luis/memo-ai/internal/state"
)

func testPaths(t *testing.T) opencode.Paths {
	t.Helper()
	root := t.TempDir()
	return opencode.Paths{
		ConfigDir:   root,
		ConfigFile:  filepath.Join(root, "opencode.json"),
		AgentsDir:   filepath.Join(root, "agents"),
		SkillsDir:   filepath.Join(root, "skills"),
		CommandsDir: filepath.Join(root, "commands"),
		PluginsDir:  filepath.Join(root, "plugins"),
		PluginDir:   filepath.Join(root, "plugins", "memo-ai"),
		MemoDir:     filepath.Join(root, "memo-ai"),
		StateFile:   filepath.Join(root, "memo-ai", "state.json"),
		BackupDir:   filepath.Join(root, "memo-ai", "backups"),
	}
}

func options() Options {
	return Options{
		Lang:          "en",
		PrimaryAgent:  "memo-orchestrator",
		PrimaryModel:  "ollama/qwen3:8b",
		SubagentModel: "",
		Subagents:     append([]string(nil), MemoSubagents...),
		EnableEngram:  true,
		EngramPath:    "/usr/bin/engram",
		InstallPlugin: true,
	}
}

func TestInstallWritesStackAndConfig(t *testing.T) {
	paths := testPaths(t)
	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if res.ConfigPath != paths.ConfigFile {
		t.Fatalf("config path = %s", res.ConfigPath)
	}
	for _, file := range []string{
		filepath.Join(paths.AgentsDir, "memo-orchestrator.md"),
		filepath.Join(paths.SkillsDir, "memo-memory", "SKILL.md"),
		filepath.Join(paths.CommandsDir, "memo.md"),
		filepath.Join(paths.PluginDir, "tui.tsx"),
		paths.StateFile,
	} {
		if !opencode.Exists(file) {
			t.Errorf("missing %s", file)
		}
	}

	doc, err := config.Load(paths.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.GetString(doc.Data, "default_agent"); got != "memo-orchestrator" {
		t.Errorf("default_agent = %q", got)
	}
	if got := config.GetString(doc.Data, "model"); got != "ollama/qwen3:8b" {
		t.Errorf("model = %q", got)
	}
	if got := config.GetString(doc.Data, "mcp", "servers", "engram", "type"); got != "local" {
		t.Errorf("engram type = %q", got)
	}
	if got, ok := config.Get(doc.Data, "experimental", "subagent_depth"); !ok || got != float64(1) {
		t.Errorf("subagent_depth = %v, ok = %v", got, ok)
	}
	allowed := permissionEffects(t, doc, "memo-orchestrator")
	if allowed["*"] != "deny" {
		t.Errorf("wildcard subagent effect = %q", allowed["*"])
	}
	for _, name := range MemoSubagents {
		if allowed[name] != "allow" {
			t.Errorf("%s effect = %q", name, allowed[name])
		}
	}

	st, err := state.Load(paths.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	if st.PrimaryAgent != "memo-orchestrator" || st.PrimaryModel != "ollama/qwen3:8b" {
		t.Errorf("unexpected state: %#v", st)
	}
	if !st.EngramMCP || !st.Plugin || len(st.Files) < 15 {
		t.Errorf("unexpected state flags/files: %#v", st)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(paths.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(paths.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) == string(second) {
		t.Fatal("state timestamps should change on re-run")
	}
	backups, err := os.ReadDir(paths.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) > 1 {
		t.Fatalf("expected one backup directory, got %d", len(backups))
	}
}

func TestInstallDisablesSubagentsAndPlugin(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.Subagents = nil
	opts.InstallPlugin = false
	if _, err := Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	allowed := permissionEffects(t, doc, "memo-orchestrator")
	if allowed["*"] != "deny" {
		t.Errorf("wildcard effect = %q", allowed["*"])
	}
	for _, name := range MemoSubagents {
		if _, ok := allowed[name]; ok {
			t.Errorf("unexpected allow rule for %s", name)
		}
	}
	if opencode.Exists(filepath.Join(paths.PluginDir, "tui.tsx")) {
		t.Error("plugin should not be installed")
	}
	st, err := state.Load(paths.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range st.Files {
		if record.Kind == "plugin" {
			t.Errorf("plugin file still recorded: %s", record.Path)
		}
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.DryRun = true
	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Writes) == 0 {
		t.Fatal("dry run should report planned writes")
	}
	if opencode.Exists(paths.ConfigFile) || opencode.Exists(paths.StateFile) {
		t.Fatal("dry run wrote files")
	}
	if opencode.Exists(filepath.Join(paths.AgentsDir, "memo-orchestrator.md")) {
		t.Fatal("dry run wrote agent files")
	}
}

func permissionEffects(t *testing.T, doc *config.Document, agent string) map[string]string {
	t.Helper()
	value, ok := config.Get(doc.Data, "agents", agent, "permissions")
	if !ok {
		t.Fatalf("missing permissions for %s", agent)
	}
	rules, ok := value.([]any)
	if !ok {
		t.Fatalf("permissions for %s are not an array", agent)
	}
	out := map[string]string{}
	for _, rule := range rules {
		m, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		action, _ := m["action"].(string)
		if action != "subagent" {
			continue
		}
		resource, _ := m["resource"].(string)
		effect, _ := m["effect"].(string)
		out[resource] = effect
	}
	return out
}
