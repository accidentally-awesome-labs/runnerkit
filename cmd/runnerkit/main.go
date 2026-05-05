package main

import (
	"os"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/cli"
	"github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

var version = "dev"

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

// buildDependencies constructs the production-binary Dependencies
// struct. Extracted from main() so cmd/runnerkit/main_test.go can
// regression-guard the wiring (Bug 4 / Task G — Plans 06-06 + 06-08
// shipped Path B + Path C prompt code paths but never wired Prompts).
func buildDependencies() cli.Dependencies {
	return cli.Dependencies{
		Version: version,
		In:      os.Stdin,
		Out:     os.Stdout,
		Err:     os.Stderr,
		TTY: ui.TerminalCapabilities{
			StdinTTY:  isTerminal(os.Stdin),
			StdoutTTY: isTerminal(os.Stdout),
			Color:     true,
			Width:     80,
		},
		Clock:         time.Now,
		CommandRunner: github.OSCommandRunner{},
		Prompts:       ui.NewCLIPrompter(os.Stdin, os.Stdout),
	}
}

func main() {
	cmd := cli.NewRootCommand(buildDependencies())
	if err := cmd.Execute(); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
