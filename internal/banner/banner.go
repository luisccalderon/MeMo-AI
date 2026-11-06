// Package banner renders the MeMo-AI terminal wordmark using the Vesper palette.
package banner

import (
	"fmt"
	"io"
)

// Vesper palette shared with the OpenCode TUI plugin.
const (
	White = "\x1b[38;2;255;255;255m"
	Cream = "\x1b[38;2;255;224;200m"
	Peach = "\x1b[38;2;255;199;153m"
	Mint  = "\x1b[38;2;153;255;228m"
	Muted = "\x1b[38;2;160;160;160m"
	Rule  = "\x1b[38;2;80;80;80m"
	Bold  = "\x1b[1m"
	Reset = "\x1b[0m"
)

var letters = []string{
	"███╗   ███╗ ███████╗ ███╗   ███╗  ██████╗ ",
	"████╗ ████║ ██╔════╝ ████╗ ████║ ██╔═══██╗",
	"██╔████╔██║ █████╗   ██╔████╔██║ ██║   ██║",
	"██║╚██╔╝██║ ██╔══╝   ██║╚██╔╝██║ ██║   ██║",
	"██║ ╚═╝ ██║ ███████╗ ██║ ╚═╝ ██║ ╚██████╔╝",
	"╚═╝     ╚═╝ ╚══════╝ ╚═╝     ╚═╝  ╚═════╝ ",
}

var shades = []string{White, Cream, Peach, Peach, Mint, Mint}

// Print writes the MeMo-AI wordmark, tagline and version line.
func Print(w io.Writer, memoVersion, openCodeVersion string) {
	oc := "not detected"
	if openCodeVersion != "" {
		oc = "OpenCode " + openCodeVersion
	}
	fmt.Fprintln(w)
	for i, line := range letters {
		fmt.Fprintf(w, "  %s%s%s\n", shades[i], line, Reset)
	}
	fmt.Fprintf(w, "  %s%s%s\n", Rule, "────────────────────────────────────────────────────────────", Reset)
	fmt.Fprintf(w, "  %sMEMORY%s %s·%s %sORCHESTRATION%s %s·%s %sEVIDENCE%s\n",
		Bold+Peach, Reset, Rule, Reset, Bold+Mint, Reset, Rule, Reset, Bold+Cream, Reset)
	fmt.Fprintf(w, "  %sv%s · %s%s\n\n", Muted, memoVersion, oc, Reset)
}
