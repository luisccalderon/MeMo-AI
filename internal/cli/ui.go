package cli

import (
	"context"
	"os"

	"github.com/luis/memo-ai/internal/term"
)

func newUI(lang string) *term.UI {
	return term.New(lang)
}

func backgroundContext() context.Context {
	return context.Background()
}

func isTTY() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
