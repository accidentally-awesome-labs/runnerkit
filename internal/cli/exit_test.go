package cli

import (
	"testing"

	"github.com/salar/runnerkit/internal/state"
)

func TestExitCodeStateSchemaTooNew(t *testing.T) {
	if ExitStateSchemaTooNew != 7 {
		t.Fatalf("ExitStateSchemaTooNew = %d, want 7", ExitStateSchemaTooNew)
	}
	err := NewExitError(ExitStateSchemaTooNew, state.ErrSchemaTooNew)
	if got := ExitCode(err); got != 7 {
		t.Fatalf("ExitCode(ExitStateSchemaTooNew) = %d, want 7", got)
	}
}
