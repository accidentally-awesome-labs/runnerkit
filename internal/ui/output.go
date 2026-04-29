package ui

// TerminalCapabilities describes the terminal features available to the CLI.
type TerminalCapabilities struct {
	StdinTTY  bool
	StdoutTTY bool
	Color     bool
	ASCII     bool
	Width     int
}

// Prompter is defined fully in the prompt/output foundation plan; it is kept
// here so command dependencies can be wired from the first CLI skeleton.
type Prompter interface{}
