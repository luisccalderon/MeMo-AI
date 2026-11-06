// Package opencode resolves OpenCode configuration locations.
package opencode

import (
	"os"
	"path/filepath"
	"runtime"
)

// Paths describes the files and directories owned or touched by MeMo-AI.
type Paths struct {
	ConfigDir   string
	ConfigFile  string
	AgentsDir   string
	SkillsDir   string
	CommandsDir string
	PluginsDir  string
	PluginDir   string
	MemoDir     string
	StateFile   string
	BackupDir   string
}

// Discover resolves the global OpenCode configuration directory.
func Discover() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	configDir := ""
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = filepath.Join(xdg, "opencode")
	} else if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			configDir = filepath.Join(appData, "opencode")
		}
	}
	if configDir == "" {
		configDir = filepath.Join(home, ".config", "opencode")
	}

	p := Paths{
		ConfigDir:   configDir,
		AgentsDir:   filepath.Join(configDir, "agents"),
		SkillsDir:   filepath.Join(configDir, "skills"),
		CommandsDir: filepath.Join(configDir, "commands"),
		PluginsDir:  filepath.Join(configDir, "plugins"),
		PluginDir:   filepath.Join(configDir, "plugins", "memo-ai"),
		MemoDir:     filepath.Join(configDir, "memo-ai"),
		StateFile:   filepath.Join(configDir, "memo-ai", "state.json"),
		BackupDir:   filepath.Join(configDir, "memo-ai", "backups"),
	}

	switch {
	case fileExists(filepath.Join(configDir, "opencode.jsonc")):
		p.ConfigFile = filepath.Join(configDir, "opencode.jsonc")
	case fileExists(filepath.Join(configDir, "opencode.json")):
		p.ConfigFile = filepath.Join(configDir, "opencode.json")
	default:
		p.ConfigFile = filepath.Join(configDir, "opencode.json")
	}
	return p, nil
}

// EnsureDirs creates the directories MeMo-AI writes to.
func (p Paths) EnsureDirs() error {
	dirs := []string{p.ConfigDir, p.AgentsDir, p.SkillsDir, p.CommandsDir, p.PluginsDir, p.MemoDir, p.BackupDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Exists reports whether a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
