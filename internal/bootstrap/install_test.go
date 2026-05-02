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

func TestApplyEphemeralRunsCommandsInOrderRedactsTokenAndAvoidsSvcSh(t *testing.T) {
	token := strings.Join([]string{"registration", "token", "ephemeral", "secret", "12345"}, "-")
	exec := &recordingExecutor{}
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-ephemeral-abc123",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"},
		ServiceUser: "runnerkit-runner",
		RunnerToken: token,
		Mode:        "ephemeral",
		Package:     RunnerPackage{Filename: "runner.tgz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	if _, err := ApplyEphemeral(context.Background(), exec, remote.Target{User: "alice", Host: "example.com", Port: 22}, opts); err != nil {
		t.Fatalf("ApplyEphemeral returned error: %v", err)
	}
	var ids []string
	for _, command := range exec.commands {
		ids = append(ids, command.ID)
	}
	want := []string{
		"fix_dependencies",
		"create_runner_user",
		"download_runner",
		"configure_ephemeral_runner",
		"install_ephemeral_finalizer",
		"install_ephemeral_service",
		"install_ephemeral_ttl_timer",
		"verify_ephemeral_service",
	}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("ephemeral command IDs = %#v, want %#v", ids, want)
	}
	configure := exec.commands[3]
	if configure.ID != "configure_ephemeral_runner" {
		t.Fatalf("configure step at index 3 = %s", configure.ID)
	}
	if configure.Env["RUNNERKIT_REGISTRATION_TOKEN"] != token {
		t.Fatalf("configure_ephemeral_runner env missing token: %#v", configure.Env)
	}
	if len(configure.RedactArgs) == 0 || configure.RedactArgs[0] != token {
		t.Fatalf("configure_ephemeral_runner redaction args missing token: %#v", configure.RedactArgs)
	}
	if strings.Contains(configure.Script, token) {
		t.Fatalf("configure_ephemeral_runner script leaked token:\n%s", configure.Script)
	}
	for _, command := range exec.commands {
		if strings.Contains(command.Script, "svc.sh install") || strings.Contains(command.Script, "svc.sh start") {
			t.Fatalf("ephemeral apply leaked svc.sh install/start loop in command %s:\n%s", command.ID, command.Script)
		}
	}
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
