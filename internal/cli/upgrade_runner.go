package cli

import (
	"errors"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

// upgradeRunnerOptions captures CLI flags for the upgrade-runner command.
// The flags stay registered while the command is disabled so existing
// scripts reach the explanation instead of an unknown-flag error.
type upgradeRunnerOptions struct {
	repo  string
	force bool
	yes   bool
}

// upgradeRunnerDisabledMessage explains why `runnerkit upgrade-runner` (and
// `doctor --fix`, which called it) is refused in this release (A-06a, P0-3).
const upgradeRunnerDisabledMessage = "upgrade-runner is disabled in this release (known issue). In v1.3.3 and earlier it re-ran the runner install without a registration token: it deleted the runner's .runner and .credentials files and ran config.sh with an empty token, which unregisters a healthy runner, and it never replaced the runner binaries."

// upgradeRunnerSelfUpdateNote is why no replacement command is needed.
const upgradeRunnerSelfUpdateNote = "You do not need it to stay current: RunnerKit never passes --disableupdate, so the GitHub runner updates itself when GitHub requires a newer version."

// newUpgradeRunnerCommand registers `runnerkit upgrade-runner`, which is
// disabled: it refuses with command_disabled (exit 2) before any GitHub, SSH
// or state access. The real fix (re-register with a fresh token and actually
// replace the binaries) is tracked as R-03.
func newUpgradeRunnerCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &upgradeRunnerOptions{}
	cmd := &cobra.Command{Use: "upgrade-runner"}
	cmd.Short = "Disabled in this release: prints the manual re-register steps"
	cmd.Long = upgradeRunnerDisabledMessage + "\n\n" + upgradeRunnerSelfUpdateNote
	cmd.Flags().StringVar(&opts.repo, "repo", "", "owner/name (used only to print the manual steps)")
	cmd.Flags().BoolVar(&opts.force, "force", false, "ignored; the command is disabled")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "ignored; the command is disabled")
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runUpgradeRunner(deps, *jsonOutput, *noColor, opts)
	}
	return cmd
}

func runUpgradeRunner(deps Dependencies, jsonOutput bool, noColor bool, opts *upgradeRunnerOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	return refuseUpgradeRunner(renderer, "upgrade-runner", opts.repo)
}

// refuseUpgradeRunner renders the command_disabled error shared by
// `upgrade-runner` and `doctor --fix` and returns the exit-2 error.
func refuseUpgradeRunner(renderer *ui.Renderer, command string, repo string) error {
	_ = renderer.Error("command_disabled", upgradeRunnerDisabledMessage, upgradeRunnerRemediation(repo))
	return NewExitError(ExitInvalidInput, errors.New(command+" is disabled in this release"))
}

func upgradeRunnerRemediation(repo string) []string {
	remediation := []string{upgradeRunnerSelfUpdateNote}
	remediation = append(remediation, ops.ManualReregisterSteps(repo, false)...)
	return append(remediation, "For a RunnerKit-created Hetzner server use runnerkit destroy instead of runnerkit down.")
}
