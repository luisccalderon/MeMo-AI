package pi

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(parts...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLauncherWindowsScriptOnlyNeedsBin(t *testing.T) {
	npm := t.TempDir()
	touch(t, npm, "opencode.cmd")
	touch(t, npm, "opencode") // extensionless shell shim: not startable without a shell
	exe := touch(t, npm, "node_modules", "@opencode", "cli", "bin", "opencode.exe")

	got := detectOpenCodeLauncher("windows", []string{"", npm}, t.TempDir(), "")
	if got.Status != LauncherNeedsBin {
		t.Fatalf("status = %v, want LauncherNeedsBin", got.Status)
	}
	if got.Candidate != exe {
		t.Fatalf("candidate = %q, want %q", got.Candidate, exe)
	}
}

func TestLauncherWindowsExeOnPathIsOK(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "opencode.exe")

	got := detectOpenCodeLauncher("windows", []string{dir}, t.TempDir(), "")
	if got.Status != LauncherOK {
		t.Fatalf("status = %v, want LauncherOK", got.Status)
	}
}

func TestLauncherWindowsStandaloneInstallIsCandidate(t *testing.T) {
	shims := t.TempDir()
	touch(t, shims, "opencode.ps1")
	home := t.TempDir()
	exe := touch(t, home, ".opencode", "bin", "opencode.exe")

	got := detectOpenCodeLauncher("windows", []string{shims}, home, "")
	if got.Status != LauncherNeedsBin || got.Candidate != exe {
		t.Fatalf("got %+v, want LauncherNeedsBin with candidate %q", got, exe)
	}
}

func TestLauncherWindowsScriptWithoutExecutable(t *testing.T) {
	shims := t.TempDir()
	touch(t, shims, "opencode.cmd")

	got := detectOpenCodeLauncher("windows", []string{shims}, t.TempDir(), "")
	if got.Status != LauncherNeedsBin || got.Candidate != "" {
		t.Fatalf("got %+v, want LauncherNeedsBin without a candidate", got)
	}
}

func TestLauncherMissing(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		got := detectOpenCodeLauncher(goos, []string{t.TempDir()}, t.TempDir(), "")
		if got.Status != LauncherMissing {
			t.Fatalf("%s: status = %v, want LauncherMissing", goos, got.Status)
		}
	}
}

func TestLauncherConfiguredBin(t *testing.T) {
	exe := touch(t, t.TempDir(), "opencode.exe")

	got := detectOpenCodeLauncher("windows", nil, t.TempDir(), exe)
	if got.Status != LauncherOK || got.Configured != exe {
		t.Fatalf("got %+v, want LauncherOK configured to %q", got, exe)
	}

	gone := filepath.Join(t.TempDir(), "absent.exe")
	got = detectOpenCodeLauncher("windows", nil, t.TempDir(), gone)
	if got.Status != LauncherBrokenBin {
		t.Fatalf("status = %v, want LauncherBrokenBin", got.Status)
	}
}

func TestLauncherUnixScriptIsOK(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "opencode")

	got := detectOpenCodeLauncher("linux", []string{dir}, t.TempDir(), "")
	if got.Status != LauncherOK {
		t.Fatalf("status = %v, want LauncherOK", got.Status)
	}
}

func TestLauncherWindowsLegacyNpmPackageIsCandidate(t *testing.T) {
	npm := t.TempDir()
	touch(t, npm, "opencode.cmd")
	exe := touch(t, npm, "node_modules", "opencode-ai", "bin", "opencode.exe")

	got := detectOpenCodeLauncher("windows", []string{npm}, t.TempDir(), "")
	if got.Status != LauncherNeedsBin || got.Candidate != exe {
		t.Fatalf("got %+v, want LauncherNeedsBin with candidate %q", got, exe)
	}
}
