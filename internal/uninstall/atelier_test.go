package uninstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
)

func atelierInstall(t *testing.T) install.Options {
	t.Helper()
	opts := installOptions()
	opts.AtelierConfig = true
	return opts
}

// A config the user supplied with --atelier-config is removed on uninstall
// only while it is unchanged.
func TestUninstallRemovesTheAtelierConfigItCopied(t *testing.T) {
	paths := testPaths(t)
	source := filepath.Join(t.TempDir(), "mine.json")
	if err := os.WriteFile(source, []byte(`{"sidebarWidth":64}`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := installOptions()
	opts.AtelierConfigFile = source
	if _, err := install.Run(paths, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.AtelierFile) {
		t.Fatal("pi-atelier.json copied by MeMo-AI must be removed")
	}
}

func TestUninstallRemovesTheAtelierConfigItWrote(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, atelierInstall(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	if pi.Exists(paths.AtelierFile) {
		t.Fatal("pi-atelier.json written by MeMo-AI must be removed")
	}
}

func TestUninstallLeavesAnEditedAtelierConfig(t *testing.T) {
	paths := testPaths(t)
	if _, err := install.Run(paths, atelierInstall(t)); err != nil {
		t.Fatal(err)
	}
	edited := `{"nerdFont":true}`
	if err := os.WriteFile(paths.AtelierFile, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(paths, Options{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(paths.AtelierFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != edited {
		t.Fatalf("pi-atelier.json = %q, want the user's edit kept", got)
	}
}
