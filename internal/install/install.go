// Package install writes the MeMo-AI stack into a Pi configuration.
package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/assets"
	"github.com/luisccalderon/MeMo-AI/internal/backup"
	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/engram"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
	"github.com/luisccalderon/MeMo-AI/internal/promptblock"
	"github.com/luisccalderon/MeMo-AI/internal/state"
)

// OrchestratorPrompt is the embedded asset written into APPEND_SYSTEM.md.
const OrchestratorPrompt = "prompts/memo-orchestrator.md"

// Options controls an installation.
//
// Subagents are intentionally absent: Pi has no built-in subagent support, so
// the stack runs inline. The agent assets stay embedded for a future release
// but are never installed.
type Options struct {
	Lang string
	// Orchestrator installs the MeMo operating loop into APPEND_SYSTEM.md.
	Orchestrator bool
	// PrimaryModel is a provider/model reference written to settings.json as
	// defaultProvider and defaultModel.
	PrimaryModel string
	EnableEngram bool
	EngramPath   string
	// InstallExtension installs the Pi extension (status badge, memo_status).
	InstallExtension bool
	// QuietStartup sets quietStartup to "header" in settings.json, so Pi keeps
	// its version header and hides the loaded-resource listing.
	QuietStartup bool
	// AtelierConfig writes the pi-atelier.json preset, but only when the user
	// opted into pi-atelier and the file does not exist yet.
	AtelierConfig bool
	// AtelierConfigFile is a pi-atelier.json the user wants applied verbatim.
	// It takes precedence over the preset and implies installing the package.
	AtelierConfigFile string
	DryRun            bool
}

// Result reports what an installation changed.
type Result struct {
	Writes        []string
	Backups       []string
	Warnings      []string
	ConfigPath    string
	BackupPath    string
	PromptPath    string
	ExtensionDir  string
	AtelierConfig bool
	State         *state.State
}

// SplitModel splits a provider/model reference at the first slash, so model
// IDs that contain slashes (for example OpenRouter ones) stay intact.
func SplitModel(ref string) (provider, model string, ok bool) {
	ref = strings.TrimSpace(ref)
	provider, model, found := strings.Cut(ref, "/")
	if !found || provider == "" || model == "" || strings.ContainsAny(ref, " \t") {
		return "", "", false
	}
	return provider, model, true
}

// Run applies the stack. It is idempotent and safe to re-run.
func Run(paths pi.Paths, opts Options) (Result, error) {
	res := Result{ConfigPath: paths.MCPFile, PromptPath: paths.PromptFile, ExtensionDir: paths.ExtensionDir}

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
			res.Writes = append(res.Writes, paths.SettingsFile+" (defaultProvider, defaultModel)")
		}
		if opts.QuietStartup {
			res.Writes = append(res.Writes, paths.SettingsFile+" (quietStartup)")
		}
		if opts.AtelierConfig && !pi.Exists(paths.AtelierFile) {
			res.Writes = append(res.Writes, paths.AtelierFile)
		}
		if opts.AtelierConfigFile != "" {
			res.Writes = append(res.Writes, paths.AtelierFile+" (from "+opts.AtelierConfigFile+")")
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
	st.ModelApplied = applyModel(paths, stamp, opts, &res)
	if applyQuietStartup(paths, stamp, opts, &res) {
		st.QuietStartup = true
	}
	res.AtelierConfig = applyAtelierConfig(paths, st, stamp, opts, &res)

	st.Version = state.Version
	st.Language = opts.Lang
	st.Orchestrator = opts.Orchestrator
	st.PrimaryModel = opts.PrimaryModel
	st.EngramPath = opts.EngramPath
	st.EngramMCP = opts.EnableEngram && opts.EngramPath != ""
	st.Extension = opts.InstallExtension

	if err := st.Save(paths.StateFile); err != nil {
		return res, fmt.Errorf("write state: %w", err)
	}
	res.State = st
	return res, nil
}

