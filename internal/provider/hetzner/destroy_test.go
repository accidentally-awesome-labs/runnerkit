package hetzner

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
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
	want := []string{"delete:server", "delete:ssh_key", "delete:firewall", "delete:primary", "get:server", "get:ssh_key", "get:firewall", "get:primary"}
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
