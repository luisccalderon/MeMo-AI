// Package uninstall removes only the files MeMo-AI owns.
package uninstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luis/memo-ai/internal/backup"
	"github.com/luis/memo-ai/internal/config"
	"github.com/luis/memo-ai/internal/opencode"
	"github.com/luis/memo-ai/internal/state"
)

// Options controls removal.
type Options struct {
	Force         bool
	RestoreConfig bool
}

// Result reports what was removed.
type Result struct {
	Removed  []string
	Skipped  []string
	Warnings []string
	Restored string
}

// Run removes the installed files and optionally restores the config backup.
func Run(paths opencode.Paths, opts Options) (Result, error) {
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
	removeEmptyParents(paths.PluginDir, paths.ConfigDir)

	// Restore the configuration captured before the first install. When there is
	// no backup (an older install), remove only the keys MeMo-AI manages.
	if opts.RestoreConfig && st.Config.Backup != "" {
		stamp := "uninstall-" + backup.Stamp()
		if opencode.Exists(st.Config.Path) {
			if saved, err := backup.Snapshot(paths.BackupDir, stamp, st.Config.Path, filepath.Base(st.Config.Path)); err == nil && saved != "" {
				res.Warnings = append(res.Warnings, "Current config snapshotted at "+saved)
			}
		}
		if err := backup.Copy(st.Config.Backup, st.Config.Path); err != nil {
			res.Warnings = append(res.Warnings, "Could not restore config: "+err.Error())
		} else {
			res.Restored = st.Config.Path
		}
	} else {
		if err := cleanConfig(paths, st, &res); err != nil {
			res.Warnings = append(res.Warnings, "Could not clean config: "+err.Error())
		}
	}

	_ = os.Remove(paths.StateFile)
	removeEmptyParents(filepath.Dir(paths.StateFile), paths.ConfigDir)
	return res, nil
}

func cleanConfig(paths opencode.Paths, st *state.State, res *Result) error {
	doc, err := config.Load(paths.ConfigFile)
	if err != nil {
		return err
	}
	changed := false

	if command, ok := config.Get(doc.Data, "mcp", "servers", "engram", "command"); ok {
		if list, ok := command.([]any); ok && len(list) > 0 {
			if first, _ := list[0].(string); first != "" {
				config.Delete(doc.Data, "mcp", "servers", "engram")
				changed = true
			}
		}
	}
	if stripManagedPermissions(doc.Data, st.PrimaryAgent) {
		changed = true
	}
	if stripManagedPermissions(doc.Data, "memo-orchestrator") {
		changed = true
	}
	for _, name := range append([]string{"memo-orchestrator"}, memoSubagents()...) {
		if _, ok := config.Get(doc.Data, "agents", name); ok {
			config.Delete(doc.Data, "agents", name)
			changed = true
		}
	}
	if config.GetString(doc.Data, "default_agent") == st.PrimaryAgent {
		config.Delete(doc.Data, "default_agent")
		changed = true
	}
	if st.PrimaryModel != "" && config.GetString(doc.Data, "model") == st.PrimaryModel {
		config.Delete(doc.Data, "model")
		changed = true
	}
	if depth, ok := config.Get(doc.Data, "experimental", "subagent_depth"); ok {
		if n, ok := depth.(float64); ok && n == 1 {
			config.Delete(doc.Data, "experimental", "subagent_depth")
			changed = true
		}
	}
	if !changed {
		return nil
	}
	pruneEmpty(doc.Data)
	if err := doc.Save(); err != nil {
		return err
	}
	res.Removed = append(res.Removed, paths.ConfigFile+" (cleaned)")
	return nil
}

// stripManagedPermissions removes the subagent rules MeMo-AI adds to an agent
// while preserving every other rule the user configured.
func stripManagedPermissions(root map[string]any, agent string) bool {
	if agent == "" {
		return false
	}
	agents, ok := root["agents"].(map[string]any)
	if !ok {
		return false
	}
	entry, ok := agents[agent].(map[string]any)
	if !ok {
		return false
	}
	rules, ok := entry["permissions"].([]any)
	if !ok {
		return false
	}
	managed := map[string]bool{"*": true}
	for _, name := range memoSubagents() {
		managed[name] = true
	}
	kept := make([]any, 0, len(rules))
	for _, rule := range rules {
		m, ok := rule.(map[string]any)
		if ok {
			action, _ := m["action"].(string)
			resource, _ := m["resource"].(string)
			if action == "subagent" && managed[resource] {
				continue
			}
		}
		kept = append(kept, rule)
	}
	if len(kept) == len(rules) {
		return false
	}
	if len(kept) == 0 {
		delete(entry, "permissions")
	} else {
		entry["permissions"] = kept
	}
	if len(entry) == 0 {
		delete(agents, agent)
	}
	return true
}

func pruneEmpty(root map[string]any) {
	if agents, ok := root["agents"].(map[string]any); ok && len(agents) == 0 {
		delete(root, "agents")
	}
	if mcp, ok := root["mcp"].(map[string]any); ok {
		if servers, ok := mcp["servers"].(map[string]any); ok && len(servers) == 0 {
			delete(mcp, "servers")
		}
		if len(mcp) == 0 {
			delete(root, "mcp")
		}
	}
	if experimental, ok := root["experimental"].(map[string]any); ok && len(experimental) == 0 {
		delete(root, "experimental")
	}
}

func memoSubagents() []string {
	return []string{"memo-explorer", "memo-writer", "memo-reviewer", "memo-curator"}
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
