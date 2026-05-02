package state

import (
	"errors"
	"fmt"

	"github.com/salar/runnerkit/internal/errcodes"
)

// ErrSchemaTooNew is returned when state.json was written by a newer
// RunnerKit than the current binary knows about. Refuse-to-mutate per
// CONTEXT decision D-09. Maps to ExitStateSchemaTooNew (=7) in cli/exit.go.
//
// The error text embeds the stable RKD-STATE-004 code and a See: URL so
// callers that surface err.Error() directly to users always include the
// canonical troubleshooting reference (D-15).
var ErrSchemaTooNew = errors.New(errcodes.FormatLine(errcodes.StateSchemaTooNew) +
	"\n\nrunnerkit state schema_version is newer than this CLI knows; upgrade RunnerKit (run `runnerkit upgrade`) to read this state")

type migrationFn func(State) (State, error)

// forwardMigrations maps fromVersion -> migration that produces (fromVersion+1).
// Add entries here ONLY in forward order. Never delete; never renumber.
var forwardMigrations = map[string]migrationFn{
	"1": migrateV1ToV2,
}

// Migrate runs forward-only migrations from state.SchemaVersion to SchemaVersion.
// Returns ErrSchemaTooNew (refuse-to-mutate) when state was written by a
// newer CLI than the current binary knows about.
//
// CALLER CONTRACT: store.Load is responsible for writing the side-by-side
// backup of the ORIGINAL raw bytes BEFORE invoking Migrate (so the backup
// persists even if migration logic itself fails). See store.Load.
func Migrate(state State) (State, error) {
	if state.SchemaVersion == "" {
		state.SchemaVersion = SchemaVersion
	}
	if cmpVersion(state.SchemaVersion, SchemaVersion) > 0 {
		return State{}, ErrSchemaTooNew
	}
	if state.Repositories == nil {
		state.Repositories = []RepositoryState{}
	}
	for cmpVersion(state.SchemaVersion, SchemaVersion) < 0 {
		from := state.SchemaVersion
		fn, ok := forwardMigrations[from]
		if !ok {
			return State{}, fmt.Errorf("no migration from schema_version %q", from)
		}
		next, err := fn(state)
		if err != nil {
			return State{}, fmt.Errorf("migration from schema_version %q failed: %w", from, err)
		}
		if cmpVersion(next.SchemaVersion, from) <= 0 {
			return State{}, fmt.Errorf("migration from schema_version %q did not advance version (got %q)", from, next.SchemaVersion)
		}
		state = next
	}
	return state, nil
}

// migrateV1ToV2 is an identity migration: no field semantics changed in v2,
// but the framework + side-by-side backup is what REL-05 requires. Future
// v2->v3 migrations attach to forwardMigrations.
func migrateV1ToV2(s State) (State, error) {
	s.SchemaVersion = "2"
	return s, nil
}

// cmpVersion compares two SchemaVersion strings ("1", "2", ...) numerically.
// SchemaVersion is intentionally a small monotonic integer string; we don't
// want full semver here (state schema is not a public API surface).
// Returns -1 if a<b, 0 if equal, +1 if a>b. Empty string sorts as 0.
func cmpVersion(a, b string) int {
	ai := parseSchema(a)
	bi := parseSchema(b)
	switch {
	case ai < bi:
		return -1
	case ai > bi:
		return +1
	default:
		return 0
	}
}

func parseSchema(v string) int {
	if v == "" {
		return 0
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return -1
	}
	return n
}
