package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
)

// A-21: a release that ships without a passing real-job BYO gate refuses
// BYO `up` and `register` unless --accept-known-issues is passed. The
// release binary turns the refusal on (cmd/runnerkit/main.go); these tests
// turn it on through Dependencies.

type byoGateRun struct {
	remote *fakeRemoteExecutor
	github *fakePermittedGitHubService
	cloud  *provider.FakeProvider
	output string
	err    error
}

func runWithBYOGate(t *testing.T, args ...string) byoGateRun {
	t.Helper()
	run := byoGateRun{remote: newFakeRemoteExecutor(), github: newFakePermittedGitHubService(), cloud: &provider.FakeProvider{}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:               "test-version",
		Out:                   &out,
		Err:                   &errOut,
		StateBaseDir:          t.TempDir(),
		GitHub:                run.github,
		RemoteExecutor:        run.remote,
		Providers:             provider.NewRegistry(run.cloud),
		Sleep:                 noSleep,
		BYOUnsupportedRelease: true,
	})
	cmd.SetArgs(args)
	run.err = cmd.Execute()
	run.output = out.String() + errOut.String()
	return run
}

// assertRefusedBeforeAuthAndSSH: the refusal comes before VerifyAuth (which
// creates a registration token) and before any SSH probe or command.
func assertRefusedBeforeAuthAndSSH(t *testing.T, run byoGateRun) {
	t.Helper()
	if run.err == nil || ExitCode(run.err) != ExitInvalidInput {
		t.Fatalf("want exit %d, got %v\n%s", ExitInvalidInput, run.err, run.output)
	}
	if run.remote.probeCalls != 0 || len(run.remote.runs) != 0 {
		t.Fatalf("SSH used before the refusal: probe=%d runs=%d", run.remote.probeCalls, len(run.remote.runs))
	}
	if run.github.authCalls != 0 || run.github.tokenCalls != 0 {
		t.Fatalf("GitHub auth or token call before the refusal: auth=%d token=%d", run.github.authCalls, run.github.tokenCalls)
	}
}

func TestUpBYO_RefusesWithoutAcceptKnownIssues(t *testing.T) {
	run := runWithBYOGate(t, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	assertRefusedBeforeAuthAndSSH(t, run)
	for _, want := range []string{"BYO setup is not supported in this release", "--accept-known-issues", "#known-issues"} {
		if !strings.Contains(run.output, want) {
			t.Fatalf("output missing %q:\n%s", want, run.output)
		}
	}
}

func TestRegisterBYO_RefusesWithoutAcceptKnownIssues(t *testing.T) {
	run := runWithBYOGate(t, "register", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	assertRefusedBeforeAuthAndSSH(t, run)
	if !strings.Contains(run.output, "--accept-known-issues") {
		t.Fatalf("output must name --accept-known-issues:\n%s", run.output)
	}
}

// A dry run also connects over SSH for preflight, so it is refused too.
func TestUpBYO_DryRunRefusedWithoutAcceptKnownIssues(t *testing.T) {
	assertRefusedBeforeAuthAndSSH(t, runWithBYOGate(t, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--dry-run", "--no-color"))
}

func TestUpBYO_RefusalJSONCode(t *testing.T) {
	run := runWithBYOGate(t, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	assertRefusedBeforeAuthAndSSH(t, run)
	var payload struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	start := strings.Index(run.output, "{")
	if start < 0 {
		t.Fatalf("no JSON in output:\n%s", run.output)
	}
	if err := json.NewDecoder(strings.NewReader(run.output[start:])).Decode(&payload); err != nil {
		t.Fatalf("decode: %v\n%s", err, run.output)
	}
	if payload.OK || payload.Error.Code != byoUnsupportedReleaseCode {
		t.Fatalf("want error code %s, got %+v\n%s", byoUnsupportedReleaseCode, payload, run.output)
	}
}

func TestUpBYO_AcceptKnownIssuesProceeds(t *testing.T) {
	run := runWithBYOGate(t, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--accept-known-issues", "--no-color")
	if run.err != nil {
		t.Fatalf("up --accept-known-issues returned error: %v\n%s", run.err, run.output)
	}
	if run.github.tokenCalls == 0 || len(run.remote.runs) == 0 {
		t.Fatalf("setup did not run: token=%d runs=%d", run.github.tokenCalls, len(run.remote.runs))
	}
}

// The refusal is BYO-only: the experimental cloud path keeps its own gates.
func TestUpCloud_NotRefusedByBYOGate(t *testing.T) {
	run := runWithBYOGate(t, "up", "--repo", "owner/repo", "--cloud", "hetzner", "--experimental", "--cloud-region", "fsn1", "--yes", "--dry-run", "--no-color")
	if run.err != nil {
		t.Fatalf("cloud dry-run returned error: %v\n%s", run.err, run.output)
	}
	if strings.Contains(run.output, byoUnsupportedReleaseCode) || run.cloud.PlanCalls != 1 {
		t.Fatalf("cloud dry-run hit the BYO refusal (plan calls %d):\n%s", run.cloud.PlanCalls, run.output)
	}
}
