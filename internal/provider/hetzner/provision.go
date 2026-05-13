package hetzner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

const (
	// CloudInitUserDataVersion is written into /var/lib/runnerkit/cloud-init.json
	// on provisioned VMs and used as a state default when the inventory omits it.
	CloudInitUserDataVersion = "runnerkit-cloud-init-v3"
	defaultSSHUser           = "runnerkit-admin"
	tagRunnerKitTrue         = "runnerkit=true"
	tagManagedTrue           = "managed=true"
	tagModePersistent        = "mode=persistent"
)

type Provider struct {
	Env       map[string]string
	Client    Client
	NewClient func(token string) Client
	// Log receives optional structured lifecycle events when non-nil and
	// enabled at info (see WithLogger).
	Log *slog.Logger
	// Sleep is an optional injection point for time.Sleep. Used by the
	// Bug 30 (Plan 06-12) destroy retry loop so tests can fast-forward
	// without burning wall-clock time. Defaults to time.Sleep when nil.
	Sleep func(time.Duration)
}

type Option func(*Provider)

func WithClient(client Client) Option {
	return func(p *Provider) { p.Client = client }
}

func WithClientFactory(factory func(token string) Client) Option {
	return func(p *Provider) { p.NewClient = factory }
}

// WithSleep injects a custom sleep implementation for tests. Bug 30
// (Plan 06-12, 2026-05-06): the destroy retry loop on 409
// must_be_unassigned uses this hook to fast-forward in unit tests.
func WithSleep(sleep func(time.Duration)) Option {
	return func(p *Provider) { p.Sleep = sleep }
}

