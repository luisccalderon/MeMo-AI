package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/assets"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
)

func atelierOptions() Options {
	opts := options()
	opts.AtelierConfig = true
	return opts
}

func seedFile(t *testing.T, paths pi.Paths, path, content string) {
	t.Helper()
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func atelierPreset(t *testing.T) []byte {
	t.Helper()
	raw, err := assets.Read(pi.AtelierConfigAsset)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAtelierConfigIsWrittenWhenMissing(t *testing.T) {
	paths := testPaths(t)

	res, err := Run(paths, atelierOptions())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(paths.AtelierFile)
	if err != nil {
		t.Fatalf("pi-atelier.json was not written: %v", err)
	}
	if !bytes.Equal(got, atelierPreset(t)) {
		t.Fatalf("pi-atelier.json = %q, want the shipped preset", got)
	}
	if !res.AtelierConfig {
		t.Error("the result must report the file so the installer can show it")
	}
	if res.State.AtelierConfig.Path != paths.AtelierFile || res.State.AtelierConfig.Kind != "atelier-preset" {
		t.Errorf("state must record the preset, got %#v", res.State.AtelierConfig)
	}
}

func TestAtelierConfigLeavesAnExistingFileAlone(t *testing.T) {
	paths := testPaths(t)
	seedFile(t, paths, paths.AtelierFile, `{"nerdFont":true,"sidebarWidth":60}`)

	res, err := Run(paths, atelierOptions())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(paths.AtelierFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"nerdFont":true,"sidebarWidth":60}`; string(got) != want {
		t.Fatalf("pi-atelier.json = %q, want the user's own settings kept", got)
	}
	if res.State.AtelierConfig.Path != "" {
		t.Error("a file MeMo-AI did not write must not be recorded as applied")
	}
}

func TestAtelierConfigIsNotWrittenUnlessRequested(t *testing.T) {
	paths := testPaths(t)

	if _, err := Run(paths, options()); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.AtelierFile) {
		t.Fatal("pi-atelier.json must not be written without the option")
	}
}

func TestAtelierConfigDryRunWritesNothing(t *testing.T) {
	paths := testPaths(t)
	opts := atelierOptions()
	opts.DryRun = true

	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.AtelierFile) {
		t.Fatal("dry run created pi-atelier.json")
	}
	if !containsWrite(res.Writes, paths.AtelierFile) {
		t.Fatalf("dry run must report the planned write, got %v", res.Writes)
	}
}

// The user's own file wins over the preset and is applied verbatim.
func TestAtelierConfigFileIsAppliedVerbatim(t *testing.T) {
	paths := testPaths(t)
	source := filepath.Join(t.TempDir(), "mine.json")
	mine := "{\n  \"nerdFont\": true,\n  \"preset\": \"minimal\",\n  \"sidebarWidth\": 52\n}\n"
	if err := os.WriteFile(source, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := options()
	opts.AtelierConfigFile = source

	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(paths.AtelierFile)
	if err != nil {
		t.Fatalf("pi-atelier.json was not written: %v", err)
	}
	if string(got) != mine {
		t.Fatalf("pi-atelier.json = %q, want the user's file verbatim", got)
	}
	if res.State.AtelierConfig.Kind != "atelier-config" {
		t.Errorf("kind = %q, want atelier-config", res.State.AtelierConfig.Kind)
	}
}

// A file that is not a JSON object would be written and then ignored by the
// package, so it is rejected with a warning instead.
func TestAtelierConfigFileRejectsNonObjectJSON(t *testing.T) {
	paths := testPaths(t)
	source := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(source, []byte(`["nerdFont"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := options()
	opts.AtelierConfigFile = source

	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.AtelierFile) {
		t.Fatal("an invalid config must not be written")
	}
	if len(res.Warnings) == 0 {
		t.Fatal("rejecting the file must warn the user")
	}
}

func TestAtelierConfigFileReplacesAndBacksUpAnExistingFile(t *testing.T) {
	paths := testPaths(t)
	seedFile(t, paths, paths.AtelierFile, `{"nerdFont":false}`)
	source := filepath.Join(t.TempDir(), "mine.json")
	if err := os.WriteFile(source, []byte(`{"sidebarWidth":64}`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := options()
	opts.AtelierConfigFile = source

	res, err := Run(paths, opts)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(paths.AtelierFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"sidebarWidth":64}` {
		t.Fatalf("pi-atelier.json = %q, want the user's file", got)
	}
	if len(res.Backups) == 0 {
		t.Fatal("replacing an existing file must leave a backup")
	}
}

func containsWrite(writes []string, want string) bool {
	for _, write := range writes {
		if write == want {
			return true
		}
	}
	return false
}
