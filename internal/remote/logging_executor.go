package remote

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
)

// LoggingExecutor wraps an Executor and emits structured slog events for
// each remote Probe, ProbeHostKey, and Run (command id, target, duration,
// exit status). Stdout/stderr bodies are only logged at debug level and are
// passed through redactor with truncation to limit accidental secret leakage.
type LoggingExecutor struct {
	Inner    Executor
	Logger   *slog.Logger
	Redactor *redact.Redactor
}

// WrapWithLogging returns e unchanged when log is nil or disabled at info.
func WrapWithLogging(e Executor, log *slog.Logger, red *redact.Redactor) Executor {
	if e == nil || log == nil {
		return e
	}
	if !log.Enabled(context.Background(), slog.LevelInfo) {
		return e
	}
	if red == nil {
		red = redact.New()
	}
	return LoggingExecutor{Inner: e, Logger: log, Redactor: red}
}

func (w LoggingExecutor) Probe(ctx context.Context, target Target) (ProbeResult, error) {
	start := time.Now()
	res, err := w.Inner.Probe(ctx, target)
	if w.Logger.Enabled(ctx, slog.LevelInfo) {
		w.Logger.InfoContext(ctx, "remote.probe",
			slog.String("target", targetSummary(target)),
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", err == nil),
		)
	}
	if err != nil && w.Logger.Enabled(ctx, slog.LevelWarn) {
		w.Logger.WarnContext(ctx, "remote.probe.error", slog.String("err", err.Error()))
	}
	return res, err
}

func (w LoggingExecutor) ProbeHostKey(ctx context.Context, target Target) (HostKey, error) {
	inner, ok := w.Inner.(HostKeyProber)
	if !ok {
		return HostKey{}, nil
	}
	start := time.Now()
	hk, err := inner.ProbeHostKey(ctx, target)
	if w.Logger.Enabled(ctx, slog.LevelInfo) {
		w.Logger.InfoContext(ctx, "remote.probe_host_key",
			slog.String("target", targetSummary(target)),
			slog.Duration("duration", time.Since(start)),
			slog.Bool("ok", err == nil),
			slog.String("fingerprint", hk.Fingerprint),
		)
	}
	return hk, err
}

func (w LoggingExecutor) Run(ctx context.Context, target Target, command Command) (Result, error) {
	start := time.Now()
	res, err := w.Inner.Run(ctx, target, command)
	dur := time.Since(start)
	if w.Logger.Enabled(ctx, slog.LevelInfo) {
		attrs := []slog.Attr{
			slog.String("command_id", command.ID),
			slog.String("target", targetSummary(target)),
			slog.Duration("duration", dur),
			slog.Int("exit_code", res.ExitCode),
			slog.Bool("exec_err", err != nil),
			slog.Bool("sudo", command.Sudo),
		}
		if err != nil {
			attrs = append(attrs, slog.String("err", err.Error()))
		}
		w.Logger.LogAttrs(ctx, slog.LevelInfo, "remote.run", attrs...)
	}
	if w.Logger.Enabled(ctx, slog.LevelDebug) {
		const max = 2048
		out := truncateForLog(w.Redactor.String(res.Stdout), max)
		errOut := truncateForLog(w.Redactor.String(res.Stderr), max)
		if strings.TrimSpace(out+errOut) != "" {
			w.Logger.DebugContext(ctx, "remote.run.io",
				slog.String("command_id", command.ID),
				slog.String("stdout", out),
				slog.String("stderr", errOut),
			)
		}
	}
	return res, err
}

func targetSummary(t Target) string {
	if strings.TrimSpace(t.Host) == "" {
		return ""
	}
	port := t.Port
	if port == 0 {
		port = 22
	}
	u := strings.TrimSpace(t.User)
	if u == "" {
		return t.Host + ":" + strconv.Itoa(port)
	}
	return u + "@" + t.Host + ":" + strconv.Itoa(port)
}

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…(truncated)"
}
