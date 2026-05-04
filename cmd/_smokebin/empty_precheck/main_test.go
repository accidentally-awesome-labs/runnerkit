package main

import (
	"context"
	"strings"
	"testing"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

// fakeClient implements hcloudClient for unit tests; lets each test
// inject a different starting inventory of Hetzner resources.
type fakeClient struct {
	servers    []*hcloud.Server
	sshKeys    []*hcloud.SSHKey
	primaryIPs []*hcloud.PrimaryIP
	firewalls  []*hcloud.Firewall
}

func (f *fakeClient) AllServers(_ context.Context) ([]*hcloud.Server, error) {
	return f.servers, nil
}

func (f *fakeClient) AllSSHKeys(_ context.Context) ([]*hcloud.SSHKey, error) {
	return f.sshKeys, nil
}

func (f *fakeClient) AllPrimaryIPs(_ context.Context) ([]*hcloud.PrimaryIP, error) {
	return f.primaryIPs, nil
}

func (f *fakeClient) AllFirewalls(_ context.Context) ([]*hcloud.Firewall, error) {
	return f.firewalls, nil
}

// TestEmptyPrecheck_RefusesOnExisting locks D-12 gate 1: when a single
// runnerkit-* server exists, run() must refuse and the error must name
// the offending resource. When no runnerkit-* resource exists the
// function must return nil.
func TestEmptyPrecheck_RefusesOnExisting(t *testing.T) {
	t.Run("refuses when a runnerkit-* server is present", func(t *testing.T) {
		client := &fakeClient{
			servers: []*hcloud.Server{
				{ID: 1, Name: "runnerkit-test-server"},
			},
		}
		err := run(context.Background(), client)
		if err == nil {
			t.Fatal("expected error when runnerkit-test-server exists; got nil")
		}
		if !strings.Contains(err.Error(), "runnerkit-test-server") {
			t.Fatalf("error must name the offending resource; got %q", err.Error())
		}
	})

	t.Run("allows when only unrelated resources exist", func(t *testing.T) {
		client := &fakeClient{
			servers: []*hcloud.Server{
				{ID: 2, Name: "unrelated-server"},
			},
		}
		if err := run(context.Background(), client); err != nil {
			t.Fatalf("expected nil for unrelated-server; got %v", err)
		}
	})

	t.Run("allows on a fully empty project", func(t *testing.T) {
		client := &fakeClient{}
		if err := run(context.Background(), client); err != nil {
			t.Fatalf("expected nil on empty project; got %v", err)
		}
	})
}

// TestEmptyPrecheck_AllResourceTypes locks the D-12 gate 1 contract that
// the precheck scans ALL FOUR resource types Phase 4 creates: Server,
// SSHKey, PrimaryIP, Firewall. Each type's runnerkit-* prefix must
// independently trip the gate.
func TestEmptyPrecheck_AllResourceTypes(t *testing.T) {
	cases := []struct {
		name       string
		client     *fakeClient
		wantSubstr string
	}{
		{
			name: "ssh key with runnerkit- prefix refuses",
			client: &fakeClient{
				sshKeys: []*hcloud.SSHKey{
					{ID: 10, Name: "runnerkit-test-key"},
				},
			},
			wantSubstr: "runnerkit-test-key",
		},
		{
			name: "primary ip with runnerkit- prefix refuses",
			client: &fakeClient{
				primaryIPs: []*hcloud.PrimaryIP{
					{ID: 20, Name: "runnerkit-test-ip"},
				},
			},
			wantSubstr: "runnerkit-test-ip",
		},
		{
			name: "firewall with runnerkit- prefix refuses",
			client: &fakeClient{
				firewalls: []*hcloud.Firewall{
					{ID: 30, Name: "runnerkit-test-fw"},
				},
			},
			wantSubstr: "runnerkit-test-fw",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(context.Background(), tc.client)
			if err == nil {
				t.Fatalf("expected error for %s; got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("error must mention %q; got %q", tc.wantSubstr, err.Error())
			}
		})
	}
}
