package preflight

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// Bugs 7 + 8 — gap doc 06-GAP-byo-sudo-handling.md.
//
// Bug 7: internal/remote/system.go::SystemExecutor.Run returns
// *exec.ExitError for any non-zero remote exit. The preflight switch
// at internal/preflight/checks.go::Run previously required
// probeErr == nil for every classification branch, so a real ssh
// probe that exits non-zero (e.g. `sudo -n install --version` exit 1
// with stderr "sudo: a password is required") would never reach the
// WARNING branch -- it would fall into the default ERROR branch
// ("sudo probe failed: ..."). Plan 06-07 attempt-5 surfaced this as
// a hard preflight failure; the fix made the switch tolerant of
// *exec.ExitError. The probe Script literal was later changed from
// `sudo -n true` to `sudo -n install --version >/dev/null` (Plan
// 06-13, Bug 31) so the probe binds to byo-prepare's scoped sudoers
// allowlist; the Bug 7 stderr-classification contract is unchanged.
//
// Bug 8: runNetworkCheck calls `curl -fsS https://github.com/...`. The
// `-f` flag makes curl exit 22 on HTTP 4xx, so an anonymous probe of
// api.github.com that hits the 60-req/hr rate limit returns exit 22
// even though the network is reachable. The preflight check then
// reports a connectivity failure rather than the actual cause.

type errReturningFake struct {
	probe      remote.ProbeResult
	runResults map[string]remote.Result
	runErrors  map[string]error
}

func (f errReturningFake) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return f.probe, nil
}
func (f errReturningFake) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	res := remote.Result{ExitCode: 0}
	if f.runResults != nil {
		if r, ok := f.runResults[command.ID]; ok {
			res = r
		}
	}
	var err error
	if f.runErrors != nil {
		err = f.runErrors[command.ID]
	}
	return res, err
}

func TestCheckPrivilege_PasswordRequired_WhenExecutorReturnsExitErr(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	exec := errReturningFake{
		probe: probe,
		runResults: map[string]remote.Result{
			"probe_sudo_n": {ExitCode: 1, Stderr: "sudo: a password is required"},
		},
		runErrors: map[string]error{
			"probe_sudo_n": errors.New("ssh: command exited 1"),
		},
	}
	target := remote.Target{User: "salar", Host: "mckee", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	res, ok := report.Result(CheckPrivilegePasswordReq)
	if !ok {
		t.Fatalf("expected CheckPrivilegePasswordReq result. got results: %+v", report.Results)
	}
	if res.Severity != SeverityWarning {
		t.Fatalf("expected WARNING severity for password-required when executor returns ExitError, got %v", res.Severity)
	}
	for _, r := range report.Results {
		if r.ID == CheckPrivilege && r.Severity == SeverityFailure {
			t.Fatalf("did not expect CheckPrivilege FAILURE for ExitError-with-password-stderr; got %#v", r)
		}
	}
}

func TestCheckPrivilege_NoSudoers_WhenExecutorReturnsExitErr(t *testing.T) {
	probe := passingProbe("ubuntu", "x86_64")
	exec := errReturningFake{
		probe: probe,
		runResults: map[string]remote.Result{
			"probe_sudo_n": {ExitCode: 1, Stderr: "alice may not run sudo on mckee"},
		},
		runErrors: map[string]error{
			"probe_sudo_n": errors.New("ssh: command exited 1"),
		},
	}
	target := remote.Target{User: "alice", Host: "mckee", Port: 22}
	report, err := Run(context.Background(), exec, target, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if _, ok := report.Result(CheckPrivilegeNoSudo); !ok {
		t.Fatalf("expected CheckPrivilegeNoSudo result. got results: %+v", report.Results)
	}
}

func TestRunNetworkCheck_Script_DoesNotUseFailFlag(t *testing.T) {
	t.Parallel()
	// Bug 8 regression guard: the github + api scripts must not pass
	// -f to curl, otherwise an anonymous rate-limit response (HTTP 403
	// from api.github.com when the host IP exhausts the 60 req/hr
	// anonymous limit) is misclassified as a connectivity failure.
	src, err := readChecksGoSource()
	if err != nil {
		t.Fatalf("read checks.go: %v", err)
	}
	for _, marker := range []string{
		"host.network.github.github",
		"host.network.github.api",
	} {
		if !strings.Contains(src, marker) {
			t.Fatalf("checks.go missing marker %q (refactor without breaking the regression guard?)", marker)
		}
	}
	// The probe must use a flag set that distinguishes "network reachable"
	// (any HTTP response) from "transport error" (DNS, TLS, connection
	// refused). `curl -sS -o /dev/null -w '%%{http_code}'` is the
	// canonical form. The legacy `-fsS` form is forbidden because it
	// conflates HTTP-level failures with transport failures.
	if strings.Contains(src, `"curl -fsS https://github.com`) ||
		strings.Contains(src, `"curl -fsS https://api.github.com`) {
		t.Fatalf("checks.go still uses `curl -fsS` for github probes — Bug 8 not closed")
	}
}

func readChecksGoSource() (string, error) {
	b, err := os.ReadFile("checks.go")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
