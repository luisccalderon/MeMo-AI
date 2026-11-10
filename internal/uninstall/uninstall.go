// Package uninstall removes only the files MeMo-AI owns.
package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/backup"
	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/promptblock"
	"github.com/luisccalderon/MeMo-AI/internal/state"
)

// Options controls removal.
type Options struct {
	Force bool
	// RestoreConfig restores the pre-install mcp.json snapshot instead of
	// removing only the Engram entry.
	RestoreConfig bool
}

// Result reports what was removed.
type Result struct {
	Removed  []string
	Skipped  []string
	Warnings []string
	Restored string
}

// Run removes the installed files, the orchestrator prompt block and the
// Engram MCP entry.
func Run(paths omp.Paths, opts Options) (Result, error) {
	var res Result
	st, err := state.Load(paths.StateFile)
	if err != nil {
		return res, err
	}
	if st.InstalledAt == "" && len(st.Files) == 0 {
		res.Warnings = append(res.Warnings, "No MeMo-AI installation found.")
		return res, nil
	}

	dirs := map[string]bool{}
	for _, record := range st.Files {
		hash, hashErr := state.HashFile(record.Path)
		if hashErr != nil {
			res.Removed = append(res.Removed, record.Path+" (already missing)")
			continue
		}
		if hash != record.SHA256 && !opts.Force {
			res.Skipped = append(res.Skipped, record.Path)
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s was modified; use --force to remove it.", record.Path))
			continue
		}
		if err := os.Remove(record.Path); err != nil && !os.IsNotExist(err) {
			return res, err
		}
		res.Removed = append(res.Removed, record.Path)
		dirs[filepath.Dir(record.Path)] = true
	}

	// Remove empty directories, deepest first.
	for dir := range dirs {
		removeEmptyParents(dir, paths.ConfigDir)
	}
	removeEmptyParents(paths.ExtensionDir, paths.ConfigDir)

	if err := cleanPrompt(paths, &res); err != nil {
		res.Warnings = append(res.Warnings, "Could not clean "+paths.PromptFile+": "+err.Error())
	}

	// Restore the mcp.json captured before the first install. Without a
	// backup, or without --restore, remove only the entry MeMo-AI manages.
	if opts.RestoreConfig && st.Config.Backup != "" {
		stamp := "uninstall-" + backup.Stamp()
		if omp.Exists(st.Config.Path) {
			if saved, err := backup.Snapshot(paths.BackupDir, stamp, st.Config.Path, filepath.Base(st.Config.Path)); err == nil && saved != "" {
				res.Warnings = append(res.Warnings, "Current config snapshotted at "+saved)
			}
		}
		if err := backup.Copy(st.Config.Backup, st.Config.Path); err != nil {
			res.Warnings = append(res.Warnings, "Could not restore config: "+err.Error())
		} else {
			res.Restored = st.Config.Path
		}
	} else if err := cleanMCP(paths, st, &res); err != nil {
		res.Warnings = append(res.Warnings, "Could not clean config: "+err.Error())
	}

	if st.ModelApplied {
		res.Warnings = append(res.Warnings,
			"The omp default model was left as is. Change it with /model or `omp config reset modelRoles.default`.")
	}

	_ = os.Remove(paths.StateFile)
	removeEmptyParents(filepath.Dir(paths.StateFile), paths.ConfigDir)
	return res, nil
}

// cleanPrompt strips the managed block from APPEND_SYSTEM.md. Text the user
// wrote around it is preserved; the file is deleted only when nothing is left.
func cleanPrompt(paths omp.Paths, res *Result) error {
	raw, err := os.ReadFile(paths.PromptFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	stripped, found := promptblock.Strip(string(raw))
	if !found {
		return nil
	}
	if strings.TrimSpace(stripped) == "" {
		if err := os.Remove(paths.PromptFile); err != nil {
			return err
		}
		res.Removed = append(res.Removed, paths.PromptFile)
		return nil
	}
	if err := os.WriteFile(paths.PromptFile, []byte(stripped), 0o644); err != nil {
		return err
	}
	res.Removed = append(res.Removed, paths.PromptFile+" (managed block)")
	return nil
}

// cleanMCP removes the Engram server only when it still points at the binary
// MeMo-AI configured, so a server the user re-pointed is left alone.
func cleanMCP(paths omp.Paths, st *state.State, res *Result) error {
	if !st.EngramMCP || !omp.Exists(paths.MCPFile) {
		return nil
	}
	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		return err
	}
	current := config.GetString(doc.Data, "mcpServers", "engram", "command")
	if current == "" || current != st.EngramPath {
		return nil
	}
	config.Delete(doc.Data, "mcpServers", "engram")
	if servers, ok := doc.Data["mcpServers"].(map[string]any); ok && len(servers) == 0 {
		delete(doc.Data, "mcpServers")
	}
	if len(doc.Data) == 0 && !st.Config.Existed {
		if err := os.Remove(paths.MCPFile); err != nil {
			return err
		}
		res.Removed = append(res.Removed, paths.MCPFile)
		return nil
	}
	if err := doc.Save(); err != nil {
		return err
	}
	res.Removed = append(res.Removed, paths.MCPFile+" (cleaned)")
	return nil
}

func removeEmptyParents(dir, stop string) {
	for {
		if dir == "" || dir == stop || dir == "." || dir == string(filepath.Separator) {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
