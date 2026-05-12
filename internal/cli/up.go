package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/errcodes"
	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/labels"
	"github.com/accidentally-awesome-labs/runnerkit/internal/preflight"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider/hetzner"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/runmode"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
	"github.com/accidentally-awesome-labs/runnerkit/internal/workflow"
	"github.com/spf13/cobra"
)

const (
	defaultRunnerPollInterval = 2 * time.Second
	defaultRunnerPollTimeout  = 60 * time.Second
)

type upOptions struct {
	repo                  string
	yes                   bool
	nonInteractive        bool
	dryRun                bool
	allowPublicRepoRisk   bool
	allowEphemeralBYORisk bool
	replace               bool
	host                  string
	sshPort               int
	sshKey                string
	allowUnknownLinux     bool
	cloud                 string
	cloudRegion           string
	cloudProfile          string
	sshAllowedCIDR        string
	mode                  string
	ephemeralTTL          time.Duration
	registerLifecycleOnly bool // true for `runnerkit register` (SEED-002 foundation gate)
}

type GitHubService interface {
	Repository(ctx context.Context, repo gh.Repo) (gh.Repo, error)
	VerifyAuth(ctx context.Context, repo gh.Repo) (gh.PermissionStatus, error)
	VerifyRunnerManagementRead(ctx context.Context, repo gh.Repo) (gh.PermissionStatus, error)
	CreateRegistrationToken(ctx context.Context, repo gh.Repo) (gh.RunnerToken, error)
	CreateRemovalToken(ctx context.Context, repo gh.Repo) (gh.RunnerToken, error)
	ListRunners(ctx context.Context, repo gh.Repo) ([]gh.Runner, error)
	DeleteRunner(ctx context.Context, repo gh.Repo, runnerID int64) error
}

func newUpCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &upOptions{sshPort: 22, cloudRegion: provider.HetznerDefaultRegion, cloudProfile: provider.HetznerDefaultServerType, sshAllowedCIDR: provider.HetznerDefaultSSHAllowedCIDR}
	cmd := &cobra.Command{Use: "up"}
	cmd.Short = "Set up a BYO GitHub Actions runner"
	cmd.Long = "Connect a BYO Linux host, preflight it over SSH, bootstrap a non-root persistent runner service, and print RunnerKit label guidance."
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runUp(deps, *jsonOutput, *noColor, opts)
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "accept safe defaults without interactive confirmation")
	cmd.Flags().BoolVar(&opts.nonInteractive, "non-interactive", false, "fail instead of prompting for missing input")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "preview the BYO preflight and bootstrap plan without installing")
	cmd.Flags().BoolVar(&opts.allowPublicRepoRisk, "allow-public-repo-risk", false, "explicitly acknowledge public repository persistent-runner risk")
	cmd.Flags().BoolVar(&opts.replace, "replace", false, "replace existing saved foundation state for --repo when used with --yes")
	cmd.Flags().StringVar(&opts.host, "host", "", "BYO SSH target as user@host or user@host:port")
	cmd.Flags().IntVar(&opts.sshPort, "ssh-port", 22, "SSH port for the target host")
	cmd.Flags().StringVar(&opts.sshKey, "ssh-key", "", "SSH private key path reference for the target host")
	cmd.Flags().BoolVar(&opts.allowUnknownLinux, "allow-unknown-linux", false, "try best-effort install on unverified Linux distributions")
	cmd.Flags().StringVar(&opts.cloud, "cloud", "", "recommended cloud provider; only hetzner is supported in Phase 4")
	cmd.Flags().StringVar(&opts.cloudRegion, "cloud-region", provider.HetznerDefaultRegion, "provider region/location for cloud runner")
	cmd.Flags().StringVar(&opts.cloudProfile, "cloud-profile", provider.HetznerDefaultServerType, "provider server profile for cloud runner")
	cmd.Flags().StringVar(&opts.sshAllowedCIDR, "ssh-allowed-cidr", provider.HetznerDefaultSSHAllowedCIDR, "SSH ingress CIDR for cloud runner")
	cmd.Flags().StringVar(&opts.mode, "mode", "", "runner mode: persistent or ephemeral")
	cmd.Flags().DurationVar(&opts.ephemeralTTL, "ephemeral-ttl", runmode.DefaultEphemeralTTL, "TTL safeguard for ephemeral runners")
	cmd.Flags().BoolVar(&opts.allowEphemeralBYORisk, "allow-ephemeral-byo-risk", false, "acknowledge that BYO ephemeral mode is not a clean VM for risky repositories")

	return cmd
}

func runUp(deps Dependencies, jsonOutput bool, noColor bool, opts *upOptions) error {
	defer maybeShowUpdateNotice(deps, jsonOutput)
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	resolution, err := resolveUpRepo(ctx, deps, renderer, opts)
	if err != nil {
		return err
	}
	store := rkstate.NewStore(deps.StateBaseDir)

	repo, err := deps.GitHub.Repository(ctx, resolution.Repo)
	if err != nil {
		message := fmt.Sprintf("RunnerKit can't read repository metadata for %s.", resolution.Repo.FullName)
		_ = renderer.Error("github_permission_denied", message, []string{gh.FineGrainedTokenRemediation(resolution.Repo), "Verify GitHub credentials can read repository metadata for " + resolution.Repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	modeDecision, err := resolveModeDecision(ctx, deps, renderer, repo, opts, jsonOutput)
	if err != nil {
		return err
	}
	decision := gh.EvaluateSafety(repo, gh.SafetyOptions{AllowPublicRepoRisk: opts.allowPublicRepoRisk})
	if err := enforceModeSafetyDecision(ctx, deps, renderer, repo, decision, &modeDecision, opts, jsonOutput); err != nil {
		return err
	}

	setupPath, err := resolveSetupPath(ctx, deps, renderer, repo, opts, jsonOutput)
	if err != nil {
		return err
	}
	if setupPath == setupPathCloud {
		return runCloudUp(ctx, deps, renderer, repo, decision, modeDecision, opts, jsonOutput)
	}

	if deps.Explain() && !jsonOutput {
		why, runs, takes := explainBYOSetup()
		printExplainBlock(deps.Err, "BYO runner setup (runnerkit up / register)", why, runs, takes)
	}

	status, err := deps.GitHub.VerifyAuth(ctx, repo)
	if err != nil || !status.OK {
		message := fmt.Sprintf("RunnerKit can't create a repository runner registration token for %s.", repo.FullName)
		remediation := status.Remediation
		if len(remediation) == 0 {
			remediation = []string{"Create a fine-grained token scoped only to " + repo.FullName + " with repository Administration read/write and Metadata read, then pass it with RUNNERKIT_GITHUB_TOKEN for this command."}
		}
		// Append the stable RKD-AUTH-004 code + See: URL after the
		// existing remediation copy (D-15). Append (not prepend) so
		// existing tests that index remediation[0] keep working.
		remediation = append(remediation, errcodes.FormatLine(errcodes.AuthRunnerManagementPermissionDenied))
		_ = renderer.Error("github_permission_denied", message, remediation)
		if err == nil {
			err = errors.New(message)
		}
		return NewExitError(ExitGitHubAuth, err)
	}

	target, err := resolveBYOTarget(ctx, deps, renderer, opts, jsonOutput)
	if err != nil {
		return err
	}
	existing, exists, err := store.GetRepository(repo.FullName)
	if err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + " and re-run runnerkit up."})
		return NewExitError(ExitStateIO, err)
	}
	hostKey, acceptedAt, err := verifyTargetHostKey(ctx, deps, renderer, opts, jsonOutput, target, existing, exists)
	if err != nil {
		return err
	}

	report, err := preflight.Run(ctx, deps.RemoteExecutor, target, preflight.Options{AllowUnknownLinux: opts.allowUnknownLinux})
	if err != nil {
		_ = renderer.Error("ssh_preflight_failed", "RunnerKit could not complete SSH preflight.", []string{err.Error()})
		return NewExitError(ExitSafetyGate, err)
	}
	if !report.Passed() {
		return renderPreflightFailure(renderer, jsonOutput, report)
	}
	if !jsonOutput && !opts.dryRun {
		writeBYOChecklistHuman(deps.Out, deps.TTY, deps.StateBaseDir, repo.FullName, target, 1)
	}

	arch := defaultString(report.Arch, labels.DefaultArch)
	labelSet := buildModeLabelSet(repo, modeDecision, arch)
	runnerPkg, err := bootstrap.PackageFor("linux", arch)
	if err != nil {
		_ = renderer.Error("unsupported_runner_package", err.Error(), []string{"Use linux/x64 or linux/arm64 for the Phase 2 BYO persistent runner path."})
		return NewExitError(ExitSafetyGate, err)
	}
	bootstrapOpts := buildBootstrapOptions(repo, labelSet, runnerPkg, report)
	if modeDecision.Mode == runmode.ModeEphemeral {
		bootstrapOpts = ephemeralBootstrapOptions(bootstrapOpts, opts.ephemeralTTL)
	}
	bootstrapPlan := bootstrap.Plan(bootstrapOpts)

	if r, ok := report.Result(preflight.CheckPrivilegePasswordReq); ok && r.Severity == preflight.SeverityWarning {
		return RenderHostInstallRequired(renderer, jsonOutput, deps.Version)
	}

	if opts.registerLifecycleOnly {
		if err := verifyBYOFoundationForRegister(ctx, deps, renderer, jsonOutput, target); err != nil {
			return err
		}
	}

	if opts.dryRun {
		if err := renderModeTradeoffs(renderer, jsonOutput, repo, modeDecision, opts.ephemeralTTL); err != nil {
			return err
		}
		return renderDryRun(renderer, jsonOutput, repo, status.Source, decision.Warnings, store.Path(), target, report, labelSet, bootstrapPlan, modeDecision, opts.ephemeralTTL)
	}

	runners, err := deps.GitHub.ListRunners(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_runner_list_failed", "RunnerKit can't list repository self-hosted runners.", []string{"Verify GitHub credentials can list repository runners for " + repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	if existingRunner, found := gh.FindRunnerByName(runners, labelSet.RunnerName); found {
		// Bug 17 (Plan 06-07 attempt-14, 2026-05-06): only refuse when the
		// existing runner is NOT one of ours. The runner name is deterministic
		// per (repo, host, mode), so a re-run of `runnerkit up` always sees its
		// own previously-registered runner. config.sh --replace handles
		// re-registration end-to-end.
		if !isRunnerKitManagedRunner(existingRunner) {
			return runnerNameConflict(renderer, labelSet.RunnerName, existingRunner)
		}
	}

	if err := confirmBootstrapPlan(ctx, deps, renderer, opts, jsonOutput, target); err != nil {
		return err
	}
	if !jsonOutput && !opts.dryRun {
		writeBYOChecklistHuman(deps.Out, deps.TTY, deps.StateBaseDir, repo.FullName, target, 2)
	}
	token, err := deps.GitHub.CreateRegistrationToken(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_permission_denied", "RunnerKit can't create a fresh runner registration token.", []string{gh.FineGrainedTokenRemediation(repo)})
		return NewExitError(ExitGitHubAuth, err)
	}
	renderer.Redactor().Register(redact.RunnerRegistrationToken, token.Token)
	bootstrapOpts.RunnerToken = token.Token
	if !jsonOutput && !opts.dryRun {
		writeBYOChecklistHuman(deps.Out, deps.TTY, deps.StateBaseDir, repo.FullName, target, 3)
	}
	if modeDecision.Mode == runmode.ModeEphemeral {
		if result, err := bootstrap.ApplyEphemeral(ctx, deps.RemoteExecutor, target, bootstrapOpts); err != nil {
			var serviceErr bootstrap.ServiceNotActiveError
			if errors.As(err, &serviceErr) {
				remediation := []string{"Run systemctl status " + bootstrapOpts.EphemeralServiceName + " on the host or re-run runnerkit up after fixing the service."}
				if stderr := strings.TrimSpace(serviceErr.Stderr); stderr != "" {
					remediation = append(remediation, "Remote stderr ("+serviceErr.CommandID+"): "+renderer.Redactor().String(stderr))
				}
				_ = renderer.Error("runner_service_not_active", "RunnerKit installed the ephemeral runner but the service is not active.", remediation)
				return NewExitError(ExitSafetyGate, err)
			}
			remediation := []string{"Review the remote host output, fix the issue, and re-run runnerkit up."}
			if cmdID, stderr := lastCommandFailureContext(result, err); stderr != "" {
				remediation = append(remediation, "Remote stderr ("+cmdID+"): "+renderer.Redactor().String(stderr))
			}
			_ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the ephemeral runner install plan.", remediation)
			return NewExitError(ExitSafetyGate, err)
		}
	} else {
		if result, err := bootstrap.Apply(ctx, deps.RemoteExecutor, target, bootstrapOpts); err != nil {
			var serviceErr bootstrap.ServiceNotActiveError
			if errors.As(err, &serviceErr) {
				remediation := []string{"Run sudo ./svc.sh status in the runner directory or re-run runnerkit up after fixing the service."}
				if stderr := strings.TrimSpace(serviceErr.Stderr); stderr != "" {
					remediation = append(remediation, "Remote stderr ("+serviceErr.CommandID+"): "+renderer.Redactor().String(stderr))
				}
				_ = renderer.Error("runner_service_not_active", "RunnerKit installed the runner but the service is not active.", remediation)
				return NewExitError(ExitSafetyGate, err)
			}
			remediation := []string{"Review the remote host output, fix the issue, and re-run runnerkit up."}
			if cmdID, stderr := lastCommandFailureContext(result, err); stderr != "" {
				remediation = append(remediation, "Remote stderr ("+cmdID+"): "+renderer.Redactor().String(stderr))
			}
			_ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the BYO runner install plan.", remediation)
			return NewExitError(ExitSafetyGate, err)
		}
	}

	if !jsonOutput && !opts.dryRun {
		writeBYOChecklistHuman(deps.Out, deps.TTY, deps.StateBaseDir, repo.FullName, target, 4)
	}

	onlineRunner, ok, err := waitForRunnerOnline(ctx, deps, repo, labelSet.RunnerName, labelSet.Labels)
	if err != nil {
		return err
	}
	if !ok {
		_ = renderer.Error("runner_online_timeout", "RunnerKit could not verify the GitHub runner came online with the expected labels.", []string{"Check the remote service status and GitHub repository Actions runner settings, then re-run runnerkit up."})
		return NewExitError(ExitSafetyGate, errors.New("runner_online_timeout"))
	}
	if !jsonOutput && !opts.dryRun {
		writeBYOChecklistHuman(deps.Out, deps.TTY, deps.StateBaseDir, repo.FullName, target, 5)
	}

	if modeDecision.Mode == runmode.ModeEphemeral {
		repoState := buildEphemeralBYORepositoryState(deps, repo, status.Source, decision, modeDecision, labelSet, target, hostKey, acceptedAt, bootstrapOpts, onlineRunner, opts.ephemeralTTL)
		if err := saveRepositoryState(ctx, deps, renderer, opts, jsonOutput, store, repo.FullName, repoState); err != nil {
			return err
		}
		if jsonOutput {
			return renderer.JSON(ephemeralCompletionJSON(repo.FullName, modeDecision, mergeWarnings(decision.Warnings, modeDecision.Warnings), store.Path(), target, labelSet, bootstrapOpts, onlineRunner, opts.ephemeralTTL, false))
		}
		return renderEphemeralCompletionHuman(renderer, repo.FullName, modeDecision, mergeWarnings(decision.Warnings, modeDecision.Warnings), store.Path(), target, labelSet, bootstrapOpts, onlineRunner, opts.ephemeralTTL, false)
	}

	repoState := buildBYORepositoryState(deps, repo, status.Source, decision, labelSet, target, hostKey, acceptedAt, bootstrapOpts, onlineRunner)
	if err := saveRepositoryState(ctx, deps, renderer, opts, jsonOutput, store, repo.FullName, repoState); err != nil {
		return err
	}

	if jsonOutput {
		return renderer.JSON(upCompletionJSON(repo.FullName, decision.Warnings, store.Path(), target, labelSet, bootstrapOpts, onlineRunner))
	}
	return renderCompletionHuman(renderer, decision.Warnings, store.Path(), target, labelSet, bootstrapOpts, onlineRunner)
}

