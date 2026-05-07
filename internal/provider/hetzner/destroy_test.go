package hetzner

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

type destroyFakeClient struct {
	calls     []string
	found     bool
	deleteErr map[string]error
}

func (f *destroyFakeClient) GetLocation(context.Context, string) (*hcloud.Location, error) {
	return nil, nil
}
func (f *destroyFakeClient) GetServerType(context.Context, string) (*hcloud.ServerType, error) {
	return nil, nil
}
func (f *destroyFakeClient) GetImage(context.Context, string) (*hcloud.Image, error) { return nil, nil }
func (f *destroyFakeClient) CreateSSHKey(context.Context, hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error) {
	return nil, nil
}
func (f *destroyFakeClient) CreateFirewall(context.Context, hcloud.FirewallCreateOpts) (*hcloud.Firewall, error) {
	return nil, nil
}
func (f *destroyFakeClient) CreateServer(context.Context, hcloud.ServerCreateOpts) (*hcloud.Server, *hcloud.Action, error) {
	return nil, nil, nil
}
func (f *destroyFakeClient) WaitForAction(context.Context, *hcloud.Action) error { return nil }
func (f *destroyFakeClient) GetServer(context.Context, int) (*hcloud.Server, error) {
	f.calls = append(f.calls, "get:server")
	if f.found {
		return &hcloud.Server{ID: 1}, nil
	}
	return nil, nil
}
func (f *destroyFakeClient) GetSSHKey(context.Context, int) (*hcloud.SSHKey, error) {
	f.calls = append(f.calls, "get:ssh_key")
	if f.found {
		return &hcloud.SSHKey{ID: 2}, nil
	}
	return nil, nil
}
func (f *destroyFakeClient) GetFirewall(context.Context, int) (*hcloud.Firewall, error) {
	f.calls = append(f.calls, "get:firewall")
	if f.found {
		return &hcloud.Firewall{ID: 3}, nil
	}
	return nil, nil
}
func (f *destroyFakeClient) GetPrimaryIP(context.Context, int) (*hcloud.PrimaryIP, error) {
	f.calls = append(f.calls, "get:primary")
	if f.found {
		return &hcloud.PrimaryIP{ID: 4}, nil
	}
	return nil, nil
}
func (f *destroyFakeClient) DeleteServer(context.Context, int) error {
	f.calls = append(f.calls, "delete:server")
	return f.deleteErr["server"]
}
func (f *destroyFakeClient) DeleteSSHKey(context.Context, int) error {
	f.calls = append(f.calls, "delete:ssh_key")
	return f.deleteErr["ssh_key"]
}
func (f *destroyFakeClient) DeleteFirewall(context.Context, int) error {
	f.calls = append(f.calls, "delete:firewall")
	return f.deleteErr["firewall"]
}
func (f *destroyFakeClient) DeletePrimaryIP(context.Context, int) error {
	f.calls = append(f.calls, "delete:primary")
	return f.deleteErr["primary"]
}

// Bug 23 (Plan 06-10): minimal detach/unassign stubs so the fake
// satisfies the extended Client interface. The original ordering tests
// (TestDestroyDeletesThenVerifyDescribesBeforeSuccess,
// TestVerifyDestroyedReportsBillableResourcesStillFound,
// TestDestroyTreatsAlreadyAbsentAsSkippedSuccess) use destroyRef()
// which only carries primary_ipv4 (no IPv6, no server in those flows
// where we'd skip detach), so these stubs simply record the call for
// debugging without changing the existing assertions.
func (f *destroyFakeClient) DetachFirewallFromServer(context.Context, int, int) error {
	f.calls = append(f.calls, "detach:firewall")
	return f.deleteErr["detach_firewall"]
}
func (f *destroyFakeClient) UnassignPrimaryIP(context.Context, int) error {
	f.calls = append(f.calls, "unassign:primary")
	return f.deleteErr["unassign_primary"]
}

func destroyRef() state.ProviderRef {
	ids := map[string]string{"server": "101", "ssh_key": "202", "firewall": "303", "primary_ipv4": "404"}
	return state.ProviderRef{Kind: "hetzner", Name: "hetzner", ResourceIDs: ids, IDs: ids}
}

