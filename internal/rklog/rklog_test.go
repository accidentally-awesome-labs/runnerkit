package rklog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFromEnv_Discard(t *testing.T) {
	t.Setenv(envLogLevel, "")
	var buf bytes.Buffer
	log := NewFromEnv(&buf)
	if log.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("expected discard logger")
	}
}

func TestNewFromEnv_InfoJSON(t *testing.T) {
	t.Setenv(envLogLevel, "info")
	var buf bytes.Buffer
	log := NewFromEnv(&buf)
	log.Info("hello", "k", "v")
	line := strings.TrimSpace(buf.String())
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("json: %v line=%q", err, line)
	}
	if obj["msg"] != "hello" || obj["k"] != "v" {
		t.Fatalf("unexpected record: %#v", obj)
	}
}

func TestParseLevel_Table(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"off", "discard"},
		{"0", "discard"},
		{"DEBUG", "debug"},
		{"Info", "info"},
		{"bogus", "info"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got := parseLevel(tc.in)
			var want slog.Level
			switch tc.want {
			case "discard":
				want = slog.Level(1000)
			case "debug":
				want = slog.LevelDebug
			case "info":
				want = slog.LevelInfo
			}
			if got != want {
				t.Fatalf("parseLevel(%q)=%v want %v", tc.in, got, want)
			}
		})
	}
}

func TestResolveDestination_Table(t *testing.T) {
	var fallback bytes.Buffer
	out, err := resolveDestination(&fallback, "")
	if err != nil {
		t.Fatalf("resolve empty: %v", err)
	}
	if out != &fallback {
		t.Fatalf("empty destination should use fallback writer")
	}

	out, err = resolveDestination(&fallback, "stdout")
	if err != nil {
		t.Fatalf("resolve stdout: %v", err)
	}
	if out != os.Stdout {
		t.Fatalf("stdout destination should return os.Stdout")
	}
}

func TestResolveDestination_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runnerkit", "logs.jsonl")
	out, err := resolveDestination(nil, "file:"+path)
	if err != nil {
		t.Fatalf("resolve file destination: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("from-file-dest", "k", "v")
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(fileBytes), "from-file-dest") {
		t.Fatalf("expected log line in file, got: %s", string(fileBytes))
	}
}
