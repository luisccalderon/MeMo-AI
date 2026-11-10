// Package term contains the dependency-free terminal UI used by the installer.
package term

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/luisccalderon/MeMo-AI/internal/banner"
	"github.com/luisccalderon/MeMo-AI/internal/i18n"
)

// Option is one choice in a Select or MultiSelect prompt.
type Option struct {
	Label       string
	Description string
	Value       string
}

// UI writes prompts to Out and reads answers from In.
type UI struct {
	In       *bufio.Reader
	Out      io.Writer
	Lang     string
	Colorize bool
}

// New returns a UI bound to stdin/stdout.
func New(lang string) *UI {
	return &UI{
		In:       bufio.NewReader(os.Stdin),
		Out:      os.Stdout,
		Lang:     i18n.Normalize(lang),
		Colorize: colorEnabled(os.Stdout),
	}
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (u *UI) paint(code, text string) string {
	if !u.Colorize {
		return text
	}
	return code + text + banner.Reset
}

// Title prints a section title.
func (u *UI) Title(text string) {
	fmt.Fprintf(u.Out, "\n%s\n", u.paint(banner.Bold+banner.Peach, text))
}

// Rule prints a horizontal rule.
func (u *UI) Rule() {
	fmt.Fprintln(u.Out, u.paint(banner.Rule, strings.Repeat("─", 64)))
}

// Info prints a neutral line.
func (u *UI) Info(format string, args ...any) {
	fmt.Fprintf(u.Out, "  "+format+"\n", args...)
}

// Muted prints a dim line.
func (u *UI) Muted(format string, args ...any) {
	fmt.Fprintf(u.Out, "  %s\n", u.paint(banner.Muted, fmt.Sprintf(format, args...)))
}

// Success prints a mint line.
func (u *UI) Success(format string, args ...any) {
	fmt.Fprintf(u.Out, "  %s %s\n", u.paint(banner.Mint, "✓"), fmt.Sprintf(format, args...))
}

// Warn prints a peach warning line.
func (u *UI) Warn(format string, args ...any) {
	fmt.Fprintf(u.Out, "  %s %s\n", u.paint(banner.Peach, "!"), fmt.Sprintf(format, args...))
}

// Error prints a red-ish error line.
func (u *UI) Error(format string, args ...any) {
	fmt.Fprintf(u.Out, "  %s %s\n", u.paint("\x1b[38;2;255;120;120m", "✗"), fmt.Sprintf(format, args...))
}

// Banner prints the wordmark.
func (u *UI) Banner(memoVersion, piVersion string) {
	banner.Print(u.Out, memoVersion, piVersion)
}

func (u *UI) readLine() (string, error) {
	line, err := u.In.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Select shows a numbered list and returns the chosen index.
func (u *UI) Select(prompt string, options []Option, defaultIdx int) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options")
	}
	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}
	fmt.Fprintf(u.Out, "\n%s %s\n", u.paint(banner.Bold+banner.Cream, "›"), u.paint(banner.Bold, prompt))
	for i, opt := range options {
		marker := " "
		if i == defaultIdx {
			marker = u.paint(banner.Mint, "●")
		}
		fmt.Fprintf(u.Out, "  %s %s %s\n", marker, u.paint(banner.Peach, fmt.Sprintf("%d)", i+1)), opt.Label)
		if opt.Description != "" {
			fmt.Fprintf(u.Out, "        %s\n", u.paint(banner.Muted, opt.Description))
		}
	}
	for {
		fmt.Fprintf(u.Out, "  %s [1-%d] (%s %d): ", u.paint(banner.Muted, "→"), len(options), i18n.T(u.Lang, "common.default"), defaultIdx+1)
		line, err := u.readLine()
		if err != nil {
			return defaultIdx, err
		}
		if line == "" {
			return defaultIdx, nil
		}
		n, err := strconv.Atoi(line)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		u.Warn("Opción inválida / Invalid option")
	}
}

// MultiSelect shows a numbered list and returns the selected indices.
func (u *UI) MultiSelect(prompt string, options []Option, defaults []int) ([]int, error) {
	if len(options) == 0 {
		return nil, nil
	}
	fmt.Fprintf(u.Out, "\n%s %s\n", u.paint(banner.Bold+banner.Cream, "›"), u.paint(banner.Bold, prompt))
	for i, opt := range options {
		marker := "○"
		for _, d := range defaults {
			if d == i {
				marker = u.paint(banner.Mint, "●")
				break
			}
		}
		fmt.Fprintf(u.Out, "  %s %s %s\n", marker, u.paint(banner.Peach, fmt.Sprintf("%d)", i+1)), opt.Label)
		if opt.Description != "" {
			fmt.Fprintf(u.Out, "        %s\n", u.paint(banner.Muted, opt.Description))
		}
	}
	defaultText := "all"
	if len(defaults) == 0 {
		defaultText = "none"
	}
	for {
		fmt.Fprintf(u.Out, "  %s %s [1-%d] (%s: %s): ", u.paint(banner.Muted, "→"),
			u.paint(banner.Muted, "1,2,3"), len(options), i18n.T(u.Lang, "common.default"), defaultText)
		line, err := u.readLine()
		if err != nil {
			if line == "" {
				return append([]int(nil), defaults...), err
			}
		}
		if line == "" {
			return append([]int(nil), defaults...), nil
		}
		if strings.EqualFold(line, "none") || strings.EqualFold(line, "ninguno") || line == "0" {
			return nil, nil
		}
		if strings.EqualFold(line, "all") || strings.EqualFold(line, "todos") {
			out := make([]int, len(options))
			for i := range options {
				out[i] = i
			}
			return out, nil
		}
		var out []int
		valid := true
		for _, part := range strings.Split(line, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 || n > len(options) {
				valid = false
				break
			}
			out = append(out, n-1)
		}
		if valid {
			return out, nil
		}
		u.Warn("Selección inválida / Invalid selection")
	}
}

// Confirm asks a yes/no question.
func (u *UI) Confirm(prompt string, def bool) (bool, error) {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		fmt.Fprintf(u.Out, "\n%s %s [%s]: ", u.paint(banner.Bold+banner.Cream, "?"), prompt, u.paint(banner.Muted, hint))
		line, err := u.readLine()
		if err != nil {
			return def, err
		}
		switch strings.ToLower(line) {
		case "":
			return def, nil
		case "y", "yes", "s", "si", "sí":
			return true, nil
		case "n", "no":
			return false, nil
		}
		u.Warn("Responde sí o no / Answer yes or no")
	}
}

// Input asks for free text with an optional default.
func (u *UI) Input(prompt, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(u.Out, "\n%s %s [%s]: ", u.paint(banner.Bold+banner.Cream, "?"), prompt, u.paint(banner.Muted, def))
	} else {
		fmt.Fprintf(u.Out, "\n%s %s: ", u.paint(banner.Bold+banner.Cream, "?"), prompt)
	}
	line, err := u.readLine()
	if err != nil && line == "" {
		return def, err
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}
