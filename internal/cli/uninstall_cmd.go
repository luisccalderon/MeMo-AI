package cli

import (
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/omp"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/term"
	"github.com/luisccalderon/MeMo-AI/internal/uninstall"
)

func runUninstall(paths omp.Paths, st *state.State, lang string, opts globals) int {
	ui := term.New(lang)
	ui.Title(i18n.T(lang, "uninstall.title"))

	restore := st.Config.Backup != "" || opts.restore
	if !opts.yes && isTTY() {
		ok, _ := ui.Confirm(i18n.T(lang, "uninstall.confirm"), true)
		if !ok {
			ui.Muted(i18n.T(lang, "wizard.cancelled"))
			return 0
		}
		if st.Config.Backup != "" {
			restore, _ = ui.Confirm(i18n.T(lang, "uninstall.restore"), true)
		}
	}

	result, err := uninstall.Run(paths, uninstall.Options{
		Force:         opts.force,
		RestoreConfig: restore,
	})
	if err != nil {
		ui.Error(err.Error())
		return 1
	}
	for _, removed := range result.Removed {
		ui.Info("removed %s", removed)
	}
	for _, skipped := range result.Skipped {
		ui.Warn("%s: %s", i18n.T(lang, "uninstall.skipped"), skipped)
	}
	for _, warning := range result.Warnings {
		ui.Warn("%s", warning)
	}
	if result.Restored != "" {
		ui.Success("%s: %s", i18n.T(lang, "uninstall.restored"), result.Restored)
	}
	ui.Title(i18n.T(lang, "uninstall.done"))
	return 0
}
