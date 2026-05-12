package hetzner

import (
	"context"
	"errors"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

type fakeClient struct {
	calls []string

	location   *hcloud.Location
	serverType *hcloud.ServerType
	image      *hcloud.Image

	sshKey        *hcloud.SSHKey
	firewall      *hcloud.Firewall
	server        *hcloud.Server
	action        *hcloud.Action
	createSrvErr  error
	waitErr       error
	getServerErr  error
	validationErr map[string]error
	waitActionIDs []int
	getServerIDs  []int

	sshKeyOpts   hcloud.SSHKeyCreateOpts
	firewallOpts hcloud.FirewallCreateOpts
	serverOpts   hcloud.ServerCreateOpts
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		location:   &hcloud.Location{Name: "fsn1"},
		serverType: &hcloud.ServerType{Name: "cpx22"},
		image:      &hcloud.Image{Name: "ubuntu-24.04"},
		sshKey:     &hcloud.SSHKey{ID: 101, Name: "key", Fingerprint: "SHA256:sshkey"},
		firewall:   &hcloud.Firewall{ID: 202, Name: "fw"},
		server: &hcloud.Server{
			ID:     303,
			Name:   "server",
			Status: hcloud.ServerStatusRunning,
			PublicNet: hcloud.ServerPublicNet{
				IPv4: hcloud.ServerPublicNetIPv4{ID: 404, IP: net.ParseIP("203.0.113.10")},
				IPv6: hcloud.ServerPublicNetIPv6{ID: 505, IP: net.ParseIP("2001:db8::10")},
			},
		},
		action: &hcloud.Action{ID: 606},
	}
}

func (f *fakeClient) GetLocation(context.Context, string) (*hcloud.Location, error) {
	f.calls = append(f.calls, "lookup:location")
	if f.validationErr != nil && f.validationErr["location"] != nil {
		return nil, f.validationErr["location"]
	}
	return f.location, nil
}
func (f *fakeClient) GetServerType(context.Context, string) (*hcloud.ServerType, error) {
	f.calls = append(f.calls, "lookup:server_type")
	if f.validationErr != nil && f.validationErr["server_type"] != nil {
		return nil, f.validationErr["server_type"]
	}
	return f.serverType, nil
}
func (f *fakeClient) GetImage(context.Context, string) (*hcloud.Image, error) {
	f.calls = append(f.calls, "lookup:image")
	if f.validationErr != nil && f.validationErr["image"] != nil {
		return nil, f.validationErr["image"]
	}
	return f.image, nil
}
func (f *fakeClient) CreateSSHKey(_ context.Context, opts hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error) {
	f.calls = append(f.calls, "ssh_key")
	f.sshKeyOpts = opts
	return f.sshKey, nil
}
func (f *fakeClient) CreateFirewall(_ context.Context, opts hcloud.FirewallCreateOpts) (*hcloud.Firewall, error) {
	f.calls = append(f.calls, "firewall")
	f.firewallOpts = opts
	return f.firewall, nil
}
func (f *fakeClient) CreateServer(_ context.Context, opts hcloud.ServerCreateOpts) (*hcloud.Server, *hcloud.Action, error) {
	f.calls = append(f.calls, "server")
	f.serverOpts = opts
	return f.server, f.action, f.createSrvErr
}
func (f *fakeClient) WaitForAction(_ context.Context, action *hcloud.Action) error {
	if action != nil {
		f.waitActionIDs = append(f.waitActionIDs, action.ID)
	}
	return f.waitErr
}
func (f *fakeClient) GetServer(_ context.Context, id int) (*hcloud.Server, error) {
	f.getServerIDs = append(f.getServerIDs, id)
	return f.server, f.getServerErr
}
func (f *fakeClient) GetSSHKey(context.Context, int) (*hcloud.SSHKey, error) { return f.sshKey, nil }
func (f *fakeClient) GetFirewall(context.Context, int) (*hcloud.Firewall, error) {
	return f.firewall, nil
}
func (f *fakeClient) GetPrimaryIP(context.Context, int) (*hcloud.PrimaryIP, error) { return nil, nil }
func (f *fakeClient) DeleteServer(context.Context, int) error                      { return nil }
func (f *fakeClient) DeleteSSHKey(context.Context, int) error                      { return nil }
func (f *fakeClient) DeleteFirewall(context.Context, int) error                    { return nil }
func (f *fakeClient) DeletePrimaryIP(context.Context, int) error                   { return nil }
func (f *fakeClient) DetachFirewallFromServer(context.Context, int, int) error     { return nil }
func (f *fakeClient) UnassignPrimaryIP(context.Context, int) error                 { return nil }

