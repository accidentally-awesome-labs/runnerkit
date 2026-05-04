package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

// seedRepoState writes a single repository state to a fresh state dir and
// returns the dir path. The caller passes the resulting dir as
// StateBaseDir to the CLI under test.
func seedRepoState(t *testing.T, repo rkstate.RepositoryState) string {
	t.Helper()
	dir := t.TempDir()
	store := rkstate.NewStore(dir)
	state := rkstate.State{SchemaVersion: rkstate.SchemaVersion, Repositories: []rkstate.RepositoryState{repo}}
	if err := store.Save(state); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	return dir
}

// TestUpgradeRunner_Persistent_ReAppliesWithNewPin: a persistent BYO
// fixture with stale RunnerTemplateVersion is upgraded; the fake remote
// executor records bootstrap commands whose download_runner script
// contains the bundled pin. State is updated to the bundled pin only on
// success.
func TestUpgradeRunner_Persistent_ReAppliesWithNewPin(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	repo.RunnerTemplateVersion = "2.330.0"
	stateDir := seedRepoState(t, repo)

	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: remoteExec,
		StateBaseDir:   stateDir,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("upgrade-runner returned error: %v\nstderr=%s", err, errOut.String())
	}

	// Apply was called: at least the persistent bootstrap commands flowed.
	wanted := map[string]bool{
		"fix_dependencies":  false,
		"create_runner_user": false,
		"download_runner":   false,
		"configure_runner":  false,
		"install_service":   false,
		"verify_service":    false,
	}
	for _, c := range remoteExec.runs {
		if _, ok := wanted[c.ID]; ok {
			wanted[c.ID] = true
		}
	}
	for id, seen := range wanted {
		if !seen {
			t.Fatalf("expected persistent bootstrap.Apply command %q to be invoked", id)
		}
	}
	// download_runner must reference the bundled pin.
	var downloadScript string
	for _, c := range remoteExec.runs {
		if c.ID == "download_runner" {
			downloadScript = c.Script
			break
		}
	}
	if !strings.Contains(downloadScript, bootstrap.RunnerVersion) {
		t.Fatalf("download_runner script did not reference bundled pin %q; script:\n%s", bootstrap.RunnerVersion, downloadScript)
	}

	// State was updated to the bundled pin only after Apply succeeded.
	store := rkstate.NewStore(stateDir)
	loaded, ok, err := store.GetRepository(testsupport.TestRepoFullName)
	if err != nil || !ok {
		t.Fatalf("GetRepository err=%v ok=%v", err, ok)
	}
	if loaded.RunnerTemplateVersion != bootstrap.RunnerVersion {
		t.Fatalf("RunnerTemplateVersion = %q after upgrade; want %q", loaded.RunnerTemplateVersion, bootstrap.RunnerVersion)
	}
}

// TestUpgradeRunner_Ephemeral_TerminalNoOp: ephemeral runner that has
// already terminated (FinalizerStatus completed) results in no Apply call
// and a clear no-op message. Exit code 0.
func TestUpgradeRunner_Ephemeral_TerminalNoOp(t *testing.T) {
	repo := testsupport.EphemeralBYORepositoryState()
	repo.Ephemeral.FinalizerStatus = "completed"
	stateDir := seedRepoState(t, repo)

	remoteExec := newFakeRemoteExecutor()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: remoteExec,
		StateBaseDir:   stateDir,
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("upgrade-runner terminal ephemeral returned error: %v\nstderr=%s", err, errOut.String())
	}
	if len(remoteExec.runs) != 0 {
		var ids []string
		for _, c := range remoteExec.runs {
			ids = append(ids, c.ID)
		}
		t.Fatalf("expected no Apply commands for terminated ephemeral; got %v", ids)
	}
	if !strings.Contains(out.String(), "Ephemeral runner is one-shot and already terminated") {
		t.Fatalf("expected terminal-ephemeral notice in stdout; got %q", out.String())
	}
}

// TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce: ephemeral runner
// in waiting state refuses without --force; with --force, ApplyEphemeral
// is invoked.
func TestUpgradeRunner_Ephemeral_WaitingRefusesWithoutForce(t *testing.T) {
	repo := testsupport.EphemeralBYORepositoryState()
	repo.Ephemeral.FinalizerStatus = "waiting"

	// Without --force: refuses.
	{
		stateDir := seedRepoState(t, repo)
		remoteExec := newFakeRemoteExecutor()
		var out, errOut bytes.Buffer
		cmd := NewRootCommand(Dependencies{
			Version:        "test-version",
			Out:            &out,
			Err:            &errOut,
			GitHub:         newFakePermittedGitHubService(),
			RemoteExecutor: remoteExec,
			StateBaseDir:   stateDir,
			Sleep:          noSleep,
		})
		cmd.SetArgs([]string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes", "--no-color"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected refuse-without-force error, got nil")
		}
		if got := ExitCode(err); got != ExitInvalidInput {
			t.Fatalf("ExitCode = %d, want ExitInvalidInput=%d", got, ExitInvalidInput)
		}
		if len(remoteExec.runs) != 0 {
			t.Fatalf("expected no remote calls when refusing waiting ephemeral; got %d", len(remoteExec.runs))
		}
	}

	// With --force: ApplyEphemeral is invoked.
	{
		stateDir := seedRepoState(t, repo)
		remoteExec := newFakeRemoteExecutor()
		var out, errOut bytes.Buffer
		cmd := NewRootCommand(Dependencies{
			Version:        "test-version",
			Out:            &out,
			Err:            &errOut,
			GitHub:         newFakePermittedGitHubService(),
			RemoteExecutor: remoteExec,
			StateBaseDir:   stateDir,
			Sleep:          noSleep,
		})
		cmd.SetArgs([]string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes", "--force", "--no-color"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("--force should succeed: %v\nstderr=%s", err, errOut.String())
		}
		// ApplyEphemeral has install_ephemeral_service among its commands.
		seen := false
		for _, c := range remoteExec.runs {
			if c.ID == "install_ephemeral_service" || c.ID == "verify_ephemeral_service" {
				seen = true
				break
			}
		}
		if !seen {
			var ids []string
			for _, c := range remoteExec.runs {
				ids = append(ids, c.ID)
			}
			t.Fatalf("expected ephemeral bootstrap commands when --force; got %v", ids)
		}
	}
}
