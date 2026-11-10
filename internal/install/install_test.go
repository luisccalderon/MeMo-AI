package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
	"github.com/luisccalderon/MeMo-AI/internal/promptblock"
	"github.com/luisccalderon/MeMo-AI/internal/state"
)

func testPaths(t *testing.T) pi.Paths {
	t.Helper()
	return pi.At(filepath.Join(t.TempDir(), "agent"))
}

func options() Options {
	return Options{
		Lang:             "en",
		Orchestrator:     true,
		PrimaryModel:     "ollama/qwen3:8b",
		EnableEngram:     true,
		EngramPath:       "/usr/bin/engram",
		InstallExtension: true,
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestInstallWritesStack(t *testing.T) {
	paths := testPaths(t)

	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		filepath.Join(paths.SkillsDir, "memo-memory", "SKILL.md"),
		filepath.Join(paths.PromptsDir, "memo.md"),
		filepath.Join(paths.PromptsDir, "memo-changes.md"),
		filepath.Join(paths.PromptsDir, "memo-tasks.md"),
		filepath.Join(paths.ExtensionDir, "index.ts"),
		paths.PromptFile,
		paths.MCPFile,
		paths.SettingsFile,
		paths.StateFile,
	} {
		if !pi.Exists(file) {
			t.Errorf("missing %s", file)
		}
	}

	prompt := readFile(t, paths.PromptFile)
	if !promptblock.Has(prompt) || !strings.Contains(prompt, "Operating loop (ODD)") {
		t.Errorf("orchestrator block missing from %s", paths.PromptFile)
	}

	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.GetString(doc.Data, "mcpServers", "engram", "type"); got != "stdio" {
		t.Errorf("engram type = %q", got)
	}
	if got := config.GetString(doc.Data, "mcpServers", "engram", "command"); got != "/usr/bin/engram" {
		t.Errorf("engram command = %q", got)
	}
	if got := config.GetString(doc.Data, "mcpServers", "engram", "exposure"); got != "direct" {
		t.Errorf("engram exposure = %q, want direct so Pi declares the memory tools to the model", got)
	}
	args, _ := config.Get(doc.Data, "mcpServers", "engram", "args")
	if list, ok := args.([]any); !ok || len(list) != 2 || list[0] != "mcp" {
		t.Errorf("engram args = %#v", args)
	}

	if len(res.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", res.Warnings)
	}

	st, err := state.Load(paths.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Orchestrator || st.PrimaryModel != "ollama/qwen3:8b" || !st.ModelApplied {
		t.Errorf("unexpected state: %#v", st)
	}
	if !st.EngramMCP || !st.Extension || st.Version != state.Version || len(st.Files) < 12 {
		t.Errorf("unexpected state flags/files: %#v", st)
	}
}

func TestSubagentsAreNeverInstalled(t *testing.T) {
	paths := testPaths(t)
	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if pi.Exists(filepath.Join(paths.ConfigDir, "agents")) {
		t.Error("Pi has no subagent support: no agents directory must be written")
	}
	for _, record := range res.State.Files {
		if record.Kind == "agent" || strings.Contains(filepath.ToSlash(record.Path), "/agents/") {
			t.Errorf("agent file recorded: %s", record.Path)
		}
		if strings.HasSuffix(record.Path, "memo-agents.md") {
			t.Errorf("the /memo-agents template needs subagents and must not ship: %s", record.Path)
		}
	}
}

func TestPromptTemplatesAreNotWrittenToCommands(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(filepath.Join(paths.ConfigDir, "commands")) {
		t.Error("Pi reads slash commands from prompts/, not commands/")
	}
}

func TestModelIsWrittenToSettingsPreservingTheRest(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const userSettings = `{"theme":"dark","defaultThinkingLevel":"high","packages":["npm:foo"]}`
	if err := os.WriteFile(paths.SettingsFile, []byte(userSettings), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.GetString(doc.Data, "defaultProvider"); got != "ollama" {
		t.Errorf("defaultProvider = %q", got)
	}
	if got := config.GetString(doc.Data, "defaultModel"); got != "qwen3:8b" {
		t.Errorf("defaultModel = %q", got)
	}
	if got := config.GetString(doc.Data, "theme"); got != "dark" {
		t.Errorf("unrelated setting lost: theme = %q", got)
	}
	if got := config.GetString(doc.Data, "defaultThinkingLevel"); got != "high" {
		t.Errorf("unrelated setting lost: defaultThinkingLevel = %q", got)
	}
	if _, ok := doc.Data["packages"]; !ok {
		t.Error("unrelated setting lost: packages")
	}
	backedUp := false
	for _, b := range res.Backups {
		if filepath.Base(b) == "settings.json" {
			backedUp = true
		}
	}
	if !backedUp {
		t.Errorf("existing settings.json should be snapshotted, backups: %v", res.Backups)
	}
}

func TestModelKeepsSlashesInTheModelID(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.PrimaryModel = "vllm/Qwen/Qwen3-Coder-30B-A3B-Instruct"
	if _, err := Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.GetString(doc.Data, "defaultProvider"); got != "vllm" {
		t.Errorf("defaultProvider = %q", got)
	}
	if got := config.GetString(doc.Data, "defaultModel"); got != "Qwen/Qwen3-Coder-30B-A3B-Instruct" {
		t.Errorf("defaultModel = %q", got)
	}
}

func TestSplitModel(t *testing.T) {
	for ref, want := range map[string][2]string{
		"ollama/qwen3:8b":      {"ollama", "qwen3:8b"},
		"openrouter/a/b":       {"openrouter", "a/b"},
		" lm-studio/model-x  ": {"lm-studio", "model-x"},
	} {
		provider, model, ok := SplitModel(ref)
		if !ok || provider != want[0] || model != want[1] {
			t.Errorf("SplitModel(%q) = %q, %q, %v", ref, provider, model, ok)
		}
	}
	for _, bad := range []string{"", "qwen3", "/model", "provider/", "a b/c"} {
		if _, _, ok := SplitModel(bad); ok {
			t.Errorf("SplitModel(%q) should fail", bad)
		}
	}
}

func TestUnreadableSettingsAreLeftAloneWithAWarning(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const broken = "{ not json"
	if err := os.WriteFile(paths.SettingsFile, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if readFile(t, paths.SettingsFile) != broken {
		t.Fatal("a settings.json that cannot be parsed must not be rewritten")
	}
	if !strings.Contains(strings.Join(res.Warnings, "\n"), "/model ollama/qwen3:8b") {
		t.Fatalf("warning lacks the manual command: %v", res.Warnings)
	}
	if res.State.ModelApplied {
		t.Fatal("state must not claim the model was applied")
	}
}

func TestInstallWithoutModelDoesNotCreateSettings(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.PrimaryModel = ""
	if _, err := Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.SettingsFile) {
		t.Fatal("settings.json must not be created when no model is chosen")
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	first, err := state.Load(paths.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	firstPrompt := readFile(t, paths.PromptFile)
	firstMCP := readFile(t, paths.MCPFile)
	firstSettings := readFile(t, paths.SettingsFile)
	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.State.Files) != len(first.Files) || res.State.InstalledAt != first.InstalledAt {
		t.Fatalf("re-run changed the manifest: %d files, installed %s", len(res.State.Files), res.State.InstalledAt)
	}
	if firstPrompt != readFile(t, paths.PromptFile) || firstMCP != readFile(t, paths.MCPFile) || firstSettings != readFile(t, paths.SettingsFile) {
		t.Fatal("prompt, mcp.json and settings.json must be byte-stable across re-runs")
	}
	if !res.State.ModelApplied {
		t.Fatal("an already-applied model must still be reported as applied")
	}
	backups, err := os.ReadDir(paths.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("a clean re-run should create no backups, got %d", len(backups))
	}
}

func TestInstallPreservesUserPromptAndServers(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const userPrompt = "# House rules\n\nAlways answer in haiku.\n"
	if err := os.WriteFile(paths.PromptFile, []byte(userPrompt), 0o644); err != nil {
		t.Fatal(err)
	}
	const userMCP = `{"mcpServers":{"github":{"type":"http","url":"https://example.test/mcp"}},"disabledServers":["x"]}`
	if err := os.WriteFile(paths.MCPFile, []byte(userMCP), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if prompt := readFile(t, paths.PromptFile); !strings.HasPrefix(prompt, userPrompt) || !promptblock.Has(prompt) {
		t.Errorf("user prompt text not preserved:\n%.200s", prompt)
	}
	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.GetString(doc.Data, "mcpServers", "github", "url"); got != "https://example.test/mcp" {
		t.Errorf("user MCP server lost: %q", got)
	}
	if _, ok := doc.Data["disabledServers"]; !ok {
		t.Error("unrelated mcp.json key lost")
	}
	if len(res.Backups) != 2 {
		t.Errorf("expected prompt and mcp.json backups, got %v", res.Backups)
	}
}

func TestReconfigureRemovesDisabledParts(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	opts := options()
	opts.Orchestrator = false
	opts.InstallExtension = false
	opts.EnableEngram = false
	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, gone := range []string{
		filepath.Join(paths.ExtensionDir, "index.ts"),
		paths.ExtensionDir,
		paths.PromptFile,
	} {
		if pi.Exists(gone) {
			t.Errorf("%s should be removed", gone)
		}
	}
	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := config.Get(doc.Data, "mcpServers", "engram"); ok {
		t.Error("engram server should be removed when Engram is disabled")
	}
	for _, record := range res.State.Files {
		if record.Kind == "extension" {
			t.Errorf("stale file still recorded: %s", record.Path)
		}
	}
}

func TestDisabledExtensionKeepsUserEdits(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(paths.ExtensionDir, "index.ts")
	if err := os.WriteFile(edited, []byte("my edits\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := options()
	opts.InstallExtension = false
	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	if readFile(t, edited) != "my edits\n" {
		t.Fatal("user-modified extension must be preserved")
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected a warning about the preserved file")
	}
}

func TestEngramDisabledDoesNotCreateMCPFile(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.EnableEngram = false
	if _, err := Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.MCPFile) {
		t.Fatal("mcp.json should not be created without Engram")
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
	if pi.Exists(paths.ConfigDir) {
		t.Fatal("dry run wrote files")
	}
}
