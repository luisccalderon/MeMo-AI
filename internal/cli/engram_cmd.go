package cli

import (
	"github.com/luisccalderon/MeMo-AI/internal/config"
	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/engram"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
)

func runEngram(paths omp.Paths, env detect.Environment, st *state.State, lang string, opts globals, args []string) int {
	ui := term.New(lang)
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}

	switch action {
	case "install":
		if env.EngramPath != "" {
			ui.Success("%s: %s", i18n.T(lang, "engram.installed"), env.EngramPath)
			return 0
		}
		ui.Title(i18n.T(lang, "engram.installing"))
		path, err := engram.Install(backgroundContext(), ui.Out)
		if err != nil {
			ui.Error(err.Error())
			return 1
		}
		ui.Success("%s: %s", i18n.T(lang, "engram.installed_ok"), path)

		if st.InstalledAt == "" {
			ui.Info("%s", i18n.T(lang, "engram.configure"))
			return 0
		}
		if !opts.yes && isTTY() {
			ok, _ := ui.Confirm(i18n.T(lang, "wizard.engram_enable"), true)
			if !ok {
				return 0
			}
		}
		result, err := install.Run(paths, install.Options{
			Lang:             lang,
			Orchestrator:     st.Orchestrator,
			PrimaryModel:     st.PrimaryModel,
			SubagentModel:    st.SubagentModel,
			Subagents:        st.Subagents,
			EnableEngram:     true,
			EngramPath:       path,
			InstallExtension: st.Extension,
			SetModel:         install.OmpModelSetter(env.OmpPath),
		})
		if err != nil {
			ui.Error(err.Error())
			return 1
		}
		for _, warning := range result.Warnings {
			ui.Warn("%s", warning)
		}
		ui.Success("%s: %s", i18n.T(lang, "install.engram_ready"), path)
		return 0

	case "status", "":
		ui.Title(i18n.T(lang, "engram.status"))
		if env.EngramPath == "" {
			ui.Warn("%s", i18n.T(lang, "engram.missing"))
			ui.Muted("%s", i18n.T(lang, "engram.configure"))
			return 1
		}
		ui.Info("%s: %s", i18n.T(lang, "engram.installed"), env.EngramPath)
		if env.EngramVersion != "" {
			ui.Info("%s: %s", i18n.T(lang, "engram.version"), env.EngramVersion)
		}
		doc, err := config.Load(paths.MCPFile)
		if err != nil {
			ui.Warn("%s", i18n.Tf(lang, "error.config_parse", err.Error()))
			return 1
		}
		if command := config.GetString(doc.Data, "mcpServers", "engram", "command"); command != "" {
			ui.Info("MCP: %s (%s)", command, paths.MCPFile)
		} else {
			ui.Muted("%s", i18n.T(lang, "engram.configure"))
		}
		return 0

	default:
		ui.Error("unknown engram action: %s", action)
		return 2
	}
}
