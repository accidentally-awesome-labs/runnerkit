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

// TestRenderInstallScriptUsesSudoForCurlSha256SumTar asserts the
// renderer-side fix for Bug 2 of gap doc 06-GAP-byo-sudo-handling.md:
// curl, sha256sum -c -, and tar xzf must be prefixed with sudo so the
// install dir (owned by serviceUser) receives the tarball without
// `Permission denied` failures.
func TestRenderInstallScriptUsesSudoForCurlSha256SumTar(t *testing.T) {
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-local",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"},
		InstallPath: "/opt/actions-runner/runnerkit-owner-repo-local",
		WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-local",
		ServiceUser: "runnerkit-runner",
		RunnerToken: "registration-token-secret-12345",
		Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	script := RenderInstallScript(opts)
	for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf"} {
		if !strings.Contains(script, want) {
			t.Fatalf("RenderInstallScript missing %q:\n%s", want, script)
		}
	}
}

// TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar is the
// parallel assertion for the ephemeral renderer.
func TestRenderEphemeralInstallScriptUsesSudoForCurlSha256SumTar(t *testing.T) {
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-ephemeral-abc123",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"},
		InstallPath: "/opt/actions-runner/runnerkit-owner-repo-ephemeral-abc123",
		WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-ephemeral-abc123",
		ServiceUser: "runnerkit-runner",
		RunnerToken: "registration-token-ephemeral-secret-12345",
		Mode:        "ephemeral",
		Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	script := RenderEphemeralInstallScript(opts)
	for _, want := range []string{"sudo curl", "sudo sha256sum -c -", "sudo tar xzf"} {
		if !strings.Contains(script, want) {
			t.Fatalf("RenderEphemeralInstallScript missing %q:\n%s", want, script)
		}
	}
}

// TestRenderInstallScriptUsesSuForRegisterRunner asserts the
// renderer-side fix for Bug 3 of gap doc 06-GAP-byo-sudo-handling.md:
// register_runner must invoke config.sh via `sudo su -s /bin/bash -
// runnerkit-runner -c '...'` instead of `sudo -u runnerkit-runner
// ./config.sh ...`. The su form runs from a root sudo context so the
// host's sudoers needs only (root) NOPASSWD — no (ALL) runas required.
// See gap doc lines 122-199 (Bug 3 description) and lines 338-365
// (Task F) for the rationale.
func TestRenderInstallScriptUsesSuForRegisterRunner(t *testing.T) {
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-local",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"},
		InstallPath: "/opt/actions-runner/runnerkit-owner-repo-local",
		WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-local",
		ServiceUser: "runnerkit-runner",
		RunnerToken: "registration-token-secret-bug3-12345",
		Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	script := RenderInstallScript(opts)
	// PRESENCE assertion: the new su form must be in the rendered script.
	if !strings.Contains(script, "sudo su -s /bin/bash - runnerkit-runner -c") {
		t.Fatalf("RenderInstallScript missing sudo su -s /bin/bash - runnerkit-runner -c form for register_runner:\n%s", script)
	}
	// NEGATIVE assertion: the buggy `sudo -u runnerkit-runner ./config.sh` form must be GONE.
	for _, forbidden := range []string{
		"sudo -u runnerkit-runner ./config.sh",
		"sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("RenderInstallScript still contains buggy %q (Bug 3 not closed):\n%s", forbidden, script)
		}
	}
	// Token-leak invariant: $RUNNERKIT_REGISTRATION_TOKEN must remain the env-var reference; the literal token must NOT appear.
	if strings.Contains(script, opts.RunnerToken) {
		t.Fatalf("RenderInstallScript leaked registration token literal (Bug 3 fix must preserve redaction invariant):\n%s", script)
	}
}

// TestRenderEphemeralInstallScriptUsesSuForRegisterRunner is the
// parallel assertion for the ephemeral renderer.
func TestRenderEphemeralInstallScriptUsesSuForRegisterRunner(t *testing.T) {
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-ephemeral-bug3test",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"},
		InstallPath: "/opt/actions-runner/runnerkit-owner-repo-ephemeral-bug3test",
		WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-ephemeral-bug3test",
		ServiceUser: "runnerkit-runner",
		RunnerToken: "registration-token-ephemeral-bug3-secret",
		Mode:        "ephemeral",
		Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	script := RenderEphemeralInstallScript(opts)
	if !strings.Contains(script, "sudo su -s /bin/bash - runnerkit-runner -c") {
		t.Fatalf("RenderEphemeralInstallScript missing sudo su -s /bin/bash - runnerkit-runner -c form for register_runner:\n%s", script)
	}
	for _, forbidden := range []string{
		"sudo -u runnerkit-runner ./config.sh",
		"sudo -u runnerkit-runner RUNNERKIT_REGISTRATION_TOKEN",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("RenderEphemeralInstallScript still contains buggy %q (Bug 3 not closed):\n%s", forbidden, script)
		}
	}
	// Ephemeral-specific: --ephemeral flag must remain at the end of the wrapped command.
	if !strings.Contains(script, "--replace --ephemeral") {
		t.Fatalf("RenderEphemeralInstallScript missing --replace --ephemeral flag tail in wrapped command:\n%s", script)
	}
	if strings.Contains(script, opts.RunnerToken) {
		t.Fatalf("RenderEphemeralInstallScript leaked registration token literal:\n%s", script)
	}
}

func TestRenderDependencyFixScript(t *testing.T) {
	script := RenderDependencyFixScript([]string{"curl", "tar"})
	if !strings.Contains(script, "sudo apt-get install -y curl tar") || !strings.Contains(script, "sudo dnf install -y curl tar") {
		t.Fatalf("dependency fix script missing package manager paths:\n%s", script)
	}
}

