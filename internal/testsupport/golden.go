package testsupport

import (
	"strings"
	"testing"
)

func AssertEqual(t testing.TB, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func AssertNoANSI(t testing.TB, got string) {
	t.Helper()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("output contains ANSI escape sequence: %q", got)
	}
}
