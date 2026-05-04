package cli

import (
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/update"
)

// maybeShowUpdateNotice is invoked via deferred call from runUp,
// runStatus, and runDoctor. It is the integration seam for the lazy
// update check (D-06). It MUST be silent on every failure path; never
// block; never error. The notice is printed to deps.Err so it does not
// interleave with structured stdout output.
func maybeShowUpdateNotice(deps Dependencies, jsonOutput bool) {
	if jsonOutput {
		return
	}
	stateDir := deps.StateBaseDir
	if stateDir == "" {
		stateDir = state.DefaultBaseDir()
	}
	now := deps.Clock
	update.MaybePrint(jsonOutput, deps.Version, update.Deps{
		StateDir: stateDir,
		Now:      now,
	}, deps.Err)
}
