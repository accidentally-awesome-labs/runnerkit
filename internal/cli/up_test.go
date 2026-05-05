package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

func TestUpDryRunDisplaysBYOPreflightAndPlan(t *testing.T) {
	out, errOut, err := executeForTest(t, "up", "--dry-run", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up dry-run returned error: %v\nstderr: %s", err, errOut)
	}
	for _, step := range []string{"ssh-preflight", "bootstrap-plan"} {
		if !strings.Contains(out, step) {
			t.Fatalf("dry-run output missing step %q:\n%s", step, out)
		}
	}
	for _, copy := range []string{"alice@example.com:22", "runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]"} {
		if !strings.Contains(out, copy) {
			t.Fatalf("dry-run output missing copy %q:\n%s", copy, out)
		}
	}
}

func TestUpNonInteractiveRequiresRepo(t *testing.T) {
	_, errOut, err := executeForTest(t, "up", "--non-interactive", "--no-color")
	if err == nil {
		t.Fatal("expected missing repo error")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
	if !strings.Contains(errOut, "--repo owner/name") {
		t.Fatalf("missing remediation in stderr: %q", errOut)
	}
}

func TestUpJSONDryRunContract(t *testing.T) {
	out, errOut, err := executeForTest(t, "--json", "up", "--dry-run", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up json dry-run returned error: %v\nstderr: %s", err, errOut)
	}
	if strings.Contains(out, "\x1b[") || !strings.HasPrefix(out, "{") {
		t.Fatalf("json output is not machine-only: %q", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json output invalid: %v\n%s", err, out)
	}
	if payload["runner_installed"] != false || payload["redactions_applied"] != true {
		t.Fatalf("unexpected up payload: %#v", payload)
	}
}

type fakeGitCommandRunner struct {
	output string
}

func (f fakeGitCommandRunner) LookPath(name string) (string, error) { return name, nil }
func (f fakeGitCommandRunner) Run(_ context.Context, _ string, _ ...string) (string, error) {
	return f.output, nil
}

type denyingRepoPrompter struct{}

func (denyingRepoPrompter) Confirm(context.Context, ui.Prompt) (bool, error) { return false, nil }
func (denyingRepoPrompter) Select(context.Context, ui.Prompt, []ui.Option) (string, error) {
	return "", nil
}

type recordingGitHubService struct {
	authCalls int
	readCalls int
}

func (s *recordingGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	return repo, nil
}
func (s *recordingGitHubService) VerifyAuth(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	s.authCalls++
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

func (s *recordingGitHubService) VerifyRunnerManagementRead(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	s.readCalls++
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}
func (s *recordingGitHubService) CreateRegistrationToken(context.Context, github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{}, nil
}
func (s *recordingGitHubService) CreateRemovalToken(context.Context, github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{}, nil
}
func (s *recordingGitHubService) ListRunners(context.Context, github.Repo) ([]github.Runner, error) {
	return nil, nil
}
func (s *recordingGitHubService) DeleteRunner(context.Context, github.Repo, int64) error { return nil }

func TestUpConfirmsDetectedRepoBeforeAuth(t *testing.T) {
	service := &recordingGitHubService{}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:       "test-version",
		Out:           &out,
		Err:           &errOut,
		TTY:           ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:       denyingRepoPrompter{},
		CommandRunner: fakeGitCommandRunner{output: "git@github.com:owner/name.git\n"},
		GitHub:        service,
	})
	cmd.SetArgs([]string{"up", "--dry-run", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected canceled repository confirmation")
	}
	if got := ExitCode(err); got != ExitCanceled {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitCanceled)
	}
	if service.authCalls != 0 {
		t.Fatalf("auth called before repository confirmation: %d", service.authCalls)
	}
	if !strings.Contains(out.String()+errOut.String(), "Choose repository: owner/name") {
		t.Fatalf("repository choice was not shown before confirmation; stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

type permissionDeniedGitHubService struct{}

func (permissionDeniedGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	repo.Private = true
	return repo, nil
}
func (permissionDeniedGitHubService) VerifyAuth(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	return github.PermissionStatus{
		OK:          false,
		Source:      github.AuthSource{Kind: "gh", Reference: "gh"},
		Required:    []string{"Administration read/write", "Metadata read"},
		Remediation: []string{github.FineGrainedTokenRemediation(repo)},
	}, nil
}

func (permissionDeniedGitHubService) VerifyRunnerManagementRead(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	return github.PermissionStatus{
		OK:          false,
		Source:      github.AuthSource{Kind: "gh", Reference: "gh"},
		Required:    []string{"Administration read/write", "Metadata read"},
		Remediation: []string{github.FineGrainedTokenRemediation(repo)},
	}, nil
}
func (permissionDeniedGitHubService) CreateRegistrationToken(context.Context, github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{}, nil
}
func (permissionDeniedGitHubService) CreateRemovalToken(context.Context, github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{}, nil
}
func (permissionDeniedGitHubService) ListRunners(context.Context, github.Repo) ([]github.Runner, error) {
	return nil, nil
}
func (permissionDeniedGitHubService) DeleteRunner(context.Context, github.Repo, int64) error {
	return nil
}

func TestUpPermissionDeniedReturnsExitCodeThreeAndJSONError(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: permissionDeniedGitHubService{}})
	cmd.SetArgs([]string{"--json", "up", "--dry-run", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected permission denial")
	}
	if got := ExitCode(err); got != ExitGitHubAuth {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitGitHubAuth)
	}
	if strings.TrimSpace(errOut.String()) != "" {
		t.Fatalf("json mode wrote stderr: %q", errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}
	errorObject := payload["error"].(map[string]any)
	if errorObject["code"] != "github_permission_denied" {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
	remediation := errorObject["remediation"].([]any)[0].(string)
	if !strings.Contains(remediation, "fine-grained token scoped only to owner/name") {
		t.Fatalf("missing selected-repo remediation: %#v", payload)
	}
}

type publicRepoGitHubService struct {
	repo github.Repo
}

func (s publicRepoGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	if s.repo.FullName == "" {
		return repo, nil
	}
	return s.repo, nil
}
func (s publicRepoGitHubService) VerifyAuth(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

func (s publicRepoGitHubService) VerifyRunnerManagementRead(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}
func (s publicRepoGitHubService) CreateRegistrationToken(context.Context, github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{}, nil
}
func (s publicRepoGitHubService) CreateRemovalToken(context.Context, github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{}, nil
}
func (s publicRepoGitHubService) ListRunners(context.Context, github.Repo) ([]github.Runner, error) {
	return nil, nil
}
func (s publicRepoGitHubService) DeleteRunner(context.Context, github.Repo, int64) error { return nil }

func TestUpPublicRepoWithoutOverrideReturnsExitCodeFour(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version: "test-version",
		Out:     &out,
		Err:     &errOut,
		GitHub: publicRepoGitHubService{repo: github.Repo{
			Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false,
		}},
	})
	cmd.SetArgs([]string{"up", "--dry-run", "--repo", "owner/name", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected public repo safety gate")
	}
	if got := ExitCode(err); got != ExitSafetyGate {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitSafetyGate)
	}
	if !strings.Contains(errOut.String(), "WARNING: Public repository risk") {
		t.Fatalf("missing public repo warning in human stderr: %q", errOut.String())
	}
}

func TestUpPublicRepoBlocksBeforeRemoteOrTokenSideEffects(t *testing.T) {
	service := publicRepoGitHubService{repo: github.Repo{Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false}}
	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: service, RemoteExecutor: remoteExec})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("expected public repo safety gate, err=%v", err)
	}
	if remoteExec.probeCalls != 0 || len(remoteExec.runs) != 0 {
		t.Fatalf("public repo block should leave host-key/preflight/bootstrap call counts at 0; probe=%d runs=%d", remoteExec.probeCalls, len(remoteExec.runs))
	}
}

