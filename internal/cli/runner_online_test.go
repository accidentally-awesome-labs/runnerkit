package cli

import (
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
)

// Bug 16 / Plan 06-09 — gap doc 06-GAP-byo-sudo-handling.md.
//
// Plan 06-07 attempt-13 against salar@mckee-small-desktop completed
// bootstrap end-to-end (Bugs 4-15 closed) but waitForRunnerOnline
// timed out after 6 minutes:
//
//   ERROR RunnerKit could not verify the GitHub runner came online with the expected labels.
//
// `gh api repos/.../actions/runners` confirmed the runner was online
// with labels:
//   ["self-hosted", "Linux", "X64", "runnerkit",
//    "runnerkit-accidentally-awesome-labs-dat0", "persistent"]
// — exactly what RunnerKit registered, plus GitHub's auto-added
// "Linux" + "X64" (capitalized). RunnerKit's expected label set
// included lowercase "linux" + "x64" (from labels.Build, which slugs
// values via strings.ToLower). runnerOnlineWithLabels does
// case-sensitive set-membership, so "linux" never matched "Linux"
// and the loop polled until deadline.
//
// Fix: case-insensitive comparison in runnerOnlineWithLabels. GitHub
// always emits OS + arch auto-labels in canonical CamelCase (Linux,
// macOS, Windows, X64, ARM64, ARM); RunnerKit always lowercases.
// Both are correct in their own world; the matching layer must
// normalize before comparing.

func TestRunnerOnlineWithLabels_CaseInsensitiveMatch(t *testing.T) {
	t.Parallel()
	runners := []gh.Runner{{
		Name:   "runnerkit-x",
		Status: "online",
		Labels: []string{"self-hosted", "Linux", "X64", "runnerkit", "runnerkit-x", "persistent"},
	}}
	expected := []string{"self-hosted", "runnerkit", "runnerkit-x", "linux", "x64", "persistent"}
	got, ok := runnerOnlineWithLabels(runners, "runnerkit-x", expected)
	if !ok {
		t.Fatalf("expected runner to match despite label case mismatch (Bug 16); got ok=false")
	}
	if got.Name != "runnerkit-x" {
		t.Fatalf("matched runner.Name = %q, want runnerkit-x", got.Name)
	}
}
