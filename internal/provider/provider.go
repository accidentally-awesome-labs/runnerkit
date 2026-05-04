package provider

import (
	"context"
	"fmt"
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
	Provider             string `json:"provider"`
	Region               string `json:"region"`
	ServerType           string `json:"server_type"`
	Image                string `json:"image"`
	SSHUser              string `json:"ssh_user"`
	EstimatedHourlyCost  string `json:"estimated_hourly_cost"`
	EstimatedMonthlyCost string `json:"estimated_monthly_cost"`
	CostEstimateCaveat   string `json:"cost_estimate_caveat"`
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
	Provider             string            `json:"provider"`
	Region               string            `json:"region"`
	ServerType           string            `json:"server_type"`
	Image                string            `json:"image"`
	SSHUser              string            `json:"ssh_user"`
	EstimatedHourlyCost  string            `json:"estimated_hourly_cost"`
	EstimatedMonthlyCost string            `json:"estimated_monthly_cost"`
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
