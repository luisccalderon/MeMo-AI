package omp

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
	for _, key := range []string{"PI_CONFIG_DIR", "PI_CODING_AGENT_DIR", "OMP_PROFILE", "PI_PROFILE"} {
		t.Setenv(key, "")
	}
	return home
}

func TestDiscoverDefaultsToHomeOmpAgent(t *testing.T) {
	home := isolateHome(t)

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".omp", "agent")
	if p.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, want)
	}
	checks := map[string]string{
		p.MCPFile:      filepath.Join(want, "mcp.json"),
		p.SettingsFile: filepath.Join(want, "config.yml"),
		p.PromptFile:   filepath.Join(want, "APPEND_SYSTEM.md"),
		p.AgentsDir:    filepath.Join(want, "agents"),
		p.ExtensionDir: filepath.Join(want, "extensions", "memo-ai"),
		p.StateFile:    filepath.Join(want, "memo-ai", "state.json"),
	}
	for got, expected := range checks {
		if got != expected {
			t.Errorf("path = %q, want %q", got, expected)
		}
	}
}

func TestDiscoverHonorsAgentDirOverride(t *testing.T) {
	isolateHome(t)
	custom := filepath.Join(t.TempDir(), "custom-agent")
	t.Setenv("PI_CODING_AGENT_DIR", custom)

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	if p.ConfigDir != custom {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, custom)
	}
}

func TestDiscoverProfileIgnoresAgentDirOverride(t *testing.T) {
	home := isolateHome(t)
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, "elsewhere"))
	t.Setenv("PI_CONFIG_DIR", ".pi-custom")
	t.Setenv("OMP_PROFILE", "work")

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".pi-custom", "profiles", "work", "agent")
	if p.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, want)
	}
}

func TestAtPrefersExistingYamlSettings(t *testing.T) {
	dir := t.TempDir()
	alt := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(alt, []byte("theme:\n  dark: dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := At(dir).SettingsFile; got != alt {
		t.Fatalf("SettingsFile = %q, want %q", got, alt)
	}
}
