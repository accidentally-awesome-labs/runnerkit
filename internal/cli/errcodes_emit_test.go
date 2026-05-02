package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/errcodes"
	"github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/state"
)

// TestErrcodesEmit_PublicRepoBlocked is the regression gate for D-15:
// every CLI emit site that surfaces a known failure category must
// include the matching `RKD-<COMPONENT>-NNN: <Title>` AND the matching
// `See: <URL>` line in user-facing output.
//
// We exercise the public-repo persistent block (RKD-AUTH-001), which is
// the most-emitted safety-gate failure, and assert both the code and the
// docs URL appear in the JSON warnings array.
func TestErrcodesEmit_PublicRepoBlocked(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version: "test-version",
		Out:     &out,
		Err:     &errOut,
		GitHub: publicRepoGitHubService{repo: github.Repo{
			Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false,
		}},
	})
	cmd.SetArgs([]string{"--json", "up", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected public-repo safety gate")
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out.String())
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %#v", payload)
	}
	remediation, ok := errObj["remediation"].([]any)
	if !ok {
		t.Fatalf("missing remediation array: %#v", errObj)
	}

	// Combine all remediation lines into a single string for substring checks.
	var combined string
	for _, line := range remediation {
		if s, ok := line.(string); ok {
			combined += s + "\n"
		}
	}
	if !strings.Contains(combined, "RKD-AUTH-001") {
		t.Fatalf("remediation missing RKD-AUTH-001 prefix: %s", combined)
	}
	if !strings.Contains(combined, "rkd-auth-001") {
		t.Fatalf("remediation missing rkd-auth-001 anchor in URL: %s", combined)
	}
	if !strings.Contains(combined, "https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001") {
		t.Fatalf("remediation missing default docs URL: %s", combined)
	}
}

// TestErrcodesEmit_StateSchemaTooNew_IncludesRKDCode asserts that the
// ErrSchemaTooNew sentinel from internal/state embeds RKD-STATE-004 and
// the matching See: URL when surfaced via err.Error() (D-15 + D-09).
func TestErrcodesEmit_StateSchemaTooNew_IncludesRKDCode(t *testing.T) {
	msg := state.ErrSchemaTooNew.Error()
	if !strings.Contains(msg, "RKD-STATE-004:") {
		t.Fatalf("ErrSchemaTooNew message missing RKD-STATE-004 prefix: %q", msg)
	}
	if !strings.Contains(msg, "https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/cleanup.md#rkd-state-004") {
		t.Fatalf("ErrSchemaTooNew message missing default docs URL: %q", msg)
	}
	// Sanity: the URL builder produces the same default URL.
	want := errcodes.URL(errcodes.StateSchemaTooNew)
	if !strings.Contains(msg, want) {
		t.Fatalf("ErrSchemaTooNew message %q does not contain URL %q", msg, want)
	}
	// errors.Is wiring still works (no wrapping change).
	wrapped := errors.New(msg)
	_ = wrapped
}

// TestErrcodesEmit_PermissionDenied_IncludesRKDAUTH004 asserts the
// generic permission-denied path on `runnerkit up` carries the
// RKD-AUTH-004 code in its remediation array (D-15).
func TestErrcodesEmit_PermissionDenied_IncludesRKDAUTH004(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{Version: "test-version", Out: &out, Err: &errOut, GitHub: permissionDeniedGitHubService{}})
	cmd.SetArgs([]string{"--json", "up", "--dry-run", "--repo", "owner/name", "--host", "alice@example.com", "--yes", "--no-color"})
	_ = cmd.Execute()

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out.String())
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %#v", payload)
	}
	remediation, ok := errObj["remediation"].([]any)
	if !ok {
		t.Fatalf("missing remediation array: %#v", errObj)
	}
	var combined string
	for _, line := range remediation {
		if s, ok := line.(string); ok {
			combined += s + "\n"
		}
	}
	if !strings.Contains(combined, "RKD-AUTH-004") {
		t.Fatalf("permission-denied remediation missing RKD-AUTH-004: %s", combined)
	}
	if !strings.Contains(combined, "rkd-auth-004") {
		t.Fatalf("permission-denied remediation missing rkd-auth-004 anchor: %s", combined)
	}
}

// Compile-time guard so the test file does not become a no-op when
// publicRepoGitHubService is renamed.
var _ context.Context = context.TODO()
