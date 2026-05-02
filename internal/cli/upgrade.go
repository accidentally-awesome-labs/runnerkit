package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/update"
	"github.com/spf13/cobra"
)

// newUpgradeCommand registers `runnerkit upgrade`. Per D-07 the command is
// print-only: it inspects the binary path to detect the install channel
// (Homebrew vs GitHub Releases binary) and prints exact upgrade
// instructions for that channel. RunnerKit NEVER replaces its own
// binary; this avoids self-update partial-failure paths.
func newUpgradeCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "upgrade"}
	cmd.Short = "Print upgrade instructions for this RunnerKit install"
	cmd.Long = "Detect how RunnerKit was installed and print the channel-correct upgrade command. Does NOT replace the running binary."
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runUpgrade(deps, *jsonOutput, *noColor)
	}
	return cmd
}

type upgradeReport struct {
	OK       bool     `json:"ok"`
	Channel  string   `json:"channel"`
	Commands []string `json:"commands"`
	Current  string   `json:"current"`
	Latest   string   `json:"latest"`
	Notes    string   `json:"notes,omitempty"`
}

func runUpgrade(deps Dependencies, jsonOutput bool, noColor bool) error {
	execPath, err := os.Executable()
	if err != nil {
		// Fall back to "unknown" channel rather than failing — the user
		// asked us how to upgrade, and a missing exec path should not
		// drop them into a generic crash. Channel detection just becomes
		// "unknown".
		execPath = ""
	}
	channel := detectChannel(execPath)
	latest := lookupLatestSilent(deps)

	report := upgradeReport{
		OK:      true,
		Channel: channel,
		Current: deps.Version,
		Latest:  latest,
	}
	switch channel {
	case "homebrew":
		report.Commands = []string{"brew upgrade runnerkit"}
	case "binary":
		report.Commands = []string{
			"Download the latest release: https://github.com/salar/runnerkit/releases/latest",
			"Verify the cosign signature and SHA256 checksum before installing — see docs/troubleshooting/README.md.",
			"Replace the existing runnerkit binary on PATH with the new one.",
		}
	default:
		report.Channel = "unknown"
		report.Commands = []string{
			"RunnerKit cannot tell how this binary was installed. Run `which runnerkit` and follow the channel-specific instructions in docs/upgrade.md.",
		}
		report.Notes = "If installed via a custom channel, refer to docs/upgrade.md for the manual binary replacement steps."
	}

	if jsonOutput {
		return json.NewEncoder(deps.Out).Encode(report)
	}
	fmt.Fprintf(deps.Out, "RunnerKit %s detected install channel: %s\n", deps.Version, report.Channel)
	if latest != "" {
		fmt.Fprintf(deps.Out, "Latest released version: %s\n", latest)
	}
	fmt.Fprintln(deps.Out, "Upgrade instructions:")
	for _, c := range report.Commands {
		fmt.Fprintln(deps.Out, "  "+c)
	}
	return nil
}

// detectChannel inspects a binary path and returns "homebrew", "binary",
// or "unknown". Homebrew install paths on macOS contain
// `/Cellar/runnerkit/<version>/bin/runnerkit` or `/Caskroom/runnerkit/`.
// Linuxbrew uses the same path layout under /home/linuxbrew/.linuxbrew.
// Symlinks are resolved so the public `/usr/local/bin/runnerkit` shim
// still points at the Cellar path.
func detectChannel(execPath string) string {
	if execPath == "" {
		return "unknown"
	}
	abs := execPath
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		abs = resolved
	}
	if strings.Contains(abs, "/Cellar/runnerkit/") || strings.Contains(abs, "/Caskroom/runnerkit/") {
		return "homebrew"
	}
	if strings.Contains(filepath.Base(abs), "runnerkit") {
		return "binary"
	}
	return "unknown"
}

// lookupLatestSilent reads the cached update-check.json (populated by the
// lazy update notice in update_notice.go) and returns the latest tag if
// known, otherwise empty string. Never fetches over the network — we
// want `runnerkit upgrade` to be instantaneous and deterministic.
func lookupLatestSilent(deps Dependencies) string {
	stateDir := deps.StateBaseDir
	if stateDir == "" {
		stateDir = state.DefaultBaseDir()
	}
	path := filepath.Join(stateDir, "update-check.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var c update.CheckedRelease
	if err := json.Unmarshal(raw, &c); err != nil {
		return ""
	}
	return c.Latest
}
