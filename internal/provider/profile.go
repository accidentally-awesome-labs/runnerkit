package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/salar/runnerkit/internal/provider/hetzner"
	"github.com/salar/runnerkit/internal/state"
)

const (
	HetznerProvider              = "hetzner"
	HetznerDefaultRegion         = "fsn1"
	HetznerDefaultServerType     = "cpx22"
	HetznerDefaultImage          = "ubuntu-24.04"
	HetznerDefaultSSHUser        = "runnerkit-admin"
	HetznerDefaultSSHAllowedCIDR = "0.0.0.0/0"
	HetznerCostEstimateCaveat    = "Estimated cost is approximate. Provider pricing varies by region and time; billing stops only after RunnerKit-created billable resources are deleted or verified non-billable."
	HetznerTagRunnerKitTrue      = "runnerkit=true"
	HetznerTagManagedTrue        = "managed=true"
)

func DefaultHetznerProfile() Profile {
	return Profile{
		Provider:             HetznerProvider,
		Region:               HetznerDefaultRegion,
		ServerType:           HetznerDefaultServerType,
		Image:                HetznerDefaultImage,
		SSHUser:              HetznerDefaultSSHUser,
		EstimatedHourlyCost:  "approx €0.0081/hour",
		EstimatedMonthlyCost: "approx €4.90/month",
		CostEstimateCaveat:   HetznerCostEstimateCaveat,
	}
}

func HetznerProvisionPlan(input ProvisionInput) ProvisionPlan {
	profile := input.Profile
	if profile.Provider == "" {
		profile = DefaultHetznerProfile()
	}
	profile = withHetznerDefaults(profile)
	if input.SSHAllowedCIDR == "" {
		input.SSHAllowedCIDR = HetznerDefaultSSHAllowedCIDR
	}
	names := HetznerResourceNames(input)
	return ProvisionPlan{
		Provider:             profile.Provider,
		Region:               profile.Region,
		ServerType:           profile.ServerType,
		Image:                profile.Image,
		SSHUser:              profile.SSHUser,
		EstimatedHourlyCost:  profile.EstimatedHourlyCost,
		EstimatedMonthlyCost: profile.EstimatedMonthlyCost,
		CostEstimateCaveat:   profile.CostEstimateCaveat,
		Resources: []ResourcePlan{
			{Name: names["server"], Kind: "server", Billable: true, Action: "create"},
			{Name: names["ssh_key"], Kind: "ssh_key", Billable: false, Action: "create"},
			{Name: names["firewall"], Kind: "firewall", Billable: false, Action: "create"},
			{Name: "public IPv4/IPv6", Kind: "public_ip", Billable: true, Action: "attach"},
		},
		ResourceNames:        names,
		Tags:                 HetznerOwnershipTags(input),
		SSHAllowedCIDR:       input.SSHAllowedCIDR,
		Labels:               append([]string(nil), input.Labels...),
		WorkflowSnippet:      input.WorkflowSnippet,
		FutureDestroyCommand: "runnerkit destroy --repo " + input.RepoFullName,
	}
}

func HetznerResourceNames(input ProvisionInput) map[string]string {
	base := resourceBase(input)
	return map[string]string{
		"server":   base,
		"ssh_key":  base + "-ssh-key",
		"firewall": base + "-firewall",
	}
}

func HetznerOwnershipTags(input ProvisionInput) map[string]string {
	createdAt := input.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return map[string]string{
		"runnerkit":  "true",
		"managed":    "true",
		"repo":       input.RepoFullName,
		"runner":     input.RunnerName,
		"state_id":   input.StateID,
		"mode":       "persistent",
		"created_at": createdAt.Format(time.RFC3339),
	}
}

func withHetznerDefaults(profile Profile) Profile {
	defaults := DefaultHetznerProfile()
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

func resourceBase(input ProvisionInput) string {
	if input.RunnerName != "" {
		return sanitizeName(input.RunnerName)
	}
	if input.StateID != "" {
		return "runnerkit-" + sanitizeName(input.StateID)
	}
	return "runnerkit-" + sanitizeName(input.RepoFullName)
}

func sanitizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
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
	if !strings.HasPrefix(out, "runnerkit-") {
		out = "runnerkit-" + out
	}
	return out
}

type HetznerPlanProvider struct {
	Env map[string]string
}

func NewHetznerPlanProvider(env map[string]string) *HetznerPlanProvider {
	return &HetznerPlanProvider{Env: env}
}

func (p *HetznerPlanProvider) Name() string { return HetznerProvider }

func (p *HetznerPlanProvider) Validate(_ context.Context, _ ProvisionInput) (ValidationResult, error) {
	token, err := hetzner.ResolveToken(p.Env)
	if err != nil {
		var missing *hetzner.MissingTokenError
		if errors.As(err, &missing) {
			return ValidationResult{OK: false, Remediation: missing.Remediation}, err
		}
		return ValidationResult{OK: false}, err
	}
	return ValidationResult{OK: true, Source: token.Source}, nil
}

func (p *HetznerPlanProvider) Plan(_ context.Context, input ProvisionInput) (ProvisionPlan, error) {
	return HetznerProvisionPlan(input), nil
}

func (p *HetznerPlanProvider) Provision(_ context.Context, _ ProvisionInput) (ProvisionResult, error) {
	return ProvisionResult{}, errors.New("hetzner provisioning is not implemented in this plan")
}

func (p *HetznerPlanProvider) WaitReady(_ context.Context, machine Machine) (Machine, error) {
	return machine, errors.New("hetzner readiness is not implemented in this plan")
}

func (p *HetznerPlanProvider) Describe(_ context.Context, ref state.ProviderRef) (ProviderStatus, error) {
	return ProviderStatus{Kind: ref.Kind, Region: ref.Region, Found: false}, nil
}

func (p *HetznerPlanProvider) Destroy(_ context.Context, _ state.ProviderRef) (DestroyResult, error) {
	return DestroyResult{}, errors.New("hetzner destroy is not implemented in this plan")
}

func (p *HetznerPlanProvider) VerifyDestroyed(_ context.Context, _ state.ProviderRef) (VerificationResult, error) {
	return VerificationResult{OK: false}, errors.New("hetzner destroy verification is not implemented in this plan")
}