func TestUpPublicRepoOverrideJSONIncludesWarning(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		RemoteExecutor: newFakeRemoteExecutor(),
		GitHub: publicRepoGitHubService{repo: github.Repo{
			Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false,
		}},
	})
	cmd.SetArgs([]string{"--json", "up", "--dry-run", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--allow-public-repo-risk", "--no-color"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("override should allow dry run: %v\nstderr=%s", err, errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}
	warnings := payload["warnings"].([]any)
	if len(warnings) == 0 || !strings.Contains(warnings[0].(string), "public_repo_risk") {
		t.Fatalf("JSON warning missing public_repo_risk: %#v", payload)
	}
}

type missingToolCommandRunner struct{}

func (missingToolCommandRunner) LookPath(string) (string, error) { return "", errors.New("not found") }
func (missingToolCommandRunner) Run(context.Context, string, ...string) (string, error) {
	return "", errors.New("not found")
}

func TestDefaultGitHubServiceMissingCredentialsFailsClosed(t *testing.T) {
	stateDir := t.TempDir()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:       "test-version",
		Out:           &out,
		Err:           &errOut,
		CommandRunner: missingToolCommandRunner{},
		GitHubEnv:     map[string]string{},
		StateBaseDir:  stateDir,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing credentials to fail closed")
	}
	if got := ExitCode(err); got != ExitGitHubAuth {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitGitHubAuth)
	}
	if strings.TrimSpace(errOut.String()) != "" {
		t.Fatalf("json mode wrote stderr: %q", errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json error output: %v\n%s", err, out.String())
	}
	errorObject := payload["error"].(map[string]any)
	if errorObject["code"] != "github_permission_denied" {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
	remediation := errorObject["remediation"].([]any)[0].(string)
	if !strings.Contains(remediation, "fine-grained token scoped only to owner/name") {
		t.Fatalf("missing selected-repo remediation: %#v", payload)
	}
	if _, err := os.Stat(state.NewStore(stateDir).Path()); !os.IsNotExist(err) {
		t.Fatalf("missing-auth default path wrote state or stat failed unexpectedly: %v", err)
	}
}

// TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr asserts that when
// bootstrap.Apply returns an error AND the bootstrap.Result.Commands
// slice contains a failing entry with non-empty Stderr, the
// bootstrap_failed CLI error message includes the redacted form of
// that stderr in its remediation slice. This proves Bug 1 surfacing
// fix from gap doc 06-GAP-byo-sudo-handling.md (Task A surface remote
// stderr requirement) AND that the redactor invariant is preserved
// (raw token never appears in user-facing output).
func TestUp_BootstrapFailed_SurfacesRedactedRemoteStderr(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	// Force the configure_runner step to fail with a stderr that
	// contains a token-shaped string. The redactor's GitHubToken
	// pattern matches `ghp_*` prefixes and replaces with the
	// `<redacted:github-token>` sentinel.
	rawToken := "ghp_secrettoken12345abc"
	remoteExec.runResults["configure_runner"] = remote.Result{
		ExitCode: 1,
		Stderr:   "configure_runner failed: cannot use token " + rawToken + " on this host",
	}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected bootstrap_failed exit")
	}
	if got := ExitCode(err); got != ExitSafetyGate {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitSafetyGate)
	}
	// Parse the JSON error payload.
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}
	errorObject, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("payload missing error object: %#v", payload)
	}
	if errorObject["code"] != "bootstrap_failed" {
		t.Fatalf("error code = %v, want bootstrap_failed: %#v", errorObject["code"], payload)
	}
	remediationAny, ok := errorObject["remediation"].([]any)
	if !ok {
		t.Fatalf("remediation missing/not a list: %#v", errorObject)
	}
	var remediation []string
	for _, r := range remediationAny {
		remediation = append(remediation, r.(string))
	}
	combined := strings.Join(remediation, "\n")
	// Surfacing assertion: at least one remediation line must contain
	// the redacted sentinel for the leaked token.
	if !strings.Contains(combined, "<redacted:github-token>") {
		t.Fatalf("expected redacted sentinel in remediation; remediation=%v", remediation)
	}
	// Redaction invariant: the raw token MUST NOT appear anywhere in
	// the captured output (json payload combined with stderr).
	combinedAll := out.String() + errOut.String()
	if strings.Contains(combinedAll, rawToken) {
		t.Fatalf("output leaked raw token %q:\n%s", rawToken, combinedAll)
	}
	// Surfacing should also reference the failing command ID so
	// users can self-diagnose.
	if !strings.Contains(combined, "configure_runner") {
		t.Fatalf("remediation does not reference failing command ID; remediation=%v", remediation)
	}
}

