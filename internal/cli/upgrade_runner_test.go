package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

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

// runDisabledLifecycleCommand runs args against seeded state with recording
// fakes and asserts the A-06a contract: exit 2, command_disabled, the
// v1.3.3 explanation plus manual steps, and no GitHub, SSH or state change.
func runDisabledLifecycleCommand(t *testing.T, repo rkstate.RepositoryState, args ...string) (string, string) {
	t.Helper()
	stateDir := seedRepoState(t, repo)
	before, err := os.ReadFile(rkstate.NewStore(stateDir).Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	remoteExec := newFakeRemoteExecutor()
	github := newFakePermittedGitHubService()
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		GitHub:         github,
		RemoteExecutor: remoteExec,
		StateBaseDir:   stateDir,
		CommandRunner:  staticCommandRunner{remote: "git@github.com:owner/repo.git"},
		Sleep:          noSleep,
	})
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	if runErr == nil || ExitCode(runErr) != ExitInvalidInput {
		t.Fatalf("%v: ExitCode=%d err=%v\nstdout=%s\nstderr=%s", args, ExitCode(runErr), runErr, out.String(), errOut.String())
	}
	if len(remoteExec.runs) != 0 || remoteExec.probeCalls != 0 {
		t.Fatalf("%v touched the host: %#v", args, remoteExec.runs)
	}
	if github.authCalls != 0 || github.readCalls != 0 || github.tokenCalls != 0 || github.listCalls != 0 {
		t.Fatalf("%v called GitHub: %#v", args, github)
	}
	after, err := os.ReadFile(rkstate.NewStore(stateDir).Path())
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("%v changed state.json (err=%v)", args, err)
	}
	return out.String(), errOut.String()
}

// A-06a (P0-3): upgrade-runner deleted .runner/.credentials and ran
// config.sh with an empty token. It must refuse without touching anything,
// whatever flags are passed and whatever the runner mode is.
func TestUpgradeRunner_Disabled_NoRemoteCalls(t *testing.T) {
	persistent := testsupport.HealthyRepositoryState()
	persistent.RunnerTemplateVersion = "2.330.0"
	ephemeral := testsupport.EphemeralBYORepositoryState()
	ephemeral.Ephemeral.FinalizerStatus = "waiting"
	for _, tc := range []struct {
		name string
		repo rkstate.RepositoryState
		args []string
	}{
		{"persistent_yes", persistent, []string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes", "--no-color"}},
		{"persistent_no_flags", persistent, []string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--no-color"}},
		{"ephemeral_force", ephemeral, []string{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--force", "--yes", "--no-color"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, errOut := runDisabledLifecycleCommand(t, tc.repo, tc.args...)
			for _, want := range []string{"upgrade-runner is disabled", "v1.3.3", ".credentials", "--disableupdate", "runnerkit down --repo owner/repo", "runnerkit up --repo owner/repo"} {
				if !strings.Contains(errOut, want) {
					t.Fatalf("refusal missing %q:\n%s", want, errOut)
				}
			}
		})
	}
}

func TestUpgradeRunner_DisabledJSONEnvelope(t *testing.T) {
	out, _ := runDisabledLifecycleCommand(t, testsupport.HealthyRepositoryState(), "--json", "upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes")
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("not json: %v\n%s", err, out)
	}
	errObj, _ := payload["error"].(map[string]any)
	if payload["ok"] != false || errObj["code"] != "command_disabled" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

// A-06a: doctor --fix only ever ran upgrade-runner; it is refused before
// any GitHub, SSH or state access, in human and JSON mode.
func TestDoctorFix_Disabled(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	repo.RunnerTemplateVersion = "2.330.0"
	_, errOut := runDisabledLifecycleCommand(t, repo, "doctor", "--repo", testsupport.TestRepoFullName, "--fix", "--yes", "--no-color")
	for _, want := range []string{"doctor --fix is disabled", "--disableupdate", "runnerkit down --repo owner/repo"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("doctor --fix refusal missing %q:\n%s", want, errOut)
		}
	}

	out, _ := runDisabledLifecycleCommand(t, repo, "--json", "doctor", "--repo", testsupport.TestRepoFullName, "--fix")
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("not json: %v\n%s", err, out)
	}
	errObj, _ := payload["error"].(map[string]any)
	if payload["ok"] != false || errObj["code"] != "command_disabled" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	// doctor --json contract: arrays, never null.
	for _, key := range []string{"next_actions", "host_incident_hints"} {
		if _, ok := payload[key].([]any); !ok {
			t.Fatalf("doctor --fix --json missing array %q: %#v", key, payload)
		}
	}
}
