package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/salar/runnerkit/internal/bootstrap"
	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/labels"
	"github.com/salar/runnerkit/internal/preflight"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/redact"
	"github.com/salar/runnerkit/internal/remote"
	rkstate "github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/ui"
	"github.com/salar/runnerkit/internal/workflow"
	"github.com/spf13/cobra"
)

const (
	defaultRunnerPollInterval = 2 * time.Second
	defaultRunnerPollTimeout  = 60 * time.Second
)

type upOptions struct {
	repo                string
	yes                 bool
	nonInteractive      bool
	dryRun              bool
	allowPublicRepoRisk bool
	replace             bool
	host                string
	sshPort             int
	sshKey              string
	allowUnknownLinux   bool
	cloud               string
	cloudRegion         string
	cloudProfile        string
	sshAllowedCIDR      string
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

	return cmd
}

func runUp(deps Dependencies, jsonOutput bool, noColor bool, opts *upOptions) error {
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
	decision := gh.EvaluateSafety(repo, gh.SafetyOptions{AllowPublicRepoRisk: opts.allowPublicRepoRisk})
	if err := enforceSafetyDecision(ctx, deps, renderer, repo, decision, opts, jsonOutput); err != nil {
		return err
	}

	setupPath, err := resolveSetupPath(ctx, deps, renderer, repo, opts, jsonOutput)
	if err != nil {
		return err
	}
	if setupPath == setupPathCloud {
		return runCloudUp(ctx, deps, renderer, repo, decision, opts, jsonOutput)
	}

	status, err := deps.GitHub.VerifyAuth(ctx, repo)
	if err != nil || !status.OK {
		message := fmt.Sprintf("RunnerKit can't create a repository runner registration token for %s.", repo.FullName)
		remediation := status.Remediation
		if len(remediation) == 0 {
			remediation = []string{"Create a fine-grained token scoped only to " + repo.FullName + " with repository Administration read/write and Metadata read, then pass it with RUNNERKIT_GITHUB_TOKEN for this command."}
		}
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

	arch := defaultString(report.Arch, labels.DefaultArch)
	labelSet := labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: arch, Mode: labels.DefaultMode})
	runnerPkg, err := bootstrap.PackageFor("linux", arch)
	if err != nil {
		_ = renderer.Error("unsupported_runner_package", err.Error(), []string{"Use linux/x64 or linux/arm64 for the Phase 2 BYO persistent runner path."})
		return NewExitError(ExitSafetyGate, err)
	}
	bootstrapOpts := buildBootstrapOptions(repo, labelSet, runnerPkg, report)
	bootstrapPlan := bootstrap.Plan(bootstrapOpts)

	if opts.dryRun {
		return renderDryRun(renderer, jsonOutput, repo, status.Source, decision.Warnings, store.Path(), target, report, labelSet, bootstrapPlan)
	}

	runners, err := deps.GitHub.ListRunners(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_runner_list_failed", "RunnerKit can't list repository self-hosted runners.", []string{"Verify GitHub credentials can list repository runners for " + repo.FullName + "."})
		return NewExitError(ExitGitHubAuth, err)
	}
	if existingRunner, found := gh.FindRunnerByName(runners, labelSet.RunnerName); found {
		return runnerNameConflict(renderer, labelSet.RunnerName, existingRunner)
	}

	if err := confirmBootstrapPlan(ctx, deps, renderer, opts, jsonOutput, target); err != nil {
		return err
	}
	token, err := deps.GitHub.CreateRegistrationToken(ctx, repo)
	if err != nil {
		_ = renderer.Error("github_permission_denied", "RunnerKit can't create a fresh runner registration token.", []string{gh.FineGrainedTokenRemediation(repo)})
		return NewExitError(ExitGitHubAuth, err)
	}
	renderer.Redactor().Register(redact.RunnerRegistrationToken, token.Token)
	bootstrapOpts.RunnerToken = token.Token
	if _, err := bootstrap.Apply(ctx, deps.RemoteExecutor, target, bootstrapOpts); err != nil {
		var serviceErr bootstrap.ServiceNotActiveError
		if errors.As(err, &serviceErr) {
			_ = renderer.Error("runner_service_not_active", "RunnerKit installed the runner but the service is not active.", []string{"Run sudo ./svc.sh status in the runner directory or re-run runnerkit up after fixing the service."})
			return NewExitError(ExitSafetyGate, err)
		}
		_ = renderer.Error("bootstrap_failed", "RunnerKit could not apply the BYO runner install plan.", []string{"Review the remote host output, fix the issue, and re-run runnerkit up."})
		return NewExitError(ExitSafetyGate, err)
	}

	onlineRunner, ok, err := waitForRunnerOnline(ctx, deps, repo, labelSet.RunnerName, labelSet.Labels)
	if err != nil {
		return err
	}
	if !ok {
		_ = renderer.Error("runner_online_timeout", "RunnerKit could not verify the GitHub runner came online with the expected labels.", []string{"Check the remote service status and GitHub repository Actions runner settings, then re-run runnerkit up."})
		return NewExitError(ExitSafetyGate, errors.New("runner_online_timeout"))
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
	cloudJSONKeyFutureDestroyCommand = "future_destroy_command"
	cloudJSONKeyEstimatedHourlyCost  = "estimated_hourly_cost"
	cloudJSONKeyEstimatedMonthlyCost = "estimated_monthly_cost"
)

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