// recordingPasswordPrompter satisfies ui.Prompter and additionally
// records that Password() was called and what value was returned.
// Used by the Path B sudo-password fallback tests below.
type recordingPasswordPrompter struct {
	password         string
	passwordCalls    int
	confirmResponses bool
	selectResponse   string
	inputResponse    string
}

func (p *recordingPasswordPrompter) Confirm(context.Context, ui.Prompt) (bool, error) {
	return p.confirmResponses, nil
}
func (p *recordingPasswordPrompter) Select(context.Context, ui.Prompt, []ui.Option) (string, error) {
	return p.selectResponse, nil
}
func (p *recordingPasswordPrompter) Input(context.Context, ui.Prompt) (string, error) {
	return p.inputResponse, nil
}
func (p *recordingPasswordPrompter) Password(_ context.Context, _ ui.Prompt) (string, error) {
	p.passwordCalls++
	return p.password, nil
}

// passwordRequiredProbe wires the fake remote executor so that
// preflight's `sudo -n true` probe reports "password is required",
// triggering CheckPrivilegePasswordReq (Plan 06-05) which Path B
// (Plan 06-06) consumes to prompt for the sudo password.
func passwordRequiredProbe(remoteExec *fakeRemoteExecutor) {
	remoteExec.runResults["probe_sudo_n"] = remote.Result{ExitCode: 1, Stderr: "sudo: a password is required"}
}

