package pi

import (
	"path/filepath"
	"testing"
)

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PI_CODING_AGENT_DIR", "")
	return home
}

func TestDiscoverDefaultsToHomePiAgent(t *testing.T) {
	home := isolateHome(t)

	p, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".pi", "agent")
	if p.ConfigDir != want {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, want)
	}
	checks := map[string]string{
		p.MCPFile:       filepath.Join(want, "mcp.json"),
		p.SettingsFile:  filepath.Join(want, "settings.json"),
		p.PromptFile:    filepath.Join(want, "APPEND_SYSTEM.md"),
		p.PromptsDir:    filepath.Join(want, "prompts"),
		p.SkillsDir:     filepath.Join(want, "skills"),
		p.ExtensionsDir: filepath.Join(want, "extensions"),
		p.ExtensionDir:  filepath.Join(want, "extensions", "memo-ai"),
		p.StateFile:     filepath.Join(want, "memo-ai", "state.json"),
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
