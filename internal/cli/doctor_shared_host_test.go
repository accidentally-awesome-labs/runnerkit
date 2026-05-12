package cli

import (
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func TestAppendSharedHostDoctorFindingAddsSiblingEvidence(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	r1 := testsupport.HealthyRepositoryState()
	r2 := state.RepositoryState{
		Repo:             gh.Repo{Host: "github.com", Owner: "owner", Name: "other", FullName: "owner/other", Private: true},
		Auth:             r1.Auth,
		Runner:           state.RunnerIdentity{Name: "runnerkit-owner-other-local", Labels: r1.Runner.Labels, Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine:          state.MachineRef{Kind: "byo-ssh", HostRef: testsupport.TestHostRef, User: "alice", Port: 22, InstallPath: "/opt/actions-runner/runnerkit-owner-other-local", WorkDir: "/var/lib/runnerkit/work/runnerkit-owner-other-local", ServiceName: "actions.runner.runnerkit-owner-other-local.service"},
		Provider:         r1.Provider,
		Cleanup:          r1.Cleanup,
		Safety:           r1.Safety,
		RunnerKitVersion: "test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	dir := t.TempDir()
	store := state.NewStore(dir)
	if err := store.Save(state.State{SchemaVersion: state.SchemaVersion, Repositories: []state.RepositoryState{r1, r2}}); err != nil {
		t.Fatal(err)
	}
	report := &ops.DoctorReport{Findings: []ops.Finding{}}
	appendSharedHostDoctorFinding(report, store, r1)
	var got *ops.Finding
	for i := range report.Findings {
		if report.Findings[i].ID == "byo.multi_repo_shared_host" {
			got = &report.Findings[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("expected byo.multi_repo_shared_host finding, got %#v", report.Findings)
	}
	if got.Severity != string(ops.SeverityPass) {
		t.Fatalf("severity = %q", got.Severity)
	}
	if !strings.Contains(got.Evidence, "owner/other") {
		t.Fatalf("evidence should mention sibling repo: %q", got.Evidence)
	}
}
