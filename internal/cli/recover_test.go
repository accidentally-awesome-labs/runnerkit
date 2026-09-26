package cli

import (
	"os"
	"strings"
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
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
	if err == nil || ExitCode(err) != ExitInvalidInput {
		t.Fatalf("recover --reinstall-service must be refused (A-06b): ExitCode=%d err=%v", ExitCode(err), err)
	}
	if len(remoteExec.CommandIDs()) != 0 {
		t.Fatalf("refused reinstall ran remote commands: %#v", remoteExec.CommandIDs())
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

// A-06b (P1-10): --reregister and --reinstall-service are refused with
// command_disabled before any GitHub, SSH or state access, and print the
// manual re-register steps.
func TestRecoverDisabledActionsRefuseWithManualSteps(t *testing.T) {
	for _, flag := range []string{"--reregister", "--reinstall-service"} {
		for _, dryRun := range []bool{false, true} {
			stateDir := t.TempDir()
			repo := saveHealthyState(t, stateDir)
			before, err := os.ReadFile(state.NewStore(stateDir).Path())
			if err != nil {
				t.Fatalf("read state: %v", err)
			}
			github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
			remoteExec := recoveryRemote("active")
			args := []string{"recover", "--repo", repo.Repo.FullName, flag, "--yes", "--no-color"}
			if dryRun {
				args = append(args, "--dry-run")
			}
			_, errOut, err := executeStatusForTest(t, stateDir, github, remoteExec, args...)
			if err == nil || ExitCode(err) != ExitInvalidInput {
				t.Fatalf("%s dry-run=%v: ExitCode=%d err=%v", flag, dryRun, ExitCode(err), err)
			}
			for _, want := range []string{"disabled", "runnerkit down --repo owner/repo", "runnerkit up --repo owner/repo"} {
				if !strings.Contains(errOut, want) {
					t.Fatalf("%s refusal missing %q:\n%s", flag, want, errOut)
				}
			}
			if len(remoteExec.CommandIDs()) != 0 || remoteExec.ProbeHostKeyCalls != 0 {
				t.Fatalf("%s refusal touched the host: %#v", flag, remoteExec.CommandIDs())
			}
			if github.CreateRegistrationTokenCalls != 0 || github.CreateRemovalTokenCalls != 0 || github.DeleteRunnerCalls != 0 || github.ListRunnersCalls != 0 {
				t.Fatalf("%s refusal called GitHub: %#v", flag, github)
			}
			after, err := os.ReadFile(state.NewStore(stateDir).Path())
			if err != nil || string(after) != string(before) {
				t.Fatalf("%s refusal changed state.json (err=%v)", flag, err)
			}
		}
	}
}

// A-06b + A-01: when the service is missing the planner returns a blocked
// plan carrying the manual re-register steps. Applying it (no --dry-run)
// must render that reason once, as an error on stderr, and mark it rendered
// so main does not print the whole block reason a second time.
func TestRecoverBlockedPlanRendersReasonOnce(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	remoteExec := recoveryRemote("not-found")
	ui.ResetErrorRendered()
	t.Cleanup(ui.ResetErrorRendered)
	out, errOut, err := executeStatusForTest(t, stateDir, github, remoteExec, "recover", "--repo", repo.Repo.FullName, "--yes", "--no-color")
	if err == nil || ExitCode(err) != ExitSafetyGate {
		t.Fatalf("blocked recover: ExitCode=%d err=%v", ExitCode(err), err)
	}
	if n := strings.Count(errOut, "Re-register by hand"); n != 1 {
		t.Fatalf("block reason printed %d times on stderr:\n%s", n, errOut)
	}
	if strings.Contains(out, "Re-register by hand") {
		t.Fatalf("block reason must go to stderr only, stdout=%s", out)
	}
	if !ui.ErrorRendered() {
		t.Fatalf("blocked plan was not marked rendered; main would print it again")
	}
	for _, id := range remoteExec.CommandIDs() {
		if strings.HasPrefix(id, "recover.") {
			t.Fatalf("blocked plan ran a recovery command: %#v", remoteExec.CommandIDs())
		}
	}
}
