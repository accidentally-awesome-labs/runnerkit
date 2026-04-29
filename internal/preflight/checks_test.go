package preflight

import (
	"context"
	"testing"

	"github.com/salar/runnerkit/internal/remote"
)

type fakePreflightExecutor struct{ probe remote.ProbeResult }

func (f fakePreflightExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return f.probe, nil
}
func (f fakePreflightExecutor) Run(context.Context, remote.Target, remote.Command) (remote.Result, error) {
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
