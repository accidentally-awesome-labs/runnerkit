package hetzner

import (
	"context"
	"errors"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

const (
	artifactProviderServer    = "provider_server"
	artifactProviderSSHKey    = "provider_ssh_key"
	artifactProviderFirewall  = "provider_firewall"
	artifactProviderPrimaryIP = "provider_primary_ip"
)

func (p *Provider) Destroy(ctx context.Context, ref state.ProviderRef) (provider.DestroyResult, error) {
	client, _, err := p.client()
	if err != nil {
		return provider.DestroyResult{}, err
	}
	ids := mergedProviderIDs(ref)
	result := provider.DestroyResult{Results: []provider.ArtifactResult{}, Pending: []string{}}
	apply := func(artifact string, id string, delete func(context.Context, int) error, pending string) {
		if strings.TrimSpace(id) == "" {
			result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifact, Status: "skipped", Message: "not tracked"})
			return
		}
		parsed, parseErr := parseID(id)
		if parseErr != nil {
			result.Partial = true
			result.Pending = append(result.Pending, pending)
			result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifact, Status: "pending", Message: parseErr.Error()})
			return
		}
		if err := delete(ctx, parsed); err != nil && !isAlreadyAbsentError(err) {
			result.Partial = true
			result.Pending = append(result.Pending, pending)
			result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifact, Status: "pending", Message: err.Error()})
			return
		}
		result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifact, Status: "done"})
	}
	// Bug 23 (Plan 06-10, 2026-05-06) + Bug 26 (Plan 06-11, 2026-05-06):
	// destroy ordering relies on Hetzner's `auto_delete=true` cascade for
	// primary IPs.
	//
	// Auto-allocated primary IPs (created via ServerCreatePublicNet
	// EnableIPv4=true / EnableIPv6=true without an explicit IPv4/IPv6
	// PrimaryIP override) carry `auto_delete=true` by default. The
	// hcloud-go v1.59.2 PrimaryIP struct exposes this as `AutoDelete bool`
	// — see hcloud/primary_ip.go. When the server is deleted, Hetzner
	// cascade-deletes those primary IPs whose AutoDelete flag is set.
	// This is empirically verified live (2026-05-06): after
	// server.Delete, a follow-up GET of the IP IDs returns 404, no
	// `server_not_stopped` or `must_be_unassigned` errors anywhere.
	//
	// Plan 06-10 Bug 23 added a manual `unassign primary IPs` step
	// before server.Delete. That step turned out to require the server
	// to be powered off (`Server must be offline for this action
	// (server_not_stopped)`). Since the cascade already handles the
	// detachment correctly, we remove the unassign step entirely and
	// rely on Hetzner's default — which also keeps destroy idempotent
	// because the explicit DeletePrimaryIP calls below now race against
	// the cascade (Hetzner returns 404 once the cascade completes; our
	// isAlreadyAbsentError already treats 404 as a no-op).
	//
	// Firewall detach STILL runs first — firewalls are not part of the
	// auto_delete cascade, and firewall.Delete rejects with
	// `resource_in_use` while still attached to the server. Detach has
	// no power-off requirement.
	//
	// Final order:
	//   1. Detach firewall from server (best-effort, 404-tolerant).
	//   2. Delete server (cascade-deletes auto_delete primary IPs).
	//   3. Delete ssh_key.
	//   4. Delete primary IPv4/IPv6 (no-op via 404 cascade; idempotent).
	//   5. Delete firewall last (now detached, no resource_in_use).
	if serverID, ok := parsedNonEmpty(ids["server"]); ok {
		if firewallID, ok := parsedNonEmpty(ids["firewall"]); ok {
			if err := client.DetachFirewallFromServer(ctx, firewallID, serverID); err != nil && !isAlreadyAbsentError(err) {
				// Detach failure is recorded as a non-fatal warning so
				// destroy still attempts the deletes; if the firewall
				// genuinely cannot be detached, firewall.Delete below
				// will surface the real error and partial cleanup will
				// keep state.
				result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderFirewall, Status: "warning", Message: "detach failed: " + err.Error()})
			}
		}
	}
	apply(artifactProviderServer, ids["server"], client.DeleteServer, "provider_server_pending")
	apply(artifactProviderSSHKey, ids["ssh_key"], client.DeleteSSHKey, "provider_ssh_key_pending")
	apply(artifactProviderPrimaryIP, ids["primary_ipv4"], client.DeletePrimaryIP, "provider_primary_ip_pending")
	apply(artifactProviderPrimaryIP, ids["primary_ipv6"], client.DeletePrimaryIP, "provider_primary_ip_pending")
	apply(artifactProviderFirewall, ids["firewall"], client.DeleteFirewall, "provider_firewall_pending")
	return result, nil
}

