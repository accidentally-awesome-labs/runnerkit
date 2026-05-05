package main

import (
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// Bug 4 / Task G: regression guard — production binary must wire a
// concrete Prompter (NOT leave Prompts == nil). Without this, both
// runnerkit byo-prepare (Path C) and runnerkit up Path B fall through
// to the misleading "no TTY" branch even on real terminals.
func TestBuildDependencies_WiresPrompts(t *testing.T) {
	t.Parallel()
	deps := buildDependencies()
	if deps.Prompts == nil {
		t.Fatal("buildDependencies() must wire a non-nil Prompts implementation")
	}
	if _, ok := deps.Prompts.(ui.PasswordPrompter); !ok {
		t.Fatal("buildDependencies() Prompts must satisfy ui.PasswordPrompter for Path B + byo-prepare sudo password collection")
	}
}
