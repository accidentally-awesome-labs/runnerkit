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

	// SudoPassword, when non-empty, causes Apply / ApplyEphemeral to
	// render sudo-prefixed commands as
	//   printf '%s\n' "$RUNNERKIT_SUDO_PASSWORD" | sudo -S <cmd>
	// The literal password value is passed via remote.Command.Env (NOT
	// interpolated into Script) and appended to RedactArgs so the
	// executor scrubs it from any captured stderr. Empty preserves the
	// existing NOPASSWD-style sudo invocation (Plan 06-05 behavior).
	// The caller (CLI) MUST register the value with redact.SudoPassword
	// before passing it here and zero the buffer in a deferred cleanup
	// after Apply returns. This is Plan 06-06 Path B's transport for
	// the prompted sudo password.
	SudoPassword string
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

// wrapSudoCommand applies Path B's `sudo -S` wrapping to a single
// remote.Command when opts.SudoPassword is set. The literal password
// flows via Env (not Script), so the rendered Script string is safe
// to log: only the env-var name appears. RedactArgs is extended so the
// executor scrubs the password from any captured stderr regardless.
//
// Behavior is deliberately a no-op when c.Sudo is false or
// SudoPassword is empty — Path B should never wrap non-sudo commands
// and Plan 06-05's NOPASSWD-style invocation must be preserved when
// the host is byo-prepared (no password needed).
//
// Bug 10 fix (Plan 06-07 attempt-7, 2026-05-05): the previous wrapper
// piped the password into a brace group containing the rewritten
// script. That structure broke any inner `printf X | sudo Y` pattern
// (e.g. `printf 'CHECKSUM' | sudo sha256sum -c -` from
// RenderInstallScript) because the inner pipe overrides sudo's
// stdin. Ubuntu's sudo defaults (use_pty + tty-scoped timestamp
// cache) did not reliably cache cred across SSH sessions, so sudo -S
// re-prompted, read from the inner printf, and treated the checksum
// string as a wrong password attempt.
//
// The fix aligns this wrapper with byo-prepare's proven structure
// (internal/cli/byo_prepare.go::runByoPrepareInstall): prime sudo's
// cred cache once with a dedicated `printf | sudo -S -v` invocation,
// then run the rewritten script WITHOUT an outer brace-group pipe.
// Each subsequent sudo -S hits the freshly-primed cred and does not
// read its stdin, so inner pipes reach their intended destination.
//
// Note: the rewrite also catches `sudo -u` because `sudo -S -u USER`
// is the supported form.
func wrapSudoCommand(c remote.Command, opts Options) remote.Command {
	if opts.SudoPassword == "" || !c.Sudo {
		return c
	}
	rewritten := RewriteSudoForPasswordPipe(c.Script)
	c.Script = "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | sudo -S -v\n" + rewritten
	if c.Env == nil {
		c.Env = map[string]string{}
	}
	c.Env["RUNNERKIT_SUDO_PASSWORD"] = opts.SudoPassword
	c.RedactArgs = append(c.RedactArgs, opts.SudoPassword)
	return c
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
		{ID: "configure_runner", Script: RenderInstallScript(opts), Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}, RedactArgs: []string{opts.RunnerToken}, Sudo: true},
		{ID: "install_service", Script: RenderServiceScript(opts), Sudo: true},
		{ID: "verify_service", Script: "set -euo pipefail\ncd " + defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName)) + "\nsudo ./svc.sh status\n", Sudo: true},
	}
	out := Result{Commands: make([]remote.Result, 0, len(commands))}
	for _, command := range commands {
		command = wrapSudoCommand(command, opts) // Path B: pipe sudo password via stdin when set.
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
		{ID: "configure_ephemeral_runner", Script: RenderEphemeralInstallScript(opts), Env: map[string]string{"RUNNERKIT_REGISTRATION_TOKEN": opts.RunnerToken}, RedactArgs: []string{opts.RunnerToken}, Sudo: true},
		{ID: "install_ephemeral_finalizer", Script: RenderEphemeralFinalizerScript(opts), Sudo: true},
		{ID: "install_ephemeral_service", Script: RenderEphemeralServiceScript(opts), Sudo: true},
		{ID: "install_ephemeral_ttl_timer", Script: RenderEphemeralTTLTimerScript(opts), Sudo: true},
		{ID: "verify_ephemeral_service", Script: fmt.Sprintf("set -euo pipefail\nsystemctl is-active %s || systemctl status %s --no-pager\n", opts.EphemeralServiceName, opts.EphemeralServiceName), Sudo: true},
	}
	out := Result{Commands: make([]remote.Result, 0, len(commands))}
	for _, command := range commands {
		command = wrapSudoCommand(command, opts) // Path B: pipe sudo password via stdin when set.
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
