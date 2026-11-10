// Package install writes the MeMo-AI stack into an oh-my-pi configuration.
package install

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/luisccalderon/MeMo-AI/internal/assets"
	"github.com/luisccalderon/MeMo-AI/internal/backup"
	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/engram"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/promptblock"
	"github.com/luisccalderon/MeMo-AI/internal/state"
)

// MemoSubagents lists the subagents shipped by MeMo-AI.
var MemoSubagents = []string{"memo-explorer", "memo-writer", "memo-reviewer", "memo-curator"}

// OrchestratorPrompt is the embedded asset written into APPEND_SYSTEM.md.
const OrchestratorPrompt = "prompts/memo-orchestrator.md"

// ModelRole is the omp model role MeMo-AI assigns the primary model to.
const ModelRole = "modelRoles.default"

// ModelSetter persists the primary model in omp's settings.
type ModelSetter func(model string) error

// Options controls an installation.
type Options struct {
	Lang string
	// Orchestrator installs the MeMo operating loop into APPEND_SYSTEM.md.
	Orchestrator  bool
	PrimaryModel  string
	SubagentModel string
	Subagents     []string
	EnableEngram  bool
	EngramPath    string
	// InstallExtension installs the omp extension (status badge, memo_status).
	InstallExtension bool
	// SetModel applies PrimaryModel. When nil the model is recorded in state
	// only and the user is told how to apply it.
	SetModel ModelSetter
	DryRun   bool
}

// Result reports what an installation changed.
type Result struct {
	Writes       []string
	Backups      []string
	Warnings     []string
	ConfigPath   string
	BackupPath   string
	PromptPath   string
	ExtensionDir string
	State        *state.State
}

// OmpModelSetter returns a ModelSetter backed by `omp config set`, or nil when
// the omp binary is unknown. omp validates the value against its own schema,
// so MeMo-AI never has to edit config.yml by hand.
func OmpModelSetter(ompPath string) ModelSetter {
	if ompPath == "" {
		return nil
	}
	return func(model string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, ompPath, "config", "set", ModelRole, model).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
}

// Run applies the stack. It is idempotent and safe to re-run.
func Run(paths omp.Paths, opts Options) (Result, error) {
	res := Result{ConfigPath: paths.MCPFile, PromptPath: paths.PromptFile, ExtensionDir: paths.ExtensionDir}
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
	planned := map[string]bool{}
	for _, file := range files {
		dest, ok := destination(paths, file.Path, opts)
		if !ok {
			continue
		}
		raw, err := assets.Read(file.Path)
		if err != nil {
			return res, err
		}
		if file.Kind == "agent" && opts.SubagentModel != "" {
			raw = withModel(raw, opts.SubagentModel)
		}
		backupPath, err := writeFile(paths, stamp, file.Path, dest, raw, opts.DryRun)
		if err != nil {
			return res, err
		}
		if backupPath != "" {
			res.Backups = append(res.Backups, backupPath)
		}
		planned[dest] = true
		res.Writes = append(res.Writes, dest)
		if !opts.DryRun {
			hash, err := state.HashFile(dest)
			if err != nil {
				return res, err
			}
			st.Record(dest, hash, file.Kind)
		}
	}

	if opts.DryRun {
		if opts.Orchestrator {
			res.Writes = append(res.Writes, paths.PromptFile+" (managed block)")
		}
		if opts.EnableEngram && opts.EngramPath != "" {
			res.Writes = append(res.Writes, paths.MCPFile)
		}
		if opts.PrimaryModel != "" {
			res.Writes = append(res.Writes, modelCommand(opts.PrimaryModel))
		}
		return res, nil
	}

	removeStaleFiles(st, planned, &res)

	if err := applyPrompt(paths, st, stamp, opts, &res); err != nil {
		return res, err
	}
	if err := applyMCP(paths, st, stamp, opts, &res); err != nil {
		return res, err
	}
	st.ModelApplied = applyModel(opts, &res)

	st.Language = opts.Lang
	st.Orchestrator = opts.Orchestrator
	st.PrimaryModel = opts.PrimaryModel
	st.SubagentModel = opts.SubagentModel
	st.Subagents = append([]string(nil), opts.Subagents...)
	st.EngramPath = opts.EngramPath
	st.EngramMCP = opts.EnableEngram && opts.EngramPath != ""
	st.Extension = opts.InstallExtension

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

// destination maps an embedded asset to its place in the omp agent directory.
// omp has no per-agent allow list for the main session, so a subagent is
// enabled by installing its file and disabled by not installing it.
func destination(paths omp.Paths, rel string, opts Options) (string, bool) {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "agents/"):
		name := strings.TrimSuffix(path.Base(rel), ".md")
		for _, enabled := range opts.Subagents {
			if enabled == name {
				return filepath.Join(paths.AgentsDir, path.Base(rel)), true
			}
		}
		return "", false
	case strings.HasPrefix(rel, "commands/"):
		return filepath.Join(paths.CommandsDir, path.Base(rel)), true
	case strings.HasPrefix(rel, "skills/"):
		sub := strings.TrimPrefix(rel, "skills/")
		return filepath.Join(paths.SkillsDir, filepath.FromSlash(sub)), true
	case strings.HasPrefix(rel, "extensions/memo-ai/"):
		if !opts.InstallExtension {
			return "", false
		}
		sub := strings.TrimPrefix(rel, "extensions/memo-ai/")
		return filepath.Join(paths.ExtensionDir, filepath.FromSlash(sub)), true
	default:
		// prompts/ is merged into APPEND_SYSTEM.md, never copied.
		return "", false
	}
}

