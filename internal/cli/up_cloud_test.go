package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider/hetzner"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

type flakyCloudInitExecutor struct {
	base              *fakeRemoteExecutor
	cloudInitErrCount int
}

func (f *flakyCloudInitExecutor) Probe(ctx context.Context, target remote.Target) (remote.ProbeResult, error) {
	return f.base.Probe(ctx, target)
}

func (f *flakyCloudInitExecutor) Run(ctx context.Context, target remote.Target, command remote.Command) (remote.Result, error) {
	if command.ID == "cloud.cloudinit.wait" && f.cloudInitErrCount > 0 {
		f.cloudInitErrCount--
		return remote.Result{ExitCode: 255, Stderr: "ssh transport not ready"}, errors.New("exit status 255")
	}
	return f.base.Run(ctx, target, command)
}

func TestUpMissingCloudIntentWithYesFailsBeforeMutation(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Providers:      provider.NewRegistry(cloud),
		StateBaseDir:   stateDir,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing cloud intent error")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
	if !strings.Contains(errOut.String(), "RunnerKit will not create billable cloud resources without an explicit") || !strings.Contains(errOut.String(), "--cloud hetzner flag and --yes.") {
		t.Fatalf("missing cloud intent copy: stdout=%q stderr=%q", out.String(), errOut.String())
	}
	// VerifyAuth, CheckRunnerManagementPermission, and CreateRegistrationToken must remain at zero before explicit cloud intent.
	if service.authCalls != 0 || service.readCalls != 0 || service.tokenCalls != 0 || remoteExec.probeCalls != 0 || len(remoteExec.runs) != 0 || cloud.ProvisionCalls != 0 || cloud.ValidateCalls != 0 || cloud.PlanCalls != 0 {
		t.Fatalf("missing cloud intent should not mutate or auth-check; auth=%d read=%d token=%d probe=%d runs=%d validate=%d plan=%d provision=%d", service.authCalls, service.readCalls, service.tokenCalls, remoteExec.probeCalls, len(remoteExec.runs), cloud.ValidateCalls, cloud.PlanCalls, cloud.ProvisionCalls)
	}
	if _, err := os.Stat(state.NewStore(stateDir).Path()); !os.IsNotExist(err) {
		t.Fatalf("missing intent wrote state or stat failed unexpectedly: %v", err)
	}
}

func TestUpCloudDryRunUsesReadCheckWithoutRegistrationToken(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Providers:      provider.NewRegistry(cloud),
		StateBaseDir:   stateDir,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--cloud", "hetzner", "--yes", "--dry-run", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	// Cloud dry-run uses VerifyRunnerManagementRead and never mints a registration token through VerifyAuth, CheckRunnerManagementPermission, or CreateRegistrationToken.
	if service.readCalls != 1 || service.authCalls != 0 || service.tokenCalls != 0 {
		t.Fatalf("cloud dry-run should use read check only; read=%d auth=%d token=%d", service.readCalls, service.authCalls, service.tokenCalls)
	}
	if remoteExec.probeCalls != 0 || len(remoteExec.runs) != 0 || cloud.ValidateCalls != 1 || cloud.PlanCalls != 1 || cloud.ProvisionCalls != 0 {
		t.Fatalf("cloud dry-run call counts mismatch: probe=%d runs=%d validate=%d plan=%d provision=%d", remoteExec.probeCalls, len(remoteExec.runs), cloud.ValidateCalls, cloud.PlanCalls, cloud.ProvisionCalls)
	}
	if _, err := os.Stat(state.NewStore(stateDir).Path()); !os.IsNotExist(err) {
		t.Fatalf("cloud dry-run wrote state or stat failed unexpectedly: %v", err)
	}
}

func TestUpHostPathDoesNotCallProvider(t *testing.T) {
	service := newFakePermittedGitHubService()
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: newFakeRemoteExecutor(), Providers: provider.NewRegistry(cloud), Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("BYO dry-run returned error: %v\nstderr=%s", err, errOut.String())
	}
	if cloud.ValidateCalls != 0 || cloud.PlanCalls != 0 || cloud.ProvisionCalls != 0 {
		t.Fatalf("BYO path called provider: %#v", cloud)
	}
	if service.authCalls == 0 {
		t.Fatal("BYO path should still use existing VerifyAuth gate")
	}
}

