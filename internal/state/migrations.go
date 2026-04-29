package state

import "fmt"

func Migrate(state State) (State, error) {
	if state.SchemaVersion == "" {
		state.SchemaVersion = SchemaVersion
	}
	if state.SchemaVersion != SchemaVersion {
		return State{}, fmt.Errorf("unsupported runnerkit state schema_version %q", state.SchemaVersion)
	}
	if state.Repositories == nil {
		state.Repositories = []RepositoryState{}
	}
	return state, nil
}
