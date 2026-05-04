package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

func TestProviderStructsMarshalStableJSONKeys(t *testing.T) {
	input := ProvisionInput{RepoFullName: "owner/name", RunnerName: "runnerkit-owner-name-local", Labels: []string{"self-hosted"}, WorkflowSnippet: "runs-on: [self-hosted]", Profile: DefaultHetznerProfile(), SSHAllowedCIDR: HetznerDefaultSSHAllowedCIDR, StateID: "state-1", CreatedAt: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)}
	plan := HetznerProvisionPlan(input)
	payload := map[string]any{
		"profile":       DefaultHetznerProfile(),
		"resource":      ResourcePlan{Name: "runnerkit-owner-name-local", Kind: "server", Billable: true, Action: "create"},
		"artifact":      ArtifactResult{Artifact: "server", Status: "planned", Message: "ok"},
		"input":         input,
		"validation":    ValidationResult{OK: true, Source: "HCLOUD_TOKEN"},
		"provision_out": ProvisionResult{Machine: Machine{Target: remote.Target{Host: "203.0.113.1", User: "runnerkit-admin", Port: 22}, Provider: state.ProviderRef{Kind: "hetzner"}}, CreatedResourceIDs: map[string]string{"server": "1"}},
		"plan":          plan,
		"status":        ProviderStatus{Kind: "hetzner", Found: true, Status: "running", Region: "fsn1", ServerType: "cpx22", Image: "ubuntu-24.04", PublicHost: "203.0.113.1", BillableResources: []string{"server"}, Drift: []string{}},
		"destroy":       DestroyResult{Results: []ArtifactResult{{Artifact: "server", Status: "deleted"}}, Pending: []string{}},
		"verify":        VerificationResult{OK: true, BillableResources: []string{}, Missing: []string{"server"}},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal provider structs: %v", err)
	}
	for _, key := range []string{"estimated_hourly_cost", "estimated_monthly_cost", "cost_estimate_caveat", "created_resource_ids", "future_destroy_command", "billable_resources", "checkpoint_required"} {
		if !strings.Contains(string(encoded), `"`+key+`"`) {
			t.Fatalf("encoded JSON missing key %q: %s", key, encoded)
		}
	}
}

func TestDefaultHetznerProfileAndPlanValues(t *testing.T) {
	profile := DefaultHetznerProfile()
	if profile.Provider != "hetzner" || profile.Region != "fsn1" || profile.ServerType != "cpx22" || profile.Image != "ubuntu-24.04" || profile.SSHUser != "runnerkit-admin" {
		t.Fatalf("unexpected default profile: %#v", profile)
	}
	if profile.CostEstimateCaveat != HetznerCostEstimateCaveat || !strings.Contains(profile.CostEstimateCaveat, "Estimated cost is approximate") {
		t.Fatalf("unexpected caveat: %q", profile.CostEstimateCaveat)
	}
	input := ProvisionInput{RepoFullName: "owner/name", RunnerName: "runnerkit-owner-name-local", Labels: []string{"self-hosted", "runnerkit"}, WorkflowSnippet: "runs-on: [self-hosted, runnerkit]", Profile: profile, SSHAllowedCIDR: "0.0.0.0/0", StateID: "state-123", CreatedAt: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)}
	plan := HetznerProvisionPlan(input)
	for _, key := range []string{"server", "ssh_key", "firewall"} {
		if plan.ResourceNames[key] == "" {
			t.Fatalf("resource name %q missing: %#v", key, plan.ResourceNames)
		}
	}
	for key, want := range map[string]string{"runnerkit": "true", "managed": "true", "repo": "owner/name", "runner": "runnerkit-owner-name-local", "state_id": "state-123", "mode": "persistent", "created_at": "2026-04-30T12:00:00Z"} {
		if got := plan.Tags[key]; got != want {
			t.Fatalf("tag %s = %q, want %q (tags %#v)", key, got, want, plan.Tags)
		}
	}
	if plan.FutureDestroyCommand != "runnerkit destroy --repo owner/name" {
		t.Fatalf("future destroy = %q", plan.FutureDestroyCommand)
	}
}

func TestHetznerOwnershipTagsRespectModeOverride(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	persistent := HetznerOwnershipTags(ProvisionInput{RepoFullName: "owner/name", RunnerName: "runnerkit-owner-name-local", StateID: "state-1", CreatedAt: now})
	if persistent["mode"] != "persistent" {
		t.Fatalf("default mode tag = %q, want persistent", persistent["mode"])
	}
	ephemeral := HetznerOwnershipTags(ProvisionInput{RepoFullName: "owner/name", RunnerName: "runnerkit-owner-name-ephemeral-abc123", StateID: "state-1", CreatedAt: now, Mode: "ephemeral"})
	if ephemeral["mode"] != "ephemeral" {
		t.Fatalf("ephemeral mode tag = %q, want ephemeral", ephemeral["mode"])
	}
}

func TestProvisionErrorWrapsCause(t *testing.T) {
	cause := errors.New("boom")
	err := &ProvisionError{Stage: "server", Err: cause}
	if !strings.Contains(err.Error(), "server") || !errors.Is(err, cause) {
		t.Fatalf("ProvisionError did not wrap cause: %v", err)
	}
}

func TestFakeProviderCounters(t *testing.T) {
	fake := &FakeProvider{NameValue: "hetzner"}
	input := ProvisionInput{RepoFullName: "owner/name", Profile: DefaultHetznerProfile()}
	if fake.Name() != "hetzner" {
		t.Fatalf("Name() = %q", fake.Name())
	}
	if _, err := fake.Validate(context.Background(), input); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if _, err := fake.Plan(context.Background(), input); err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if _, err := fake.Provision(context.Background(), input); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if _, err := fake.WaitReady(context.Background(), Machine{}); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if _, err := fake.Describe(context.Background(), state.ProviderRef{Kind: "hetzner"}); err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if _, err := fake.Destroy(context.Background(), state.ProviderRef{Kind: "hetzner"}); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := fake.VerifyDestroyed(context.Background(), state.ProviderRef{Kind: "hetzner"}); err != nil {
		t.Fatalf("VerifyDestroyed: %v", err)
	}
	if fake.ValidateCalls != 1 || fake.PlanCalls != 1 || fake.ProvisionCalls != 1 || fake.WaitReadyCalls != 1 || fake.DescribeCalls != 1 || fake.DestroyCalls != 1 || fake.VerifyDestroyedCalls != 1 {
		t.Fatalf("unexpected counters: %#v", fake)
	}
}
