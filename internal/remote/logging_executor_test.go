package remote

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

type stubInner struct {
	calls int
}

func (s *stubInner) Probe(context.Context, Target) (ProbeResult, error) {
	s.calls++
	return ProbeResult{}, nil
}

func (s *stubInner) Run(context.Context, Target, Command) (Result, error) {
	s.calls++
	return Result{ExitCode: 0}, nil
}

func TestWrapWithLogging_nilPassthrough(t *testing.T) {
	inner := &stubInner{}
	if WrapWithLogging(inner, nil, nil) != inner {
		t.Fatal("expected nil logger to skip wrap")
	}
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if WrapWithLogging(inner, log, nil) != inner {
		t.Fatal("expected disabled info to skip wrap")
	}
}

func TestLoggingExecutor_RunLogs(t *testing.T) {
	inner := &stubInner{}
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	wrapped := WrapWithLogging(inner, log, nil)
	le, ok := wrapped.(LoggingExecutor)
	if !ok {
		t.Fatalf("expected LoggingExecutor, got %T", wrapped)
	}
	w := le
	_, err := w.Run(context.Background(), Target{User: "u", Host: "h", Port: 22}, Command{ID: "test.cmd", Timeout: 1})
	if err != nil {
		t.Fatal(err)
	}
	if inner.calls != 1 {
		t.Fatalf("inner calls=%d", inner.calls)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"command_id":"test.cmd"`)) {
		t.Fatalf("missing log line: %s", buf.String())
	}
}

func TestTargetSummary(t *testing.T) {
	if g := targetSummary(Target{User: "a", Host: "b", Port: 2222}); g != "a@b:2222" {
		t.Fatalf("got %q", g)
	}
	if g := targetSummary(Target{Host: "b", Port: 0}); g != "b:22" {
		t.Fatalf("got %q", g)
	}
}
