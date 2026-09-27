package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
	"github.com/spf13/cobra"
)

type initOptions struct {
	printInstallCommand bool
	printScriptURL      bool
}

func newInitCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &initOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Print BYO host bootstrap helpers (one-time install)",
		Long:  "Shows how to run the one-time install.sh on your runner host so runnerkit register/up can use passwordless sudo for RunnerKit's bootstrap commands over SSH (root-equivalent; see docs/security-posture.md).",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInit(deps, *jsonOutput, *noColor, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.printInstallCommand, "print-install-command", false, "print the curl|sudo bash one-liner for the runner host")
	cmd.Flags().BoolVar(&opts.printScriptURL, "print-script-url", false, "print the HTTPS URL to install.sh for this CLI version")
	return cmd
}

func runInit(deps Dependencies, jsonOutput bool, noColor bool, opts *initOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	out := deps.Out
	if out == nil {
		out = io.Discard
	}
	if deps.Explain() && !jsonOutput {
		why, runs, takes := explainInitDefault()
		printExplainBlock(deps.Err, "runnerkit init", why, runs, takes)
	}
	v := deps.Version
	line := HostInstallOneLiner(v)
	url := InstallScriptReleaseURL(v)

	if opts.printScriptURL && opts.printInstallCommand {
		_ = renderer.Error("invalid_flags", "Pass only one of --print-install-command or --print-script-url.", nil)
		return NewExitError(ExitInvalidInput, errors.New("conflicting init flags"))
	}

	if opts.printScriptURL {
		if jsonOutput {
			return renderer.JSON(nextaction.MergePayload(map[string]any{
				"ok":                 true,
				"command":            "init",
				"install_script_url": url,
			}, "info", nil))
		}
		_, err := fmt.Fprintln(out, url)
		return err
	}

	if opts.printInstallCommand {
		if jsonOutput {
			p := map[string]any{
				"ok":                   true,
				"command":              "init",
				"install_script_url":   url,
				"host_install_command": line,
			}
			nextaction.MergePayload(p, "info", append(byoUnsupportedInitActions(deps), nextaction.Action{
				ID: "host_install", Severity: nextaction.SeverityInfo, Title: "Run on the runner host once", Command: line, Kind: "run_on_host",
			}))
			return renderer.JSON(p)
		}
		// stdout carries only the install line, so it can be captured.
		if deps.BYOUnsupportedRelease {
			_, _ = fmt.Fprintln(deps.Err, byoUnsupportedInitWarning)
		}
		_, err := fmt.Fprintln(out, line)
		return err
	}

	if jsonOutput {
		p := map[string]any{
			"ok":                   true,
			"command":              "init",
			"install_script_url":   url,
			"host_install_command": line,
		}
		nextaction.MergePayload(p, "info", append(byoUnsupportedInitActions(deps), nextaction.Action{
			ID: "host_install", Severity: nextaction.SeverityInfo, Title: "Run on the runner host once", Command: line, Kind: "run_on_host",
		}))
		return renderer.JSON(p)
	}

	title := "BYO host one-time install"
	var body []string
	if deps.BYOUnsupportedRelease {
		body = append(body, byoUnsupportedInitWarning)
	}
	body = append(body,
		"SSH to the Linux runner machine, then run:",
		line,
		"After that, run runnerkit up from this workstation for the first repository on the host; runnerkit register adds more repositories to it.",
	)
	return renderer.Warning(title, body, "")
}

// byoUnsupportedInitWarning: in a release without a passing real-job BYO
// gate (A-21), say so before the user changes the host's sudo rules.
const byoUnsupportedInitWarning = byoUnsupportedReleaseMessage + " runnerkit up and register refuse a BYO host unless you pass --accept-known-issues, and the install below gives your SSH user root-equivalent sudo."

func byoUnsupportedInitActions(deps Dependencies) []nextaction.Action {
	if !deps.BYOUnsupportedRelease {
		return nil
	}
	return []nextaction.Action{{ID: "byo_unsupported_release", Severity: nextaction.SeverityWarning, Title: byoUnsupportedInitWarning}}
}
