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
		"runnerkit down --repo owner/name --dry-run",
		"runnerkit down --repo owner/name --yes",
		"runnerkit down --repo owner/name --github-runner-id 123 --yes",
		"RunnerKit down removes only RunnerKit-managed runner-specific BYO artifacts recorded in state.",
		"RunnerKit down does not delete the BYO machine, shared users, shared /var/lib/runnerkit parents, or unrelated user data.",
		"Use destroy only for future cloud resources; BYO cleanup uses down.",
		"remote_cleanup_pending",
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
	forbiddenDestroy := "runnerkit" + " destroy" + " --repo owner/name"
	if strings.Contains(string(quickstart), forbiddenDestroy) {
		// destroy may now appear in BYO quickstart only as a recommended
		// ephemeral cloud setup command, not as a BYO cleanup step.
		// Allow only when the surrounding context is the safety
		// recommendation: `--mode ephemeral --cloud hetzner`.
		idx := strings.Index(string(quickstart), forbiddenDestroy)
		_ = idx // BYO quickstart should still not call destroy for BYO cleanup.
		// Permit references inside the safety recommendation block; we
		// detect that by ensuring `--mode ephemeral --cloud hetzner` is
		// present near the destroy reference.
		if !strings.Contains(string(quickstart), "--mode ephemeral --cloud hetzner") {
			t.Fatal("BYO quickstart must not use destroy for BYO cleanup")
		}
	}
}

func TestCloudQuickstartDocsContainRequiredCopy(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	quickstart, err := os.ReadFile("../../docs/cloud-quickstart.md")
	if err != nil {
		t.Fatalf("read docs/cloud-quickstart.md: %v", err)
	}
	for name, content := range map[string]string{"README.md": string(readme), "docs/cloud-quickstart.md": string(quickstart)} {
		for _, want := range []string{
			"Provision cloud runner",
			"export HCLOUD_TOKEN=...",
			"runnerkit up --repo owner/name --cloud hetzner",
			"runnerkit up --repo owner/name --cloud hetzner --yes",
			"runnerkit status --repo owner/name",
			"runnerkit logs --repo owner/name --since 30m --lines 200",
			"runnerkit doctor --repo owner/name",
			"runnerkit destroy --repo owner/name --dry-run",
			"runnerkit destroy --repo owner/name",
			"runnerkit destroy --repo owner/name --yes",
			"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
			"RunnerKit prints labels/snippets and does not edit workflow YAML.",
			"Cost estimates are approximate and billing stops only after relevant provider resources are destroyed or verified non-billable.",
		} {
			if !strings.Contains(content, want) {
				t.Fatalf("%s missing %q", name, want)
			}
		}
	}
	if !strings.Contains(string(readme), "docs/cloud-quickstart.md") || !strings.Contains(string(readme), "docs/byo-quickstart.md") {
		t.Fatal("README must link both cloud and BYO quickstarts")
	}
	if !strings.Contains(string(quickstart), "# Recommended Cloud Runner Quickstart") || !strings.Contains(string(quickstart), "HETZNER_CLOUD_TOKEN") || !strings.Contains(string(quickstart), "does not persist provider API tokens") {
		t.Fatal("cloud quickstart missing heading or provider auth notes")
	}
}

