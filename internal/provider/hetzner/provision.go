package hetzner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/salar/runnerkit/internal/provider"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
)

const (
	cloudInitVersion  = "runnerkit-cloud-init-v1"
	defaultSSHUser    = "runnerkit-admin"
	tagRunnerKitTrue  = "runnerkit=true"
	tagManagedTrue    = "managed=true"
	tagModePersistent = "mode=persistent"
)

type Provider struct {
	Env       map[string]string
	Client    Client
	NewClient func(token string) Client
}

type Option func(*Provider)

func WithClient(client Client) Option {
	return func(p *Provider) { p.Client = client }
}

func WithClientFactory(factory func(token string) Client) Option {
	return func(p *Provider) { p.NewClient = factory }
}

func NewProvider(env map[string]string, opts ...Option) *Provider {
	p := &Provider{Env: env, NewClient: func(token string) Client { return NewClient(token) }}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) Name() string { return provider.HetznerProvider }

func (p *Provider) Validate(ctx context.Context, input provider.ProvisionInput) (provider.ValidationResult, error) {
	client, source, err := p.client()
	if err != nil {
		var missing *MissingTokenError
		if errors.As(err, &missing) {
			return provider.ValidationResult{OK: false, Remediation: missing.Remediation}, err
		}
		return provider.ValidationResult{OK: false}, err
	}
	profile := withDefaults(input.Profile)
	if _, _, _, result, err := lookupProfile(ctx, client, profile); err != nil || !result.OK {
		return result, err
	}
	return provider.ValidationResult{OK: true, Source: source.Source}, nil
}

func (p *Provider) Plan(_ context.Context, input provider.ProvisionInput) (provider.ProvisionPlan, error) {
	input.Profile = withDefaults(input.Profile)
	return provider.HetznerProvisionPlan(input), nil
}

func (p *Provider) Provision(ctx context.Context, input provider.ProvisionInput) (provider.ProvisionResult, error) {
	client, _, err := p.client()
	if err != nil {
		return provider.ProvisionResult{}, err
	}
	profile := withDefaults(input.Profile)
	location, serverType, image, validation, err := lookupProfile(ctx, client, profile)
	if err != nil {
		return provider.ProvisionResult{}, err
	}
	if !validation.OK {
		return provider.ProvisionResult{}, fmt.Errorf(strings.Join(validation.Remediation, "; "))
	}
	if strings.TrimSpace(input.PublicKey) == "" {
		return provider.ProvisionResult{}, fmt.Errorf("public SSH key is required for Hetzner cloud provisioning")
	}

	plan := provider.HetznerProvisionPlan(input)
	resourceIDs := map[string]string{}
	labels := hcloudLabels(plan.Tags)

	sshKey, err := client.CreateSSHKey(ctx, hcloud.SSHKeyCreateOpts{Name: plan.ResourceNames["ssh_key"], PublicKey: strings.TrimSpace(input.PublicKey), Labels: labels})
	if err != nil {
		return provider.ProvisionResult{}, err
	}
	if sshKey != nil && sshKey.ID != 0 {
		resourceIDs["ssh_key"] = strconv.Itoa(sshKey.ID)
	}

	firewall, err := client.CreateFirewall(ctx, hcloud.FirewallCreateOpts{Name: plan.ResourceNames["firewall"], Labels: labels, Rules: firewallRules(input.SSHAllowedCIDR)})
	if firewall != nil && firewall.ID != 0 {
		resourceIDs["firewall"] = strconv.Itoa(firewall.ID)
	}
	if err != nil {
		return provider.ProvisionResult{}, provisionError("firewall", input, plan, resourceIDs, nil, err)
	}

	start := true
	automount := false
	server, action, err := client.CreateServer(ctx, hcloud.ServerCreateOpts{
		Name:             plan.ResourceNames["server"],
		ServerType:       serverType,
		Image:            image,
		SSHKeys:          []*hcloud.SSHKey{sshKey},
		Location:         location,
		UserData:         cloudInitUserData(profile.SSHUser, input.PublicKey),
		StartAfterCreate: &start,
		Labels:           labels,
		Automount:        &automount,
		Volumes:          nil,
		Firewalls:        []*hcloud.ServerCreateFirewall{{Firewall: *firewall}},
		PublicNet:        &hcloud.ServerCreatePublicNet{EnableIPv4: true, EnableIPv6: true},
	})
	if server != nil && server.ID != 0 {
		resourceIDs["server"] = strconv.Itoa(server.ID)
	}
	if action != nil && action.ID != 0 {
		resourceIDs["create_action"] = strconv.Itoa(action.ID)
	}
	addPublicNetResourceIDs(resourceIDs, server)
	if err != nil {
		return provider.ProvisionResult{}, provisionError("server", input, plan, resourceIDs, server, err)
	}
	machine := machineFromServer(input, plan, resourceIDs, server)
	return provider.ProvisionResult{Machine: machine, CreatedResourceIDs: cloneIDs(resourceIDs), CheckpointRequired: true}, nil
}

