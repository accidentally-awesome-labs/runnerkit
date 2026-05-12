package remote

import (
	"context"
	"fmt"
	"time"
)

type Executor interface {
	Probe(ctx context.Context, target Target) (ProbeResult, error)
	Run(ctx context.Context, target Target, command Command) (Result, error)
}

type HostKeyProber interface {
	ProbeHostKey(ctx context.Context, target Target) (HostKey, error)
}

type Command struct {
	ID         string
	Script     string
	Sudo       bool
	Timeout    time.Duration
	Env        map[string]string
	RedactArgs []string
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type ProbeResult struct {
	HostKey            HostKey
	OSRelease          map[string]string
	Kernel             string
	Arch               string
	Systemd            bool
	Commands           map[string]bool
	DiskAvailableBytes int64
	// MemAvailableBytes is MemAvailable from /proc/meminfo in bytes, or -1 if unknown.
	MemAvailableBytes int64
	// SwapFreeBytes is SwapFree from /proc/meminfo in bytes, or -1 if unknown.
	SwapFreeBytes    int64
	TimeSynchronized bool
	RunnerConflict   bool
}

type RemoteError struct {
	CommandID string
	ExitCode  int
	Message   string
}

func (e RemoteError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("remote command %s failed with exit code %d", e.CommandID, e.ExitCode)
}

type UnavailableExecutor struct{}

func (UnavailableExecutor) Probe(context.Context, Target) (ProbeResult, error) {
	return ProbeResult{}, fmt.Errorf("remote SSH executor is not configured")
}

func (UnavailableExecutor) Run(context.Context, Target, Command) (Result, error) {
	return Result{}, fmt.Errorf("remote SSH executor is not configured")
}
