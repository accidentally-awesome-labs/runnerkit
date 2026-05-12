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
	for _, id := range []string{CheckSSHConnectivity, CheckSSHHostKey, CheckOSRelease, CheckArch, CheckSystemd, CheckPrivilege, CheckDisk, CheckHostMemAvailable, CheckHostSwap, CheckTools, CheckNetworkGitHub, CheckTime, CheckRunnerConflict} {
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
	plentyMem := int64(16 * 1024 * 1024 * 1024)
	swap := int64(1024 * 1024 * 1024)
	return remote.ProbeResult{HostKey: remote.HostKey{Fingerprint: "SHA256:fake"}, OSRelease: map[string]string{"ID": osID}, Kernel: "linux", Arch: arch, Systemd: true, Commands: commands, DiskAvailableBytes: MinimumDiskBytes, MemAvailableBytes: plentyMem, SwapFreeBytes: swap, TimeSynchronized: true}
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

// TestCheckPrivilege_PasswordRequired asserts that when the sudo probe
// stderr indicates a password prompt is required, the report contains
// a SeverityWarning result (NOT failure) with stable ID
// host.privilege.password_required and remediation referencing one-time host install.
// Warning severity keeps report.Passed() true so the CLI can emit a dedicated host_install_required error.
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
		t.Fatalf("severity = %q, want %q", result.Severity, SeverityWarning)
	}
	if !report.Passed() {
		t.Fatalf("report.Passed() should be true for password-required warning: %#v", report.Results)
	}
	for _, want := range []string{"runnerkit init", "releases"} {
		if !strings.Contains(strings.ToLower(result.Remediation), strings.ToLower(want)) {
			t.Fatalf("remediation does not mention %q: %q", want, result.Remediation)
		}
	}
}

