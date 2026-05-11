package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/errcodes"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

type destroyOptions struct {
	repo   string
	yes    bool
	dryRun bool
}

const (
	destroyConfirmTemplate          = "Destroy cloud runner: type `destroy %s` to remove the GitHub runner registration and RunnerKit-created Hetzner resources."
	destroyEphemeralConfirmTemplate = "Destroy ephemeral cloud runner: type `destroy %s` to remove the GitHub runner registration and RunnerKit-created Hetzner resources."
	destroyInputRequiredRemedy      = "Pass --yes to apply the cloud destroy plan non-interactively, or pass --dry-run to preview only."
	destroyBillingImpactCopy        = "Future billing stops only after provider resources are deleted or verified non-billable."
	destroyIncompleteCopyTemplate   = "Cleanup incomplete. RunnerKit kept state with pending cleanup checkpoints so you can rerun runnerkit destroy --repo %s."
	destroyCompleteCopy             = "Cloud runner destroyed. GitHub runner is absent and RunnerKit-created Hetzner resources are deleted or verified non-billable."

	pendingGitHubCleanup            = "github_cleanup_pending"
	pendingRemoteCleanup            = "remote_cleanup_pending"
	pendingProviderServer           = "provider_server_pending"
	pendingProviderSSHKey           = "provider_ssh_key_pending"
	pendingProviderFirewall         = "provider_firewall_pending"
	pendingProviderPrimaryIP        = "provider_primary_ip_pending"
	pendingProviderVerification     = "provider_verification_pending"
	pendingEphemeralLogPreservation = "ephemeral_log_preservation_pending"

	verifyDestroyedMaxAttempts = 12
	verifyDestroyedRetryDelay  = 1500 * time.Millisecond
)

func newDestroyCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &destroyOptions{}
	cmd := &cobra.Command{Use: "destroy"}
	cmd.Short = "Destroy a RunnerKit-managed cloud runner and billable provider resources"
	cmd.RunE = func(_ *cobra.Command, _ []string) error { return runDestroy(deps, *jsonOutput, *noColor, opts) }
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "apply the safe default cloud destroy plan")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "preview cloud cleanup without mutation")
	return cmd
}

func runDestroy(deps Dependencies, jsonOutput bool, noColor bool, opts *destroyOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	repo, err := resolveReadOnlyRepo(ctx, deps, renderer, opts.repo, "Pass --repo owner/name for cloud cleanup.")
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
		message := "No RunnerKit-managed cloud runner is saved for `" + repo.FullName + "`."
		_ = renderer.Error("state_not_found", message, []string{"Run runnerkit up --repo " + repo.FullName + " --cloud hetzner to create one, or pass --host user@host for BYO setup."})
		return NewExitError(ExitStateIO, errors.New(message))
	}
	if !isCloudProvider(repoState.Provider) {
		message := "RunnerKit destroy is only for RunnerKit-managed cloud runners."
		_ = renderer.Error("wrong_cleanup_command", message, []string{"Use runnerkit down --repo " + repo.FullName + " for BYO cleanup."})
		return NewExitError(ExitInvalidInput, errors.New(message))
	}
	plan := ops.BuildCloudDestroyPlan(repoState, opts.dryRun)
	if opts.dryRun {
		if jsonOutput {
			return renderer.JSON(destroyPayload(repo.FullName, true, plan, []cleanupResult{}, map[string]any{"status": "not_run"}, false, []string{}, false))
		}
		return renderCloudDestroyPlanHuman(renderer, plan)
	}
	if err := confirmCloudDestroy(ctx, deps, renderer, opts, jsonOutput, repo.FullName, repoState.Runner.Mode == "ephemeral"); err != nil {
		return err
	}
	results, verification, partial, pending, stateRemoved, err := applyCloudDestroy(ctx, deps, renderer, store, repoState)
	if err != nil {
		return err
	}
	if jsonOutput {
		return renderer.JSON(destroyPayload(repo.FullName, false, plan, results, verification, partial, pending, stateRemoved))
	}
	return renderCloudDestroyResultHuman(renderer, repo.FullName, results, partial, pending)
}

