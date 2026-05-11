package hetzner

import (
	"context"
	"errors"
	"net"
	"testing"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

func TestDescribeReturnsFoundWhenServerExists(t *testing.T) {
	t.Setenv("HCLOUD_TOKEN", "fake")
	client := newFakeClient()
	client.server = &hcloud.Server{
		ID:     303,
		Status: hcloud.ServerStatusRunning,
		Datacenter: &hcloud.Datacenter{
			Location: &hcloud.Location{Name: "fsn1"},
		},
		ServerType: &hcloud.ServerType{Name: "cpx22"},
		Image:      &hcloud.Image{Name: "ubuntu-24.04"},
		PublicNet: hcloud.ServerPublicNet{
			IPv4: hcloud.ServerPublicNetIPv4{ID: 404, IP: net.ParseIP("203.0.113.10")},
		},
	}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake"}, WithClient(client))
	ref := state.ProviderRef{
		Kind:   provider.HetznerProvider,
		Region: "nbg1",
		ResourceIDs: map[string]string{
			"server":        "303",
			"ssh_key":       "101",
			"firewall":      "202",
			"primary_ipv4":  "404",
			"primary_ipv6":  "505",
		},
	}
	st, err := p.Describe(context.Background(), ref)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if !st.Found || st.Status != "running" || st.PublicHost != "203.0.113.10" {
		t.Fatalf("unexpected status: %#v", st)
	}
	if st.Region != "fsn1" || st.ServerType != "cpx22" || st.Image != "ubuntu-24.04" {
		t.Fatalf("unexpected metadata: %#v", st)
	}
	if len(st.BillableResources) != 5 {
		t.Fatalf("billable resources: %#v", st.BillableResources)
	}
}

func TestDescribeReturnsNotFoundWhenServerAbsent(t *testing.T) {
	t.Setenv("HCLOUD_TOKEN", "fake")
	client := newFakeClient()
	client.server = nil
	client.getServerErr = errors.New("404 Not Found")
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake"}, WithClient(client))
	ref := state.ProviderRef{Kind: provider.HetznerProvider, ResourceIDs: map[string]string{"server": "999"}}
	st, err := p.Describe(context.Background(), ref)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if st.Found {
		t.Fatalf("expected Found=false: %#v", st)
	}
}

func TestDescribeEmptyServerID(t *testing.T) {
	t.Setenv("HCLOUD_TOKEN", "fake")
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake"}, WithClient(newFakeClient()))
	ref := state.ProviderRef{Kind: provider.HetznerProvider, ResourceIDs: map[string]string{}}
	st, err := p.Describe(context.Background(), ref)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if st.Found {
		t.Fatalf("expected Found=false: %#v", st)
	}
}