// TestUp_SudoPasswordPrompt_Interactive asserts Path B end-to-end:
// preflight emits CheckPrivilegePasswordReq → CLI prompts via
// deps.Prompts.Password → password is registered with redact.SudoPassword
// → bootstrap.Apply sees Options.SudoPassword and wraps sudo commands
// with `sudo -S`. The literal password value MUST NOT appear in the
// captured renderer output.
func TestUp_SudoPasswordPrompt_Interactive(t *testing.T) {
	password := "correct horse battery staple"
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	passwordRequiredProbe(remoteExec)
	prompter := &recordingPasswordPrompter{password: password, confirmResponses: true}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("up returned error: %v\nstdout=%s\nstderr=%s", err, out.String(), errOut.String())
	}
	if prompter.passwordCalls == 0 {
		t.Fatalf("Prompts.Password was not called; preflight password_required path did not trigger Path B")
	}
	// At least one sudo command must have been wrapped to use sudo -S
	// + RUNNERKIT_SUDO_PASSWORD env. The fakeRemoteExecutor records the
	// rendered Command for verification.
	wrapped := false
	for _, cmd := range remoteExec.runs {
		if strings.Contains(cmd.Script, "sudo -S") && cmd.Env["RUNNERKIT_SUDO_PASSWORD"] == password {
			wrapped = true
			break
		}
	}
	if !wrapped {
		t.Fatalf("no recorded command was wrapped with sudo -S + RUNNERKIT_SUDO_PASSWORD env; recorded %d commands", len(remoteExec.runs))
	}
	// Redaction invariant: literal password must not appear in any
	// captured renderer output.
	if combined := out.String() + errOut.String(); strings.Contains(combined, password) {
		t.Fatalf("renderer output leaked raw sudo password:\n%s", combined)
	}
}

