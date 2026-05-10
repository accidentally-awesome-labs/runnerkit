package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/spf13/cobra"
)

// byoPrepareOptions captures CLI flags for the `runnerkit byo-prepare`
// command (Plan 06-06 Path C).
type byoPrepareOptions struct {
	host        string
	remove      bool
	yes         bool
	grantCISudo bool
	serviceUser string
}

// newByoPrepareCommand registers `runnerkit byo-prepare`. Path C of
// the gap-doc 2026-05-04 user decision: install a SCOPED sudoers
// entry on a BYO host so subsequent `runnerkit up` invocations run
// passwordlessly. Idempotent. Validated with `visudo -c` BEFORE the
// atomic rename so a malformed sudoers file can never lock the user
// out. `--remove` is the inverse operation.
func newByoPrepareCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &byoPrepareOptions{}
	cmd := &cobra.Command{Use: "byo-prepare"}
	cmd.Short = "Install scoped sudoers entries on a BYO host for RunnerKit bootstrap (optional: CI package installs)"
	cmd.Long = "Installs /etc/sudoers.d/runnerkit-installer (mode 0440) with a NOPASSWD entry for the minimum command set required by runnerkit up bootstrap (apt-get/dnf/yum, useradd, install, tar, systemctl, svc.sh). With --grant-ci-sudo on Linux, also installs /etc/sudoers.d/runnerkit-runner-ci so the GitHub Actions runner service user can run sudo apt-get/dnf/… without a TTY (see RKD-GH-008). Validated with visudo before atomic rename. Idempotent — safe to re-run. Use --remove to delete installed entries."
	cmd.Flags().StringVar(&opts.host, "host", "", "BYO SSH target as user@host or user@host:port (required)")
	cmd.Flags().BoolVar(&opts.remove, "remove", false, "remove RunnerKit sudoers drop-ins instead of installing them")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "skip confirmation prompts")
	cmd.Flags().BoolVar(&opts.grantCISudo, "grant-ci-sudo", false, "also install Linux CI sudoers for the Actions runner user (package managers only; Linux hosts only)")
	cmd.Flags().StringVar(&opts.serviceUser, "service-user", bootstrap.DefaultServiceUser, "service account that runs the GitHub Actions runner (must match runnerkit up --service-user if overridden)")
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runByoPrepare(deps, *jsonOutput, *noColor, opts)
	}
	return cmd
}

func runByoPrepare(deps Dependencies, jsonOutput bool, noColor bool, opts *byoPrepareOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	if strings.TrimSpace(opts.host) == "" {
		_ = renderer.Error("input_required", "RunnerKit can't continue because --host is required.", []string{"Pass --host user@host or user@host:port."})
		return NewExitError(ExitInputRequired, errors.New("--host required"))
	}
	target, err := remote.ParseTarget(opts.host, 22)
	if err != nil {
		_ = renderer.Error("invalid_host", "RunnerKit can't parse --host.", []string{err.Error()})
		return NewExitError(ExitInvalidInput, err)
	}
	if opts.remove {
		return runByoPrepareRemove(ctx, deps, renderer, target)
	}
	return runByoPrepareInstall(ctx, deps, renderer, target, opts)
}