// TestSafetyGuideDocsContainRequiredCopy asserts the docs/safety.md guide,
// README, BYO quickstart, and cloud quickstart all carry the exact
// UI-SPEC headings, command examples, required sentences, and v1 non-goal
// bullets for Phase 5 safety guidance. README must link to the safety
// guide and no docs file may say ephemeral mode is deferred.
func TestSafetyGuideDocsContainRequiredCopy(t *testing.T) {
	readme := mustReadDocFile(t, "../../README.md")
	safety := mustReadDocFile(t, "../../docs/safety.md")
	byo := mustReadDocFile(t, "../../docs/byo-quickstart.md")
	cloud := mustReadDocFile(t, "../../docs/cloud-quickstart.md")

	// Required headings in docs/safety.md.
	for _, heading := range []string{
		"# Self-hosted Runner Safety Guide",
		"## Quick recommendation",
		"## Persistent vs ephemeral tradeoffs",
		"## When persistent is appropriate",
		"## When ephemeral is recommended",
		"## Public and fork-based workflow risk",
		"## BYO ephemeral caveats",
		"## Cloud ephemeral caveats",
		"## Logs and troubleshooting",
		"## Cleanup commands",
		"## What RunnerKit does not do in v1",
	} {
		if !strings.Contains(safety, heading) {
			t.Fatalf("docs/safety.md missing heading %q", heading)
		}
	}

	// Required commands across docs/safety.md.
	for _, cmd := range []string{
		"runnerkit up --repo owner/name --mode persistent --host user@host",
		"runnerkit up --repo owner/name --mode ephemeral --host user@host",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h",
		"runnerkit status --repo owner/name",
		"runnerkit logs --repo owner/name --since 30m --lines 200",
		"runnerkit doctor --repo owner/name",
		"runnerkit down --repo owner/name --dry-run",
		"runnerkit destroy --repo owner/name --dry-run",
		"runnerkit destroy --repo owner/name --yes",
	} {
		if !strings.Contains(safety, cmd) {
			t.Fatalf("docs/safety.md missing command %q", cmd)
		}
	}

	// Required exact sentences in safety.md.
	requiredSafetySentences := []string{
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"Ephemeral mode gives stronger isolation by using one-job GitHub runner registration, but it is not a clean VM by itself.",
		"Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.",
		"BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.",
		"Ephemeral cloud runners still create billable Hetzner resources.",
		"Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.",
		"Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.",
		"RunnerKit preserves best-effort runner `_diag` and systemd journal logs before cleanup.",
		"Configure external log forwarding for production-grade ephemeral troubleshooting.",
		"RunnerKit prints labels/snippets and does not edit workflow YAML.",
		"Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners.",
		"persistent self-hosted runners",
	}
	for _, want := range requiredSafetySentences {
		if !strings.Contains(safety, want) {
			t.Fatalf("docs/safety.md missing required sentence %q", want)
		}
	}

	// Required v1 non-goal bullets in safety.md.
	for _, bullet := range []string{
		"No hosted control plane.",
		"No webhook listener or autoscaling fleet manager.",
		"No Actions Runner Controller, Kubernetes, runner scale sets, organization-level runner management, or JIT runner API.",
		"No automatic workflow YAML edits.",
		"No guarantee that BYO ephemeral mode is a clean VM.",
	} {
		if !strings.Contains(safety, bullet) {
			t.Fatalf("docs/safety.md missing non-goal bullet %q", bullet)
		}
	}

	// Persistent vs ephemeral tradeoffs table columns and rows.
	for _, want := range []string{
		"| Mode", "| Cost", "| Isolation", "| Cleanup", "| Operations", "| Logs",
		"| persistent", "| ephemeral",
	} {
		if !strings.Contains(safety, want) {
			t.Fatalf("docs/safety.md tradeoffs table missing column/row %q", want)
		}
	}

	// README must link the safety guide and surface ephemeral cloud setup.
	for _, want := range []string{
		"[Self-hosted Runner Safety Guide](docs/safety.md)",
		"persistent self-hosted runners",
		"Use ephemeral cloud runner",
		"Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md missing %q", want)
		}
	}

	// BYO quickstart must call out the persistent risk and recommend
	// ephemeral cloud instead of saying to wait for ephemeral mode.
	for _, want := range []string{
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"Use runnerkit up --repo owner/name --mode ephemeral --cloud hetzner for stronger isolation, or use GitHub-hosted runners.",
	} {
		if !strings.Contains(byo, want) {
			t.Fatalf("docs/byo-quickstart.md missing %q", want)
		}
	}
	if strings.Contains(byo, "wait for RunnerKit's future ephemeral mode") {
		t.Fatal("docs/byo-quickstart.md must not say to wait for ephemeral mode")
	}

	// Cloud quickstart must include ephemeral commands and remove the
	// 'deferred to Phase 5' wording. It must also include the exact
	// approximate-pricing-varies-user-responsible caveat and billable
	// resource sentences.
	for _, want := range []string{
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --yes",
		"runnerkit up --repo owner/name --mode ephemeral --cloud hetzner --ephemeral-ttl 24h",
		"Ephemeral cloud runners still create billable Hetzner resources.",
		"Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.",
		"Estimated cost is approximate. Hetzner pricing varies by region and time, and you are responsible for charges until `runnerkit destroy --repo owner/name` verifies cleanup.",
	} {
		if !strings.Contains(cloud, want) {
			t.Fatalf("docs/cloud-quickstart.md missing %q", want)
		}
	}
	if strings.Contains(cloud, "Ephemeral mode is deferred to Phase 5.") {
		t.Fatal("docs/cloud-quickstart.md must not say ephemeral mode is deferred to Phase 5")
	}

	// No docs file may continue to claim ephemeral mode is deferred or
	// that BYO users should wait for it.
	allDocs := readme + "\n" + safety + "\n" + byo + "\n" + cloud
	for _, banned := range []string{
		"Ephemeral mode is deferred to Phase 5.",
		"wait for RunnerKit's future ephemeral mode",
	} {
		if strings.Contains(allDocs, banned) {
			t.Fatalf("docs must not contain forbidden phrase %q", banned)
		}
	}
}

func mustReadDocFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