func TestDestroyDeletesThenVerifyDescribesBeforeSuccess(t *testing.T) {
	client := &destroyFakeClient{deleteErr: map[string]error{}}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	result, err := p.Destroy(context.Background(), destroyRef())
	if err != nil || result.Partial {
		t.Fatalf("Destroy partial=%v err=%v result=%#v", result.Partial, err, result)
	}
	verification, err := p.VerifyDestroyed(context.Background(), destroyRef())
	if err != nil || !verification.OK {
		t.Fatalf("VerifyDestroyed ok=%v err=%v verification=%#v", verification.OK, err, verification)
	}
	// Bug 26 (Plan 06-11, 2026-05-06): destroy detaches the firewall
	// from the server, then deletes the server (cascade-deletes
	// auto_delete=true primary IPs), then deletes ssh_key + primary IPs
	// (already absent via cascade — 404-tolerant) + firewall. NO
	// unassign step, because Hetzner rejects unassign with
	// `Server must be offline for this action` and the cascade handles
	// detachment without requiring power-off.
	want := []string{
		"detach:firewall",
		"delete:server",
		"delete:ssh_key",
		"delete:primary",
		"delete:firewall",
		"get:server",
		"get:ssh_key",
		"get:firewall",
		"get:primary",
	}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("calls=%#v want=%#v", client.calls, want)
	}
}

func TestVerifyDestroyedReportsBillableResourcesStillFound(t *testing.T) {
	client := &destroyFakeClient{found: true, deleteErr: map[string]error{}}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	verification, err := p.VerifyDestroyed(context.Background(), destroyRef())
	if err != nil || verification.OK || !strings.Contains(strings.Join(verification.BillableResources, ","), "server:101") {
		t.Fatalf("expected billable resources, got %#v err=%v", verification, err)
	}
}

func TestDestroyTreatsAlreadyAbsentAsSkippedSuccess(t *testing.T) {
	client := &destroyFakeClient{deleteErr: map[string]error{"server": errors.New("404 not found")}}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	result, err := p.Destroy(context.Background(), destroyRef())
	if err != nil || result.Partial {
		t.Fatalf("already-absent delete should not be partial: %#v err=%v", result, err)
	}
}

// Bug 26 (Plan 06-11, 2026-05-06): cloud destroy now relies on
// Hetzner's `auto_delete=true` cascade for primary IPs created with the
// server (the default for ServerCreatePublicNet EnableIPv4/EnableIPv6).
//
// Plan 06-10 Bug 23 added a manual `unassign` step before server.Delete
// because firewall.Delete and primary_ip.Delete reject with
// `resource_in_use` / `must_be_unassigned` while still attached. The
// unassign step turned out to require the server to be powered off
// (`Server must be offline for this action (server_not_stopped)` —
// verified live 2026-05-06 against server 129595285). Hetzner cascades
// auto_delete primary IPs on server deletion, so we drop the unassign
// step entirely. Firewall detach STILL runs first because firewalls
// are not part of the auto_delete cascade and detach has no power-off
// requirement.
//
// The expected end-to-end call order is:
//
//  1. firewall.RemoveResources(server)   — detach firewall from server
//  2. server.Delete                       — delete server (cascade-deletes IPs)
//  3. ssh_key.Delete                      — free, no ordering risk
//  4. primary_ipv4.Delete                 — already absent via cascade (404 → silent)
//  5. primary_ipv6.Delete                 — already absent via cascade (404 → silent)
//  6. firewall.Delete                     — last (now detached + free)
//
// No `unassign:*` calls anywhere. The test renamed to reflect the new
// invariant, but the old name is intentionally retained so future
// `git log -S "BeforeServerDelete"` greps still surface this commit.
func TestDestroy_AutoDeleteCascadeNoUnassign(t *testing.T) {
	client := &destroyFakeOrderedClient{}
	ref := destroyRefWithBothPrimaryIPs()
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	result, err := p.Destroy(context.Background(), ref)
	if err != nil || result.Partial {
		t.Fatalf("Destroy partial=%v err=%v result=%#v calls=%v", result.Partial, err, result, client.calls)
	}
	want := []string{
		"detach:firewall",
		"delete:server",
		"delete:ssh_key",
		"delete:primary_ipv4",
		"delete:primary_ipv6",
		"delete:firewall",
	}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("destroy call order mismatch:\n got %#v\nwant %#v", client.calls, want)
	}
	// Stronger Bug 26 guarantee: NO unassign:* call anywhere — the live
	// `server_not_stopped` failure mode is impossible because we never
	// invoke UnassignPrimaryIP.
	for _, call := range client.calls {
		if strings.HasPrefix(call, "unassign:") {
			t.Fatalf("Bug 26: destroy must NOT issue unassign:* (auto_delete cascade); got call %q in %v", call, client.calls)
		}
	}
}

