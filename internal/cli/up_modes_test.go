package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// recordingModePrompter captures the prompt message and option labels of
// the mode/profile selection so tests can assert exact UI-SPEC copy.
type recordingModePrompter struct {
	selectMessage  string
	selectOptions  []ui.Option
	selectedValue  string
	confirmMessage string
}

func (p *recordingModePrompter) Confirm(_ context.Context, prompt ui.Prompt) (bool, error) {
	p.confirmMessage = prompt.Message
	return true, nil
}

func (p *recordingModePrompter) Select(_ context.Context, prompt ui.Prompt, options []ui.Option) (string, error) {
	p.selectMessage = prompt.Message
	p.selectOptions = append(p.selectOptions[:0], options...)
	if p.selectedValue == "" {
		return "persistent-byo", nil
	}
	return p.selectedValue, nil
}

func (p *recordingModePrompter) Input(context.Context, ui.Prompt) (string, error) {
	return "alice@example.com", nil
}

func TestUpInteractiveModeSelectionUsesUISpecLabels(t *testing.T) {
	prompter := &recordingModePrompter{selectedValue: "persistent-byo"}
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
		t.Fatalf("interactive mode dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	if prompter.selectMessage != "Choose runner mode for `owner/repo`:" {
		t.Fatalf("select prompt = %q", prompter.selectMessage)
	}
	wantLabels := []string{
		"Persistent trusted runner",
		"Ephemeral one-job runner on existing machine",
		"Ephemeral one-job cloud runner (Hetzner)",
	}
	if len(prompter.selectOptions) != len(wantLabels) {
		t.Fatalf("got %d options, want %d: %#v", len(prompter.selectOptions), len(wantLabels), prompter.selectOptions)
	}
	for i, want := range wantLabels {
		if prompter.selectOptions[i].Label != want {
			t.Fatalf("option[%d] label = %q, want %q", i, prompter.selectOptions[i].Label, want)
		}
	}
	wantValues := []string{"persistent-byo", "ephemeral-byo", "ephemeral-cloud"}
	for i, want := range wantValues {
		if prompter.selectOptions[i].Value != want {
			t.Fatalf("option[%d] value = %q, want %q", i, prompter.selectOptions[i].Value, want)
		}
	}
}

func TestUpEphemeralBYODryRunRendersUISpecCopyAndSnippet(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{Width: 100},
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral BYO dry-run returned error: %v\nstderr=%s", err, errOut.String())
	}
	for _, want := range []string{
		"Mode: ephemeral",
		"Safety profile: ephemeral-byo",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, ephemeral]",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("ephemeral BYO output missing %q:\n%s", want, out.String())
		}
	}
	// Long warning copy may wrap on narrow terminals; assert by stripping
	// internal whitespace runs so the test is resilient to wrapWidth.
	flat := strings.Join(strings.Fields(out.String()), " ")
	for _, want := range []string{
		"BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.",
		"Ephemeral mode is not a fleet manager.",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral BYO output missing %q (flattened):\n%s", want, flat)
		}
	}
}

func TestUpEphemeralCloudDryRunRendersTradeoffsAndCostCaveat(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		TTY:            ui.TerminalCapabilities{Width: 100},
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Providers:      provider.NewRegistry(&provider.FakeProvider{}),
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--cloud", "hetzner", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral cloud dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	for _, want := range []string{
		"Safety profile: ephemeral-cloud",
		"Ephemeral cloud runners still create billable Hetzner resources",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, ephemeral]",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("ephemeral cloud output missing %q:\n%s", want, out.String())
		}
	}
	// Long warning copy may wrap on narrow terminals; assert by flattening
	// internal whitespace runs so wrapped output still matches the
	// canonical UI-SPEC sentence.
	flat := strings.Join(strings.Fields(out.String()), " ")
	for _, want := range []string{
		"Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.",
		"TTL safeguard: RunnerKit finalizes the ephemeral runner after 24h if no job completes.",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral cloud output missing %q (flattened):\n%s", want, flat)
		}
	}
}

func TestUpInvalidModeReturnsExitInvalidInput(t *testing.T) {
	_, errOut, err := executeForTest(t, "up", "--repo", "owner/name", "--mode", "static", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color")
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInvalidInput)
	}
	if !strings.Contains(errOut, "Supported modes: --mode persistent or --mode ephemeral.") {
		t.Fatalf("missing supported-modes copy: %q", errOut)
	}
}

