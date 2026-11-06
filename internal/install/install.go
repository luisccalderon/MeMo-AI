// Package install writes the MeMo-AI stack into an OpenCode configuration.
package install

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/luis/memo-ai/internal/assets"
	"github.com/luis/memo-ai/internal/backup"
	"github.com/luis/memo-ai/internal/config"
	"github.com/luis/memo-ai/internal/engram"
	"github.com/luis/memo-ai/internal/opencode"
	"github.com/luis/memo-ai/internal/state"
)

// MemoSubagents lists the subagents shipped by MeMo-AI.
var MemoSubagents = []string{"memo-explorer", "memo-writer", "memo-reviewer", "memo-curator"}

// Options controls an installation.
type Options struct {
	Lang          string
	PrimaryAgent  string
	PrimaryModel  string
	SubagentModel string
	Subagents     []string
	EnableEngram  bool
	EngramPath    string
	InstallPlugin bool
	DryRun        bool
}

// Result reports what an installation changed.
type Result struct {
	Writes     []string
	Backups    []string
	Warnings   []string
	ConfigPath string
	BackupPath string
	PluginDir  string
	State      *state.State
}

// Run applies the stack. It is idempotent and safe to re-run.
func Run(paths opencode.Paths, opts Options) (Result, error) {
	res := Result{ConfigPath: paths.ConfigFile, PluginDir: paths.PluginDir}
	if opts.PrimaryAgent == "" {
		opts.PrimaryAgent = "memo-orchestrator"
	}
	opts.Subagents = filterSubagents(opts.Subagents)

	if !opts.DryRun {
		if err := paths.EnsureDirs(); err != nil {
			return res, err
		}
	}

	st, err := state.Load(paths.StateFile)
	if err != nil {
		return res, fmt.Errorf("read state: %w", err)
	}

	stamp := backup.Stamp()
	files, err := assets.Walk()
	if err != nil {
		return res, err
	}
	for _, file := range files {
		dest, ok := destination(paths, file.Path, opts.InstallPlugin)
		if !ok {
			continue
		}
		raw, err := assets.Read(file.Path)
		if err != nil {
			return res, err
		}
		backupPath, err := writeFile(paths, stamp, file.Path, dest, raw, opts.DryRun)
		if err != nil {
			return res, err
		}
		if backupPath != "" {
			res.Backups = append(res.Backups, backupPath)
		}
		res.Writes = append(res.Writes, dest)
		if !opts.DryRun {
			hash, err := state.HashFile(dest)
			if err != nil {
				return res, err
			}
			st.Record(dest, hash, file.Kind)
		}
	}

	doc, err := config.Load(paths.ConfigFile)
	if err != nil {
		return res, err
	}
	configExisted := opencode.Exists(paths.ConfigFile)
	if strings.HasSuffix(strings.ToLower(paths.ConfigFile), ".jsonc") {
		if raw, err := os.ReadFile(paths.ConfigFile); err == nil && bytes.Contains(raw, []byte("//")) {
			res.Warnings = append(res.Warnings,
				"El archivo es JSONC: los comentarios se perderán al reescribirlo (hay respaldo). / JSONC comments will be lost (a backup exists).")
		}
	}

	applyConfig(doc, opts, &res)

	if opts.DryRun {
		res.Writes = append(res.Writes, paths.ConfigFile)
		return res, nil
	}

	if configExisted && st.Config.Backup == "" {
		b, err := backup.Snapshot(paths.BackupDir, stamp, paths.ConfigFile, filepath.Base(paths.ConfigFile))
		if err != nil {
			return res, err
		}
		if b != "" {
			res.Backups = append(res.Backups, b)
			st.Config.Backup = b
		}
	}
	st.Config.Path = paths.ConfigFile
	st.Config.Existed = configExisted
	if err := doc.Save(); err != nil {
		return res, fmt.Errorf("write %s: %w", paths.ConfigFile, err)
	}
	res.Writes = append(res.Writes, paths.ConfigFile)
	if !contains(res.Backups, st.Config.Backup) && st.Config.Backup != "" {
		res.Backups = append(res.Backups, st.Config.Backup)
	}
	res.BackupPath = st.Config.Backup

	if !opts.InstallPlugin {
		removePluginFiles(st, &res)
	}

	st.Language = opts.Lang
	st.PrimaryAgent = opts.PrimaryAgent
	st.PrimaryModel = opts.PrimaryModel
	st.SubagentModel = opts.SubagentModel
	st.Subagents = append([]string(nil), opts.Subagents...)
	st.EngramPath = opts.EngramPath
	st.EngramMCP = opts.EnableEngram && opts.EngramPath != ""
	st.Plugin = opts.InstallPlugin

	if err := st.Save(paths.StateFile); err != nil {
		return res, fmt.Errorf("write state: %w", err)
	}
	res.State = st
	return res, nil
}

func filterSubagents(values []string) []string {
	allowed := map[string]bool{}
	for _, name := range MemoSubagents {
		allowed[name] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if allowed[value] && !seen[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	return out
}

func destination(paths opencode.Paths, rel string, includePlugin bool) (string, bool) {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "agents/"):
		return filepath.Join(paths.AgentsDir, path.Base(rel)), true
	case strings.HasPrefix(rel, "commands/"):
		return filepath.Join(paths.CommandsDir, path.Base(rel)), true
	case strings.HasPrefix(rel, "skills/"):
		sub := strings.TrimPrefix(rel, "skills/")
		return filepath.Join(paths.SkillsDir, filepath.FromSlash(sub)), true
	case strings.HasPrefix(rel, "plugins/memo-ai/"):
		if !includePlugin {
			return "", false
		}
		sub := strings.TrimPrefix(rel, "plugins/memo-ai/")
		return filepath.Join(paths.PluginDir, filepath.FromSlash(sub)), true
	default:
		return "", false
	}
}

