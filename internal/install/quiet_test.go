package install

import (
	"os"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
)

func quietOptions() Options {
	opts := options()
	opts.QuietStartup = true
	return opts
}

func seedSettings(t *testing.T, paths pi.Paths, content string) {
	t.Helper()
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SettingsFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func quietValue(t *testing.T, paths pi.Paths) (any, bool) {
	t.Helper()
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	return config.Get(doc.Data, "quietStartup")
}

func TestQuietStartupIsWrittenPreservingTheRest(t *testing.T) {
	paths := testPaths(t)
	seedSettings(t, paths, `{"theme":"dark","packages":["npm:foo"]}`)

	res, err := Run(paths, quietOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := quietValue(t, paths); got != "header" {
		t.Fatalf("quietStartup = %v, want \"header\"", got)
	}
	doc, _ := config.Load(paths.SettingsFile)
	if got := config.GetString(doc.Data, "theme"); got != "dark" {
		t.Errorf("unrelated setting lost: theme = %q", got)
	}
	if !res.State.QuietStartup {
		t.Error("state must record that MeMo-AI applied the quiet startup")
	}
}

func TestQuietStartupLeavesAUserChoiceAlone(t *testing.T) {
	paths := testPaths(t)
	seedSettings(t, paths, `{"quietStartup":true}`)

	res, err := Run(paths, quietOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := quietValue(t, paths); got != true {
		t.Fatalf("quietStartup = %v, want the user's true", got)
	}
	if res.State.QuietStartup {
		t.Error("a value the user chose must not be recorded as applied by MeMo-AI")
	}
}

func TestQuietStartupIsNotWrittenUnlessRequested(t *testing.T) {
	paths := testPaths(t)
	seedSettings(t, paths, `{"theme":"dark"}`)

	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	if _, ok := quietValue(t, paths); ok {
		t.Fatal("quietStartup must not be written without the option")
	}
}

func TestQuietStartupSurvivesARerunWithoutTheOption(t *testing.T) {
	paths := testPaths(t)
	if _, err := Run(paths, quietOptions()); err != nil {
		t.Fatal(err)
	}
	res, err := Run(paths, options())
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := quietValue(t, paths); got != "header" {
		t.Fatalf("quietStartup = %v, want it kept", got)
	}
	if !res.State.QuietStartup {
		t.Error("state must keep the record so uninstall can undo it")
	}
}

func TestQuietStartupDryRunWritesNothing(t *testing.T) {
	paths := testPaths(t)
	opts := quietOptions()
	opts.DryRun = true
	if _, err := Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.SettingsFile) {
		t.Fatal("dry run created settings.json")
	}
}
