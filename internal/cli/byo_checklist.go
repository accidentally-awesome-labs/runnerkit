package cli

import (
	"fmt"
	"io"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/checkliststore"
)

func byoChecklistHostRef(t remote.Target) string {
	if t.Raw != "" {
		return t.Raw
	}
	if t.User != "" && t.Host != "" {
		return fmt.Sprintf("%s@%s", t.User, t.Host)
	}
	return t.Host
}

var byoChecklistTitles = []string{
	"Detect host / preflight",
	"Confirm install plan",
	"GitHub registration token",
	"Remote bootstrap install",
	"Verify runner online",
}

func byoChecklistStepStates(doneCount int, n int) []string {
	allDone := doneCount >= n
	out := make([]string, n)
	for i := 0; i < n; i++ {
		switch {
		case allDone || i < doneCount:
			out[i] = "done"
		case i == doneCount:
			out[i] = "active"
		default:
			out[i] = "pending"
		}
	}
	return out
}

// syncBYOInstallChecklist persists progress for BYO `up` / `register` (resumable across re-runs).
// doneCount is the number of steps fully completed before the active step; use n (len titles) for all done.
func syncBYOInstallChecklist(stateBase, repo string, target remote.Target, doneCount int) error {
	id := checkliststore.BYORegisterSessionID(repo, byoChecklistHostRef(target))
	n := len(byoChecklistTitles)
	states := byoChecklistStepStates(doneCount, n)
	steps := make([]checkliststore.Step, n)
	for i, title := range byoChecklistTitles {
		steps[i] = checkliststore.Step{ID: fmt.Sprintf("byo_%d", i), Title: title, Status: states[i]}
	}
	return checkliststore.Save(stateBase, &checkliststore.Doc{SessionID: id, Steps: steps})
}

func writeBYOChecklistHuman(out io.Writer, caps ui.TerminalCapabilities, stateBase, repo string, target remote.Target, doneCount int) {
	if out == nil {
		return
	}
	id := checkliststore.BYORegisterSessionID(repo, byoChecklistHostRef(target))
	_ = syncBYOInstallChecklist(stateBase, repo, target, doneCount)
	n := len(byoChecklistTitles)
	states := byoChecklistStepStates(doneCount, n)
	var uiSteps []ui.ChecklistStep
	for i, title := range byoChecklistTitles {
		var st ui.ChecklistStepStatus
		switch states[i] {
		case "done":
			st = ui.ChecklistDone
		case "active":
			st = ui.ChecklistActive
		default:
			st = ui.ChecklistTodo
		}
		uiSteps = append(uiSteps, ui.ChecklistStep{Title: title, Status: st})
	}
	_, _ = fmt.Fprintf(out, "Progress (%s):\n%s", id, ui.RenderChecklist(uiSteps, caps))
}