type setupPath string

const (
	setupPathBYO   setupPath = "byo"
	setupPathCloud setupPath = "cloud"

	cloudNoIntentCopy                = "RunnerKit will not create billable cloud resources without an explicit --cloud hetzner flag and --yes."
	cloudUnsupportedCopy             = "Supported Phase 4 cloud value: --cloud hetzner."
	cloudPrimaryCTA                  = "Provision cloud runner"
	cloudEmptyStateHeadingExample    = "No RunnerKit-managed cloud runner is saved for `owner/name`."
	cloudEmptyStateBodyExample       = "Run `runnerkit up --repo owner/name --cloud hetzner` to create one, or pass `--host user@host` to use an existing machine."
	cloudFutureCleanupExample        = "Future cleanup: runnerkit destroy --repo owner/name"
	cloudProvisioningPlanTitle       = "Cloud runner provisioning plan"
	cloudCostCaveatCopy              = "Estimated cost is approximate. Provider pricing varies by region and time; billing stops only after RunnerKit-created billable resources are deleted or verified non-billable."
	cloudProvisionConfirmationRemedy = "Pass --yes to create billable Hetzner resources after reviewing the cloud provisioning plan, or pass --dry-run to preview only."
	cloudProvisionPending            = "cloud_provision_pending"
	cloudReadinessPending            = "cloud_readiness_pending"
	cloudReadinessFailedMessage      = "Cloud machine is not ready for runner registration yet. Fix the provider or SSH readiness issue, then rerun runnerkit up --repo owner/name --cloud hetzner."
	cloudProviderSuccessExample      = "Provider: Hetzner fsn1 cpx22 ubuntu-24.04"
	cloudJSONKeyFutureDestroyCommand = "future_destroy_command"
	cloudJSONKeyEstimatedHourlyCost  = "estimated_hourly_cost"
	cloudJSONKeyEstimatedMonthlyCost = "estimated_monthly_cost"

	// Mode/profile selection copy. The exact strings below come from the
	// Phase 5 UI-SPEC and must not drift; tests in this package and in
	// docs grep these literals.
	modePromptMessage          = "Choose runner mode for `owner/name`:"
	modeOptionPersistentLabel  = "Persistent trusted runner"
	modeOptionPersistentDesc   = "Reuses one runner across trusted private jobs. Lowest ongoing friction, but unsafe for public, fork, or untrusted workflows."
	modeOptionEphemeralBYOL    = "Ephemeral one-job runner on existing machine"
	modeOptionEphemeralBYODesc = "GitHub assigns one job then deregisters the runner. The machine is reused, so this is not a clean VM."
	modeOptionEphemeralCloudL  = "Ephemeral one-job cloud runner (Hetzner)"
	modeOptionEphemeralCloudD  = "Stronger isolation for risky workloads. Creates billable resources until `runnerkit destroy` verifies cleanup."
	modePersistentDefaultNote  = "Default mode: persistent for trusted private repositories."
	modeEphemeralModeNote      = "Ephemeral mode: one GitHub job, automatic deregistration, TTL cleanup, and preserved troubleshooting logs."
	modeNotFleetWarning        = "Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online."
	modeTTLSafeguardCopy       = "TTL safeguard: RunnerKit finalizes the ephemeral runner after 24h if no job completes."
	modeLogPreservationCopy    = "Log preservation copy: RunnerKit preserves best-effort runner _diag and systemd journal logs before cleanup."
	modeBYOEphemeralCaveat     = "BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine. Do not store unrelated secrets on the host."
	modeEphemeralCloudCostCopy = "Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup."
	modeTradeoffStepTitle      = "Runner mode selection"

	// Internal mode-selection values for the interactive Select prompt.
	modeChoicePersistentBYO  = "persistent-byo"
	modeChoiceEphemeralBYO   = "ephemeral-byo"
	modeChoiceEphemeralCloud = "ephemeral-cloud"
)

// resolveModeDecision normalizes the user-supplied --mode flag, prompts
// the developer to choose mode/profile when no host or cloud was supplied
// and the terminal is interactive, and returns the typed runmode.Decision
// that downstream rendering and safety enforcement consume. It runs before
// any GitHub auth, registration-token, remote, provider, or state action
// so error remediation surfaces the supported-modes copy without leaking
// side effects.
func resolveModeDecision(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, opts *upOptions, jsonOutput bool) (runmode.Decision, error) {
	rawMode := strings.TrimSpace(opts.mode)
	if rawMode != "" {
		mode, err := runmode.Normalize(rawMode)
		if err != nil {
			_ = renderer.Error("invalid_mode", "RunnerKit can't continue because --mode "+rawMode+" is not a supported mode.", []string{err.Error()})
			return runmode.Decision{}, NewExitError(ExitInvalidInput, err)
		}
		opts.mode = mode
	}
	hostSet := strings.TrimSpace(opts.host) != ""
	cloudSet := strings.TrimSpace(opts.cloud) != ""

	// Interactive mode prompt only runs when the user hasn't already
	// disambiguated the mode/profile via --mode or --cloud and we have a
	// usable TTY+prompter; --yes/--non-interactive/--json all skip the
	// prompt and fall back to the persistent default for trusted private
	// repos.
	if rawMode == "" && !hostSet && !cloudSet &&
		!jsonOutput && !opts.nonInteractive && !opts.yes &&
		deps.TTY.StdinTTY && deps.Prompts != nil {
		message := strings.Replace(modePromptMessage, "owner/name", repo.FullName, 1)
		choice, err := deps.Prompts.Select(ctx, ui.Prompt{Message: message}, []ui.Option{
			{Value: modeChoicePersistentBYO, Label: modeOptionPersistentLabel, Description: modeOptionPersistentDesc},
			{Value: modeChoiceEphemeralBYO, Label: modeOptionEphemeralBYOL, Description: modeOptionEphemeralBYODesc},
			{Value: modeChoiceEphemeralCloud, Label: modeOptionEphemeralCloudL, Description: modeOptionEphemeralCloudD},
		})
		if err != nil {
			return runmode.Decision{}, err
		}
		switch choice {
		case modeChoiceEphemeralCloud:
			opts.mode = runmode.ModeEphemeral
			opts.cloud = provider.HetznerProvider
		case modeChoiceEphemeralBYO:
			opts.mode = runmode.ModeEphemeral
		default:
			opts.mode = runmode.ModePersistent
		}
	}

	mode := opts.mode
	if mode == "" {
		mode = runmode.ModePersistent
	}
	setupPathHint := "byo"
	if strings.TrimSpace(opts.cloud) != "" {
		setupPathHint = "cloud"
	}
	decision := runmode.Evaluate(repo, runmode.Options{
		Mode:                  mode,
		SetupPath:             setupPathHint,
		AllowPersistentRisk:   opts.allowPublicRepoRisk,
		AllowEphemeralBYORisk: opts.allowEphemeralBYORisk,
	})
	return decision, nil
}

// renderModeTradeoffs writes the mode/safety profile tradeoff bullets and
// safety warnings before any setup-path mutation. JSON output emits a
// `mode_selection` payload with the same fields. The caller chooses
// whether to suppress this in pure JSON dry-run flows that already embed
// the same fields in the dry-run payload (see renderDryRun and
// renderCloudProvisionPlan).
func renderModeTradeoffs(renderer *ui.Renderer, jsonOutput bool, repo gh.Repo, decision runmode.Decision, ttl time.Duration) error {
	if jsonOutput {
		return nil
	}
	lines := []ui.Line{
		ui.Bullet("Mode: " + decision.Mode),
		ui.Bullet("Safety profile: " + decision.SafetyProfile),
		ui.Bullet("Cost: " + decision.Tradeoffs.Cost),
		ui.Bullet("Isolation: " + decision.Tradeoffs.Isolation),
		ui.Bullet("Cleanup: " + decision.Tradeoffs.Cleanup),
		ui.Bullet("Operations: " + decision.Tradeoffs.Operations),
		ui.Bullet("Logs: " + decision.Tradeoffs.Logs),
	}
	if len(decision.RecommendedFor) > 0 {
		lines = append(lines, ui.Bullet("Recommended for: "+strings.Join(decision.RecommendedFor, ", ")))
	}
	if len(decision.NotRecommendedFor) > 0 {
		lines = append(lines, ui.Bullet("Not recommended for: "+strings.Join(decision.NotRecommendedFor, ", ")))
	}
	switch decision.SafetyProfile {
	case runmode.ProfileEphemeralBYO:
		lines = append(lines,
			ui.WarningLine(modeBYOEphemeralCaveat),
			ui.WarningLine(modeNotFleetWarning),
			ui.Bullet(modeTTLSafeguardCopy),
			ui.Bullet(modeLogPreservationCopy),
		)
	case runmode.ProfileEphemeralCloud:
		costCopy := strings.Replace(modeEphemeralCloudCostCopy, "owner/name", repo.FullName, 1)
		lines = append(lines,
			ui.WarningLine(runmode.WarningEphemeralCloudBillable),
			ui.WarningLine(costCopy),
			ui.WarningLine(modeNotFleetWarning),
			ui.Bullet(modeTTLSafeguardCopy),
			ui.Bullet(modeLogPreservationCopy),
		)
	case runmode.ProfilePersistentTrusted:
		lines = append(lines, ui.Bullet(modePersistentDefaultNote))
	case runmode.ProfilePersistentRisky:
		lines = append(lines, ui.WarningLine(runmode.WarningPublicForkPersistent))
	}
	if decision.Mode == runmode.ModeEphemeral {
		lines = append(lines, ui.Bullet(modeEphemeralModeNote))
	}
	// Surface any decision-level warnings appended by the safety
	// enforcement step (e.g. public/fork ephemeral cloud recommends the
	// safer ephemeral cloud command). De-duplicate against the canonical
	// per-profile copy already rendered above so the same sentence does
	// not appear twice.
	rendered := map[string]bool{}
	for _, line := range lines {
		rendered[line.Text] = true
	}
	for _, warning := range decision.Warnings {
		if warning == "" || rendered[warning] {
			continue
		}
		lines = append(lines, ui.WarningLine(warning))
		rendered[warning] = true
	}
	_ = ttl // ttl is documented in modeTTLSafeguardCopy; future plans may render the exact configured value.
	return renderer.Step(1, 1, modeTradeoffStepTitle, lines...)
}

// modeSelectionPayload returns the canonical map of mode-selection JSON
// keys (mode, safety_profile, ephemeral, ttl, tradeoffs, recommended_for,
// not_recommended_for, warnings) so callers embed identical fields in
// every JSON payload that fronts mode-dependent mutation.
func modeSelectionPayload(decision runmode.Decision, ttl time.Duration) map[string]any {
	warnings := decision.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	recommended := decision.RecommendedFor
	if recommended == nil {
		recommended = []string{}
	}
	notRecommended := decision.NotRecommendedFor
	if notRecommended == nil {
		notRecommended = []string{}
	}
	return map[string]any{
		"mode":                decision.Mode,
		"safety_profile":      decision.SafetyProfile,
		"ephemeral":           decision.Mode == runmode.ModeEphemeral,
		"ttl":                 ttl.String(),
		"tradeoffs":           decision.Tradeoffs,
		"recommended_for":     recommended,
		"not_recommended_for": notRecommended,
		"warnings":            warnings,
	}
}