func runByoPrepareInstall(ctx context.Context, deps Dependencies, renderer *ui.Renderer, target remote.Target, opts *byoPrepareOptions) error {
	serviceUser := strings.TrimSpace(opts.serviceUser)
	if serviceUser == "" {
		serviceUser = bootstrap.DefaultServiceUser
	}

	kernel := ""
	isLinux := false
	if opts.grantCISudo {
		if kRes, kErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{
			ID:     "detect_kernel",
			Script: bootstrap.RemoteKernelNameScript(),
		}); kErr == nil && kRes.ExitCode == 0 {
			kernel = strings.TrimSpace(kRes.Stdout)
			isLinux = kernel == "Linux"
		}
		if kernel != "" && !isLinux {
			fmt.Fprintf(deps.Out, "Note: --grant-ci-sudo applies only to Linux (detected %q). Skipping CI sudoers.\n", kernel)
		}
	}

	installerPrepared, _ := bootstrap.SudoersIsPrepared(ctx, deps.RemoteExecutor, target, target.User)
	wantCI := opts.grantCISudo && isLinux
	ciPrepared := true
	if wantCI {
		ok, _ := bootstrap.RunnerCISudoersIsPrepared(ctx, deps.RemoteExecutor, target, serviceUser)
		ciPrepared = ok
	}
	allDone := installerPrepared && (!wantCI || ciPrepared)
	if allDone {
		fmt.Fprintf(deps.Out, "Host %s is already prepared (sudoers entries match expected content).\n", target.Display())
		return nil
	}

	needMain := !installerPrepared
	needCI := wantCI && !ciPrepared
	if !needMain && !needCI {
		return nil
	}

	// Path C requires a sudo password to install each drop-in. Without
	// TTY/Prompts we can't collect it safely.
	if !deps.TTY.StdinTTY || deps.Prompts == nil {
		_ = renderer.Error("input_required", "RunnerKit needs a sudo password but no TTY is available.", []string{"Run runnerkit byo-prepare from an interactive terminal."})
		return NewExitError(ExitInputRequired, errors.New("no TTY"))
	}
	passwordPrompter, ok := deps.Prompts.(ui.PasswordPrompter)
	if !ok {
		_ = renderer.Error("input_required", "RunnerKit's prompter does not implement password input.", []string{"Run runnerkit byo-prepare from an interactive terminal that supports password prompts."})
		return NewExitError(ExitInputRequired, errors.New("prompter has no Password method"))
	}
	password, err := passwordPrompter.Password(ctx, ui.Prompt{Message: "Sudo password for " + target.Display() + ":"})
	if err != nil {
		return err
	}
	if password == "" {
		_ = renderer.Error("input_required", "RunnerKit received an empty sudo password.", []string{"Re-run runnerkit byo-prepare and enter the host's sudo password when prompted."})
		return NewExitError(ExitInputRequired, errors.New("empty sudo password"))
	}
	renderer.Redactor().Register(redact.SudoPassword, password)

	if needMain {
		content := bootstrap.RenderSudoersEntry(target.User)
		innerScript := bootstrap.RewriteSudoForPasswordPipe(bootstrap.RemoteVisudoCheckScript())
		wrapper := "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | sudo -S -v\n"
		script := wrapper + innerScript

		cmd := remote.Command{
			ID:         "install_sudoers",
			Script:     script,
			Sudo:       true,
			Env:        map[string]string{"RUNNERKIT_SUDOERS_CONTENT": content, "RUNNERKIT_SUDO_PASSWORD": password},
			RedactArgs: []string{password},
		}
		result, err := deps.RemoteExecutor.Run(ctx, target, cmd)
		if err != nil || result.ExitCode != 0 {
			stderr := strings.TrimSpace(result.Stderr)
			remediation := []string{
				"Verify the sudo password is correct and that the SSH user has sudo access on " + target.Display() + ".",
			}
			if stderr != "" {
				remediation = append(remediation, "Remote stderr: "+renderer.Redactor().String(stderr))
			}
			_ = renderer.Error("byo_prepare_failed", "RunnerKit could not install the scoped sudoers entry.", remediation)
			if err == nil {
				err = remote.RemoteError{CommandID: cmd.ID, ExitCode: result.ExitCode}
			}
			return NewExitError(ExitSafetyGate, err)
		}

		verify, verifyErr := deps.RemoteExecutor.Run(ctx, target, remote.Command{ID: "verify_sudo_n", Script: "sudo -n true"})
		if verifyErr != nil || verify.ExitCode != 0 {
			fmt.Fprintf(deps.Out, "Note: post-install `sudo -n true` probe did not pass; the scoped sudoers entry was installed but `true` is not in the NOPASSWD allowlist (this is expected — RunnerKit only allows-list the bootstrap commands).\n")
		}
	}

	if needCI {
		ciContent := bootstrap.RenderRunnerCISudoersEntry(serviceUser)
		innerCI := bootstrap.RewriteSudoForPasswordPipe(bootstrap.RemoteRunnerCIVisudoCheckScript())
		wrapper := "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | sudo -S -v\n"
		ciScript := wrapper + innerCI
		ciCmd := remote.Command{
			ID:         "install_ci_sudoers",
			Script:     ciScript,
			Sudo:       true,
			Env:        map[string]string{"RUNNERKIT_CI_SUDOERS_CONTENT": ciContent, "RUNNERKIT_SUDO_PASSWORD": password},
			RedactArgs: []string{password},
		}
		ciResult, ciErr := deps.RemoteExecutor.Run(ctx, target, ciCmd)
		if ciErr != nil || ciResult.ExitCode != 0 {
			stderr := strings.TrimSpace(ciResult.Stderr)
			remediation := []string{
				"Verify the sudo password is correct and that the SSH user has sudo access on " + target.Display() + ".",
			}
			if stderr != "" {
				remediation = append(remediation, "Remote stderr: "+renderer.Redactor().String(stderr))
			}
			_ = renderer.Error("byo_prepare_ci_failed", "RunnerKit could not install the CI sudoers entry.", remediation)
			if ciErr == nil {
				ciErr = remote.RemoteError{CommandID: ciCmd.ID, ExitCode: ciResult.ExitCode}
			}
			return NewExitError(ExitSafetyGate, ciErr)
		}
	}

	fmt.Fprintf(deps.Out, "Host %s is now prepared. Run `runnerkit up --host %s` to install the runner.\n", target.Display(), target.Display())
	return nil
}