func TestProvisionCreatesResourcesInOrderWithDefaultProfileAndTags(t *testing.T) {
	t.Setenv("HCLOUD_TOKEN", "") // fake client plus explicit env token means tests do not depend on a live shell token.
	if os.Getenv("HCLOUD_TOKEN") != "" {
		t.Fatal("test unexpectedly depends on live HCLOUD_TOKEN")
	}
	client := newFakeClient()
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	input := provisionInput()
	result, err := p.Provision(context.Background(), input)
	if err != nil {
		t.Fatalf("Provision returned error: %v", err)
	}
	wantOrder := []string{"lookup:location", "lookup:server_type", "lookup:image", "ssh_key", "firewall", "server"}
	if !reflect.DeepEqual(client.calls, wantOrder) {
		t.Fatalf("call order = %#v, want %#v", client.calls, wantOrder)
	}
	if gotCreates := createCalls(client.calls); !reflect.DeepEqual(gotCreates, []string{"ssh_key", "firewall", "server"}) {
		t.Fatalf("create order = %#v", gotCreates)
	}
	for _, opts := range []map[string]string{client.sshKeyOpts.Labels, client.firewallOpts.Labels, client.serverOpts.Labels} {
		for key, want := range map[string]string{"runnerkit": "true", "managed": "true", "repo": "owner-name", "runner": "runnerkit-owner-name", "state_id": "state-123", "mode": "persistent"} {
			if got := opts[key]; got != want {
				t.Fatalf("label %s = %q, want %q in %#v", key, got, want, opts)
			}
		}
	}
	if client.sshKeyOpts.PublicKey != input.PublicKey {
		t.Fatalf("SSH public key = %q", client.sshKeyOpts.PublicKey)
	}
	if client.firewallOpts.Rules[0].Direction != hcloud.FirewallRuleDirectionIn || client.firewallOpts.Rules[0].Protocol != hcloud.FirewallRuleProtocolTCP || *client.firewallOpts.Rules[0].Port != "22" {
		t.Fatalf("unexpected firewall rule: %#v", client.firewallOpts.Rules[0])
	}
	if got := client.firewallOpts.Rules[0].SourceIPs[0].String(); got != "203.0.113.0/24" {
		t.Fatalf("firewall source CIDR = %q", got)
	}
	if !strings.Contains(client.serverOpts.UserData, "runnerkit-admin") || !strings.Contains(client.serverOpts.UserData, input.PublicKey) {
		t.Fatalf("cloud-init user-data missing runnerkit-admin or public key:\n%s", client.serverOpts.UserData)
	}
	ud := client.serverOpts.UserData
	for _, frag := range []string{
		"write_files:",
		"/var/lib/runnerkit/installer.sudoers.staged",
		"visudo -cf /var/lib/runnerkit/installer.sudoers.staged",
		"/etc/sudoers.d/runnerkit-installer",
		"ALL=(root) NOPASSWD:",
		"runnerkit-cloud-init-v2",
	} {
		if !strings.Contains(ud, frag) {
			t.Fatalf("cloud-init user-data missing %q:\n%s", frag, ud)
		}
	}
	if client.serverOpts.ServerType.Name != "cpx22" || client.serverOpts.Image.Name != "ubuntu-24.04" || client.serverOpts.Location.Name != "fsn1" {
		t.Fatalf("unexpected server profile: %#v", client.serverOpts)
	}
	if client.serverOpts.PublicNet == nil || !client.serverOpts.PublicNet.EnableIPv4 || !client.serverOpts.PublicNet.EnableIPv6 {
		t.Fatalf("public IPv4/IPv6 were not enabled: %#v", client.serverOpts.PublicNet)
	}
	if client.serverOpts.Volumes != nil || client.serverOpts.Automount == nil || *client.serverOpts.Automount || len(client.serverOpts.Firewalls) != 1 {
		t.Fatalf("unexpected optional resources requested: volumes=%#v automount=%#v firewalls=%d", client.serverOpts.Volumes, client.serverOpts.Automount, len(client.serverOpts.Firewalls))
	}
	for key, want := range map[string]string{"ssh_key": "101", "firewall": "202", "server": "303", "primary_ipv4": "404", "primary_ipv6": "505", "create_action": "606"} {
		if got := result.CreatedResourceIDs[key]; got != want {
			t.Fatalf("resource id %s = %q, want %q (ids %#v)", key, got, want, result.CreatedResourceIDs)
		}
	}
	if !result.CheckpointRequired || result.Machine.Target.User != "runnerkit-admin" || result.Machine.Target.Host != "203.0.113.10" || result.Machine.Provider.Kind != "hetzner" || result.Machine.Provider.Region != "fsn1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

// Bug 26 (Plan 06-11, 2026-05-06): destroy.go's cascade-delete approach
// requires Hetzner to auto-allocate primary IPs with `AutoDelete: true`
// — the default for ServerCreatePublicNet when EnableIPv4 / EnableIPv6
// is true and no explicit IPv4/IPv6 *PrimaryIP override is provided.
// If a future change passes `IPv4: &PrimaryIP{...}` (or `IPv6`), the
// cascade is broken — the primary IPs survive server.Delete and
// `runnerkit destroy` falls into the live `Server must be offline for
// this action (server_not_stopped)` path on the unassign step that
// destroy.go relies on auto-cascade to skip.
//
// Regression guard: assert provision sets EnableIPv4=true,
// EnableIPv6=true, IPv4=nil, IPv6=nil. The fakeClient captures
// ServerCreateOpts so we can inspect PublicNet directly.
func TestProvisionEnablesPublicIPsWithoutOverridingForBug26(t *testing.T) {
	client := newFakeClient()
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	if _, err := p.Provision(context.Background(), provisionInput()); err != nil {
		t.Fatalf("Provision returned error: %v", err)
	}
	pn := client.serverOpts.PublicNet
	if pn == nil {
		t.Fatal("PublicNet must be set so Hetzner auto-allocates primary IPs with AutoDelete: true")
	}
	if !pn.EnableIPv4 || !pn.EnableIPv6 {
		t.Fatalf("EnableIPv4/EnableIPv6 must both be true for Bug 26 cascade; got %#v", pn)
	}
	if pn.IPv4 != nil || pn.IPv6 != nil {
		t.Fatalf("Bug 26: provision must NOT pass explicit *PrimaryIP overrides — that breaks the auto_delete=true cascade destroy.go relies on; got IPv4=%#v IPv6=%#v", pn.IPv4, pn.IPv6)
	}
}

func TestProvisionErrorAfterServerCreationCarriesPartialResourceIDs(t *testing.T) {
	client := newFakeClient()
	client.createSrvErr = errors.New("server action failed after create")
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	_, err := p.Provision(context.Background(), provisionInput())
	if err == nil {
		t.Fatal("expected ProvisionError")
	}
	var provErr *provider.ProvisionError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProvisionError, got %T: %v", err, err)
	}
	if provErr.Stage != "server" || !provErr.Result.CheckpointRequired {
		t.Fatalf("unexpected provision error: %#v", provErr)
	}
	for key, want := range map[string]string{"ssh_key": "101", "firewall": "202", "server": "303"} {
		if got := provErr.Result.CreatedResourceIDs[key]; got != want {
			t.Fatalf("partial id %s = %q, want %q (ids %#v)", key, got, want, provErr.Result.CreatedResourceIDs)
		}
	}
}

