package ui

import (
	"fmt"
	"strings"
	"time"
)

// ChecklistStepStatus is the progress state of one checklist row.
type ChecklistStepStatus string

const (
	ChecklistDone   ChecklistStepStatus = "done"
	ChecklistActive ChecklistStepStatus = "active"
	ChecklistTodo   ChecklistStepStatus = "pending"
)

// ChecklistStep is one row in a progress checklist.
type ChecklistStep struct {
	ID       string
	Title    string
	Status   ChecklistStepStatus
	Duration time.Duration
}

// RenderChecklist formats steps for human-readable CLI output.
func RenderChecklist(steps []ChecklistStep, caps TerminalCapabilities) string {
	var b strings.Builder
	for _, s := range steps {
		prefix := checklistGlyph(s.Status, caps.ASCII)
		line := prefix + " " + s.Title
		if s.Status == ChecklistDone && s.Duration > 0 {
			line += fmt.Sprintf(" (%s)", s.Duration.Round(time.Millisecond))
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func checklistGlyph(st ChecklistStepStatus, ascii bool) string {
	if ascii {
		switch st {
		case ChecklistDone:
			return "[x]"
		case ChecklistActive:
			return "[>]"
		default:
			return "[ ]"
		}
	}
	switch st {
	case ChecklistDone:
		return "[✓]"
	case ChecklistActive:
		return "[→]"
	default:
		return "[ ]"
	}
}
