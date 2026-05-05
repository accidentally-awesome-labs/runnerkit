package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// CLIPrompter is the production-binary Prompter used by cmd/runnerkit
// for interactive Confirm/Select/Password input. It satisfies both
// ui.Prompter (Confirm, Select) and ui.PasswordPrompter (Password).
//
// Password collection uses golang.org/x/term.ReadPassword on the
// underlying *os.File's fd to disable terminal echo. If the input
// stream is NOT an *os.File backed by a real TTY, Password fails
// closed with an explicit error rather than reading the password in
// plain text — protects against accidental echo when stdin is a pipe
// or a redirected file.
type CLIPrompter struct {
	in     io.Reader
	out    io.Writer
	reader *bufio.Reader
}

// NewCLIPrompter returns a CLIPrompter that reads from in and writes
// prompts to out. Pass os.Stdin / os.Stdout in production. Tests may
// pass strings.NewReader for Confirm/Select cases; Password
// deliberately rejects non-*os.File readers.
func NewCLIPrompter(in io.Reader, out io.Writer) *CLIPrompter {
	return &CLIPrompter{in: in, out: out, reader: bufio.NewReader(in)}
}

// Confirm renders prompt.Message followed by [Y/n] (default true) or
// [y/N] (default false), reads one line, and parses y/yes/n/no
// case-insensitively. Empty input returns prompt.Default. Ambiguous
// input also returns prompt.Default to keep accidental keystrokes
// from triggering destructive defaults.
func (p *CLIPrompter) Confirm(_ context.Context, prompt Prompt) (bool, error) {
	suffix := " [y/N]: "
	if prompt.Default {
		suffix = " [Y/n]: "
	}
	fmt.Fprint(p.out, prompt.Message+suffix)
	line, err := p.readLine()
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return prompt.Default, nil
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return prompt.Default, nil
	}
}

// Select renders prompt.Message followed by a numbered list of
// options, reads one line, parses it as an int, and returns
// options[N-1].Value when N is in range. Out-of-range or
// non-numeric input returns an error.
func (p *CLIPrompter) Select(_ context.Context, prompt Prompt, options []Option) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("ui: Select requires at least one option")
	}
	fmt.Fprintln(p.out, prompt.Message)
	for i, opt := range options {
		label := opt.Label
		if label == "" {
			label = opt.Value
		}
		if opt.Description != "" {
			fmt.Fprintf(p.out, "  %d) %s — %s\n", i+1, label, opt.Description)
		} else {
			fmt.Fprintf(p.out, "  %d) %s\n", i+1, label)
		}
	}
	fmt.Fprintf(p.out, "Choice [1-%d]: ", len(options))
	line, err := p.readLine()
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(line)
	n, err := strconv.Atoi(s)
	if err != nil {
		return "", fmt.Errorf("ui: invalid selection %q (expected 1-%d)", s, len(options))
	}
	if n < 1 || n > len(options) {
		return "", fmt.Errorf("ui: selection %d out of range (expected 1-%d)", n, len(options))
	}
	return options[n-1].Value, nil
}

// Password reads a sensitive value from the underlying *os.File with
// terminal echo disabled. Refuses to read if stdin is not a terminal
// — without echo suppression a password would render in plain text
// in the user's scrollback or in any tee/log capture.
func (p *CLIPrompter) Password(_ context.Context, prompt Prompt) (string, error) {
	f, ok := p.in.(*os.File)
	if !ok {
		return "", fmt.Errorf("ui: refusing to read password — stdin is not a *os.File")
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("ui: refusing to read password — stdin fd is not a terminal")
	}
	fmt.Fprint(p.out, prompt.Message+" ")
	bytes, err := term.ReadPassword(fd)
	fmt.Fprintln(p.out)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (p *CLIPrompter) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