func TestRenderEphemeralInstallScriptUsesEphemeralFlagAndRedactsToken(t *testing.T) {
	fakeToken := strings.Join([]string{"registration", "token", "ephemeral", "secret"}, "-")
	opts := Options{
		RunnerName:  "runnerkit-owner-repo-ephemeral-abc123",
		RepoURL:     "https://github.com/owner/repo",
		Labels:      []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"},
		InstallPath: "/opt/actions-runner/runnerkit-owner-repo-ephemeral-abc123",
		WorkDir:     "/var/lib/runnerkit/work/runnerkit-owner-repo-ephemeral-abc123",
		ServiceUser: "runnerkit-runner",
		RunnerToken: fakeToken,
		Mode:        "ephemeral",
		Package:     RunnerPackage{Filename: "actions-runner-linux-x64-2.334.0.tar.gz", URL: "https://example.invalid/runner.tgz", SHA256: "abc123"},
	}
	script := RenderEphemeralInstallScript(opts)
	for _, want := range []string{
		"set -euo pipefail",
		"--replace --ephemeral",
		"./config.sh --unattended --url https://github.com/owner/repo --token \"$RUNNERKIT_REGISTRATION_TOKEN\"",
		"--name runnerkit-owner-repo-ephemeral-abc123",
		"--labels self-hosted,runnerkit,runnerkit-owner-repo,linux,x64,ephemeral",
		"--work /var/lib/runnerkit/work/runnerkit-owner-repo-ephemeral-abc123",
		"RUNNERKIT_REGISTRATION_TOKEN",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("ephemeral install script missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, fakeToken) {
		t.Fatalf("ephemeral install script leaked registration token:\n%s", script)
	}
}

func TestRenderEphemeralServiceScriptUsesOneShotUnitWithoutSvcSh(t *testing.T) {
	opts := Options{
		RunnerName:           "runnerkit-owner-repo-ephemeral-abc123",
		InstallPath:          "/opt/actions-runner/runnerkit-owner-repo-ephemeral-abc123",
		ServiceUser:          "runnerkit-runner",
		Mode:                 "ephemeral",
		EphemeralServiceName: "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
		FinalizerPath:        "/usr/local/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/finalize.sh",
	}
	script := RenderEphemeralServiceScript(opts)
	for _, want := range []string{
		"Restart=no",
		"ExecStart=/opt/actions-runner/runnerkit-owner-repo-ephemeral-abc123/run.sh",
		"ExecStopPost=/usr/local/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/finalize.sh completed",
		"systemctl daemon-reload",
		"systemctl start runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
		"/etc/systemd/system/runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("ephemeral service script missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "svc.sh install") || strings.Contains(script, "svc.sh start") {
		t.Fatalf("ephemeral service script must not use svc.sh install/start loop:\n%s", script)
	}
}

func TestRenderEphemeralFinalizerScriptPreservesLogsAndRemovesCredentials(t *testing.T) {
	opts := Options{
		RunnerName:     "runnerkit-owner-repo-ephemeral-abc123",
		InstallPath:    "/opt/actions-runner/runnerkit-owner-repo-ephemeral-abc123",
		ServiceUser:    "runnerkit-runner",
		Mode:           "ephemeral",
		LogArchivePath: "/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/logs",
		FinalizerPath:  "/usr/local/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/finalize.sh",
		EphemeralServiceName: "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
	}
	script := RenderEphemeralFinalizerScript(opts)
	for _, want := range []string{
		"Runner_*.log",
		"Worker_*.log",
		"journalctl -u",
		"systemd-journal.log",
		"state.json",
		"finalizer_status",
		"rm -f .runner .credentials .credentials_rsaparams",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("ephemeral finalizer script missing %q:\n%s", want, script)
		}
	}
	for _, forbidden := range []string{"RUNNERKIT_REGISTRATION_TOKEN", "RUNNERKIT_REMOVAL_TOKEN"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("ephemeral finalizer script must not interpolate %q:\n%s", forbidden, script)
		}
	}
}

func TestRenderEphemeralTTLTimerScriptUses24hAndStopsService(t *testing.T) {
	opts := Options{
		RunnerName:              "runnerkit-owner-repo-ephemeral-abc123",
		Mode:                    "ephemeral",
		FinalizerPath:           "/usr/local/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/finalize.sh",
		EphemeralServiceName:    "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
		EphemeralTTLServiceName: "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.ttl.service",
		EphemeralTTLTimerName:   "runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.ttl.timer",
	}
	script := RenderEphemeralTTLTimerScript(opts)
	for _, want := range []string{
		"OnActiveSec=24h",
		"TTL safeguard",
		"systemctl stop runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
		"/usr/local/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/finalize.sh ttl_expired",
		"systemctl enable --now runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.ttl.timer",
		"runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.ttl.service",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("ephemeral TTL timer script missing %q:\n%s", want, script)
		}
	}
}

func TestRenderEphemeralLogPreservationScriptCopiesDiagAndJournal(t *testing.T) {
	script := RenderEphemeralLogPreservationScript(
		"/opt/actions-runner/runnerkit-owner-repo-ephemeral-abc123",
		"/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/logs",
		"runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service",
	)
	for _, want := range []string{
		"Runner_*.log",
		"Worker_*.log",
		"journalctl -u runnerkit-ephemeral.runnerkit-owner-repo-ephemeral-abc123.service -n 500 --no-pager",
		"/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-ephemeral-abc123/logs",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("ephemeral log preservation script missing %q:\n%s", want, script)
		}
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
