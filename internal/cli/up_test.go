package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ui"
)

func TestUpDryRunDisplaysPhaseOneWizard(t *testing.T) {
	out, errOut, err := executeForTest(t, "up", "--dry-run", "--repo", "owner/name", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up dry-run returned error: %v\nstderr: %s", err, errOut)
	}
	for _, step := range []string{"Welcome", "Prerequisites", "Repo/auth", "Safety checks", "State preview", "Next steps"} {
		if !strings.Contains(out, step) {
			t.Fatalf("dry-run output missing step %q:\n%s", step, out)
		}
	}
	for _, copy := range []string{"Phase 1 does not install a runner yet", "Will not install a runner in Phase 1"} {
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
	out, errOut, err := executeForTest(t, "--json", "up", "--dry-run", "--repo", "owner/name", "--yes", "--no-color")
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
}

func (s *recordingGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	return repo, nil
}
func (s *recordingGitHubService) VerifyAuth(_ context.Context, repo github.Repo) (github.PermissionStatus, error) {
	s.authCalls++
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

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

func TestUpPermissionDeniedReturnsExitCodeThreeAndJSONError(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: permissionDeniedGitHubService{}})
	cmd.SetArgs([]string{"--json", "up", "--dry-run", "--repo", "owner/name", "--yes", "--no-color"})
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

func TestUpPublicRepoOverrideJSONIncludesWarning(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version: "test-version",
		Out:     &out,
		Err:     &errOut,
		GitHub: publicRepoGitHubService{repo: github.Repo{
			Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false,
		}},
	})
	cmd.SetArgs([]string{"--json", "up", "--dry-run", "--repo", "owner/name", "--yes", "--allow-public-repo-risk", "--no-color"})
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
