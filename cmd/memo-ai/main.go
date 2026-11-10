// Command memo-ai installs the MeMo-AI coding-agent stack on top of Pi.
package main

import (
	"os"

	"github.com/luisccalderon/MeMo-AI/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