// WithLogger attaches an optional slog.Logger for Hetzner provision/destroy
// lifecycle events (bounded fields only — no secrets).
func WithLogger(log *slog.Logger) Option {
	return func(p *Provider) { p.Log = log }
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
		return provider.ProvisionResult{}, errors.New(strings.Join(validation.Remediation, "; "))
	}
	if strings.TrimSpace(input.PublicKey) == "" {
		return provider.ProvisionResult{}, fmt.Errorf("public SSH key is required for Hetzner cloud provisioning")
	}
	if p.Log != nil && p.Log.Enabled(ctx, slog.LevelInfo) {
		p.Log.InfoContext(ctx, "hetzner.provision.begin",
			slog.String("repo", input.RepoFullName),
			slog.String("runner", input.RunnerName),
			slog.String("state_id", input.StateID),
			slog.String("region", profile.Region),
		)
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
	// Bug 26 (Plan 06-11, 2026-05-06): pass EnableIPv4/EnableIPv6 only,
	// without overriding the IPv4/IPv6 PrimaryIP fields. This makes
	// Hetzner auto-allocate fresh primary IPs that carry
	// `AutoDelete: true` by default — which is the property destroy.go
	// relies on for cascade-delete (no `Server must be offline` error
	// when deleting). See destroy.go for the full cascade explanation;
	// providing an explicit *PrimaryIP here would override the default
	// and break the cascade, so DO NOT add IPv4/IPv6 fields below
	// without setting AutoDelete: true on a separately-allocated IP and
	// updating destroy.go's flow accordingly.
	server, action, err := client.CreateServer(ctx, hcloud.ServerCreateOpts{
		Name:             plan.ResourceNames["server"],
		ServerType:       serverType,
		Image:            image,
		SSHKeys:          []*hcloud.SSHKey{sshKey},
		Location:         location,
		UserData:         cloudInitUserData(profile.SSHUser, input.PublicKey, input.ExtraPackages),
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
	if p.Log != nil && p.Log.Enabled(ctx, slog.LevelInfo) {
		p.Log.InfoContext(ctx, "hetzner.provision.end",
			slog.String("repo", input.RepoFullName),
			slog.String("server_id", resourceIDs["server"]),
			slog.Bool("checkpoint_required", true),
		)
	}
	return provider.ProvisionResult{Machine: machine, CreatedResourceIDs: cloneIDs(resourceIDs), CheckpointRequired: true}, nil
}

func (p *Provider) Describe(ctx context.Context, ref state.ProviderRef) (provider.ProviderStatus, error) {
	client, _, err := p.client()
	if err != nil {
		return provider.ProviderStatus{}, err
	}
	ids := mergedProviderIDs(ref)
	serverIDStr := ids["server"]
	if strings.TrimSpace(serverIDStr) == "" {
		return provider.ProviderStatus{Kind: hetznerKind(ref), Region: ref.Region, Found: false}, nil
	}
	parsed, parseErr := parseID(serverIDStr)
	if parseErr != nil {
		return provider.ProviderStatus{
			Kind:   hetznerKind(ref),
			Region: ref.Region,
			Found:  false,
			Error:  parseErr.Error(),
		}, nil
	}
	server, err := client.GetServer(ctx, parsed)
	if err != nil {
		if isAlreadyAbsentError(err) {
			return provider.ProviderStatus{Kind: hetznerKind(ref), Region: ref.Region, Found: false}, nil
		}
		return provider.ProviderStatus{}, err
	}
	if server == nil {
		return provider.ProviderStatus{Kind: hetznerKind(ref), Region: ref.Region, Found: false}, nil
	}
	ipv4, ipv6 := publicIPs(server)
	publicHost := ipv4
	if publicHost == "" {
		publicHost = ipv6
	}
	region := ref.Region
	if server.Datacenter != nil && server.Datacenter.Location != nil && server.Datacenter.Location.Name != "" {
		region = server.Datacenter.Location.Name
	}
	serverType := ""
	if server.ServerType != nil {
		serverType = server.ServerType.Name
	}
	imageName := ""
	if server.Image != nil {
		imageName = server.Image.Name
	}
	return provider.ProviderStatus{
		Kind:              hetznerKind(ref),
		Found:             true,
		Status:            serverStatus(server),
		Region:            region,
		ServerType:        serverType,
		Image:             imageName,
		PublicHost:        publicHost,
		BillableResources: billableResourceLinesFromIDs(ids),
		Drift:             nil,
	}, nil
}

func hetznerKind(ref state.ProviderRef) string {
	if strings.TrimSpace(ref.Kind) != "" {
		return ref.Kind
	}
	return provider.HetznerProvider
}

func billableResourceLinesFromIDs(ids map[string]string) []string {
	out := []string{}
	for _, key := range []string{"server", "ssh_key", "firewall", "primary_ipv4", "primary_ipv6"} {
		if v := strings.TrimSpace(ids[key]); v != "" {
			out = append(out, key+":"+v)
		}
	}
	return out
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

func cloudInitUserData(user string, publicKey string, extraPackages []string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		user = defaultSSHUser
	}
	publicKey = strings.TrimSpace(publicKey)
	// Scoped sudoers (same rules as install.sh / byo-prepare) are applied as
	// root during cloud-init so bootstrap works even when the cloud-init
	// `users[].sudo` NOPASSWD stanza is ignored or mis-applied on some images.
	sudoers := strings.TrimSuffix(bootstrap.RenderSudoersEntry(user), "\n")
	var sudoersBlock strings.Builder
	for _, line := range strings.Split(sudoers, "\n") {
		sudoersBlock.WriteString("      ")
		sudoersBlock.WriteString(line)
		sudoersBlock.WriteByte('\n')
	}
	var packagesBlock strings.Builder
	packagesBlock.WriteString("  - sudo\n")
	for _, pkg := range bootstrap.BaselinePackages {
		packagesBlock.WriteString("  - ")
		packagesBlock.WriteString(pkg)
		packagesBlock.WriteByte('\n')
	}
	for _, pkg := range extraPackages {
		packagesBlock.WriteString("  - ")
		packagesBlock.WriteString(pkg)
		packagesBlock.WriteByte('\n')
	}
	return fmt.Sprintf(`#cloud-config
users:
  - default
  - name: %s
    groups: sudo
    shell: /bin/bash
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    ssh_authorized_keys:
      - %s
write_files:
  - path: /var/lib/runnerkit/installer.sudoers.staged
    owner: root:root
    permissions: '0440'
    content: |
%sapt:
  sources_list: |
    deb $MIRROR $RELEASE main restricted universe
    deb $MIRROR $RELEASE-updates main restricted universe
    deb $MIRROR $RELEASE-security main restricted universe
package_update: true
packages:
%sruncmd:
  - mkdir -p /var/lib/runnerkit
  - sh -c 'visudo -cf /var/lib/runnerkit/installer.sudoers.staged && install -m 0440 -o root -g root /var/lib/runnerkit/installer.sudoers.staged /etc/sudoers.d/runnerkit-installer'
  - printf '{"cloud_init_version":"%s"}\n' > /var/lib/runnerkit/cloud-init.json
`, user, publicKey, sudoersBlock.String(), packagesBlock.String(), CloudInitUserDataVersion)
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
	resourceIDs := cloneIDs(ids)
	cloud := state.CloudInventory{
		Provider:      provider.HetznerProvider,
		ServerID:      resourceIDs["server"],
		ServerName:    plan.ResourceNames["server"],
		ServerStatus:  serverStatus(server),
		Region:        plan.Region,
		Datacenter:    datacenterName(server),
		ServerType:    plan.ServerType,
		Image:         plan.Image,
		PublicIPv4:    publicIPv4,
		PublicIPv6:    publicIPv6,
		PrimaryIPv4ID: resourceIDs["primary_ipv4"],
		PrimaryIPv6ID: resourceIDs["primary_ipv6"],
		// Bug 30 (Plan 06-12, 2026-05-06): IPs auto-allocated via
		// ServerCreatePublicNet EnableIPv4/EnableIPv6 carry
		// AutoDelete=true on the wire. Record the flag in state so
		// destroy.go can skip the explicit DeletePrimaryIP call and let
		// the auto_delete cascade (triggered by server.Delete) handle
		// the IPs — avoiding the 409 must_be_unassigned race during the
		// cascade window. Plan 06-11 Bug 26 locked in EnableIPv4=true,
		// EnableIPv6=true, IPv4=nil, IPv6=nil; this flag follows from
		// that contract.
		PrimaryIPv4AutoDelete: resourceIDs["primary_ipv4"] != "",
		PrimaryIPv6AutoDelete: resourceIDs["primary_ipv6"] != "",
		SSHKeyID:              resourceIDs["ssh_key"],
		SSHKeyName:            plan.ResourceNames["ssh_key"],
		SSHKeyFingerprint:     "",
		FirewallID:            resourceIDs["firewall"],
		FirewallName:          plan.ResourceNames["firewall"],
		Tags:                  cloneTags(plan.Tags),
		CostProfile: state.CostProfileRef{
			Provider:             plan.Provider,
			Region:               plan.Region,
			ServerType:           plan.ServerType,
			Image:                plan.Image,
			EstimatedHourlyCost:  plan.EstimatedHourlyCost,
			EstimatedMonthlyCost: plan.EstimatedMonthlyCost,
			Caveat:               plan.CostEstimateCaveat,
		},
		CloudInitVersion: CloudInitUserDataVersion,
	}
	providerRef := state.ProviderRef{Kind: provider.HetznerProvider, Name: provider.HetznerProvider, IDs: resourceIDs, Region: plan.Region, Profile: plan.ServerType, ResourceIDs: cloneIDs(resourceIDs), Tags: cloneTags(plan.Tags), Cloud: cloud}
	return provider.Machine{
		Target:      remote.Target{User: defaultSSHUser, Host: host, Port: 22, Raw: defaultSSHUser + "@" + host + ":22"},
		Provider:    providerRef,
		PublicIPv4:  publicIPv4,
		PublicIPv6:  publicIPv6,
		ResourceIDs: cloneIDs(resourceIDs),
	}
}

func serverStatus(server *hcloud.Server) string {
	if server == nil || server.Status == "" {
		return "provisioning"
	}
	return string(server.Status)
}

func datacenterName(server *hcloud.Server) string {
	if server == nil || server.Datacenter == nil {
		return ""
	}
	return server.Datacenter.Name
}

func cloneTags(tags map[string]string) map[string]string {
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
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

// HostKeyProbeOptions configures the retry budget for
// ProbeHostKeyWithRetry. All fields are optional; zero values fall
// back to the documented defaults below.
//
// Bug 22 (Plan 06-10, 2026-05-06): cloud-init writes
// /etc/ssh/ssh_host_*_key.pub during the boot phase, typically 30-90s
// after the Hetzner API reports the server status as "running". A
// single-shot ssh-keyscan + ProbeHostKey landed in that window
// returned an empty fingerprint, surfacing as
// `SSH host key fingerprint was not observed` and aborting `runnerkit
// up --cloud hetzner` after billable resources had already been
// created. ProbeHostKeyWithRetry tolerates that window by retrying
// with a small interval so the host_key entry shows up before the
// caller exits the readiness phase.
type HostKeyProbeOptions struct {
	// Attempts is the maximum number of probe calls. Default: 60.
	Attempts int
	// Interval is the wait between attempts. Default: 5 seconds.
	// At the default budget that gives ~5 minutes of total wall-clock
	// runway, easily covering Hetzner cloud-init's typical 30-90s
	// host-key install window with headroom for residential-IP
	// retries.
	Interval time.Duration
	// Sleep is an optional injection point for tests so they can
	// fast-forward without burning real wall-clock seconds. Default:
	// time.Sleep.
	Sleep func(time.Duration)
}

const (
	defaultHostKeyProbeAttempts = 60
	defaultHostKeyProbeInterval = 5 * time.Second
)

// ProbeHostKeyWithRetry repeatedly invokes prober.ProbeHostKey until
// the returned HostKey carries a non-empty Fingerprint, the context
// is cancelled, or opts.Attempts is exhausted. On success it returns
// the observed HostKey; on exhaustion it returns the LAST observed
// error (or a generic "not observed" error if every attempt returned
// nil err + empty fingerprint) so callers can surface a useful
// diagnostic. Empty fingerprints with a nil error count as failures
// because Hetzner cloud-init can return an empty ssh-keyscan result
// even when the SSH layer accepts the connection.
func ProbeHostKeyWithRetry(ctx context.Context, prober remote.HostKeyProber, target remote.Target, opts HostKeyProbeOptions) (remote.HostKey, error) {
	attempts := opts.Attempts
	if attempts <= 0 {
		attempts = defaultHostKeyProbeAttempts
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = defaultHostKeyProbeInterval
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	var (
		lastErr error
		lastKey remote.HostKey
	)
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return remote.HostKey{}, err
		}
		key, err := prober.ProbeHostKey(ctx, target)
		lastKey = key
		lastErr = err
		if err == nil && remote.NormalizeHostKey(key).Fingerprint != "" {
			return key, nil
		}
		if i < attempts-1 {
			sleep(interval)
		}
	}
	if lastErr != nil {
		return lastKey, lastErr
	}
	return lastKey, errors.New("SSH host key fingerprint was not observed after retry budget exhausted")
}
