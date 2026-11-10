package uninstall

import (
	"os"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/install"
)

func quietInstall(t *testing.T) (opts install.Options) {
	t.Helper()
	opts = installOptions()
	opts.QuietStartup = true
	return opts
}

func TestUninstallRemovesTheQuietStartupItApplied(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, quietInstall(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := config.Get(doc.Data, "quietStartup"); ok {
		t.Fatalf("quietStartup = %v, want it removed", value)
	}
	if got := config.GetString(doc.Data, "defaultModel"); got != "qwen3:8b" {
		t.Errorf("unrelated setting lost: defaultModel = %q", got)
	}
}

func TestUninstallLeavesAQuietStartupTheUserChanged(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, quietInstall(t)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SettingsFile, []byte(`{"quietStartup":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := config.Get(doc.Data, "quietStartup"); value != true {
		t.Fatalf("quietStartup = %v, want the user's true", value)
	}
}

func TestUninstallLeavesAQuietStartupItDidNotApply(t *testing.T) {
	paths := testPaths(t)
	seed(t, paths.SettingsFile, `{"quietStartup":"header"}`)
	if _, err := install.Run(paths, quietInstall(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := config.Get(doc.Data, "quietStartup"); value != "header" {
		t.Fatalf("quietStartup = %v, want the user's own \"header\" kept", value)
	}
}
