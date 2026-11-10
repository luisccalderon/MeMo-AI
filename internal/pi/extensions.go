package pi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// FreeModelExtension is a third-party Pi package that registers free model
// providers. MeMo-AI never edits Pi's settings.json for these: `pi install`
// owns the settings entry, so the packages stay upgradable through Pi itself.
type FreeModelExtension struct {
	Name   string
	Source string
}

// FreeModelExtensions lists the optional packages that unlock free models in Pi.
// Both are community packages, installed only when the user opts in.
var FreeModelExtensions = []FreeModelExtension{
	{Name: "pi-free", Source: "npm:pi-free"},
	{Name: "opencode-pi", Source: "npm:opencode-pi"},
}

// InstalledExtensions returns the package sources recorded in Pi's settings.json.
// A missing file is not an error: Pi creates it on first run.
func InstalledExtensions(settingsPath string) ([]string, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var settings struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse %s: %w", settingsPath, err)
	}
	return settings.Packages, nil
}

// HasExtension reports whether source is already listed in settings.json.
// An unreadable settings file counts as "not installed" so the caller can
// offer the install instead of failing.
func HasExtension(settingsPath, source string) bool {
	packages, err := InstalledExtensions(settingsPath)
	if err != nil {
		return false
	}
	for _, pkg := range packages {
		if pkg == source {
			return true
		}
	}
	return false
}

// InstallExtension runs `pi install <source>` with the detected Pi binary.
func InstallExtension(ctx context.Context, bin, source string, out io.Writer) error {
	if bin == "" {
		return errors.New("pi binary not found; install it from https://pi.dev")
	}
	cmd := exec.CommandContext(ctx, bin, "install", source)
	if out != nil {
		cmd.Stdout = out
		cmd.Stderr = out
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pi install %s: %w", source, err)
	}
	return nil
}
