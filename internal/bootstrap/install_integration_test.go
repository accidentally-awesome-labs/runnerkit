//go:build integration

// Package bootstrap real-shell integration tests live under the
// `integration` build tag. They are NEVER executed by `go test ./...`
// and only run when explicitly invoked via `make test-integration` or
// `go test -tags=integration ./internal/bootstrap/...`. They additionally
// require RUNNERKIT_INTEGRATION=1 because sudo-prefixed commands need
// passwordless sudo (or a sudo shim) on the test machine; see plan
// 06-05 Task 2 Step 2.5 for the rationale and CI-safety contract.
package bootstrap

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// shellExecutor invokes commands via a local bash shell. It is gated to
// the integration build tag and only used to prove that real-shell
// semantics match the renderer expectations. Sudo-prefixed commands
// require NOPASSWD sudo on the test machine (or a sudo shim) — the
// test harness skips when RUNNERKIT_INTEGRATION is unset.
type shellExecutor struct{ workingDir string }

func (s *shellExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return remote.ProbeResult{}, nil
}

func (s *shellExecutor) Run(_ context.Context, _ remote.Target, c remote.Command) (remote.Result, error) {
	cmd := exec.Command("bash", "-c", c.Script)
	cmd.Dir = s.workingDir
	out, err := cmd.CombinedOutput()
	result := remote.Result{Stdout: string(out)}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		result.Stderr = string(out)
		return result, nil
	} else if err != nil {
		result.ExitCode = -1
		result.Stderr = err.Error()
		return result, err
	}
	return result, nil
}

// TestApply_DownloadRunner_RealShell exercises the actual download_runner
// shell sequence against a real bash shell with a tmpfs sandbox, proving
// that the sudo-prefixed curl/sha256sum/tar lands the tarball + extracts
// config.sh into the install dir owned by the configured service user.
// This closes the fakeExecutor-only test gap that hid Bug 2 from Plan
// 02-02 through the 06-04 live BYO smoke (gap doc Task E).
func TestApply_DownloadRunner_RealShell(t *testing.T) {
	if os.Getenv("RUNNERKIT_INTEGRATION") == "" {
		t.Skip("set RUNNERKIT_INTEGRATION=1 to run; requires NOPASSWD sudo on the test machine")
	}
	tmp := t.TempDir()
	tarballPath := filepath.Join(tmp, "fake-runner.tgz")
	sha := buildFakeRunnerTarball(t, tarballPath)

	// Serve the fake tarball over httptest so the curl line in the
	// rendered script can reach it via http://127.0.0.1:<port>/...
	server := httptest.NewServer(http.FileServer(http.Dir(tmp)))
	defer server.Close()

	installPath := filepath.Join(tmp, "install")
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "runnerkit-runner"
	}
	opts := Options{
		RunnerName:  "runnerkit-it-test",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted"},
		InstallPath: installPath,
		WorkDir:     filepath.Join(tmp, "work"),
		ServiceUser: currentUser,
		RunnerToken: "registration-token-itest",
		Package:     RunnerPackage{Filename: "fake-runner.tgz", URL: server.URL + "/fake-runner.tgz", SHA256: sha},
	}

	// Drive the same Command literal Apply emits so the integration
	// test exercises the real-shell semantics that fakeExecutor-only
	// unit tests hid (gap doc Task E rationale).
	dl := downloadRunnerCommand(opts)
	executor := &shellExecutor{workingDir: tmp}
	result, err := executor.Run(context.Background(), remote.Target{}, dl)
	if err != nil {
		t.Fatalf("download_runner shell exec returned error: %v\noutput:\n%s", err, result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Fatalf("download_runner shell exec exit=%d, stderr:\n%s", result.ExitCode, result.Stderr)
	}

	if _, err := os.Stat(filepath.Join(installPath, "fake-runner.tgz")); err != nil {
		t.Fatalf("tarball not found at %s: %v", installPath, err)
	}
	if _, err := os.Stat(filepath.Join(installPath, "config.sh")); err != nil {
		t.Fatalf("extracted config.sh not found at %s: %v", installPath, err)
	}
}

// buildFakeRunnerTarball creates a minimal .tar.gz containing a single
// config.sh file at the archive root. Returns the SHA-256 hex digest
// the rendered sha256sum -c - line will assert against.
func buildFakeRunnerTarball(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tarball: %v", err)
	}
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	body := []byte("#!/bin/bash\necho fake config.sh\n")
	hdr := &tar.Header{Name: "config.sh", Mode: 0755, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
	// Compute sha256 of the finished file.
	f2, err := os.Open(path)
	if err != nil {
		t.Fatalf("reopen tarball: %v", err)
	}
	defer f2.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f2); err != nil {
		t.Fatalf("hash: %v", err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
