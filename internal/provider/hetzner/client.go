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

func parseID(id string) (int, error) { return strconv.Atoi(id) }