func confirmCloudDestroy(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *destroyOptions, jsonOutput bool, fullName string, ephemeral bool) error {
	if opts.yes {
		return nil
	}
	if jsonOutput || !deps.TTY.StdinTTY || deps.Prompts == nil {
		message := "RunnerKit can't continue because cloud destroy requires confirmation."
		_ = renderer.Error("input_required", message, []string{destroyInputRequiredRemedy})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	inputPrompter, ok := deps.Prompts.(interface {
		Input(context.Context, ui.Prompt) (string, error)
	})
	if !ok {
		message := "RunnerKit can't continue because cloud destroy requires typed confirmation."
		_ = renderer.Error("input_required", message, []string{destroyInputRequiredRemedy})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	template := destroyConfirmTemplate
	if ephemeral {
		template = destroyEphemeralConfirmTemplate
	}
	want := "destroy " + fullName
	got, err := inputPrompter.Input(ctx, ui.Prompt{Message: fmt.Sprintf(template, fullName)})
	if err != nil {
		return err
	}
	if got != want {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return NewExitError(ExitCanceled, errors.New(message))
	}
	return nil
}

func renderCloudDestroyPlanHuman(renderer *ui.Renderer, plan ops.CloudDestroyPlan) error {
	lines := []ui.Line{ui.WarningLine(ops.CloudDestroyBillingWarning)}
	for _, artifact := range plan.Artifacts {
		line := artifact.Description + ": " + artifact.Action
		if artifact.Blocked {
			lines = append(lines, ui.ErrorLine(line+" (blocked: "+artifact.BlockReason+")"))
		} else {
			lines = append(lines, ui.Bullet(line))
		}
	}
	lines = append(lines, ui.WarningLine(destroyBillingImpactCopy), ui.Next("Next: run runnerkit destroy --repo "+plan.Repo+" --yes after reviewing the plan."))
	return renderer.Step(1, 1, "Cloud destroy plan", lines...)
}

func applyCloudDestroy(ctx context.Context, deps Dependencies, renderer *ui.Renderer, store rkstate.Store, repoState rkstate.RepositoryState) ([]cleanupResult, map[string]any, bool, []string, bool, error) {
	renderer.Redactor().Register(redact.MachineRef, repoState.Machine.HostRef)
	registerKnownCloudProviderSecrets(renderer)
	results := []cleanupResult{}
	pending := []string{}
	partial := false
	stateRemoved := false
	status := collectStatus(ctx, deps, store.Path(), repoState, true)
	sshReachable := status.Observed.SSH.Reachable
	target, targetErr := targetFromState(repoState)
	resolvedUnit := repoState.Machine.ServiceName
	if sshReachable && targetErr == nil {
		resolvedUnit = ops.ResolveActionsRunnerSystemdUnit(ctx, deps.RemoteExecutor, target, repoState.Machine.ServiceName)
	}

	// Ephemeral cloud runners preserve _diag and journal logs to the
	// host log archive before file removal so cloud destroy never
	// silently discards troubleshooting evidence.
	if repoState.Runner.Mode == "ephemeral" && sshReachable && targetErr == nil {
		preserveResult, preserveErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{
			ID:      "ephemeral.logs.preserve",
			Script:  bootstrap.RenderEphemeralLogPreservationScript(repoState.Machine.InstallPath, repoState.Ephemeral.LogArchivePath, repoState.Machine.ServiceName),
			Sudo:    true,
			Timeout: 60 * time.Second,
		})
		if preserveErr != nil || preserveResult.ExitCode != 0 {
			partial = true
			pending = appendUnique(pending, pendingEphemeralLogPreservation)
		}
	}
	if sshReachable && targetErr == nil {
		removal, err := deps.GitHub.CreateRemovalToken(ctx, repoState.Repo)
		if err != nil {
			partial = true
			pending = appendUnique(pending, pendingRemoteCleanup)
			results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudRemoteRunner), Status: "pending", Message: pendingRemoteCleanup})
		} else {
			renderer.Redactor().Register(redact.RunnerRemovalToken, removal.Token)
			result, runErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "destroy.remote_runner", Script: cloudDestroyRemoteScript(repoState, resolvedUnit), Env: map[string]string{"RUNNERKIT_REMOVAL_TOKEN": removal.Token}, RedactArgs: []string{removal.Token}, Timeout: 90 * time.Second})
			if runErr != nil || result.ExitCode != 0 {
				partial = true
				pending = appendUnique(pending, pendingRemoteCleanup)
				msg := pendingRemoteCleanup
				if detail := remoteCleanupDetail(renderer.Redactor(), result); detail != "" {
					msg += " (" + detail + ")"
				}
				results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudRemoteRunner), Status: "pending", Message: msg})
			} else {
				results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudRemoteRunner), Status: "done"})
			}
		}
	} else {
		partial = true
		pending = appendUnique(pending, pendingRemoteCleanup)
		results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudRemoteRunner), Status: "pending", Message: pendingRemoteCleanup})
	}

	deleted, err := deleteGitHubRunnerCandidate(ctx, deps, repoState)
	if err != nil {
		partial = true
		pending = appendUnique(pending, pendingGitHubCleanup)
		results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudGitHubRunner), Status: "pending", Message: pendingGitHubCleanup})
	} else if deleted == 0 {
		results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudGitHubRunner), Status: "skipped", Message: "GitHub runner already absent"})
	} else {
		results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudGitHubRunner), Status: "done", Message: fmt.Sprintf("deleted runner id %d", deleted)})
	}

	cloudProvider, ok := deps.Providers.Get(provider.HetznerProvider)
	if !ok || cloudProvider == nil {
		partial = true
		pending = appendUnique(pending, pendingProviderVerification)
		verification := map[string]any{"ok": false, "error": "provider dependency unavailable"}
		_ = keepCloudDestroyState(store, repoState, pending)
		return results, verification, partial, pending, false, nil
	}
	destroyResult, destroyErr := cloudProvider.Destroy(ctx, repoState.Provider)
	for _, item := range destroyResult.Results {
		results = append(results, cleanupResult{Artifact: item.Artifact, Status: item.Status, Message: item.Message})
	}
	if destroyErr != nil || destroyResult.Partial {
		partial = true
		for _, item := range destroyResult.Pending {
			pending = appendUnique(pending, item)
		}
	}
	providerVerification, verifyErr := verifyDestroyedWithRetry(ctx, deps, cloudProvider, repoState.Provider)
	billable := providerVerification.BillableResources
	if billable == nil {
		billable = []string{}
	}
	missing := providerVerification.Missing
	if missing == nil {
		missing = []string{}
	}
	verification := map[string]any{"ok": providerVerification.OK, "billable_resources": billable, "missing": missing, "error": providerVerification.Error}
	if verifyErr != nil || !providerVerification.OK {
		partial = true
		pending = appendUnique(pending, pendingProviderVerification)
		if verifyErr != nil {
			verification["error"] = verifyErr.Error()
		}
	}
	if partial {
		if err := keepCloudDestroyState(store, repoState, pending); err != nil {
			return results, verification, partial, pending, false, NewExitError(ExitStateIO, err)
		}
		return results, verification, true, pending, false, nil
	}
	removed, err := store.RemoveRepository(repoState.Repo.FullName)
	if err != nil {
		return results, verification, false, pending, false, NewExitError(ExitStateIO, err)
	}
	stateRemoved = removed
	results = append(results, cleanupResult{Artifact: string(ops.ArtifactCloudLocalState), Status: "done"})
	return results, verification, false, pending, stateRemoved, nil
}

