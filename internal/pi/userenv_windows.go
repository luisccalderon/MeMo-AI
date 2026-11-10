//go:build windows

package pi

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// UserEnv reads a user-scope environment variable from the registry, so a
// value set after this process started is still seen. PowerShell is used
// instead of `reg query` because it returns UTF-8 and expands %VAR%
// references, which paths under a non-ASCII profile need.
func UserEnv(name string) (string, bool) {
	if !envName.MatchString(name) {
		return "", false
	}
	script := "[Console]::OutputEncoding=[Text.Encoding]::UTF8; " +
		"$v=[Environment]::GetEnvironmentVariable('" + name + "','User'); " +
		"if($v){[Console]::Out.Write([Environment]::ExpandEnvironmentVariables($v))}"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(string(out))
	return value, value != ""
}

// SetUserEnv persists a user-scope environment variable. Only processes
// started afterwards see it.
func SetUserEnv(name, value string) error {
	out, err := exec.Command("setx", name, value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("setx %s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
