package ui

import (
	"context"
	"strings"
	"testing"
)

// Bug 4 / Task G: assert that the production binary has a concrete
// Prompter implementation that satisfies both ui.Prompter and
// ui.PasswordPrompter. The interface-only state shipped in Plan 06-06
// silently leaves the binary's deps.Prompts == nil and surfaces a
// misleading "no TTY" error in real terminals.

func TestNewCLIPrompter_SatisfiesPrompter(t *testing.T) {
	t.Parallel()
	var p Prompter = NewCLIPrompter(strings.NewReader(""), &nopWriter{})
	if p == nil {
		t.Fatal("expected non-nil Prompter")
	}
}

func TestNewCLIPrompter_SatisfiesPasswordPrompter(t *testing.T) {
	t.Parallel()
	var p Prompter = NewCLIPrompter(strings.NewReader(""), &nopWriter{})
	if _, ok := p.(PasswordPrompter); !ok {
		t.Fatal("expected CLIPrompter to satisfy PasswordPrompter")
	}
}

func TestCLIPrompter_Confirm_EmptyInputReturnsDefault(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		def   bool
		want  bool
	}{
		{"\n", true, true},
		{"\n", false, false},
		{"", true, true},
	}
	for _, tc := range cases {
		p := NewCLIPrompter(strings.NewReader(tc.input), &nopWriter{})
		got, err := p.Confirm(context.Background(), Prompt{Message: "ok?", Default: tc.def})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != tc.want {
			t.Fatalf("input=%q default=%v got=%v want=%v", tc.input, tc.def, got, tc.want)
		}
	}
}

func TestCLIPrompter_Confirm_AcceptsAffirmative(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"y\n", "Y\n", "yes\n", "YES\n", "Yes\n"} {
		p := NewCLIPrompter(strings.NewReader(in), &nopWriter{})
		got, err := p.Confirm(context.Background(), Prompt{Message: "ok?", Default: false})
		if err != nil {
			t.Fatalf("input %q: unexpected err: %v", in, err)
		}
		if !got {
			t.Fatalf("input %q: want true got false", in)
		}
	}
}

func TestCLIPrompter_Confirm_RejectsNegative(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"n\n", "N\n", "no\n", "NO\n"} {
		p := NewCLIPrompter(strings.NewReader(in), &nopWriter{})
		got, err := p.Confirm(context.Background(), Prompt{Message: "ok?", Default: true})
		if err != nil {
			t.Fatalf("input %q: unexpected err: %v", in, err)
		}
		if got {
			t.Fatalf("input %q: want false got true", in)
		}
	}
}

func TestCLIPrompter_Select_ParsesNumericChoice(t *testing.T) {
	t.Parallel()
	options := []Option{
		{Value: "alpha", Label: "Alpha"},
		{Value: "beta", Label: "Beta"},
		{Value: "gamma", Label: "Gamma"},
	}
	p := NewCLIPrompter(strings.NewReader("2\n"), &nopWriter{})
	got, err := p.Select(context.Background(), Prompt{Message: "pick"}, options)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "beta" {
		t.Fatalf("want beta got %q", got)
	}
}

func TestCLIPrompter_Select_RejectsOutOfBounds(t *testing.T) {
	t.Parallel()
	options := []Option{{Value: "alpha"}, {Value: "beta"}}
	p := NewCLIPrompter(strings.NewReader("9\n"), &nopWriter{})
	if _, err := p.Select(context.Background(), Prompt{Message: "pick"}, options); err == nil {
		t.Fatal("expected error for out-of-bounds choice")
	}
}

func TestCLIPrompter_Password_RequiresTerminalWhenStdinNotTTY(t *testing.T) {
	t.Parallel()
	p := NewCLIPrompter(strings.NewReader("hunter2\n"), &nopWriter{})
	pp, ok := p.(PasswordPrompter)
	if !ok {
		t.Fatal("expected PasswordPrompter")
	}
	if _, err := pp.Password(context.Background(), Prompt{Message: "Sudo password:"}); err == nil {
		t.Fatal("expected error when stdin is not a terminal — refusing to read password from non-TTY would echo the password in plain text")
	}
}

type nopWriter struct{}

func (*nopWriter) Write(p []byte) (int, error) { return len(p), nil }
