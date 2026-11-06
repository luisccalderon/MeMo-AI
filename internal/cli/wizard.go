package cli

import (
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/engram"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/opencode"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
	"github.com/luisccalderon/MeMo-AI/internal/version"
)

func runWizard(paths opencode.Paths, env detect.Environment, st *state.State, opts globals, lang string) int {
	ui := term.New(lang)
	ui.Banner(version.Version, env.OpenCodeVersion)

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

	doc, err := config.Load(paths.ConfigFile)
	if err != nil {
		ui.Error(i18n.Tf(lang, "error.config_parse", err.Error()))
		return 1
	}
	currentAgent := config.GetString(doc.Data, "default_agent")
	currentModel := config.GetString(doc.Data, "model")

	// Primary agent.
	agentID := opts.agent
	if agentID == "" {
		agentID = chooseAgent(ui, paths, st, currentAgent, interactive, lang)
	}

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

	// TUI plugin.
	installPlugin := !opts.noPlugin
	if interactive && !opts.noPlugin {
		installPlugin, _ = ui.Confirm(i18n.T(lang, "wizard.plugin"), true)
	}

	// Summary.
	if interactive {
		printSummary(ui, lang, agentID, model, subagents, subagentModel, engramPath, enableEngram, installPlugin)
		ok, err := ui.Confirm(i18n.T(lang, "wizard.confirm"), true)
		if err != nil || !ok {
			ui.Muted(i18n.T(lang, "wizard.cancelled"))
			return 0
		}
	}

	ui.Title(i18n.T(lang, "wizard.installing"))
	result, err := install.Run(paths, install.Options{
		Lang:          lang,
		PrimaryAgent:  agentID,
		PrimaryModel:  model,
		SubagentModel: subagentModel,
		Subagents:     subagents,
		EnableEngram:  enableEngram && engramPath != "",
		EngramPath:    engramPath,
		InstallPlugin: installPlugin,
		DryRun:        opts.dryRun,
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
	ui.Success("%s: %s", i18n.T(lang, "install.config_updated"), result.ConfigPath)
	ui.Success("%s: %d", i18n.T(lang, "install.files_written"), len(result.Writes))
	if enableEngram && engramPath != "" {
		ui.Success("%s: %s", i18n.T(lang, "install.engram_ready"), engramPath)
	} else {
		ui.Muted("%s", i18n.T(lang, "install.engram_skipped"))
	}
	if installPlugin {
		ui.Success("%s: %s", i18n.T(lang, "install.plugin_ready"), result.PluginDir)
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
	if env.OpenCodePath != "" {
		ui.Info("%s: %s (%s)", i18n.T(lang, "wizard.opencode"), env.OpenCodeVersion, env.OpenCodePath)
	} else {
		ui.Warn("%s: %s", i18n.T(lang, "wizard.opencode"), i18n.T(lang, "engram.missing"))
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

func chooseAgent(ui *term.UI, paths opencode.Paths, st *state.State, current string, interactive bool, lang string) string {
	if !interactive {
		if st.PrimaryAgent != "" {
			return st.PrimaryAgent
		}
		if current != "" {
			return current
		}
		return "memo-orchestrator"
	}

	agents := detect.Agents(paths)
	var ordered []detect.Agent
	seen := map[string]bool{}
	add := func(agent detect.Agent) {
		if agent.ID == "" || seen[agent.ID] {
			return
		}
		seen[agent.ID] = true
		ordered = append(ordered, agent)
	}
	add(detect.Agent{
		ID:          "memo-orchestrator",
		Name:        "MeMo Orchestrator",
		Description: "Memory-first orchestrator with Engram and optional subagents",
		Mode:        "primary",
	})
	for _, agent := range agents {
		if agent.Mode == "primary" || agent.Mode == "all" || agent.Mode == "" {
			add(agent)
		}
	}

	options := make([]term.Option, 0, len(ordered))
	defaultIdx := 0
	for i, agent := range ordered {
		label := agent.ID
		if agent.ID == "memo-orchestrator" {
			label += "  (" + i18n.T(lang, "common.recommended") + ")"
		}
		options = append(options, term.Option{Label: label, Description: agent.Description})
		if agent.ID == st.PrimaryAgent || agent.ID == current {
			defaultIdx = i
		}
	}
	idx, err := ui.Select(i18n.T(lang, "wizard.primary_agent"), options, defaultIdx)
	if err != nil {
		return "memo-orchestrator"
	}
	return ordered[idx].ID
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
	options = append(options, term.Option{Label: i18n.T(lang, "common.inherit"), Description: "No root model override"})
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
	options = append(options, term.Option{Label: i18n.T(lang, "common.inherit"), Description: "Use the primary agent model"})
	values := []string{""}
	if primaryModel != "" {
		options = append(options, term.Option{Label: primaryModel, Description: "Same as the primary agent"})
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

func printSummary(ui *term.UI, lang, agent, model string, subagents []string, subagentModel, engramPath string, enableEngram, plugin bool) {
	ui.Title(i18n.T(lang, "wizard.confirm"))
	ui.Info("Primary agent: %s", agent)
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
	ui.Info("TUI plugin: %v", plugin)
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
