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

// A-21: this release ships without a passing real-job BYO gate, so the
// release binary must refuse BYO setup without --accept-known-issues.
func TestBuildDependencies_RefusesUnsupportedBYO(t *testing.T) {
	t.Parallel()
	if !buildDependencies().BYOUnsupportedRelease {
		t.Fatal("buildDependencies() must set BYOUnsupportedRelease in a release that does not support BYO")
	}
}
