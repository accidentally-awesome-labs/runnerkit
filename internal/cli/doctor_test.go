package cli

import (
	"strings"
	"testing"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/remote"
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