// shortEphemeralIDFn is the seam tests override to make ephemeral
// runner names deterministic. Production uses crypto/rand so
// consecutive ephemeral runs do not collide in GitHub.
var shortEphemeralIDFn = func() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a stable but non-empty value if randomness is
		// unavailable; uniqueness is best-effort and still better than a
		// fixed name.
		return "rk0001"
	}
	return hex.EncodeToString(b[:])
}

func shortEphemeralID() string { return shortEphemeralIDFn() }

// ephemeralSuffix returns a deterministic short id derived from now. It
// is exposed for tests that want a stable suffix; the runtime code uses
// shortEphemeralID for crypto/rand uniqueness.
func ephemeralSuffix(now time.Time) string {
	return strings.ToLower(now.UTC().Format("20060102t150405"))
}

// ephemeralLogArchivePath returns the canonical host path RunnerKit
// preserves _diag and journal logs to before cleanup.
func ephemeralLogArchivePath(runnerName string) string {
	return "/var/lib/runnerkit/ephemeral/" + runnerName + "/logs"
}

// ephemeralServiceName returns the canonical systemd unit name for an
// ephemeral runner.
func ephemeralServiceName(runnerName string) string {
	return "runnerkit-ephemeral." + runnerName + ".service"
}

// ephemeralCleanupCommand returns the cleanup command appropriate for
// the runner's setup path: `runnerkit destroy` for cloud (billable) or
// `runnerkit down` for BYO.
func ephemeralCleanupCommand(repoFullName string, cloud bool) string {
	if cloud {
		return "runnerkit destroy --repo " + repoFullName
	}
	return "runnerkit down --repo " + repoFullName
}

// ephemeralBootstrapOptions populates the ephemeral-specific fields of
// bootstrap.Options on top of the persistent defaults so ApplyEphemeral
// can write the one-shot service, finalizer, TTL timer, and log
// archive without diverging from the existing buildBootstrapOptions
// shape used by the BYO/cloud persistent paths.
func ephemeralBootstrapOptions(opts bootstrap.Options, ttl time.Duration) bootstrap.Options {
	if ttl == 0 {
		ttl = runmode.DefaultEphemeralTTL
	}
	opts.Mode = runmode.ModeEphemeral
	opts.EphemeralTTL = ttl
	opts.LogArchivePath = ephemeralLogArchivePath(opts.RunnerName)
	opts.FinalizerPath = "/usr/local/lib/runnerkit/ephemeral/" + opts.RunnerName + "/finalize.sh"
	opts.EphemeralServiceName = ephemeralServiceName(opts.RunnerName)
	opts.EphemeralTTLServiceName = "runnerkit-ephemeral." + opts.RunnerName + ".ttl.service"
	opts.EphemeralTTLTimerName = "runnerkit-ephemeral." + opts.RunnerName + ".ttl.timer"
	return opts
}

// buildModeLabelSet builds a labels.LabelSet for the chosen mode. The
// persistent path keeps the existing `runnerkit-owner-repo-local` runner
// name and persistent runs-on snippet; ephemeral mode uses
// `runnerkit-owner-repo-ephemeral-<short-id>` and emits the ephemeral
// runs-on snippet.
func buildModeLabelSet(repo gh.Repo, decision runmode.Decision, arch string) labels.LabelSet {
	if decision.Mode == runmode.ModeEphemeral {
		runnerName := labels.EphemeralRunnerName(repo, shortEphemeralID())
		return labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: arch, Mode: labels.ModeEphemeral, RunnerName: runnerName})
	}
	return labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: arch, Mode: labels.ModePersistent})
}

func resolveSetupPath(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, opts *upOptions, jsonOutput bool) (setupPath, error) {
	if strings.TrimSpace(opts.host) != "" {
		return setupPathBYO, nil
	}
	cloud := strings.ToLower(strings.TrimSpace(opts.cloud))
	if cloud != "" {
		if cloud == provider.HetznerProvider {
			return setupPathCloud, nil
		}
		_ = renderer.Error("invalid_cloud_provider", "RunnerKit does not support cloud provider "+opts.cloud+" in Phase 4.", []string{cloudUnsupportedCopy})
		return "", NewExitError(ExitInvalidInput, errors.New("unsupported cloud provider"))
	}
	// When mode is already chosen (via --mode or the interactive mode
	// prompt in resolveModeDecision), interpret a missing --cloud and
	// --host as the BYO path. resolveBYOTarget will prompt or error for
	// the SSH host as needed.
	if strings.TrimSpace(opts.mode) != "" {
		return setupPathBYO, nil
	}
	if !jsonOutput && !opts.nonInteractive && !opts.yes && deps.TTY.StdinTTY && deps.Prompts != nil {
		choice, err := deps.Prompts.Select(ctx, ui.Prompt{Message: "Choose setup path for `" + repo.FullName + "`:"}, []ui.Option{
			{Value: string(setupPathBYO), Label: "Use existing SSH host (BYO)"},
			{Value: string(setupPathCloud), Label: "Provision recommended cloud runner (Hetzner)"},
		})
		if err != nil {
			return "", err
		}
		if choice == string(setupPathCloud) {
			opts.cloud = provider.HetznerProvider
			return setupPathCloud, nil
		}
		return setupPathBYO, nil
	}
	_ = renderer.Error("input_required", cloudNoIntentCopy, []string{"Pass --host user@host for BYO setup or pass --cloud hetzner --yes to provision the recommended cloud runner."})
	return "", NewExitError(ExitInputRequired, errors.New(cloudNoIntentCopy))
}

func runCloudUp(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, decision gh.SafetyDecision, modeDecision runmode.Decision, opts *upOptions, jsonOutput bool) error {
	cloudProvider, ok := deps.Providers.Get(provider.HetznerProvider)
	if !ok || cloudProvider == nil {
		_ = renderer.Error("invalid_cloud_provider", "RunnerKit does not support cloud provider hetzner in Phase 4.", []string{cloudUnsupportedCopy})
		return NewExitError(ExitInvalidInput, errors.New("cloud provider hetzner not registered"))
	}
	status, err := deps.GitHub.VerifyRunnerManagementRead(ctx, repo)
	if err != nil || !status.OK {
		message := fmt.Sprintf("RunnerKit can't verify repository runner management read access for %s without minting a registration token.", repo.FullName)
		remediation := status.Remediation
		if len(remediation) == 0 {
			remediation = []string{gh.FineGrainedTokenRemediation(repo)}
		}
		// Append the stable RKD-AUTH-004 code + See: URL after the
		// existing remediation copy (D-15).
		remediation = append(remediation, errcodes.FormatLine(errcodes.AuthRunnerManagementPermissionDenied))
		_ = renderer.Error("github_permission_denied", message, remediation)
		if err == nil {
			err = errors.New(message)
		}
		return NewExitError(ExitGitHubAuth, err)
	}
	input := buildCloudProvisionInput(deps, repo, opts)
	if modeDecision.Mode == runmode.ModeEphemeral {
		input.Mode = runmode.ModeEphemeral
		runnerName := labels.EphemeralRunnerName(repo, shortEphemeralID())
		input.RunnerName = runnerName
		input.StateID = runnerName
		input.Labels = []string{"self-hosted", "runnerkit", labels.RepoScopedLabel(repo), labels.DefaultOS, labels.DefaultArch, labels.ModeEphemeral}
		input.WorkflowSnippet = labels.WorkflowSnippet(input.Labels)
	}
	registerKnownCloudProviderSecrets(renderer)
	validation, err := cloudProvider.Validate(ctx, input)
	if err != nil || !validation.OK {
		message := "Hetzner credentials are missing. Export HCLOUD_TOKEN or HETZNER_CLOUD_TOKEN, then rerun runnerkit up --repo " + repo.FullName + " --cloud hetzner."
		remediation := validation.Remediation
		if len(remediation) == 0 {
			remediation = []string{"Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>", "Re-run runnerkit up --repo " + repo.FullName + " --cloud hetzner"}
		}
		// Append the stable RKD-PROV-004 code + See: URL after the
		// existing remediation copy (D-15). Append (not prepend) so
		// existing tests that index remediation[0] keep working.
		remediation = append(remediation, errcodes.FormatLine(errcodes.ProvHCloudTokenMissing))
		_ = renderer.Error("provider_credentials_missing", message, remediation)
		if err == nil {
			err = errors.New(message)
		}
		return NewExitError(ExitInputRequired, err)
	}
	plan, err := cloudProvider.Plan(ctx, input)
	if err != nil {
		_ = renderer.Error("cloud_plan_failed", "RunnerKit could not build the cloud provisioning plan.", []string{err.Error()})
		return NewExitError(ExitSafetyGate, err)
	}
	if opts.dryRun {
		if err := renderModeTradeoffs(renderer, jsonOutput, repo, modeDecision, opts.ephemeralTTL); err != nil {
			return err
		}
		return renderCloudProvisionPlan(renderer, jsonOutput, repo, plan, modeDecision, opts.ephemeralTTL)
	}
	store := rkstate.NewStore(deps.StateBaseDir)
	replaceExisting, err := confirmCloudStateReplaceBeforeProvision(ctx, deps, renderer, opts, jsonOutput, store, repo.FullName)
	if err != nil {
		return err
	}
	if err := confirmCloudProvisionPlan(ctx, deps, renderer, opts, jsonOutput, repo, plan, modeDecision); err != nil {
		return err
	}
	result, err := cloudProvider.Provision(ctx, input)
	if err != nil {
		var provisionErr *provider.ProvisionError
		if errors.As(err, &provisionErr) && len(provisionErr.Result.CreatedResourceIDs) > 0 {
			if saveErr := saveCloudPendingRepository(ctx, deps, renderer, opts, jsonOutput, store, repo, status.Source, decision, input, plan, provisionErr.Result, cloudProvisionPending, "provider", replaceExisting); saveErr != nil {
				return saveErr
			}
			_ = renderer.Error("cloud_provision_failed", "Hetzner provisioning failed after billable resources were created.", []string{err.Error(), "Next action: runnerkit destroy --repo " + repo.FullName})
			return NewExitError(ExitSafetyGate, err)
		}
		_ = renderer.Error("cloud_provision_failed", "Hetzner provisioning failed before a complete cloud machine was ready.", []string{err.Error(), "Re-run with --dry-run to preview the cloud provisioning plan.", "If billable resources were created, run runnerkit destroy --repo " + repo.FullName + "."})
		return NewExitError(ExitSafetyGate, err)
	}
	if result.CheckpointRequired {
		if err := saveCloudPendingRepository(ctx, deps, renderer, opts, jsonOutput, store, repo, status.Source, decision, input, plan, result, cloudProvisionPending, "provider", replaceExisting); err != nil {
			return err
		}
	}
	readyMachine, err := cloudProvider.WaitReady(ctx, result.Machine)
	if err != nil {
		readinessResult := result
		readinessResult.Machine = readyMachine
		_ = saveCloudPendingRepository(ctx, deps, renderer, opts, jsonOutput, store, repo, status.Source, decision, input, plan, readinessResult, cloudReadinessPending, "readiness", true)
		return renderCloudReadinessFailure(renderer, repo, err)
	}
	report, hostKey, readyMachine, err := waitCloudTargetReady(ctx, deps, readyMachine)
	if err != nil {
		readinessResult := result
		readinessResult.Machine = readyMachine
		_ = saveCloudPendingRepository(ctx, deps, renderer, opts, jsonOutput, store, repo, status.Source, decision, input, plan, readinessResult, cloudReadinessPending, "readiness", true)
		return renderCloudReadinessFailure(renderer, repo, err)
	}
	arch := defaultString(report.Arch, labels.DefaultArch)
	labelSet := labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: arch, Mode: labels.DefaultMode})
	if modeDecision.Mode == runmode.ModeEphemeral {
		labelSet = labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: arch, Mode: labels.ModeEphemeral, RunnerName: input.RunnerName})
	}
	runnerPkg, err := bootstrap.PackageFor("linux", arch)
	if err != nil {
		_ = renderer.Error("unsupported_runner_package", err.Error(), []string{"Use linux/x64 or linux/arm64 for the recommended cloud runner path."})
		return NewExitError(ExitSafetyGate, err)
	}
	bootstrapOpts := buildBootstrapOptions(repo, labelSet, runnerPkg, report)
	if modeDecision.Mode == runmode.ModeEphemeral {
		bootstrapOpts = ephemeralBootstrapOptions(bootstrapOpts, opts.ephemeralTTL)
	}

	runners, err := deps.GitHub.ListRunners(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_runner_list_failed", "RunnerKit can't list repository self-hosted runners.", []string{"Verify GitHub credentials can list repository runners for " + repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	if existingRunner, found := gh.FindRunnerByName(runners, labelSet.RunnerName); found {
		// Bug 17 (Plan 06-07 attempt-14, 2026-05-06): only refuse when the
		// existing runner is NOT one of ours. The runner name is deterministic
		// per (repo, host, mode), so a re-run of `runnerkit up` always sees its
		// own previously-registered runner. config.sh --replace handles
		// re-registration end-to-end.
		if !isRunnerKitManagedRunner(existingRunner) {
			return runnerNameConflict(renderer, labelSet.RunnerName, existingRunner)
		}
	}
	token, err := deps.GitHub.CreateRegistrationToken(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_permission_denied", "RunnerKit can't create a fresh runner registration token.", []string{gh.FineGrainedTokenRemediation(repo)})
		return NewExitError(ExitGitHubAuth, err)
	}
	renderer.Redactor().Register(redact.RunnerRegistrationToken, token.Token)
	bootstrapOpts.RunnerToken = token.Token
	if modeDecision.Mode == runmode.ModeEphemeral {
		if _, err := bootstrap.ApplyEphemeral(ctx, deps.RemoteExecutor, readyMachine.Target, bootstrapOpts); err != nil {
			var serviceErr bootstrap.ServiceNotActiveError
			if errors.As(err, &serviceErr) {
				_ = renderer.Error("runner_service_not_active", "RunnerKit installed the ephemeral cloud runner but the service is not active.", []string{"Run systemctl status " + bootstrapOpts.EphemeralServiceName + " on the host or run runnerkit logs --repo " + repo.FullName + " --since 30m after fixing the service."})
				return NewExitError(ExitSafetyGate, err)
			}
			_ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the ephemeral cloud runner install plan.", []string{"Review the remote host output, fix the issue, and run runnerkit destroy --repo " + repo.FullName + " if you need to stop billing."})
			return NewExitError(ExitSafetyGate, err)
		}
	} else {
		if _, err := bootstrap.Apply(ctx, deps.RemoteExecutor, readyMachine.Target, bootstrapOpts); err != nil {
			var serviceErr bootstrap.ServiceNotActiveError
			if errors.As(err, &serviceErr) {
				_ = renderer.Error("runner_service_not_active", "RunnerKit installed the cloud runner but the service is not active.", []string{"Run sudo ./svc.sh status in the runner directory or run runnerkit logs --repo " + repo.FullName + " --since 30m after fixing the service."})
				return NewExitError(ExitSafetyGate, err)
			}
			_ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the cloud runner install plan.", []string{"Review the remote host output, fix the issue, and run runnerkit destroy --repo " + repo.FullName + " if you need to stop billing."})
			return NewExitError(ExitSafetyGate, err)
		}
	}
	onlineRunner, ok, err := waitForRunnerOnline(ctx, deps, repo, labelSet.RunnerName, labelSet.Labels)
	if err != nil {
		return err
	}
	if !ok {
		_ = renderer.Error("runner_online_timeout", "RunnerKit could not verify the cloud GitHub runner came online with the expected labels.", []string{"Check runnerkit logs --repo " + repo.FullName + " --since 30m, then run runnerkit destroy --repo " + repo.FullName + " if you need to stop billing."})
		return NewExitError(ExitSafetyGate, errors.New("runner_online_timeout"))
	}

	finalResult := result
	finalResult.Machine = readyMachine
	if modeDecision.Mode == runmode.ModeEphemeral {
		repoState := buildEphemeralCloudRepositoryState(deps, repo, status.Source, decision, modeDecision, labelSet, readyMachine.Target, hostKey, input, plan, finalResult, bootstrapOpts, onlineRunner, opts.sshKey, opts.ephemeralTTL)
		if err := store.SaveRepository(repoState, true); err != nil {
			_ = renderer.Error("state_io_failed", "RunnerKit can't save final ephemeral cloud runner state.", []string{"Check permissions for " + store.Path() + " and run runnerkit destroy --repo " + repo.FullName + " if billable resources were created."})
			return NewExitError(ExitStateIO, err)
		}
		if jsonOutput {
			return renderer.JSON(ephemeralCompletionJSON(repo.FullName, modeDecision, mergeWarnings(decision.Warnings, modeDecision.Warnings), store.Path(), readyMachine.Target, labelSet, bootstrapOpts, onlineRunner, opts.ephemeralTTL, true))
		}
		return renderEphemeralCompletionHuman(renderer, repo.FullName, modeDecision, mergeWarnings(decision.Warnings, modeDecision.Warnings), store.Path(), readyMachine.Target, labelSet, bootstrapOpts, onlineRunner, opts.ephemeralTTL, true)
	}
	repoState := buildCloudRepositoryState(deps, repo, status.Source, decision, labelSet, readyMachine.Target, hostKey, input, plan, finalResult, bootstrapOpts, onlineRunner, opts.sshKey)
	if err := store.SaveRepository(repoState, true); err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't save final cloud runner state.", []string{"Check permissions for " + store.Path() + " and run runnerkit destroy --repo " + repo.FullName + " if billable resources were created."})
		return NewExitError(ExitStateIO, err)
	}
	if jsonOutput {
		return renderer.JSON(cloudCompletionJSON(repo.FullName, store.Path(), plan, finalResult, labelSet, bootstrapOpts, onlineRunner))
	}
	return renderCloudCompletionHuman(renderer, decision.Warnings, store.Path(), plan, finalResult, labelSet, bootstrapOpts, onlineRunner)
}

