package cli

import (
	"errors"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/runmode"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// v1.3.4 harm-reduction gates (backlog A-05, A-08, A-15). Cloud
// provisioning and BYO ephemeral mode are experimental and must be opted
// into explicitly; ephemeral cloud is disabled outright because the VM is
// never destroyed after its job; and `up` never overwrites saved state
// that still records RunnerKit-created Hetzner resources.
const (
	experimentalRequiredCode   = "experimental_required"
	ephemeralCloudDisabledCode = "ephemeral_cloud_disabled"
	cloudRegionRequiredCode    = "cloud_region_required"
	cloudStateExistsCode       = "cloud_state_exists"

	experimentalFlagUsage = "opt in to experimental, unsupported paths: --cloud (billed by the provider) and BYO --mode ephemeral (not isolation)"
	modeFlagUsage         = "runner mode: persistent, or ephemeral (experimental; not isolation; requires --experimental)"

	ephemeralCloudDisabledMessage = "Ephemeral cloud runners are disabled in this release: the VM is never destroyed after its job and keeps billing (known issue)."
	ephemeralCloudDisabledRemedy  = "For untrusted or public code, use GitHub-hosted runners (free and unlimited for public repositories)."

	cloudExperimentalMessage     = "--cloud is experimental in this release: it creates servers billed by Hetzner and is unsupported."
	ephemeralBYOExperimentalCopy = "--mode ephemeral on a BYO host is experimental: it is not isolation (the host is reused between jobs) and it is broken on hosts prepared by install.sh."
	cloudRegionRequiredMessage   = "--cloud hetzner needs an explicit --cloud-region; RunnerKit does not pick a default location."

	// modeBYOEphemeralExperimental labels BYO ephemeral as experimental
	// and not isolation in the mode tradeoff output.
	modeBYOEphemeralExperimental = "BYO ephemeral mode is experimental and not isolation: the host is reused between jobs, and it is broken on hosts prepared by install.sh."
)

// enforceExperimentalGates refuses experimental or disabled setup paths
// using only the parsed flags: no repository resolution, GitHub, SSH,
// provider, or state access. `up` and `register` run it in PreRunE, and
// runUp re-runs it after the interactive setup/mode prompts because those
// can set --cloud or --mode on the user's behalf.
//
// Order matters: an unsupported --cloud value is reported as such, and
// ephemeral cloud is refused even with --experimental.
func enforceExperimentalGates(renderer *ui.Renderer, opts *upOptions) error {
	cloud := strings.ToLower(strings.TrimSpace(opts.cloud))
	ephemeral := strings.EqualFold(strings.TrimSpace(opts.mode), runmode.ModeEphemeral)
	if cloud != "" {
		if cloud != provider.HetznerProvider {
			_ = renderer.Error("invalid_cloud_provider", "RunnerKit does not support cloud provider "+opts.cloud+" in Phase 4.", []string{cloudUnsupportedCopy})
			return NewExitError(ExitInvalidInput, errors.New("unsupported cloud provider"))
		}
		if ephemeral {
			return refuseEphemeralCloud(renderer)
		}
		if !opts.experimental {
			_ = renderer.Error(experimentalRequiredCode, cloudExperimentalMessage, []string{
				"Re-run with --experimental --cloud-region <location> only if you accept Hetzner billing for an unsupported path.",
				"Or pass --host user@host to use a machine you already run.",
			})
			return NewExitError(ExitInvalidInput, errors.New(experimentalRequiredCode))
		}
		if strings.TrimSpace(opts.cloudRegion) == "" {
			serverType := defaultString(opts.cloudProfile, provider.HetznerDefaultServerType)
			_ = renderer.Error(cloudRegionRequiredCode, cloudRegionRequiredMessage, []string{
				"Check in the Hetzner Cloud Console where server type " + serverType + " is currently offered, then pass --cloud-region <location>.",
			})
			return NewExitError(ExitInvalidInput, errors.New(cloudRegionRequiredCode))
		}
		return nil
	}
	if ephemeral && !opts.experimental {
		_ = renderer.Error(experimentalRequiredCode, ephemeralBYOExperimentalCopy, []string{
			"Use --mode persistent (the default) for a trusted private repository.",
			"For untrusted or public code, use GitHub-hosted runners.",
			"Re-run with --experimental only if you accept these limits.",
		})
		return NewExitError(ExitInvalidInput, errors.New(experimentalRequiredCode))
	}
	return nil
}

// refuseEphemeralCloud renders the ephemeral_cloud_disabled refusal. It
// must fire before Provider.Validate so no provider call happens.
func refuseEphemeralCloud(renderer *ui.Renderer) error {
	_ = renderer.Error(ephemeralCloudDisabledCode, ephemeralCloudDisabledMessage, []string{
		ephemeralCloudDisabledRemedy,
		"For a trusted private repository, use a persistent runner: --mode persistent.",
	})
	return NewExitError(ExitInvalidInput, errors.New(ephemeralCloudDisabledCode))
}

// liveCloudResourceIDs lists the RunnerKit-created Hetzner resource IDs a
// saved cloud state still records (server, SSH key, firewall, primary
// IPs), formatted as kind:id. An empty list means nothing billable is
// known to exist.
func liveCloudResourceIDs(ref rkstate.ProviderRef) []string {
	ids := map[string]string{}
	for _, source := range []map[string]string{ref.IDs, ref.ResourceIDs} {
		for k, v := range source {
			if strings.TrimSpace(v) != "" {
				ids[k] = v
			}
		}
	}
	for key, value := range map[string]string{
		"server":       ref.Cloud.ServerID,
		"ssh_key":      ref.Cloud.SSHKeyID,
		"firewall":     ref.Cloud.FirewallID,
		"primary_ipv4": ref.Cloud.PrimaryIPv4ID,
		"primary_ipv6": ref.Cloud.PrimaryIPv6ID,
	} {
		if strings.TrimSpace(ids[key]) == "" && strings.TrimSpace(value) != "" {
			ids[key] = value
		}
	}
	return cloudProviderResourceIDList(ids)
}

// refuseIfLiveCloudState fails with cloud_state_exists when the saved
// state for fullName is a RunnerKit-managed cloud runner that still
// records provider resource IDs. Overwriting it (via --replace or the
// interactive "replace owner/name" phrase) would drop the only record of
// servers that keep billing, so the user must run destroy first. BYO
// state and cloud state with no recorded IDs are left to the existing
// replace flow.
func refuseIfLiveCloudState(renderer *ui.Renderer, existing rkstate.RepositoryState, exists bool, fullName string) error {
	if !exists || !isCloudProvider(existing.Provider) {
		return nil
	}
	ids := liveCloudResourceIDs(existing.Provider)
	if len(ids) == 0 {
		return nil
	}
	message := "RunnerKit can't replace saved state for " + fullName + ": it records RunnerKit-managed Hetzner resources that may still be billing (" + strings.Join(ids, ", ") + ")."
	_ = renderer.Error(cloudStateExistsCode, message, []string{
		"Run runnerkit destroy --repo " + fullName + " first (verifies deletion with Hetzner), then re-run runnerkit up.",
	})
	return NewExitError(ExitInvalidInput, errors.New(cloudStateExistsCode))
}
