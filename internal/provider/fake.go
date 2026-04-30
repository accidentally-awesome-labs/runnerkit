package provider

import (
	"context"

	"github.com/salar/runnerkit/internal/state"
)

type FakeProvider struct {
	NameValue string

	ValidateResult VerificationPair
	PlanResult     ProvisionPlan
	ProvisionOut   ProvisionResult
	WaitReadyOut   Machine
	DescribeOut    ProviderStatus
	DestroyOut     DestroyResult
	VerifyOut      VerificationResult

	ValidateErr        error
	PlanErr            error
	ProvisionErr       error
	WaitReadyErr       error
	DescribeErr        error
	DestroyErr         error
	VerifyDestroyedErr error

	ValidateCalls        int
	PlanCalls            int
	ProvisionCalls       int
	WaitReadyCalls       int
	DescribeCalls        int
	DestroyCalls         int
	VerifyDestroyedCalls int

	ValidateInputs []ProvisionInput
	PlanInputs     []ProvisionInput
	ProvisionInput []ProvisionInput
}

type VerificationPair struct {
	Result ValidationResult
}

func (f *FakeProvider) Name() string {
	if f.NameValue == "" {
		return HetznerProvider
	}
	return f.NameValue
}

func (f *FakeProvider) Validate(_ context.Context, input ProvisionInput) (ValidationResult, error) {
	f.ValidateCalls++
	f.ValidateInputs = append(f.ValidateInputs, input)
	if f.ValidateResult.Result.OK || f.ValidateResult.Result.Source != "" || len(f.ValidateResult.Result.Remediation) > 0 {
		return f.ValidateResult.Result, f.ValidateErr
	}
	return ValidationResult{OK: true, Source: "fake"}, f.ValidateErr
}

func (f *FakeProvider) Plan(_ context.Context, input ProvisionInput) (ProvisionPlan, error) {
	f.PlanCalls++
	f.PlanInputs = append(f.PlanInputs, input)
	if f.PlanResult.Provider != "" {
		return f.PlanResult, f.PlanErr
	}
	return HetznerProvisionPlan(input), f.PlanErr
}

func (f *FakeProvider) Provision(_ context.Context, input ProvisionInput) (ProvisionResult, error) {
	f.ProvisionCalls++
	f.ProvisionInput = append(f.ProvisionInput, input)
	return f.ProvisionOut, f.ProvisionErr
}

func (f *FakeProvider) WaitReady(_ context.Context, machine Machine) (Machine, error) {
	f.WaitReadyCalls++
	if f.WaitReadyOut.Target.Host != "" || f.WaitReadyOut.Provider.Kind != "" {
		return f.WaitReadyOut, f.WaitReadyErr
	}
	return machine, f.WaitReadyErr
}

func (f *FakeProvider) Describe(_ context.Context, ref state.ProviderRef) (ProviderStatus, error) {
	f.DescribeCalls++
	if f.DescribeOut.Kind != "" {
		return f.DescribeOut, f.DescribeErr
	}
	return ProviderStatus{Kind: ref.Kind, Found: true}, f.DescribeErr
}

func (f *FakeProvider) Destroy(_ context.Context, _ state.ProviderRef) (DestroyResult, error) {
	f.DestroyCalls++
	return f.DestroyOut, f.DestroyErr
}

func (f *FakeProvider) VerifyDestroyed(_ context.Context, _ state.ProviderRef) (VerificationResult, error) {
	f.VerifyDestroyedCalls++
	return f.VerifyOut, f.VerifyDestroyedErr
}
