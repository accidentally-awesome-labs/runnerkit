package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
)

const (
	ExitSuccess           = 0
	ExitUnexpected        = 1
	ExitInvalidInput      = 2
	ExitGitHubAuth        = 3
	ExitSafetyGate        = 4
	ExitStateIO           = 5
	ExitInputRequired     = 6
	ExitStateSchemaTooNew = 7
	ExitCanceled          = 130
)

// ExitError carries a typed process exit code through Cobra command execution.
type ExitError struct {
	Code int
	Err  error
}

func NewExitError(code int, err error) *ExitError {
	if err == nil {
		err = fmt.Errorf("exit with code %d", code)
	}
	return &ExitError{Code: code, Err: err}
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ExitCode maps command errors to the process exit code contract.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	if errors.Is(err, context.Canceled) {
		return ExitCanceled
	}
	return ExitUnexpected
}

// ReportUnrenderedError is called by main after Execute fails. When no
// Renderer has already shown the failure (alreadyRendered == false), it
// writes the error so that no non-zero exit is silent (P1-1): a
// `runnerkit: <err>` line on stderr, and in --json mode additionally an
// {"ok":false,"error":{...}} envelope on stdout for machine consumers.
// Cobra's "Did you mean this?" suggestions are part of the error string and
// survive verbatim.
func ReportUnrenderedError(stdout, stderr io.Writer, args []string, err error, alreadyRendered bool) {
	if err == nil || alreadyRendered {
		return
	}
	exitCode := ExitCode(err)
	msg := strings.TrimSpace(redact.New().String(err.Error()))
	if msg == "" {
		msg = fmt.Sprintf("command failed with exit code %d", exitCode)
	}
	if stderr != nil {
		_, _ = fmt.Fprintf(stderr, "runnerkit: %s\n", msg)
		if exitCode == ExitInvalidInput {
			_, _ = fmt.Fprintln(stderr, usageHint)
		}
	}
	if stdout == nil || !argsRequestJSON(args) {
		return
	}
	code := "command_failed"
	remediation := []string{}
	switch exitCode {
	case ExitInvalidInput:
		code = "cli_usage"
		remediation = []string{usageHint}
	case ExitCanceled:
		code = "canceled"
	}
	payload := map[string]any{
		"ok":        false,
		"exit_code": exitCode,
		"error": map[string]any{
			"code":        code,
			"message":     msg,
			"remediation": remediation,
		},
	}
	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return
	}
	_, _ = stdout.Write(append(encoded, '\n'))
}

const usageHint = "Run 'runnerkit --help' for usage."

// argsRequestJSON reports whether --json (or --json=<true>) appears before a
// "--" terminator. It deliberately does not rely on Cobra having parsed the
// flags, because the failures it serves are often parse failures.
func argsRequestJSON(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--json" {
			return true
		}
		if value, ok := strings.CutPrefix(arg, "--json="); ok {
			parsed, err := strconv.ParseBool(value)
			return err == nil && parsed
		}
	}
	return false
}
