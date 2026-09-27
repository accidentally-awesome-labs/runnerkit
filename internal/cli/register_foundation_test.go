package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
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
	got := registerFoundationUpCommand("owner/repo", &upOptions{host: "alice@example.com", sshPort: 2222, sshKey: "/home/a b/.ssh/id_ed25519"})
	want := "runnerkit up --repo owner/repo --host alice@example.com --ssh-port 2222 --ssh-key '/home/a b/.ssh/id_ed25519'"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
