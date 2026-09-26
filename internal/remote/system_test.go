package remote

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// TestSSHArgsSuppressKnownHostsNoise guards P1-15: without
// LogLevel=ERROR every bootstrap failure's stderr started with ssh's
// "Permanently added ... to the list of known hosts" warning.
func TestSSHArgsSuppressKnownHostsNoise(t *testing.T) {
	args := sshArgs(Target{User: "alice", Host: "example.com", Port: 22}, "bash -s")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-o LogLevel=ERROR") {
		t.Fatalf("sshArgs missing -o LogLevel=ERROR: %v", args)
	}
	// Options must precede the destination, or ssh treats them as the
	// remote command.
	if args[len(args)-2] != "alice@example.com" || args[len(args)-1] != "bash -s" {
		t.Fatalf("destination/remote command not last: %v", args)
	}
}

func TestRemoteErrorUnwrapsUnderlyingError(t *testing.T) {
	inner := &exec.ExitError{}
	err := error(RemoteError{CommandID: "setup_runner_image", ExitCode: 4, Err: inner})
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr != inner {
		t.Fatalf("errors.As(*exec.ExitError) failed for %v", err)
	}
	if got := err.Error(); got != "remote command setup_runner_image failed with exit code 4" {
		t.Fatalf("Error() = %q", got)
	}
}
