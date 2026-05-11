package cli

import (
	"errors"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/runmode"
	"github.com/spf13/cobra"
)

func newRegisterCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &upOptions{sshPort: 22, cloudRegion: provider.HetznerDefaultRegion, cloudProfile: provider.HetznerDefaultServerType, sshAllowedCIDR: provider.HetznerDefaultSSHAllowedCIDR}
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a GitHub Actions runner on your BYO Linux host",
		Long:  "Same workflow as `runnerkit up` for bring-your-own SSH hosts: run `runnerkit init` for the one-time host install, then register from your workstation.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRegister(deps, *jsonOutput, *noColor, opts)
		},
	}
	cmd.Flags().StringVar(&opts.repo, "repo", "", "target GitHub repository as owner/name")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "accept safe defaults without interactive confirmation")
	cmd.Flags().BoolVar(&opts.nonInteractive, "non-interactive", false, "fail instead of prompting for missing input")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "preview the BYO preflight and bootstrap plan without installing")
	cmd.Flags().BoolVar(&opts.allowPublicRepoRisk, "allow-public-repo-risk", false, "explicitly acknowledge public repository persistent-runner risk")
	cmd.Flags().BoolVar(&opts.replace, "replace", false, "replace existing saved foundation state for --repo when used with --yes")
	cmd.Flags().StringVar(&opts.host, "host", "", "BYO SSH target as user@host or user@host:port")
	cmd.Flags().IntVar(&opts.sshPort, "ssh-port", 22, "SSH port for the target host")
	cmd.Flags().StringVar(&opts.sshKey, "ssh-key", "", "SSH private key path reference for the target host")
	cmd.Flags().BoolVar(&opts.allowUnknownLinux, "allow-unknown-linux", false, "try best-effort install on unverified Linux distributions")
	cmd.Flags().StringVar(&opts.mode, "mode", "", "runner mode: persistent or ephemeral")
	cmd.Flags().DurationVar(&opts.ephemeralTTL, "ephemeral-ttl", runmode.DefaultEphemeralTTL, "TTL safeguard for ephemeral runners")
	cmd.Flags().BoolVar(&opts.allowEphemeralBYORisk, "allow-ephemeral-byo-risk", false, "acknowledge that BYO ephemeral mode is not a clean VM for risky repositories")
	return cmd
}

func runRegister(deps Dependencies, jsonOutput bool, noColor bool, opts *upOptions) error {
	if strings.TrimSpace(opts.cloud) != "" {
		renderer := newRenderer(deps, jsonOutput, noColor)
		_ = renderer.Error("invalid_register_cloud", "runnerkit register is BYO-only; omit --cloud or use runnerkit up for cloud provisioning.", []string{"Run `runnerkit up --repo ... --cloud hetzner` to provision Hetzner."})
		return NewExitError(ExitInvalidInput, errors.New("register does not support --cloud"))
	}
	return runUp(deps, jsonOutput, noColor, opts)
}
