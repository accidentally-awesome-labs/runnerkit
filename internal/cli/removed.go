package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

const byoPrepareRemovedMessage = "byo-prepare was removed in v1.0.8. The v1.3.3 release notes that mention it are wrong. See the README 'Known issues' section."

// newRemovedByoPrepareCommand is a hidden tombstone for `runnerkit
// byo-prepare`. The command was deleted in v1.0.8 but later release notes
// still told users to run it; without this stub Cobra failed with an
// unrendered "unknown command" and the binary exited silently (P0-1/P1-1).
// Flag parsing is disabled so any historical flag combination (--host …)
// reaches the explanation instead of an "unknown flag" error.
func newRemovedByoPrepareCommand(deps Dependencies, noColor *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "byo-prepare",
		Short:              "Removed in v1.0.8",
		Hidden:             true,
		DisableFlagParsing: true,
		// The command must also be reachable with arbitrary positional args.
		Args: cobra.ArbitraryArgs,
	}
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		renderer := newRenderer(deps, argsRequestJSON(args), *noColor)
		_ = renderer.Error("command_removed", byoPrepareRemovedMessage, []string{
			"For a password-sudo host, run runnerkit init --print-install-command and execute the printed install.sh line on the host once.",
			"Then run runnerkit up --host user@host --repo owner/name.",
		})
		return NewExitError(ExitInvalidInput, errors.New("byo-prepare was removed in v1.0.8"))
	}
	return cmd
}
