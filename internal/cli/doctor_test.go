package cli

import (
	"bytes"
	"strings"
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func doctorRemoteFailedServiceWithOOMLogs() *testsupport.RemoteExecutor {
	plentyMem := int64(16 * 1024 * 1024 * 1024)
	swap := int64(1024 * 1024 * 1024)
	return &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		ProbeResult:        remote.ProbeResult{HostKey: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Kernel: "linux", Arch: "x86_64", OSRelease: map[string]string{"ID": "ubuntu"}, Systemd: true, Commands: map[string]bool{"sudo": true, "curl": true, "tar": true, "gzip": true, "sha256sum": true, "id": true, "useradd": true, "install": true, "timedatectl": true}, DiskAvailableBytes: 2147483648, MemAvailableBytes: plentyMem, SwapFreeBytes: swap, TimeSynchronized: true},
		Results: map[string]remote.Result{
			ops.CommandStatusSSHReachable: {ExitCode: 0},
			ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=failed\nSubState=failed\nUnitFileState=enabled\nExecMainStatus=1\n", ExitCode: 0},
			"doctor.path.install":         {ExitCode: 0},
			"doctor.path.work":            {ExitCode: 0},
			"doctor.preflight":            {ExitCode: 0},
			"host.network.github.github": {ExitCode: 0},
			"host.network.github.api":     {ExitCode: 0},
			ops.CommandDoctorJournalRunner: {Stdout: "May 10 10:00:00 host Runner.Listener[123]: ld terminated with signal 9 [Killed]\n", ExitCode: 0},
			ops.CommandDoctorJournalKernel: {Stdout: "Out of memory: Killed process 999 (ld)\n", ExitCode: 0},
		},
	}
}

func doctorRemote(active bool) *testsupport.RemoteExecutor {
	activeState := "active"
	exit := 0
	if !active {
		activeState = "failed"
		exit = 1
	}
	plentyMem := int64(16 * 1024 * 1024 * 1024)
	swap := int64(1024 * 1024 * 1024)
	return &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		ProbeResult:        remote.ProbeResult{HostKey: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"}, Kernel: "linux", Arch: "x86_64", OSRelease: map[string]string{"ID": "ubuntu"}, Systemd: true, Commands: map[string]bool{"sudo": true, "curl": true, "tar": true, "gzip": true, "sha256sum": true, "id": true, "useradd": true, "install": true, "timedatectl": true}, DiskAvailableBytes: 2147483648, MemAvailableBytes: plentyMem, SwapFreeBytes: swap, TimeSynchronized: true},
		Results: map[string]remote.Result{
			ops.CommandStatusSSHReachable: {ExitCode: exit},
			ops.CommandStatusSystemdShow:  {Stdout: "LoadState=loaded\nActiveState=" + activeState + "\nSubState=" + activeState + "\nUnitFileState=enabled\nExecMainStatus=0\n", ExitCode: 0},
			"doctor.path.install":         {ExitCode: 0},
			"doctor.path.work":            {ExitCode: 0},
			"doctor.preflight":            {ExitCode: 0},
			"host.network.github.github":  {ExitCode: 0},
			"host.network.github.api":     {ExitCode: 0},
			ops.CommandDoctorJournalRunner: {Stdout: "", ExitCode: 0},
			ops.CommandDoctorJournalKernel: {Stdout: "", ExitCode: 0},
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
	if !strings.Contains(out, `"redactions_applied":true`) || !strings.Contains(out, `"findings"`) || !strings.Contains(out, "service_active") || !strings.Contains(out, `"host_incident_hints"`) {
		t.Fatalf("json doctor missing contract fields:\n%s", out)
	}
	if strings.Contains(out, `"host_incident_hints":null`) || strings.Contains(out, `"next_actions":null`) {
		t.Fatalf("json doctor must encode slices as arrays, not null:\n%s", out)
	}
	if !strings.Contains(out, `"host_incident_hints":[`) {
		t.Fatalf("json doctor missing host_incident_hints array:\n%s", out)
	}
}

func TestDoctorHostIncidentHintsWhenServiceFailed(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	exec := doctorRemoteFailedServiceWithOOMLogs()
	out, errOut, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, exec, "doctor", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("doctor returned error: %v\nstderr=%s", err, errOut)
	}
	if !strings.Contains(out, "likely_linker_sigkill") && !strings.Contains(out, "likely_kernel_oom") {
		t.Fatalf("expected OOM/linker hints in doctor output:\n%s", out)
	}
	outJSON, _, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}, doctorRemoteFailedServiceWithOOMLogs(), "--json", "doctor", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("json doctor: %v", err)
	}
	if !strings.Contains(outJSON, "likely_kernel_oom") || !strings.Contains(outJSON, "host_incident_hints") {
		t.Fatalf("json doctor missing host incident hints: %s", outJSON)
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

func TestDoctorHandlesInvalidSavedHostRefWithoutCrashing(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.HealthyRepositoryState()
	repo.Machine.HostRef = "not-a-valid-target"
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	github := &testsupport.GitHubService{
		Runners: []gh.Runner{testsupport.HealthyRunner()},
	}
	out, errOut, err := executeStatusForTest(t, stateDir, github, doctorRemote(true), "doctor", "--repo", repo.Repo.FullName, "--verbose", "--no-color")
	if err != nil {
		t.Fatalf("doctor with invalid host ref should render findings, got err=%v stderr=%s", err, errOut)
	}
	if !strings.Contains(out, "install_path_missing") || !strings.Contains(out, "work_dir_missing") {
		t.Fatalf("doctor missing degraded-path findings for invalid host ref:\n%s", out)
	}
}
