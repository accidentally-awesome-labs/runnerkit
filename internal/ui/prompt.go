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

// PasswordPrompter is an optional capability satisfied by Prompter
// implementations that can collect a sensitive value from the user
// (e.g. the host's sudo password for Plan 06-06 Path B fallback).
// Callers MUST type-assert via interface{ Password(...) } so legacy
// Prompter implementations that don't support secret input remain
// compatible. Returned values MUST never be logged or echoed and
// SHOULD be registered with redact.SudoPassword by the caller before
// they propagate further.
type PasswordPrompter interface {
	Password(ctx context.Context, prompt Prompt) (string, error)
}
