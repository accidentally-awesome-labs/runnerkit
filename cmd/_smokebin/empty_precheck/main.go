// Command empty_precheck is the RED-phase stub for D-12 gate 1.
// The real implementation lands in the GREEN commit.
package main

import (
	"context"
	"errors"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

func main() {}

type hcloudClient interface {
	AllServers(ctx context.Context) ([]*hcloud.Server, error)
	AllSSHKeys(ctx context.Context) ([]*hcloud.SSHKey, error)
	AllPrimaryIPs(ctx context.Context) ([]*hcloud.PrimaryIP, error)
	AllFirewalls(ctx context.Context) ([]*hcloud.Firewall, error)
}

func run(ctx context.Context, client hcloudClient) error {
	return errors.New("empty_precheck: not implemented (RED stub)")
}