// parsedNonEmpty returns (parsedInt, true) when the id string is
// present and parseable; (0, false) otherwise. Used to gate the new
// detach/unassign steps without short-circuiting the existing apply
// loop, which still records skipped/pending statuses for missing or
// malformed IDs.
func parsedNonEmpty(id string) (int, bool) {
	if strings.TrimSpace(id) == "" {
		return 0, false
	}
	parsed, err := parseID(id)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (p *Provider) VerifyDestroyed(ctx context.Context, ref state.ProviderRef) (provider.VerificationResult, error) {
	client, _, err := p.client()
	if err != nil {
		return provider.VerificationResult{}, err
	}
	ids := mergedProviderIDs(ref)
	verification := provider.VerificationResult{OK: true, BillableResources: []string{}, Missing: []string{}}
	check := func(kind string, id string, get func(context.Context, int) (bool, error)) {
		if strings.TrimSpace(id) == "" {
			verification.Missing = append(verification.Missing, kind)
			return
		}
		parsed, parseErr := parseID(id)
		if parseErr != nil {
			verification.OK = false
			verification.BillableResources = append(verification.BillableResources, kind+":"+id)
			verification.Error = parseErr.Error()
			return
		}
		found, err := get(ctx, parsed)
		if err != nil && !isAlreadyAbsentError(err) {
			verification.OK = false
			verification.BillableResources = append(verification.BillableResources, kind+":"+id)
			verification.Error = err.Error()
			return
		}
		if found {
			verification.OK = false
			verification.BillableResources = append(verification.BillableResources, kind+":"+id)
		}
	}
	check("server", ids["server"], func(ctx context.Context, id int) (bool, error) {
		v, err := client.GetServer(ctx, id)
		return v != nil, err
	})
	check("ssh_key", ids["ssh_key"], func(ctx context.Context, id int) (bool, error) {
		v, err := client.GetSSHKey(ctx, id)
		return v != nil, err
	})
	check("firewall", ids["firewall"], func(ctx context.Context, id int) (bool, error) {
		v, err := client.GetFirewall(ctx, id)
		return v != nil, err
	})
	check("primary_ipv4", ids["primary_ipv4"], func(ctx context.Context, id int) (bool, error) {
		v, err := client.GetPrimaryIP(ctx, id)
		return v != nil, err
	})
	check("primary_ipv6", ids["primary_ipv6"], func(ctx context.Context, id int) (bool, error) {
		v, err := client.GetPrimaryIP(ctx, id)
		return v != nil, err
	})
	return verification, nil
}

func mergedProviderIDs(ref state.ProviderRef) map[string]string {
	out := map[string]string{}
	for _, source := range []map[string]string{ref.IDs, ref.ResourceIDs} {
		for k, v := range source {
			if strings.TrimSpace(v) != "" {
				out[k] = v
			}
		}
	}
	if ref.Cloud.ServerID != "" {
		out["server"] = ref.Cloud.ServerID
	}
	if ref.Cloud.SSHKeyID != "" {
		out["ssh_key"] = ref.Cloud.SSHKeyID
	}
	if ref.Cloud.FirewallID != "" {
		out["firewall"] = ref.Cloud.FirewallID
	}
	if ref.Cloud.PrimaryIPv4ID != "" {
		out["primary_ipv4"] = ref.Cloud.PrimaryIPv4ID
	}
	if ref.Cloud.PrimaryIPv6ID != "" {
		out["primary_ipv6"] = ref.Cloud.PrimaryIPv6ID
	}
	return out
}

func isAlreadyAbsentError(err error) bool {
	if err == nil {
		return false
	}
	var target interface{ StatusCode() int }
	if errors.As(err, &target) && target.StatusCode() == 404 {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "404") || strings.Contains(text, "not found") || strings.Contains(text, "not_found")
}
