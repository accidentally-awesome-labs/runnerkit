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

func main() {
	cmd := cli.NewRootCommand(cli.Dependencies{
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
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
