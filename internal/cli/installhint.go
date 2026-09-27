package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
)

const upstreamReleaseRepo = "https://github.com/accidentally-awesome-labs/runnerkit"

// InstallScriptReleaseURL returns the HTTPS URL for install.sh at a release tag (e.g. v1.2.3).
func InstallScriptReleaseURL(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" || tag == "dev" {
		return upstreamReleaseRepo + "/releases/latest/download/install.sh"
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return fmt.Sprintf("%s/releases/download/%s/install.sh", upstreamReleaseRepo, tag)
}

// HostInstallOneLiner returns a copy-paste command to run on the runner host (SSH session).
func HostInstallOneLiner(cliVersion string) string {
	url := InstallScriptReleaseURL(cliVersion)
	return fmt.Sprintf(`curl -fsSL %q | sudo bash`, url)
}

// RenderHostInstallRequired emits error JSON or human text when BYO remote sudo is password-protected.
func RenderHostInstallRequired(renderer *ui.Renderer, jsonOutput bool, cliVersion string) error {
	line := HostInstallOneLiner(cliVersion)
	remediation := []string{
		"SSH to the runner host and run the one-liner below once (interactive sudo), then retry.",
		line,
	}
	if jsonOutput {
		payload := map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":        "host_install_required",
				"message":     "RunnerKit needs passwordless sudo for bootstrap commands on the remote host. Run the one-time install on the host first.",
				"remediation": remediation,
			},
		}
		nextaction.MergePayload(payload, "bootstrap_blocked", nextaction.InstallHostActions(line))
		if err := renderer.JSON(payload); err != nil {
			return err
		}
		return NewExitError(ExitInputRequired, fmt.Errorf("host_install_required"))
	}
	_ = renderer.Error("host_install_required", "Remote sudo requires a password. Run the one-time host install, then retry.", remediation)
	return NewExitError(ExitInputRequired, fmt.Errorf("host_install_required"))
}

// RenderLifecycleFoundationMissing is returned when `runnerkit register`
// reaches a host that has no runnerkit-runner user. install.sh only writes
// the sudoers fragment; the user is created by the bootstrap `runnerkit up`
// runs, so the first repository on a host needs `up` (upCommand), and
// register adds further repositories. Before v1.3.4 this told users to
// re-run install.sh, which never fixed it.
func RenderLifecycleFoundationMissing(renderer *ui.Renderer, jsonOutput bool, upCommand string) error {
	message := "This host has no runnerkit-runner user yet: register adds a repository to a host that runnerkit up has already set up."
	remediation := []string{
		"Set this repository up with runnerkit up instead (it prepares the host and creates the user):",
		upCommand,
		"Then use runnerkit register for further repositories on this host.",
	}
	if jsonOutput {
		payload := map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":        "lifecycle_foundation_missing",
				"message":     message,
				"remediation": remediation,
			},
		}
		nextaction.MergePayload(payload, "bootstrap_blocked", []nextaction.Action{{
			ID:       "run_up_for_first_repository",
			Severity: nextaction.SeverityBlocking,
			Title:    "Set up the first repository on this host with runnerkit up",
			Command:  upCommand,
			Kind:     "run_local",
		}})
		if err := renderer.JSON(payload); err != nil {
			return err
		}
		return NewExitError(ExitInputRequired, fmt.Errorf("lifecycle_foundation_missing"))
	}
	_ = renderer.Error("lifecycle_foundation_missing", message, remediation)
	return NewExitError(ExitInputRequired, fmt.Errorf("lifecycle_foundation_missing"))
}

// registerFoundationUpCommand is the `runnerkit up` line that sets up the
// repository `register` was asked for, with the same SSH options.
func registerFoundationUpCommand(repoFullName string, opts *upOptions) string {
	parts := []string{"runnerkit", "up", "--repo", repoFullName, "--host", opts.host}
	if opts.sshPort != 0 && opts.sshPort != 22 {
		parts = append(parts, "--ssh-port", strconv.Itoa(opts.sshPort))
	}
	if strings.TrimSpace(opts.sshKey) != "" {
		parts = append(parts, "--ssh-key", opts.sshKey)
	}
	for i, part := range parts {
		if strings.ContainsAny(part, " \t'\"$`\\;&|<>()*?[]{}~!#") {
			parts[i] = shellQuote(part)
		}
	}
	return strings.Join(parts, " ")
}