// When detach steps return 404 (already absent — e.g. server was
// already deleted out-of-band), destroy must keep going and complete
// the remaining cleanup, NOT stall in partial state.
func TestDestroyTreatsAlreadyAbsentDetachAsSuccess(t *testing.T) {
	client := &destroyFakeOrderedClient{
		detachErr: map[string]error{
			"firewall": errors.New("404 not found"),
		},
	}
	ref := destroyRefWithBothPrimaryIPs()
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	result, err := p.Destroy(context.Background(), ref)
	if err != nil || result.Partial {
		t.Fatalf("already-absent detach should not be partial: result=%#v err=%v calls=%v", result, err, client.calls)
	}
}

// destroyRefWithBothPrimaryIPs is the cloud destroy state used by Bug
// 23 tests: server + ssh_key + firewall + IPv4 + IPv6 all tracked.
func destroyRefWithBothPrimaryIPs() state.ProviderRef {
	ids := map[string]string{
		"server":       "101",
		"ssh_key":      "202",
		"firewall":     "303",
		"primary_ipv4": "404",
		"primary_ipv6": "505",
	}
	return state.ProviderRef{Kind: "hetzner", Name: "hetzner", ResourceIDs: ids, IDs: ids}
}

// destroyFakeOrderedClient extends destroyFakeClient with the
// detach/unassign hooks Bug 23 requires.
type destroyFakeOrderedClient struct {
	calls     []string
	detachErr map[string]error
}

func (f *destroyFakeOrderedClient) GetLocation(context.Context, string) (*hcloud.Location, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) GetServerType(context.Context, string) (*hcloud.ServerType, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) GetImage(context.Context, string) (*hcloud.Image, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) CreateSSHKey(context.Context, hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) CreateFirewall(context.Context, hcloud.FirewallCreateOpts) (*hcloud.Firewall, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) CreateServer(context.Context, hcloud.ServerCreateOpts) (*hcloud.Server, *hcloud.Action, error) {
	return nil, nil, nil
}
func (f *destroyFakeOrderedClient) WaitForAction(context.Context, *hcloud.Action) error {
	return nil
}
func (f *destroyFakeOrderedClient) GetServer(context.Context, int) (*hcloud.Server, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) GetSSHKey(context.Context, int) (*hcloud.SSHKey, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) GetFirewall(context.Context, int) (*hcloud.Firewall, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) GetPrimaryIP(context.Context, int) (*hcloud.PrimaryIP, error) {
	return nil, nil
}
func (f *destroyFakeOrderedClient) DeleteServer(context.Context, int) error {
	f.calls = append(f.calls, "delete:server")
	return nil
}
func (f *destroyFakeOrderedClient) DeleteSSHKey(context.Context, int) error {
	f.calls = append(f.calls, "delete:ssh_key")
	return nil
}
func (f *destroyFakeOrderedClient) DeleteFirewall(context.Context, int) error {
	f.calls = append(f.calls, "delete:firewall")
	return nil
}
func (f *destroyFakeOrderedClient) DeletePrimaryIP(context.Context, int) error {
	// Distinguish IPv4 vs IPv6 deletes by call sequence: tests assert
	// the V4-then-V6 ordering.
	if !sliceContains(f.calls, "delete:primary_ipv4") {
		f.calls = append(f.calls, "delete:primary_ipv4")
	} else {
		f.calls = append(f.calls, "delete:primary_ipv6")
	}
	return nil
}
func (f *destroyFakeOrderedClient) DetachFirewallFromServer(_ context.Context, firewallID int, serverID int) error {
	_ = firewallID
	_ = serverID
	f.calls = append(f.calls, "detach:firewall")
	return f.detachErr["firewall"]
}
func (f *destroyFakeOrderedClient) UnassignPrimaryIP(_ context.Context, id int) error {
	// Distinguish IPv4 vs IPv6 unassigns by tag in calls.
	if !sliceContains(f.calls, "unassign:primary_ipv4") {
		f.calls = append(f.calls, "unassign:primary_ipv4")
		return f.detachErr["primary_ipv4"]
	}
	f.calls = append(f.calls, "unassign:primary_ipv6")
	return f.detachErr["primary_ipv6"]
}

func sliceContains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
