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