func runCloudUp(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, decision gh.SafetyDecision, opts *upOptions, jsonOutput bool) error {
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
		_ = renderer.Error("github_permission_denied", message, remediation)
		if err == nil {
			err = errors.New(message)
		}
		return NewExitError(ExitGitHubAuth, err)
	}
	input := buildCloudProvisionInput(deps, repo, opts)
	registerKnownCloudProviderSecrets(renderer)
	validation, err := cloudProvider.Validate(ctx, input)
	if err != nil || !validation.OK {
		message := "Hetzner credentials are missing. Export HCLOUD_TOKEN or HETZNER_CLOUD_TOKEN, then rerun runnerkit up --repo " + repo.FullName + " --cloud hetzner."
		remediation := validation.Remediation
		if len(remediation) == 0 {
			remediation = []string{"Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>", "Re-run runnerkit up --repo " + repo.FullName + " --cloud hetzner"}
		}
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
		return renderCloudProvisionPlan(renderer, jsonOutput, repo, plan)
	}
	store := rkstate.NewStore(deps.StateBaseDir)
	replaceExisting, err := confirmCloudStateReplaceBeforeProvision(ctx, deps, renderer, opts, jsonOutput, store, repo.FullName)
	if err != nil {
		return err
	}
	if err := confirmCloudProvisionPlan(ctx, deps, renderer, opts, jsonOutput, repo, plan); err != nil {
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
	if jsonOutput {
		return renderer.JSON(map[string]any{"ok": true, "command": "up", "repo": repo.FullName, "runner_installed": false, "state_saved": result.CheckpointRequired, "cloud_plan": plan})
	}
	return renderer.Step(1, 1, "Cloud provisioning accepted", ui.Success("Cloud provisioning request accepted."), ui.Bullet("Runner installation continues after cloud readiness checks."), ui.Next("runnerkit destroy --repo "+repo.FullName))
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
		StateID:         labelSet.RunnerName,
		CreatedAt:       createdAt,
	}
}

