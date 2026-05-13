package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/workflow"
)

type Options struct {
	RunnerName   string
	RepoURL      string
	Labels       []string
	InstallPath  string
	WorkDir      string
	ServiceUser  string
	RunnerToken  string
	Package      RunnerPackage
	MissingTools []string

	// ExtraPackages are additional OS packages (e.g. libsecret-1-dev,
	// dbus-x11) to install alongside MissingTools during the
	// fix_dependencies step. Specified via --extra-packages or
	// .runnerkit/config.yaml extra_packages.
	ExtraPackages []string

	// CloudProvisioned skips BaselinePackages in fix_dependencies on the
	// cloud path because cloud-init already installed them.
	CloudProvisioned bool

	// OSReleaseID is the /etc/os-release ID field (e.g. "ubuntu",
	// "debian"). Used to gate Ubuntu-specific provisioning steps like
	// setup_runner_image.
	OSReleaseID string

	// ImageSetupVersion, when non-empty, records the image setup marker
	// version currently on the host. Used by upgrade-runner to decide
	// whether to re-run setup_runner_image.
	ImageSetupVersion string

	// RunnerCacheRoot, when set, overrides SharedRunnerCacheRoot for the
	// download_runner step (tests and non-default layouts). Production
	// leaves this empty so tarballs cache under /opt/actions-runner/runnerkit-shared-bin.
	RunnerCacheRoot string

	// Ephemeral mode controls whether RenderEphemeral* and ApplyEphemeral
	// emit a one-shot scoped runner unit instead of the persistent
	// svc.sh install/start loop. The value is "ephemeral" or empty.
	Mode string

	// EphemeralTTL safeguards an ephemeral runner that never picks up a
	// job. ApplyEphemeral defaults zero to 24h.
	EphemeralTTL time.Duration

	// LogArchivePath is the host path RunnerKit writes preserved
	// _diag/journal logs to before deleting runner files. Defaults to
	// /var/lib/runnerkit/ephemeral/<runner>/logs.
	LogArchivePath string

	// FinalizerPath is the host path of the script the systemd unit's
	// ExecStopPost hook calls (and the TTL timer triggers). Defaults to
	// /usr/local/lib/runnerkit/ephemeral/<runner>/finalize.sh.
	FinalizerPath string

	// EphemeralServiceName/EphemeralTTLServiceName/EphemeralTTLTimerName
	// override the unit names used by ApplyEphemeral. They default to
	// runnerkit-ephemeral.<runner>.{service,ttl.service,ttl.timer}.
	EphemeralServiceName    string
	EphemeralTTLServiceName string
	EphemeralTTLTimerName   string
}

type Result struct {
	Commands []remote.Result
}

// ServiceNotActiveError signals that install_service / verify_service
// (persistent path) or install_ephemeral_service / verify_ephemeral_service
// (ephemeral path) exited non-zero. Bug 12 (Plan 06-07 attempt-9, 2026-05-06)
// added CommandID + Stderr so the user-facing remediation can surface
// the failing step's actual remote output instead of a generic
// "service not active" message.
type ServiceNotActiveError struct {
	Err       error
	CommandID string
	Stderr    string
}

func (e ServiceNotActiveError) Error() string {
	if e.Err != nil {
		return "runner_service_not_active: " + e.Err.Error()
	}
	return "runner_service_not_active"
}

func Plan(opts Options) workflow.Plan { return workflow.BootstrapPlan() }

