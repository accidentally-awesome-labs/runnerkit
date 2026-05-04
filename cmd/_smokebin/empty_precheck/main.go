// Command empty_precheck implements D-12 gate 1: refuse the live cloud
// smoke if the configured Hetzner project contains any pre-existing
// `runnerkit-*` managed servers, ssh-keys, primary-ips, or firewalls.
//
// Phase 4 creates exactly these four resource types per `runnerkit up
// --cloud hetzner`; finding any with the `runnerkit-` Name prefix means
// a previous smoke leaked or a previous up failed mid-provision.
//
// This binary is excluded from `go build ./...` by the `_smokebin`
// directory's `_` prefix (Go convention) and is invoked by
// scripts/smoke/hetzner-empty-precheck.sh via `go run`.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

const namePrefix = "runnerkit-"

// hcloudClient is the subset of *hcloud.Client we use; lets tests inject a fake.
type hcloudClient interface {
	AllServers(ctx context.Context) ([]*hcloud.Server, error)
	AllSSHKeys(ctx context.Context) ([]*hcloud.SSHKey, error)
	AllPrimaryIPs(ctx context.Context) ([]*hcloud.PrimaryIP, error)
	AllFirewalls(ctx context.Context) ([]*hcloud.Firewall, error)
}

type realClient struct{ c *hcloud.Client }

func (r realClient) AllServers(ctx context.Context) ([]*hcloud.Server, error) {
	return r.c.Server.All(ctx)
}

func (r realClient) AllSSHKeys(ctx context.Context) ([]*hcloud.SSHKey, error) {
	return r.c.SSHKey.All(ctx)
}

func (r realClient) AllPrimaryIPs(ctx context.Context) ([]*hcloud.PrimaryIP, error) {
	return r.c.PrimaryIP.All(ctx)
}

func (r realClient) AllFirewalls(ctx context.Context) ([]*hcloud.Firewall, error) {
	return r.c.Firewall.All(ctx)
}

func main() {
	if err := run(context.Background(), nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, client hcloudClient) error {
	if client == nil {
		token := os.Getenv("HCLOUD_TOKEN")
		if token == "" {
			return fmt.Errorf("HCLOUD_TOKEN required")
		}
		client = realClient{c: hcloud.NewClient(hcloud.WithToken(token))}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var orphans []string

	servers, err := client.AllServers(ctx)
	if err != nil {
		return fmt.Errorf("list servers: %w", err)
	}
	for _, s := range servers {
		if strings.HasPrefix(s.Name, namePrefix) {
			orphans = append(orphans, fmt.Sprintf("server: %s (id %d)", s.Name, s.ID))
		}
	}

	keys, err := client.AllSSHKeys(ctx)
	if err != nil {
		return fmt.Errorf("list ssh keys: %w", err)
	}
	for _, k := range keys {
		if strings.HasPrefix(k.Name, namePrefix) {
			orphans = append(orphans, fmt.Sprintf("ssh-key: %s (id %d)", k.Name, k.ID))
		}
	}

	ips, err := client.AllPrimaryIPs(ctx)
	if err != nil {
		return fmt.Errorf("list primary ips: %w", err)
	}
	for _, p := range ips {
		if strings.HasPrefix(p.Name, namePrefix) {
			orphans = append(orphans, fmt.Sprintf("primary-ip: %s (id %d)", p.Name, p.ID))
		}
	}

	fws, err := client.AllFirewalls(ctx)
	if err != nil {
		return fmt.Errorf("list firewalls: %w", err)
	}
	for _, f := range fws {
		if strings.HasPrefix(f.Name, namePrefix) {
			orphans = append(orphans, fmt.Sprintf("firewall: %s (id %d)", f.Name, f.ID))
		}
	}

	if len(orphans) > 0 {
		return fmt.Errorf(
			"D-12 gate 1: Hetzner project contains %d pre-existing runnerkit-* resources; refuse to run live smoke. Resources:\n  - %s\nClean up with `runnerkit destroy --yes` for each, or via Hetzner Console.",
			len(orphans),
			strings.Join(orphans, "\n  - "),
		)
	}
	return nil
}
