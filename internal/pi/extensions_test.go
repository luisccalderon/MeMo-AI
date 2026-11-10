package pi

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSettings(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return path
}

func TestInstalledExtensionsReadsPackages(t *testing.T) {
	path := writeSettings(t, `{"defaultProvider":"anthropic","packages":["npm:pi-free","npm:opencode-pi"]}`)

	got, err := InstalledExtensions(path)
	if err != nil {
		t.Fatalf("InstalledExtensions: %v", err)
	}
	if len(got) != 2 || got[0] != "npm:pi-free" || got[1] != "npm:opencode-pi" {
		t.Fatalf("packages = %v, want [npm:pi-free npm:opencode-pi]", got)
	}
}

func TestInstalledExtensionsMissingFileIsNotAnError(t *testing.T) {
	got, err := InstalledExtensions(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("InstalledExtensions: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("packages = %v, want none", got)
	}
}

func TestInstalledExtensionsReportsBrokenJSON(t *testing.T) {
	path := writeSettings(t, `{"packages":[`)

	if _, err := InstalledExtensions(path); err == nil {
		t.Fatal("expected an error for malformed settings.json")
	}
}

func TestHasExtension(t *testing.T) {
	path := writeSettings(t, `{"packages":["npm:pi-free"]}`)

	if !HasExtension(path, "npm:pi-free") {
		t.Fatal("expected npm:pi-free to be installed")
	}
	if HasExtension(path, "npm:opencode-pi") {
		t.Fatal("expected npm:opencode-pi to be missing")
	}
}

func TestHasExtensionWithoutSettings(t *testing.T) {
	if HasExtension(filepath.Join(t.TempDir(), "absent.json"), "npm:pi-free") {
		t.Fatal("expected a missing settings file to report nothing installed")
	}
}
