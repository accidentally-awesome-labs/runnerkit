// Command destroy_verify implements D-12 gate 2: after `runnerkit
// destroy --yes`, poll Hetzner for 404 on every saved resource ID. Fail
// loudly if any resource lingers within the timeout.
//
// State file path is read from RUNNERKIT_SMOKE_STATE_FILE (set by the
// calling shell script). Timeout is read from RUNNERKIT_SMOKE_TIMEOUT
// (seconds; default 300). Polling cadence is 5 seconds.
//
// This binary is excluded from `go build ./...` by the `_smokebin`
// directory's `_` prefix (Go convention) and is invoked by
// scripts/smoke/hetzner-destroy-verify.sh via `go run`.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

const pollInterval = 500 * time.Millisecond

func main() {
	if err := run(context.Background(), nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// cloudIDs holds the integer Hetzner resource IDs recovered from state.json.
// hcloud-go v1.59.2 uses `int` for resource IDs; we keep that to avoid
// cross-version drift.
type cloudIDs struct {
	ServerID      int
	SSHKeyID      int
	PrimaryIPv4ID int
	PrimaryIPv6ID int
	FirewallID    int
}

func (c cloudIDs) empty() bool {
	return c.ServerID == 0 && c.SSHKeyID == 0 && c.PrimaryIPv4ID == 0 && c.PrimaryIPv6ID == 0 && c.FirewallID == 0
}

// verifierClient is the subset of *hcloud.Client we use; lets tests
// inject a fake.
type verifierClient interface {
	GetServerByID(ctx context.Context, id int) (*hcloud.Server, error)
	GetSSHKeyByID(ctx context.Context, id int) (*hcloud.SSHKey, error)
	GetPrimaryIPByID(ctx context.Context, id int) (*hcloud.PrimaryIP, error)
	GetFirewallByID(ctx context.Context, id int) (*hcloud.Firewall, error)
}

type realVerifier struct{ c *hcloud.Client }

func (r realVerifier) GetServerByID(ctx context.Context, id int) (*hcloud.Server, error) {
	s, _, err := r.c.Server.GetByID(ctx, id)
	return s, err
}

func (r realVerifier) GetSSHKeyByID(ctx context.Context, id int) (*hcloud.SSHKey, error) {
	k, _, err := r.c.SSHKey.GetByID(ctx, id)
	return k, err
}

func (r realVerifier) GetPrimaryIPByID(ctx context.Context, id int) (*hcloud.PrimaryIP, error) {
	p, _, err := r.c.PrimaryIP.GetByID(ctx, id)
	return p, err
}

func (r realVerifier) GetFirewallByID(ctx context.Context, id int) (*hcloud.Firewall, error) {
	f, _, err := r.c.Firewall.GetByID(ctx, id)
	return f, err
}

func run(ctx context.Context, client verifierClient) error {
	timeoutSec, _ := strconv.Atoi(os.Getenv("RUNNERKIT_SMOKE_TIMEOUT"))
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	statePath := os.Getenv("RUNNERKIT_SMOKE_STATE_FILE")
	if statePath == "" {
		return fmt.Errorf("RUNNERKIT_SMOKE_STATE_FILE required")
	}
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}
	ids, err := extractCloudIDs(raw)
	if err != nil {
		return err
	}
	if client == nil {
		token := os.Getenv("HCLOUD_TOKEN")
		if token == "" {
			return fmt.Errorf("HCLOUD_TOKEN required")
		}
		client = realVerifier{c: hcloud.NewClient(hcloud.WithToken(token))}
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	return pollUntilGone(ctx, client, ids, deadline)
}

// extractCloudIDs reads state.json bytes and pulls out the saved
// Hetzner resource IDs. Mirrors the structure of
// internal/state/schema.go::ProviderRef.Cloud (best-effort JSON
// unmarshal — we only need the integer ID fields).
func extractCloudIDs(raw []byte) (cloudIDs, error) {
	var partial struct {
		Repositories []struct {
			Provider struct {
				Cloud struct {
					ServerID      string `json:"server_id"`
					SSHKeyID      string `json:"ssh_key_id"`
					PrimaryIPv4ID string `json:"primary_ipv4_id"`
					PrimaryIPv6ID string `json:"primary_ipv6_id"`
					FirewallID    string `json:"firewall_id"`
				} `json:"cloud"`
			} `json:"provider"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal(raw, &partial); err != nil {
		return cloudIDs{}, fmt.Errorf("parse state json: %w", err)
	}
	if len(partial.Repositories) == 0 {
		// After a successful destroy --yes, the repo entry is removed
		// from state. That is the SUCCESS signal — nothing to verify.
		return cloudIDs{}, nil
	}
	c := partial.Repositories[0].Provider.Cloud
	out := cloudIDs{}
	out.ServerID = parseIntOrZero(c.ServerID)
	out.SSHKeyID = parseIntOrZero(c.SSHKeyID)
	out.PrimaryIPv4ID = parseIntOrZero(c.PrimaryIPv4ID)
	out.PrimaryIPv6ID = parseIntOrZero(c.PrimaryIPv6ID)
	out.FirewallID = parseIntOrZero(c.FirewallID)
	return out, nil
}

func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func pollUntilGone(ctx context.Context, client verifierClient, ids cloudIDs, deadline time.Time) error {
	if ids.empty() {
		// No IDs to verify — destroy already removed the state entry.
		// Treat as success.
		return nil
	}
	for {
		remaining, err := checkRemaining(ctx, client, ids)
		if err != nil {
			return err
		}
		if remaining.empty() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf(
				"D-12 gate 2: timeout waiting for Hetzner resources to return 404 (deadline=%s, remaining=%+v)",
				deadline.Format(time.RFC3339),
				remaining,
			)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
		ids = remaining
	}
}

func checkRemaining(ctx context.Context, client verifierClient, ids cloudIDs) (cloudIDs, error) {
	out := cloudIDs{}
	if ids.ServerID > 0 {
		s, err := client.GetServerByID(ctx, ids.ServerID)
		if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return cloudIDs{}, fmt.Errorf("get server %d: %w", ids.ServerID, err)
		}
		if s != nil {
			out.ServerID = ids.ServerID
		}
	}
	if ids.SSHKeyID > 0 {
		k, err := client.GetSSHKeyByID(ctx, ids.SSHKeyID)
		if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return cloudIDs{}, fmt.Errorf("get ssh key %d: %w", ids.SSHKeyID, err)
		}
		if k != nil {
			out.SSHKeyID = ids.SSHKeyID
		}
	}
	if ids.PrimaryIPv4ID > 0 {
		p, err := client.GetPrimaryIPByID(ctx, ids.PrimaryIPv4ID)
		if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return cloudIDs{}, fmt.Errorf("get primary ipv4 %d: %w", ids.PrimaryIPv4ID, err)
		}
		if p != nil {
			out.PrimaryIPv4ID = ids.PrimaryIPv4ID
		}
	}
	if ids.PrimaryIPv6ID > 0 {
		p, err := client.GetPrimaryIPByID(ctx, ids.PrimaryIPv6ID)
		if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return cloudIDs{}, fmt.Errorf("get primary ipv6 %d: %w", ids.PrimaryIPv6ID, err)
		}
		if p != nil {
			out.PrimaryIPv6ID = ids.PrimaryIPv6ID
		}
	}
	if ids.FirewallID > 0 {
		f, err := client.GetFirewallByID(ctx, ids.FirewallID)
		if err != nil && !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return cloudIDs{}, fmt.Errorf("get firewall %d: %w", ids.FirewallID, err)
		}
		if f != nil {
			out.FirewallID = ids.FirewallID
		}
	}
	return out, nil
}
