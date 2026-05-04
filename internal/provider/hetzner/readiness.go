package hetzner

import (
	"context"
	"fmt"
	"time"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
)

var readinessPollInterval = 500 * time.Millisecond
var readinessMaxAttempts = 60

// WaitReady waits for Hetzner-provider readiness only: create action completion,
// server running state, and assigned public networking. SSH/cloud-init/preflight
// readiness stays in the CLI layer.
func (p *Provider) WaitReady(ctx context.Context, machine provider.Machine) (provider.Machine, error) {
	client, _, err := p.client()
	if err != nil {
		return machine, err
	}
	ids := cloneIDs(machine.Provider.IDs)
	for k, v := range machine.Provider.ResourceIDs {
		ids[k] = v
	}
	for k, v := range machine.ResourceIDs {
		ids[k] = v
	}
	if actionID := ids["create_action"]; actionID != "" {
		parsed, err := parseID(actionID)
		if err != nil {
			return machine, fmt.Errorf("invalid Hetzner create action id %q: %w", actionID, err)
		}
		if err := client.WaitForAction(ctx, &hcloud.Action{ID: parsed}); err != nil {
			return machine, err
		}
	}
	serverID, err := parseID(ids["server"])
	if err != nil || serverID == 0 {
		return machine, fmt.Errorf("missing Hetzner server id for readiness")
	}
	attempts := readinessMaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		server, err := client.GetServer(ctx, serverID)
		if err != nil {
			return machine, err
		}
		if server != nil && server.Status == hcloud.ServerStatusRunning {
			ipv4, ipv6 := publicIPs(server)
			if ipv4 != "" || ipv6 != "" {
				return updateMachineFromReadyServer(machine, server), nil
			}
		}
		if attempt == attempts-1 {
			break
		}
		if readinessPollInterval > 0 {
			select {
			case <-ctx.Done():
				return machine, ctx.Err()
			case <-time.After(readinessPollInterval):
			}
		}
	}
	return machine, fmt.Errorf("Hetzner server %d is not running with a public IP yet", serverID)
}

func updateMachineFromReadyServer(machine provider.Machine, server *hcloud.Server) provider.Machine {
	ipv4, ipv6 := publicIPs(server)
	host := ipv4
	if host == "" {
		host = ipv6
	}
	machine.Target.Host = host
	if machine.Target.User == "" {
		machine.Target.User = defaultSSHUser
	}
	if machine.Target.Port == 0 {
		machine.Target.Port = 22
	}
	machine.Target.Raw = machine.Target.Display()
	machine.PublicIPv4 = ipv4
	machine.PublicIPv6 = ipv6
	if machine.ResourceIDs == nil {
		machine.ResourceIDs = map[string]string{}
	}
	addPublicNetResourceIDs(machine.ResourceIDs, server)
	if machine.Provider.IDs == nil {
		machine.Provider.IDs = map[string]string{}
	}
	addPublicNetResourceIDs(machine.Provider.IDs, server)
	if machine.Provider.ResourceIDs == nil {
		machine.Provider.ResourceIDs = map[string]string{}
	}
	addPublicNetResourceIDs(machine.Provider.ResourceIDs, server)
	machine.Provider.Cloud.ServerStatus = string(server.Status)
	machine.Provider.Cloud.PublicIPv4 = ipv4
	machine.Provider.Cloud.PublicIPv6 = ipv6
	machine.Provider.Cloud.PrimaryIPv4ID = machine.ResourceIDs["primary_ipv4"]
	machine.Provider.Cloud.PrimaryIPv6ID = machine.ResourceIDs["primary_ipv6"]
	return machine
}