// TestUp_SudoPasswordPrompt_NonInteractive_Fails asserts that
// `--non-interactive` against a host whose preflight reports
// password_required fails with remediation pointing at
// `runnerkit byo-prepare` AND the RKD-BOOT-015 docs anchor; it MUST
// never call deps.Prompts.Password.
func TestUp_SudoPasswordPrompt_NonInteractive_Fails(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	passwordRequiredProbe(remoteExec)
	prompter := &recordingPasswordPrompter{password: "should-not-be-used"}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		TTY:            ui.TerminalCapabilities{StdinTTY: false, StdoutTTY: false, Width: 80},
		Prompts:        prompter,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--non-interactive", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected non-interactive sudo password to fail")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
	if prompter.passwordCalls != 0 {
		t.Fatalf("Prompts.Password called in non-interactive mode: %d", prompter.passwordCalls)
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "runnerkit byo-prepare") {
		t.Fatalf("error remediation missing runnerkit byo-prepare:\n%s", combined)
	}
	if !strings.Contains(combined, "RKD-BOOT-015") {
		t.Fatalf("error remediation missing RKD-BOOT-015:\n%s", combined)
	}
}

// TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive asserts that
// --yes (accept safe defaults) does NOT disable Path B prompting.
// Per gap doc constraint at lines 177-178: --yes is for safe-defaults,
// sudo password is a separate human-input concern.
func TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive(t *testing.T) {
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	passwordRequiredProbe(remoteExec)
	prompter := &recordingPasswordPrompter{password: "yespassword", confirmResponses: true}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		Prompts:        prompter,
		GitHub:         service,
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	// --yes is set but --non-interactive is NOT. Prompt MUST still fire.
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("up with --yes (interactive sudo) returned error: %v\nstderr=%s", err, errOut.String())
	}
	if prompter.passwordCalls == 0 {
		t.Fatal("Prompts.Password was not called even though --yes does NOT imply --non-interactive")
	}
}

func TestDefaultGitHubServiceUsesRealMetadataAndBlocksPublicRepo(t *testing.T) {
	var sawRegistration, sawRepository bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ghp_cli_fixturetoken12345" {
			t.Fatalf("Authorization header = %q, want bearer CLI fixture token", got)
		}
		switch r.URL.Path {
		case "/repos/owner/name/actions/runners/registration-token":
			sawRegistration = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "registration-token-cli-fixture-12345", "expires_at": "2026-04-29T03:00:00Z"})
		case "/repos/owner/name":
			sawRepository = true
			_, _ = w.Write([]byte(`{"full_name":"owner/name","private":false,"fork":false,"owner":{"login":"owner"},"name":"name"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:       "test-version",
		Out:           &out,
		Err:           &errOut,
		CommandRunner: missingToolCommandRunner{},
		// GitHubEnv: map[string]string{"RUNNERKIT_GITHUB_TOKEN":"ghp_cli_fixturetoken12345"}
		GitHubEnv:        map[string]string{"RUNNERKIT_GITHUB_TOKEN": "ghp_cli_fixturetoken12345"},
		GitHubBaseURL:    server.URL,
		GitHubHTTPClient: server.Client(),
	})
	cmd.SetArgs([]string{"up", "--repo", "owner/name", "--dry-run", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected public repo safety gate")
	}
	if got := ExitCode(err); got != ExitSafetyGate {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitSafetyGate)
	}
	if !strings.Contains(errOut.String(), "WARNING: Public repository risk") {
		t.Fatalf("missing public repo warning in stderr: %q", errOut.String())
	}
	if sawRegistration || !sawRepository {
		t.Fatalf("expected repository endpoint before safety block and no registration token request, registration=%t repository=%t", sawRegistration, sawRepository)
	}
	combinedOutput := out.String() + errOut.String()
	for _, raw := range []string{"ghp_cli_fixturetoken12345", "registration-token-cli-fixture-12345"} {
		if strings.Contains(combinedOutput, raw) {
			t.Fatalf("output leaked raw token %q:\n%s", raw, combinedOutput)
		}
	}
}
