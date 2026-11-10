// Package omp resolves oh-my-pi (omp) configuration locations.
package omp

import (
	"os"
	"path/filepath"
)

// Paths describes the files and directories owned or touched by MeMo-AI.
type Paths struct {
	// ConfigDir is the omp agent directory, `~/.omp/agent` by default.
	ConfigDir string
	// MCPFile is the JSON file MeMo-AI merges the Engram server into.
	MCPFile string
	// SettingsFile is omp's own YAML settings. MeMo-AI never writes it
	// directly; model roles are set through `omp config set`.
	SettingsFile string
	// PromptFile is APPEND_SYSTEM.md, where the orchestrator block lives.
	PromptFile    string
	AgentsDir     string
	SkillsDir     string
	CommandsDir   string
	ExtensionsDir string
	ExtensionDir  string
	MemoDir       string
	StateFile     string
	BackupDir     string
}

// Discover resolves the global omp agent directory.
//
// Resolution mirrors omp: a named profile (OMP_PROFILE, or the legacy
// PI_PROFILE) lives under `<root>/profiles/<name>/agent` and ignores
// PI_CODING_AGENT_DIR; otherwise PI_CODING_AGENT_DIR wins, then
// `<root>/agent`. The root directory name is `.omp` unless PI_CONFIG_DIR
// overrides it.
func Discover() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	rootName := ".omp"
	if name := os.Getenv("PI_CONFIG_DIR"); name != "" {
		rootName = name
	}
	root := filepath.Join(home, rootName)

	profile := os.Getenv("OMP_PROFILE")
	if profile == "" {
		profile = os.Getenv("PI_PROFILE")
	}

	configDir := filepath.Join(root, "agent")
	switch {
	case profile != "":
		configDir = filepath.Join(root, "profiles", profile, "agent")
	case os.Getenv("PI_CODING_AGENT_DIR") != "":
		configDir = os.Getenv("PI_CODING_AGENT_DIR")
	}
	return At(configDir), nil
}

// At builds the path set for a given agent directory.
func At(configDir string) Paths {
	settings := filepath.Join(configDir, "config.yml")
	if alt := filepath.Join(configDir, "config.yaml"); !fileExists(settings) && fileExists(alt) {
		settings = alt
	}
	return Paths{
		ConfigDir:     configDir,
		MCPFile:       filepath.Join(configDir, "mcp.json"),
		SettingsFile:  settings,
		PromptFile:    filepath.Join(configDir, "APPEND_SYSTEM.md"),
		AgentsDir:     filepath.Join(configDir, "agents"),
		SkillsDir:     filepath.Join(configDir, "skills"),
		CommandsDir:   filepath.Join(configDir, "commands"),
		ExtensionsDir: filepath.Join(configDir, "extensions"),
		ExtensionDir:  filepath.Join(configDir, "extensions", "memo-ai"),
		MemoDir:       filepath.Join(configDir, "memo-ai"),
		StateFile:     filepath.Join(configDir, "memo-ai", "state.json"),
		BackupDir:     filepath.Join(configDir, "memo-ai", "backups"),
	}
}

// EnsureDirs creates the directories MeMo-AI always writes to. Optional
// directories (agents, extensions) are created on demand by the installer.
func (p Paths) EnsureDirs() error {
	dirs := []string{p.ConfigDir, p.SkillsDir, p.CommandsDir, p.MemoDir, p.BackupDir}
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
