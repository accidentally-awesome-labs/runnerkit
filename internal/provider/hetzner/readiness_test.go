package hetzner

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
)

func TestWaitReadyWaitsForActionRunningStatusAndPublicIP(t *testing.T) {
	oldInterval := readinessPollInterval
	oldAttempts := readinessMaxAttempts
	readinessPollInterval = 0
	readinessMaxAttempts = 1
	defer func() { readinessPollInterval, readinessMaxAttempts = oldInterval, oldAttempts }()
	client := newFakeClient()
	client.server = &hcloud.Server{
		ID:     303,
		Status: hcloud.ServerStatusRunning,
		PublicNet: hcloud.ServerPublicNet{
			IPv4: hcloud.ServerPublicNetIPv4{ID: 404, IP: net.ParseIP("203.0.113.10")},
		},
	}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	machine := provider.Machine{
		Target:      remote.Target{User: "runnerkit-admin", Port: 22},
		ResourceIDs: map[string]string{"server": "303", "create_action": "606"},
		Provider:    state.ProviderRef{Kind: "hetzner", IDs: map[string]string{"server": "303", "create_action": "606"}},
	}
	ready, err := p.WaitReady(context.Background(), machine)
	if err != nil {
		t.Fatalf("WaitReady returned error: %v", err)
	}
	if len(client.waitActionIDs) != 1 || client.waitActionIDs[0] != 606 {
		t.Fatalf("WaitForAction IDs = %#v", client.waitActionIDs)
	}
	if len(client.getServerIDs) != 1 || client.getServerIDs[0] != 303 {
		t.Fatalf("GetServer IDs = %#v", client.getServerIDs)
	}
	if ready.Target.Host != "203.0.113.10" || ready.PublicIPv4 != "203.0.113.10" || ready.Provider.Cloud.ServerStatus != "running" || ready.ResourceIDs["primary_ipv4"] != "404" {
		t.Fatalf("unexpected ready machine: %#v", ready)
	}
}

func TestWaitReadyRequiresRunningServerWithPublicIP(t *testing.T) {
	oldInterval := readinessPollInterval
	oldAttempts := readinessMaxAttempts
	readinessPollInterval = 0
	readinessMaxAttempts = 1
	defer func() { readinessPollInterval, readinessMaxAttempts = oldInterval, oldAttempts }()
	client := newFakeClient()
	client.server = &hcloud.Server{ID: 303, Status: hcloud.ServerStatusInitializing}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	_, err := p.WaitReady(context.Background(), provider.Machine{ResourceIDs: map[string]string{"server": "303"}})
	if err == nil || !strings.Contains(err.Error(), "not running with a public IP") {
		t.Fatalf("expected public IP readiness error, got %v", err)
	}
}

func TestWaitReadyHonorsContextWhilePolling(t *testing.T) {
	oldInterval := readinessPollInterval
	oldAttempts := readinessMaxAttempts
	readinessPollInterval = time.Hour
	readinessMaxAttempts = 2
	defer func() { readinessPollInterval, readinessMaxAttempts = oldInterval, oldAttempts }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := newFakeClient()
	client.server = &hcloud.Server{ID: 303, Status: hcloud.ServerStatusInitializing}
	p := NewProvider(map[string]string{EnvHCLOUDToken: "fake-token"}, WithClient(client))
	_, err := p.WaitReady(ctx, provider.Machine{ResourceIDs: map[string]string{"server": "303"}})
	if err == nil {
		t.Fatal("expected canceled context")
	}
}
