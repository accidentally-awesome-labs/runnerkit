package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

// Provider defines the cloud lifecycle boundary used by RunnerKit.
type Provider interface {
	Name() string
	Validate(ctx context.Context, input ProvisionInput) (ValidationResult, error)
	Plan(ctx context.Context, input ProvisionInput) (ProvisionPlan, error)
	Provision(ctx context.Context, input ProvisionInput) (ProvisionResult, error)
	WaitReady(ctx context.Context, machine Machine) (Machine, error)
	Describe(ctx context.Context, ref state.ProviderRef) (ProviderStatus, error)
	Destroy(ctx context.Context, ref state.ProviderRef) (DestroyResult, error)
	VerifyDestroyed(ctx context.Context, ref state.ProviderRef) (VerificationResult, error)
}

// Registry maps provider names to implementations. Phase 4 registers only Hetzner.
type Registry map[string]Provider

func NewRegistry(providers ...Provider) Registry {
	registry := Registry{}
	for _, p := range providers {
		if p == nil || p.Name() == "" {
			continue
		}
		registry[p.Name()] = p
	}
	return registry
}

func (r Registry) Get(name string) (Provider, bool) {
	p, ok := r[name]
	return p, ok
}

type Profile struct {
	Provider           string `json:"provider"`
	Region             string `json:"region"`
	ServerType         string `json:"server_type"`
	Image              string `json:"image"`
	SSHUser            string `json:"ssh_user"`
	CostEstimateCaveat string `json:"cost_estimate_caveat"`
}

// CostSourceHetznerAPI marks a CostEstimate whose amounts were read from the
// Hetzner Cloud pricing API (GET /server_types, GET /pricing).
const CostSourceHetznerAPI = "hetzner_api"

// CostEstimate is a price for the planned resources over one billing
// period. Every amount is taken from the provider's pricing API response
// (RunnerKit ships no price constants); totals are the exact decimal sum of
// the per-resource API figures listed in Components. Amount is the gross
// total, the figure the account is billed.
type CostEstimate struct {
	Amount     string          `json:"amount"`
	Net        string          `json:"net"`
	Gross      string          `json:"gross"`
	Currency   string          `json:"currency"`
	Period     string          `json:"period"`
	Source     string          `json:"source"`
	FetchedAt  time.Time       `json:"fetched_at"`
	Components []CostComponent `json:"components,omitempty"`
}

// CostComponent is one priced resource inside a CostEstimate. Net and Gross
// are the strings the pricing API returned, unmodified.
type CostComponent struct {
	Resource string `json:"resource"`
	Net      string `json:"net"`
	Gross    string `json:"gross"`
}

// Label renders one component for humans, e.g. "server:cpx22 4.49 net /
// 5.34 gross".
func (c CostComponent) Label() string {
	return c.Resource + " " + trimDecimal(c.Net) + " net / " + trimDecimal(c.Gross) + " gross"
}

// CostSourceNote is the provenance label printed next to live prices.
func CostSourceNote(fetchedAt time.Time) string {
	return "reported by the Hetzner API at " + fetchedAt.UTC().Format(time.RFC3339) + "; excludes traffic overage"
}

// Label renders the estimate for humans, e.g. "EUR 5.39/month gross (4.53
// net)". A nil estimate renders as "" so callers never print a made-up price.
func (c *CostEstimate) Label() string {
	if c == nil || strings.TrimSpace(c.Amount) == "" {
		return ""
	}
	return fmt.Sprintf("%s %s/%s gross (%s net)", c.Currency, trimDecimal(c.Gross), c.Period, trimDecimal(c.Net))
}

// Summary is Label plus the provenance note; state files store this string.
func (c *CostEstimate) Summary() string {
	label := c.Label()
	if label == "" {
		return ""
	}
	return label + ", " + CostSourceNote(c.FetchedAt)
}

// trimDecimal drops trailing fractional zeros for display while keeping at
// least two decimals ("4.4900000000000000" -> "4.49", "0.0072000" ->
// "0.0072"). The value is unchanged; JSON keeps the API strings.
func trimDecimal(value string) string {
	value = strings.TrimSpace(value)
	dot := strings.IndexByte(value, '.')
	if dot < 0 {
		return value
	}
	trimmed := strings.TrimRight(value, "0")
	if keep := dot + 3; len(trimmed) < keep {
		if len(value) < keep {
			return value
		}
		trimmed = value[:keep]
	}
	return trimmed
}

// UnpricedLocationError reports that the provider's pricing API has no price
// for a resource RunnerKit would create in the chosen location. Plan and
// Provision return it before any create call so no billable resource is
// created without a price shown to the user.
type UnpricedLocationError struct {
	ServerType string
	Location   string
	// Resource is "server" when the server type has no price there, or
	// "primary_ipv4" when the location has no primary IPv4 price.
	Resource string
}