func registerKnownCloudProviderSecrets(renderer *ui.Renderer) {
	for _, key := range []string{"HCLOUD_TOKEN", "HETZNER_CLOUD_TOKEN"} {
		renderer.Redactor().Register(redact.ProviderCredential, os.Getenv(key))
	}
}

func buildCloudProvisionInput(deps Dependencies, repo gh.Repo, opts *upOptions) provider.ProvisionInput {
	profile := provider.DefaultHetznerProfile()
	profile.Region = defaultString(opts.cloudRegion, provider.HetznerDefaultRegion)
	profile.ServerType = defaultString(opts.cloudProfile, provider.HetznerDefaultServerType)
	labelSet := labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: labels.DefaultArch, Mode: labels.DefaultMode})
	createdAt := deps.Clock()
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return provider.ProvisionInput{
		RepoFullName:    repo.FullName,
		RunnerName:      labelSet.RunnerName,
		Labels:          labelSet.Labels,
		WorkflowSnippet: labelSet.RunsOnYAML,
		Profile:         profile,
		SSHAllowedCIDR:  defaultString(opts.sshAllowedCIDR, provider.HetznerDefaultSSHAllowedCIDR),
		PublicKey:       resolveCloudPublicKey(opts),
		StateID:         labelSet.RunnerName,
		CreatedAt:       createdAt,
	}
}

func resolveCloudPublicKey(opts *upOptions) string {
	candidates := []string{}
	if opts != nil && strings.TrimSpace(opts.sshKey) != "" {
		keyPath := strings.TrimSpace(opts.sshKey)
		if strings.HasSuffix(keyPath, ".pub") {
			candidates = append(candidates, keyPath)
		} else {
			candidates = append(candidates, keyPath+".pub")
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".ssh", "id_ed25519.pub"), filepath.Join(home, ".ssh", "id_rsa.pub"))
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		if publicKey := strings.TrimSpace(string(data)); publicKey != "" {
			return publicKey
		}
	}
	return ""
}

func renderCloudReadinessFailure(renderer *ui.Renderer, repo gh.Repo, cause error) error {
	message := strings.Replace(cloudReadinessFailedMessage, "owner/name", repo.FullName, 1)
	remediation := []string{"Run runnerkit destroy --repo " + repo.FullName + " if billable Hetzner resources were created and you want to stop billing."}
	if cause != nil {
		remediation = append([]string{cause.Error()}, remediation...)
	}
	_ = renderer.Error("cloud_readiness_failed", message, remediation)
	return NewExitError(ExitSafetyGate, errors.New("cloud_readiness_failed"))
}

func waitCloudTargetReady(ctx context.Context, deps Dependencies, machine provider.Machine) (preflight.Report, remote.HostKey, provider.Machine, error) {
	target := machine.Target
	if target.User == "" {
		target.User = provider.HetznerDefaultSSHUser
	}
	if target.Port == 0 {
		target.Port = 22
	}
	machine.Target = target
	hostKey, err := probeCloudHostKey(ctx, deps.RemoteExecutor, target)
	if err != nil {
		return preflight.Report{}, remote.HostKey{}, machine, err
	}
	hostKey = remote.NormalizeHostKey(hostKey)
	if hostKey.Fingerprint == "" {
		return preflight.Report{}, remote.HostKey{}, machine, errors.New("SSH host key was not observed for cloud machine")
	}
	// Bug 29 (Plan 06-12, 2026-05-06): give cloud-init readiness an
	// explicit Timeout aligned with hetzner.HostKeyProbeOptions
	// (Plan 06-10 Bug 22 — 60×5s = 300s) so cloud-up has a single
	// coherent wall-clock for SSH host-key install + cloud-init
	// completion. Plan 06-07 attempt-17 cloud smoke aborted at 42s
	// with cloud_readiness_failed because this command had no Timeout
	// — Hetzner cpx22 + ubuntu-24.04 cloud-init typically needs
	// 60-120s. RUNNERKIT_CLOUD_INIT_TIMEOUT overrides for slower
	// regions / images.
	// Run cloud-init wait as root: Hetzner injects the uploaded SSH key for
	// root at provision time, while the configured profile SSH user
	// (runnerkit-admin) is created by cloud-init itself. Waiting as root avoids
	// an auth race where runnerkit-admin does not exist yet.
	waitTarget := target
	waitTarget.User = "root"
	result, err := runCloudInitWaitWithRetry(ctx, deps, waitTarget)
	if err != nil {
		stderr := strings.TrimSpace(result.Stderr)
		if stderr != "" {
			return preflight.Report{}, hostKey, machine, remote.RemoteError{
				CommandID: "cloud.cloudinit.wait",
				ExitCode:  result.ExitCode,
				Message:   "cloud-init readiness failed: " + stderr,
			}
		}
		return preflight.Report{}, hostKey, machine, err
	}
	if result.ExitCode != 0 {
		return preflight.Report{}, hostKey, machine, remote.RemoteError{CommandID: "cloud.cloudinit.wait", ExitCode: result.ExitCode, Message: "cloud-init readiness failed"}
	}
	report, err := preflight.Run(ctx, deps.RemoteExecutor, target, preflight.Options{AllowUnknownLinux: false, RequirePasswordlessSudo: true})
	if err != nil {
		return report, hostKey, machine, err
	}
	if !report.Passed() {
		return report, hostKey, machine, errors.New("cloud preflight failed before runner registration")
	}
	return report, hostKey, machine, nil
}

// runCloudInitWaitWithRetry tolerates transient SSH transport failures while
// cloud-init is still converging. In live Hetzner smoke runs the host can pass
// provider readiness yet still return intermittent ssh exit-status-255 failures
// for the first few attempts.
//
// cloudInitWaitScript must not use `cloud-init status --wait || test -f
// /var/lib/cloud/instance/boot-finished`: when runcmd fails (e.g. visudo on
// the RunnerKit sudoers drop-in), cloud-init can still leave boot-finished
// present while status is `error`, and the `||` branch would exit 0 — SSH
// bootstrap then hits `sudo: a password is required` within seconds.
func cloudInitWaitScript() string {
	return `set -euo pipefail
if command -v cloud-init >/dev/null 2>&1; then
  cloud-init status --wait || {
    echo "runnerkit: cloud-init status --wait failed" >&2
    cloud-init status --long 2>&1 || true
    exit 1
  }
  if cloud-init status 2>/dev/null | tr -d '\r' | grep -q '^status: error'; then
    echo "runnerkit: cloud-init finished with status=error (check runcmd / visudo for /etc/sudoers.d/runnerkit-installer)" >&2
    cloud-init status --long 2>&1 || true
    exit 1
  fi
  if ! cloud-init status 2>/dev/null | tr -d '\r' | head -n 1 | grep -qE '^status: (done|disabled)'; then
    echo "runnerkit: cloud-init did not reach done/disabled after --wait" >&2
    cloud-init status --long 2>&1 || true
    exit 1
  fi
  exit 0
fi
if test -f /var/lib/cloud/instance/boot-finished; then
  exit 0
fi
echo "runnerkit: cloud-init missing and boot-finished not present" >&2
exit 1
`
}

func runCloudInitWaitWithRetry(ctx context.Context, deps Dependencies, target remote.Target) (remote.Result, error) {
	timeout := cloudInitTimeoutFromEnv()
	command := remote.Command{
		ID:      "cloud.cloudinit.wait",
		Script:  cloudInitWaitScript(),
		Timeout: timeout,
	}
	deadline := time.Now().Add(timeout)
	interval := 2 * time.Second
	var lastErr error
	for {
		result, err := deps.RemoteExecutor.Run(ctx, target, command)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return result, lastErr
		}
		if sleepErr := deps.Sleep(ctx, interval); sleepErr != nil {
			return result, sleepErr
		}
	}
}

// defaultCloudInitTimeout is intentionally longer than the host-key
// probe budget to absorb slower cloud-init/user-data convergence under
// real Hetzner load. Host key visibility can precede SSH auth readiness,
// so cloud.cloudinit.wait must have additional runway.
const defaultCloudInitTimeout = 10 * time.Minute

// cloudInitTimeoutFromEnv resolves RUNNERKIT_CLOUD_INIT_TIMEOUT into a
// usable Duration. Empty / unparseable / non-positive values fall back
// to defaultCloudInitTimeout. Bug 29 (Plan 06-12, 2026-05-06): the live
// attempt-17 smoke aborted at 42s because the cloud.cloudinit.wait
// command had no explicit Timeout. The default 10m gives Hetzner
// cpx22 + ubuntu-24.04 cloud-init + SSH-auth convergence headroom; smoke
// harnesses can override with a smaller value via the env var.
func cloudInitTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("RUNNERKIT_CLOUD_INIT_TIMEOUT"))
	if raw == "" {
		return defaultCloudInitTimeout
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultCloudInitTimeout
	}
	return parsed
}

// probeCloudHostKey runs the cloud SSH host-key readiness probe with
// retry/backoff so cloud-init's host-key install window (~30-90s on a
// fresh Ubuntu 24.04 image) does not abort `runnerkit up --cloud
// hetzner` after billable resources have already been created. Bug 22
// (Plan 06-10, 2026-05-06).
func probeCloudHostKey(ctx context.Context, executor remote.Executor, target remote.Target) (remote.HostKey, error) {
	if prober, ok := executor.(remote.HostKeyProber); ok {
		return hetzner.ProbeHostKeyWithRetry(ctx, prober, target, hetzner.HostKeyProbeOptions{})
	}
	probe, err := executor.Probe(ctx, target)
	if err != nil {
		return remote.HostKey{}, err
	}
	return probe.HostKey, nil
}

