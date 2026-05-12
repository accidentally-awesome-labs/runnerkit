package cli

import "testing"

func TestByoChecklistStepStates(t *testing.T) {
	t.Parallel()
	n := 5
	if s := byoChecklistStepStates(0, n); s[0] != "active" || s[1] != "pending" {
		t.Fatalf("%v", s)
	}
	if s := byoChecklistStepStates(5, n); s[4] != "done" {
		t.Fatalf("all done: %v", s)
	}
}
