package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/remote"
)

type recordingExecutor struct{ commands []remote.Command }

func (r *recordingExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return remote.ProbeResult{}, nil
}
func (r *recordingExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	r.commands = append(r.commands, command)
	return remote.Result{ExitCode: 0}, nil
}

func TestApplyRunsBootstrapCommandsInOrderAndRedactsToken(t *testing.T) {
	token := "registration-token-secret-12345"
	exec := &recordingExecutor{}
	opts := Options{RunnerName: "runnerkit-owner-repo-local", RepoURL: "https://github.com/owner/repo", Labels: []string{"self-hosted", "runnerkit"}, ServiceUser: "runnerkit-runner", RunnerToken: token, Package: RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"}}
	if _, err := Apply(context.Background(), exec, remote.Target{User: "alice", Host: "example.com", Port: 22}, opts); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	var ids []string
	for _, command := range exec.commands {
		ids = append(ids, command.ID)
	}
	want := []string{"fix_dependencies", "create_runner_user", "download_runner", "configure_runner", "install_service", "verify_service"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("command IDs = %#v, want %#v", ids, want)
	}
	configure := exec.commands[3]
	if configure.ID != "configure_runner" {
		t.Fatalf("command before install_service = %s, want configure_runner", configure.ID)
	}
	if configure.Env["RUNNERKIT_REGISTRATION_TOKEN"] != token {
		t.Fatalf("configure token env missing: %#v", configure.Env)
	}
	if len(configure.RedactArgs) == 0 || configure.RedactArgs[0] != token {
		t.Fatalf("configure redaction args missing token: %#v", configure.RedactArgs)
	}
	if strings.Contains(configure.Script, token) {
		t.Fatalf("configure script leaked token:\n%s", configure.Script)
	}
}