// Apply runs the persistent BYO bootstrap sequence. SEED-002 / multi-repo:
// each call uses an independent InstallPath/RunnerName/WorkDir; steps are
// safe to repeat on a host that already has another runnerkit-* install —
// create_runner_user is idempotent, download_runner uses a versioned shared
// tarball cache (see downloadRunnerCommand), configure/install_service are
// scoped to opts.InstallPath and the per-repo systemd unit.
func Apply(ctx context.Context, exec remote.Executor, target remote.Target, opts Options) (Result, error) {
	if exec == nil {
		exec = remote.UnavailableExecutor{}
	}
	normalizeOptions(&opts)
	allPackages := mergePackages(opts.MissingTools, opts.ExtraPackages, opts.CloudProvisioned)
	commands := []remote.Command{
		{ID: "fix_dependencies", Script: RenderDependencyFixScript(allPackages), Sudo: true},
	}
	if isUbuntuLike(opts.OSReleaseID) {
		commands = append(commands, remote.Command{
			ID: "setup_runner_image", Script: RenderImageSetupScript(opts.ServiceUser, opts.ImageSetupVersion), Sudo: true,
		})
	}
	commands = append(commands,
		remote.Command{ID: "create_runner_user", Script: fmt.Sprintf("set -euo pipefail\nid -u %s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %s\n", opts.ServiceUser, opts.ServiceUser), Sudo: true},
		downloadRunnerCommand(opts),
		remote.Command{ID: "configure_runner", Script: RenderInstallScript(opts), Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}, RedactArgs: []string{opts.RunnerToken}, Sudo: true},
		remote.Command{ID: "install_service", Script: RenderServiceScript(opts), Sudo: true},
		remote.Command{ID: "verify_service", Script: "set -euo pipefail\ncd " + defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName)) + "\nsudo ./svc.sh status\n", Sudo: true},
	)
	out := Result{Commands: make([]remote.Result, 0, len(commands))}
	for _, command := range commands {
		result, err := exec.Run(ctx, target, command)
		out.Commands = append(out.Commands, result)
		if err != nil || result.ExitCode != 0 {
			if command.ID == "verify_service" || command.ID == "install_service" {
				return out, ServiceNotActiveError{Err: err, CommandID: command.ID, Stderr: result.Stderr}
			}
			if err != nil {
				return out, err
			}
			return out, remote.RemoteError{CommandID: command.ID, ExitCode: result.ExitCode}
		}
	}
	return out, nil
}

// ApplyEphemeral runs the bounded one-shot ephemeral install plan: it
// reuses the dependency/user/download steps, configures the runner with
// `--ephemeral`, installs the RunnerKit-managed finalizer/service/TTL
// timer, and verifies the service started — without running the
// persistent svc.sh install/start loop. The registration token flows
// through the configure step env and is registered for redaction.
// SEED-002: safe alongside other runnerkit-* installs on the same host
// when each Options uses distinct paths and unit names.
//
// Failures of install_ephemeral_service, install_ephemeral_ttl_timer,
// or verify_ephemeral_service surface as ServiceNotActiveError so the
// CLI can render the same `runner_service_not_active` exit code copy
// it uses for the persistent path.
func ApplyEphemeral(ctx context.Context, exec remote.Executor, target remote.Target, opts Options) (Result, error) {
	if exec == nil {
		exec = remote.UnavailableExecutor{}
	}
	normalizeOptions(&opts)
	allPackages := mergePackages(opts.MissingTools, opts.ExtraPackages, opts.CloudProvisioned)
	commands := []remote.Command{
		{ID: "fix_dependencies", Script: RenderDependencyFixScript(allPackages), Sudo: true},
	}
	if isUbuntuLike(opts.OSReleaseID) {
		commands = append(commands, remote.Command{
			ID: "setup_runner_image", Script: RenderImageSetupScript(opts.ServiceUser, opts.ImageSetupVersion), Sudo: true,
		})
	}
	commands = append(commands,
		remote.Command{ID: "create_runner_user", Script: fmt.Sprintf("set -euo pipefail\nid -u %s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %s\n", opts.ServiceUser, opts.ServiceUser), Sudo: true},
		downloadRunnerCommand(opts),
		remote.Command{ID: "configure_ephemeral_runner", Script: RenderEphemeralInstallScript(opts), Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}, RedactArgs: []string{opts.RunnerToken}, Sudo: true},
		remote.Command{ID: "install_ephemeral_finalizer", Script: RenderEphemeralFinalizerScript(opts), Sudo: true},
		remote.Command{ID: "install_ephemeral_service", Script: RenderEphemeralServiceScript(opts), Sudo: true},
		remote.Command{ID: "install_ephemeral_ttl_timer", Script: RenderEphemeralTTLTimerScript(opts), Sudo: true},
		remote.Command{ID: "verify_ephemeral_service", Script: fmt.Sprintf("set -euo pipefail\nsystemctl is-active %s || systemctl status %s --no-pager\n", opts.EphemeralServiceName, opts.EphemeralServiceName), Sudo: true},
	)
	out := Result{Commands: make([]remote.Result, 0, len(commands))}
	for _, command := range commands {
		result, err := exec.Run(ctx, target, command)
		out.Commands = append(out.Commands, result)
		if err != nil || result.ExitCode != 0 {
			switch command.ID {
			case "install_ephemeral_service", "install_ephemeral_ttl_timer", "verify_ephemeral_service":
				return out, ServiceNotActiveError{Err: err, CommandID: command.ID, Stderr: result.Stderr}
			}
			if err != nil {
				return out, err
			}
			return out, remote.RemoteError{CommandID: command.ID, ExitCode: result.ExitCode}
		}
	}
	return out, nil
}

