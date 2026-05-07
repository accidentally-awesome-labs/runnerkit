package hetzner

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

const (
	artifactProviderServer    = "provider_server"
	artifactProviderSSHKey    = "provider_ssh_key"
	artifactProviderFirewall  = "provider_firewall"
	artifactProviderPrimaryIP = "provider_primary_ip"
)

// defaultDestroyPrimaryIPTimeout bounds the Bug 30 retry loop for legacy
// state (AutoDelete=false) where DeletePrimaryIP can transiently return
// 409 must_be_unassigned while the auto_delete cascade is in flight.
const defaultDestroyPrimaryIPTimeout = 30 * time.Second

// destroyPrimaryIPTimeoutFromEnv resolves
// RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT into a Duration. Empty /
// unparseable / non-positive values fall back to
// defaultDestroyPrimaryIPTimeout. Bug 30 (Plan 06-12, 2026-05-06).
func destroyPrimaryIPTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT"))
	if raw == "" {
		return defaultDestroyPrimaryIPTimeout
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultDestroyPrimaryIPTimeout
	}
	return parsed
}

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
	// Bug 30 (Plan 06-12, 2026-05-06): cascade-aware primary-IP delete.
	// When state records AutoDelete=true (the post-Plan-06-12 default
	// for IPs auto-allocated via ServerCreatePublicNet EnableIPv4/IPv6),
	// SKIP the explicit call entirely — the cascade triggered by
	// server.Delete handles the IP. For legacy state (AutoDelete unset
	// or false) we keep calling DeletePrimaryIP but wrap it in a
	// bounded retry loop that treats 409 must_be_unassigned as a
	// transient cascade-in-flight signal until the IP returns 404
	// (cascade complete) or the timeout expires.
	applyPrimaryIPDelete := makePrimaryIPDeleter(ctx, &result, client.DeletePrimaryIP, p.Sleep)
	applyPrimaryIPDelete(ids["primary_ipv4"], ref.Cloud.PrimaryIPv4AutoDelete)
	applyPrimaryIPDelete(ids["primary_ipv6"], ref.Cloud.PrimaryIPv6AutoDelete)
	apply(artifactProviderFirewall, ids["firewall"], client.DeleteFirewall, "provider_firewall_pending")
	return result, nil
}

// makePrimaryIPDeleter returns a closure that handles the Bug 30 dual
// path: cascade-skip (AutoDelete=true) and bounded 409 retry
// (legacy/AutoDelete=false). The closure mutates the passed-in result
// in place to keep the call sites at the bottom of Destroy short.
func makePrimaryIPDeleter(ctx context.Context, result *provider.DestroyResult, delete func(context.Context, int) error, sleep func(time.Duration)) func(idStr string, autoDelete bool) {
	if sleep == nil {
		sleep = time.Sleep
	}
	return func(idStr string, autoDelete bool) {
		if strings.TrimSpace(idStr) == "" {
			result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "skipped", Message: "not tracked"})
			return
		}
		if autoDelete {
			// Plan 06-12 Bug 30: AutoDelete=true means the
			// auto_delete cascade triggered by server.Delete handles
			// this IP. Skip the explicit DeletePrimaryIP call so the
			// destroy report does not race the cascade-in-flight
			// window (Hetzner returns 409 must_be_unassigned during
			// that window — see isCascadeInFlightError).
			result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "skipped", Message: "auto_delete cascade"})
			return
		}
		parsed, parseErr := parseID(idStr)
		if parseErr != nil {
			result.Partial = true
			result.Pending = append(result.Pending, "provider_primary_ip_pending")
			result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "pending", Message: parseErr.Error()})
			return
		}
		// Legacy fallback (AutoDelete unset/false): retry on 409
		// must_be_unassigned until 404 (cascade complete via
		// isAlreadyAbsentError) or the bounded timeout expires.
		deadline := time.Now().Add(destroyPrimaryIPTimeoutFromEnv())
		for {
			err := delete(ctx, parsed)
			if err == nil || isAlreadyAbsentError(err) {
				result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "done"})
				return
			}
			if !isCascadeInFlightError(err) || time.Now().After(deadline) {
				result.Partial = true
				result.Pending = append(result.Pending, "provider_primary_ip_pending")
				result.Results = append(result.Results, provider.ArtifactResult{Artifact: artifactProviderPrimaryIP, Status: "pending", Message: err.Error()})
				return
			}
			sleep(1 * time.Second)
		}
	}
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

// isCascadeInFlightError returns true when err signals the Hetzner
// auto_delete cascade is still in flight on the server side: HTTP 409
// with `must_be_unassigned` in the response message. Bug 30 (Plan
// 06-12, 2026-05-06): destroy retries DeletePrimaryIP on this signal
// until 404 (cascade complete -> isAlreadyAbsentError) or the bounded
// RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT expires.
//
// Test fakes that don't implement StatusCode() but include the
// canonical substring still match — the substring is a strong enough
// signal (the wire-level error always includes it) and keeps unit
// tests free of hcloud-go internals.
func isCascadeInFlightError(err error) bool {
	if err == nil {
		return false
	}
	if !strings.Contains(strings.ToLower(err.Error()), "must_be_unassigned") {
		return false
	}
	var target interface{ StatusCode() int }
	if errors.As(err, &target) {
		return target.StatusCode() == 409
	}
	// Substring fallback: real hcloud-go errors always implement
	// StatusCode(), so this branch is reserved for test fakes that
	// surface the canonical message without status wrapping.
	return true
}
