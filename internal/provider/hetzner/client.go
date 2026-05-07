package hetzner

import (
	"context"
	"strconv"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

// Client wraps the small Hetzner Cloud API surface RunnerKit needs for Phase 4.
type Client interface {
	GetLocation(ctx context.Context, name string) (*hcloud.Location, error)
	GetServerType(ctx context.Context, name string) (*hcloud.ServerType, error)
	GetImage(ctx context.Context, name string) (*hcloud.Image, error)
	CreateSSHKey(ctx context.Context, opts hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error)
	CreateFirewall(ctx context.Context, opts hcloud.FirewallCreateOpts) (*hcloud.Firewall, error)
	CreateServer(ctx context.Context, opts hcloud.ServerCreateOpts) (*hcloud.Server, *hcloud.Action, error)
	WaitForAction(ctx context.Context, action *hcloud.Action) error
	GetServer(ctx context.Context, id int) (*hcloud.Server, error)
	GetSSHKey(ctx context.Context, id int) (*hcloud.SSHKey, error)
	GetFirewall(ctx context.Context, id int) (*hcloud.Firewall, error)
	GetPrimaryIP(ctx context.Context, id int) (*hcloud.PrimaryIP, error)
	DeleteServer(ctx context.Context, id int) error
	DeleteSSHKey(ctx context.Context, id int) error
	DeleteFirewall(ctx context.Context, id int) error
	DeletePrimaryIP(ctx context.Context, id int) error
	// DetachFirewallFromServer detaches the given firewall from the
	// given server before either is deleted. Bug 23 (Plan 06-10): the
	// Hetzner API rejects firewall.Delete with `resource_in_use` while
	// the firewall is still attached, so destroy must detach first.
	DetachFirewallFromServer(ctx context.Context, firewallID int, serverID int) error
	// UnassignPrimaryIP detaches a Primary IP from whatever server it
	// is currently assigned to. Bug 23 (Plan 06-10) added this method
	// to support a manual unassign-before-delete path. Bug 26 (Plan
	// 06-11) now relies on Hetzner's `auto_delete=true` cascade for
	// primary IPs auto-allocated with the server, so destroy.go no
	// longer calls UnassignPrimaryIP. The method remains on the
	// interface as a future fallback for legacy state where a primary
	// IP carries `auto_delete=false` (would require server power-off
	// first; the live `Server must be offline` error path).
	UnassignPrimaryIP(ctx context.Context, id int) error
}

type APIClient struct {
	client *hcloud.Client
}

func NewClient(token string) *APIClient {
	return &APIClient{client: hcloud.NewClient(hcloud.WithToken(token))}
}

func (c *APIClient) GetLocation(ctx context.Context, name string) (*hcloud.Location, error) {
	location, _, err := c.client.Location.GetByName(ctx, name)
	return location, err
}

func (c *APIClient) GetServerType(ctx context.Context, name string) (*hcloud.ServerType, error) {
	serverType, _, err := c.client.ServerType.GetByName(ctx, name)
	return serverType, err
}

func (c *APIClient) GetImage(ctx context.Context, name string) (*hcloud.Image, error) {
	image, _, err := c.client.Image.GetByName(ctx, name)
	return image, err
}

func (c *APIClient) CreateSSHKey(ctx context.Context, opts hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error) {
	sshKey, _, err := c.client.SSHKey.Create(ctx, opts)
	return sshKey, err
}

func (c *APIClient) CreateFirewall(ctx context.Context, opts hcloud.FirewallCreateOpts) (*hcloud.Firewall, error) {
	result, _, err := c.client.Firewall.Create(ctx, opts)
	return result.Firewall, err
}

func (c *APIClient) CreateServer(ctx context.Context, opts hcloud.ServerCreateOpts) (*hcloud.Server, *hcloud.Action, error) {
	result, _, err := c.client.Server.Create(ctx, opts)
	return result.Server, result.Action, err
}

func (c *APIClient) WaitForAction(ctx context.Context, action *hcloud.Action) error {
	if action == nil || action.ID == 0 {
		return nil
	}
	return c.client.Action.WaitFor(ctx, action)
}

func (c *APIClient) GetServer(ctx context.Context, id int) (*hcloud.Server, error) {
	server, _, err := c.client.Server.GetByID(ctx, id)
	return server, err
}

func (c *APIClient) GetSSHKey(ctx context.Context, id int) (*hcloud.SSHKey, error) {
	sshKey, _, err := c.client.SSHKey.GetByID(ctx, id)
	return sshKey, err
}

func (c *APIClient) GetFirewall(ctx context.Context, id int) (*hcloud.Firewall, error) {
	firewall, _, err := c.client.Firewall.GetByID(ctx, id)
	return firewall, err
}

func (c *APIClient) GetPrimaryIP(ctx context.Context, id int) (*hcloud.PrimaryIP, error) {
	primaryIP, _, err := c.client.PrimaryIP.GetByID(ctx, id)
	return primaryIP, err
}

func (c *APIClient) DeleteServer(ctx context.Context, id int) error {
	_, err := c.client.Server.Delete(ctx, &hcloud.Server{ID: id})
	return err
}

func (c *APIClient) DeleteSSHKey(ctx context.Context, id int) error {
	_, err := c.client.SSHKey.Delete(ctx, &hcloud.SSHKey{ID: id})
	return err
}

func (c *APIClient) DeleteFirewall(ctx context.Context, id int) error {
	_, err := c.client.Firewall.Delete(ctx, &hcloud.Firewall{ID: id})
	return err
}

func (c *APIClient) DeletePrimaryIP(ctx context.Context, id int) error {
	_, err := c.client.PrimaryIP.Delete(ctx, &hcloud.PrimaryIP{ID: id})
	return err
}

// DetachFirewallFromServer issues firewalls/{id}/actions/remove_from_resources
// to dissociate the firewall from the server. Bug 23 (Plan 06-10).
// Already-absent (404) errors bubble up unchanged so the caller's
// isAlreadyAbsentError helper can treat them as a no-op.
func (c *APIClient) DetachFirewallFromServer(ctx context.Context, firewallID int, serverID int) error {
	_, _, err := c.client.Firewall.RemoveResources(ctx, &hcloud.Firewall{ID: firewallID}, []hcloud.FirewallResource{{
		Type:   hcloud.FirewallResourceTypeServer,
		Server: &hcloud.FirewallResourceServer{ID: serverID},
	}})
	return err
}

// UnassignPrimaryIP issues primary_ips/{id}/actions/unassign so the
// follow-up DeletePrimaryIP call does not see `must_be_unassigned`.
// Bug 23 (Plan 06-10).
func (c *APIClient) UnassignPrimaryIP(ctx context.Context, id int) error {
	_, _, err := c.client.PrimaryIP.Unassign(ctx, id)
	return err
}

func parseID(id string) (int, error) { return strconv.Atoi(id) }