func TestUpUnsupportedCloudProviderErrors(t *testing.T) {
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: newFakePermittedGitHubService(), RemoteExecutor: newFakeRemoteExecutor(), Providers: provider.NewRegistry(cloud), Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--cloud", "digitalocean", "--yes", "--dry-run", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInvalidInput)
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "Supported Phase 4 cloud value: --cloud hetzner.") {
		t.Fatalf("missing unsupported provider remediation: %s", combined)
	}
	if cloud.ValidateCalls != 0 || cloud.PlanCalls != 0 || cloud.ProvisionCalls != 0 {
		t.Fatalf("unsupported provider should not call provider: %#v", cloud)
	}
}

type recordingSetupPathPrompter struct {
	selectCalls    []recordedSetupPathSelect
	confirmCalls   int
	selectedValues []string // one per Select call
}

type recordedSetupPathSelect struct {
	prompt       string
	optionLabels []string
}

func (p *recordingSetupPathPrompter) Confirm(context.Context, ui.Prompt) (bool, error) {
	p.confirmCalls++
	return true, nil
}

func (p *recordingSetupPathPrompter) Select(_ context.Context, prompt ui.Prompt, options []ui.Option) (string, error) {
	idx := len(p.selectCalls)
	var labels []string
	for _, opt := range options {
		labels = append(labels, opt.Label)
	}
	p.selectCalls = append(p.selectCalls, recordedSetupPathSelect{prompt: prompt.Message, optionLabels: labels})
	if idx < len(p.selectedValues) {
		return p.selectedValues[idx], nil
	}
	if len(options) > 0 {
		return options[0].Value, nil
	}
	return "", nil
}

func (p *recordingSetupPathPrompter) Input(context.Context, ui.Prompt) (string, error) {
	return "alice@example.com", nil
}

func TestUpInteractiveNoHostOffersBYOAndCloudChoices(t *testing.T) {
	prompter := &recordingSetupPathPrompter{selectedValues: []string{"byo", "persistent"}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Providers:      provider.NewRegistry(&provider.FakeProvider{}),
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--dry-run", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive BYO dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	if len(prompter.selectCalls) < 2 {
		t.Fatalf("expected at least 2 Select calls, got %d", len(prompter.selectCalls))
	}

	// Step 1: setup-path (BYO vs Cloud).
	step1 := prompter.selectCalls[0]
	if step1.prompt != "Where should the runner run?" {
		t.Fatalf("step 1 prompt = %q", step1.prompt)
	}
	for _, want := range []string{"Bring Your Own machine (BYO)", "Cloud (Hetzner)"} {
		found := false
		for _, label := range step1.optionLabels {
			if label == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("step 1 missing option %q in %v", want, step1.optionLabels)
		}
	}

	// Step 2: mode (Persistent vs Ephemeral).
	step2 := prompter.selectCalls[1]
	if step2.prompt != "Choose runner mode for `owner/repo`:" {
		t.Fatalf("step 2 prompt = %q", step2.prompt)
	}
	for _, want := range []string{"Persistent trusted runner", "Ephemeral one-job runner"} {
		found := false
		for _, label := range step2.optionLabels {
			if label == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("step 2 missing option %q in %v", want, step2.optionLabels)
		}
	}
}

func TestUpCloudDryRunHumanRendersProvisionPlanContract(t *testing.T) {
	service := newFakePermittedGitHubService()
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: newFakeRemoteExecutor(), Providers: provider.NewRegistry(cloud), Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--dry-run", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	stdout := out.String()
	for _, want := range []string{
		"Cloud runner provisioning plan",
		"Estimated cost is approximate",
		"Provider pricing varies by region",
		"time; billing stops",
		"Provider: hetzner",
		"Region: fsn1",
		"Server type: cpx22",
		"Image: ubuntu-24.04",
		"Resources: server, SSH key, firewall, public IPv4/IPv6",
		"Not created: backups, snapshots, volumes, floating IPs",
		"Tags: runnerkit=true, managed=true",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]",
		"Future cleanup: runnerkit destroy --repo owner/name",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("cloud plan output missing %q:\n%s", want, stdout)
		}
	}
}

