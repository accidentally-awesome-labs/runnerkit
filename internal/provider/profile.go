package provider

import (
	"strings"
	"time"
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
	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = "persistent"
	}
	return map[string]string{
		"runnerkit":  "true",
		"managed":    "true",
		"repo":       input.RepoFullName,
		"runner":     input.RunnerName,
		"state_id":   input.StateID,
		"mode":       mode,
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
