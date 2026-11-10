package cli

import (
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/engram"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
	"github.com/luisccalderon/MeMo-AI/internal/version"
)

func runWizard(paths omp.Paths, env detect.Environment, st *state.State, opts globals, lang string) int {
	ui := term.New(lang)
	ui.Banner(version.Version, env.OmpVersion)

	interactive := !opts.yes && isTTY()
	if !interactive && !opts.yes && !opts.dryRun {
		ui.Error(i18n.T(lang, "error.no_tty"))
		return 2
	}

	// Language is the first question only on a fresh interactive install.
	if opts.lang == "" && st.Language == "" && interactive {
		idx, err := ui.Select(i18n.T(lang, "wizard.language"), []term.Option{
			{Label: "Español", Description: "Prompts y mensajes en español"},
			{Label: "English", Description: "Prompts and messages in English"},
		}, 0)
		if err == nil {
			if idx == 1 {
				lang = i18n.English
			} else {
				lang = i18n.Spanish
			}
			ui.Lang = lang
		}
	}

	printDetection(ui, env, lang)

	// omp owns its settings file; the only model MeMo-AI knows is the one
	// it applied on a previous run.
	currentModel := st.PrimaryModel

	// Orchestrator prompt.
	orchestrator := resolveOrchestrator(ui, st, opts, interactive, lang)

	// Primary model.
	model := opts.model
	if model == "-" || strings.EqualFold(model, "inherit") {
		model = ""
	}
	if model == "" {
		model = chooseModel(ui, env, st, currentModel, interactive, lang)
	}
	if model != "" && !strings.Contains(model, "/") {
		ui.Warn("El modelo debe tener el formato provider/model. / Model must be provider/model.")
		model = chooseModel(ui, env, st, currentModel, interactive, lang)
	}

	// Subagents.
	subagents := resolveSubagents(ui, st, opts, interactive, lang)
	subagentModel := opts.subagentModel
	if subagentModel == "-" || strings.EqualFold(subagentModel, "inherit") {
		subagentModel = ""
	}
	if len(subagents) > 0 && subagentModel == "" && interactive {
		subagentModel = chooseSubagentModel(ui, env, model, lang)
	}

	// Engram.
	engramPath := env.EngramPath
	enableEngram := false
	if !opts.noEngram {
		if engramPath == "" && (opts.installEngram || (interactive && confirmInstallEngram(ui, lang))) {
			ui.Muted(i18n.T(lang, "engram.installing"))
			path, err := engram.Install(backgroundContext(), ui.Out)
			if err != nil {
				ui.Warn("Engram: " + err.Error())
			} else {
				engramPath = path
				ui.Success(i18n.T(lang, "engram.installed_ok") + " " + path)
			}
		}
		if engramPath != "" {
			if interactive {
				enableEngram, _ = ui.Confirm(i18n.T(lang, "wizard.engram_enable"), true)
			} else {
				enableEngram = true
			}
		}
	}

	// omp extension.
	installExtension := !opts.noExtension
	if interactive && !opts.noExtension {
		installExtension, _ = ui.Confirm(i18n.T(lang, "wizard.extension"), true)
	}

	// Summary.
	if interactive {
		printSummary(ui, lang, orchestrator, model, subagents, subagentModel, engramPath, enableEngram, installExtension)
		ok, err := ui.Confirm(i18n.T(lang, "wizard.confirm"), true)
		if err != nil || !ok {
			ui.Muted(i18n.T(lang, "wizard.cancelled"))
			return 0
		}
	}

	ui.Title(i18n.T(lang, "wizard.installing"))
	result, err := install.Run(paths, install.Options{
		Lang:             lang,
		Orchestrator:     orchestrator,
		PrimaryModel:     model,
		SubagentModel:    subagentModel,
		Subagents:        subagents,
		EnableEngram:     enableEngram && engramPath != "",
		EngramPath:       engramPath,
		InstallExtension: installExtension,
		SetModel:         install.OmpModelSetter(env.OmpPath),
		DryRun:           opts.dryRun,
	})
	if err != nil {
		ui.Error(err.Error())
		return 1
	}

	if opts.dryRun {
		ui.Warn(i18n.T(lang, "install.dry_run"))
		for _, write := range result.Writes {
			ui.Info("would write %s", write)
		}
		return 0
	}

	for _, warning := range result.Warnings {
		ui.Warn(warning)
	}
	for _, b := range result.Backups {
		ui.Muted("%s: %s", i18n.T(lang, "install.config_backup"), b)
	}
	if orchestrator {
		ui.Success("%s: %s", i18n.T(lang, "install.prompt_ready"), result.PromptPath)
	} else {
		ui.Muted("%s", i18n.T(lang, "install.prompt_skipped"))
	}
	ui.Success("%s: %d", i18n.T(lang, "install.files_written"), len(result.Writes))
	if enableEngram && engramPath != "" {
		ui.Success("%s: %s", i18n.T(lang, "install.engram_ready"), engramPath)
	} else {
		ui.Muted("%s", i18n.T(lang, "install.engram_skipped"))
	}
	if installExtension {
		ui.Success("%s: %s", i18n.T(lang, "install.extension_ready"), result.ExtensionDir)
	}
	if len(subagents) == 0 {
		ui.Muted("%s", i18n.T(lang, "install.subagents_off"))
	}

	report := doctorReport(paths, env, lang)
	okCount, warnCount, failCount, _ := report.Counts()
	ui.Title(i18n.T(lang, "doctor.title"))
	ui.Info("%d OK · %d warnings · %d failures", okCount, warnCount, failCount)
	if failCount > 0 {
		for _, check := range report.Checks {
			if check.Status == "fail" {
				ui.Error("%s: %s", check.Name, check.Detail)
			}
		}
	}
	ui.Title(i18n.T(lang, "wizard.done"))
	ui.Info("%s", i18n.T(lang, "wizard.next"))
	return 0
}

