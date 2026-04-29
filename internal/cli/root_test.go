package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/github"
)

func executeForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, err bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version: "test-version",
		Out:     &out,
		Err:     &err,
		GitHub:  fakePermittedGitHubService{},
	})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), err.String(), runErr
}

type fakePermittedGitHubService struct{}

func (fakePermittedGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	if repo.Host == "" {
		repo.Host = "github.com"
	}
	repo.Private = true
	return repo, nil
}

func (fakePermittedGitHubService) VerifyAuth(_ context.Context, _ github.Repo) (github.PermissionStatus, error) {
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

func TestRootHelpListsRunnerKitAndUp(t *testing.T) {
	out, _, err := executeForTest(t, "--help")
	if err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(out, "RunnerKit") {
		t.Fatalf("help missing RunnerKit: %q", out)
	}
	if !strings.Contains(out, "up") {
		t.Fatalf("help missing up command: %q", out)
	}
}

func TestVersionJSONContract(t *testing.T) {
	out, _, err := executeForTest(t, "--json", "version")
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("json output contains ansi: %q", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("version output is not json: %v\n%s", err, out)
	}
	if payload["ok"] != true || payload["command"] != "version" || payload["version"] != "test-version" || payload["redactions_applied"] != true {
		t.Fatalf("unexpected version payload: %#v", payload)
	}
}

func TestInvalidFlagMapsToExitCodeTwo(t *testing.T) {
	_, _, err := executeForTest(t, "--definitely-not-a-flag")
	if err == nil {
		t.Fatal("expected invalid flag error")
	}
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInvalidInput)
	}
}

func TestNormalizeDependenciesDefaultsToRealGitHubAndOSCommandRunner(t *testing.T) {
	deps := normalizeDependencies(Dependencies{})
	if _, ok := deps.CommandRunner.(github.OSCommandRunner); !ok {
		t.Fatalf("default CommandRunner = %T, want github.OSCommandRunner", deps.CommandRunner)
	}
	if _, ok := deps.GitHub.(*github.Service); !ok {
		t.Fatalf("default GitHub = %T, want *github.Service", deps.GitHub)
	}
}