func TestCheckPrivilege_RequirePasswordlessSudo_FailsPassed(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
		"probe_sudo_n": {ExitCode: 1, Stderr: "sudo: a password is required"},
	}}
	target := remote.Target{User: "runnerkit-admin", Host: "203.0.113.10", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{RequirePasswordlessSudo: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	result, ok := report.Result(CheckPrivilegeCloudBootstrap)
	if !ok {
		t.Fatalf("report missing %q: %#v", CheckPrivilegeCloudBootstrap, report.Results)
	}
	if result.Severity != SeverityFailure {
		t.Fatalf("severity = %q, want failure", result.Severity)
	}
	if report.Passed() {
		t.Fatalf("report.Passed() must be false when RequirePasswordlessSudo and password sudo: %#v", report.Results)
	}
	if _, hasWarn := report.Result(CheckPrivilegePasswordReq); hasWarn {
		t.Fatalf("did not expect password_required warning when RequirePasswordlessSudo: %#v", report.Results)
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

// TestCheckPrivilege_AllowsScopedSudoers asserts that the probe at
// internal/preflight/checks.go:148 uses a Script literal that is
// present in byo-prepare's scoped sudoers allowlist (per
// internal/bootstrap/sudoers.go::RenderSudoersEntry). Bug 31
// (Plan 06-13, 2026-05-08): the prior literal `sudo -n true` was
// NOT in the allowlist, so a Path-C-prepared host (byo-prepare ran
// successfully) still fell through to Path B's TTY prompt during
// `runnerkit up`. The fix swaps the Script to
// `sudo -n install --version >/dev/null` because /usr/bin/install
// IS in the byo-prepare allowlist (and is also a RequiredTools
// member, so it is guaranteed present on hosts that pass earlier
// preflight steps).
//
// Test has two assertions:
//  1. Behavioral: a fake executor returning ExitCode=0 + the
//     live-smoke-confirmed install --version stdout classifies as
//     SeverityPass (passwordless sudo). (See gap doc Bug 31
//     lines 1465-1467 for the exact `install (GNU coreutils)
//     9.4` evidence.) This branch alone passes pre+post fix
//     because the fake keys on Command.ID rather than Script.
//  2. Source-code binding: the checks.go source contains the new
//     literal `sudo -n install --version` AND does NOT contain
//     the old literal `Script: "sudo -n true"`. This is the RED
//     gate that fails on the pre-fix code and passes after Task 2.
//
// See: .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md
// (Bug 31 lines 1433-1554) and Plan 06-13.
func TestCheckPrivilege_AllowsScopedSudoers(t *testing.T) {
	// Sub-assertion 1: behavioral (independent of Script literal)
	probe := passingProbe("ubuntu", "x86_64")
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
		"probe_sudo_n": {ExitCode: 0, Stdout: "install (GNU coreutils) 9.4\n"},
	}}
	target := remote.Target{User: "salar", Host: "mckee-small-desktop", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	result, ok := report.Result(CheckPrivilege)
	if !ok {
		t.Fatalf("report missing %q result (Bug 31 / Plan 06-13): %#v", CheckPrivilege, report.Results)
	}
	if result.Severity != SeverityPass {
		t.Fatalf("Path-C-prepared host probe should classify as SeverityPass; got %q (Bug 31 / Plan 06-13)", result.Severity)
	}
	if !report.Passed() {
		t.Fatalf("report.Passed() should be true on Path-C-prepared host (Bug 31 / Plan 06-13): %#v", report.Results)
	}
	if !strings.Contains(strings.ToLower(result.Message), "passwordless sudo") {
		t.Fatalf("message should mention passwordless sudo: %q (Bug 31 / Plan 06-13)", result.Message)
	}

	// Sub-assertion 2: source-code binding to byo-prepare allowlist
	// (RED gate -- fails pre-fix because checks.go still has
	// `Script: "sudo -n true"`).
	src, srcErr := readChecksGoSource()
	if srcErr != nil {
		t.Fatalf("read checks.go source: %v", srcErr)
	}
	if !strings.Contains(src, "sudo -n install --version") {
		t.Fatalf("checks.go missing new probe literal `sudo -n install --version` — Bug 31 (Plan 06-13) requires the privilege probe to use a command in byo-prepare's scoped allowlist. See .planning/phases/06-release-upgrade-docs-and-v1-validation/06-GAP-byo-sudo-handling.md Bug 31.")
	}
	if strings.Contains(src, `Script: "sudo -n true"`) {
		t.Fatalf("checks.go still uses old probe literal `Script: \"sudo -n true\"` — Bug 31 (Plan 06-13) replaced this with `sudo -n install --version >/dev/null` because `true` is NOT in byo-prepare's scoped sudoers allowlist (see internal/bootstrap/sudoers.go::RenderSudoersEntry).")
	}
}

func TestPreflightWarnsLowMem(t *testing.T) {
	t.Setenv("RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES", "")
	probe := passingProbe("ubuntu", "x86_64")
	probe.MemAvailableBytes = 512 * 1024 * 1024
	probe.SwapFreeBytes = 1024 * 1024 * 1024
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{"probe_sudo_n": {ExitCode: 0}}}
	report, err := Run(context.Background(), exec, remote.Target{User: "alice", Host: "example.com", Port: 22}, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, ok := report.Result(CheckHostMemAvailable)
	if !ok || r.Severity != SeverityWarning {
		t.Fatalf("expected mem warning, got ok=%v %#v", ok, r)
	}
	if !report.Passed() {
		t.Fatalf("mem warning must not fail preflight.Passed(): %#v", report.Results)
	}
}

func TestPreflightMemWarnRespectsEnvBytes(t *testing.T) {
	t.Setenv("RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES", "268435456") // 256 MiB — 512 MiB observed should pass
	probe := passingProbe("ubuntu", "x86_64")
	probe.MemAvailableBytes = 512 * 1024 * 1024
	exec := fakePreflightExecutor{probe: probe, runResults: map[string]remote.Result{
		"probe_sudo_n": {ExitCode: 0},
	}}
	report, err := Run(context.Background(), exec, remote.Target{User: "alice", Host: "example.com", Port: 22}, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, ok := report.Result(CheckHostMemAvailable)
	if !ok || r.Severity != SeverityPass {
		t.Fatalf("expected mem pass with high env threshold, got ok=%v %#v", ok, r)
	}
}
