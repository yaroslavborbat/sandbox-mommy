package main

import (
	"os"

	"github.com/fatih/color"

	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox"
)

func main() {
	if err := sandbox.NewSandboxCommand().Execute(); err != nil {
		red := color.New(color.FgRed)
		_, _ = red.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
