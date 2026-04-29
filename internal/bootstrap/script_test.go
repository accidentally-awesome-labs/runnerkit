package bootstrap

import (
	"strings"
	"testing"
)

func TestRenderInstallAndServiceScripts(t *testing.T) {
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-local",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"},
		InstallPath: "/opt/actions-runner/runnerkit-owner-repo-local",
		WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-local",
		ServiceUser: "runnerkit-runner",
		RunnerToken: strings.Join([]string{"registration-token", "secret-12345"}, "-"),
		Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	install := RenderInstallScript(opts)
	for _, want := range []string{"set -euo pipefail", "runnerkit-runner", "/opt/actions-runner/runnerkit-owner-repo-local", "/var/lib/runnerkit", "sha256sum -c -", "RUNNERKIT_REGISTRATION_TOKEN", "./config.sh --unattended --url https://github.com/owner/repo --token \"$RUNNERKIT_REGISTRATION_TOKEN\""} {
		if !strings.Contains(install, want) {
			t.Fatalf("install script missing %q:\n%s", want, install)
		}
	}
	if strings.Contains(install, opts.RunnerToken) || strings.Contains(install, "set -x") {
		t.Fatalf("install script leaked token or enabled tracing:\n%s", install)
	}
	service := RenderServiceScript(opts)
	if !strings.Contains(service, "sudo ./svc.sh install runnerkit-runner") || strings.Contains(service, "sudo ./svc.sh install root") {
		t.Fatalf("service script has wrong service user:\n%s", service)
	}
}

func TestRenderDependencyFixScript(t *testing.T) {
	script := RenderDependencyFixScript([]string{"curl", "tar"})
	if !strings.Contains(script, "sudo apt-get install -y curl tar") || !strings.Contains(script, "sudo dnf install -y curl tar") {
		t.Fatalf("dependency fix script missing package manager paths:\n%s", script)
	}
}