func (e *UnpricedLocationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Resource == "primary_ipv4" {
		return fmt.Sprintf("Hetzner reports no primary IPv4 price in %s; choose another --cloud-region", e.Location)
	}
	return fmt.Sprintf("Hetzner reports no price for %s in %s; choose another --cloud-region", e.ServerType, e.Location)
}

type ResourcePlan struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Billable bool   `json:"billable"`
	Action   string `json:"action"`
	ID       string `json:"id,omitempty"`
}

type ArtifactResult struct {
	Artifact string `json:"artifact"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

type ProvisionInput struct {
	RepoFullName    string    `json:"repo_full_name"`
	RunnerName      string    `json:"runner_name"`
	Labels          []string  `json:"labels"`
	WorkflowSnippet string    `json:"workflow_snippet"`
	Profile         Profile   `json:"profile"`
	SSHAllowedCIDR  string    `json:"ssh_allowed_cidr"`
	PublicKey       string    `json:"public_key,omitempty"`
	StateID         string    `json:"state_id"`
	CreatedAt       time.Time `json:"created_at"`

	// Mode tags the provisioned cloud resources with the chosen runner
	// mode. Phase 5 ephemeral cloud sets this to "ephemeral"; persistent
	// cloud setup leaves it empty so HetznerOwnershipTags falls back to
	// the existing "persistent" default.
	Mode string `json:"mode,omitempty"`

	// ExtraPackages are additional OS packages to install via cloud-init
	// (cloud path) or apt-get/dnf during bootstrap (BYO path). Specified
	// via --extra-packages flag or .runnerkit/config.yaml extra_packages.
	ExtraPackages []string `json:"extra_packages,omitempty"`
}

type ValidationResult struct {
	OK          bool     `json:"ok"`
	Source      string   `json:"source,omitempty"`
	Remediation []string `json:"remediation,omitempty"`
}

type ProvisionResult struct {
	Machine            Machine           `json:"machine"`
	CreatedResourceIDs map[string]string `json:"created_resource_ids,omitempty"`
	CheckpointRequired bool              `json:"checkpoint_required"`
}

type ProvisionError struct {
	Stage  string          `json:"stage"`
	Result ProvisionResult `json:"result"`
	Err    error           `json:"-"`
}

func (e *ProvisionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Stage == "" {
		return fmt.Sprintf("cloud provision failed: %v", e.Err)
	}
	return fmt.Sprintf("cloud provision failed at %s: %v", e.Stage, e.Err)
}

func (e *ProvisionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type ProvisionPlan struct {
	Provider   string `json:"provider"`
	Region     string `json:"region"`
	ServerType string `json:"server_type"`
	Image      string `json:"image"`
	SSHUser    string `json:"ssh_user"`
	// EstimatedHourlyCost and EstimatedMonthlyCost are filled from the
	// provider pricing API by Provider.Plan; nil means no price was
	// obtained and callers must not offer to create resources.
	EstimatedHourlyCost  *CostEstimate     `json:"estimated_hourly_cost"`
	EstimatedMonthlyCost *CostEstimate     `json:"estimated_monthly_cost"`
	CostEstimateCaveat   string            `json:"cost_estimate_caveat"`
	Resources            []ResourcePlan    `json:"resources"`
	ResourceNames        map[string]string `json:"resource_names"`
	Tags                 map[string]string `json:"tags"`
	SSHAllowedCIDR       string            `json:"ssh_allowed_cidr"`
	Labels               []string          `json:"labels"`
	WorkflowSnippet      string            `json:"workflow_snippet"`
	FutureDestroyCommand string            `json:"future_destroy_command"`
	Warnings             []string          `json:"warnings,omitempty"`
}

type Machine struct {
	Target      remote.Target     `json:"target"`
	Provider    state.ProviderRef `json:"provider"`
	PublicIPv4  string            `json:"public_ipv4,omitempty"`
	PublicIPv6  string            `json:"public_ipv6,omitempty"`
	ResourceIDs map[string]string `json:"resource_ids,omitempty"`
}

type ProviderStatus struct {
	Kind              string   `json:"kind"`
	Found             bool     `json:"found"`
	Status            string   `json:"status"`
	Region            string   `json:"region"`
	ServerType        string   `json:"server_type"`
	Image             string   `json:"image"`
	PublicHost        string   `json:"public_host"`
	BillableResources []string `json:"billable_resources"`
	Drift             []string `json:"drift"`
	Error             string   `json:"error,omitempty"`
}

type DestroyResult struct {
	Results []ArtifactResult `json:"results"`
	Partial bool             `json:"partial"`
	Pending []string         `json:"pending"`
}

type VerificationResult struct {
	OK                bool     `json:"ok"`
	BillableResources []string `json:"billable_resources"`
	Missing           []string `json:"missing"`
	Error             string   `json:"error,omitempty"`
}
