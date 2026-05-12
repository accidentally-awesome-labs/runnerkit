package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

func TestListJSONContractAndHostFilter(t *testing.T) {
	stateDir := t.TempDir()
	now := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	r1 := testsupport.HealthyRepositoryState()
	r2 := state.RepositoryState{
		Repo:             gh.Repo{Host: "github.com", Owner: "owner", Name: "other", FullName: "owner/other", Private: true},
		Auth:             r1.Auth,
		Runner:           state.RunnerIdentity{Name: "runnerkit-owner-other-local", Labels: r1.Runner.Labels, Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine:          state.MachineRef{Kind: "byo-ssh", HostRef: "bob@other.example.com:22", User: "bob", Port: 22, InstallPath: "/opt/actions-runner/runnerkit-owner-other-local", WorkDir: "/var/lib/runnerkit/work/runnerkit-owner-other-local", ServiceName: "actions.runner.runnerkit-owner-other-local.service"},
		Provider:         r1.Provider,
		Cleanup:          r1.Cleanup,
		Safety:           r1.Safety,
		RunnerKitVersion: "test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	st := state.State{SchemaVersion: state.SchemaVersion, Repositories: []state.RepositoryState{r1, r2}}
	if err := state.NewStore(stateDir).Save(st); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	in := strings.NewReader("")
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		In:             in,
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Sleep:          noSleep,
		TTY:            ui.TerminalCapabilities{StdinTTY: false, StdoutTTY: false, Width: 80},
	})
	cmd.SetArgs([]string{"--json", "list", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list: %v stderr=%s", err, errOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &payload); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	if payload["ok"] != true || payload["command"] != "list" {
		t.Fatalf("payload: %#v", payload)
	}
	if _, ok := payload["schema_version"]; !ok {
		t.Fatalf("missing schema_version: %s", out.String())
	}
	hosts, _ := payload["hosts"].([]any)
	if len(hosts) != 2 {
		t.Fatalf("want 2 host groups, got %d: %s", len(hosts), out.String())
	}
	next, ok := payload["next_actions"].([]any)
	if !ok || next == nil {
		t.Fatalf("next_actions must be array: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	cmd.SetArgs([]string{"--json", "list", "--host", "alice@example.com", "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list --host: %v", err)
	}
	var p2 map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &p2); err != nil {
		t.Fatal(err)
	}
	h2, _ := p2["hosts"].([]any)
	if len(h2) != 1 {
		t.Fatalf("filter want 1 host, got %v", h2)
	}
}

func TestUnregisterAliasFindsDownCommand(t *testing.T) {
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		StateBaseDir:   t.TempDir(),
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Sleep:          noSleep,
	})
	sub, _, err := cmd.Find([]string{"unregister"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if sub == nil || sub.Name() != "down" {
		t.Fatalf("unregister alias: got %#v", sub)
	}
}
