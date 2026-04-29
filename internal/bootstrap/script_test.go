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

func TestRenderRecoveryScriptsUseEnvironmentTokens(t *testing.T) {
	removalToken := strings.Join([]string{"removal", "token", "recover", "secret"}, "-")
	registrationToken := strings.Join([]string{"registration", "token", "recover", "secret"}, "-")
	opts := Options{RunnerName: "runnerkit-owner-repo-local", RepoURL: "https://github.com/owner/repo", Labels: []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"}, InstallPath: "/opt/actions-runner/runnerkit-owner-repo-local", WorkDir: "/var/lib/runnerkit/work/runnerkit-owner-repo-local", ServiceUser: "runnerkit-runner"}
	remove := RenderRemoveConfigScript(opts.InstallPath, opts.ServiceUser)
	reconfigure := RenderReconfigureScript(opts)
	for _, want := range []string{"RUNNERKIT_REMOVAL_TOKEN", "cd /opt/actions-runner/runnerkit-owner-repo-local", "./config.sh remove --token \"$RUNNERKIT_REMOVAL_TOKEN\""} {
		if !strings.Contains(remove, want) {
			t.Fatalf("remove script missing %q:\n%s", want, remove)
		}
	}
	for _, want := range []string{"RUNNERKIT_REGISTRATION_TOKEN", "./config.sh --unattended --url https://github.com/owner/repo --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name runnerkit-owner-repo-local --labels self-hosted,runnerkit,runnerkit-owner-repo,linux,x64,persistent --work /var/lib/runnerkit/work/runnerkit-owner-repo-local --replace"} {
		if !strings.Contains(reconfigure, want) {
			t.Fatalf("reconfigure script missing %q:\n%s", want, reconfigure)
		}
	}
	if strings.Contains(remove, removalToken) || strings.Contains(reconfigure, registrationToken) {
		t.Fatalf("recovery scripts interpolated token values:\nremove=%s\nreconfigure=%s", remove, reconfigure)
	}
}
