package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/runmode"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// runRegisterWithoutFoundation runs `register` against a host whose
// runnerkit-runner user is missing (the foundation probe exits 1).
func runRegisterWithoutFoundation(t *testing.T, args ...string) (string, error) {
	t.Helper()
	remoteExec := newFakeRemoteExecutor()
	remoteExec.runResults["verify_runnerkit_foundation"] = remote.Result{ExitCode: 1}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "v9.9.9",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   t.TempDir(),
		TTY:            ui.TerminalCapabilities{StdinTTY: false, StdoutTTY: false, Width: 80},
		Prompts:        ui.NewCLIPrompter(strings.NewReader(""), &errOut),
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: remoteExec,
		Sleep:          noSleep,
	})
	cmd.SetArgs(args)
	err := cmd.Execute()
	for _, run := range remoteExec.runs {
		if run.ID == "configure_runner" || run.ID == "install_service" {
			t.Fatalf("register must stop before bootstrap; ran %s", run.ID)
		}
	}
	return out.String() + errOut.String(), err
}

// lifecycle_foundation_missing used to say "re-run install.sh", which only
// writes sudoers and never creates the runner user; `up` does.
func TestRegister_FoundationMissingPointsToUp(t *testing.T) {
	output, err := runRegisterWithoutFoundation(t, "register", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("want exit %d, got %v", ExitInputRequired, err)
	}
	for _, want := range []string{"no runnerkit-runner user", "runnerkit up --repo owner/repo --host alice@example.com"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "install.sh") {
		t.Fatalf("remediation must not send users back to install.sh:\n%s", output)
	}
}

func TestRegister_FoundationMissingJSONNextAction(t *testing.T) {
	output, err := runRegisterWithoutFoundation(t, "--json", "register", "--repo", "owner/repo", "--host", "alice@example.com", "--non-interactive", "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("want exit %d, got %v", ExitInputRequired, err)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		NextActions []struct {
			Command string `json:"command"`
			Kind    string `json:"kind"`
		} `json:"next_actions"`
	}
	start := strings.Index(output, "{")
	if start < 0 {
		t.Fatalf("no JSON in output:\n%s", output)
	}
	if err := json.NewDecoder(strings.NewReader(output[start:])).Decode(&payload); err != nil {
		t.Fatalf("decode: %v\n%s", err, output)
	}
	if payload.Error.Code != "lifecycle_foundation_missing" || len(payload.NextActions) != 1 {
		t.Fatalf("unexpected payload: %+v\n%s", payload, output)
	}
	if got := payload.NextActions[0]; got.Command != "runnerkit up --repo owner/repo --host alice@example.com" || got.Kind != "run_local" {
		t.Fatalf("next action = %+v", got)
	}
}

func TestRegisterFoundationUpCommandKeepsSSHOptions(t *testing.T) {
	target := remote.Target{User: "alice", Host: "example.com", Port: 2222, Raw: "alice@example.com"}
	got := registerFoundationUpCommand("owner/repo", target, runmode.ModePersistent, &upOptions{host: "alice@example.com", sshPort: 2222, sshKey: "/home/a b/.ssh/id_ed25519"})
	want := "runnerkit up --repo owner/repo --host alice@example.com --ssh-port 2222 --ssh-key '/home/a b/.ssh/id_ed25519'"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// A host typed at the prompt is not in opts.host; the command must still
// name it.
func TestRegisterFoundationUpCommandUsesPromptedHost(t *testing.T) {
	target := remote.Target{User: "alice", Host: "example.com", Port: 2200, Raw: "alice@example.com:2200"}
	got := registerFoundationUpCommand("owner/repo", target, runmode.ModePersistent, &upOptions{sshPort: 22})
	if want := "runnerkit up --repo owner/repo --host alice@example.com:2200"; got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// Following the advice must not turn an ephemeral request into a persistent
// runner, or a dry run into a real install.
func TestRegisterFoundationUpCommandKeepsModeAndRiskFlags(t *testing.T) {
	target := remote.Target{User: "alice", Host: "example.com", Port: 22, Raw: "alice@example.com"}
	opts := &upOptions{host: "alice@example.com", sshPort: 22, allowEphemeralBYORisk: true, ephemeralTTL: 2 * time.Hour, extraPackages: "libfoo-dev", allowUnknownLinux: true, allowPublicRepoRisk: true, acceptKnownIssues: true, dryRun: true}
	got := registerFoundationUpCommand("owner/repo", target, runmode.ModeEphemeral, opts)
	want := "runnerkit up --repo owner/repo --host alice@example.com --mode ephemeral --experimental --allow-ephemeral-byo-risk --ephemeral-ttl 2h0m0s --extra-packages libfoo-dev --allow-public-repo-risk --allow-unknown-linux --accept-known-issues --dry-run"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	opts.ephemeralTTL = runmode.DefaultEphemeralTTL
	if got := registerFoundationUpCommand("owner/repo", target, runmode.ModeEphemeral, opts); strings.Contains(got, "--ephemeral-ttl") {
		t.Fatalf("default TTL must not be spelled out: %s", got)
	}
}

func TestRegister_FoundationMissingKeepsEphemeralMode(t *testing.T) {
	output, err := runRegisterWithoutFoundation(t, "--json", "register", "--repo", "owner/repo", "--host", "alice@example.com", "--mode", "ephemeral", "--experimental", "--allow-ephemeral-byo-risk", "--non-interactive", "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("want exit %d, got %v\n%s", ExitInputRequired, err, output)
	}
	if want := "runnerkit up --repo owner/repo --host alice@example.com --mode ephemeral --experimental --allow-ephemeral-byo-risk"; !strings.Contains(output, want) {
		t.Fatalf("next action must keep the ephemeral mode (%q):\n%s", want, output)
	}
}
