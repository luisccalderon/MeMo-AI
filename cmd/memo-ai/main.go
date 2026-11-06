// Command memo-ai configures a memory-first orchestration stack for OpenCode.
package main

import (
	"os"

	"github.com/luis/memo-ai/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
