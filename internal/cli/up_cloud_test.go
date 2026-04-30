package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/ui"
)

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
	selectPrompt  string
	optionLabels  []string
	confirmCalls  int
	selectedValue string
}

func (p *recordingSetupPathPrompter) Confirm(context.Context, ui.Prompt) (bool, error) {
	p.confirmCalls++
	return true, nil
}

func (p *recordingSetupPathPrompter) Select(_ context.Context, prompt ui.Prompt, options []ui.Option) (string, error) {
	p.selectPrompt = prompt.Message
	p.optionLabels = p.optionLabels[:0]
	for _, opt := range options {
		p.optionLabels = append(p.optionLabels, opt.Label)
	}
	if p.selectedValue == "" {
		return "byo", nil
	}
	return p.selectedValue, nil
}

func (p *recordingSetupPathPrompter) Input(context.Context, ui.Prompt) (string, error) {
	return "alice@example.com", nil
}

func TestUpInteractiveNoHostOffersBYOAndCloudChoices(t *testing.T) {
	prompter := &recordingSetupPathPrompter{selectedValue: "byo"}
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
	if prompter.selectPrompt != "Choose setup path for `owner/repo`:" {
		t.Fatalf("select prompt = %q", prompter.selectPrompt)
	}
	joined := strings.Join(prompter.optionLabels, "\n")
	for _, want := range []string{"Use existing SSH host (BYO)", "Provision recommended cloud runner (Hetzner)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing option %q in %#v", want, prompter.optionLabels)
		}
	}
}