// withModel pins a model selector in an agent file's YAML frontmatter.
func withModel(raw []byte, model string) []byte {
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		return raw
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return raw
	}
	head := text[:4+end]
	line := "model: " + strconv.Quote(model)
	var kept []string
	for _, existing := range strings.Split(head, "\n") {
		if !strings.HasPrefix(existing, "model:") {
			kept = append(kept, existing)
		}
	}
	return []byte(strings.Join(kept, "\n") + "\n" + line + text[4+end:])
}

func writeFile(paths omp.Paths, stamp, rel, dest string, data []byte, dry bool) (string, error) {
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
	return backupPath, writeAtomic(dest, data)
}

func writeAtomic(dest string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".memo-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	return os.Rename(name, dest)
}

// removeStaleFiles deletes files a previous run installed that this run no
// longer plans: a disabled subagent, a disabled extension, or an asset dropped
// by a newer version. Files the user edited are kept and stay recorded.
func removeStaleFiles(st *state.State, planned map[string]bool, res *Result) {
	var keep []state.FileRecord
	for _, record := range st.Files {
		if planned[record.Path] {
			keep = append(keep, record)
			continue
		}
		hash, err := state.HashFile(record.Path)
		if err != nil {
			continue // already gone
		}
		if hash != record.SHA256 {
			res.Warnings = append(res.Warnings,
				"Se conservó un archivo modificado por el usuario: "+record.Path+" / Kept a user-modified file.")
			keep = append(keep, record)
			continue
		}
		if err := os.Remove(record.Path); err == nil {
			res.Writes = append(res.Writes, "removed "+record.Path)
			removeIfEmpty(filepath.Dir(record.Path))
		}
	}
	st.Files = keep
}

func removeIfEmpty(dir string) {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		_ = os.Remove(dir)
	}
}

