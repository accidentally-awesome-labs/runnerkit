// Package rklog configures RunnerKit's optional structured logger (slog)
// for debugging and future telemetry. Logs go to stderr in JSON by default
// when enabled via RUNNERKIT_LOG.
package rklog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	envLogLevel = "RUNNERKIT_LOG"
	envLogDest  = "RUNNERKIT_LOG_DEST"
)

// NewFromEnv returns a slog.Logger for RunnerKit. When RUNNERKIT_LOG is
// unset, empty, or "off"/"0"/"false", logs are discarded. Otherwise the
// value sets the minimum level: debug, info, warn, error (case-insensitive).
// Unknown non-empty values default to info. When w is nil, os.Stderr is used.
func NewFromEnv(w io.Writer) *slog.Logger {
	level := parseLevel(os.Getenv(envLogLevel))
	if level >= slog.Level(1000) {
		return slog.New(discardHandler{})
	}
	out, err := resolveDestination(w, os.Getenv(envLogDest))
	if err != nil {
		// Fallback to stderr with a minimal line; keep logging enabled.
		fmt.Fprintf(os.Stderr, "runnerkit: invalid %s, falling back to stderr: %v\n", envLogDest, err)
		out = os.Stderr
	}
	opts := &slog.HandlerOptions{Level: level}
	if level <= slog.LevelDebug {
		opts.AddSource = true
	}
	return slog.New(slog.NewJSONHandler(out, opts))
}

func resolveDestination(fallback io.Writer, raw string) (io.Writer, error) {
	dest := strings.TrimSpace(raw)
	if fallback == nil {
		fallback = os.Stderr
	}
	switch strings.ToLower(dest) {
	case "", "stderr":
		return fallback, nil
	case "stdout":
		return os.Stdout, nil
	}
	if strings.HasPrefix(strings.ToLower(dest), "file:") {
		dest = strings.TrimSpace(dest[len("file:"):])
	}
	if dest == "" {
		return nil, fmt.Errorf("empty file destination")
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(dest, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func parseLevel(raw string) slog.Level {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "", "0", "off", "false", "no":
		return slog.Level(1000) // sentinel: discard
	case "debug":
		return slog.LevelDebug
	case "info", "1", "true", "yes", "on":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }
