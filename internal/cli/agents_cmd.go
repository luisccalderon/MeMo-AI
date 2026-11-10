package cli

import (
	"fmt"

	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
)

// runAgents reports the MeMo-AI configuration. Pi has no built-in subagents,
// so the orchestrator always works inline and no agent files are installed.
func runAgents(_ pi.Paths, env detect.Environment, st *state.State, lang string) int {
	ui := term.New(lang)
	ui.Title(i18n.T(lang, "agents.title"))

	if st.InstalledAt == "" {
		ui.Muted("%s", i18n.T(lang, "agents.no_config"))
	} else {
		ui.Info("%s: %v", i18n.T(lang, "agents.primary"), st.Orchestrator)
		if st.PrimaryModel != "" {
			ui.Info("Model: %s", st.PrimaryModel)
		}
	}
	ui.Info("%s: %s", i18n.T(lang, "agents.subagents"), i18n.T(lang, "agents.subagents_unsupported"))

	ui.Title(i18n.T(lang, "agents.local"))
	if len(env.Models) == 0 {
		ui.Muted("%s", i18n.T(lang, "wizard.no_local_models"))
	} else {
		seen := map[string]bool{}
		for _, m := range env.Models {
			if seen[m.Ref()] {
				continue
			}
			seen[m.Ref()] = true
			detail := m.Detail
			if detail == "" {
				detail = m.Endpoint
			}
			fmt.Printf("  %-40s %s\n", m.Ref(), detail)
		}
	}
	return 0
}
