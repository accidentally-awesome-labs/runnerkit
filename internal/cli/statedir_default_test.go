package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/checkliststore"
)

// A-03 (P1-3): StateBaseDir must default to the real state directory so
// config.json and sessions/ never land in the process CWD.

func TestNormalizeDependencies_DefaultsStateBaseDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RUNNERKIT_STATE_DIR", dir)
	deps := normalizeDependencies(Dependencies{})
	if deps.StateBaseDir != dir {
		t.Fatalf("StateBaseDir = %q, want %q", deps.StateBaseDir, dir)
	}
	injected := t.TempDir()
	if got := normalizeDependencies(Dependencies{StateBaseDir: injected}).StateBaseDir; got != injected {
		t.Fatalf("injected StateBaseDir overridden: %q", got)
	}
}

func TestDefaultStateBaseDir_NoWritesIntoCWD(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("RUNNERKIT_STATE_DIR", stateDir)
	cwd := t.TempDir()
	chdirForTest(t, cwd)

	deps := normalizeDependencies(Dependencies{})
	if err := SaveUserConfig(deps.StateBaseDir, UserConfig{DoctorIgnoreFindingIDs: []string{"host_mem_low"}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	target := remote.Target{User: "alice", Host: "example.com"}
	if err := syncBYOInstallChecklist(deps.StateBaseDir, "owner/repo", target, 1); err != nil {
		t.Fatalf("save checklist: %v", err)
	}
	var out bytes.Buffer
	writeBYOChecklistHuman(&out, deps.TTY, deps.StateBaseDir, "owner/repo", target, 2)

	// Callers that bypass normalizeDependencies and pass "" must still
	// resolve to the state dir, never the CWD.
	if err := SaveUserConfig("", UserConfig{}); err != nil {
		t.Fatalf("save config with empty base: %v", err)
	}
	if err := checkliststore.Save("", &checkliststore.Doc{SessionID: "x"}); err != nil {
		t.Fatalf("save checklist with empty base: %v", err)
	}

	assertDirEmpty(t, cwd)
	if _, err := os.Stat(filepath.Join(stateDir, "config.json")); err != nil {
		t.Fatalf("config.json not in state dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "sessions")); err != nil {
		t.Fatalf("sessions/ not in state dir: %v", err)
	}
}

func TestDoctorIgnore_PersistsToStateDirNotCWD(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("RUNNERKIT_STATE_DIR", stateDir)
	cwd := t.TempDir()
	chdirForTest(t, cwd)
	repo := saveHealthyState(t, stateDir)

	var out, errOut bytes.Buffer
	// StateBaseDir deliberately not injected: this is the production wiring.
	cmd := NewRootCommand(Dependencies{Out: &out, Err: &errOut, GitHub: &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, RemoteExecutor: doctorRemote(true), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"doctor", "--repo", repo.Repo.FullName, "--ignore", "host_mem_low", "--no-color"})
	_ = cmd.Execute()

	assertDirEmpty(t, cwd)
	cfg, err := LoadUserConfig(stateDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	found := false
	for _, id := range cfg.DoctorIgnoreFindingIDs {
		if id == "host_mem_low" {
			found = true
		}
	}
	if !found {
		t.Fatalf("doctor --ignore did not persist to %s: %#v\nstdout=%s\nstderr=%s", stateDir, cfg, out.String(), errOut.String())
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func assertDirEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected %s to stay empty, found %v", dir, names)
	}
}
