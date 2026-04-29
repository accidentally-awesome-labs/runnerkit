package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/workflow"
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
		{ID: "download_runner", Script: fmt.Sprintf("set -euo pipefail\nsudo install -d -o %s -g %s %s\ncd %s\ncurl -fL --retry 3 --connect-timeout 10 -o %s %s\nprintf '%%s  %%s\n' '%s' '%s' | sha256sum -c -\ntar xzf %s --skip-old-files\n", opts.ServiceUser, opts.ServiceUser, opts.InstallPath, opts.InstallPath, opts.Package.Filename, opts.Package.URL, opts.Package.SHA256, opts.Package.Filename, opts.Package.Filename), Sudo: true},
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
}