func (p *Provider) WaitReady(_ context.Context, machine provider.Machine) (provider.Machine, error) {
	return machine, errors.New("hetzner readiness is not implemented in this plan")
}

func (p *Provider) Describe(_ context.Context, ref state.ProviderRef) (provider.ProviderStatus, error) {
	return provider.ProviderStatus{Kind: ref.Kind, Region: ref.Region, Found: false}, nil
}

func (p *Provider) Destroy(_ context.Context, _ state.ProviderRef) (provider.DestroyResult, error) {
	return provider.DestroyResult{}, errors.New("hetzner destroy is not implemented in this plan")
}

func (p *Provider) VerifyDestroyed(_ context.Context, _ state.ProviderRef) (provider.VerificationResult, error) {
	return provider.VerificationResult{OK: false}, errors.New("hetzner destroy verification is not implemented in this plan")
}

func (p *Provider) client() (Client, TokenSource, error) {
	source, err := ResolveToken(p.Env)
	if err != nil {
		return nil, TokenSource{}, err
	}
	if p.Client != nil {
		return p.Client, source, nil
	}
	factory := p.NewClient
	if factory == nil {
		factory = func(token string) Client { return NewClient(token) }
	}
	return factory(source.Token), source, nil
}

func withDefaults(profile provider.Profile) provider.Profile {
	defaults := provider.DefaultHetznerProfile()
	if profile.Provider == "" {
		profile.Provider = defaults.Provider
	}
	if profile.Region == "" {
		profile.Region = defaults.Region
	}
	if profile.ServerType == "" {
		profile.ServerType = defaults.ServerType
	}
	if profile.Image == "" {
		profile.Image = defaults.Image
	}
	if profile.SSHUser == "" {
		profile.SSHUser = defaults.SSHUser
	}
	if profile.EstimatedHourlyCost == "" {
		profile.EstimatedHourlyCost = defaults.EstimatedHourlyCost
	}
	if profile.EstimatedMonthlyCost == "" {
		profile.EstimatedMonthlyCost = defaults.EstimatedMonthlyCost
	}
	if profile.CostEstimateCaveat == "" {
		profile.CostEstimateCaveat = defaults.CostEstimateCaveat
	}
	return profile
}

func lookupProfile(ctx context.Context, client Client, profile provider.Profile) (*hcloud.Location, *hcloud.ServerType, *hcloud.Image, provider.ValidationResult, error) {
	location, err := client.GetLocation(ctx, profile.Region)
	if err != nil {
		return nil, nil, nil, provider.ValidationResult{OK: false, Remediation: []string{"Verify Hetzner location " + profile.Region + " is available, then rerun runnerkit up --cloud hetzner."}}, err
	}
	if location == nil {
		return nil, nil, nil, provider.ValidationResult{OK: false, Remediation: []string{"Hetzner location " + profile.Region + " is unavailable; choose --cloud-region fsn1 or another supported location."}}, nil
	}
	serverType, err := client.GetServerType(ctx, profile.ServerType)
	if err != nil {
		return nil, nil, nil, provider.ValidationResult{OK: false, Remediation: []string{"Verify Hetzner server type " + profile.ServerType + " is available, then rerun runnerkit up --cloud hetzner."}}, err
	}
	if serverType == nil {
		return nil, nil, nil, provider.ValidationResult{OK: false, Remediation: []string{"Hetzner server type " + profile.ServerType + " is unavailable; choose --cloud-profile cpx22 or another supported profile."}}, nil
	}
	image, err := client.GetImage(ctx, profile.Image)
	if err != nil {
		return nil, nil, nil, provider.ValidationResult{OK: false, Remediation: []string{"Verify Hetzner image " + profile.Image + " is available, then rerun runnerkit up --cloud hetzner."}}, err
	}
	if image == nil {
		return nil, nil, nil, provider.ValidationResult{OK: false, Remediation: []string{"Hetzner image " + profile.Image + " is unavailable; use ubuntu-24.04 for the recommended profile."}}, nil
	}
	return location, serverType, image, provider.ValidationResult{OK: true}, nil
}

