package ui

import (
	"strings"
	"testing"
)

func TestRenderChecklist(t *testing.T) {
	t.Parallel()
	steps := []ChecklistStep{
		{Title: "One", Status: ChecklistDone},
		{Title: "Two", Status: ChecklistActive},
		{Title: "Three", Status: ChecklistTodo},
	}
	out := RenderChecklist(steps, TerminalCapabilities{ASCII: true})
	if !strings.Contains(out, "[x] One") || !strings.Contains(out, "[>] Two") {
		t.Fatalf("ascii checklist:\n%s", out)
	}
}
