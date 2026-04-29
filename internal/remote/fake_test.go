package remote

import (
	"context"
	"testing"
)

type fakeExecutor struct {
	probe ProbeResult
	runs  []Command
}

func (f *fakeExecutor) Probe(context.Context, Target) (ProbeResult, error) { return f.probe, nil }
func (f *fakeExecutor) Run(_ context.Context, _ Target, command Command) (Result, error) {
	f.runs = append(f.runs, command)
	return Result{ExitCode: 0}, nil
}

func TestFakeExecutorSatisfiesExecutor(t *testing.T) {
	var exec Executor = &fakeExecutor{probe: ProbeResult{HostKey: HostKey{Fingerprint: "SHA256:test"}}}
	probe, err := exec.Probe(context.Background(), Target{User: "alice", Host: "example.com", Port: 22})
	if err != nil || probe.HostKey.Fingerprint == "" {
		t.Fatalf("Probe() = %#v, %v", probe, err)
	}
}