// SharedRunnerCacheRoot is the host directory holding one copy of the
// actions-runner tarball per RunnerPackage.Version (SEED-002 Phase C).
const SharedRunnerCacheRoot = "/opt/actions-runner/runnerkit-shared-bin"

// downloadRunnerCommand returns the bootstrap "download_runner"
// remote.Command shared by Apply and ApplyEphemeral. The install
// directory is created with `sudo install -d -o serviceUser` so it
// is owned by the service user; curl, sha256sum -c -, and tar xzf
// must therefore run with sudo so root can write into a directory
// the SSH user does not own. Plain (non-sudo) curl/sha256sum/tar
// against this dir hits `Permission denied` on any host where the
// SSH user is not the service user — see gap doc
// 06-GAP-byo-sudo-handling.md Bug 2.
//
// Tarballs are downloaded once per version under SharedRunnerCacheRoot;
// each repo install dir only runs tar extract from that cache (saves
// bandwidth and disk for multi-repo hosts).
func downloadRunnerCommand(opts Options) remote.Command {
	normalizeOptions(&opts)
	ver := opts.Package.Version
	if strings.TrimSpace(ver) == "" {
		ver = RunnerVersion
	}
	cacheRoot := SharedRunnerCacheRoot
	if strings.TrimSpace(opts.RunnerCacheRoot) != "" {
		cacheRoot = strings.TrimSpace(opts.RunnerCacheRoot)
	}
	cacheDir := filepath.Join(cacheRoot, ver)
	cacheTar := filepath.Join(cacheDir, opts.Package.Filename)
	installPath := defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName))
	return remote.Command{
		ID: "download_runner",
		Script: fmt.Sprintf("set -euo pipefail\n"+
			"CACHE_DIR=%s\n"+
			"CACHE_TAR=%s\n"+
			"sudo install -d -o root -g root \"$CACHE_DIR\"\n"+
			"if [ ! -f \"$CACHE_TAR\" ]; then\n"+
			"  sudo curl -fL --retry 3 --connect-timeout 10 -o \"$CACHE_TAR\" %s\n"+
			"  ( cd \"$CACHE_DIR\" && printf '%%s  %%s\\n' '%s' '%s' | sudo sha256sum -c - )\n"+
			"fi\n"+
			"sudo install -d -o %s -g %s %s\n"+
			"sudo tar xzf \"$CACHE_TAR\" -C %s --skip-old-files\n",
			shellSingleQuoted(cacheDir),
			shellSingleQuoted(cacheTar),
			opts.Package.URL,
			opts.Package.SHA256, opts.Package.Filename,
			opts.ServiceUser, opts.ServiceUser, installPath,
			installPath),
		Sudo: true,
	}
}

