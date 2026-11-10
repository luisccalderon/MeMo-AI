// Package state records what MeMo-AI installed so it can be updated or removed safely.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Version is the state schema version.
const Version = 2

// FileRecord is one file written by MeMo-AI.
type FileRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}

// ConfigRecord tracks a shared file MeMo-AI edits in place and its pre-install backup.
type ConfigRecord struct {
	Path    string `json:"path"`
	Backup  string `json:"backup,omitempty"`
	Existed bool   `json:"existed"`
}

// State is persisted at <config>/memo-ai/state.json.
type State struct {
	Version      int    `json:"version"`
	Language     string `json:"language"`
	InstalledAt  string `json:"installed_at"`
	UpdatedAt    string `json:"updated_at"`
	Orchestrator bool   `json:"orchestrator"`
	PrimaryModel string `json:"primary_model,omitempty"`
	// ModelApplied reports whether PrimaryModel reached the omp settings.
	ModelApplied  bool     `json:"model_applied"`
	Subagents     []string `json:"subagents"`
	SubagentModel string   `json:"subagent_model,omitempty"`
	EngramPath    string   `json:"engram_path,omitempty"`
	EngramMCP     bool     `json:"engram_mcp"`
	Extension     bool     `json:"extension"`
	// Config is omp's mcp.json; Prompt is APPEND_SYSTEM.md.
	Config ConfigRecord `json:"config"`
	Prompt ConfigRecord `json:"prompt"`
	Files  []FileRecord `json:"files"`
}

// Load reads state, returning an empty state when the file is missing.
func Load(path string) (*State, error) {
	s := &State{Version: Version}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, err
	}
	if s.Version == 0 {
		s.Version = Version
	}
	return s, nil
}

// Save writes state atomically.
func (s *State) Save(path string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if s.InstalledAt == "" {
		s.InstalledAt = now
	}
	s.UpdatedAt = now
	if s.Version == 0 {
		s.Version = Version
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Record adds or replaces a file record.
func (s *State) Record(path, hash, kind string) {
	for i := range s.Files {
		if s.Files[i].Path == path {
			s.Files[i].SHA256 = hash
			s.Files[i].Kind = kind
			return
		}
	}
	s.Files = append(s.Files, FileRecord{Path: path, SHA256: hash, Kind: kind})
}

// Find returns a file record by path.
func (s *State) Find(path string) (FileRecord, bool) {
	for _, f := range s.Files {
		if f.Path == path {
			return f, true
		}
	}
	return FileRecord{}, false
}

// HashFile returns the SHA-256 of a file.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
