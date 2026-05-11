package main

import (
	"testing"
)

// Bug 4 / Task G: regression guard — production binary must wire a
// concrete Prompter (NOT leave Prompts == nil).
func TestBuildDependencies_WiresPrompts(t *testing.T) {
	t.Parallel()
	deps := buildDependencies()
	if deps.Prompts == nil {
		t.Fatal("buildDependencies() must wire a non-nil Prompts implementation")
	}
}