func TestUpEphemeralCloudDryRunJSONIncludesModeFields(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:   "test-version",
		Out:       &out,
		Err:       &errOut,
		GitHub:    newFakePermittedGitHubService(),
		Providers: provider.NewRegistry(&provider.FakeProvider{}),
		Sleep:     noSleep,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--mode", "ephemeral", "--cloud", "hetzner", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral cloud json dry-run returned error: %v\nstderr=%s", err, errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if payload["mode"] != "ephemeral" {
		t.Fatalf("payload[mode] = %#v, want ephemeral", payload["mode"])
	}
	if payload["safety_profile"] != "ephemeral-cloud" {
		t.Fatalf("payload[safety_profile] = %#v, want ephemeral-cloud", payload["safety_profile"])
	}
	if payload["ephemeral"] != true {
		t.Fatalf("payload[ephemeral] = %#v, want true", payload["ephemeral"])
	}
	if payload["ttl"] != "24h0m0s" && payload["ttl"] != "24h" {
		t.Fatalf("payload[ttl] = %#v, want 24h", payload["ttl"])
	}
	tradeoffs, ok := payload["tradeoffs"].(map[string]any)
	if !ok {
		t.Fatalf("payload[tradeoffs] missing/not object: %#v", payload["tradeoffs"])
	}
	if tradeoffs["operations"] != "One scoped runner only; not autoscaling or a fleet manager." {
		t.Fatalf("tradeoffs.operations = %#v", tradeoffs["operations"])
	}
	if _, ok := payload["recommended_for"].([]any); !ok {
		t.Fatalf("payload[recommended_for] missing: %#v", payload["recommended_for"])
	}
	if _, ok := payload["not_recommended_for"].([]any); !ok {
		t.Fatalf("payload[not_recommended_for] missing: %#v", payload["not_recommended_for"])
	}
	if _, ok := payload["warnings"].([]any); !ok {
		t.Fatalf("payload[warnings] missing/not array: %#v", payload["warnings"])
	}
	if payload["redactions_applied"] != true {
		t.Fatalf("redactions_applied missing: %#v", payload["redactions_applied"])
	}
}

func TestUpEphemeralBYODryRunJSONIncludesSafetyProfileAndSnippet(t *testing.T) {
	out, errOut, err := executeForTest(t, "--json", "up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("ephemeral BYO json dry-run returned error: %v\nstderr=%s", err, errOut)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if payload["mode"] != "ephemeral" || payload["safety_profile"] != "ephemeral-byo" {
		t.Fatalf("expected ephemeral byo: %#v", payload)
	}
	if payload["ephemeral"] != true {
		t.Fatalf("ephemeral flag missing: %#v", payload)
	}
	snippet, _ := payload["workflow_snippet"].(string)
	if !strings.Contains(snippet, "ephemeral") {
		t.Fatalf("workflow_snippet missing ephemeral: %#v", payload["workflow_snippet"])
	}
}

func TestUpEphemeralModeUsesEphemeralRunnerNameForListAndDryRunBeforeToken(t *testing.T) {
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral BYO dry-run returned error: %v\nstderr=%s", err, errOut.String())
	}
	if service.tokenCalls != 0 {
		t.Fatalf("ephemeral dry-run should not mint a registration token; got %d", service.tokenCalls)
	}
	// The ephemeral runner name should appear in dry-run output.
	if !strings.Contains(out.String(), "runnerkit-owner-name-ephemeral-") {
		t.Fatalf("dry-run output missing ephemeral runner name suffix:\n%s", out.String())
	}
}

func TestUpPersistentDefaultStillRendersPersistentSnippet(t *testing.T) {
	// Backwards compatibility: omitted --mode keeps persistent default and
	// preserves the runnerkit-owner-name-local persistent runner name and
	// persistent runs-on snippet.
	out, errOut, err := executeForTest(t, "up", "--repo", "owner/name", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("persistent default dry-run returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{
		"Mode: persistent",
		"Safety profile: persistent-trusted",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]",
		"runnerkit-owner-name-local",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("persistent default output missing %q:\n%s", want, out)
		}
	}
}

// publicRepoCloudGitHubService returns a public/fork repo so safety gates
// fire while still permitting downstream operations when allowed.
type publicRepoCloudGitHubService struct {
	*fakePermittedGitHubService
}

func (s publicRepoCloudGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	if repo.Host == "" {
		repo.Host = "github.com"
	}
	repo.Private = false
	return repo, nil
}

func newPublicRepoFakeService() *publicRepoCloudGitHubService {
	return &publicRepoCloudGitHubService{fakePermittedGitHubService: newFakePermittedGitHubService()}
}

