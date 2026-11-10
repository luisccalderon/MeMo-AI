// Package pi resolves Pi (pi-coding-agent) configuration locations.
package pi

import (
	"os"
	"path/filepath"
)

// Paths describes the files and directories owned or touched by MeMo-AI.
type Paths struct {
	// ConfigDir is the Pi agent directory, `~/.pi/agent` by default.
	ConfigDir string
	// MCPFile is the JSON file MeMo-AI merges the Engram server into.
	MCPFile string
	// SettingsFile is Pi's settings.json. MeMo-AI only touches the default
	// model keys in it.
	SettingsFile string
	// AtelierFile is pi-atelier's user-scope settings file. MeMo-AI writes a
	// preset there only when the user opts into the package and the file does
	// not exist yet.
	AtelierFile string
	// ModelsFile is Pi's models.json, where custom providers are declared.
	// MeMo-AI only reads it.
	ModelsFile string
	// PromptFile is APPEND_SYSTEM.md, where the orchestrator block lives.
	PromptFile string
	// PromptsDir holds Pi prompt templates, which Pi exposes as slash commands.
	PromptsDir    string
	SkillsDir     string
	ExtensionsDir string
	ExtensionDir  string
	MemoDir       string
	StateFile     string
	BackupDir     string
}

// Discover resolves the global Pi agent directory.
//
// Resolution mirrors Pi: PI_CODING_AGENT_DIR wins, otherwise `~/.pi/agent`.
func Discover() (Paths, error) {
	if dir := os.Getenv("PI_CODING_AGENT_DIR"); dir != "" {
		return At(dir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	return At(filepath.Join(home, ".pi", "agent")), nil
}

// At builds the path set for a given agent directory.
func At(configDir string) Paths {
	return Paths{
		ConfigDir:     configDir,
		MCPFile:       filepath.Join(configDir, "mcp.json"),
		SettingsFile:  filepath.Join(configDir, "settings.json"),
		AtelierFile:   filepath.Join(configDir, "pi-atelier.json"),
		ModelsFile:    filepath.Join(configDir, "models.json"),
		PromptFile:    filepath.Join(configDir, "APPEND_SYSTEM.md"),
		PromptsDir:    filepath.Join(configDir, "prompts"),
		SkillsDir:     filepath.Join(configDir, "skills"),
		ExtensionsDir: filepath.Join(configDir, "extensions"),
		ExtensionDir:  filepath.Join(configDir, "extensions", "memo-ai"),
		MemoDir:       filepath.Join(configDir, "memo-ai"),
		StateFile:     filepath.Join(configDir, "memo-ai", "state.json"),
		BackupDir:     filepath.Join(configDir, "memo-ai", "backups"),
	}
}

// EnsureDirs creates the directories MeMo-AI always writes to. Optional
// directories (extensions) are created on demand by the installer.
func (p Paths) EnsureDirs() error {
	dirs := []string{p.ConfigDir, p.SkillsDir, p.PromptsDir, p.MemoDir, p.BackupDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Exists reports whether a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