func cloudDestroyRemoteScript(repoState rkstate.RepositoryState, systemdUnit string) string {
	install := shellQuote(repoState.Machine.InstallPath)
	work := shellQuote(repoState.Machine.WorkDir)
	u := shellQuote(systemdUnit)
	svc := "set -euo pipefail\ncd " + install + " && sudo ./svc.sh stop || true\nsudo systemctl stop " + u + " || true\nsleep 3\n"
	teardown := "cd " + install + " && sudo ./svc.sh uninstall || true\nsudo systemctl disable --now " + u + " || true\n"
	remove := bootstrap.RenderRemoveConfigScript(repoState.Machine.InstallPath, bootstrap.DefaultServiceUser)
	rm := "sudo rm -rf -- " + install + " " + work + "\n"
	return svc + "\n" + teardown + "\n" + remove + "\n" + rm
}

func verifyDestroyedWithRetry(ctx context.Context, deps Dependencies, cloud provider.Provider, ref rkstate.ProviderRef) (provider.VerificationResult, error) {
	sleep := deps.Sleep
	if sleep == nil {
		sleep = func(_ context.Context, d time.Duration) error {
			time.Sleep(d)
			return nil
		}
	}
	var last provider.VerificationResult
	var lastErr error
	for attempt := 0; attempt < verifyDestroyedMaxAttempts; attempt++ {
		v, err := cloud.VerifyDestroyed(ctx, ref)
		last, lastErr = v, err
		if err == nil && v.OK {
			return v, nil
		}
		if attempt == verifyDestroyedMaxAttempts-1 {
			break
		}
		if err := sleep(ctx, verifyDestroyedRetryDelay); err != nil {
			return last, err
		}
	}
	return last, lastErr
}

