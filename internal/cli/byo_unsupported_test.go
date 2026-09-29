package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
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
	return runWithBYOGateOn(t, newFakeRemoteExecutor(), args...)
}

func runWithBYOGateOn(t *testing.T, remoteExec *fakeRemoteExecutor, args ...string) byoGateRun {
	t.Helper()
	run := byoGateRun{remote: remoteExec, github: newFakePermittedGitHubService(), cloud: &provider.FakeProvider{}}
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

// A register that got past the refusal with --accept-known-issues and then
// finds no runnerkit-runner user must print an up command that is not
// refused in turn.
func TestRegisterBYO_FoundationHintKeepsAcceptKnownIssues(t *testing.T) {
	remoteExec := newFakeRemoteExecutor()
	remoteExec.runResults["verify_runnerkit_foundation"] = remote.Result{ExitCode: 1}
	run := runWithBYOGateOn(t, remoteExec, "--json", "register", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--accept-known-issues", "--no-color")
	if run.err == nil || ExitCode(run.err) != ExitInputRequired {
		t.Fatalf("want lifecycle_foundation_missing (exit %d), got %v\n%s", ExitInputRequired, run.err, run.output)
	}
	if want := "runnerkit up --repo owner/repo --host alice@example.com --accept-known-issues"; !strings.Contains(run.output, want) {
		t.Fatalf("next action must keep --accept-known-issues (%q):\n%s", want, run.output)
	}
}

func TestInit_WarnsBYOUnsupportedBeforeHostInstall(t *testing.T) {
	human := runWithBYOGate(t, "init", "--no-color")
	warn := strings.Index(human.output, "BYO setup is not supported in this release")
	install := strings.Index(human.output, "SSH to the Linux runner machine")
	if human.err != nil || warn < 0 || install < 0 || warn > install {
		t.Fatalf("init must warn before the install step (err %v):\n%s", human.err, human.output)
	}

	// --print-install-command keeps stdout to the install line; the
	// warning goes to stderr.
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: t.TempDir(), BYOUnsupportedRelease: true})
	cmd.SetArgs([]string{"init", "--print-install-command", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(out.String()), "\n"); len(lines) != 1 || !strings.Contains(lines[0], "install.sh") {
		t.Fatalf("stdout must be only the install line:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "--accept-known-issues") {
		t.Fatalf("stderr must carry the warning:\n%s", errOut.String())
	}

	jsonRun := runWithBYOGate(t, "--json", "init", "--print-install-command", "--no-color")
	var payload struct {
		NextActions []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
		} `json:"next_actions"`
	}
	start := strings.Index(jsonRun.output, "{")
	if start < 0 || json.NewDecoder(strings.NewReader(jsonRun.output[start:])).Decode(&payload) != nil {
		t.Fatalf("no JSON:\n%s", jsonRun.output)
	}
	if len(payload.NextActions) != 2 || payload.NextActions[0].ID != byoUnsupportedReleaseCode || payload.NextActions[0].Severity != "warning" || payload.NextActions[1].ID != "host_install" {
		t.Fatalf("next actions = %+v", payload.NextActions)
	}
}

// install.sh is fetched from the release tag, so its success message is
// the last thing a user reads before running up.
func TestInstallShSaysBYOUnsupported(t *testing.T) {
	body, err := os.ReadFile("../../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "BYO setup is not supported in RunnerKit v1.3.4: up and register need --accept-known-issues") {
		t.Fatal("install.sh's success message must say BYO is unsupported in this release")
	}
}
