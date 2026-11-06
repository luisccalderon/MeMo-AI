package cli

import (
	"encoding/json"
	"fmt"

	"github.com/luisccalderon/MeMo-AI/internal/banner"
	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/doctor"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/opencode"
	"github.com/luisccalderon/MeMo-AI/internal/term"
)

func doctorReport(paths opencode.Paths, env detect.Environment, lang string) doctor.Report {
	return doctor.Run(paths, env, lang)
}

func runDoctor(paths opencode.Paths, env detect.Environment, lang string, opts globals) int {
	report := doctorReport(paths, env, lang)
	if opts.json {
		out, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Println(err)
			return 1
		}
		fmt.Println(string(out))
		if report.Failed() {
			return 1
		}
		return 0
	}

	ui := term.New(lang)
	ui.Title(i18n.T(lang, "doctor.title"))
	for _, check := range report.Checks {
		switch check.Status {
		case doctor.OK:
			fmt.Printf("  %s %-20s %s\n", uiColor(ui, banner.Mint, "OK"), check.Name, check.Detail)
		case doctor.Warn:
			fmt.Printf("  %s %-20s %s\n", uiColor(ui, banner.Peach, "WARN"), check.Name, check.Detail)
		case doctor.Fail:
			fmt.Printf("  %s %-20s %s\n", uiColor(ui, "\x1b[38;2;255;120;120m", "FAIL"), check.Name, check.Detail)
		default:
			fmt.Printf("  %s %-20s %s\n", uiColor(ui, banner.Muted, "SKIP"), check.Name, check.Detail)
		}
		if check.Hint != "" {
			ui.Muted("%s", check.Hint)
		}
	}
	ok, warn, fail, skip := report.Counts()
	fmt.Println()
	ui.Info("%d OK · %d warnings · %d failures · %d skipped", ok, warn, fail, skip)
	if fail > 0 {
		return 1
	}
	return 0
}

func uiColor(ui *term.UI, color, text string) string {
	if !ui.Colorize {
		return text
	}
	return color + text + banner.Reset
}
