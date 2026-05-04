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
