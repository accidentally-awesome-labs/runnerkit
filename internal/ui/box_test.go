package ui

import (
	"strings"
	"testing"
)

func TestRenderBoxed_ASCII(t *testing.T) {
	t.Parallel()
	got := RenderBoxed("user@host", "runnerkit status --repo o/r", "Do the thing.", false, 60)
	want := `Copy and paste this on user@host:

Do the thing.

+----------------------------------------------------------+
| runnerkit status --repo o/r                              |
+----------------------------------------------------------+`
	if got != want {
		t.Fatalf("RenderBoxed mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestRenderBoxed_Unicode(t *testing.T) {
	t.Parallel()
	got := RenderBoxed("h", "echo hi", "", true, 40)
	want := `Copy and paste this on h:

┌──────────────────────────────────────┐
│ echo hi                              │
└──────────────────────────────────────┘`
	if got != want {
		t.Fatalf("RenderBoxed unicode mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestRenderBoxed_wrap(t *testing.T) {
	t.Parallel()
	long := "one two three four five six seven eight nine ten"
	got := RenderBoxed("", long, "", false, 36)
	lines := strings.Split(got, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected multiple lines, got %d lines:\n%s", len(lines), got)
	}
	if !strings.Contains(got, "one") || !strings.Contains(got, "ten") {
		t.Fatalf("wrapped output missing tokens:\n%s", got)
	}
}
