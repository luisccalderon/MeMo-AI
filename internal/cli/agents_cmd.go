package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
)

func runAgents(paths omp.Paths, env detect.Environment, st *state.State, lang string) int {
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
	if len(st.Subagents) > 0 {
		ui.Info("%s: %s", i18n.T(lang, "agents.subagents"), strings.Join(st.Subagents, ", "))
	} else if st.InstalledAt != "" {
		ui.Info("%s: %s", i18n.T(lang, "agents.subagents"), i18n.T(lang, "common.none"))
	}

	ui.Title(i18n.T(lang, "agents.available"))
	agents := detect.Agents(paths)
	sort.SliceStable(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	for _, agent := range agents {
		marker := " "
		for _, enabled := range st.Subagents {
			if agent.ID == enabled {
				marker = "*"
			}
		}
		fmt.Printf("  %s %-22s %-10s %s\n", marker, agent.ID, agent.Mode, agent.Description)
	}

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
