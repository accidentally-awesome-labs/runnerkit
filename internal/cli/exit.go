package cli

import (
	"context"
	"errors"
	"fmt"
)

const (
	ExitSuccess       = 0
	ExitUnexpected    = 1
	ExitInvalidInput  = 2
	ExitGitHubAuth    = 3
	ExitSafetyGate    = 4
	ExitStateIO       = 5
	ExitInputRequired = 6
	ExitCanceled      = 130
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