func keepCloudDestroyState(store rkstate.Store, repoState rkstate.RepositoryState, pending []string) error {
	now := time.Now()
	repoState.Cleanup.Notes = append([]string(nil), pending...)
	repoState.Operations = nil
	for _, item := range pending {
		artifact := strings.TrimSuffix(item, "_pending")
		repoState.Operations = append(repoState.Operations, rkstate.OperationCheckpoint{Command: "destroy", Artifact: artifact, Status: "pending", Message: item, UpdatedAt: now})
	}
	repoState.UpdatedAt = now
	return store.UpdateRepository(repoState)
}

func renderCloudDestroyResultHuman(renderer *ui.Renderer, repo string, results []cleanupResult, partial bool, pending []string) error {
	lines := []ui.Line{}
	if partial {
		lines = append(lines, ui.WarningLine(fmt.Sprintf(destroyIncompleteCopyTemplate, repo)))
		// Surface the canonical RKD codes alongside the partial-cleanup
		// banner so the See: URL points users to the right entry (D-15).
		lines = append(lines, ui.Bullet(errcodes.FormatLine(errcodes.CleanDestroyPartial)))
		lines = append(lines, ui.Bullet(errcodes.FormatLine(errcodes.ProvHCloudPartialDestroy)))
	} else {
		lines = append(lines, ui.Success(destroyCompleteCopy))
	}
	for _, result := range results {
		lines = append(lines, ui.Bullet(result.Artifact+": "+result.Status+" "+result.Message))
	}
	if len(pending) > 0 {
		lines = append(lines, ui.Bullet("pending: "+strings.Join(pending, ", ")))
		// If the ephemeral log-preservation finalizer is in the pending
		// list, emit the dedicated RKD-CLEAN-004 reference.
		for _, p := range pending {
			if p == pendingEphemeralLogPreservation {
				lines = append(lines, ui.Bullet(errcodes.FormatLine(errcodes.CleanEphemeralLogPreserveFailed)))
				break
			}
		}
	}
	return renderer.Step(1, 1, "cloud destroy", lines...)
}

func destroyPayload(repo string, dryRun bool, plan ops.CloudDestroyPlan, results []cleanupResult, verification any, partial bool, pending []string, stateRemoved bool) map[string]any {
	if results == nil {
		results = []cleanupResult{}
	}
	if pending == nil {
		pending = []string{}
	}
	return map[string]any{"ok": !partial, "command": "destroy", "repo": repo, "dry_run": dryRun, "plan": plan, "results": results, "provider_verification": verification, "partial_cleanup": partial, "pending": pending, "state_removed": stateRemoved}
}
