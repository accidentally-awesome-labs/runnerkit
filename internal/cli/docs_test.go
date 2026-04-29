package cli

import (
	"os"
	"strings"
	"testing"
)

func TestBYOQuickstartDocsContainRequiredCopy(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	quickstart, err := os.ReadFile("../../docs/byo-quickstart.md")
	if err != nil {
		t.Fatalf("read docs/byo-quickstart.md: %v", err)
	}
	combined := string(readme) + "\n" + string(quickstart)
	for _, want := range []string{
		"BYO Persistent Runner Quickstart",
		"BYO persistent runner quickstart",
		"docs/byo-quickstart.md",
		"runnerkit up --repo owner/name --host user@host",
		"Persistent self-hosted runners are intended for trusted private repositories",
		"RunnerKit does not edit or commit workflow YAML for you.",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
		"runnerkit status --repo owner/name",
		"runnerkit logs --repo owner/name --lines 50",
		"runnerkit logs --repo owner/name --since 30m --lines 200",
		"runnerkit doctor --repo owner/name",
		"Start with RunnerKit's read-only operations commands before manual SSH troubleshooting.",
		"Review logs before sharing; redaction is best-effort for workflow-produced secrets.",
		"runnerkit recover --repo owner/name --dry-run",
		"runnerkit recover --repo owner/name --restart-service --yes",
		"runnerkit recover --repo owner/name --reinstall-service --yes",
		"runnerkit recover --repo owner/name --reregister --yes",
		"Do not blindly rerun runnerkit up for recovery; start with status, logs, doctor, and recover --dry-run.",
		"RunnerKit fails closed on SSH host-key mismatch and will not recover until you verify the machine identity.",
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("docs missing %q", want)
		}
	}
	forbidden := "doctor" + " --" + "fix"
	if strings.Contains(combined, forbidden) {
		t.Fatal("docs must not introduce the forbidden doctor mutation flag")
	}
	badRecoveryCopy := "rerun runnerkit up for recovery"
	allowedWarning := "Do not blindly rerun runnerkit up for recovery"
	if strings.Contains(combined, badRecoveryCopy) && !strings.Contains(combined, allowedWarning) {
		t.Fatal("docs must only mention rerunning up for recovery as a warning")
	}
}