// applyPrompt keeps the orchestrator block in APPEND_SYSTEM.md in sync with
// the Orchestrator option, leaving the user's own text untouched.
func applyPrompt(paths omp.Paths, st *state.State, stamp string, opts Options, res *Result) error {
	raw, err := os.ReadFile(paths.PromptFile)
	existed := err == nil
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", paths.PromptFile, err)
	}
	current := string(raw)

	var next string
	if opts.Orchestrator {
		body, err := assets.Read(OrchestratorPrompt)
		if err != nil {
			return err
		}
		next = promptblock.Upsert(current, string(body))
	} else {
		stripped, found := promptblock.Strip(current)
		if !found {
			return nil
		}
		next = stripped
	}
	if next == current {
		return nil
	}

	if existed && st.Prompt.Backup == "" {
		b, err := backup.Snapshot(paths.BackupDir, stamp, paths.PromptFile, filepath.Base(paths.PromptFile))
		if err != nil {
			return err
		}
		if b != "" {
			st.Prompt.Backup = b
			res.Backups = append(res.Backups, b)
		}
	}
	if st.Prompt.Path == "" {
		st.Prompt.Existed = existed
	}
	st.Prompt.Path = paths.PromptFile

	if strings.TrimSpace(next) == "" {
		if err := os.Remove(paths.PromptFile); err != nil && !os.IsNotExist(err) {
			return err
		}
		res.Writes = append(res.Writes, "removed "+paths.PromptFile)
		return nil
	}
	if err := writeAtomic(paths.PromptFile, []byte(next)); err != nil {
		return fmt.Errorf("write %s: %w", paths.PromptFile, err)
	}
	res.Writes = append(res.Writes, paths.PromptFile)
	return nil
}

// applyMCP merges the Engram server into mcp.json. The file is only created
// or rewritten when there is something to add or remove.
func applyMCP(paths omp.Paths, st *state.State, stamp string, opts Options, res *Result) error {
	enable := opts.EnableEngram && opts.EngramPath != ""
	if opts.EnableEngram && opts.EngramPath == "" {
		res.Warnings = append(res.Warnings,
			"Engram no está instalado: se omitió el servidor MCP. / Engram is not installed: MCP server skipped.")
	}

	existed := omp.Exists(paths.MCPFile)
	if !enable && !existed {
		return nil
	}
	doc, err := config.Load(paths.MCPFile)
	if err != nil {
		return err
	}

	current := config.GetString(doc.Data, "mcpServers", "engram", "command")
	switch {
	case enable:
		if current != "" && current != opts.EngramPath {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("Se reemplazó el comando MCP de Engram (%s → %s). / Replaced Engram MCP command.",
					current, opts.EngramPath))
		}
		servers := config.EnsureMap(doc.Data, "mcpServers")
		want := map[string]any{
			"type":    "stdio",
			"command": opts.EngramPath,
			"args":    engram.MCPArgs(),
		}
		if existed && config.FormatJSON(servers["engram"]) == config.FormatJSON(want) {
			return nil
		}
		servers["engram"] = want
	case st.EngramMCP && current != "" && current == st.EngramPath:
		// Engram was enabled by a previous run and is now turned off.
		config.Delete(doc.Data, "mcpServers", "engram")
		if servers, ok := doc.Data["mcpServers"].(map[string]any); ok && len(servers) == 0 {
			delete(doc.Data, "mcpServers")
		}
	default:
		return nil
	}

	if existed && st.Config.Backup == "" {
		b, err := backup.Snapshot(paths.BackupDir, stamp, paths.MCPFile, filepath.Base(paths.MCPFile))
		if err != nil {
			return err
		}
		if b != "" {
			st.Config.Backup = b
			res.Backups = append(res.Backups, b)
		}
	}
	if st.Config.Path == "" {
		st.Config.Existed = existed
	}
	st.Config.Path = paths.MCPFile
	if err := doc.Save(); err != nil {
		return fmt.Errorf("write %s: %w", paths.MCPFile, err)
	}
	res.Writes = append(res.Writes, paths.MCPFile)
	res.BackupPath = st.Config.Backup
	return nil
}

// applyModel reports whether the model reached omp's settings.
func applyModel(opts Options, res *Result) bool {
	if opts.PrimaryModel == "" {
		return false
	}
	manual := modelCommand(opts.PrimaryModel)
	if opts.SetModel == nil {
		res.Warnings = append(res.Warnings,
			"No se encontró omp: aplica el modelo con `"+manual+"`. / omp not found: apply the model with that command.")
		return false
	}
	if err := opts.SetModel(opts.PrimaryModel); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo aplicar el modelo (%v). Ejecuta `%s`. / Could not apply the model; run that command.", err, manual))
		return false
	}
	res.Writes = append(res.Writes, manual)
	return true
}

func modelCommand(model string) string {
	return "omp config set " + ModelRole + " " + model
}
