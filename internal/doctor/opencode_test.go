package doctor

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
)

func TestOpenCodeCLICheckStatuses(t *testing.T) {
	cases := []struct {
		name     string
		launcher pi.OpenCodeLauncher
		status   Status
		hint     string
	}{
		{"starts from PATH", pi.OpenCodeLauncher{Status: pi.LauncherOK}, OK, ""},
		{"configured", pi.OpenCodeLauncher{Status: pi.LauncherOK, Configured: `C:\oc\opencode.exe`}, OK, ""},
		{"script only", pi.OpenCodeLauncher{Status: pi.LauncherNeedsBin, Candidate: `C:\oc\opencode.exe`}, Warn, `C:\oc\opencode.exe`},
		{"script only, no executable", pi.OpenCodeLauncher{Status: pi.LauncherNeedsBin}, Warn, "opencode.exe"},
		{"stale variable", pi.OpenCodeLauncher{Status: pi.LauncherBrokenBin, Configured: `C:\gone.exe`}, Warn, pi.OpenCodeBinEnv},
		{"not installed", pi.OpenCodeLauncher{Status: pi.LauncherMissing}, Warn, "opencode.ai"},
	}
	for _, tc := range cases {
		_, detail, status, hint := openCodeCLICheck(tc.launcher)
		if status != tc.status {
			t.Errorf("%s: status = %s (%s), want %s", tc.name, status, detail, tc.status)
		}
		if !strings.Contains(hint, tc.hint) {
			t.Errorf("%s: hint = %q, want it to mention %q", tc.name, hint, tc.hint)
		}
	}
}

func TestOpenCodeCLICheckOnlyWhenBridgeInstalled(t *testing.T) {
	paths := pi.At(filepath.Join(t.TempDir(), "agent"))
	for _, check := range Run(paths, detect.Environment{}, "en").Checks {
		if check.Name == "OpenCode CLI" {
			t.Fatal("OpenCode CLI must not be reported when opencode-pi is not installed")
		}
	}
}