func firewallRules(cidr string) []hcloud.FirewallRule {
	_, ipnet, err := net.ParseCIDR(defaultCIDR(cidr))
	if err != nil {
		_, ipnet, _ = net.ParseCIDR(provider.HetznerDefaultSSHAllowedCIDR)
	}
	port := "22"
	desc := "RunnerKit SSH readiness access"
	return []hcloud.FirewallRule{{Direction: hcloud.FirewallRuleDirectionIn, SourceIPs: []net.IPNet{*ipnet}, Protocol: hcloud.FirewallRuleProtocolTCP, Port: &port, Description: &desc}}
}

func defaultCIDR(cidr string) string {
	if strings.TrimSpace(cidr) == "" {
		return provider.HetznerDefaultSSHAllowedCIDR
	}
	return strings.TrimSpace(cidr)
}

func cloudInitUserData(user string, publicKey string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		user = defaultSSHUser
	}
	publicKey = strings.TrimSpace(publicKey)
	return fmt.Sprintf(`#cloud-config
users:
  - default
  - name: %s
    groups: sudo
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - %s
package_update: true
packages:
  - sudo
runcmd:
  - mkdir -p /var/lib/runnerkit
  - printf '{"cloud_init_version":"%s"}\n' > /var/lib/runnerkit/cloud-init.json
`, user, publicKey, cloudInitVersion)
}

func hcloudLabels(tags map[string]string) map[string]string {
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = sanitizeLabelValue(v)
	}
	return out
}

func sanitizeLabelValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "runnerkit"
	}
	if len(out) > 63 {
		out = strings.Trim(out[:63], "-")
	}
	return out
}

func addPublicNetResourceIDs(ids map[string]string, server *hcloud.Server) {
	if server == nil {
		return
	}
	if server.PublicNet.IPv4.ID != 0 {
		ids["primary_ipv4"] = strconv.Itoa(server.PublicNet.IPv4.ID)
	}
	if server.PublicNet.IPv6.ID != 0 {
		ids["primary_ipv6"] = strconv.Itoa(server.PublicNet.IPv6.ID)
	}
}

func machineFromServer(input provider.ProvisionInput, plan provider.ProvisionPlan, ids map[string]string, server *hcloud.Server) provider.Machine {
	publicIPv4, publicIPv6 := publicIPs(server)
	host := publicIPv4
	if host == "" {
		host = publicIPv6
	}
	providerRef := state.ProviderRef{Kind: provider.HetznerProvider, IDs: cloneIDs(ids), Region: plan.Region}
	return provider.Machine{
		Target:      remote.Target{User: provider.HetznerDefaultSSHUser, Host: host, Port: 22, Raw: provider.HetznerDefaultSSHUser + "@" + host + ":22"},
		Provider:    providerRef,
		PublicIPv4:  publicIPv4,
		PublicIPv6:  publicIPv6,
		ResourceIDs: cloneIDs(ids),
	}
}

func publicIPs(server *hcloud.Server) (string, string) {
	if server == nil {
		return "", ""
	}
	ipv4 := ""
	if ip := server.PublicNet.IPv4.IP; ip != nil && !ip.IsUnspecified() {
		ipv4 = ip.String()
	}
	ipv6 := ""
	if ip := server.PublicNet.IPv6.IP; ip != nil && !ip.IsUnspecified() {
		ipv6 = ip.String()
	} else if server.PublicNet.IPv6.Network != nil {
		ipv6 = server.PublicNet.IPv6.Network.String()
	}
	return ipv4, ipv6
}

func provisionError(stage string, input provider.ProvisionInput, plan provider.ProvisionPlan, ids map[string]string, server *hcloud.Server, err error) error {
	return &provider.ProvisionError{Stage: stage, Result: provider.ProvisionResult{Machine: machineFromServer(input, plan, ids, server), CreatedResourceIDs: cloneIDs(ids), CheckpointRequired: len(ids) > 0}, Err: err}
}

func cloneIDs(ids map[string]string) map[string]string {
	out := make(map[string]string, len(ids))
	for k, v := range ids {
		out[k] = v
	}
	return out
}

var _ provider.Provider = (*Provider)(nil)