func renderCloudProvisionPlan(renderer *ui.Renderer, jsonOutput bool, repo gh.Repo, plan provider.ProvisionPlan, modeDecision runmode.Decision, ttl time.Duration) error {
	caveat := defaultString(plan.CostEstimateCaveat, cloudCostCaveatCopy)
	// Override the persistent default labels/snippet for ephemeral cloud
	// dry-run output so the rendered runs-on snippet matches the chosen
	// mode. The provider plan is generated before the mode label set is
	// applied to ProvisionInput in 05-02; for 05-01 we only need to make
	// sure the dry-run output advertises the ephemeral runs-on snippet.
	planLabels := plan.Labels
	planSnippet := plan.WorkflowSnippet
	if modeDecision.Mode == runmode.ModeEphemeral {
		ephemeralLabels := []string{"self-hosted", "runnerkit", labels.RepoScopedLabel(repo), labels.DefaultOS, labels.DefaultArch, labels.ModeEphemeral}
		planLabels = ephemeralLabels
		planSnippet = labels.WorkflowSnippet(ephemeralLabels)
	}
	if jsonOutput {
		payload := map[string]any{
			"ok":               true,
			"command":          "up",
			"repo":             repo.FullName,
			"cloud_plan":       plan,
			"runner_installed": false,
			"state_saved":      false,
			"workflow_snippet": planSnippet,
			"labels":           planLabels,
		}
		for k, v := range modeSelectionPayload(modeDecision, ttl) {
			payload[k] = v
		}
		return renderer.JSON(payload)
	}
	costRepoCopy := strings.Replace(modeEphemeralCloudCostCopy, "owner/name", repo.FullName, 1)
	lines := []ui.Line{
		ui.WarningLine(caveat),
		ui.Bullet("Provider: " + plan.Provider),
		ui.Bullet("Region: " + plan.Region),
		ui.Bullet("Server type: " + plan.ServerType),
		ui.Bullet("Image: " + plan.Image),
		ui.Bullet("Estimated cost: " + plan.EstimatedHourlyCost + ", " + plan.EstimatedMonthlyCost),
		ui.Bullet("Resources: server, SSH key, firewall, public IPv4/IPv6"),
		ui.Bullet("Not created: backups, snapshots, volumes, floating IPs"),
		ui.Bullet("Resource names: " + formatCloudResourceNames(plan.ResourceNames)),
		ui.Bullet("Tags: " + formatCloudTags(plan.Tags)),
		ui.Bullet("SSH allowed CIDR: " + plan.SSHAllowedCIDR),
		ui.Bullet("Labels: [" + strings.Join(planLabels, ", ") + "]"),
		ui.Bullet(planSnippet),
	}
	if modeDecision.SafetyProfile == runmode.ProfileEphemeralCloud {
		lines = append(lines,
			ui.WarningLine(costRepoCopy),
			ui.Bullet(modeTTLSafeguardCopy),
		)
	}
	lines = append(lines, ui.Next("Future cleanup: "+plan.FutureDestroyCommand))
	return renderer.Step(1, 1, cloudProvisioningPlanTitle, lines...)
}

func formatCloudResourceNames(names map[string]string) string {
	if len(names) == 0 {
		return "server, ssh_key, firewall"
	}
	parts := []string{}
	for _, key := range []string{"server", "ssh_key", "firewall"} {
		if value := names[key]; value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	return strings.Join(parts, ", ")
}

func formatCloudTags(tags map[string]string) string {
	if len(tags) == 0 {
		return "runnerkit=true, managed=true"
	}
	parts := []string{}
	for _, key := range []string{"runnerkit", "managed", "repo", "runner", "state_id", "mode", "created_at"} {
		if value := tags[key]; value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	return strings.Join(parts, ", ")
}

func confirmCloudProvisionPlan(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, repo gh.Repo, plan provider.ProvisionPlan, modeDecision runmode.Decision) error {
	if err := renderCloudProvisionPlan(renderer, jsonOutput, repo, plan, modeDecision, opts.ephemeralTTL); err != nil {
		return err
	}
	if opts.yes {
		return nil
	}
	if jsonOutput || opts.nonInteractive || !deps.TTY.StdinTTY || deps.Prompts == nil {
		message := "RunnerKit can't continue because creating billable Hetzner resources requires confirmation."
		_ = renderer.Error("input_required", message, []string{cloudProvisionConfirmationRemedy})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Create billable Hetzner resources for `" + repo.FullName + "`?", Default: false})
	if err != nil {
		return err
	}
	if !confirmed {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return NewExitError(ExitCanceled, errors.New(message))
	}
	return nil
}

func confirmCloudStateReplaceBeforeProvision(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, store rkstate.Store, fullName string) (bool, error) {
	if _, exists, err := store.GetRepository(fullName); err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + " and re-run runnerkit up."})
		return false, NewExitError(ExitStateIO, err)
	} else if exists {
		if opts.replace {
			return true, nil
		}
		return confirmStateReplace(ctx, deps, renderer, opts, fullName, jsonOutput)
	}
	return false, nil
}

func saveCloudPendingRepository(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, store rkstate.Store, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, input provider.ProvisionInput, plan provider.ProvisionPlan, result provider.ProvisionResult, checkpointMessage string, artifact string, replace bool) error {
	repoState := buildCloudPendingRepositoryState(deps, repo, source, decision, input, plan, result, checkpointMessage, artifact, opts.sshKey)
	if err := store.SaveRepository(repoState, replace); err != nil {
		if errors.Is(err, rkstate.ErrRepositoryExists) {
			return replacementRequired(renderer, repo.FullName)
		}
		_ = renderer.Error("state_io_failed", "RunnerKit can't save cloud provisioning state.", []string{"Check permissions for " + store.Path() + " and run runnerkit destroy --repo " + repo.FullName + " if billable resources were created."})
		return NewExitError(ExitStateIO, err)
	}
	return nil
}

func buildCloudPendingRepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, input provider.ProvisionInput, plan provider.ProvisionPlan, result provider.ProvisionResult, checkpointMessage string, artifact string, keyPathRef string) rkstate.RepositoryState {
	now := deps.Clock()
	if now.IsZero() {
		now = time.Now()
	}
	labelSet := labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: labels.DefaultArch, Mode: labels.DefaultMode})
	if input.RunnerName != "" {
		labelSet.RunnerName = input.RunnerName
	}
	if len(input.Labels) > 0 {
		labelSet.Labels = append([]string(nil), input.Labels...)
	}
	if input.WorkflowSnippet != "" {
		labelSet.RunsOnYAML = input.WorkflowSnippet
	}
	resourceIDs := mergeCloudResourceIDs(result)
	providerRef := result.Machine.Provider
	providerRef.Kind = defaultString(providerRef.Kind, provider.HetznerProvider)
	providerRef.Name = defaultString(providerRef.Name, provider.HetznerProvider)
	providerRef.Region = defaultString(providerRef.Region, plan.Region)
	providerRef.Profile = defaultString(providerRef.Profile, plan.ServerType)
	if providerRef.IDs == nil {
		providerRef.IDs = cloneStringMap(resourceIDs)
	}
	providerRef.ResourceIDs = cloneStringMap(resourceIDs)
	providerRef.Tags = cloneStringMap(plan.Tags)
	providerRef.Cloud = mergeCloudInventory(providerRef.Cloud, result, plan, resourceIDs)
	return rkstate.RepositoryState{
		Repo: repo,
		Auth: rkstate.AuthReference{Source: defaultString(source.Kind, "gh"), Reference: defaultString(source.Reference, source.Kind)},
		Runner: rkstate.RunnerIdentity{
			Name:            labelSet.RunnerName,
			Labels:          labelSet.Labels,
			WorkflowSnippet: labelSet.RunsOnYAML,
			Mode:            labels.DefaultMode,
			OS:              labels.DefaultOS,
			Arch:            labels.DefaultArch,
		},
		Machine: rkstate.MachineRef{
			Kind:       "cloud-ssh",
			HostRef:    result.Machine.Target.Display(),
			User:       result.Machine.Target.User,
			Port:       result.Machine.Target.Port,
			KeyPathRef: keyPathRef,
		},
		Provider: providerRef,
		Cleanup: rkstate.CleanupMetadata{
			ManagedPaths:        []string{},
			ProviderResourceIDs: cloudProviderResourceIDList(resourceIDs),
			Notes:               []string{checkpointMessage},
		},
		Safety:           cloudSafetyMetadata(decision, now),
		Operations:       []rkstate.OperationCheckpoint{{Command: "up", Artifact: artifact, Status: "pending", Message: checkpointMessage, UpdatedAt: now}},
		RunnerKitVersion: deps.Version,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func buildCloudRepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, labelSet labels.LabelSet, target remote.Target, hostKey remote.HostKey, input provider.ProvisionInput, plan provider.ProvisionPlan, result provider.ProvisionResult, opts bootstrap.Options, onlineRunner gh.Runner, keyPathRef string) rkstate.RepositoryState {
	now := deps.Clock()
	if now.IsZero() {
		now = time.Now()
	}
	if labelSet.RunnerName == "" {
		labelSet = labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: opts.Package.Arch, Mode: labels.DefaultMode})
	}
	if len(input.Labels) > 0 {
		labelSet.Labels = append([]string(nil), input.Labels...)
	}
	if input.WorkflowSnippet != "" {
		labelSet.RunsOnYAML = input.WorkflowSnippet
	}
	if target.User == "" {
		target.User = provider.HetznerDefaultSSHUser
	}
	if target.Port == 0 {
		target.Port = 22
	}
	resourceIDs := mergeCloudResourceIDs(result)
	providerRef := result.Machine.Provider
	providerRef.Kind = defaultString(providerRef.Kind, provider.HetznerProvider)
	providerRef.Name = defaultString(providerRef.Name, provider.HetznerProvider)
	providerRef.Region = defaultString(providerRef.Region, plan.Region)
	providerRef.Profile = defaultString(providerRef.Profile, plan.ServerType)
	if providerRef.IDs == nil {
		providerRef.IDs = cloneStringMap(resourceIDs)
	}
	providerRef.ResourceIDs = cloneStringMap(resourceIDs)
	providerRef.Tags = cloneStringMap(plan.Tags)
	providerRef.Cloud = mergeCloudInventory(providerRef.Cloud, result, plan, resourceIDs)
	if providerRef.Cloud.PublicIPv4 == "" {
		providerRef.Cloud.PublicIPv4 = result.Machine.PublicIPv4
	}
	if providerRef.Cloud.PublicIPv6 == "" {
		providerRef.Cloud.PublicIPv6 = result.Machine.PublicIPv6
	}
	if providerRef.Cloud.ServerStatus == "" || providerRef.Cloud.ServerStatus == "provisioning" {
		providerRef.Cloud.ServerStatus = "running"
	}

	return rkstate.RepositoryState{
		Repo: repo,
		Auth: rkstate.AuthReference{Source: defaultString(source.Kind, "gh"), Reference: defaultString(source.Reference, source.Kind)},
		Runner: rkstate.RunnerIdentity{
			Name:            labelSet.RunnerName,
			Labels:          append([]string(nil), labelSet.Labels...),
			WorkflowSnippet: labelSet.RunsOnYAML,
			Mode:            labels.DefaultMode,
			OS:              labels.DefaultOS,
			Arch:            defaultString(opts.Package.Arch, labels.DefaultArch),
		},
		Machine: rkstate.MachineRef{
			Kind:               "cloud-ssh",
			HostRef:            target.Display(),
			User:               target.User,
			Port:               target.Port,
			KeyPathRef:         keyPathRef,
			HostKeyAlgorithm:   hostKey.Algorithm,
			HostKeyFingerprint: hostKey.Fingerprint,
			InstallPath:        opts.InstallPath,
			WorkDir:            opts.WorkDir,
			ServiceName:        runnerServiceName(labelSet.RunnerName),
		},
		Provider: providerRef,
		Cleanup: rkstate.CleanupMetadata{
			GitHubRunnerID:      onlineRunner.ID,
			ManagedPaths:        []string{opts.InstallPath, opts.WorkDir},
			ProviderResourceIDs: cloudProviderResourceIDList(resourceIDs),
		},
		Safety:           cloudSafetyMetadata(decision, now),
		Operations:       []rkstate.OperationCheckpoint{},
		RunnerKitVersion: deps.Version,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func mergeCloudResourceIDs(result provider.ProvisionResult) map[string]string {
	out := map[string]string{}
	for _, source := range []map[string]string{result.Machine.Provider.IDs, result.Machine.Provider.ResourceIDs, result.Machine.ResourceIDs, result.CreatedResourceIDs} {
		for k, v := range source {
			if strings.TrimSpace(v) != "" {
				out[k] = v
			}
		}
	}
	return out
}

func mergeCloudInventory(existing rkstate.CloudInventory, result provider.ProvisionResult, plan provider.ProvisionPlan, resourceIDs map[string]string) rkstate.CloudInventory {
	cloud := existing
	cloud.Provider = defaultString(cloud.Provider, provider.HetznerProvider)
	cloud.ServerID = defaultString(cloud.ServerID, resourceIDs["server"])
	cloud.ServerName = defaultString(cloud.ServerName, plan.ResourceNames["server"])
	cloud.ServerStatus = defaultString(cloud.ServerStatus, "provisioning")
	cloud.Region = defaultString(cloud.Region, plan.Region)
	cloud.ServerType = defaultString(cloud.ServerType, plan.ServerType)
	cloud.Image = defaultString(cloud.Image, plan.Image)
	cloud.PublicIPv4 = defaultString(cloud.PublicIPv4, result.Machine.PublicIPv4)
	cloud.PublicIPv6 = defaultString(cloud.PublicIPv6, result.Machine.PublicIPv6)
	cloud.PrimaryIPv4ID = defaultString(cloud.PrimaryIPv4ID, resourceIDs["primary_ipv4"])
	cloud.PrimaryIPv6ID = defaultString(cloud.PrimaryIPv6ID, resourceIDs["primary_ipv6"])
	cloud.SSHKeyID = defaultString(cloud.SSHKeyID, resourceIDs["ssh_key"])
	cloud.SSHKeyName = defaultString(cloud.SSHKeyName, plan.ResourceNames["ssh_key"])
	cloud.FirewallID = defaultString(cloud.FirewallID, resourceIDs["firewall"])
	cloud.FirewallName = defaultString(cloud.FirewallName, plan.ResourceNames["firewall"])
	if cloud.Tags == nil {
		cloud.Tags = cloneStringMap(plan.Tags)
	}
	if cloud.CostProfile.Provider == "" {
		cloud.CostProfile = rkstate.CostProfileRef{Provider: plan.Provider, Region: plan.Region, ServerType: plan.ServerType, Image: plan.Image, EstimatedHourlyCost: plan.EstimatedHourlyCost, EstimatedMonthlyCost: plan.EstimatedMonthlyCost, Caveat: plan.CostEstimateCaveat}
	}
	cloud.CloudInitVersion = defaultString(cloud.CloudInitVersion, hetzner.CloudInitUserDataVersion)
	return cloud
}

func cloudProviderResourceIDList(resourceIDs map[string]string) []string {
	out := []string{}
	for _, key := range []string{"server", "ssh_key", "firewall", "primary_ipv4", "primary_ipv6"} {
		if value := resourceIDs[key]; strings.TrimSpace(value) != "" {
			out = append(out, key+":"+value)
		}
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloudSafetyMetadata(decision gh.SafetyDecision, now time.Time) rkstate.SafetyMetadata {
	safety := rkstate.SafetyMetadata{Code: decision.Code, Allowed: decision.Allowed, Warnings: decision.Warnings}
	if decision.Code == gh.SafetyCodePublicRisk && decision.Allowed && len(decision.Warnings) > 0 {
		safety.AcceptedOverride = gh.AllowPublicRepoRiskFlag
		safety.AcceptedAt = &now
	}
	return safety
}

func resolveBYOTarget(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool) (remote.Target, error) {
	raw := strings.TrimSpace(opts.host)
	if raw == "" {
		if jsonOutput || opts.nonInteractive || !deps.TTY.StdinTTY {
			message := "RunnerKit can't continue because a BYO SSH host is required."
			_ = renderer.Error("input_required", message, []string{"Pass --host user@host for BYO setup."})
			return remote.Target{}, NewExitError(ExitInputRequired, errors.New(message+" Pass --host user@host for BYO setup."))
		}
		inputPrompter, ok := deps.Prompts.(interface {
			Input(context.Context, ui.Prompt) (string, error)
		})
		if !ok {
			message := "RunnerKit can't continue because SSH host input requires an interactive prompt."
			_ = renderer.Error("input_required", message, []string{"Pass --host user@host for BYO setup."})
			return remote.Target{}, NewExitError(ExitInputRequired, errors.New(message))
		}
		var err error
		raw, err = inputPrompter.Input(ctx, ui.Prompt{Message: "SSH target (user@host):"})
		if err != nil {
			return remote.Target{}, err
		}
	}
	target, err := remote.ParseTarget(raw, opts.sshPort)
	if err != nil {
		_ = renderer.Error("invalid_ssh_target", "RunnerKit can't parse the BYO SSH target.", []string{err.Error(), "Pass --host user@host or --host user@host:port."})
		return remote.Target{}, NewExitError(ExitInvalidInput, err)
	}
	target.KeyPath = opts.sshKey
	return target, nil
}

func verifyTargetHostKey(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, target remote.Target, existing rkstate.RepositoryState, exists bool) (remote.HostKey, *time.Time, error) {
	probe, err := deps.RemoteExecutor.Probe(ctx, target)
	if err != nil {
		_ = renderer.Error("ssh_probe_failed", "RunnerKit could not inspect the SSH host key.", []string{"Verify SSH access to " + target.Display() + " and re-run runnerkit up."})
		return remote.HostKey{}, nil, NewExitError(ExitSafetyGate, err)
	}
	observed := remote.NormalizeHostKey(probe.HostKey)
	if observed.Fingerprint == "" {
		observed.Fingerprint = remote.FingerprintSHA256([]byte(target.Display()))
	}
	storedFingerprint := ""
	if exists {
		storedFingerprint = existing.Machine.HostKeyFingerprint
	}
	decision, decisionErr := remote.DecideHostKey(storedFingerprint, observed)
	if decision == remote.HostKeyMismatch {
		_ = renderer.Error(remote.HostKeyMismatchCode, "RunnerKit stopped because the SSH host key fingerprint changed.", []string{"Expected " + storedFingerprint + " but observed " + observed.Fingerprint + ". Verify the host before continuing."})
		return remote.HostKey{}, nil, NewExitError(ExitSafetyGate, decisionErr)
	}
	if decision == remote.HostKeyAccepted {
		return observed, existing.Machine.HostKeyAcceptedAt, nil
	}
	if opts.yes {
		now := deps.Clock()
		return observed, &now, nil
	}
	if jsonOutput || opts.nonInteractive || !deps.TTY.StdinTTY || deps.Prompts == nil {
		message := "RunnerKit can't continue until you accept the SSH host key."
		_ = renderer.Error("input_required", message, []string{"Re-run interactively or pass --yes after verifying " + observed.Fingerprint + " for " + target.Display() + "."})
		return remote.HostKey{}, nil, NewExitError(ExitInputRequired, errors.New(message))
	}
	confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Accept SSH host key " + observed.Fingerprint + " for " + target.Display() + "?", Default: false})
	if err != nil {
		return remote.HostKey{}, nil, err
	}
	if !confirmed {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return remote.HostKey{}, nil, NewExitError(ExitCanceled, errors.New(message))
	}
	now := deps.Clock()
	return observed, &now, nil
}

func renderPreflightFailure(renderer *ui.Renderer, jsonOutput bool, report preflight.Report) error {
	if jsonOutput {
		_ = renderer.JSON(map[string]any{"ok": false, "error": map[string]any{"code": "ssh_preflight_failed", "message": "SSH preflight failed before runner installation."}, "ssh-preflight": report.Results})
	} else {
		_ = renderPreflightHuman(renderer, report)
		_ = renderer.Error("ssh_preflight_failed", "SSH preflight failed before runner installation.", []string{"Fix failed checks or pass --allow-unknown-linux only for unverified Linux distributions you trust."})
	}
	return NewExitError(ExitSafetyGate, errors.New("ssh_preflight_failed"))
}

func renderDryRun(renderer *ui.Renderer, jsonOutput bool, repo gh.Repo, source gh.AuthSource, warnings []string, statePath string, target remote.Target, report preflight.Report, labelSet labels.LabelSet, plan workflow.Plan, modeDecision runmode.Decision, ttl time.Duration) error {
	if jsonOutput {
		payload := map[string]any{
			"ok":               true,
			"command":          "up",
			"repo":             repo.FullName,
			"auth_source":      defaultString(source.Kind, "gh"),
			"runner_installed": false,
			"state_saved":      false,
			"state_path":       statePath,
			"runner_name":      labelSet.RunnerName,
			"labels":           labelSet.Labels,
			"machine_target":   target.Display(),
			"workflow_snippet": labelSet.RunsOnYAML,
			"warnings":         mergeWarnings(warnings, modeDecision.Warnings),
			"ssh-preflight":    report.Results,
			"bootstrap-plan":   plan,
		}
		for k, v := range modeSelectionPayload(modeDecision, ttl) {
			if k == "warnings" {
				continue // warnings are already merged with safety warnings above.
			}
			payload[k] = v
		}
		return renderer.JSON(payload)
	}
	if err := renderPreflightHuman(renderer, report); err != nil {
		return err
	}
	return renderer.Step(2, 2, "bootstrap-plan", ui.Bullet("Runner name: "+labelSet.RunnerName), ui.Bullet("Target: "+target.Display()), ui.Bullet("Labels: ["+strings.Join(labelSet.Labels, ", ")+"]"), ui.Bullet(labelSet.RunsOnYAML), ui.WarningLine(labelSet.Warning), ui.Bullet("Dry run: no state file was written and no runner was installed."))
}

// mergeWarnings concatenates safety and mode-selection warnings while
// preserving order and avoiding nil slices in JSON output.
func mergeWarnings(safety []string, mode []string) []string {
	out := make([]string, 0, len(safety)+len(mode))
	seen := map[string]bool{}
	for _, warning := range safety {
		if warning == "" || seen[warning] {
			continue
		}
		seen[warning] = true
		out = append(out, warning)
	}
	for _, warning := range mode {
		if warning == "" || seen[warning] {
			continue
		}
		seen[warning] = true
		out = append(out, warning)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func renderPreflightHuman(renderer *ui.Renderer, report preflight.Report) error {
	lines := []ui.Line{ui.Bullet("Target: " + report.Target.Display())}
	for _, result := range report.Results {
		line := ui.Bullet(result.ID + ": " + string(result.Severity))
		if result.Severity == preflight.SeverityFailure {
			line = ui.ErrorLine(result.ID + ": " + result.Message)
		} else if result.Severity == preflight.SeverityWarning {
			line = ui.WarningLine(result.ID + ": " + result.Message)
		}
		lines = append(lines, line)
	}
	return renderer.Step(1, 2, "ssh-preflight", lines...)
}

func verifyBYOFoundationForRegister(ctx context.Context, deps Dependencies, renderer *ui.Renderer, jsonOutput bool, target remote.Target) error {
	cmd := remote.Command{
		ID:     "verify_runnerkit_foundation",
		Script: bootstrap.FoundationUserProbeScript(),
		Sudo:   false,
	}
	res, err := deps.RemoteExecutor.Run(ctx, target, cmd)
	if err != nil || res.ExitCode != 0 {
		return RenderLifecycleFoundationMissing(renderer, jsonOutput, deps.Version)
	}
	return nil
}

func buildBootstrapOptions(repo gh.Repo, labelSet labels.LabelSet, pkg bootstrap.RunnerPackage, report preflight.Report) bootstrap.Options {
	installPath := filepath.Join("/opt/actions-runner", labelSet.RunnerName)
	workDir := filepath.Join("/var/lib/runnerkit/work", labelSet.RunnerName)
	return bootstrap.Options{
		RunnerName:   labelSet.RunnerName,
		RepoURL:      "https://github.com/" + repo.FullName,
		Labels:       labelSet.Labels,
		InstallPath:  installPath,
		WorkDir:      workDir,
		ServiceUser:  bootstrap.DefaultServiceUser,
		Package:      pkg,
		MissingTools: report.FixableTools,
	}
}

func confirmBootstrapPlan(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, target remote.Target) error {
	if opts.yes {
		return nil
	}
	if jsonOutput || opts.nonInteractive || !deps.TTY.StdinTTY || deps.Prompts == nil {
		message := "RunnerKit can't continue because applying the BYO runner install plan requires confirmation."
		_ = renderer.Error("input_required", message, []string{"Pass --yes to apply the install plan non-interactively, or use --dry-run to preview without changing the host."})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Apply BYO runner install plan to " + target.Display() + "?", Default: false})
	if err != nil {
		return err
	}
	if !confirmed {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return NewExitError(ExitCanceled, errors.New(message))
	}
	return nil
}

// isRunnerKitManagedRunner reports whether the given GitHub runner
// was previously registered by RunnerKit. Detected via the canonical
// `runnerkit` label that labels.Build always emits. Used by the
// runner-name pre-bootstrap collision check so re-runs of
// `runnerkit up` don't refuse to re-register their own runners
// (Bug 17 / Task T).
func isRunnerKitManagedRunner(r gh.Runner) bool {
	for _, label := range r.Labels {
		if strings.EqualFold(label, "runnerkit") {
			return true
		}
	}
	return false
}

func runnerNameConflict(renderer *ui.Renderer, runnerName string, existing gh.Runner) error {
	message := "RunnerKit can't continue because a GitHub runner named " + runnerName + " already exists."
	_ = renderer.Error("runner_name_conflict", message, []string{"Remove or rename the existing GitHub runner " + runnerName + ", then re-run runnerkit up."})
	return NewExitError(ExitSafetyGate, fmt.Errorf("runner_name_conflict: %s is %s", existing.Name, existing.Status))
}

func waitForRunnerOnline(ctx context.Context, deps Dependencies, repo gh.Repo, name string, expectedLabels []string) (gh.Runner, bool, error) {
	deadline := time.Now().Add(deps.PollTimeout)
	for {
		runners, err := deps.GitHub.ListRunners(ctx, repo)
		if err != nil {
			_ = newRenderer(deps, false, true).Error("github_runner_list_failed", "RunnerKit can't list repository self-hosted runners.", []string{"Verify GitHub credentials can list repository runners for " + repo.FullName + "."})
			return gh.Runner{}, false, NewExitError(ExitGitHubAuth, err)
		}
		if runner, ok := runnerOnlineWithLabels(runners, name, expectedLabels); ok {
			return runner, true, nil
		}
		if !time.Now().Before(deadline) {
			return gh.Runner{}, false, nil
		}
		if err := deps.Sleep(ctx, deps.PollInterval); err != nil {
			return gh.Runner{}, false, err
		}
	}
}

func runnerOnlineWithLabels(runners []gh.Runner, name string, expectedLabels []string) (gh.Runner, bool) {
	for _, runner := range runners {
		if runner.Name != name || runner.Status != "online" {
			continue
		}
		// Bug 16 (Plan 06-07 attempt-13, 2026-05-06): case-insensitive
		// match. GitHub auto-adds OS + arch labels in canonical CamelCase
		// (Linux, macOS, Windows, X64, ARM64) while RunnerKit's
		// labels.Build slug-lowercases its values. Both are correct in
		// their own world; the matching layer must normalize.
		actual := map[string]bool{}
		for _, label := range runner.Labels {
			actual[strings.ToLower(label)] = true
		}
		allPresent := true
		for _, label := range expectedLabels {
			if !actual[strings.ToLower(label)] {
				allPresent = false
				break
			}
		}
		if allPresent {
			return runner, true
		}
	}
	return gh.Runner{}, false
}

func saveRepositoryState(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, store rkstate.Store, fullName string, repoState rkstate.RepositoryState) error {
	replace := opts.replace
	if _, exists, err := store.GetRepository(fullName); err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + " and re-run runnerkit up."})
		return NewExitError(ExitStateIO, err)
	} else if exists && !replace {
		confirmedReplace, err := confirmStateReplace(ctx, deps, renderer, opts, fullName, jsonOutput)
		if err != nil {
			return err
		}
		replace = confirmedReplace
	}
	if err := store.SaveRepository(repoState, replace); err != nil {
		if errors.Is(err, rkstate.ErrRepositoryExists) {
			return replacementRequired(renderer, fullName)
		}
		_ = renderer.Error("state_io_failed", "RunnerKit can't save runner state.", []string{"Check permissions for " + store.Path() + " and re-run runnerkit up."})
		return NewExitError(ExitStateIO, err)
	}
	return nil
}

func buildBYORepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, labelSet labels.LabelSet, target remote.Target, hostKey remote.HostKey, acceptedAt *time.Time, opts bootstrap.Options, onlineRunner gh.Runner) rkstate.RepositoryState {
	now := deps.Clock()
	if now.IsZero() {
		now = time.Now()
	}
	if acceptedAt == nil {
		acceptedAt = &now
	}
	safety := rkstate.SafetyMetadata{Code: decision.Code, Allowed: decision.Allowed, Warnings: decision.Warnings}
	if decision.Code == gh.SafetyCodePublicRisk && decision.Allowed && len(decision.Warnings) > 0 {
		safety.AcceptedOverride = gh.AllowPublicRepoRiskFlag
		safety.AcceptedAt = &now
	}
	return rkstate.RepositoryState{
		Repo: repo,
		Auth: rkstate.AuthReference{Source: defaultString(source.Kind, "gh"), Reference: defaultString(source.Reference, source.Kind)},
		Runner: rkstate.RunnerIdentity{
			Name:            labelSet.RunnerName,
			Labels:          labelSet.Labels,
			WorkflowSnippet: labelSet.RunsOnYAML,
			Mode:            labels.DefaultMode,
			OS:              labels.DefaultOS,
			Arch:            opts.Package.Arch,
		},
		Machine: rkstate.MachineRef{
			Kind:               "byo-ssh",
			HostRef:            target.Display(),
			User:               target.User,
			Port:               target.Port,
			KeyPathRef:         target.KeyPath,
			HostKeyAlgorithm:   hostKey.Algorithm,
			HostKeyFingerprint: hostKey.Fingerprint,
			HostKeyAcceptedAt:  acceptedAt,
			InstallPath:        opts.InstallPath,
			WorkDir:            opts.WorkDir,
			ServiceName:        runnerServiceName(labelSet.RunnerName),
		},
		Provider:         rkstate.ProviderRef{Kind: "byo", IDs: map[string]string{}},
		Cleanup:          rkstate.CleanupMetadata{GitHubRunnerID: onlineRunner.ID, ManagedPaths: []string{opts.InstallPath, "/var/lib/runnerkit"}, ProviderResourceIDs: []string{}},
		Safety:           safety,
		RunnerKitVersion: deps.Version,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func renderCompletionHuman(renderer *ui.Renderer, warnings []string, statePath string, target remote.Target, labelSet labels.LabelSet, opts bootstrap.Options, onlineRunner gh.Runner) error {
	lines := []ui.Line{
		ui.Success("Runner name: " + labelSet.RunnerName),
		ui.Bullet("Machine target: " + target.Display()),
		ui.Bullet("Service name: " + runnerServiceName(labelSet.RunnerName)),
		ui.Bullet("Labels: [" + strings.Join(labelSet.Labels, ", ") + "]"),
		ui.Bullet(labelSet.RunsOnYAML),
		ui.WarningLine("Do not use runs-on: self-hosted alone for RunnerKit-managed runners."),
		ui.Bullet("GitHub runner ID: " + fmt.Sprintf("%d", onlineRunner.ID)),
		ui.Bullet("State path: " + statePath),
		ui.Next("Add the runs-on snippet above to the workflow job you want to run on this runner."),
		ui.Bullet("Later cleanup will be handled by a future runnerkit down flow; do not delete the BYO host manually if you want RunnerKit state to stay accurate."),
		ui.Bullet("Install path: " + opts.InstallPath),
	}
	for _, warning := range warnings {
		lines = append(lines, ui.WarningLine(warning))
	}
	return renderer.Step(1, 1, "BYO runner ready", lines...)
}

func upCompletionJSON(repo string, warnings []string, statePath string, target remote.Target, labelSet labels.LabelSet, opts bootstrap.Options, onlineRunner gh.Runner) map[string]any {
	if warnings == nil {
		warnings = []string{}
	}
	p := map[string]any{
		"ok":               true,
		"command":          "up",
		"repo":             repo,
		"runner_installed": true,
		"runner_name":      labelSet.RunnerName,
		"labels":           labelSet.Labels,
		"machine_target":   target.Display(),
		"service_name":     runnerServiceName(labelSet.RunnerName),
		"workflow_snippet": labelSet.RunsOnYAML,
		"github_runner_id": onlineRunner.ID,
		"state_path":       statePath,
		"warnings":         warnings,
		"next_steps": []string{
			"Add the runs-on snippet above to the workflow job you want to run on this runner.",
			"Do not use runs-on: self-hosted alone for RunnerKit-managed runners.",
		},
		"install_path": opts.InstallPath,
	}
	nextaction.MergePayload(p, "complete", []nextaction.Action{
		{ID: "add_workflow_snippet", Severity: nextaction.SeverityInfo, Title: "Add the workflow snippet to your job", Kind: "run_local"},
	})
	return p
}

func renderCloudCompletionHuman(renderer *ui.Renderer, warnings []string, statePath string, plan provider.ProvisionPlan, result provider.ProvisionResult, labelSet labels.LabelSet, opts bootstrap.Options, onlineRunner gh.Runner) error {
	cloud := result.Machine.Provider.Cloud
	publicHost := defaultString(result.Machine.PublicIPv4, result.Machine.Target.Host)
	destroyCommand := defaultString(plan.FutureDestroyCommand, "runnerkit destroy --repo "+strings.TrimPrefix(labelSet.RunnerName, "runnerkit-"))
	lines := []ui.Line{
		ui.Success("Runner name: " + labelSet.RunnerName),
		ui.Bullet("Machine target: " + result.Machine.Target.Display()),
		ui.Bullet("Service name: " + runnerServiceName(labelSet.RunnerName)),
		ui.Bullet("Labels: [" + strings.Join(labelSet.Labels, ", ") + "]"),
		ui.Bullet(labelSet.RunsOnYAML),
		ui.WarningLine("Do not use runs-on: self-hosted alone for RunnerKit-managed runners."),
		ui.Bullet("GitHub runner ID: " + fmt.Sprintf("%d", onlineRunner.ID)),
		ui.Bullet("Provider: Hetzner " + defaultString(cloud.Region, plan.Region) + " " + defaultString(cloud.ServerType, plan.ServerType) + " " + defaultString(cloud.Image, plan.Image)),
		ui.Bullet("Public host: " + publicHost),
		ui.Bullet("Billable resources: " + strings.Join(cloudProviderResourceIDList(mergeCloudResourceIDs(result)), ", ")),
		ui.Bullet("Cleanup: " + destroyCommand),
		ui.Bullet("State path: " + statePath),
		ui.Next("Add the runs-on snippet above to the workflow job you want to run on this runner."),
		ui.Bullet("Install path: " + opts.InstallPath),
	}
	for _, warning := range warnings {
		lines = append(lines, ui.WarningLine(warning))
	}
	return renderer.Step(1, 1, "Cloud runner ready", lines...)
}

func cloudCompletionJSON(repo string, statePath string, plan provider.ProvisionPlan, result provider.ProvisionResult, labelSet labels.LabelSet, opts bootstrap.Options, onlineRunner gh.Runner) map[string]any {
	resourceIDs := cloudProviderResourceIDList(mergeCloudResourceIDs(result))
	cloud := result.Machine.Provider.Cloud
	return map[string]any{
		"ok":                 true,
		"command":            "up",
		"repo":               repo,
		"runner_installed":   true,
		"state_saved":        true,
		"runner_name":        labelSet.RunnerName,
		"labels":             labelSet.Labels,
		"machine_target":     result.Machine.Target.Display(),
		"service_name":       runnerServiceName(labelSet.RunnerName),
		"workflow_snippet":   labelSet.RunsOnYAML,
		"github_runner_id":   onlineRunner.ID,
		"state_path":         statePath,
		"provider":           plan.Provider,
		"cloud":              cloud,
		"billable_resources": resourceIDs,
		"destroy_command":    "runnerkit destroy --repo " + repo,
		"install_path":       opts.InstallPath,
	}
}

// buildEphemeralBYORepositoryState builds the state row that BYO
// ephemeral up persists. It is a thin wrapper around the persistent
// BYO state builder that overrides the Mode/SafetyProfile/ServiceName
// fields and populates EphemeralMetadata.
func buildEphemeralBYORepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, modeDecision runmode.Decision, labelSet labels.LabelSet, target remote.Target, hostKey remote.HostKey, acceptedAt *time.Time, opts bootstrap.Options, onlineRunner gh.Runner, ttl time.Duration) rkstate.RepositoryState {
	state := buildBYORepositoryState(deps, repo, source, decision, labelSet, target, hostKey, acceptedAt, opts, onlineRunner)
	state.Runner.Mode = runmode.ModeEphemeral
	state.Machine.ServiceName = ephemeralServiceName(labelSet.RunnerName)
	state.Safety.SafetyProfile = modeDecision.SafetyProfile
	state.Ephemeral = ephemeralMetadataFor(deps, repo, labelSet.RunnerName, ttl, false)
	return state
}

// buildEphemeralCloudRepositoryState builds the state row that cloud
// ephemeral up persists. It reuses the persistent cloud state builder
// then overrides Mode/SafetyProfile/ServiceName and populates
// EphemeralMetadata with the cloud cleanup command.
func buildEphemeralCloudRepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, modeDecision runmode.Decision, labelSet labels.LabelSet, target remote.Target, hostKey remote.HostKey, input provider.ProvisionInput, plan provider.ProvisionPlan, result provider.ProvisionResult, opts bootstrap.Options, onlineRunner gh.Runner, keyPathRef string, ttl time.Duration) rkstate.RepositoryState {
	state := buildCloudRepositoryState(deps, repo, source, decision, labelSet, target, hostKey, input, plan, result, opts, onlineRunner, keyPathRef)
	state.Runner.Mode = runmode.ModeEphemeral
	state.Machine.ServiceName = ephemeralServiceName(labelSet.RunnerName)
	state.Safety.SafetyProfile = modeDecision.SafetyProfile
	state.Ephemeral = ephemeralMetadataFor(deps, repo, labelSet.RunnerName, ttl, true)
	return state
}

func ephemeralMetadataFor(deps Dependencies, repo gh.Repo, runnerName string, ttl time.Duration, cloud bool) rkstate.EphemeralMetadata {
	now := deps.Clock()
	if now.IsZero() {
		now = time.Now()
	}
	if ttl == 0 {
		ttl = runmode.DefaultEphemeralTTL
	}
	expires := now.Add(ttl)
	return rkstate.EphemeralMetadata{
		Enabled:         true,
		TTL:             "24h",
		ExpiresAt:       &expires,
		LogArchivePath:  ephemeralLogArchivePath(runnerName),
		FinalizerStatus: "pending",
		CleanupCommand:  ephemeralCleanupCommand(repo.FullName, cloud),
	}
}

// renderEphemeralCompletionHuman renders the human-readable completion
// step for an ephemeral runner. It always begins with "Ephemeral runner
// ready" and includes the canonical UI-SPEC sentences asserted by
// integration tests.
func renderEphemeralCompletionHuman(renderer *ui.Renderer, repoFullName string, modeDecision runmode.Decision, warnings []string, statePath string, target remote.Target, labelSet labels.LabelSet, opts bootstrap.Options, onlineRunner gh.Runner, ttl time.Duration, cloud bool) error {
	cleanup := ephemeralCleanupCommand(repoFullName, cloud)
	logArchive := defaultString(opts.LogArchivePath, ephemeralLogArchivePath(labelSet.RunnerName))
	lines := []ui.Line{
		ui.Success("Ephemeral runner ready: " + labelSet.RunnerName),
		ui.Bullet("Mode: ephemeral"),
		ui.Bullet("Safety profile: " + modeDecision.SafetyProfile),
		ui.Bullet("Machine target: " + target.Display()),
		ui.Bullet("Service name: " + ephemeralServiceName(labelSet.RunnerName)),
		ui.Bullet("Labels: [" + strings.Join(labelSet.Labels, ", ") + "]"),
		ui.Bullet(labelSet.RunsOnYAML),
		ui.WarningLine("Do not use runs-on: self-hosted alone for RunnerKit-managed runners."),
		ui.Bullet("GitHub runner ID: " + fmt.Sprintf("%d", onlineRunner.ID)),
		ui.Bullet("Log archive: " + logArchive),
		ui.Bullet("State path: " + statePath),
		ui.Bullet("GitHub will assign at most one job to this runner, then automatically deregister it."),
		ui.Bullet("TTL safeguard: RunnerKit finalizes the ephemeral runner after 24h if no job completes."),
		ui.Bullet("RunnerKit preserves best-effort runner _diag and systemd journal logs before cleanup."),
		ui.Next("Cleanup after the job: " + cleanup),
		ui.Bullet("Install path: " + opts.InstallPath),
		ui.WarningLine("Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online."),
	}
	for _, warning := range warnings {
		lines = append(lines, ui.WarningLine(warning))
	}
	_ = ttl // ttl is implied in the canonical 24h sentence above.
	return renderer.Step(1, 1, "Ephemeral runner ready", lines...)
}

func ephemeralCompletionJSON(repoFullName string, modeDecision runmode.Decision, warnings []string, statePath string, target remote.Target, labelSet labels.LabelSet, opts bootstrap.Options, onlineRunner gh.Runner, ttl time.Duration, cloud bool) map[string]any {
	if warnings == nil {
		warnings = []string{}
	}
	cleanup := ephemeralCleanupCommand(repoFullName, cloud)
	logArchive := defaultString(opts.LogArchivePath, ephemeralLogArchivePath(labelSet.RunnerName))
	if ttl == 0 {
		ttl = runmode.DefaultEphemeralTTL
	}
	return map[string]any{
		"ok":               true,
		"command":          "up",
		"repo":             repoFullName,
		"runner_installed": true,
		"runner_name":      labelSet.RunnerName,
		"labels":           labelSet.Labels,
		"machine_target":   target.Display(),
		"service_name":     ephemeralServiceName(labelSet.RunnerName),
		"workflow_snippet": labelSet.RunsOnYAML,
		"github_runner_id": onlineRunner.ID,
		"state_path":       statePath,
		"warnings":         warnings,
		"install_path":     opts.InstallPath,
		"mode":             runmode.ModeEphemeral,
		"safety_profile":   modeDecision.SafetyProfile,
		"ephemeral":        true,
		"ttl":              ttl.String(),
		"log_archive":      logArchive,
		"cleanup_command":  cleanup,
	}
}

// enforceModeSafetyDecision applies the Phase 5 safety policy that
// depends on both the GitHub repo trust state and the chosen runner mode.
//
// The function MUST run before VerifyAuth, VerifyRunnerManagementRead,
// CreateRegistrationToken, the remote probe, provider Validate/Plan/
// Provision, or any state save so risky combinations cannot leak
// side effects. The four cases are:
//
//   - ProfilePersistentRisky without --allow-public-repo-risk: block with
//     the public_repo_risk error code, render the exact UI-SPEC body,
//     ephemeral cloud recommendation, and dangerous-override copy.
//   - ProfilePersistentRisky with --allow-public-repo-risk: keep the
//     typed acknowledgement flow but use the Phase 5 prompt copy and
//     surface DangerousPersistentOverrideCopy as part of the warnings.
//   - ProfileEphemeralCloud: never block on public/fork — ephemeral cloud
//     is the recommended path. Just append the safer-recommendation
//     warnings to the decision so renderModeTradeoffs surfaces them.
//   - ProfileEphemeralBYO on public/fork: require either typed input
//     `use ephemeral byo for owner/name` or non-interactive
//     `--allow-ephemeral-byo-risk --yes`; otherwise block with the
//     BYO clean-VM caveat.
func enforceModeSafetyDecision(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, decision gh.SafetyDecision, modeDecision *runmode.Decision, opts *upOptions, jsonOutput bool) error {
	switch modeDecision.SafetyProfile {
	case runmode.ProfileEphemeralCloud:
		// Public/fork ephemeral cloud is the recommended safer path.
		// Append the recommendation strings so renderModeTradeoffs and
		// downstream warnings make the safer choice visible. The exact
		// `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner`
		// command stays in the warning list so docs greps and human
		// output assertions both succeed.
		if !repo.Private || repo.Fork {
			modeDecision.Warnings = append(modeDecision.Warnings,
				runmode.WarningPublicForkPersistent,
				"Use ephemeral cloud runner: runnerkit up --repo "+repo.FullName+" --mode ephemeral --cloud hetzner",
			)
		}
		return nil
	case runmode.ProfileEphemeralBYO:
		if !repo.Private || repo.Fork {
			return enforceEphemeralBYOAcknowledgement(ctx, deps, renderer, repo, opts, jsonOutput)
		}
		return nil
	case runmode.ProfilePersistentRisky:
		if !opts.allowPublicRepoRisk {
			return blockPersistentRiskWithUISpecCopy(renderer, decision, jsonOutput)
		}
		// --allow-public-repo-risk allowed: require typed acknowledgement
		// when the user is interactive without --yes, using the Phase 5
		// prompt text and surfacing DangerousPersistentOverrideCopy.
		if deps.TTY.StdinTTY && !opts.yes {
			return acknowledgePersistentPublicRisk(ctx, deps, renderer, repo)
		}
		return nil
	}

	// ProfilePersistentTrusted: nothing extra to enforce; allow flow.
	if !decision.Allowed {
		// Defensive: handle fork_risk where mode profile is still trusted
		// (e.g., private+fork). Reuse the existing fork_risk warning.
		if jsonOutput {
			_ = renderer.Error(decision.Code, "WARNING: Fork repository risk", decision.Warnings)
		} else {
			_ = renderer.Warning("WARNING: Fork repository risk", decision.Warnings, "Use a trusted private repository before persistent setup.")
		}
		return NewExitError(ExitSafetyGate, errors.New(decision.Code))
	}
	return nil
}

func blockPersistentRiskWithUISpecCopy(renderer *ui.Renderer, decision gh.SafetyDecision, jsonOutput bool) error {
	// RKD-AUTH-001: every emit site prepends the stable code + See: URL
	// (D-15). Existing tests assert on `public_repo_risk` and the
	// `WARNING: Public repository risk` title — those still hold because
	// we ADD a code line, we don't replace existing copy.
	rkd := errcodes.FormatLine(errcodes.AuthPublicRepoBlocked)
	warnings := []string{rkd, gh.PublicRepoRiskBody, gh.PublicRepoRiskNextAction, gh.DangerousPersistentOverrideCopy}
	// Preserve any decision-level warnings already populated so callers
	// that surface decision.Warnings (e.g., state) keep them, but always
	// surface the canonical UI-SPEC copy first.
	for _, w := range decision.Warnings {
		if w != "" {
			warnings = append(warnings, w)
		}
	}
	if jsonOutput {
		_ = renderer.Error(gh.SafetyCodePublicRisk, gh.PublicRepoRiskTitle, warnings)
	} else {
		_ = renderer.Warning(gh.PublicRepoRiskTitle, []string{rkd, gh.PublicRepoRiskBody, gh.DangerousPersistentOverrideCopy}, gh.PublicRepoRiskNextAction)
	}
	return NewExitError(ExitSafetyGate, errors.New(gh.SafetyCodePublicRisk))
}

func acknowledgePersistentPublicRisk(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo) error {
	inputPrompter, ok := deps.Prompts.(interface {
		Input(context.Context, ui.Prompt) (string, error)
	})
	if !ok {
		message := "RunnerKit can't continue because public repository risk acknowledgement requires typed confirmation."
		_ = renderer.Error("input_required", message, []string{"Type allow persistent public repo risk for " + repo.FullName + " in an interactive terminal or pass --yes only after reviewing the risk.", gh.DangerousPersistentOverrideCopy})
		return NewExitError(ExitInputRequired, errors.New(message))
	}
	want := "allow persistent public repo risk for " + repo.FullName
	got, err := inputPrompter.Input(ctx, ui.Prompt{Message: want, Help: gh.DangerousPersistentOverrideCopy})
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

func enforceEphemeralBYOAcknowledgement(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, opts *upOptions, jsonOutput bool) error {
	// Non-interactive happy path: --allow-ephemeral-byo-risk --yes.
	if opts.allowEphemeralBYORisk && opts.yes {
		return nil
	}
	// Interactive: require typed input "use ephemeral byo for owner/name".
	if deps.TTY.StdinTTY && deps.Prompts != nil && !jsonOutput {
		inputPrompter, ok := deps.Prompts.(interface {
			Input(context.Context, ui.Prompt) (string, error)
		})
		if ok {
			want := "use ephemeral byo for " + repo.FullName
			got, err := inputPrompter.Input(ctx, ui.Prompt{Message: want, Help: runmode.WarningEphemeralBYONotCleanVM})
			if err != nil {
				return err
			}
			if got == want {
				return nil
			}
			message := "Canceled; no changes made."
			_ = renderer.Error("canceled", message, nil)
			return NewExitError(ExitCanceled, errors.New(message))
		}
	}
	// Otherwise: block with the BYO clean-VM caveat and remediation.
	// Append the stable RKD-AUTH-003 code + See: URL after the existing
	// remediation copy (D-15). Append (not prepend) so existing tests
	// that index remediation[0] keep working.
	message := runmode.WarningEphemeralBYONotCleanVM
	remediation := []string{
		"Use runnerkit up --repo " + repo.FullName + " --mode ephemeral --cloud hetzner for stronger isolation, or pass --allow-ephemeral-byo-risk --yes only after reviewing the risk.",
		errcodes.FormatLine(errcodes.AuthEphemeralBYOPublicForkAck),
	}
	_ = renderer.Error("ephemeral_byo_risk", message, remediation)
	return NewExitError(ExitSafetyGate, errors.New("ephemeral_byo_risk"))
}

func resolveUpRepo(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions) (gh.Resolution, error) {
	if opts.repo == "" && (!deps.TTY.StdinTTY || opts.nonInteractive) {
		message := "RunnerKit can't continue because repository scope is required before auth or state actions apply."
		remediation := []string{gh.TargetRemediation(nil)[0]}
		_ = renderer.Error("input_required", message, remediation)
		return gh.Resolution{}, NewExitError(ExitInputRequired, errors.New(message+" Pass --repo owner/name."))
	}
	resolution, err := gh.ResolveTarget(ctx, gh.ResolveOptions{Repo: opts.repo, CommandRunner: deps.CommandRunner})
	if err != nil {
		message := fmt.Sprintf("RunnerKit can't continue because %s.", err.Error())
		remediation := gh.TargetRemediation(err)
		code := ExitInvalidInput
		if opts.repo == "" {
			code = ExitInputRequired
		}
		_ = renderer.Error("invalid_repo", message, remediation)
		return gh.Resolution{}, NewExitError(code, err)
	}
	if resolution.NeedsConfirmation {
		if err := renderer.Step(3, 6, "Choose repository", ui.PromptLine("Choose repository: "+resolution.Repo.FullName)); err != nil {
			return gh.Resolution{}, err
		}
		if deps.Prompts == nil {
			message := "RunnerKit can't continue because repository confirmation requires an interactive prompt."
			_ = renderer.Error("input_required", message, []string{"Pass --repo " + resolution.Repo.FullName + " --yes to confirm the target repository."})
			return gh.Resolution{}, NewExitError(ExitInputRequired, errors.New(message))
		}
		confirmed, err := deps.Prompts.Confirm(ctx, ui.Prompt{Message: "Choose repository: " + resolution.Repo.FullName, Default: true})
		if err != nil {
			return gh.Resolution{}, err
		}
		if !confirmed {
			message := "Canceled; no changes made."
			_ = renderer.Error("canceled", message, nil)
			return gh.Resolution{}, NewExitError(ExitCanceled, errors.New(message))
		}
	}
	return resolution, nil
}

func confirmStateReplace(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, fullName string, jsonOutput bool) (bool, error) {
	if jsonOutput || opts.yes || opts.nonInteractive || !deps.TTY.StdinTTY {
		return false, replacementRequired(renderer, fullName)
	}
	inputPrompter, ok := deps.Prompts.(interface {
		Input(context.Context, ui.Prompt) (string, error)
	})
	if !ok {
		return false, replacementRequired(renderer, fullName)
	}
	want := "replace " + fullName
	got, err := inputPrompter.Input(ctx, ui.Prompt{Message: "Type " + want + " to overwrite the existing RunnerKit state for this repository."})
	if err != nil {
		return false, err
	}
	if got != want {
		message := "Canceled; no changes made."
		_ = renderer.Error("canceled", message, nil)
		return false, NewExitError(ExitCanceled, errors.New(message))
	}
	return true, nil
}

func replacementRequired(renderer *ui.Renderer, fullName string) error {
	message := "RunnerKit can't continue because saved foundation state already exists for " + fullName + "."
	_ = renderer.Error("input_required", message, []string{"Type replace " + fullName + " in interactive mode, or re-run with --yes --replace after reviewing the existing state."})
	return NewExitError(ExitInputRequired, errors.New(message))
}

func repoSafetyStatus(repo gh.Repo) string {
	if !repo.Private {
		return gh.SafetyCodePublicRisk
	}
	if repo.Fork {
		return gh.SafetyCodeForkRisk
	}
	return gh.SafetyCodeOK
}

func boolWord(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func runnerServiceName(runnerName string) string {
	return "actions.runner." + runnerName + ".service"
}

// lastCommandFailureContext extracts the failing command's ID and
// stderr from a bootstrap.Result + the err returned by Apply /
// ApplyEphemeral so callers can surface remote diagnostics in
// bootstrap_failed messages. The CommandID comes from
// remote.RemoteError when present (the typical exit-code path);
// stderr comes from the trailing entry of result.Commands. Returns
// empty strings if no useful context is available.
func lastCommandFailureContext(result bootstrap.Result, err error) (string, string) {
	var stderr string
	if len(result.Commands) > 0 {
		stderr = strings.TrimSpace(result.Commands[len(result.Commands)-1].Stderr)
	}
	commandID := ""
	var remoteErr remote.RemoteError
	if errors.As(err, &remoteErr) {
		commandID = remoteErr.CommandID
	}
	if commandID == "" {
		commandID = "unknown"
	}
	return commandID, stderr
}
