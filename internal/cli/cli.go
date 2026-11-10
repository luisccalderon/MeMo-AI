// Package cli implements the memo-ai command line interface.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/banner"
	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
	"github.com/luisccalderon/MeMo-AI/internal/state"
	"github.com/luisccalderon/MeMo-AI/internal/version"
)

type globals struct {
	lang           string
	model          string
	subagentModel  string
	yes            bool
	dryRun         bool
	noSubagents    bool
	noEngram       bool
	noOrchestrator bool
	noExtension    bool
	installEngram  bool
	force          bool
	json           bool
	restore        bool
	help           bool
	version        bool
}

type invocation struct {
	command string
	args    []string
	opts    globals
}

// Run executes the CLI and returns a process exit code.
func Run(args []string) int {
	inv, err := parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memo-ai: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run `memo-ai help` for usage.")
		return 2
	}

	if inv.opts.version || inv.command == "version" {
		fmt.Printf("memo-ai %s (%s)\n", version.Version, version.Channel)
		return 0
	}
	if inv.opts.help || inv.command == "help" {
		printHelp(inv.opts.lang)
		return 0
	}

	paths, err := pi.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "memo-ai: cannot resolve Pi paths: %v\n", err)
		return 1
	}
	st, _ := state.Load(paths.StateFile)
	lang := resolveLang(inv.opts.lang, st.Language)

	env := detect.Detect(backgroundContext())

	switch inv.command {
	case "", "install", "configure":
		return runWizard(paths, env, st, inv.opts, lang)
	case "doctor":
		return runDoctor(paths, env, lang, inv.opts)
	case "agents":
		return runAgents(paths, env, st, lang)
	case "uninstall":
		return runUninstall(paths, st, lang, inv.opts)
	case "engram":
		return runEngram(paths, env, st, lang, inv.opts, inv.args)
	default:
		fmt.Fprintf(os.Stderr, "memo-ai: %s\n", i18n.Tf(lang, "error.unknown_command", inv.command))
		printHelp(lang)
		return 2
	}
}

func resolveLang(flagValue, stored string) string {
	if flagValue != "" {
		return i18n.Normalize(flagValue)
	}
	if stored != "" {
		return i18n.Normalize(stored)
	}
	if env := os.Getenv("MEMO_AI_LANG"); env != "" {
		return i18n.Normalize(env)
	}
	if lang := os.Getenv("LANG"); strings.HasPrefix(strings.ToLower(lang), "es") {
		return i18n.Spanish
	}
	return i18n.Spanish
}

func parse(args []string) (invocation, error) {
	var inv invocation
	valueFlags := map[string]*string{
		"lang":           &inv.opts.lang,
		"model":          &inv.opts.model,
		"subagent-model": &inv.opts.subagentModel,
	}
	boolFlags := map[string]*bool{
		"yes":             &inv.opts.yes,
		"dry-run":         &inv.opts.dryRun,
		"no-subagents":    &inv.opts.noSubagents,
		"no-engram":       &inv.opts.noEngram,
		"no-orchestrator": &inv.opts.noOrchestrator,
		"no-extension":    &inv.opts.noExtension,
		"install-engram":  &inv.opts.installEngram,
		"force":           &inv.opts.force,
		"json":            &inv.opts.json,
		"restore":         &inv.opts.restore,
		"help":            &inv.opts.help,
		"version":         &inv.opts.version,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h":
			inv.opts.help = true
			continue
		case "-v":
			inv.opts.version = true
			continue
		}
		if !strings.HasPrefix(arg, "--") {
			if inv.command == "" {
				inv.command = arg
			} else {
				inv.args = append(inv.args, arg)
			}
			continue
		}

		name := strings.TrimPrefix(arg, "--")
		value := ""
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			name, value = name[:idx], name[idx+1:]
		}
		if target, ok := valueFlags[name]; ok {
			if value == "" {
				if i+1 >= len(args) {
					return inv, fmt.Errorf("flag --%s needs a value", name)
				}
				i++
				value = args[i]
			}
			*target = value
			continue
		}
		if target, ok := boolFlags[name]; ok {
			*target = true
			continue
		}
		return inv, fmt.Errorf("unknown flag --%s", name)
	}
	return inv, nil
}

func printHelp(lang string) {
	ui := newUI(lang)
	ui.Banner(version.Version, "")
	fmt.Println(i18n.T(lang, "help.intro"))
	fmt.Println()
	fmt.Println(i18n.T(lang, "help.usage"))
	fmt.Println()
	fmt.Printf("%s\n", i18n.T(lang, "help.commands"))
	for _, row := range [][2]string{
		{"install", i18n.T(lang, "help.cmd.install")},
		{"configure", i18n.T(lang, "help.cmd.configure")},
		{"doctor", i18n.T(lang, "help.cmd.doctor")},
		{"agents", i18n.T(lang, "help.cmd.agents")},
		{"engram [status|install]", i18n.T(lang, "help.cmd.engram")},
		{"uninstall", i18n.T(lang, "help.cmd.uninstall")},
		{"version", i18n.T(lang, "help.cmd.version")},
	} {
		fmt.Printf("  %-24s %s\n", row[0], row[1])
	}
	fmt.Println()
	fmt.Println(i18n.T(lang, "help.flags"))
	for _, row := range [][2]string{
		{"--lang es|en", i18n.T(lang, "help.flag.lang")},
		{"--yes", i18n.T(lang, "help.flag.yes")},
		{"--dry-run", i18n.T(lang, "help.flag.dry-run")},
		{"--model <provider/model>", i18n.T(lang, "help.flag.model")},
		{"--no-orchestrator", i18n.T(lang, "help.flag.no-orchestrator")},
		{"--no-engram", i18n.T(lang, "help.flag.no-engram")},
		{"--no-extension", i18n.T(lang, "help.flag.no-extension")},
	} {
		fmt.Printf("  %-28s %s\n", row[0], row[1])
	}
	fmt.Println()
	fmt.Println(i18n.T(lang, "help.examples"))
	fmt.Println("  memo-ai")
	fmt.Println("  memo-ai install --lang es --model ollama/qwen3:8b")
	fmt.Println("  memo-ai install --yes --no-engram")
	fmt.Println("  memo-ai doctor")
	fmt.Println()
	fmt.Printf("%s %s\n", banner.Muted, i18n.T(lang, "app.tagline"))
}
