package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
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

type ServiceNotActiveError struct{ Err error }

func (e ServiceNotActiveError) Error() string {
	if e.Err != nil {
		return "runner_service_not_active: " + e.Err.Error()
	}
	return "runner_service_not_active"
}

func Plan(opts Options) workflow.Plan { return workflow.BootstrapPlan() }

func Apply(ctx context.Context, exec remote.Executor, target remote.Target, opts Options) (Result, error) {
	if exec == nil {
		exec = remote.UnavailableExecutor{}
	}
	normalizeOptions(&opts)
	commands := []remote.Command{
		{ID: "fix_dependencies", Script: RenderDependencyFixScript(opts.MissingTools), Sudo: true},
		{ID: "create_runner_user", Script: fmt.Sprintf("set -euo pipefail\nid -u %s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %s\n", opts.ServiceUser, opts.ServiceUser), Sudo: true},
		downloadRunnerCommand(opts),
		{ID: "configure_runner", Script: RenderInstallScript(opts), Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}, RedactArgs: []string{opts.RunnerToken}},
		{ID: "install_service", Script: RenderServiceScript(opts), Sudo: true},
		{ID: "verify_service", Script: "set -euo pipefail\nsudo ./svc.sh status\n", Sudo: true},
	}
	out := Result{Commands: make([]remote.Result, 0, len(commands))}
	for _, command := range commands {
		result, err := exec.Run(ctx, target, command)
		out.Commands = append(out.Commands, result)
		if err != nil || result.ExitCode != 0 {
			if command.ID == "verify_service" || command.ID == "install_service" {
				return out, ServiceNotActiveError{Err: err}
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
	commands := []remote.Command{
		{ID: "fix_dependencies", Script: RenderDependencyFixScript(opts.MissingTools), Sudo: true},
		{ID: "create_runner_user", Script: fmt.Sprintf("set -euo pipefail\nid -u %s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %s\n", opts.ServiceUser, opts.ServiceUser), Sudo: true},
		downloadRunnerCommand(opts),
		{ID: "configure_ephemeral_runner", Script: RenderEphemeralInstallScript(opts), Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}, RedactArgs: []string{opts.RunnerToken}},
		{ID: "install_ephemeral_finalizer", Script: RenderEphemeralFinalizerScript(opts), Sudo: true},
		{ID: "install_ephemeral_service", Script: RenderEphemeralServiceScript(opts), Sudo: true},
		{ID: "install_ephemeral_ttl_timer", Script: RenderEphemeralTTLTimerScript(opts), Sudo: true},
		{ID: "verify_ephemeral_service", Script: fmt.Sprintf("set -euo pipefail\nsystemctl is-active %s || systemctl status %s --no-pager\n", opts.EphemeralServiceName, opts.EphemeralServiceName), Sudo: true},
	}
	out := Result{Commands: make([]remote.Result, 0, len(commands))}
	for _, command := range commands {
		result, err := exec.Run(ctx, target, command)
		out.Commands = append(out.Commands, result)
		if err != nil || result.ExitCode != 0 {
			switch command.ID {
			case "install_ephemeral_service", "install_ephemeral_ttl_timer", "verify_ephemeral_service":
				return out, ServiceNotActiveError{Err: err}
			}
			if err != nil {
				return out, err
			}
			return out, remote.RemoteError{CommandID: command.ID, ExitCode: result.ExitCode}
		}
	}
	return out, nil
}

// downloadRunnerCommand returns the bootstrap "download_runner"
// remote.Command shared by Apply and ApplyEphemeral. The install
// directory is created with `sudo install -d -o serviceUser` so it
// is owned by the service user; curl, sha256sum -c -, and tar xzf
// must therefore run with sudo so root can write into a directory
// the SSH user does not own. Plain (non-sudo) curl/sha256sum/tar
// against this dir hits `Permission denied` on any host where the
// SSH user is not the service user — see gap doc
// 06-GAP-byo-sudo-handling.md Bug 2.
func downloadRunnerCommand(opts Options) remote.Command {
	return remote.Command{
		ID: "download_runner",
		Script: fmt.Sprintf("set -euo pipefail\nsudo install -d -o %s -g %s %s\ncd %s\nsudo curl -fL --retry 3 --connect-timeout 10 -o %s %s\nprintf '%%s  %%s\n' '%s' '%s' | sudo sha256sum -c -\nsudo tar xzf %s --skip-old-files\n",
			opts.ServiceUser, opts.ServiceUser, opts.InstallPath,
			opts.InstallPath,
			opts.Package.Filename, opts.Package.URL,
			opts.Package.SHA256, opts.Package.Filename,
			opts.Package.Filename),
		Sudo: true,
	}
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
