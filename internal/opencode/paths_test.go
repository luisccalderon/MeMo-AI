package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestDiscoverUsesXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	isolateHome(t)

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "opencode")
	if p.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, want)
	}
	if p.AgentsDir != filepath.Join(want, "agents") {
		t.Fatalf("AgentsDir = %q", p.AgentsDir)
	}
	if p.ConfigFile != filepath.Join(want, "opencode.json") {
		t.Fatalf("ConfigFile = %q", p.ConfigFile)
	}
}

func TestDiscoverFallsBackToHomeConfig(t *testing.T) {
	home := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "opencode")
	if p.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, want)
	}
}

func TestDiscoverPrefersJSONCOverJSON(t *testing.T) {
	home := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(configDir, "opencode.json")
	jsoncPath := filepath.Join(configDir, "opencode.jsonc")
	if err := os.WriteFile(jsonPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsoncPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if p.ConfigFile != jsoncPath {
		t.Fatalf("ConfigFile = %q, want %q", p.ConfigFile, jsoncPath)
	}
}