func printDetection(ui *term.UI, env detect.Environment, lang string) {
	ui.Title(i18n.T(lang, "wizard.detected"))
	if env.OmpPath != "" {
		ui.Info("%s: %s (%s)", i18n.T(lang, "wizard.omp"), env.OmpVersion, env.OmpPath)
	} else {
		ui.Warn("%s", i18n.T(lang, "error.omp_missing"))
	}
	if env.EngramPath != "" {
		ui.Info("%s: %s", i18n.T(lang, "wizard.engram"), displayEngram(env))
	} else {
		ui.Muted("%s: %s", i18n.T(lang, "wizard.engram"), i18n.T(lang, "engram.missing"))
	}
	if len(env.Models) == 0 {
		ui.Muted("%s", i18n.T(lang, "wizard.no_local_models"))
		return
	}
	seen := map[string]bool{}
	var refs []string
	for _, model := range env.Models {
		if !seen[model.Ref()] {
			refs = append(refs, model.Ref())
			seen[model.Ref()] = true
		}
	}
	ui.Info("%s: %s", i18n.T(lang, "wizard.local_models"), strings.Join(refs, ", "))
}

func displayEngram(env detect.Environment) string {
	if env.EngramVersion != "" {
		return env.EngramVersion + " (" + env.EngramPath + ")"
	}
	return env.EngramPath
}

// resolveOrchestrator decides whether the MeMo operating loop is installed
// into APPEND_SYSTEM.md. A previous choice is kept on non-interactive re-runs.
func resolveOrchestrator(ui *term.UI, st *state.State, opts globals, interactive bool, lang string) bool {
	if opts.noOrchestrator {
		return false
	}
	previous := st.InstalledAt == "" || st.Orchestrator
	if !interactive {
		return previous
	}
	enable, err := ui.Confirm(i18n.T(lang, "wizard.orchestrator"), previous)
	if err != nil {
		return previous
	}
	return enable
}

func chooseModel(ui *term.UI, env detect.Environment, st *state.State, current string, interactive bool, lang string) string {
	local := uniqueModels(env.Models)
	if !interactive {
		if st.PrimaryModel != "" {
			return st.PrimaryModel
		}
		return current
	}

	var options []term.Option
	defaultIdx := 0
	for _, model := range local {
		options = append(options, term.Option{Label: model.Ref(), Description: model.Detail})
		if model.Ref() == st.PrimaryModel || model.Ref() == current {
			defaultIdx = len(options) - 1
		}
	}
	if current != "" {
		options = append(options, term.Option{Label: current, Description: i18n.T(lang, "common.current")})
		if st.PrimaryModel == "" {
			defaultIdx = len(options) - 1
		}
	}
	manualIdx := len(options)
	options = append(options, term.Option{Label: i18n.T(lang, "common.manual"), Description: i18n.T(lang, "wizard.model_hint")})
	inheritIdx := len(options)
	options = append(options, term.Option{Label: i18n.T(lang, "common.inherit"), Description: "Leave omp model roles untouched"})
	if len(local) == 0 && current == "" {
		defaultIdx = manualIdx
	}

	ui.Muted("%s", i18n.T(lang, "wizard.model_hint"))
	idx, err := ui.Select(i18n.T(lang, "wizard.primary_model"), options, defaultIdx)
	if err != nil {
		return current
	}
	switch idx {
	case manualIdx:
		return promptModel(ui, current, lang)
	case inheritIdx:
		return ""
	default:
		if idx >= 0 && idx < len(local) {
			return local[idx].Ref()
		}
		if idx >= 0 && idx < len(options) {
			return options[idx].Label
		}
		return current
	}
}