func renderCloudProvisionPlan(renderer *ui.Renderer, jsonOutput bool, repo gh.Repo, plan provider.ProvisionPlan) error {
	caveat := defaultString(plan.CostEstimateCaveat, cloudCostCaveatCopy)
	if jsonOutput {
		return renderer.JSON(map[string]any{
			"ok":               true,
			"command":          "up",
			"repo":             repo.FullName,
			"cloud_plan":       plan,
			"runner_installed": false,
			"state_saved":      false,
			"workflow_snippet": plan.WorkflowSnippet,
			"labels":           plan.Labels,
		})
	}
	return renderer.Step(1, 1, cloudProvisioningPlanTitle,
		ui.WarningLine(caveat),
		ui.Bullet("Provider: "+plan.Provider),
		ui.Bullet("Region: "+plan.Region),
		ui.Bullet("Server type: "+plan.ServerType),
		ui.Bullet("Image: "+plan.Image),
		ui.Bullet("Estimated cost: "+plan.EstimatedHourlyCost+", "+plan.EstimatedMonthlyCost),
		ui.Bullet("Resources: server, SSH key, firewall, public IPv4/IPv6"),
		ui.Bullet("Not created: backups, snapshots, volumes, floating IPs"),
		ui.Bullet("Resource names: "+formatCloudResourceNames(plan.ResourceNames)),
		ui.Bullet("Tags: "+formatCloudTags(plan.Tags)),
		ui.Bullet("SSH allowed CIDR: "+plan.SSHAllowedCIDR),
		ui.Bullet("Labels: ["+strings.Join(plan.Labels, ", ")+"]"),
		ui.Bullet(plan.WorkflowSnippet),
		ui.Next("Future cleanup: "+plan.FutureDestroyCommand),
	)
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

func confirmCloudProvisionPlan(ctx context.Context, deps Dependencies, renderer *ui.Renderer, opts *upOptions, jsonOutput bool, repo gh.Repo, plan provider.ProvisionPlan) error {
	if err := renderCloudProvisionPlan(renderer, jsonOutput, repo, plan); err != nil {
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
	repoState := buildCloudRepositoryState(deps, repo, source, decision, input, plan, result, checkpointMessage, artifact, opts.sshKey)
	if err := store.SaveRepository(repoState, replace); err != nil {
		if errors.Is(err, rkstate.ErrRepositoryExists) {
			return replacementRequired(renderer, repo.FullName)
		}
		_ = renderer.Error("state_io_failed", "RunnerKit can't save cloud provisioning state.", []string{"Check permissions for " + store.Path() + " and run runnerkit destroy --repo " + repo.FullName + " if billable resources were created."})
		return NewExitError(ExitStateIO, err)
	}
	return nil
}

func buildCloudRepositoryState(deps Dependencies, repo gh.Repo, source gh.AuthSource, decision gh.SafetyDecision, input provider.ProvisionInput, plan provider.ProvisionPlan, result provider.ProvisionResult, checkpointMessage string, artifact string, keyPathRef string) rkstate.RepositoryState {
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
	cloud.CloudInitVersion = defaultString(cloud.CloudInitVersion, "runnerkit-cloud-init-v1")
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

func renderDryRun(renderer *ui.Renderer, jsonOutput bool, repo gh.Repo, source gh.AuthSource, warnings []string, statePath string, target remote.Target, report preflight.Report, labelSet labels.LabelSet, plan workflow.Plan) error {
	if jsonOutput {
		return renderer.JSON(map[string]any{
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
			"warnings":         warnings,
			"ssh-preflight":    report.Results,
			"bootstrap-plan":   plan,
		})
	}
	if err := renderPreflightHuman(renderer, report); err != nil {
		return err
	}
	return renderer.Step(2, 2, "bootstrap-plan", ui.Bullet("Runner name: "+labelSet.RunnerName), ui.Bullet("Target: "+target.Display()), ui.Bullet("Labels: ["+strings.Join(labelSet.Labels, ", ")+"]"), ui.Bullet(labelSet.RunsOnYAML), ui.WarningLine(labelSet.Warning), ui.Bullet("Dry run: no state file was written and no runner was installed."))
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
		actual := map[string]bool{}
		for _, label := range runner.Labels {
			actual[label] = true
		}
		allPresent := true
		for _, label := range expectedLabels {
			if !actual[label] {
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
	return map[string]any{
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
}

func enforceSafetyDecision(ctx context.Context, deps Dependencies, renderer *ui.Renderer, repo gh.Repo, decision gh.SafetyDecision, opts *upOptions, jsonOutput bool) error {
	if decision.Allowed {
		if decision.Code == gh.SafetyCodePublicRisk && opts.allowPublicRepoRisk && deps.TTY.StdinTTY && !opts.yes {
			inputPrompter, ok := deps.Prompts.(interface {
				Input(context.Context, ui.Prompt) (string, error)
			})
			if !ok {
				message := "RunnerKit can't continue because public repository risk acknowledgement requires typed confirmation."
				_ = renderer.Error("input_required", message, []string{"Type allow public repo risk for " + repo.FullName + " in an interactive terminal or pass --yes only after reviewing the risk."})
				return NewExitError(ExitInputRequired, errors.New(message))
			}
			want := "allow public repo risk for " + repo.FullName
			got, err := inputPrompter.Input(ctx, ui.Prompt{Message: want})
			if err != nil {
				return err
			}
			if got != want {
				message := "Canceled; no changes made."
				_ = renderer.Error("canceled", message, nil)
				return NewExitError(ExitCanceled, errors.New(message))
			}
		}
		return nil
	}
	if jsonOutput {
		_ = renderer.Error(decision.Code, gh.PublicRepoRiskTitle, append(decision.Warnings, gh.PublicRepoRiskNextAction))
	} else if decision.Code == gh.SafetyCodePublicRisk {
		_ = renderer.Warning(gh.PublicRepoRiskTitle, []string{gh.PublicRepoRiskBody}, gh.PublicRepoRiskNextAction)
	} else {
		_ = renderer.Warning("WARNING: Fork repository risk", decision.Warnings, "Use a trusted private repository before persistent setup.")
	}
	return NewExitError(ExitSafetyGate, errors.New(decision.Code))
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