// shellSingleQuoted returns s wrapped in single quotes for sh -c safe embedding.
func shellSingleQuoted(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func normalizeOptions(opts *Options) {
	if opts.ServiceUser == "" {
		opts.ServiceUser = DefaultServiceUser
	}
	if opts.InstallPath == "" {
		opts.InstallPath = filepath.Join("/opt/actions-runner", opts.RunnerName)
	}
	if opts.WorkDir == "" {
		opts.WorkDir = filepath.Join("/var/lib/runnerkit/work", opts.RunnerName)
	}
	if opts.Mode == "ephemeral" {
		if opts.LogArchivePath == "" {
			opts.LogArchivePath = "/var/lib/runnerkit/ephemeral/" + opts.RunnerName + "/logs"
		}
		if opts.FinalizerPath == "" {
			opts.FinalizerPath = "/usr/local/lib/runnerkit/ephemeral/" + opts.RunnerName + "/finalize.sh"
		}
		if opts.EphemeralServiceName == "" {
			opts.EphemeralServiceName = "runnerkit-ephemeral." + opts.RunnerName + ".service"
		}
		if opts.EphemeralTTLServiceName == "" {
			opts.EphemeralTTLServiceName = "runnerkit-ephemeral." + opts.RunnerName + ".ttl.service"
		}
		if opts.EphemeralTTLTimerName == "" {
			opts.EphemeralTTLTimerName = "runnerkit-ephemeral." + opts.RunnerName + ".ttl.timer"
		}
		if opts.EphemeralTTL == 0 {
			opts.EphemeralTTL = 24 * time.Hour
		}
	}
}

// BaselinePackages matches the apt package set from the GitHub-hosted
// Ubuntu 24.04 runner image (actions/runner-images). RunnerKit always
// installs these during fix_dependencies so self-hosted runners have
// the same toolchain CI workflows expect.
var BaselinePackages = []string{
	"acl", "aria2", "autoconf", "automake", "binutils", "bison",
	"brotli", "build-essential", "bzip2", "coreutils", "curl",
	"dbus", "dnsutils", "dpkg-dev", "fakeroot", "file", "findutils",
	"flex", "fonts-noto-color-emoji", "ftp", "g++", "gcc", "gnupg2",
	"haveged", "iproute2", "iputils-ping", "jq", "libnss3-tools",
	"libsqlite3-dev", "libssl-dev", "libtool", "libyaml-dev",
	"locales", "lz4", "m4", "make", "mediainfo", "mercurial",
	"net-tools", "netcat", "openssh-client", "p7zip-full",
	"p7zip-rar", "parallel", "patchelf", "pigz", "pkg-config",
	"pollinate", "python-is-python3", "rpm", "rsync", "shellcheck",
	"sphinxsearch", "sqlite3", "ssh", "sshpass", "sudo", "swig",
	"systemd-coredump", "tar", "telnet", "texinfo", "time", "tk",
	"tree", "tzdata", "unzip", "upx", "wget", "xvfb", "xz-utils",
	"zip", "zsync",
}

// mergePackages deduplicates missingTools, BaselinePackages (unless
// cloud-init already installed them), and extraPackages into a single
// slice for RenderDependencyFixScript.
func mergePackages(missingTools, extraPackages []string, cloudProvisioned bool) []string {
	var baseline []string
	if !cloudProvisioned {
		baseline = BaselinePackages
	}
	seen := make(map[string]bool, len(missingTools)+len(baseline)+len(extraPackages))
	var merged []string
	for _, sources := range [][]string{missingTools, baseline, extraPackages} {
		for _, pkg := range sources {
			if !seen[pkg] {
				seen[pkg] = true
				merged = append(merged, pkg)
			}
		}
	}
	return merged
}

// isUbuntuLike returns true for distro IDs where the full runner
// image setup script (Docker, Chrome, NodeSource, etc.) is supported.
func isUbuntuLike(osReleaseID string) bool {
	switch osReleaseID {
	case "ubuntu", "debian", "linuxmint":
		return true
	}
	return false
}
