package testsupport

import (
	"encoding/json"
	"strings"
	"testing"
)

func DecodeJSON[T any](t testing.TB, input string) T {
	t.Helper()
	var out T
	if err := json.Unmarshal([]byte(input), &out); err != nil {
		t.Fatalf("decode json: %v\n%s", err, input)
	}
	return out
}

func RequireContains(t testing.TB, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}

func RequireNotContains(t testing.TB, got string, forbidden string) {
	t.Helper()
	if strings.Contains(got, forbidden) {
		t.Fatalf("output contains forbidden %q:\n%s", forbidden, got)
	}
}

func RequireRedactionsApplied(t testing.TB, got string) {
	t.Helper()
	RequireContains(t, got, `"redactions_applied":true`)
}
