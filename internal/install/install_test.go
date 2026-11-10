package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/promptblock"
	"github.com/luisccalderon/MeMo-AI/internal/state"
)

func testPaths(t *testing.T) omp.Paths {
	t.Helper()
	return omp.At(filepath.Join(t.TempDir(), "agent"))
}

func options() Options {
	return Options{
		Lang:             "en",
		Orchestrator:     true,
		PrimaryModel:     "ollama/qwen3:8b",
		Subagents:        append([]string(nil), MemoSubagents...),
		EnableEngram:     true,
		EngramPath:       "/usr/bin/engram",
		InstallExtension: true,
		SetModel:         func(string) error { return nil },
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
	var applied string
	opts := options()
	opts.SetModel = func(model string) error { applied = model; return nil }

	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		filepath.Join(paths.AgentsDir, "memo-explorer.md"),
		filepath.Join(paths.AgentsDir, "memo-writer.md"),
		filepath.Join(paths.SkillsDir, "memo-memory", "SKILL.md"),
		filepath.Join(paths.CommandsDir, "memo.md"),
		filepath.Join(paths.CommandsDir, "memo-changes.md"),
		filepath.Join(paths.CommandsDir, "memo-tasks.md"),
		filepath.Join(paths.ExtensionDir, "index.ts"),
		paths.PromptFile,
		paths.MCPFile,
		paths.StateFile,
	} {
		if !omp.Exists(file) {
			t.Errorf("missing %s", file)
		}
	}
	// The orchestrator is a prompt block, never a task agent.
	if omp.Exists(filepath.Join(paths.AgentsDir, "memo-orchestrator.md")) {
		t.Error("orchestrator must not be installed as an agent file")
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
	args, _ := config.Get(doc.Data, "mcpServers", "engram", "args")
	if list, ok := args.([]any); !ok || len(list) != 2 || list[0] != "mcp" {
		t.Errorf("engram args = %#v", args)
	}

	if applied != "ollama/qwen3:8b" {
		t.Errorf("model setter received %q", applied)
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
	if !st.EngramMCP || !st.Extension || len(st.Files) < 15 {
		t.Errorf("unexpected state flags/files: %#v", st)
	}
}

func TestAgentFilesUseOmpFrontmatter(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	for _, name := range MemoSubagents {
		text := readFile(t, filepath.Join(paths.AgentsDir, name+".md"))
		head, _, _ := strings.Cut(strings.TrimPrefix(text, "---\n"), "\n---")
		for _, want := range []string{"name: " + name, "description: ", `spawns: ""`} {
			if !strings.Contains(head, want) {
				t.Errorf("%s frontmatter lacks %q:\n%s", name, want, head)
			}
		}
		for _, stale := range []string{"mode:", "permissions:", "steps:"} {
			if strings.Contains(head, stale) {
				t.Errorf("%s frontmatter still has OpenCode key %q", name, stale)
			}
		}
		if strings.Contains(head, "model:") {
			t.Errorf("%s should inherit its model when none is chosen", name)
		}
	}
}

func TestSubagentModelIsPinnedInFrontmatter(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.SubagentModel = "ollama/qwen3:8b"
	if _, err := Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	text := readFile(t, filepath.Join(paths.AgentsDir, "memo-writer.md"))
	head, body, _ := strings.Cut(strings.TrimPrefix(text, "---\n"), "\n---")
	if !strings.Contains(head, `model: "ollama/qwen3:8b"`) {
		t.Errorf("model not pinned:\n%s", head)
	}
	if !strings.Contains(body, "focused implementation worker") {
		t.Error("agent body was damaged")
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
	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.State.Files) != len(first.Files) || res.State.InstalledAt != first.InstalledAt {
		t.Fatalf("re-run changed the manifest: %d files, installed %s", len(res.State.Files), res.State.InstalledAt)
	}
	if firstPrompt != readFile(t, paths.PromptFile) || firstMCP != readFile(t, paths.MCPFile) {
		t.Fatal("prompt and mcp.json must be byte-stable across re-runs")
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
	opts.Subagents = []string{"memo-explorer"}
	opts.InstallExtension = false
	opts.EnableEngram = false
	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}

	if !omp.Exists(filepath.Join(paths.AgentsDir, "memo-explorer.md")) {
		t.Error("enabled subagent should stay")
	}
	for _, gone := range []string{
		filepath.Join(paths.AgentsDir, "memo-writer.md"),
		filepath.Join(paths.ExtensionDir, "index.ts"),
		paths.ExtensionDir,
		paths.PromptFile,
	} {
		if omp.Exists(gone) {
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
		if record.Kind == "extension" || strings.HasSuffix(record.Path, "memo-writer.md") {
			t.Errorf("stale file still recorded: %s", record.Path)
		}
	}
}

func TestDisabledSubagentKeepsUserEdits(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(paths.AgentsDir, "memo-writer.md")
	if err := os.WriteFile(edited, []byte("my edits\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := options()
	opts.Subagents = nil
	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	if readFile(t, edited) != "my edits\n" {
		t.Fatal("user-modified agent must be preserved")
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
	if omp.Exists(paths.MCPFile) {
		t.Fatal("mcp.json should not be created without Engram")
	}
}

func TestModelFailureIsAWarningWithTheManualCommand(t *testing.T) {
	for name, setter := range map[string]ModelSetter{
		"omp missing": nil,
		"omp failed":  func(string) error { return errors.New("boom") },
	} {
		t.Run(name, func(t *testing.T) {
			opts := options()
			opts.SetModel = setter
			res, err := Run(testPaths(t), opts)
			if err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(res.Warnings, "\n")
			if !strings.Contains(joined, "omp config set modelRoles.default ollama/qwen3:8b") {
				t.Fatalf("warning lacks the manual command: %q", joined)
			}
			if res.State.ModelApplied {
				t.Fatal("state must not claim the model was applied")
			}
		})
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	paths := testPaths(t)
	opts := options()
	opts.DryRun = true
	opts.SetModel = func(string) error {
		t.Error("dry run must not apply the model")
		return nil
	}
	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Writes) == 0 {
		t.Fatal("dry run should report planned writes")
	}
	if omp.Exists(paths.ConfigDir) {
		t.Fatal("dry run wrote files")
	}
}