func promptModel(ui *term.UI, current, lang string) string {
	for {
		value, err := ui.Input(i18n.T(lang, "wizard.model_manual"), current)
		if err != nil {
			return current
		}
		value = strings.TrimSpace(value)
		if value == "" || value == "-" || strings.EqualFold(value, "inherit") {
			return ""
		}
		if strings.Contains(value, "/") && !strings.ContainsAny(value, " \t") {
			return value
		}
		ui.Warn("Formato inválido. Usa provider/model. / Invalid format. Use provider/model.")
	}
}

func resolveSubagents(ui *term.UI, st *state.State, opts globals, interactive bool, lang string) []string {
	if opts.noSubagents {
		return nil
	}
	defaultEnabled := true
	if st.InstalledAt != "" && len(st.Subagents) == 0 {
		defaultEnabled = false
	}
	if !interactive {
		// Re-running install/configure must preserve the previous decision,
		// including an explicit "no subagents".
		if st.InstalledAt != "" {
			return append([]string(nil), st.Subagents...)
		}
		return append([]string(nil), install.MemoSubagents...)
	}

	enable, _ := ui.Confirm(i18n.T(lang, "wizard.subagents"), defaultEnabled)
	if !enable {
		return nil
	}

	options := []term.Option{
		{Label: "memo-explorer", Description: "Read-only exploration and code maps"},
		{Label: "memo-writer", Description: "Focused implementation for one bounded unit"},
		{Label: "memo-reviewer", Description: "Independent read-only review"},
		{Label: "memo-curator", Description: "Engram memory consolidation"},
	}
	defaults := make([]int, len(options))
	for i := range defaults {
		defaults[i] = i
	}
	if len(st.Subagents) > 0 {
		defaults = nil
		for i, name := range install.MemoSubagents {
			for _, saved := range st.Subagents {
				if name == saved {
					defaults = append(defaults, i)
				}
			}
		}
	}
	indices, err := ui.MultiSelect(i18n.T(lang, "wizard.subagents_choose"), options, defaults)
	if err != nil {
		return append([]string(nil), install.MemoSubagents...)
	}
	var out []string
	for _, idx := range indices {
		if idx >= 0 && idx < len(install.MemoSubagents) {
			out = append(out, install.MemoSubagents[idx])
		}
	}
	return out
}

func chooseSubagentModel(ui *term.UI, env detect.Environment, primaryModel, lang string) string {
	var options []term.Option
	options = append(options, term.Option{Label: i18n.T(lang, "common.inherit"), Description: "Let omp pick the subagent model"})
	values := []string{""}
	if primaryModel != "" {
		options = append(options, term.Option{Label: primaryModel, Description: "Same as the main session"})
		values = append(values, primaryModel)
	}
	for _, model := range uniqueModels(env.Models) {
		if model.Ref() == primaryModel {
			continue
		}
		options = append(options, term.Option{Label: model.Ref(), Description: model.Detail})
		values = append(values, model.Ref())
	}
	idx, err := ui.Select(i18n.T(lang, "wizard.subagent_model"), options, 0)
	if err != nil || idx < 0 || idx >= len(values) {
		return ""
	}
	return values[idx]
}

func confirmInstallEngram(ui *term.UI, lang string) bool {
	ui.Muted("%s", i18n.T(lang, "wizard.engram_missing"))
	ok, _ := ui.Confirm(i18n.T(lang, "wizard.engram_install"), true)
	return ok
}

func printSummary(ui *term.UI, lang string, orchestrator bool, model string, subagents []string, subagentModel, engramPath string, enableEngram, extension bool) {
	ui.Title(i18n.T(lang, "wizard.confirm"))
	ui.Info("Orchestrator prompt: %v", orchestrator)
	if model == "" {
		ui.Info("Model: %s", i18n.T(lang, "common.inherit"))
	} else {
		ui.Info("Model: %s", model)
	}
	if len(subagents) == 0 {
		ui.Info("Subagents: %s", i18n.T(lang, "common.none"))
	} else {
		ui.Info("Subagents: %s", strings.Join(subagents, ", "))
	}
	if subagentModel != "" {
		ui.Info("Subagent model: %s", subagentModel)
	}
	if enableEngram && engramPath != "" {
		ui.Info("Engram: %s", engramPath)
	} else {
		ui.Info("Engram: %s", i18n.T(lang, "common.no"))
	}
	ui.Info("omp extension: %v", extension)
}

func uniqueModels(models []detect.LocalModel) []detect.LocalModel {
	seen := map[string]bool{}
	var out []detect.LocalModel
	for _, model := range models {
		if seen[model.Ref()] {
			continue
		}
		seen[model.Ref()] = true
		out = append(out, model)
	}
	return out
}
