package cli

import (
	"fmt"
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
// runs against a host that has not completed the one-time install (the
// runnerkit-runner user is missing). Remediation matches host install.
func RenderLifecycleFoundationMissing(renderer *ui.Renderer, jsonOutput bool, cliVersion string) error {
	line := HostInstallOneLiner(cliVersion)
	remediation := []string{
		"SSH to the runner host and run the one-liner below once so the shared runner user exists, then retry `runnerkit register`.",
		line,
		"From this machine you can also run `runnerkit init` for copy-paste instructions.",
	}
	if jsonOutput {
		payload := map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":        "lifecycle_foundation_missing",
				"message":     "RunnerKit register requires the one-time host install before adding repo runners.",
				"remediation": remediation,
			},
		}
		nextaction.MergePayload(payload, "bootstrap_blocked", nextaction.InstallHostActions(line))
		if err := renderer.JSON(payload); err != nil {
			return err
		}
		return NewExitError(ExitInputRequired, fmt.Errorf("lifecycle_foundation_missing"))
	}
	_ = renderer.Error("lifecycle_foundation_missing", "The shared runner user is missing on this host. Complete the one-time host install, then retry register.", remediation)
	return NewExitError(ExitInputRequired, fmt.Errorf("lifecycle_foundation_missing"))
}
