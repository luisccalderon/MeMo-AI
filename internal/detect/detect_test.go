package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// isolate hides any real Pi installation: HOME points at a temp dir, PATH is
// empty and the override variables are cleared.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PI_BIN", "")
	t.Setenv("PI_CODING_AGENT_DIR", "")
	t.Setenv("PATH", t.TempDir())
	return home
}

func touch(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func launcher(t *testing.T, home string) string {
	t.Helper()
	name := "pi"
	if runtime.GOOS == "windows" {
		name = "pi.cmd"
	}
	return touch(t, filepath.Join(home, ".pi", "agent", "bin", name))
}

func TestFindPiReadsManagedInstallEntrypoint(t *testing.T) {
	home := isolate(t)
	entrypoint := launcher(t, home)
	// A launcher the installer did not record must not win over the recorded one.
	touch(t, filepath.Join(home, ".pi", "agent", "bin", "pi.exe"))

	raw, err := json.Marshal(map[string]any{
		"kind":          "pi-managed-install",
		"schemaVersion": 1,
		"entrypoint":    map[string]string{"type": "script", "path": entrypoint},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(home, ".pi", "agent", "install", "managed-install.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if got := FindPi(); got != entrypoint {
		t.Fatalf("FindPi() = %q, want the managed entrypoint %q", got, entrypoint)
	}
}

func TestFindPiFallsBackToAgentBinLauncher(t *testing.T) {
	home := isolate(t)
	want := launcher(t, home)

	if got := FindPi(); got != want {
		t.Fatalf("FindPi() = %q, want the agent dir launcher %q", got, want)
	}
}

func TestFindPiHonorsPIBin(t *testing.T) {
	isolate(t)
	override := touch(t, filepath.Join(t.TempDir(), "custom-pi"))

	t.Setenv("PI_BIN", override)
	if got := FindPi(); got != override {
		t.Fatalf("FindPi() = %q, want PI_BIN %q", got, override)
	}
}

func TestFindPiReturnsEmptyWhenNothingIsInstalled(t *testing.T) {
	isolate(t)
	if got := FindPi(); got != "" {
		t.Fatalf("FindPi() = %q, want empty", got)
	}
}
