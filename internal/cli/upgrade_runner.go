package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/labels"
	"github.com/accidentally-awesome-labs/runnerkit/internal/preflight"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/spf13/cobra"
)

// upgradeRunnerOptions captures CLI flags for the upgrade-runner command.
type upgradeRunnerOptions struct {
	repo  string
	force bool
	yes   bool
}

// newUpgradeRunnerCommand registers `runnerkit upgrade-runner`. The
// command re-applies the bootstrap (Apply for persistent, ApplyEphemeral
// for ephemeral) against the saved MachineRef using the bundled runner
// pin (`bootstrap.RunnerVersion`). It refuses without `--force` when an
// ephemeral runner is currently waiting/busy (Phase 5 invariant).
func newUpgradeRunnerCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &upgradeRunnerOptions{}
	cmd := &cobra.Command{Use: "upgrade-runner"}
	cmd.Short = "Re-apply runner bootstrap with the bundled runner pin"
	cmd.Long = "Reinstalls the GitHub Actions runner on the saved host with the runner version bundled in this RunnerKit release. Idempotent: safe to re-run."
	cmd.Flags().StringVar(&opts.repo, "repo", "", "owner/name (defaults to current dir's git remote)")
	cmd.Flags().BoolVar(&opts.force, "force", false, "force upgrade even when an ephemeral runner is currently waiting or busy")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip confirmation prompt")
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runUpgradeRunner(deps, *jsonOutput, *noColor, opts)
	}
	return cmd
}

func runUpgradeRunner(deps Dependencies, jsonOutput bool, noColor bool, opts *upgradeRunnerOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()

	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name or run runnerkit upgrade-runner from a GitHub repository.")
	if err != nil {
		return err
	}
	store := rkstate.NewStore(deps.StateBaseDir)
	repoState, ok, err := store.GetRepository(repo.FullName)
	if err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	if !ok {
		message := "No RunnerKit-managed runner found for " + repo.FullName + "."
		_ = renderer.Error("state_not_found", message, []string{"Run runnerkit up --repo " + repo.FullName + " --host user@host first."})
		return NewExitError(ExitStateIO, errors.New(message))
	}

	currentPin := repoState.RunnerTemplateVersion
	bundled := bootstrap.RunnerVersion

	// Ephemeral lifecycle gate (D-08, Phase 5 invariant): refuse to
	// reapply on top of a registered ephemeral runner unless --force.
	if repoState.Ephemeral.Enabled {
		switch repoState.Ephemeral.FinalizerStatus {
		case "completed", "ttl_expired":
			fmt.Fprintln(deps.Out, "Ephemeral runner is one-shot and already terminated. The next `runnerkit up --mode ephemeral` will use the bundled pin "+bundled+".")
			return nil
		case "waiting", "busy", "":
			if !opts.force {
				message := fmt.Sprintf("ephemeral runner finalizer status %q; refusing without --force (would drop the queued ephemeral runner registration)", repoState.Ephemeral.FinalizerStatus)
				_ = renderer.Error("ephemeral_upgrade_refused", message, []string{"Re-run with --force to proceed and drop the queued registration."})
				return NewExitError(ExitInvalidInput, errors.New(message))
			}
		}
	}

	// Plan-before-mutation print (Phase 2/4 contract).
	fmt.Fprintf(deps.Out, "Upgrade runner pin: %s -> %s\n", defaultPin(currentPin), bundled)
	fmt.Fprintf(deps.Out, "Target host: %s\n", repoState.Machine.HostRef)

	if !opts.yes {
		message := "RunnerKit can't continue because applying the runner upgrade requires confirmation."
		_ = renderer.Error("input_required", message, []string{"Pass --yes to apply the upgrade non-interactively."})
		return NewExitError(ExitInputRequired, errors.New(message))
	}

	target, err := targetFromState(repoState)
	if err != nil {
		_ = renderer.Error("state_target_invalid", err.Error(), []string{"Re-run runnerkit up to refresh the saved machine target."})
		return NewExitError(ExitStateIO, err)
	}

	// Build bootstrap options that mirror what `runnerkit up` would have
	// generated for this saved state. The bundled pin flows through
	// PackageFor so the download_runner step references the new runner
	// tarball.
	// SEED-002: download_runner uses a shared versioned tarball cache on the host;
	// upgrade-runner re-applies Apply and benefits from that cache when the
	// pinned version's archive is already present.
	arch := defaultString(repoState.Runner.Arch, labels.DefaultArch)
	pkg, err := bootstrap.PackageFor("linux", arch)
	if err != nil {
		_ = renderer.Error("unsupported_runner_package", err.Error(), []string{"Use linux/x64 or linux/arm64 for the BYO persistent runner path."})
		return NewExitError(ExitSafetyGate, err)
	}
	bopts := bootstrap.Options{
		RunnerName:    repoState.Runner.Name,
		RepoURL:       "https://github.com/" + repoState.Repo.FullName,
		Labels:        repoState.Runner.Labels,
		InstallPath:   repoState.Machine.InstallPath,
		WorkDir:       repoState.Machine.WorkDir,
		ServiceUser:   bootstrap.DefaultServiceUser,
		Package:       pkg,
		MissingTools:  nil,
		ExtraPackages: repoState.ExtraPackages,
	}
	if repoState.Ephemeral.Enabled {
		bopts.Mode = "ephemeral"
		bopts.LogArchivePath = repoState.Ephemeral.LogArchivePath
		bopts.EphemeralServiceName = repoState.Machine.ServiceName
	}

	// Best-effort preflight tools probe so the script doesn't try to
	// install missing tools that are already present. Non-fatal.
	if report, err := preflight.Run(ctx, deps.RemoteExecutor, target, preflight.Options{RunnerName: repoState.Runner.Name, AllowUnknownLinux: true}); err == nil {
		bopts.MissingTools = report.FixableTools
	}

	if repoState.Ephemeral.Enabled {
		if _, err := bootstrap.ApplyEphemeral(ctx, deps.RemoteExecutor, target, bopts); err != nil {
			_ = renderer.Error("upgrade_runner_failed", "RunnerKit could not re-apply the ephemeral runner install plan.", []string{"Review the remote host output and re-run runnerkit upgrade-runner after fixing the issue."})
			return NewExitError(ExitUnexpected, fmt.Errorf("apply-ephemeral: %w", err))
		}
	} else {
		if _, err := bootstrap.Apply(ctx, deps.RemoteExecutor, target, bopts); err != nil {
			_ = renderer.Error("upgrade_runner_failed", "RunnerKit could not re-apply the persistent runner install plan.", []string{"Review the remote host output and re-run runnerkit upgrade-runner after fixing the issue."})
			return NewExitError(ExitUnexpected, fmt.Errorf("apply: %w", err))
		}
	}

	// Persist the new pin only after Apply returns nil so a partial
	// failure does not leave state claiming the upgrade succeeded.
	repoState.RunnerTemplateVersion = bundled
	if err := store.UpdateRepository(repoState); err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't update saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	fmt.Fprintf(deps.Out, "Runner pin updated to %s.\n", bundled)
	return nil
}

func defaultPin(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
