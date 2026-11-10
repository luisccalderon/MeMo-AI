package pi

import (
	"os"
	"path/filepath"
	"runtime"
)

// OpenCodeBinEnv is the variable opencode-pi reads to locate the OpenCode CLI.
const OpenCodeBinEnv = "OPENCODE_PI_BIN"

// OpenCodePiSource is the package that bridges the OpenCode CLI into Pi.
const OpenCodePiSource = "npm:opencode-pi"

// LauncherStatus says whether opencode-pi can start the OpenCode CLI.
type LauncherStatus int

const (
	// LauncherOK means `opencode` starts as is, or OPENCODE_PI_BIN points to a file.
	LauncherOK LauncherStatus = iota
	// LauncherMissing means no OpenCode CLI was found at all.
	LauncherMissing
	// LauncherNeedsBin means only script launchers (.cmd/.ps1) are on PATH.
	// opencode-pi starts the CLI without a shell, which cannot run those on
	// Windows, so OPENCODE_PI_BIN has to point to the real executable.
	LauncherNeedsBin
	// LauncherBrokenBin means OPENCODE_PI_BIN is set but its target is not a file.
	LauncherBrokenBin
)

// OpenCodeLauncher is the result of inspecting how opencode-pi would start OpenCode.
type OpenCodeLauncher struct {
	Status LauncherStatus
	// Configured is the current OPENCODE_PI_BIN value, if any.
	Configured string
	// Candidate is the real executable OPENCODE_PI_BIN should point to. It is
	// empty when none was found.
	Candidate string
}

// DetectOpenCodeLauncher inspects the current machine. The user-scope value of
// OPENCODE_PI_BIN counts as configured even when this process started before
// it was set.
func DetectOpenCodeLauncher() OpenCodeLauncher {
	configured := os.Getenv(OpenCodeBinEnv)
	if configured == "" {
		configured, _ = UserEnv(OpenCodeBinEnv)
	}
	home, _ := os.UserHomeDir()
	return detectOpenCodeLauncher(runtime.GOOS, filepath.SplitList(os.Getenv("PATH")), home, configured)
}

func detectOpenCodeLauncher(goos string, pathDirs []string, home, configured string) OpenCodeLauncher {
	if configured != "" {
		status := LauncherOK
		if !isRegularFile(configured) {
			status = LauncherBrokenBin
		}
		return OpenCodeLauncher{Status: status, Configured: configured, Candidate: openCodeExecutable(goos, pathDirs, home)}
	}

	if goos != "windows" {
		// A shebang script is directly executable here, so any launcher works.
		for _, dir := range pathDirs {
			if dir != "" && isRegularFile(filepath.Join(dir, "opencode")) {
				return OpenCodeLauncher{Status: LauncherOK}
			}
		}
		return OpenCodeLauncher{Status: LauncherMissing}
	}

	// Without a shell, Windows only resolves a bare name to .com or .exe.
	for _, dir := range pathDirs {
		if dir == "" {
			continue
		}
		for _, name := range []string{"opencode.exe", "opencode.com"} {
			if isRegularFile(filepath.Join(dir, name)) {
				return OpenCodeLauncher{Status: LauncherOK}
			}
		}
	}

	candidate := openCodeExecutable(goos, pathDirs, home)
	if candidate == "" && !hasScriptLauncher(pathDirs) {
		return OpenCodeLauncher{Status: LauncherMissing}
	}
	return OpenCodeLauncher{Status: LauncherNeedsBin, Candidate: candidate}
}

// openCodeExecutable finds the real OpenCode binary behind the script
// launchers: the npm global layout next to each launcher on PATH, then the
// standalone installer location.
func openCodeExecutable(goos string, pathDirs []string, home string) string {
	if goos != "windows" {
		return ""
	}
	var candidates []string
	for _, dir := range pathDirs {
		if dir == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(dir, "node_modules", "@opencode", "cli", "bin", "opencode.exe"),
			filepath.Join(dir, "node_modules", "opencode-ai", "bin", "opencode.exe"),
		)
	}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".opencode", "bin", "opencode.exe"))
	}
	for _, candidate := range candidates {
		if isRegularFile(candidate) {
			return candidate
		}
	}
	return ""
}

func hasScriptLauncher(pathDirs []string) bool {
	for _, dir := range pathDirs {
		if dir == "" {
			continue
		}
		for _, name := range []string{"opencode.cmd", "opencode.bat", "opencode.ps1"} {
			if isRegularFile(filepath.Join(dir, name)) {
				return true
			}
		}
	}
	return false
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