func runByoPrepareRemove(ctx context.Context, deps Dependencies, renderer *ui.Renderer, target remote.Target) error {
	// First try without a password — many setups will already have
	// sudo available either via the runnerkit-installer entry itself
	// or unrelated user-managed NOPASSWD config.
	cmd := remote.Command{ID: "remove_sudoers", Script: bootstrap.RemoteSudoersRemoveScript(), Sudo: true}
	result, err := deps.RemoteExecutor.Run(ctx, target, cmd)
	if err == nil && result.ExitCode == 0 {
		fmt.Fprintf(deps.Out, "Removed RunnerKit sudoers drop-ins from host %s.\n", target.Display())
		return nil
	}

	// Fall back to interactive prompt if the first attempt failed and
	// we have a TTY.
	if deps.TTY.StdinTTY && deps.Prompts != nil {
		if passwordPrompter, ok := deps.Prompts.(ui.PasswordPrompter); ok {
			password, perr := passwordPrompter.Password(ctx, ui.Prompt{Message: "Sudo password for " + target.Display() + ":"})
			if perr != nil {
				return perr
			}
			if password != "" {
				renderer.Redactor().Register(redact.SudoPassword, password)
				cmd.Env = map[string]string{"RUNNERKIT_SUDO_PASSWORD": password}
				cmd.RedactArgs = []string{password}
				cmd.Script = "printf '%s\\n' \"$RUNNERKIT_SUDO_PASSWORD\" | sudo -S rm -f " + bootstrap.SudoersFilePath + " " + bootstrap.RunnerCISudoersFilePath
				result, err = deps.RemoteExecutor.Run(ctx, target, cmd)
				if err == nil && result.ExitCode == 0 {
					fmt.Fprintf(deps.Out, "Removed RunnerKit sudoers drop-ins from host %s.\n", target.Display())
					return nil
				}
			}
		}
	}

	stderr := strings.TrimSpace(result.Stderr)
	remediation := []string{"Verify the SSH user has sudo access on " + target.Display() + " and that the entry exists."}
	if stderr != "" {
		remediation = append(remediation, "Remote stderr: "+renderer.Redactor().String(stderr))
	}
	_ = renderer.Error("byo_prepare_remove_failed", "RunnerKit could not remove the sudoers entry.", remediation)
	if err == nil {
		err = remote.RemoteError{CommandID: cmd.ID, ExitCode: result.ExitCode}
	}
	return NewExitError(ExitSafetyGate, err)
}
