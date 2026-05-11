// Package nextaction defines the versioned JSON contract for CLI --json output.
// Schema is additive-only; bump SchemaVersion only for incompatible changes.
package nextaction

const SchemaVersion = 1

// Severity indicates how strongly an action blocks progress.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBlocking Severity = "blocking"
)

// Action is one suggested next step for humans or agents.
type Action struct {
	ID       string   `json:"id"`
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	Command  string   `json:"command,omitempty"`
	Kind     string   `json:"kind,omitempty"` // run_on_host: copy-paste on runner host; run_local: maintainer machine
}

// MergePayload adds schema_version, optional stage, and next_actions to a command JSON map.
func MergePayload(base map[string]any, stage string, actions []Action) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	base["schema_version"] = SchemaVersion
	if stage != "" {
		base["stage"] = stage
	}
	list := make([]map[string]any, len(actions))
	for i := range actions {
		list[i] = actionToMap(actions[i])
	}
	base["next_actions"] = list
	return base
}

func actionToMap(a Action) map[string]any {
	m := map[string]any{
		"id":       a.ID,
		"severity": a.Severity,
		"title":    a.Title,
	}
	if a.Command != "" {
		m["command"] = a.Command
	}
	if a.Kind != "" {
		m["kind"] = a.Kind
	}
	return m
}

// InstallHostActions returns next_actions when the BYO host needs one-time install.sh.
func InstallHostActions(installOneLiner string) []Action {
	return []Action{
		{
			ID:       "install_runnerkit_host_prereqs",
			Severity: SeverityBlocking,
			Title:    "Run the one-time host install on the runner machine (interactive sudo once)",
			Command:  installOneLiner,
			Kind:     "run_on_host",
		},
	}
}