func TestValidateLooksUpRecommendedProfileBeforeCreate(t *testing.T) {
	client := newFakeClient()
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	result, err := p.Validate(context.Background(), provisionInput())
	if err != nil || !result.OK {
		t.Fatalf("Validate = %#v, %v", result, err)
	}
	if !reflect.DeepEqual(client.calls, []string{"lookup:location", "lookup:server_type", "lookup:image"}) {
		t.Fatalf("Validate call order = %#v", client.calls)
	}
	if gotCreates := createCalls(client.calls); len(gotCreates) != 0 {
		t.Fatalf("Validate created resources: %#v", gotCreates)
	}
}

func TestValidateUnavailableImageReturnsRemediationWithoutCreate(t *testing.T) {
	client := newFakeClient()
	client.image = nil
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	result, err := p.Validate(context.Background(), provisionInput())
	if err != nil {
		t.Fatalf("Validate returned lookup error: %v", err)
	}
	if result.OK || !strings.Contains(strings.Join(result.Remediation, "\n"), "ubuntu-24.04") {
		t.Fatalf("unexpected validation result: %#v", result)
	}
	if gotCreates := createCalls(client.calls); len(gotCreates) != 0 {
		t.Fatalf("Validate created resources: %#v", gotCreates)
	}
}

// Bug 22 (Plan 06-10, 2026-05-06): cloud SSH host-key readiness probe
// must retry across the cloud-init host-key-install window (~30-90s on
// fresh Ubuntu 24.04 images). The previous single-shot probe failed
// with `SSH host key fingerprint was not observed` because cloud-init
// had not yet written /etc/ssh/ssh_host_*_key.pub by the time the
// probe ran. ProbeHostKeyWithRetry wraps a HostKeyProber with bounded
// retries + exponential-ish backoff and surfaces the last attempt's
// error if the budget is exhausted.
func TestProbeHostKeyWithRetrySucceedsOnLaterAttempt(t *testing.T) {
	prober := &fakeHostKeyProber{
		responses: []hostKeyResponse{
			{err: errors.New("SSH host key fingerprint was not observed")},
			{err: errors.New("SSH host key fingerprint was not observed")},
			{key: stubHostKey("SHA256:cloudinithostkey")},
		},
	}
	clock := &fakeClock{}
	hostKey, err := ProbeHostKeyWithRetry(context.Background(), prober, fakeTarget(), HostKeyProbeOptions{Attempts: 5, Interval: 100 * time.Millisecond, Sleep: clock.Sleep})
	if err != nil {
		t.Fatalf("expected eventual success, got err=%v after %d calls", err, prober.calls)
	}
	if hostKey.Fingerprint != "SHA256:cloudinithostkey" {
		t.Fatalf("unexpected hostKey fingerprint: %q", hostKey.Fingerprint)
	}
	if prober.calls != 3 {
		t.Fatalf("expected 3 probe attempts; got %d", prober.calls)
	}
	if len(clock.sleeps) != 2 {
		t.Fatalf("expected 2 backoff sleeps between attempts; got %d (%#v)", len(clock.sleeps), clock.sleeps)
	}
}

