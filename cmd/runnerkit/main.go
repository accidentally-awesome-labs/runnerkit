package main

import (
	"os"
	"time"

	"github.com/salar/runnerkit/internal/cli"
)

var version = "dev"

func main() {
	cmd := cli.NewRootCommand(cli.Dependencies{
		Version: version,
		In:      os.Stdin,
		Out:     os.Stdout,
		Err:     os.Stderr,
		Clock:   time.Now,
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
