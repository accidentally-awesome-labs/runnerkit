package ui

import "context"

type Prompt struct {
	Message string
	Default bool
	Help    string
}

type Option struct {
	Value       string
	Label       string
	Description string
}

type Prompter interface {
	Confirm(ctx context.Context, prompt Prompt) (bool, error)
	Select(ctx context.Context, prompt Prompt, options []Option) (string, error)
}

// InputPrompter is an optional capability satisfied by Prompter
// implementations that can collect a free-form line of text (typed
// confirmation phrases such as "destroy owner/repo", or a BYO SSH
// target). Callers type-assert deps.Prompts to ui.InputPrompter
// rather than declaring anonymous interfaces, so the production
// CLIPrompter is checked against the same contract at compile time.
type InputPrompter interface {
	Input(ctx context.Context, prompt Prompt) (string, error)
}

// PasswordPrompter is an optional capability satisfied by Prompter
// implementations that can collect a sensitive value from the user
// (e.g. the host's sudo password for Plan 06-06 Path B fallback).
// Callers MUST type-assert to ui.PasswordPrompter so legacy
// Prompter implementations that don't support secret input remain
// compatible. Returned values MUST never be logged or echoed and
// SHOULD be registered with redact.SudoPassword by the caller before
// they propagate further.
type PasswordPrompter interface {
	Password(ctx context.Context, prompt Prompt) (string, error)
}