// When the retry budget is exhausted (cloud-init never writes a host
// key, or the IP is permanently unreachable), ProbeHostKeyWithRetry
// must return the LAST observed error so the caller can surface a
// useful diagnostic to the user.
func TestProbeHostKeyWithRetryReturnsLastErrorOnExhaustion(t *testing.T) {
	prober := &fakeHostKeyProber{
		responses: []hostKeyResponse{
			{err: errors.New("SSH host key fingerprint was not observed")},
			{err: errors.New("SSH host key fingerprint was not observed")},
			{err: errors.New("SSH host key fingerprint was not observed")},
		},
	}
	clock := &fakeClock{}
	hostKey, err := ProbeHostKeyWithRetry(context.Background(), prober, fakeTarget(), HostKeyProbeOptions{Attempts: 3, Interval: 50 * time.Millisecond, Sleep: clock.Sleep})
	if err == nil {
		t.Fatalf("expected error after exhaustion; got hostKey=%#v", hostKey)
	}
	if !strings.Contains(err.Error(), "SSH host key fingerprint was not observed") {
		t.Fatalf("expected last-attempt error surface; got %v", err)
	}
	if prober.calls != 3 {
		t.Fatalf("expected exactly 3 attempts at exhaustion; got %d", prober.calls)
	}
}

