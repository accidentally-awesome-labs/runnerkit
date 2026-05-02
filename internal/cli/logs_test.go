package cli

import (
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/ops"
	"github.com/salar/runnerkit/internal/remote"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
)

func logsRemoteWithSecrets() *testsupport.RemoteExecutor {
	secretLog := strings.Join([]string{
		"registration-token-secret-logs removal-token-secret-logs github_pat_secretlogs HCLOUD_TOKEN=supersecret alice@example.com:22",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"key-material",
		"-----END OPENSSH PRIVATE KEY-----",
	}, "\n")
	return &testsupport.RemoteExecutor{Results: map[string]remote.Result{
		ops.CommandLogsSystemdJournal: {Stdout: secretLog, ExitCode: 0},
		ops.CommandLogsRunnerDiagList: {Stdout: "/opt/actions-runner/runnerkit-owner-repo-local/_diag/Runner_1.log\n", ExitCode: 0},
		ops.CommandLogsRunnerDiagTail: {Stdout: secretLog, ExitCode: 0},
	}}
}

func TestLogsDefaultsAndRedactsHumanOutput(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	out, errOut, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{}, logsRemoteWithSecrets(), "logs", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("logs returned error: %v\nstderr=%s", err, errOut)
	}
	for _, want := range []string{"Step 1 of 1: runner logs", "Since: 1h", "Lines: 200", "collection summary", "systemd journal", "runner diag", "Review logs before sharing", "redaction is best-effort", "Next: runnerkit doctor --repo owner/repo"} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs output missing %q:\n%s", want, out)
		}
	}
	for _, raw := range []string{"registration-token-secret-logs", "removal-token-secret-logs", "github_pat_secretlogs", "HCLOUD_TOKEN=supersecret", "-----BEGIN OPENSSH PRIVATE KEY-----", "alice@example.com:22"} {
		if strings.Contains(out, raw) {
			t.Fatalf("logs leaked %q:\n%s", raw, out)
		}
	}
	for _, redacted := range []string{"<redacted:runner-registration-token>", "<redacted:runner-removal-token>", "<redacted:github-token>", "<redacted:provider-credential>", "<redacted:ssh-private-key>", "<redacted:machine-ref>"} {
		if !strings.Contains(out, redacted) {
			t.Fatalf("logs missing redaction %q:\n%s", redacted, out)
		}
	}
}

func TestLogsJSONRedactionsApplied(t *testing.T) {
	stateDir := t.TempDir()
	repo := saveHealthyState(t, stateDir)
	out, _, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{}, logsRemoteWithSecrets(), "--json", "logs", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("json logs returned error: %v", err)
	}
	if !strings.Contains(out, `"redactions_applied":true`) || !strings.Contains(out, `"since":"1h"`) || !strings.Contains(out, `"lines":200`) {
		t.Fatalf("json logs missing contract fields:\n%s", out)
	}
	for _, raw := range []string{"registration-token-secret-logs", "removal-token-secret-logs", "github_pat_secretlogs", "HCLOUD_TOKEN=supersecret", "alice@example.com:22"} {
		if strings.Contains(out, raw) {
			t.Fatalf("json logs leaked %q:\n%s", raw, out)
		}
	}
}

func TestLogsEphemeralRendersForwardingWarningAndArchiveSections(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.HealthyRepositoryState()
	repo.Runner.Mode = "ephemeral"
	repo.Ephemeral = state.EphemeralMetadata{Enabled: true, TTL: "24h", LogArchivePath: "/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-local/logs", FinalizerStatus: "pending", CleanupCommand: "runnerkit down --repo owner/repo"}
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	exec := &testsupport.RemoteExecutor{Results: map[string]remote.Result{
		ops.CommandLogsSystemdJournal:       {Stdout: "journal", ExitCode: 0},
		ops.CommandLogsRunnerDiagList:       {Stdout: "/opt/actions-runner/runnerkit-owner-repo-local/_diag/Runner_1.log\n", ExitCode: 0},
		ops.CommandLogsRunnerDiagTail:       {Stdout: "diag", ExitCode: 0},
		ops.CommandLogsEphemeralArchiveList: {Stdout: "/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-local/logs/Runner_1.log\n/var/lib/runnerkit/ephemeral/runnerkit-owner-repo-local/logs/systemd-journal.log\n", ExitCode: 0},
		ops.CommandLogsEphemeralArchiveTail: {Stdout: "preserved", ExitCode: 0},
	}}
	out, _, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{}, exec, "logs", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("ephemeral logs returned error: %v", err)
	}
	for _, want := range []string{
		"/var/lib/runnerkit/ephemeral/",
		"RunnerKit preserves best-effort logs only; configure external log forwarding for production-grade ephemeral troubleshooting.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("ephemeral logs missing %q:\n%s", want, out)
		}
	}
}

func TestLogsCloudProviderMetadata(t *testing.T) {
	stateDir := t.TempDir()
	repo := testsupport.CloudRepositoryState()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("save state: %v", err)
	}
	out, _, err := executeStatusForTest(t, stateDir, &testsupport.GitHubService{}, logsRemoteWithSecrets(), "logs", "--repo", repo.Repo.FullName, "--no-color")
	if err != nil {
		t.Fatalf("cloud logs returned error: %v", err)
	}
	for _, want := range []string{"Provider: Hetzner fsn1 cpx22 ubuntu-24.04", "Billable resources: server:srv-123", "systemd journal"} {
		if !strings.Contains(out, want) {
			t.Fatalf("cloud logs missing %q:\n%s", want, out)
		}
	}
}