// destination maps an embedded asset to its place in the Pi agent directory.
//
// agents/ is deliberately not installed: Pi has no built-in subagent support,
// so those assets are kept in the binary for a future release. commands/ are
// Pi prompt templates, which Pi exposes as slash commands from prompts/.
func destination(paths pi.Paths, rel string, opts Options) (string, bool) {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "agents/"):
		return "", false
	case strings.HasPrefix(rel, "commands/"):
		return filepath.Join(paths.PromptsDir, path.Base(rel)), true
	case strings.HasPrefix(rel, "skills/"):
		sub := strings.TrimPrefix(rel, "skills/")
		return filepath.Join(paths.SkillsDir, filepath.FromSlash(sub)), true
	case strings.HasPrefix(rel, "extensions/memo-ai/"):
		if !opts.InstallExtension {
			return "", false
		}
		sub := strings.TrimPrefix(rel, "extensions/memo-ai/")
		return filepath.Join(paths.ExtensionDir, filepath.FromSlash(sub)), true
	case strings.HasPrefix(rel, "atelier/"):
		// A preset for a third-party package: written by applyAtelierConfig,
		// only when the user opted in and owns no file yet.
		return "", false
	default:
		// prompts/ is merged into APPEND_SYSTEM.md, never copied.
		return "", false
	}
}

func writeFile(paths pi.Paths, stamp, rel, dest string, data []byte, dry bool) (string, error) {
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
func applyPrompt(paths pi.Paths, st *state.State, stamp string, opts Options, res *Result) error {
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
func applyMCP(paths pi.Paths, st *state.State, stamp string, opts Options, res *Result) error {
	enable := opts.EnableEngram && opts.EngramPath != ""
	if opts.EnableEngram && opts.EngramPath == "" {
		res.Warnings = append(res.Warnings,
			"Engram no está instalado: se omitió el servidor MCP. / Engram is not installed: MCP server skipped.")
	}

	existed := pi.Exists(paths.MCPFile)
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
		// Pi's default exposure ("codemode") hides a server's tools from the
		// model behind scripts. Engram is a small tool set the agent uses on
		// every turn, so it is declared directly like a built-in tool.
		want := map[string]any{
			"type":     "stdio",
			"command":  opts.EngramPath,
			"args":     engram.MCPArgs(),
			"exposure": "direct",
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

// applyQuietStartup sets quietStartup in Pi's settings.json and reports
// whether this run wrote it. A value the user already chose ("header" or
// true) is left alone, and so is a settings file that cannot be parsed.
func applyQuietStartup(paths pi.Paths, stamp string, opts Options, res *Result) bool {
	if !opts.QuietStartup {
		return false
	}
	existed := pi.Exists(paths.SettingsFile)
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo leer settings.json (%v); el arranque silencioso no se aplicó. / Could not read settings.json; quiet startup was not applied.", err))
		return false
	}
	if current, ok := config.Get(doc.Data, "quietStartup"); ok && (current == pi.QuietStartupValue || current == true) {
		return false
	}
	if existed {
		b, err := backup.Snapshot(paths.BackupDir, stamp, paths.SettingsFile, filepath.Base(paths.SettingsFile))
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("No se pudo respaldar settings.json (%v). / Could not back up settings.json.", err))
			return false
		}
		if b != "" {
			res.Backups = append(res.Backups, b)
		}
	}
	doc.Data["quietStartup"] = pi.QuietStartupValue
	if err := doc.Save(); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo aplicar el arranque silencioso (%v). / Could not apply quiet startup.", err))
		return false
	}
	res.Writes = append(res.Writes, paths.SettingsFile+" (quietStartup)")
	return true
}

// applyAtelierConfig writes pi-atelier.json and reports whether this run wrote
// it. With AtelierConfigFile the user's own file wins and is applied verbatim;
// otherwise the embedded preset is used, and only when no file exists yet,
// because the package owns the keys a user may have changed.
func applyAtelierConfig(paths pi.Paths, st *state.State, stamp string, opts Options, res *Result) bool {
	if opts.AtelierConfigFile != "" {
		raw, err := os.ReadFile(opts.AtelierConfigFile)
		if err != nil {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("No se pudo leer %s (%v). / Could not read %s.", opts.AtelierConfigFile, err, opts.AtelierConfigFile))
			return false
		}
		if err := validateAtelierConfig(raw); err != nil {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s no es un JSON de objeto válido (%v). / %s is not a valid JSON object.", opts.AtelierConfigFile, err, opts.AtelierConfigFile))
			return false
		}
		return writeAtelierConfig(paths, st, stamp, raw, "atelier-config", res)
	}
	if !opts.AtelierConfig || pi.Exists(paths.AtelierFile) {
		return false
	}
	raw, err := assets.Read(pi.AtelierConfigAsset)
	if err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo leer el preset de pi-atelier (%v). / Could not read the pi-atelier preset.", err))
		return false
	}
	return writeAtelierConfig(paths, st, stamp, raw, "atelier-preset", res)
}