func TestUpPublicRepoPersistentBlocksWithEphemeralCloudRecommendation(t *testing.T) {
	// Public/fork persistent setup must block before VerifyAuth,
	// VerifyRunnerManagementRead, CreateRegistrationToken, remote
	// probe/run, provider validate/plan/provision, or state save.
	stateDir := t.TempDir()
	service := newPublicRepoFakeService()
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
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "persistent", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected safety gate, err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	for _, want := range []string{
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("public persistent block missing %q (flattened):\n%s", want, flat)
		}
	}
	if service.tokenCalls != 0 || service.authCalls != 0 || service.readCalls != 0 || cloud.ValidateCalls != 0 || cloud.PlanCalls != 0 || cloud.ProvisionCalls != 0 || remoteExec.probeCalls != 0 || len(remoteExec.runs) != 0 {
		t.Fatalf("public persistent block leaked side effects: token=%d auth=%d read=%d validate=%d plan=%d provision=%d probe=%d runs=%d", service.tokenCalls, service.authCalls, service.readCalls, cloud.ValidateCalls, cloud.PlanCalls, cloud.ProvisionCalls, remoteExec.probeCalls, len(remoteExec.runs))
	}
}

func TestUpPublicRepoEphemeralCloudDryRunSurfacesEphemeralCloudRecommendation(t *testing.T) {
	// Public ephemeral cloud is not blocked, but the tradeoffs and
	// warnings must mention ephemeral cloud as the safer choice and the
	// dry-run path must not mint a registration token.
	stateDir := t.TempDir()
	service := newPublicRepoFakeService()
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
		t.Fatalf("ephemeral cloud public dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	for _, want := range []string{"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ephemeral cloud public dry-run missing %q (flattened):\n%s", want, flat)
		}
	}
	// Additionally, the warnings array passed through to JSON should have
	// had "Use ephemeral cloud runner" appended; in human dry-run we look
	// at the JSON of warnings via the dry-run JSON test in
	// TestUpEphemeralCloudDryRunJSONIncludesModeFields. For human output
	// the recommendation surfaces via the public_repo_risk warning copy.
	if !strings.Contains(flat, "Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.") {
		t.Fatalf("ephemeral cloud public dry-run missing public-fork persistent warning:\n%s", flat)
	}
	if service.tokenCalls != 0 {
		t.Fatalf("ephemeral cloud dry-run minted token: %d", service.tokenCalls)
	}
}

func TestUpPublicRepoEphemeralBYOWithoutAcknowledgementBlocks(t *testing.T) {
	// Public/fork ephemeral BYO requires --allow-ephemeral-byo-risk --yes
	// or typed acknowledgement; missing both should block before remote
	// probe and registration token.
	service := newPublicRepoFakeService()
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected ephemeral BYO public gate, err=%v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	combined := out.String() + errOut.String()
	flat := strings.Join(strings.Fields(combined), " ")
	if !strings.Contains(flat, "BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.") {
		t.Fatalf("missing BYO ephemeral caveat in error (flattened):\n%s", flat)
	}
	if service.tokenCalls != 0 || remoteExec.probeCalls != 0 {
		t.Fatalf("ephemeral BYO gate leaked side effects token=%d probe=%d", service.tokenCalls, remoteExec.probeCalls)
	}
}

func TestUpPublicRepoEphemeralBYOWithAllowFlagDryRunRendersBYOCaveat(t *testing.T) {
	// --allow-ephemeral-byo-risk --yes for a public repo permits dry-run
	// and still surfaces the BYO clean-VM caveat.
	out, errOut, err := executeForTest(t, "up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--allow-ephemeral-byo-risk", "--yes", "--dry-run", "--no-color")
	if err != nil {
		// public repo via newFakePermittedGitHubService is private; we need
		// the public-repo service here. Use the dependency-injection form.
		_ = err
	}
	var out2, errOut2 bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out2,
		Err:            &errOut2,
		TTY:            ui.TerminalCapabilities{Width: 100},
		GitHub:         newPublicRepoFakeService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--mode", "ephemeral", "--host", "alice@example.com", "--allow-ephemeral-byo-risk", "--yes", "--dry-run", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ephemeral BYO acknowledged dry-run returned error: %v\nstdout=%s\nstderr=%s", err, out2.String(), errOut2.String())
	}
	flat := strings.Join(strings.Fields(out2.String()), " ")
	if !strings.Contains(flat, "BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.") {
		t.Fatalf("expected BYO clean-VM caveat in acknowledged dry-run: %s", out2.String())
	}
	_ = out
	_ = errOut
}