func TestUpCloudDryRunJSONRendersProvisionPlanWithoutTokenLeak(t *testing.T) {
	const fakeToken = "hcloud-secret"
	t.Setenv("HCLOUD_TOKEN", fakeToken)
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:   "test-version",
		Out:       &out,
		Err:       &errOut,
		GitHub:    newFakePermittedGitHubService(),
		Providers: provider.NewRegistry(&provider.FakeProvider{}),
		Sleep:     noSleep,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--dry-run", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud json dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	if strings.Contains(combined, fakeToken) {
		t.Fatalf("cloud output leaked fake token %q:\n%s", fakeToken, combined)
	}
	for _, want := range []string{`"cloud_plan"`, `"provider":"hetzner"`, `"runner_installed":false`, `"state_saved":false`, `"redactions_applied":true`, `"future_destroy_command":"runnerkit destroy --repo owner/name"`, `"estimated_hourly_cost"`, `"estimated_monthly_cost"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("json output missing %q:\n%s", want, out.String())
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	plan := payload["cloud_plan"].(map[string]any)
	if plan["provider"] != "hetzner" || plan["region"] != "fsn1" || plan["server_type"] != "cpx22" || plan["image"] != "ubuntu-24.04" {
		t.Fatalf("unexpected cloud_plan: %#v", plan)
	}
}

type cloudConfirmationPrompter struct {
	confirmPrompt string
	confirmResult bool
}

func (p *cloudConfirmationPrompter) Confirm(_ context.Context, prompt ui.Prompt) (bool, error) {
	p.confirmPrompt = prompt.Message
	return p.confirmResult, nil
}

func (p *cloudConfirmationPrompter) Select(context.Context, ui.Prompt, []ui.Option) (string, error) {
	return "cloud", nil
}

func TestUpCloudInteractiveConfirmationPrecedesProvision(t *testing.T) {
	prompter := &cloudConfirmationPrompter{confirmResult: true}
	machine := cloudReadyMachineForTest()
	cloud := &provider.FakeProvider{ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true}, WaitReadyOut: machine}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Providers:      provider.NewRegistry(cloud),
		StateBaseDir:   t.TempDir(),
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("confirmed cloud plan returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	if prompter.confirmPrompt != "Create billable Hetzner resources for `owner/name`?" {
		t.Fatalf("confirm prompt = %q", prompter.confirmPrompt)
	}
	if cloud.ProvisionCalls != 1 {
		t.Fatalf("confirmed cloud setup should call Provision once after prompt, got %d", cloud.ProvisionCalls)
	}
}

func TestUpCloudDeclinedConfirmationSkipsProvision(t *testing.T) {
	prompter := &cloudConfirmationPrompter{confirmResult: false}
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:      "test-version",
		Out:          &out,
		Err:          &errOut,
		TTY:          ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:      prompter,
		GitHub:       newFakePermittedGitHubService(),
		Providers:    provider.NewRegistry(cloud),
		StateBaseDir: t.TempDir(),
		Sleep:        noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitCanceled {
		t.Fatalf("expected canceled confirmation, err=%v", err)
	}
	if prompter.confirmPrompt != "Create billable Hetzner resources for `owner/name`?" {
		t.Fatalf("confirm prompt = %q", prompter.confirmPrompt)
	}
	if cloud.ProvisionCalls != 0 {
		t.Fatalf("declined confirmation should not call Provision, got %d", cloud.ProvisionCalls)
	}
}

func TestUpCloudNoTTYWithoutYesSkipsProvision(t *testing.T) {
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: newFakePermittedGitHubService(), Providers: provider.NewRegistry(cloud), StateBaseDir: t.TempDir(), Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("expected input-required confirmation, err=%v", err)
	}
	if !strings.Contains(out.String(), "Cloud runner provisioning plan") || !strings.Contains(errOut.String(), "Pass --yes to create billable Hetzner resources") {
		t.Fatalf("expected plan then confirmation remediation; stdout=%s stderr=%s", out.String(), errOut.String())
	}
	if cloud.ProvisionCalls != 0 {
		t.Fatalf("no-TTY missing --yes should not call Provision, got %d", cloud.ProvisionCalls)
	}
}

func TestUpCloudMissingCredentialsUsesUISpecCopy(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:   "test-version",
		Out:       &out,
		Err:       &errOut,
		GitHub:    newFakePermittedGitHubService(),
		Providers: provider.NewRegistry(hetzner.NewProvider(map[string]string{})),
		Sleep:     noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--dry-run", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("expected missing credentials input-required, err=%v", err)
	}
	combined := out.String() + errOut.String()
	for _, want := range []string{"Hetzner credentials are missing. Export HCLOUD_TOKEN", "HETZNER_CLOUD_TOKEN", "runnerkit up --repo owner/name --cloud", "hetzner.", "Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("missing credential copy %q:\n%s", want, combined)
		}
	}
}

func TestUpCloudProvisionErrorPersistsPendingStateAndDestroyNextAction(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	ids := map[string]string{"server": "srv-123", "ssh_key": "key-123", "firewall": "fw-123", "primary_ipv4": "ip-123"}
	partial := provider.ProvisionResult{
		Machine: provider.Machine{
			Target:      remote.Target{User: "runnerkit-admin", Host: "203.0.113.10", Port: 22, Raw: "runnerkit-admin@203.0.113.10:22"},
			PublicIPv4:  "203.0.113.10",
			ResourceIDs: ids,
			Provider:    state.ProviderRef{Kind: "hetzner", Region: "fsn1", IDs: ids},
		},
		CreatedResourceIDs: ids,
		CheckpointRequired: true,
	}
	cloud := &provider.FakeProvider{ProvisionErr: &provider.ProvisionError{Stage: "server", Result: partial, Err: errors.New("server action failed")}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:      "test-version",
		Out:          &out,
		Err:          &errOut,
		GitHub:       service,
		Providers:    provider.NewRegistry(cloud),
		StateBaseDir: stateDir,
		Sleep:        noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected safety gate provision error, err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "Hetzner provisioning failed after billable resources were created.") || !strings.Contains(combined, "runnerkit destroy --repo owner/name") {
		t.Fatalf("missing provision failure and destroy next action:\n%s", combined)
	}
	if service.authCalls != 0 || service.tokenCalls != 0 {
		t.Fatalf("provision failure should not mint registration tokens; auth=%d token=%d", service.authCalls, service.tokenCalls)
	}
	loaded, found, loadErr := state.NewStore(stateDir).GetRepository("owner/name")
	if loadErr != nil || !found {
		t.Fatalf("pending cloud state not saved found=%v err=%v", found, loadErr)
	}
	if len(loaded.Operations) != 1 || loaded.Operations[0].Message != "cloud_provision_pending" || loaded.Operations[0].Artifact != "provider" {
		t.Fatalf("missing cloud_provision_pending operation: %#v", loaded.Operations)
	}
	joinedIDs := strings.Join(loaded.Cleanup.ProviderResourceIDs, ",")
	for _, want := range []string{"server:srv-123", "ssh_key:key-123", "firewall:fw-123", "primary_ipv4:ip-123"} {
		if !strings.Contains(joinedIDs, want) {
			t.Fatalf("provider resource ids missing %q: %#v", want, loaded.Cleanup.ProviderResourceIDs)
		}
	}
	if loaded.Provider.Cloud.ServerID != "srv-123" || loaded.Provider.Cloud.SSHKeyID != "key-123" || loaded.Provider.Cloud.FirewallID != "fw-123" || loaded.Provider.Cloud.PrimaryIPv4ID != "ip-123" || loaded.Provider.Cloud.CostProfile.Provider != "hetzner" {
		t.Fatalf("cloud inventory not persisted: %#v", loaded.Provider.Cloud)
	}
}

func TestUpCloudReadinessFailureBlocksRegistrationTokenAndKeepsPendingState(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	machine := cloudReadyMachineForTest()
	cloud := &provider.FakeProvider{
		ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true},
		WaitReadyErr: errors.New("provider still provisioning"),
	}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         service,
		RemoteExecutor: newFakeRemoteExecutor(),
		Providers:      provider.NewRegistry(cloud),
		StateBaseDir:   stateDir,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected readiness failure, err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "Cloud machine is not ready for runner registration yet") || !strings.Contains(combined, "runnerkit destroy --repo owner/name") {
		t.Fatalf("missing readiness failure guidance:\n%s", combined)
	}
	if service.authCalls != 0 || service.tokenCalls != 0 {
		t.Fatalf("readiness failure should not call registration-token auth; auth=%d token=%d", service.authCalls, service.tokenCalls)
	}
	if cloud.ProvisionCalls != 1 || cloud.WaitReadyCalls != 1 {
		t.Fatalf("provider call counts provision=%d wait=%d", cloud.ProvisionCalls, cloud.WaitReadyCalls)
	}
	loaded, found, loadErr := state.NewStore(stateDir).GetRepository("owner/name")
	if loadErr != nil || !found {
		t.Fatalf("pending state not saved found=%v err=%v", found, loadErr)
	}
	if len(loaded.Operations) != 1 || loaded.Operations[0].Message != "cloud_readiness_pending" || loaded.Operations[0].Artifact != "readiness" {
		t.Fatalf("missing cloud_readiness_pending operation: %#v", loaded.Operations)
	}
}

func TestUpCloudReadinessSuccessPrecedesRegistrationToken(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	machine := cloudReadyMachineForTest()
	cloud := &provider.FakeProvider{
		ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true},
		WaitReadyOut: machine,
	}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Providers:      provider.NewRegistry(cloud),
		StateBaseDir:   stateDir,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud readiness success returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	if cloud.ProvisionCalls != 1 || cloud.WaitReadyCalls != 1 || service.tokenCalls != 1 || service.authCalls != 0 {
		t.Fatalf("unexpected call counts provision=%d wait=%d token=%d auth=%d", cloud.ProvisionCalls, cloud.WaitReadyCalls, service.tokenCalls, service.authCalls)
	}
	ids := []string{}
	for _, command := range remoteExec.runs {
		ids = append(ids, command.ID)
	}
	joined := strings.Join(ids, ",")
	for _, want := range []string{"cloud.cloudinit.wait", "host.network.github.github", "host.network.github.api", "configure_runner", "install_service", "verify_service"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing readiness/bootstrap command %q in %v", want, ids)
		}
	}
	if indexOf(ids, "cloud.cloudinit.wait") > indexOf(ids, "host.network.github.github") || indexOf(ids, "configure_runner") > indexOf(ids, "verify_service") {
		t.Fatalf("cloud readiness/bootstrap order mismatch: %v", ids)
	}
	if len(cloud.ProvisionInput) != 1 {
		t.Fatalf("expected one provision input, got %d", len(cloud.ProvisionInput))
	}
	labels := strings.Join(cloud.ProvisionInput[0].Labels, ",")
	for _, want := range []string{"runnerkit-owner-name", "linux", "x64", "persistent"} {
		if !strings.Contains(labels, want) {
			t.Fatalf("cloud labels missing %q: %s", want, labels)
		}
	}
	loaded, found, loadErr := state.NewStore(stateDir).GetRepository("owner/name")
	if loadErr != nil || !found {
		t.Fatalf("final cloud state not saved found=%v err=%v", found, loadErr)
	}
	stateJSON, _ := json.Marshal(loaded)
	for _, want := range []string{`"kind":"cloud-ssh"`, `"kind":"hetzner"`, `"provider":"hetzner"`, `"server_id":"srv-123"`, `"ssh_key_id":"key-123"`, `"firewall_id":"fw-123"`, `"provider_resource_ids"`} {
		if !strings.Contains(string(stateJSON), want) {
			t.Fatalf("saved cloud state missing %s:\n%s", want, stateJSON)
		}
	}
	if strings.Contains(string(stateJSON), cloudProvisionPending) || strings.Contains(string(stateJSON), cloudReadinessPending) || len(loaded.Operations) != 0 {
		t.Fatalf("final cloud state retained pending checkpoints: %#v\n%s", loaded.Operations, stateJSON)
	}
	combined := out.String() + errOut.String()
	for _, want := range []string{"Provider: Hetzner fsn1 cpx22 ubuntu-24.04", "Cleanup: runnerkit destroy --repo owner/name", "Billable resources: server:srv-123, ssh_key:key-123, firewall:fw-123"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("cloud completion output missing %q:\n%s", want, combined)
		}
	}
}

func TestUpCloudEphemeralPlanRendersHetznerCostCaveatBeforeProvision(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	cloud := &provider.FakeProvider{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:      "test-version",
		Out:          &out,
		Err:          &errOut,
		GitHub:       service,
		Providers:    provider.NewRegistry(cloud),
		StateBaseDir: stateDir,
		Sleep:        noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--cloud", "hetzner", "--yes", "--dry-run", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral cloud dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	flat := strings.Join(strings.Fields(out.String()), " ")
	if !strings.Contains(flat, "Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.") {
		t.Fatalf("ephemeral cloud plan missing exact Hetzner cost caveat:\n%s", flat)
	}
	if cloud.ProvisionCalls != 0 {
		t.Fatalf("dry-run must not call Provision: %d", cloud.ProvisionCalls)
	}
}

func cloudReadyMachineForTest() provider.Machine {
	ids := map[string]string{"server": "srv-123", "ssh_key": "key-123", "firewall": "fw-123", "primary_ipv4": "ip-123"}
	return provider.Machine{
		Target:      remote.Target{User: "runnerkit-admin", Host: "203.0.113.10", Port: 22, Raw: "runnerkit-admin@203.0.113.10:22"},
		PublicIPv4:  "203.0.113.10",
		ResourceIDs: ids,
		Provider: state.ProviderRef{
			Kind:        "hetzner",
			Name:        "hetzner",
			Region:      "fsn1",
			Profile:     "cpx22",
			IDs:         ids,
			ResourceIDs: ids,
			Cloud:       state.CloudInventory{Provider: "hetzner", ServerID: "srv-123", Region: "fsn1", ServerType: "cpx22", Image: "ubuntu-24.04", PublicIPv4: "203.0.113.10"},
		},
	}
}

func indexOf(values []string, want string) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return len(values) + 1
}

func TestBuildCloudProvisionInputReadsPublicKeyFromSSHKeyFlag(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "runnerkit_ed25519")
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest runnerkit@example"
	if err := os.WriteFile(keyPath+".pub", []byte(publicKey+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := buildCloudProvisionInput(Dependencies{Clock: func() time.Time { return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) }}, github.Repo{Owner: "owner", Name: "name", FullName: "owner/name"}, &upOptions{sshKey: keyPath}, nil)
	if input.PublicKey != publicKey {
		t.Fatalf("PublicKey = %q, want %q", input.PublicKey, publicKey)
	}
}

// Bug 29 (Plan 06-12, 2026-05-06): the cloud-init wait command at
// up.go:908 had no explicit Timeout, so it inherited whatever default
// the executor applied. Plan 06-07 attempt-17 cloud smoke aborted at
// 42s with `cloud_readiness_failed` even though Hetzner cpx22 +
// ubuntu-24.04 cloud-init typically needs 60-120s.
//
// The fix gives the command an explicit Timeout aligned with
// hetzner.HostKeyProbeOptions (default 60×5s = 300s) and exposes
// RUNNERKIT_CLOUD_INIT_TIMEOUT as an override for slower regions /
// images.
//
// This test asserts (via the runs slice on the fake executor) that
// the cloud.cloudinit.wait command carries a non-zero Timeout >= 120s
// by default, that RUNNERKIT_CLOUD_INIT_TIMEOUT="45s" overrides it
// to 45s, and that an unparseable value falls back to the default.
func TestWaitCloudTargetReady_HonorsCloudInitTimeoutBudget(t *testing.T) {
	cases := []struct {
		name        string
		envValue    string
		setEnv      bool
		wantTimeout time.Duration
	}{
		{name: "default_budget", setEnv: false, wantTimeout: 10 * time.Minute},
		{name: "override_45s", setEnv: true, envValue: "45s", wantTimeout: 45 * time.Second},
		{name: "invalid_falls_back", setEnv: true, envValue: "not-a-duration", wantTimeout: 10 * time.Minute},
		{name: "empty_falls_back", setEnv: true, envValue: "", wantTimeout: 10 * time.Minute},
		{name: "zero_falls_back", setEnv: true, envValue: "0s", wantTimeout: 10 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv("RUNNERKIT_CLOUD_INIT_TIMEOUT", tc.envValue)
			} else {
				t.Setenv("RUNNERKIT_CLOUD_INIT_TIMEOUT", "")
			}
			stateDir := t.TempDir()
			service := newFakePermittedGitHubService()
			remoteExec := newFakeRemoteExecutor()
			machine := cloudReadyMachineForTest()
			cloud := &provider.FakeProvider{
				ProvisionOut: provider.ProvisionResult{Machine: machine, CreatedResourceIDs: machine.ResourceIDs, CheckpointRequired: true},
				WaitReadyOut: machine,
			}
			var out, errOut bytes.Buffer
			cmd := NewRootCommand(Dependencies{
				Version:        "test-version",
				Out:            &out,
				Err:            &errOut,
				GitHub:         service,
				RemoteExecutor: remoteExec,
				Providers:      provider.NewRegistry(cloud),
				StateBaseDir:   stateDir,
				Sleep:          noSleep,
			})
			cmd.SetArgs([]string{"up", "--repo", "owner/name", "--cloud", "hetzner", "--yes", "--no-color"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("up cloud returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
			}
			var found *remote.Command
			for i := range remoteExec.runs {
				if remoteExec.runs[i].ID == "cloud.cloudinit.wait" {
					cmd := remoteExec.runs[i]
					found = &cmd
					break
				}
			}
			if found == nil {
				ids := []string{}
				for _, c := range remoteExec.runs {
					ids = append(ids, c.ID)
				}
				t.Fatalf("Bug 29: expected cloud.cloudinit.wait command in run list; got IDs=%v", ids)
			}
			if found.Timeout != tc.wantTimeout {
				t.Fatalf("Bug 29: cloud.cloudinit.wait Timeout = %v, want %v (env=%q set=%v)", found.Timeout, tc.wantTimeout, tc.envValue, tc.setEnv)
			}
			if tc.wantTimeout < 120*time.Second {
				return
			}
			// Default budget guard: must be >= 120s (Hetzner cpx22 typical
			// cloud-init wall-clock with retry headroom).
			if found.Timeout < 120*time.Second {
				t.Fatalf("Bug 29: default cloud-init Timeout must be >= 120s; got %v", found.Timeout)
			}
		})
	}
}

func TestWaitCloudTargetReady_RetriesTransientCloudInitSSHError(t *testing.T) {
	t.Setenv("RUNNERKIT_CLOUD_INIT_TIMEOUT", "30s")
	base := newFakeRemoteExecutor()
	exec := &flakyCloudInitExecutor{base: base, cloudInitErrCount: 2}
	machine := cloudReadyMachineForTest()
	deps := Dependencies{
		RemoteExecutor: exec,
		Sleep:          noSleep,
	}
	report, hostKey, readyMachine, err := waitCloudTargetReady(context.Background(), deps, machine)
	if err != nil {
		t.Fatalf("waitCloudTargetReady returned error after transient cloud-init SSH failures: %v", err)
	}
	if hostKey.Fingerprint == "" {
		t.Fatal("expected normalized host key fingerprint")
	}
	if !report.Passed() {
		t.Fatalf("preflight report should pass: %#v", report.Results)
	}
	if readyMachine.Target.Host == "" {
		t.Fatalf("ready machine target host must be populated: %#v", readyMachine.Target)
	}
	if exec.cloudInitErrCount != 0 {
		t.Fatalf("expected transient cloud-init SSH errors to be exhausted, remaining=%d", exec.cloudInitErrCount)
	}
}

func TestWaitCloudTargetReady_UsesRootForCloudInitWait(t *testing.T) {
	base := newFakeRemoteExecutor()
	machine := cloudReadyMachineForTest()
	deps := Dependencies{
		RemoteExecutor: base,
		Sleep:          noSleep,
	}
	_, _, _, err := waitCloudTargetReady(context.Background(), deps, machine)
	if err != nil {
		t.Fatalf("waitCloudTargetReady returned error: %v", err)
	}
	seenCloudInitWait := false
	for i, cmd := range base.runs {
		if cmd.ID == "cloud.cloudinit.wait" {
			seenCloudInitWait = true
			if i >= len(base.runTargets) {
				t.Fatalf("missing recorded run target for cloud.cloudinit.wait")
			}
			if base.runTargets[i].User != "root" {
				t.Fatalf("cloud.cloudinit.wait must run as root to avoid runnerkit-admin creation race; got user=%q", base.runTargets[i].User)
			}
			break
		}
	}
	if !seenCloudInitWait {
		t.Fatalf("expected cloud.cloudinit.wait command in runs: %#v", base.runs)
	}
}

func TestCloudInitWaitScript_NoBootFinishedOrMaskOnCloudInit(t *testing.T) {
	s := cloudInitWaitScript()
	if strings.Contains(s, "|| test -f /var/lib/cloud/instance/boot-finished") {
		t.Fatalf("must not OR boot-finished after cloud-init --wait: cloud-init status=error can still leave boot-finished, falsely succeeding readiness")
	}
	if !strings.Contains(s, "status: error") {
		t.Fatalf("expected explicit status=error rejection in script")
	}
}

func TestParseExtraPackages(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"  ", nil},
		{"libsecret-1-dev,dbus-x11,gnome-keyring", []string{"libsecret-1-dev", "dbus-x11", "gnome-keyring"}},
		{"  libsecret-1-dev , dbus-x11 ", []string{"libsecret-1-dev", "dbus-x11"}},
		{"curl,,tar", []string{"curl", "tar"}},
		{"bad;pkg,good-pkg", []string{"good-pkg"}},
		{"ok_pkg:amd64", []string{"ok_pkg:amd64"}},
	}
	for _, tt := range tests {
		got := parseExtraPackages(tt.input)
		if len(got) != len(tt.want) {
			t.Fatalf("parseExtraPackages(%q) = %v, want %v", tt.input, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("parseExtraPackages(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestIsValidPackageName(t *testing.T) {
	valid := []string{"curl", "libsecret-1-dev", "pkg:amd64", "gcc-12", "libc++abi-dev", "a.b_c"}
	for _, name := range valid {
		if !isValidPackageName(name) {
			t.Fatalf("isValidPackageName(%q) = false, want true", name)
		}
	}
	invalid := []string{"", "bad;name", "pkg name", "$(cmd)", "pkg&&echo", "a/b"}
	for _, name := range invalid {
		if isValidPackageName(name) {
			t.Fatalf("isValidPackageName(%q) = true, want false", name)
		}
	}
}

func TestResolveExtraPackagesDeduplicates(t *testing.T) {
	got := resolveExtraPackages("curl,libsecret-1-dev", []string{"libsecret-1-dev", "dbus-x11"}, nil)
	want := []string{"curl", "libsecret-1-dev", "dbus-x11"}
	if len(got) != len(want) {
		t.Fatalf("resolveExtraPackages = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("resolveExtraPackages[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildCloudProvisionInputIncludesExtraPackages(t *testing.T) {
	pkgs := []string{"libsecret-1-dev", "dbus-x11"}
	input := buildCloudProvisionInput(
		Dependencies{Clock: func() time.Time { return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) }},
		github.Repo{Owner: "owner", Name: "name", FullName: "owner/name"},
		&upOptions{},
		pkgs,
	)
	if len(input.ExtraPackages) != 2 || input.ExtraPackages[0] != "libsecret-1-dev" {
		t.Fatalf("ExtraPackages = %v, want %v", input.ExtraPackages, pkgs)
	}
}

func TestResolveExtraPackagesMergesAutoDetected(t *testing.T) {
	got := resolveExtraPackages("curl", nil, []string{"libssl-dev", "curl"})
	want := []string{"curl", "libssl-dev"}
	if len(got) != len(want) {
		t.Fatalf("resolveExtraPackages = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("resolveExtraPackages[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
