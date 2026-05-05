package preflight

import (
	"context"
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

const (
	CheckSSHConnectivity = "ssh.connectivity"
	CheckSSHHostKey      = "ssh.host_key"
	CheckOSRelease       = "host.os_release"
	CheckArch            = "host.arch"
	CheckSystemd         = "host.systemd"
	CheckPrivilege       = "host.privilege"
	// CheckPrivilegePasswordReq is emitted when the SSH user can run
	// sudo but only after entering a password. RunnerKit's bootstrap
	// commands run over a non-interactive SSH channel and cannot
	// answer a sudo prompt, so this case must be surfaced separately
	// from the bare "sudo missing" failure. Severity is warning so
	// report.Passed() stays true and the bootstrap path remains
	// reachable for Path B fallback (Plan 06-06).
	CheckPrivilegePasswordReq = "host.privilege.password_required"
	// CheckPrivilegeNoSudo is emitted when the SSH user is not
	// listed in sudoers on the remote host. Severity is failure;
	// remediation points the maintainer at adding the user to
	// sudoers or picking a host where they already are.
	CheckPrivilegeNoSudo = "host.privilege.no_sudo"
	CheckDisk            = "host.disk"
	CheckTools           = "host.tools"
	CheckNetworkGitHub   = "host.network.github"
	CheckTime            = "host.time"
	CheckRunnerConflict  = "runner.conflict"

	MinimumDiskBytes int64 = 2147483648
)

type Severity string

const (
	SeverityPass    Severity = "pass"
	SeverityWarning Severity = "warning"
	SeverityFailure Severity = "failure"
)

type Check struct {
	ID          string
	Description string
}

type Result struct {
	Check       Check
	ID          string
	Severity    Severity
	Message     string
	Remediation string
	Fixable     bool
}

type Options struct {
	AllowUnknownLinux bool
	RunnerName        string
}

type Report struct {
	Target       remote.Target
	OS           string
	Arch         string
	OSReleaseID  string
	Results      []Result
	FixableTools []string
}

func (r Report) Passed() bool {
	for _, result := range r.Results {
		if result.Severity == SeverityFailure {
			return false
		}
	}
	return true
}

func (r Report) Result(id string) (Result, bool) {
	for _, result := range r.Results {
		if result.ID == id {
			return result, true
		}
	}
	return Result{}, false
}

func Run(ctx context.Context, executor remote.Executor, target remote.Target, options Options) (Report, error) {
	if executor == nil {
		executor = remote.UnavailableExecutor{}
	}
	report := Report{Target: target, OS: "linux"}
	probe, err := executor.Probe(ctx, target)
	if err != nil {
		report.Results = append(report.Results, failure(CheckSSHConnectivity, "SSH connection failed.", "Verify SSH access to "+target.Display()+" and re-run runnerkit up."))
		return report, nil
	}
	report.Results = append(report.Results, pass(CheckSSHConnectivity, "SSH connection succeeded."))
	if probe.HostKey.Fingerprint != "" || len(probe.HostKey.PublicKey) > 0 {
		report.Results = append(report.Results, pass(CheckSSHHostKey, "SSH host key accepted."))
	} else {
		report.Results = append(report.Results, failure(CheckSSHHostKey, "SSH host key was not observed.", "Re-run with a host that presents an SSH host key."))
	}
	osID := strings.ToLower(strings.TrimSpace(probe.OSRelease["ID"]))
	report.OSReleaseID = osID
	kernel := strings.ToLower(strings.TrimSpace(probe.Kernel))
	if kernel == "" {
		kernel = "linux"
	}
	if !strings.Contains(kernel, "linux") {
		report.Results = append(report.Results, failure(CheckOSRelease, "Remote host is not Linux.", "Use a Linux systemd host for the BYO persistent runner path."))
	} else if osID == "" || !isRecognizedLinux(osID) {
		message := "Unknown Linux distribution; pass --allow-unknown-linux to try best-effort install."
		if options.AllowUnknownLinux {
			report.Results = append(report.Results, warning(CheckOSRelease, message, "Proceeding with best-effort Linux bootstrap."))
		} else {
			report.Results = append(report.Results, failure(CheckOSRelease, message, message))
		}
	} else {
		report.Results = append(report.Results, pass(CheckOSRelease, "Supported Linux distribution detected: "+osID+"."))
	}
	arch, ok := NormalizeArch(probe.Arch)
	report.Arch = arch
	if !ok {
		report.Results = append(report.Results, failure(CheckArch, "Unsupported architecture: "+probe.Arch+".", "Use a host with supported architecture x64 or arm64."))
	} else {
		report.Results = append(report.Results, pass(CheckArch, "Supported architecture detected: "+arch+"."))
	}
	if probe.Systemd {
		report.Results = append(report.Results, pass(CheckSystemd, "systemd is available."))
	} else {
		report.Results = append(report.Results, failure(CheckSystemd, "systemd is required for the Phase 2 managed service.", "Use a systemd Linux host."))
	}
	if !probe.Commands["sudo"] {
		report.Results = append(report.Results, failure(CheckPrivilege, "sudo is required for setup commands.", "Grant sudo for installation or use a host where sudo is available."))
	} else {
		// Probe whether the SSH user can run sudo non-interactively. The
		// bootstrap path runs over a non-interactive channel and cannot
		// answer a sudo password prompt, so a host with sudo-binary present
		// but a password requirement must NOT pass preflight as if it were
		// passwordless. See gap doc 06-GAP-byo-sudo-handling.md Task A.
		probeResult, probeErr := executor.Run(ctx, target, remote.Command{ID: "probe_sudo_n", Script: "sudo -n true"})
		// Bug 7 fix: classify based on the remote stderr regardless of
		// whether the executor returns a non-nil err. internal/remote/system.go::SystemExecutor.Run
		// surfaces *exec.ExitError for any non-zero remote exit, so a
		// real probe with stderr "sudo: a password is required" lands
		// here with probeErr != nil. The discriminator is the stderr
		// content, NOT the err type — so we ignore probeErr in the
		// classification branches and only use it as a last-resort
		// signal in the default case.
		switch {
		case probeErr == nil && probeResult.ExitCode == 0:
			report.Results = append(report.Results, pass(CheckPrivilege, "Passwordless sudo available for setup commands."))
		case strings.Contains(probeResult.Stderr, "password is required") || strings.Contains(probeResult.Stderr, "a terminal is required"):
			report.Results = append(report.Results, warning(CheckPrivilegePasswordReq, "sudo requires a password — RunnerKit will prompt or use byo-prepare.", "Run `runnerkit byo-prepare --host user@host` to install a scoped sudoers entry, or re-run `runnerkit up` interactively to be prompted for the sudo password."))
		case strings.Contains(probeResult.Stderr, "may not run sudo"):
			report.Results = append(report.Results, failure(CheckPrivilegeNoSudo, "User is not in sudoers on the remote host.", "Add the SSH user to sudoers or pick a host where they are."))
		default:
			stderr := strings.TrimSpace(probeResult.Stderr)
			if stderr == "" && probeErr != nil {
				stderr = probeErr.Error()
			}
			if stderr == "" {
				stderr = fmt.Sprintf("exit %d", probeResult.ExitCode)
			}
			report.Results = append(report.Results, failure(CheckPrivilege, "sudo probe failed: "+stderr, "Verify SSH access and that sudo is installed on the host."))
		}
	}
	if probe.DiskAvailableBytes >= MinimumDiskBytes {
		report.Results = append(report.Results, pass(CheckDisk, "At least 2 GiB is available."))
	} else {
		report.Results = append(report.Results, failure(CheckDisk, fmt.Sprintf("At least 2147483648 bytes are required; observed %d.", probe.DiskAvailableBytes), "Free disk space under /opt and /var/lib before installing the runner."))
	}
	missing := missingTools(probe.Commands)
	if len(missing) == 0 {
		report.Results = append(report.Results, pass(CheckTools, "Required tools are present."))
	} else {
		report.FixableTools = missing
		report.Results = append(report.Results, Result{Check: check(CheckTools), ID: CheckTools, Severity: SeverityWarning, Message: "Missing tools: " + strings.Join(missing, ", "), Remediation: "RunnerKit can install missing tools after you approve the bootstrap plan.", Fixable: true})
	}
	// Bug 8 fix: drop -f so HTTP-level errors (e.g. 403 from
	// api.github.com when the host IP exhausts the 60-req/hr anonymous
	// rate limit) do not masquerade as connectivity failures. Use
	// --max-time + --connect-timeout to keep the probe bounded; rely
	// on curl's exit code (0 means request completed at the HTTP
	// layer, regardless of HTTP status) to signal reachability.
	githubOK := runNetworkCheck(ctx, executor, target, "host.network.github.github", "curl -sS --connect-timeout 5 --max-time 10 -o /dev/null https://github.com")
	apiOK := runNetworkCheck(ctx, executor, target, "host.network.github.api", "curl -sS --connect-timeout 5 --max-time 10 -o /dev/null https://api.github.com")
	if githubOK && apiOK {
		report.Results = append(report.Results, pass(CheckNetworkGitHub, "Outbound HTTPS to GitHub and api.github.com works."))
	} else {
		report.Results = append(report.Results, failure(CheckNetworkGitHub, "Outbound HTTPS to GitHub failed.", "Allow HTTPS egress to https://github.com and https://api.github.com."))
	}
	if probe.TimeSynchronized || !probe.Commands["timedatectl"] {
		report.Results = append(report.Results, pass(CheckTime, "Remote clock appears usable."))
	} else {
		report.Results = append(report.Results, warning(CheckTime, "Remote time synchronization is not confirmed.", "Enable NTP/time sync if TLS or token expiry errors occur."))
	}
	if probe.RunnerConflict {
		report.Results = append(report.Results, failure(CheckRunnerConflict, "A runner install or service conflict already exists.", "Remove the existing runner or wait for Phase 3 cleanup support before retrying."))
	} else {
		report.Results = append(report.Results, pass(CheckRunnerConflict, "No existing RunnerKit runner conflict detected."))
	}
	return report, nil
}

func runNetworkCheck(ctx context.Context, executor remote.Executor, target remote.Target, id string, script string) bool {
	result, err := executor.Run(ctx, target, remote.Command{ID: id, Script: script})
	return err == nil && result.ExitCode == 0
}

func NormalizeArch(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x86_64", "amd64", "x64":
		return "x64", true
	case "aarch64", "arm64":
		return "arm64", true
	default:
		return "", false
	}
}

func RequiredTools() []string {
	return []string{"curl", "tar", "gzip", "sha256sum", "id", "useradd", "install"}
}

func missingTools(commands map[string]bool) []string {
	var missing []string
	for _, tool := range RequiredTools() {
		if !commands[tool] {
			missing = append(missing, tool)
		}
	}
	return missing
}

func isRecognizedLinux(id string) bool {
	switch id {
	case "ubuntu", "debian", "linuxmint", "fedora", "centos", "rhel", "rocky", "almalinux", "arch", "opensuse-leap", "opensuse-tumbleweed":
		return true
	default:
		return false
	}
}

func pass(id string, message string) Result {
	return Result{Check: check(id), ID: id, Severity: SeverityPass, Message: message}
}
func warning(id string, message string, remediation string) Result {
	return Result{Check: check(id), ID: id, Severity: SeverityWarning, Message: message, Remediation: remediation}
}
func failure(id string, message string, remediation string) Result {
	return Result{Check: check(id), ID: id, Severity: SeverityFailure, Message: message, Remediation: remediation}
}
func check(id string) Check { return Check{ID: id, Description: id} }
