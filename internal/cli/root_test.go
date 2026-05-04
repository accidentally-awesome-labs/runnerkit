package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/labels"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

func executeForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, err bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &err,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Sleep:          noSleep,
	})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), err.String(), runErr
}

type fakePermittedGitHubService struct {
	listCalls  int
	authCalls  int
	readCalls  int
	tokenCalls int
	runners    []github.Runner
}

func newFakePermittedGitHubService() *fakePermittedGitHubService {
	return &fakePermittedGitHubService{}
}

func (s *fakePermittedGitHubService) Repository(_ context.Context, repo github.Repo) (github.Repo, error) {
	if repo.Host == "" {
		repo.Host = "github.com"
	}
	repo.Private = true
	return repo, nil
}

func (s *fakePermittedGitHubService) VerifyAuth(_ context.Context, _ github.Repo) (github.PermissionStatus, error) {
	s.authCalls++
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

func (s *fakePermittedGitHubService) VerifyRunnerManagementRead(_ context.Context, _ github.Repo) (github.PermissionStatus, error) {
	s.readCalls++
	return github.PermissionStatus{OK: true, Source: github.AuthSource{Kind: "gh", Reference: "gh"}}, nil
}

func (s *fakePermittedGitHubService) CreateRegistrationToken(_ context.Context, _ github.Repo) (github.RunnerToken, error) {
	s.tokenCalls++
	return github.RunnerToken{Token: fakeRegistrationToken(), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (s *fakePermittedGitHubService) CreateRemovalToken(_ context.Context, _ github.Repo) (github.RunnerToken, error) {
	return github.RunnerToken{Token: strings.Join([]string{"remove-token", "secret-12345"}, "-"), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (s *fakePermittedGitHubService) ListRunners(_ context.Context, repo github.Repo) ([]github.Runner, error) {
	s.listCalls++
	if s.runners != nil {
		return s.runners, nil
	}
	if s.listCalls == 1 {
		return nil, nil
	}
	labelSet := labels.Build(repo, labels.Options{OS: labels.DefaultOS, Arch: labels.DefaultArch, Mode: labels.DefaultMode})
	return []github.Runner{{ID: 123, Name: labelSet.RunnerName, OS: "linux", Status: "online", Labels: append([]string(nil), labelSet.Labels...)}}, nil
}

func (s *fakePermittedGitHubService) DeleteRunner(context.Context, github.Repo, int64) error {
	return nil
}

func fakeRegistrationToken() string {
	return strings.Join([]string{"registration-token", "secret-12345"}, "-")
}

type fakeRemoteExecutor struct {
	probe      remote.ProbeResult
	probeCalls int
	runs       []remote.Command
	runResults map[string]remote.Result
	runErrs    map[string]error
}

func newFakeRemoteExecutor() *fakeRemoteExecutor {
	return &fakeRemoteExecutor{probe: passingProbe(), runResults: map[string]remote.Result{}, runErrs: map[string]error{}}
}

func passingProbe() remote.ProbeResult {
	commands := map[string]bool{"sudo": true, "curl": true, "tar": true, "gzip": true, "sha256sum": true, "id": true, "useradd": true, "install": true, "timedatectl": true}
	return remote.ProbeResult{HostKey: remote.HostKey{Algorithm: "ssh-ed25519", Fingerprint: "SHA256:fakehostfingerprint"}, OSRelease: map[string]string{"ID": "ubuntu"}, Kernel: "linux", Arch: "x86_64", Systemd: true, Commands: commands, DiskAvailableBytes: 2147483648, TimeSynchronized: true}
}

func (f *fakeRemoteExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	f.probeCalls++
	return f.probe, nil
}

func (f *fakeRemoteExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	f.runs = append(f.runs, command)
	if f.runResults != nil {
		if result, ok := f.runResults[command.ID]; ok {
			return result, f.runErrs[command.ID]
		}
	}
	return remote.Result{Stdout: "ok", ExitCode: 0}, nil
}

func noSleep(context.Context, time.Duration) error { return nil }

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
