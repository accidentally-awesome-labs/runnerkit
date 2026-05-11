package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func recoveryRemote(activeState string) *testsupport.RemoteExecutor {
	loadState := "loaded"
	if activeState == "not-found" {
		loadState = "not-found"
	}
	return &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			ops.CommandStatusSSHReachable: {ExitCode: 0},
			ops.CommandStatusSystemdShow:  {Stdout: "LoadState=" + loadState + "\nActiveState=" + activeState + "\nSubState=" + activeState + "\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
			"recover.service.restart":     {ExitCode: 0},
			"recover.service.reinstall":   {ExitCode: 0},
			"recover.service.verify":      {ExitCode: 0},
			"recover.service.stop":        {ExitCode: 0},
			"recover.service.uninstall":   {ExitCode: 0},
			"recover.runner.remove":       {ExitCode: 0},
			"recover.runner.configure":    {ExitCode: 0},
			"recover.runner.start":        {ExitCode: 0},
		},
	}
}

func commandIDsContain(exec *testsupport.RemoteExecutor, want string) bool {
	for _, id := range exec.CommandIDs() {
		if id == want {
			return true
		}
	}
	return false
}

func TestRecoverDryRunRestartReinstallMissingYesAndHostKeyBlock(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	remoteExec := recoveryRemote("failed")
	out, _, err := executeStatusForTest(t, stateDir, github, remoteExec, "recover", "--repo", repo.Repo.FullName, "--dry-run", "--no-color")
	if err != nil {
		t.Fatalf("recover dry-run returned error: %v", err)
	}
	if !strings.Contains(out, "Step 1 of 1: recovery plan") || !strings.Contains(out, "Restart systemd service actions.runner.runnerkit-owner-repo-local.service") {
		t.Fatalf("dry-run missing recovery plan:\n%s", out)
	}
	if commandIDsContain(remoteExec, "recover.service.restart") {
		t.Fatalf("dry-run ran restart command: %#v", remoteExec.CommandIDs())
	}

	remoteExec = recoveryRemote("failed")
	_, _, err = executeStatusForTest(t, stateDir, github, remoteExec, "recover", "--repo", repo.Repo.FullName, "--restart-service", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("recover restart returned error: %v", err)
	}
	if !commandIDsContain(remoteExec, "recover.service.restart") || !commandIDsContain(remoteExec, "recover.service.verify") {
		t.Fatalf("restart did not run expected commands: %#v", remoteExec.CommandIDs())
	}
	if github.CreateRegistrationTokenCalls != 0 || github.CreateRemovalTokenCalls != 0 || github.DeleteRunnerCalls != 0 {
		t.Fatalf("service restart should not create/delete tokens: %#v", github)
	}

	remoteExec = recoveryRemote("not-found")
	_, _, err = executeStatusForTest(t, stateDir, github, remoteExec, "recover", "--repo", repo.Repo.FullName, "--reinstall-service", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("recover reinstall returned error: %v", err)
	}
	if !commandIDsContain(remoteExec, "recover.service.reinstall") || !commandIDsContain(remoteExec, "recover.service.verify") {
		t.Fatalf("reinstall did not run expected commands: %#v", remoteExec.CommandIDs())
	}

	_, _, err = executeStatusForTest(t, stateDir, github, recoveryRemote("failed"), "recover", "--repo", repo.Repo.FullName, "--restart-service", "--no-color")
	if err == nil || ExitCode(err) != ExitInputRequired {
		t.Fatalf("missing --yes ExitCode=%d err=%v", ExitCode(err), err)
	}

	blockedRemote := &testsupport.RemoteExecutor{ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:changed"}}
	_, _, err = executeStatusForTest(t, stateDir, github, blockedRemote, "recover", "--repo", repo.Repo.FullName, "--restart-service", "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("host-key mismatch ExitCode=%d err=%v", ExitCode(err), err)
	}
	if commandIDsContain(blockedRemote, "recover.service.restart") {
		t.Fatalf("host-key mismatch ran recovery command: %#v", blockedRemote.CommandIDs())
	}
}

func TestRecoverReregisterUpdatesGitHubRunnerID(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	removalToken := strings.Join([]string{"removal", "token", "recover", "secret"}, "-")
	registrationToken := strings.Join([]string{"registration", "token", "recover", "secret"}, "-")
	github := &testsupport.GitHubService{RemovalToken: gh.RunnerToken{Token: removalToken, ExpiresAt: time.Now().Add(time.Hour)}, RegistrationToken: gh.RunnerToken{Token: registrationToken, ExpiresAt: time.Now().Add(time.Hour)}, Runners: []gh.Runner{{ID: 456, Name: repo.Runner.Name, Status: "online", Labels: repo.Runner.Labels}}}
	remoteExec := recoveryRemote("active")
	out, errOut, err := executeStatusForTest(t, stateDir, github, remoteExec, "--json", "recover", "--repo", repo.Repo.FullName, "--reregister", "--yes", "--no-color")
	if err != nil {
		t.Fatalf("recover reregister returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{"recover.service.stop", "recover.service.uninstall", "recover.runner.remove", "recover.runner.configure", "recover.runner.start"} {
		if !commandIDsContain(remoteExec, want) {
			t.Fatalf("reregister missing command %q in %#v", want, remoteExec.CommandIDs())
		}
	}
	if !strings.Contains(out, `"github_runner_id":456`) || !strings.Contains(out, `"state_updated":true`) {
		t.Fatalf("recover json missing updated runner ID:\n%s", out)
	}
	stateBytes, err := os.ReadFile(state.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, raw := range []string{removalToken, registrationToken} {
		if strings.Contains(out, raw) || strings.Contains(errOut, raw) || strings.Contains(string(stateBytes), raw) {
			t.Fatalf("token leaked: %q\nstdout=%s\nstderr=%s\nstate=%s", raw, out, errOut, stateBytes)
		}
	}
	var persisted state.State
	if err := json.Unmarshal(stateBytes, &persisted); err != nil {
		t.Fatalf("state json invalid: %v", err)
	}
	updated := persisted.Repositories[0]
	if updated.Cleanup.GitHubRunnerID != 456 || len(updated.Operations) == 0 || updated.Operations[0].Artifact != "github_runner_id" {
		t.Fatalf("state not updated with checkpoint: %#v", updated)
	}
}
