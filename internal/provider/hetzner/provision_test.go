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

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/salar/runnerkit/internal/provider"
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
func (f *fakeClient) GetPrimaryIP(context.Context, int) (*hcloud.PrimaryIP, error) { return nil, nil }
func (f *fakeClient) DeleteServer(context.Context, int) error                      { return nil }
func (f *fakeClient) DeleteSSHKey(context.Context, int) error                      { return nil }
func (f *fakeClient) DeleteFirewall(context.Context, int) error                    { return nil }
func (f *fakeClient) DeletePrimaryIP(context.Context, int) error                   { return nil }

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
