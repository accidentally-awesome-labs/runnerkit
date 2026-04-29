package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/salar/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

// Dependencies are injectable command dependencies used by tests and main.
type Dependencies struct {
	Version string
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	TTY     ui.TerminalCapabilities
	Prompts ui.Prompter
	Clock   func() time.Time
}

func normalizeDependencies(deps Dependencies) Dependencies {
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if deps.In == nil {
		deps.In = io.Reader(nil)
	}
	if deps.Out == nil {
		deps.Out = io.Discard
	}
	if deps.Err == nil {
		deps.Err = io.Discard
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	if deps.TTY.Width == 0 {
		deps.TTY.Width = 80
	}
	return deps
}

// NewRootCommand constructs the runnerkit command tree.
func NewRootCommand(deps Dependencies) *cobra.Command {
	deps = normalizeDependencies(deps)

	var jsonOutput bool
	var noColor bool

	root := &cobra.Command{Use: "runnerkit"}
	root.Short = "Prepare and manage GitHub Actions self-hosted runners"
	root.Long = "RunnerKit prepares and manages GitHub Actions self-hosted runners from a CLI-first workflow."
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetIn(deps.In)
	root.SetOut(deps.Out)
	root.SetErr(deps.Err)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return NewExitError(ExitInvalidInput, err)
	})

	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "write machine-readable JSON to stdout")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable ANSI color output")

	root.AddCommand(newVersionCommand(deps, &jsonOutput))
	root.AddCommand(newUpPlaceholderCommand())

	_ = noColor // wired for downstream renderer support.
	return root
}

func newVersionCommand(deps Dependencies, jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "version"}
	cmd.Short = "Print version information"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		if *jsonOutput {
			payload := map[string]any{
				"ok":                 true,
				"command":            "version",
				"version":            deps.Version,
				"redactions_applied": true,
			}
			enc := json.NewEncoder(deps.Out)
			enc.SetEscapeHTML(false)
			return enc.Encode(payload)
		}
		_, err := fmt.Fprintf(deps.Out, "RunnerKit %s\n", deps.Version)
		return err
	}
	return cmd
}

func newUpPlaceholderCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Prepare foundation",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Phase 1 does not install a runner yet.")
			return err
		},
	}
}