func writeFile(paths opencode.Paths, stamp, rel, dest string, data []byte, dry bool) (string, error) {
	backupPath := ""
	if existing, err := os.ReadFile(dest); err == nil {
		if bytes.Equal(existing, data) {
			return "", nil
		}
		if !dry {
			b, err := backup.Snapshot(paths.BackupDir, stamp, dest, filepath.ToSlash(rel))
			if err != nil {
				return "", err
			}
			backupPath = b
		}
	}
	if dry {
		return backupPath, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return backupPath, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".memo-*")
	if err != nil {
		return backupPath, err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return backupPath, err
	}
	if err := tmp.Close(); err != nil {
		return backupPath, err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return backupPath, err
	}
	if err := os.Rename(name, dest); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

func applyConfig(doc *config.Document, opts Options, res *Result) {
	if _, ok := doc.Data["$schema"]; !ok {
		doc.Data["$schema"] = "https://opencode.ai/config.json"
	}
	doc.Data["default_agent"] = opts.PrimaryAgent
	if opts.PrimaryModel != "" {
		doc.Data["model"] = opts.PrimaryModel
	}

	agents := config.EnsureMap(doc.Data, "agents")
	permissions := mergeSubagentPermissions(agents[opts.PrimaryAgent], opts.Subagents)

	primary := config.EnsureMap(agents, opts.PrimaryAgent)
	if opts.PrimaryModel != "" {
		primary["model"] = opts.PrimaryModel
	}
	primary["permissions"] = permissions

	// Keep memo-orchestrator ready even when the user selects another primary
	// agent, so switching later does not require a reconfigure.
	if opts.PrimaryAgent != "memo-orchestrator" {
		orchestrator := config.EnsureMap(agents, "memo-orchestrator")
		if opts.PrimaryModel != "" {
			orchestrator["model"] = opts.PrimaryModel
		}
		orchestrator["permissions"] = permissions
	}

	for _, name := range opts.Subagents {
		sub := config.EnsureMap(agents, name)
		if opts.SubagentModel != "" {
			sub["model"] = opts.SubagentModel
		}
	}

	if opts.PrimaryAgent == "memo-orchestrator" && opts.PrimaryModel != "" {
		doc.Data["model"] = opts.PrimaryModel
	}

	if len(opts.Subagents) > 0 {
		experimental := config.EnsureMap(doc.Data, "experimental")
		if _, ok := experimental["subagent_depth"]; !ok {
			experimental["subagent_depth"] = 1
		}
	}

	if opts.EnableEngram {
		if opts.EngramPath == "" {
			res.Warnings = append(res.Warnings,
				"Engram no está instalado: se omitió el servidor MCP. / Engram is not installed: MCP server skipped.")
		} else {
			servers := config.EnsureMap(doc.Data, "mcp", "servers")
			if existing, ok := servers["engram"].(map[string]any); ok {
				if command, ok := existing["command"].([]any); ok && len(command) > 0 {
					if current, _ := command[0].(string); current != "" && current != opts.EngramPath {
						res.Warnings = append(res.Warnings,
							fmt.Sprintf("Se reemplazó el comando MCP de Engram (%s → %s). / Replaced Engram MCP command.",
								current, opts.EngramPath))
					}
				}
			}
			servers["engram"] = map[string]any{
				"type":    "local",
				"command": engram.MCPCommand(opts.EngramPath),
			}
		}
	}
}

func mergeSubagentPermissions(existing any, enabled []string) []any {
	managed := map[string]bool{"*": true}
	for _, name := range MemoSubagents {
		managed[name] = true
	}

	var kept []any
	if rules, ok := existing.([]any); ok {
		for _, rule := range rules {
			m, ok := rule.(map[string]any)
			if !ok {
				kept = append(kept, rule)
				continue
			}
			action, _ := m["action"].(string)
			resource, _ := m["resource"].(string)
			if action == "subagent" && managed[resource] {
				continue
			}
			kept = append(kept, rule)
		}
	}

	rules := []any{
		map[string]any{"action": "subagent", "resource": "*", "effect": "deny"},
	}
	for _, name := range enabled {
		rules = append(rules, map[string]any{
			"action": "subagent", "resource": name, "effect": "allow",
		})
	}
	return append(kept, rules...)
}

func removePluginFiles(st *state.State, res *Result) {
	var keep []state.FileRecord
	for _, record := range st.Files {
		if record.Kind != "plugin" {
			keep = append(keep, record)
			continue
		}
		hash, err := state.HashFile(record.Path)
		if err == nil && hash == record.SHA256 {
			if err := os.Remove(record.Path); err == nil {
				res.Writes = append(res.Writes, "removed "+record.Path)
			}
			continue
		}
		res.Warnings = append(res.Warnings,
			"Se conservó un plugin modificado por el usuario: "+record.Path+" / Kept a user-modified plugin file.")
		keep = append(keep, record)
	}
	st.Files = keep
}

func contains(values []string, value string) bool {
	if value == "" {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
