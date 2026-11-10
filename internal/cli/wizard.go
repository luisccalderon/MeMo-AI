package cli

import (
	"os"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/engram"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
	"github.com/luisccalderon/MeMo-AI/internal/version"
)

func runWizard(paths pi.Paths, env detect.Environment, st *state.State, opts globals, lang string) int {
	ui := term.New(lang)
	ui.Banner(version.Version, env.PiVersion)

	interactive := !opts.yes && isTTY()
	if !interactive && !opts.yes && !opts.dryRun {
		ui.Error(i18n.T(lang, "error.no_tty"))
		return 2
	}

	// A config to copy requires the package and cannot be combined with the
	// flag that says "never install it". Validate the source before asking
	// anything else, so a typo never reaches the summary.
	if opts.atelierConfig != "" {
		if opts.noAtelier {
			ui.Error("%s", i18n.T(lang, "error.atelier_config_conflict"))
			return 2
		}
		if _, err := os.Stat(opts.atelierConfig); err != nil {
			ui.Error("%s: %v", i18n.T(lang, "error.atelier_config_missing"), err)
			return 2
		}
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

	// Pi owns its settings file; the only model MeMo-AI knows is the one
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

	// Subagents are not supported on Pi yet. --no-subagents and
	// --subagent-model are still accepted so existing scripts keep working.
	if opts.subagentModel != "" {
		ui.Warn("%s", i18n.T(lang, "wizard.subagents_ignored"))
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

	// Pi extension.
	installExtension := !opts.noExtension
	if interactive && !opts.noExtension {
		installExtension, _ = ui.Confirm(i18n.T(lang, "wizard.extension"), true)
	}

	// Optional free model providers. Both are community packages, so they are
	// installed only when the user opts in and never ship inside the binary.
	installFree := false
	if !opts.noFreeModels {
		switch {
		case opts.freeModels:
			installFree = true
		case interactive:
			ui.Muted("%s", i18n.T(lang, "wizard.free_models_hint"))
			installFree, _ = ui.Confirm(i18n.T(lang, "wizard.free_models"), true)
		}
	}

	// opencode-pi starts the OpenCode CLI without a shell. On Windows that fails
	// when only script launchers are on PATH, so offer to point OPENCODE_PI_BIN
	// at the real executable. An explicit --free-models is consent enough.
	openCodeBin := ""
	if installFree {
		launcher := pi.DetectOpenCodeLauncher()
		needsBin := launcher.Status == pi.LauncherNeedsBin || launcher.Status == pi.LauncherBrokenBin
		switch {
		case !needsBin:
		case launcher.Candidate == "":
			ui.Warn("%s", i18n.T(lang, "wizard.opencode_bin_missing"))
		case interactive:
			ui.Muted("%s", i18n.T(lang, "wizard.opencode_bin_hint"))
			if launcher.Configured != "" {
				ui.Muted("%s: %s", i18n.T(lang, "common.current"), launcher.Configured)
			}
			ui.Muted("%s=%s", pi.OpenCodeBinEnv, launcher.Candidate)
			if ok, _ := ui.Confirm(i18n.T(lang, "wizard.opencode_bin"), true); ok {
				openCodeBin = launcher.Candidate
			}
		case launcher.Configured == "":
			// Unattended runs never replace a value the user already chose.
			openCodeBin = launcher.Candidate
		}
	}

	// Optional status rail and activity sidebar. A community package that owns
	// Pi's custom footer and editor, so it is installed only on request and
	// never bundled with MeMo-AI. An explicit --atelier-config is consent for
	// both the package and the file it carries.
	installAtelier := false
	if !opts.noAtelier {
		switch {
		case opts.atelier || opts.atelierConfig != "":
			installAtelier = true
		case interactive:
			ui.Muted("%s", i18n.T(lang, "wizard.atelier_hint"))
			installAtelier, _ = ui.Confirm(i18n.T(lang, "wizard.atelier"), false)
		}
	}

	// Quiet startup: Pi keeps its version header and hides the resource
	// listing; the extension lists the prompt templates next to the mark.
	quietStartup := false
	if !opts.noQuietStartup {
		switch {
		case opts.quietStartup:
			quietStartup = true
		case interactive:
			ui.Muted("%s", i18n.T(lang, "wizard.quiet_startup_hint"))
			quietStartup, _ = ui.Confirm(i18n.T(lang, "wizard.quiet_startup"), true)
		}
	}

	// Summary.
	if interactive {
		printSummary(ui, lang, orchestrator, model, engramPath, enableEngram, installExtension, installFree, installAtelier, openCodeBin, quietStartup)
		if installAtelier {
			source := i18n.T(lang, "wizard.atelier_config_preset")
			if opts.atelierConfig != "" {
				source = opts.atelierConfig
			}
			ui.Info("pi-atelier config: %s", source)
		}
		ok, err := ui.Confirm(i18n.T(lang, "wizard.confirm"), true)
		if err != nil || !ok {
			ui.Muted(i18n.T(lang, "wizard.cancelled"))
			return 0
		}
	}

	ui.Title(i18n.T(lang, "wizard.installing"))
	result, err := install.Run(paths, install.Options{
		Lang:              lang,
		Orchestrator:      orchestrator,
		PrimaryModel:      model,
		EnableEngram:      enableEngram && engramPath != "",
		EngramPath:        engramPath,
		InstallExtension:  installExtension,
		QuietStartup:      quietStartup,
		AtelierConfig:     installAtelier && opts.atelierConfig == "",
		AtelierConfigFile: opts.atelierConfig,
		DryRun:            opts.dryRun,
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
		if openCodeBin != "" {
			ui.Info("would set %s=%s", pi.OpenCodeBinEnv, openCodeBin)
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
	if result.State != nil && result.State.QuietStartup && quietStartup {
		ui.Success("%s", i18n.T(lang, "install.quiet_startup_ready"))
	}
	for _, ext := range pi.FreeModelExtensions {
		if !installFree {
			continue
		}
		if pi.HasExtension(paths.SettingsFile, ext.Source) {
			ui.Muted("%s: %s", i18n.T(lang, "install.free_models_present"), ext.Source)
			continue
		}
		ui.Muted("%s: %s", i18n.T(lang, "install.free_models_installing"), ext.Source)
		if err := pi.InstallExtension(backgroundContext(), env.PiPath, ext.Source, ui.Out); err != nil {
			ui.Warn("%s: %v", i18n.T(lang, "install.free_models_failed"), err)
			continue
		}
		ui.Success("%s: %s", i18n.T(lang, "install.free_models_ready"), ext.Source)
	}
	if installAtelier {
		switch {
		case pi.HasExtension(paths.SettingsFile, pi.AtelierSource):
			ui.Muted("%s: %s", i18n.T(lang, "install.atelier_present"), pi.AtelierSource)
		default:
			ui.Muted("%s: %s", i18n.T(lang, "install.atelier_installing"), pi.AtelierSource)
			if err := pi.InstallExtension(backgroundContext(), env.PiPath, pi.AtelierSource, ui.Out); err != nil {
				ui.Warn("%s: %v", i18n.T(lang, "install.atelier_failed"), err)
			} else {
				ui.Success("%s: %s", i18n.T(lang, "install.atelier_ready"), pi.AtelierSource)
			}
		}
		if result.AtelierConfig {
			if opts.atelierConfig != "" {
				ui.Success("%s: %s", i18n.T(lang, "install.atelier_config_from"), opts.atelierConfig)
			} else {
				ui.Success("%s", i18n.T(lang, "install.atelier_config_ready"))
			}
		} else if pi.Exists(paths.AtelierFile) {
			ui.Muted("%s", i18n.T(lang, "install.atelier_config_kept"))
		}
	}
	// The variable only makes sense once the bridge is really installed.
	if openCodeBin != "" && pi.HasExtension(paths.SettingsFile, pi.OpenCodePiSource) {
		if err := pi.SetUserEnv(pi.OpenCodeBinEnv, openCodeBin); err != nil {
			ui.Warn("%s: %v", i18n.T(lang, "install.opencode_bin_failed"), err)
		} else {
			// Let the health check below see it without re-reading the registry.
			_ = os.Setenv(pi.OpenCodeBinEnv, openCodeBin)
			ui.Success("%s: %s=%s", i18n.T(lang, "install.opencode_bin_ready"), pi.OpenCodeBinEnv, openCodeBin)
			ui.Muted("%s", i18n.T(lang, "install.opencode_bin_restart"))
		}
	}
	ui.Muted("%s", i18n.T(lang, "install.subagents_off"))

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
	if env.PiPath != "" {
		ui.Info("%s: %s (%s)", i18n.T(lang, "wizard.pi"), env.PiVersion, env.PiPath)
	} else {
		ui.Warn("%s", i18n.T(lang, "error.pi_missing"))
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
	options = append(options, term.Option{Label: i18n.T(lang, "common.inherit"), Description: "Leave Pi's default model untouched"})
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

func confirmInstallEngram(ui *term.UI, lang string) bool {
	ui.Muted("%s", i18n.T(lang, "wizard.engram_missing"))
	ok, _ := ui.Confirm(i18n.T(lang, "wizard.engram_install"), true)
	return ok
}

func printSummary(ui *term.UI, lang string, orchestrator bool, model, engramPath string, enableEngram, extension, freeModels, atelier bool, openCodeBin string, quietStartup bool) {
	ui.Title(i18n.T(lang, "wizard.confirm"))
	ui.Info("Orchestrator prompt: %v", orchestrator)
	if model == "" {
		ui.Info("Model: %s", i18n.T(lang, "common.inherit"))
	} else {
		ui.Info("Model: %s", model)
	}
	ui.Info("Subagents: %s", i18n.T(lang, "agents.subagents_unsupported"))
	if enableEngram && engramPath != "" {
		ui.Info("Engram: %s", engramPath)
	} else {
		ui.Info("Engram: %s", i18n.T(lang, "common.no"))
	}
	ui.Info("Pi extension: %v", extension)
	ui.Info("Free models (pi-free, opencode-pi): %v", freeModels)
	ui.Info("Status rail (pi-atelier): %v", atelier)
	ui.Info("Quiet startup: %v", quietStartup)
	if openCodeBin != "" {
		ui.Info("%s: %s", pi.OpenCodeBinEnv, openCodeBin)
	}
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
