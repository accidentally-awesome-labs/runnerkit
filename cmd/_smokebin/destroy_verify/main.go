// Command destroy_verify is the RED-phase stub for D-12 gate 2.
// The real implementation lands in the GREEN commit.
package main

import (
	"context"
	"errors"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

func main() {}

type verifierClient interface {
	GetServerByID(ctx context.Context, id int) (*hcloud.Server, error)
	GetSSHKeyByID(ctx context.Context, id int) (*hcloud.SSHKey, error)
	GetPrimaryIPByID(ctx context.Context, id int) (*hcloud.PrimaryIP, error)
	GetFirewallByID(ctx context.Context, id int) (*hcloud.Firewall, error)
}

func run(ctx context.Context, client verifierClient) error {
	return errors.New("destroy_verify: not implemented (RED stub)")
}