// writeAtelierConfig writes raw to pi-atelier.json, snapshotting an existing
// file first, and records the hash so uninstall can undo only what it wrote.
func writeAtelierConfig(paths pi.Paths, st *state.State, stamp string, raw []byte, kind string, res *Result) bool {
	if existing, err := os.ReadFile(paths.AtelierFile); err == nil {
		if bytes.Equal(existing, raw) {
			return false
		}
		saved, err := backup.Snapshot(paths.BackupDir, stamp, paths.AtelierFile, filepath.Base(paths.AtelierFile))
		if err != nil {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("No se pudo respaldar pi-atelier.json (%v). / Could not back up pi-atelier.json.", err))
			return false
		}
		if saved != "" {
			res.Backups = append(res.Backups, saved)
		}
	} else if err := os.MkdirAll(filepath.Dir(paths.AtelierFile), 0o755); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo crear el directorio de pi-atelier (%v). / Could not create the pi-atelier directory.", err))
		return false
	}
	if err := os.WriteFile(paths.AtelierFile, raw, 0o644); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo escribir pi-atelier.json (%v). / Could not write pi-atelier.json.", err))
		return false
	}
	hash, err := state.HashFile(paths.AtelierFile)
	if err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo leer pi-atelier.json tras escribirlo (%v). / Could not read pi-atelier.json back.", err))
		return false
	}
	st.AtelierConfig = state.FileRecord{Path: paths.AtelierFile, SHA256: hash, Kind: kind}
	res.Writes = append(res.Writes, paths.AtelierFile)
	return true
}

// validateAtelierConfig accepts only a JSON object: the package reads named
// keys, so an array or a scalar would be written and then silently ignored.
func validateAtelierConfig(raw []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if doc == nil {
		return errors.New("empty object")
	}
	return nil
}

// applyModel sets defaultProvider and defaultModel in Pi's settings.json and
// reports whether the model reached it. Every other key, and every other
// setting, is preserved. A settings file that cannot be parsed is left
// untouched and reported as a warning so a hand-edited file is never clobbered.
func applyModel(paths pi.Paths, stamp string, opts Options, res *Result) bool {
	if opts.PrimaryModel == "" {
		return false
	}
	provider, model, ok := SplitModel(opts.PrimaryModel)
	if !ok {
		res.Warnings = append(res.Warnings,
			"El modelo debe tener el formato provider/model. / Model must be provider/model.")
		return false
	}
	manual := "/model " + opts.PrimaryModel

	existed := pi.Exists(paths.SettingsFile)
	doc, err := config.Load(paths.SettingsFile)
	if err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo leer settings.json (%v); elige el modelo con `%s`. / Could not read settings.json; choose the model with that command.", err, manual))
		return false
	}
	if config.GetString(doc.Data, "defaultProvider") == provider && config.GetString(doc.Data, "defaultModel") == model {
		return true
	}
	if existed {
		b, err := backup.Snapshot(paths.BackupDir, stamp, paths.SettingsFile, filepath.Base(paths.SettingsFile))
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("No se pudo respaldar settings.json (%v). / Could not back up settings.json.", err))
			return false
		}
		if b != "" {
			res.Backups = append(res.Backups, b)
		}
	}
	doc.Data["defaultProvider"] = provider
	doc.Data["defaultModel"] = model
	if err := doc.Save(); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("No se pudo aplicar el modelo (%v). Usa `%s`. / Could not apply the model; use that command.", err, manual))
		return false
	}
	res.Writes = append(res.Writes, paths.SettingsFile+" (defaultProvider, defaultModel)")
	return true
}
