package uninstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luis/memo-ai/internal/config"
	"github.com/luis/memo-ai/internal/install"
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

func installOptions() install.Options {
	return install.Options{
		Lang:          "en",
		PrimaryAgent:  "memo-orchestrator",
		PrimaryModel:  "ollama/qwen3:8b",
		Subagents:     append([]string(nil), install.MemoSubagents...),
		EnableEngram:  true,
		EngramPath:    "/usr/bin/engram",
		InstallPlugin: true,
	}
}

func TestUninstallRestoresConfigBackup(t *testing.T) {
	paths := testPaths(t)
	original := []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"model\": \"anthropic/claude-sonnet\"\n}\n")
	if err := os.WriteFile(paths.ConfigFile, original, 0o644); err != nil {
		t.Fatal(err)
	}
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
	got, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("config not restored:\n%s", got)
	}
	if opencode.Exists(paths.StateFile) {
		t.Fatal("state file should be removed")
	}
	if opencode.Exists(filepath.Join(paths.AgentsDir, "memo-orchestrator.md")) {
		t.Fatal("agent file should be removed")
	}
}

func TestUninstallCleansManagedConfigWithoutBackup(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"default_agent", "model", "agents", "mcp", "experimental"} {
		if _, ok := doc.Data[key]; ok {
			t.Errorf("managed key %q should be removed: %#v", key, doc.Data[key])
		}
	}
}

func TestUninstallSkipsModifiedFile(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(paths.AgentsDir, "memo-writer.md")
	if err := os.WriteFile(path, []byte("my edits\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(paths, Options{RestoreConfig: true})
	if err != nil {
		t.Fatal(err)
	}
	if !opencode.Exists(path) {
		t.Fatal("modified file should be preserved")
	}
	if len(res.Skipped) == 0 {
		t.Fatal("expected a skipped file")
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

func TestStateIsRemovedAfterUninstall(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, installOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{RestoreConfig: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := state.Load(paths.StateFile); err != nil {
		t.Fatal(err)
	}
	if opencode.Exists(paths.StateFile) {
		t.Fatal("state should be removed")
	}
}
