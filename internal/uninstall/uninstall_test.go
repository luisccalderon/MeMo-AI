package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
)

func testPaths(t *testing.T) omp.Paths {
	t.Helper()
	return omp.At(filepath.Join(t.TempDir(), "agent"))
}

func installOptions() install.Options {
	return install.Options{
		Lang:             "en",
		Orchestrator:     true,
		PrimaryModel:     "ollama/qwen3:8b",
		Subagents:        append([]string(nil), install.MemoSubagents...),
		EnableEngram:     true,
		EngramPath:       "/usr/bin/engram",
		InstallExtension: true,
		SetModel:         func(string) error { return nil },
	}
}

func seed(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRestoresMCPBackup(t *testing.T) {
	paths := testPaths(t)
	const original = "{\n  \"mcpServers\": {\n    \"github\": {\n      \"type\": \"http\",\n      \"url\": \"https://example.test/mcp\"\n    }\n  }\n}\n"
	seed(t, paths.MCPFile, original)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	res, err := Run(paths, Options{RestoreConfig: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Restored == "" {
		t.Fatal("expected restore path")
	}
	got, err := os.ReadFile(paths.MCPFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("mcp.json not restored:\n%s", got)
	}
	if omp.Exists(paths.StateFile) {
		t.Fatal("state file should be removed")
	}
	if omp.Exists(filepath.Join(paths.AgentsDir, "memo-writer.md")) {
		t.Fatal("agent file should be removed")
	}
}

func TestUninstallRemovesOnlyEngramWithoutRestore(t *testing.T) {
	paths := testPaths(t)
	seed(t, paths.MCPFile, `{"mcpServers":{"github":{"type":"http","url":"https://example.test/mcp"}}}`)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := config.Get(doc.Data, "mcpServers", "engram"); ok {
		t.Error("engram server should be removed")
	}
	if got := config.GetString(doc.Data, "mcpServers", "github", "url"); got == "" {
		t.Error("user server should be preserved")
	}
}

func TestUninstallLeavesNothingOnFreshDirectory(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{paths.MCPFile, paths.PromptFile, paths.AgentsDir, paths.SkillsDir, paths.CommandsDir, paths.ExtensionsDir, paths.StateFile} {
		if omp.Exists(gone) {
			t.Errorf("%s should be removed", gone)
		}
	}
}

func TestUninstallKeepsUserPromptText(t *testing.T) {
	paths := testPaths(t)
	const userPrompt = "# House rules\n\nAlways answer in haiku.\n"
	seed(t, paths.PromptFile, userPrompt)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(paths.PromptFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != userPrompt {
		t.Fatalf("prompt = %q, want %q", got, userPrompt)
	}
}

func TestUninstallLeavesRepointedEngramServer(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	seed(t, paths.MCPFile, `{"mcpServers":{"engram":{"type":"stdio","command":"/opt/my/engram","args":["mcp"]}}}`)
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := config.GetString(doc.Data, "mcpServers", "engram", "command"); got != "/opt/my/engram" {
		t.Fatalf("user-managed engram server was changed: %q", got)
	}
}

func TestUninstallSkipsModifiedFile(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(paths.AgentsDir, "memo-writer.md")
	seed(t, path, "my edits\n")
	res, err := Run(paths, Options{RestoreConfig: true})
	if err != nil {
		t.Fatal(err)
	}
	if !omp.Exists(path) {
		t.Fatal("modified file should be preserved")
	}
	if len(res.Skipped) == 0 {
		t.Fatal("expected a skipped file")
	}
}

func TestUninstallMentionsTheModelItLeaves(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	res, err := Run(paths, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(res.Warnings, "\n"), "modelRoles.default") {
		t.Fatalf("expected a note about the model: %v", res.Warnings)
	}
}

func TestUninstallNoInstallation(t *testing.T) {
	res, err := Run(testPaths(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("expected a warning")
	}
}
