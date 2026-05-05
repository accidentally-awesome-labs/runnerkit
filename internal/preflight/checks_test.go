package preflight

import (
	"context"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

type fakePreflightExecutor struct {
	probe remote.ProbeResult
	// runResults keys probe Command.IDs to the canned remote.Result
	// returned when Run is invoked. Missing keys default to ExitCode=0.
	runResults map[string]remote.Result
}

func (f fakePreflightExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return f.probe, nil
}
func (f fakePreflightExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	if f.runResults != nil {
		if result, ok := f.runResults[command.ID]; ok {
			return result, nil
		}
	}
	return remote.Result{ExitCode: 0}, nil
}

func TestNormalizeArch(t *testing.T) {
	if got, ok := NormalizeArch("x86_64"); !ok || got != "x64" {
		t.Fatalf("x86_64 maps to %q ok=%t, want x64 true", got, ok)
	}
	if got, ok := NormalizeArch("aarch64"); !ok || got != "arm64" {
		t.Fatalf("aarch64 maps to %q ok=%t, want arm64 true", got, ok)
	}
}

func TestUnknownLinuxBlocksUnlessAllowed(t *testing.T) {
	probe := passingProbe("mysterylinux", "x86_64")
	target := remote.Target{User: "alice", Host: "example.com", Port: 22}
	blocked, err := Run(context.Background(), fakePreflightExecutor{probe: probe}, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if blocked.Passed() {
		t.Fatalf("unknown Linux passed without AllowUnknownLinux: %#v", blocked.Results)
	}
	result, _ := blocked.Result(CheckOSRelease)
	if result.Severity != SeverityFailure {
		t.Fatalf("unknown linux severity = %q, want failure", result.Severity)
	}

	allowed, err := Run(context.Background(), fakePreflightExecutor{probe: probe}, target, Options{AllowUnknownLinux: true})
	if err != nil {
		t.Fatalf("Run allowed returned error: %v", err)
	}
	if !allowed.Passed() {
		t.Fatalf("unknown Linux with AllowUnknownLinux should pass as warning: %#v", allowed.Results)
	}
	result, _ = allowed.Result(CheckOSRelease)
	if result.Severity != SeverityWarning {
		t.Fatalf("unknown linux allowed severity = %q, want warning", result.Severity)
	}
}

func TestRunEmitsAllStableCheckIDs(t *testing.T) {
	report, err := Run(context.Background(), fakePreflightExecutor{probe: passingProbe("ubuntu", "aarch64")}, remote.Target{User: "alice", Host: "example.com", Port: 22}, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, id := range []string{CheckSSHConnectivity, CheckSSHHostKey, CheckOSRelease, CheckArch, CheckSystemd, CheckPrivilege, CheckDisk, CheckTools, CheckNetworkGitHub, CheckTime, CheckRunnerConflict} {
		if _, ok := report.Result(id); !ok {
			t.Fatalf("report missing %s: %#v", id, report.Results)
		}
	}
	if report.Arch != "arm64" {
		t.Fatalf("Arch = %q, want arm64", report.Arch)
	}
}

func passingProbe(osID, arch string) remote.ProbeResult {
	commands := map[string]bool{"sudo": true, "curl": true, "tar": true, "gzip": true, "sha256sum": true, "id": true, "useradd": true, "install": true, "timedatectl": true}
	return remote.ProbeResult{HostKey: remote.HostKey{Fingerprint: "SHA256:fake"}, OSRelease: map[string]string{"ID": osID}, Kernel: "linux", Arch: arch, Systemd: true, Commands: commands, DiskAvailableBytes: MinimumDiskBytes, TimeSynchronized: true}
}

// TestCheckPrivilege_Passwordless asserts that when `sudo -n true` exits 0
// the report contains a SeverityPass result with the stable
// `host.privilege` ID and the message mentions passwordless sudo.
func TestCheckPrivilege_Passwordless(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
		"probe_sudo_n": {ExitCode: 0},
	}}
	target := remote.Target{User: "alice", Host: "example.com", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	result, ok := report.Result(CheckPrivilege)
	if !ok {
		t.Fatalf("report missing %q result: %#v", CheckPrivilege, report.Results)
	}
	if result.Severity != SeverityPass {
		t.Fatalf("severity = %q, want %q", result.Severity, SeverityPass)
	}
	if !strings.Contains(strings.ToLower(result.Message), "passwordless sudo") {
		t.Fatalf("message does not mention passwordless sudo: %q", result.Message)
	}
}

// TestCheckPrivilege_PasswordRequired asserts that when `sudo -n true`
// stderr indicates a password prompt is required, the report contains
// a SeverityWarning result (NOT failure) with stable ID
// host.privilege.password_required and remediation referencing
// `runnerkit byo-prepare` plus the interactive prompt fallback.
// Warning severity is required so report.Passed() stays true and the
// bootstrap path remains reachable for Path B fallback (Plan 06-06).
func TestCheckPrivilege_PasswordRequired(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
		"probe_sudo_n": {ExitCode: 1, Stderr: "sudo: a password is required"},
	}}
	target := remote.Target{User: "alice", Host: "example.com", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	result, ok := report.Result(CheckPrivilegePasswordReq)
	if !ok {
		t.Fatalf("report missing %q result: %#v", CheckPrivilegePasswordReq, report.Results)
	}
	if result.Severity != SeverityWarning {
		t.Fatalf("severity = %q, want %q (warning so Path B fallback can run)", result.Severity, SeverityWarning)
	}
	if !report.Passed() {
		t.Fatalf("report.Passed() should be true for password-required warning so Path B can take over: %#v", report.Results)
	}
	for _, want := range []string{"runnerkit byo-prepare", "interactive"} {
		if !strings.Contains(strings.ToLower(result.Remediation), strings.ToLower(want)) {
			t.Fatalf("remediation does not mention %q: %q", want, result.Remediation)
		}
	}
}

// TestCheckPrivilege_NotInSudoers asserts that when `sudo -n true`
// stderr indicates the user is not in sudoers the report contains a
// SeverityFailure result with stable ID host.privilege.no_sudo and a
// remediation pointing the maintainer at adding the user to sudoers.
func TestCheckPrivilege_NotInSudoers(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
		"probe_sudo_n": {ExitCode: 1, Stderr: "user alice may not run sudo on host"},
	}}
	target := remote.Target{User: "alice", Host: "example.com", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	result, ok := report.Result(CheckPrivilegeNoSudo)
	if !ok {
		t.Fatalf("report missing %q result: %#v", CheckPrivilegeNoSudo, report.Results)
	}
	if result.Severity != SeverityFailure {
		t.Fatalf("severity = %q, want %q", result.Severity, SeverityFailure)
	}
	if report.Passed() {
		t.Fatalf("not-in-sudoers must fail report.Passed(): %#v", report.Results)
	}
	if !strings.Contains(strings.ToLower(result.Remediation), "sudoers") {
		t.Fatalf("remediation should mention sudoers: %q", result.Remediation)
	}
}

// TestCheckPrivilege_SudoMissing asserts that when the sudo binary is
// not present on the host the existing failure path is preserved
// (probe is not invoked; stable ID remains host.privilege).
func TestCheckPrivilege_SudoMissing(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	probe.Commands["sudo"] = false
	exec := fakePreflightExecutor{probe: probe}
	target := remote.Target{User: "alice", Host: "example.com", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	result, ok := report.Result(CheckPrivilege)
	if !ok {
		t.Fatalf("report missing %q result: %#v", CheckPrivilege, report.Results)
	}
	if result.Severity != SeverityFailure {
		t.Fatalf("severity = %q, want %q", result.Severity, SeverityFailure)
	}
	if !strings.Contains(result.Message, "sudo is required") {
		t.Fatalf("message should reference sudo requirement: %q", result.Message)
	}
}
