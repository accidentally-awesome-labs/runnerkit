package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

func executeWithStateDir(t *testing.T, stateDir string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: newFakePermittedGitHubService(), RemoteExecutor: newFakeRemoteExecutor(), Sleep: noSleep})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	return out.String(), errOut.String(), runErr
}

func TestUpDryRunDoesNotCreateStateFile(t *testing.T) {
	stateDir := t.TempDir()
	out, errOut, err := executeWithStateDir(t, stateDir, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--dry-run", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("dry-run returned error: %v\nstderr=%s", err, errOut)
	}
	if !strings.Contains(out, "runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]") {
		t.Fatalf("dry-run missing recommended runs-on snippet:\n%s", out)
	}
	if _, err := os.Stat(state.NewStore(stateDir).Path()); !os.IsNotExist(err) {
		t.Fatalf("dry-run created state file or stat failed unexpectedly: %v", err)
	}
}

func TestUpSaveJSONAndStateShowJSONAreRedacted(t *testing.T) {
	stateDir := t.TempDir()
	out, errOut, err := executeWithStateDir(t, stateDir, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("up save returned error: %v\nstderr=%s", err, errOut)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("up output is not JSON: %v\n%s", err, out)
	}
	if payload["runner_installed"] != true || payload["repo"] != "owner/repo" || payload["redactions_applied"] != true {
		t.Fatalf("unexpected up JSON payload: %#v", payload)
	}
	if _, ok := payload["state_path"].(string); !ok {
		t.Fatalf("up JSON missing state_path: %#v", payload)
	}
	if _, err := os.Stat(state.NewStore(stateDir).Path()); err != nil {
		t.Fatalf("state file not saved: %v", err)
	}

	showOut, showErr, err := executeWithStateDir(t, stateDir, "--json", "state", "show", "--repo", "owner/repo", "--no-color")
	if err != nil {
		t.Fatalf("state show returned error: %v\nstderr=%s", err, showErr)
	}
	for _, forbidden := range []string{"token", "registration_token", "remove_token", "private_key", "provider_credential"} {
		if strings.Contains(showOut, forbidden) {
			t.Fatalf("state show leaked forbidden token text %q:\n%s", forbidden, showOut)
		}
	}
	if !strings.Contains(showOut, `"redactions_applied":true`) {
		t.Fatalf("state show JSON missing redactions flag:\n%s", showOut)
	}
}

func TestUpBYOHappyPathSmoke(t *testing.T) {
	const fakeSmokeToken = "registration-token-secret-12345"
	const fakeSensitiveRemoteOutput = "ssh-private-key-secret"
	stateDir := t.TempDir()
	service := newFakePermittedGitHubService()
	remoteExec := newFakeRemoteExecutor()
	remoteExec.probe.HostKey.Fingerprint = "SHA256:fakehostfingerprint"
	remoteExec.runResults["configure_runner"] = remote.Result{Stdout: fakeSmokeToken + " " + fakeSensitiveRemoteOutput, Stderr: fakeSensitiveRemoteOutput, ExitCode: 0}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: service, RemoteExecutor: remoteExec, Sleep: noSleep})
	cmd.SetArgs([]string{"up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("up happy path returned error: %v\nstderr=%s", err, errOut.String())
	}
	stdout := out.String()
	for _, want := range []string{"BYO runner ready", "runnerkit-owner-repo-local", "alice@example.com:22", "runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]", "Add the runs-on snippet above"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, forbidden := range []string{fakeSmokeToken, fakeSensitiveRemoteOutput} {
		if strings.Contains(stdout, forbidden) || strings.Contains(errOut.String(), forbidden) || strings.Contains(string(stateBytes), forbidden) {
			t.Fatalf("secret %q leaked\nstdout=%s\nstderr=%s\nstate=%s", forbidden, stdout, errOut.String(), stateBytes)
		}
	}
	for _, want := range []string{"\"github_runner_id\": 123", "\"host_key_fingerprint\": \"SHA256:fakehostfingerprint\"", "\"service_name\""} {
		if !strings.Contains(string(stateBytes), want) {
			t.Fatalf("state missing %q:\n%s", want, stateBytes)
		}
	}

	jsonStateDir := t.TempDir()
	jsonOut, jsonErr, err := executeWithStateDir(t, jsonStateDir, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("json smoke returned error: %v\nstderr=%s", err, jsonErr)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("json smoke output invalid: %v\n%s", err, jsonOut)
	}
	if payload["runner_installed"] != true || payload["redactions_applied"] != true {
		t.Fatalf("unexpected json smoke payload: %#v", payload)
	}
}

func TestUpExistingStateRequiresReplaceConfirmationOrFlag(t *testing.T) {
	stateDir := t.TempDir()
	if _, errOut, err := executeWithStateDir(t, stateDir, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color"); err != nil {
		t.Fatalf("initial save failed: %v\nstderr=%s", err, errOut)
	}
	out, _, err := executeWithStateDir(t, stateDir, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--no-color")
	if err == nil {
		t.Fatal("expected existing state to require --replace")
	}
	if got := ExitCode(err); got != ExitInputRequired {
		t.Fatalf("ExitCode() = %d, want %d", got, ExitInputRequired)
	}
	if !strings.Contains(out, "--replace") && !strings.Contains(out, "replace owner/repo") {
		t.Fatalf("existing-state remediation missing replacement instructions:\n%s", out)
	}
	if _, errOut, err := executeWithStateDir(t, stateDir, "--json", "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--replace", "--no-color"); err != nil {
		t.Fatalf("--replace save failed: %v\nstderr=%s", err, errOut)
	}
}
