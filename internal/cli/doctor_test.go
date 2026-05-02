package cli

import (
	"bytes"
	"strings"
	"testing"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
)

func doctorRemote(active bool) *testsupport.RemoteExecutor {
	activeState := "active"
	exit := 0
	if !active {
		activeState = "failed"
		exit = 1
	}
	return &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		ProbeResult:        remote.ProbeResult{HostKey: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Kernel: "linux", Arch: "x86_64", OSRelease: map[string]string{"ID": "ubuntu"}, Systemd: true, Commands: map[string]bool{"sudo": true, "curl": true, "tar": true, "gzip": true, "sha256sum": true, "id": true, "useradd": true, "install": true, "timedatectl": true}, DiskAvailableBytes: 2147483648, TimeSynchronized: true},
		Results: map[string]remote.Result{
			ops.CommandStatusSSHReachable: {ExitCode: exit},
			ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=" + activeState + "\nSubState=" + activeState + "\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
			"doctor.path.install":         {ExitCode: 0},
			"doctor.path.work":            {ExitCode: 0},
			"doctor.preflight":            {ExitCode: 0},
			"host.network.github.github":  {ExitCode: 0},
			"host.network.github.api":     {ExitCode: 0},
		},
	}
}

func TestDoctorHidesPassFindingsUnlessVerboseAndReadOnly(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	out, errOut, err := executeStatusForTest(t, stateDir, github, doctorRemote(true), "doctor", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("doctor returned error: %v\nstderr=%s", err, errOut)
	}
	if strings.Contains(out, "state_present (pass)") {
		t.Fatalf("non-verbose doctor should hide pass findings:\n%s", out)
	}
	if github.CreateRegistrationTokenCalls != 0 || github.CreateRemovalTokenCalls != 0 || github.DeleteRunnerCalls != 0 {
		t.Fatalf("doctor mutated GitHub: %#v", github)
	}
	out, _, err = executeStatusForTest(t, stateDir, github, doctorRemote(true), "doctor", "--repo", repo.Repo.FullName, "--verbose", "--no-color")
	if err != nil {
		t.Fatalf("verbose doctor returned error: %v", err)
	}
	if !strings.Contains(out, "state_present (pass)") || !strings.Contains(out, "logs_available (pass)") {
		t.Fatalf("verbose doctor missing pass findings:\n%s", out)
	}
}

func TestDoctorRedactsMachineRefAndJSONIncludesFindings(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	out, _, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, doctorRemote(false), "doctor", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}
	if strings.Contains(out, "alice@example.com:22") || !strings.Contains(out, "<redacted:machine-ref>") {
		t.Fatalf("doctor did not redact machine ref:\n%s", out)
	}
	out, _, err = executeStatusForTest(t, stateDir, &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, doctorRemote(true), "--json", "doctor", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("json doctor returned error: %v", err)
	}
	if !strings.Contains(out, `"redactions_applied":true`) || !strings.Contains(out, `"findings"`) || !strings.Contains(out, "service_active") {
		t.Fatalf("json doctor missing contract fields:\n%s", out)
	}
}

func TestDoctorCloudProviderDriftRemediationAndReadOnly(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.CloudRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	cloud := &provider.FakeProvider{DescribeOut: provider.ProviderStatus{Kind: "hetzner", Found: true, Status: "running", Region: "fsn1", ServerType: "cpx22", Image: "ubuntu-24.04", PublicHost: "203.0.113.10", BillableResources: []string{"server:srv-123"}, Drift: []string{"firewall missing"}}}
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: doctorRemote(true), Providers: provider.NewRegistry(cloud), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
	cmd.SetArgs([]string{"doctor", "--repo", repo.Repo.FullName, "--no-color"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud doctor returned error: %v\nstderr=%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "provider_drift") || !strings.Contains(out.String(), "runnerkit destroy --repo owner/repo --dry-run") {
		t.Fatalf("doctor missing provider drift remediation:\n%s", out.String())
	}
	if cloud.DescribeCalls != 1 || cloud.ProvisionCalls != 0 || cloud.DestroyCalls != 0 || github.CreateRegistrationTokenCalls != 0 || github.CreateRemovalTokenCalls != 0 {
		t.Fatalf("doctor mutated dependencies: cloud=%#v github=%#v", cloud, github)
	}
}

// TestDoctorEphemeralCompletedRecommendsCleanup proves that when an
// ephemeral runner has been auto-deregistered by GitHub and the
// finalizer reported completed, doctor surfaces the ephemeral_completed
// finding and recommends the saved cleanup command (runnerkit destroy
// for cloud, runnerkit down for BYO).
func TestDoctorEphemeralCompletedRecommendsCleanup(t *testing.T) {
	t.Run("cloud", func(t *testing.T) {
		stateDir := t.TempDir()
		repo := testsupport.EphemeralCloudRepositoryState()
		repo.Ephemeral.FinalizerStatus = "completed"
		if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
			t.Fatalf("save state: %v", err)
		}
		github := &testsupport.GitHubService{}
		cloud := &provider.FakeProvider{DescribeOut: provider.ProviderStatus{Kind: "hetzner", Found: true, Status: "running", Region: "fsn1"}}
		var out, errOut bytes.Buffer
		cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, StateBaseDir: stateDir, GitHub: github, RemoteExecutor: doctorRemote(true), Providers: provider.NewRegistry(cloud), CommandRunner: staticCommandRunner{remote: "git@github.com:owner/repo.git"}, Sleep: noSleep})
		cmd.SetArgs([]string{"doctor", "--repo", repo.Repo.FullName, "--verbose", "--no-color"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("ephemeral cloud doctor returned error: %v\nstderr=%s", err, errOut.String())
		}
		flat := strings.Join(strings.Fields(out.String()), " ")
		if !strings.Contains(flat, "ephemeral_completed") {
			t.Fatalf("expected ephemeral_completed finding in:\n%s", out.String())
		}
		if !strings.Contains(flat, "runnerkit destroy --repo owner/repo") {
			t.Fatalf("expected destroy remediation for cloud ephemeral:\n%s", out.String())
		}
	})
	t.Run("byo", func(t *testing.T) {
		stateDir := t.TempDir()
		repo := testsupport.EphemeralBYORepositoryState()
		repo.Ephemeral.FinalizerStatus = "completed"
		if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
			t.Fatalf("save state: %v", err)
		}
		github := &testsupport.GitHubService{}
		out, errOut, err := executeStatusForTest(t, stateDir, github, doctorRemote(true), "doctor", "--repo", repo.Repo.FullName, "--verbose", "--no-color")
		if err != nil {
			t.Fatalf("ephemeral byo doctor returned error: %v\nstderr=%s", err, errOut)
		}
		flat := strings.Join(strings.Fields(out), " ")
		if !strings.Contains(flat, "ephemeral_completed") {
			t.Fatalf("expected ephemeral_completed finding in:\n%s", out)
		}
		if !strings.Contains(flat, "runnerkit down --repo owner/repo") {
			t.Fatalf("expected down remediation for byo ephemeral:\n%s", out)
		}
	})
}