// An empty Fingerprint counts as a failure even if no error is
// returned — Cloud-init can return an empty ssh-keyscan result while
// the connection itself succeeds; retry until a real fingerprint
// appears.
func TestProbeHostKeyWithRetryRetriesOnEmptyFingerprint(t *testing.T) {
	prober := &fakeHostKeyProber{
		responses: []hostKeyResponse{
			{key: stubHostKey("")}, // empty
			{key: stubHostKey("SHA256:eventualhostkey")},
		},
	}
	clock := &fakeClock{}
	hostKey, err := ProbeHostKeyWithRetry(context.Background(), prober, fakeTarget(), HostKeyProbeOptions{Attempts: 4, Interval: 10 * time.Millisecond, Sleep: clock.Sleep})
	if err != nil {
		t.Fatalf("retry on empty fingerprint should eventually succeed; err=%v", err)
	}
	if hostKey.Fingerprint != "SHA256:eventualhostkey" {
		t.Fatalf("unexpected hostKey: %#v", hostKey)
	}
	if prober.calls != 2 {
		t.Fatalf("expected 2 attempts; got %d", prober.calls)
	}
}

// Defaults: empty Attempts/Interval should produce a usable retry
// budget out of the box (no caller required to set them).
func TestProbeHostKeyWithRetryUsesDefaultsWhenOptionsZero(t *testing.T) {
	prober := &fakeHostKeyProber{responses: []hostKeyResponse{{key: stubHostKey("SHA256:firsttry")}}}
	clock := &fakeClock{}
	_, err := ProbeHostKeyWithRetry(context.Background(), prober, fakeTarget(), HostKeyProbeOptions{Sleep: clock.Sleep})
	if err != nil {
		t.Fatalf("zero-value options should still succeed on first attempt: %v", err)
	}
	if prober.calls != 1 {
		t.Fatalf("expected 1 probe call; got %d", prober.calls)
	}
}

type hostKeyResponse struct {
	key remote.HostKey
	err error
}

type fakeHostKeyProber struct {
	responses []hostKeyResponse
	calls     int
}

func (f *fakeHostKeyProber) ProbeHostKey(_ context.Context, _ remote.Target) (remote.HostKey, error) {
	idx := f.calls
	f.calls++
	if idx >= len(f.responses) {
		idx = len(f.responses) - 1
	}
	return f.responses[idx].key, f.responses[idx].err
}

type fakeClock struct {
	sleeps []time.Duration
}

func (c *fakeClock) Sleep(d time.Duration) {
	c.sleeps = append(c.sleeps, d)
}

func stubHostKey(fingerprint string) remote.HostKey {
	return remote.HostKey{Algorithm: "ssh-ed25519", Fingerprint: fingerprint, PublicKey: []byte(fingerprint)}
}

func fakeTarget() remote.Target {
	return remote.Target{User: "runnerkit-admin", Host: "203.0.113.10", Port: 22, Raw: "runnerkit-admin@203.0.113.10:22"}
}

func provisionInput() provider.ProvisionInput {
	return provider.ProvisionInput{
		RepoFullName:    "owner/name",
		RunnerName:      "runnerkit-owner-name",
		Labels:          []string{"self-hosted", "runnerkit", "runnerkit-owner-name", "linux", "x64", "persistent"},
		WorkflowSnippet: "runs-on: [self-hosted, runnerkit, runnerkit-owner-name, linux, x64, persistent]",
		Profile:         provider.DefaultHetznerProfile(),
		SSHAllowedCIDR:  "203.0.113.0/24",
		PublicKey:       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakerunnerkit runnerkit@example",
		StateID:         "state-123",
		CreatedAt:       time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	}
}

func createCalls(calls []string) []string {
	out := []string{}
	for _, call := range calls {
		if call == "ssh_key" || call == "firewall" || call == "server" {
			out = append(out, call)
		}
	}
	return out
}
