package cli

import (
	"go/version"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The docs tests below pin the user-facing copy to what v1.3.4 actually
// ships: cloud and BYO ephemeral behind --experimental, an explicit
// --cloud-region, no ephemeral cloud, live Hetzner API pricing, the
// disabled lifecycle mutators, and the security disclosures (H-08, H-09).

func TestBYOQuickstartDocsContainRequiredCopy(t *testing.T) {
	readme := mustReadDocFile(t, "../../README.md")
	quickstart := mustReadDocFile(t, "../../docs/byo-quickstart.md")
	combined := readme + "\n" + quickstart
	for _, want := range []string{
		"BYO Persistent Runner Quickstart",
		"BYO persistent runner quickstart",
		"docs/byo-quickstart.md",
		"runnerkit init --print-install-command",
		"runnerkit up --repo owner/name --host user@host",
		"Persistent self-hosted runners are intended for trusted private repositories",
		"RunnerKit does not edit or commit workflow YAML for you.",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
		"runnerkit status --repo owner/name",
		"runnerkit logs --repo owner/name --since 30m --lines 200",
		"runnerkit doctor --repo owner/name",
		"docs/troubleshooting/host-resources.md",
		"Start with RunnerKit's read-only operations commands before manual SSH troubleshooting.",
		"Review logs before sharing; redaction is best-effort for workflow-produced secrets.",
		"runnerkit recover --repo owner/name --dry-run",
		"runnerkit recover --repo owner/name --restart-service --yes",
		"Do not blindly rerun runnerkit up for recovery; start with status, logs, doctor, and recover --dry-run.",
		"RunnerKit fails closed on SSH host-key mismatch and will not recover until you verify the machine identity.",
		"runnerkit down --repo owner/name --dry-run",
		"runnerkit down --repo owner/name --yes",
		"runnerkit down --repo owner/name --github-runner-id 123 --yes",
		"RunnerKit down removes only RunnerKit-managed runner-specific BYO artifacts recorded in state.",
		"RunnerKit down does not delete the BYO machine, shared users, shared /var/lib/runnerkit parents, or unrelated user data.",
		"Use destroy only for RunnerKit-created Hetzner servers; BYO cleanup uses down.",
		"remote_cleanup_pending",
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("README.md + docs/byo-quickstart.md missing %q", want)
		}
	}

	// The BYO quickstart must disclose the privilege model, the repaired
	// install.sh and how the repair was validated.
	for _, want := range []string{
		"Ubuntu x86_64",
		"root-equivalent",
		"`docker` group",
		"run the current one again",
		"fake GitHub API",
		"a real GitHub job run is still required",
		"`recover --reinstall-service` and `recover --reregister` are disabled",
		"`runnerkit upgrade-runner` is disabled",
		"security-posture.md#if-you-already-installed-runnerkit",
	} {
		if !strings.Contains(quickstart, want) {
			t.Fatalf("docs/byo-quickstart.md missing %q", want)
		}
	}

	// Disabled commands must never be offered as a command to run.
	for _, banned := range []string{
		"runnerkit recover --repo owner/name --reinstall-service --yes",
		"runnerkit recover --repo owner/name --reregister --yes",
		"runnerkit upgrade-runner --repo",
		"runnerkit doctor --repo owner/name --fix",
		"runnerkit destroy --repo owner/name",
		"~75",
		"scoped passwordless sudo",
	} {
		if strings.Contains(quickstart, banned) {
			t.Fatalf("docs/byo-quickstart.md must not contain %q", banned)
		}
	}
	badRecoveryCopy := "rerun runnerkit up for recovery"
	allowedWarning := "Do not blindly rerun runnerkit up for recovery"
	if strings.Contains(combined, badRecoveryCopy) && !strings.Contains(combined, allowedWarning) {
		t.Fatal("docs must only mention rerunning up for recovery as a warning")
	}
}

func TestCloudQuickstartDocsContainRequiredCopy(t *testing.T) {
	readme := mustReadDocFile(t, "../../README.md")
	quickstart := mustReadDocFile(t, "../../docs/cloud-quickstart.md")
	// The README's v1.3.3 warning quotes the old invented estimate; that
	// is the only place it may appear.
	oldEstimateWarning := "The \"approx €4.90/month\" cloud estimate is invented"
	if !strings.Contains(readme, oldEstimateWarning) {
		t.Fatalf("README.md must warn that the v1.3.3 estimate %q", oldEstimateWarning)
	}
	readmeWithoutWarning := strings.Replace(readme, oldEstimateWarning, "", 1)
	for name, content := range map[string]string{"README.md": readmeWithoutWarning, "docs/cloud-quickstart.md": quickstart} {
		for _, want := range []string{
			"export HCLOUD_TOKEN=...",
			"--experimental --cloud hetzner --cloud-region <location>",
			"runnerkit status --repo owner/name",
			"runnerkit logs --repo owner/name --since 30m --lines 200",
			"runnerkit doctor --repo owner/name",
			"docs/cloud-quickstart.md",
			"runnerkit destroy --repo owner/name --dry-run",
			"runnerkit destroy --repo owner/name",
			"reported by the Hetzner API",
			"Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup.",
		} {
			if name == "docs/cloud-quickstart.md" && want == "docs/cloud-quickstart.md" {
				want = "docs/troubleshooting/host-resources.md"
			}
			if !strings.Contains(content, want) {
				t.Fatalf("%s missing %q", name, want)
			}
		}
		// Cloud must never be shown without the opt-in flags, and no
		// invented price may appear.
		for _, banned := range []string{
			"runnerkit up --repo owner/name --cloud hetzner",
			"4.90",
			"approx €",
			"Recommended cloud runner",
			"recommended cloud",
		} {
			if strings.Contains(content, banned) {
				t.Fatalf("%s must not contain %q", name, banned)
			}
		}
	}
	if !strings.Contains(readme, "docs/cloud-quickstart.md") || !strings.Contains(readme, "docs/byo-quickstart.md") {
		t.Fatal("README must link both cloud and BYO quickstarts")
	}
	for _, want := range []string{
		"# Hetzner Cloud Runner Quickstart (experimental)",
		"## Provision cloud runner",
		"HETZNER_CLOUD_TOKEN",
		"does not persist provider API tokens",
		"`experimental_required`",
		"`cloud_region_required`",
		"`ephemeral_cloud_disabled`",
		"`cloud_location_unpriced`",
		"`cloud_state_exists`",
		"`wrong_cleanup_command`",
		`"source": "hetzner_api"`,
		`"fetched_at"`,
		`"components"`,
		"runnerkit-cloud-init-v3",
		"not fail fast",
		"0.0.0.0/0",
		"root-equivalent",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
		"RunnerKit prints labels/snippets and does not edit workflow YAML.",
		"runnerkit=true",
	} {
		if !strings.Contains(quickstart, want) {
			t.Fatalf("docs/cloud-quickstart.md missing %q", want)
		}
	}
	for _, banned := range []string{
		"Ruby",
		"cleaned up automatically",
		".runnerkit/config.yaml",
		"~75",
		"--ephemeral-ttl",
		"runnerkit upgrade-runner",
	} {
		if strings.Contains(quickstart, banned) {
			t.Fatalf("docs/cloud-quickstart.md must not contain %q", banned)
		}
	}
}

// TestSafetyGuideDocsContainRequiredCopy asserts docs/safety.md retracts
// the old "stronger isolation" / ephemeral-cloud advice (H-09, A-08) and
// routes public or untrusted work to GitHub-hosted runners.
func TestSafetyGuideDocsContainRequiredCopy(t *testing.T) {
	readme := mustReadDocFile(t, "../../README.md")
	safety := mustReadDocFile(t, "../../docs/safety.md")
	byo := mustReadDocFile(t, "../../docs/byo-quickstart.md")

	for _, heading := range []string{
		"# Self-hosted Runner Safety Guide",
		"## Quick recommendation",
		"## Persistent vs ephemeral tradeoffs",
		"## When persistent is appropriate",
		"## Public and fork-based workflow risk",
		"## BYO ephemeral caveats",
		"## Cloud ephemeral caveats",
		"## Logs and troubleshooting",
		"## Cleanup commands",
		"## What RunnerKit does not do",
	} {
		if !strings.Contains(safety, heading) {
			t.Fatalf("docs/safety.md missing heading %q", heading)
		}
	}
	if strings.Contains(safety, "## When ephemeral is recommended") {
		t.Fatal("docs/safety.md must not recommend ephemeral mode")
	}

	for _, cmd := range []string{
		"runnerkit up --repo owner/name --mode persistent --host user@host",
		"runnerkit up --repo owner/name --mode ephemeral --experimental --host user@host",
		"runnerkit status --repo owner/name",
		"runnerkit logs --repo owner/name --since 30m --lines 200",
		"runnerkit doctor --repo owner/name",
		"runnerkit doctor --repo owner/name --deep",
		"runnerkit down --repo owner/name --dry-run",
		"runnerkit destroy --repo owner/name --dry-run",
		"runnerkit destroy --repo owner/name --yes",
	} {
		if !strings.Contains(safety, cmd) {
			t.Fatalf("docs/safety.md missing command %q", cmd)
		}
	}

	for _, want := range []string{
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"use\n  GitHub-hosted runners.",
		"Ephemeral mode is a one-job GitHub runner registration. It is not isolation\nand not a clean VM.",
		"Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online.",
		"BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine.",
		"It has known defects (the finalizer runs unprivileged and the TTL is\n  ignored) and is untested in this release.",
		"Ephemeral cloud runners are disabled in v1.3.4",
		"`ephemeral_cloud_disabled`",
		"Billing stops only after\n`runnerkit destroy --repo owner/name` verifies cleanup.",
		"RunnerKit preserves best-effort runner `_diag` and systemd journal logs before cleanup.",
		"Heavy workflows can **OOM** small VMs; preflight warns on low **MemAvailable** / missing swap, and `runnerkit doctor --deep` can flag likely kernel or linker kills from bounded journals (**RKD-BOOT-016..018**). See [Host resources and OOM](troubleshooting/host-resources.md).",
		"RunnerKit prints labels/snippets and does not edit workflow YAML.",
		"Do not use `runs-on: self-hosted` alone for RunnerKit-managed runners.",
		"persistent self-hosted runners",
		"root-equivalent sudoers",
		"security-posture.md",
		"`wrong_cleanup_command`",
	} {
		if !strings.Contains(safety, want) {
			t.Fatalf("docs/safety.md missing required text %q", want)
		}
	}
	for _, bullet := range []string{
		"No hosted control plane.",
		"No webhook listener or autoscaling fleet manager.",
		"No Actions Runner Controller, Kubernetes, runner scale sets, organization-level runner management, or JIT runner API.",
		"No automatic workflow YAML edits.",
		"No isolation between jobs, in any mode.",
	} {
		if !strings.Contains(safety, bullet) {
			t.Fatalf("docs/safety.md missing non-goal bullet %q", bullet)
		}
	}
	for _, want := range []string{
		"| Mode", "| Isolation", "| Cleanup", "| Operations", "| Logs",
		"| persistent", "| ephemeral",
	} {
		if !strings.Contains(safety, want) {
			t.Fatalf("docs/safety.md tradeoffs table missing column/row %q", want)
		}
	}
	// The retracted claims must be gone (the correction note may name
	// "stronger isolation per job" only inside quotes).
	for _, banned := range []string{
		"Use ephemeral cloud runner",
		"where you want stronger isolation",
		"if you need stronger isolation",
		"--ephemeral-ttl",
		"finalized and cleaned up",
	} {
		if strings.Contains(safety, banned) {
			t.Fatalf("docs/safety.md must not contain %q", banned)
		}
	}

	for _, want := range []string{
		"[Self-hosted Runner Safety Guide](docs/safety.md)",
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"persistent self-hosted runners",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
		"For public or untrusted code, use GitHub-hosted runners.",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md missing %q", want)
		}
	}
	for _, banned := range []string{"Use ephemeral cloud runner", "Ephemeral mode gives stronger isolation"} {
		if strings.Contains(readme, banned) {
			t.Fatalf("README.md must not contain %q", banned)
		}
	}

	for _, want := range []string{
		"Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows.",
		"Use GitHub-hosted runners for those; RunnerKit's ephemeral mode is not isolation.",
	} {
		if !strings.Contains(byo, want) {
			t.Fatalf("docs/byo-quickstart.md missing %q", want)
		}
	}
}

// TestSafetyDocsGrepContract guards phrases across files and makes sure
// "autoscaling fleet manager" only ever appears as a non-goal.
func TestSafetyDocsGrepContract(t *testing.T) {
	files := map[string]string{
		"README.md":                mustReadDocFile(t, "../../README.md"),
		"docs/safety.md":           mustReadDocFile(t, "../../docs/safety.md"),
		"docs/byo-quickstart.md":   mustReadDocFile(t, "../../docs/byo-quickstart.md"),
		"docs/cloud-quickstart.md": mustReadDocFile(t, "../../docs/cloud-quickstart.md"),
	}
	mustContainAcrossFiles := []struct {
		text  string
		paths []string
	}{
		{"persistent self-hosted runners", []string{"README.md", "docs/safety.md"}},
		{"Ephemeral mode is not a fleet manager", []string{"docs/safety.md"}},
		{"GitHub-hosted runners", []string{"README.md", "docs/safety.md", "docs/byo-quickstart.md", "docs/cloud-quickstart.md"}},
		{"root-equivalent", []string{"README.md", "docs/byo-quickstart.md", "docs/cloud-quickstart.md"}},
		{"Configure external log forwarding if you need complete job logs.", []string{"docs/safety.md"}},
	}
	for _, expectation := range mustContainAcrossFiles {
		for _, path := range expectation.paths {
			if !strings.Contains(files[path], expectation.text) {
				t.Fatalf("%s missing required text %q", path, expectation.text)
			}
		}
	}

	for path, content := range files {
		idx := 0
		for {
			pos := strings.Index(content[idx:], "autoscaling fleet manager")
			if pos < 0 {
				break
			}
			absolute := idx + pos
			lineStart := strings.LastIndex(content[:absolute], "\n")
			if lineStart < 0 {
				lineStart = 0
			} else {
				lineStart++
			}
			lower := strings.ToLower(content[lineStart:absolute])
			if !strings.Contains(lower, "no ") && !strings.Contains(lower, "not a ") && !strings.Contains(lower, " or ") && !strings.Contains(lower, "without ") {
				t.Fatalf("%s mentions \"autoscaling fleet manager\" without negation; surrounding context: %q", path, content[lineStart:absolute+len("autoscaling fleet manager")+8])
			}
			idx = absolute + len("autoscaling fleet manager")
		}
	}
}

// userFacingMarkdown returns README, the root policy docs and every
// Markdown file under docs/, keyed by repo-relative path.
func userFacingMarkdown(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, name := range []string{"README.md", "CHANGELOG.md", "CONTRIBUTING.md", "SECURITY.md", "CLAUDE.md"} {
		out[name] = mustReadDocFile(t, "../../"+name)
	}
	err := filepath.WalkDir("../../docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, relErr := filepath.Rel("../..", path)
		if relErr != nil {
			return relErr
		}
		out[filepath.ToSlash(rel)] = mustReadDocFile(t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs: %v", err)
	}
	if len(out) < 15 {
		t.Fatalf("expected README, root docs and docs/**/*.md, found only %d files", len(out))
	}
	return out
}

// TestNoDocsRecommendEphemeralCloud is the docs half of A-08's
// TestNoCopyRecommendsEphemeralCloud: no Markdown may show the
// ephemeral-cloud command, which is disabled because the VM kept billing.
func TestNoDocsRecommendEphemeralCloud(t *testing.T) {
	for path, content := range userFacingMarkdown(t) {
		if strings.Contains(content, "--mode ephemeral --cloud") {
			t.Errorf("%s shows the disabled ephemeral cloud command (--mode ephemeral --cloud)", path)
		}
	}
}

// TestDocsDropStaleAndUnsafeAdvice fails on advice that is wrong or harmful
// for v1.3.4 (H-09 "Docs to fix", A-06, SEC-9).
func TestDocsDropStaleAndUnsafeAdvice(t *testing.T) {
	banned := []string{
		"repo,workflow",                              // SEC-9: workflow scope is not needed
		"runnerkit upgrade-runner --repo",            // disabled (A-06a)
		"--reinstall-service --yes",                  // disabled (A-06b)
		"--reinstall-service --dry-run",              // disabled (A-06b)
		"--reregister --yes",                         // disabled (A-06b)
		"--reregister --dry-run",                     // disabled (A-06b)
		"runnerkit doctor --repo owner/name --fix",   // disabled (A-06a)
		"idempotent — safe to re-run",                // docs/upgrade.md (P0-3)
		"rm -rf /opt/actions-runner/runnerkit-*",     // wipes every repo's runner
		"rm $HOME/.local/state/runnerkit/state.json", // drops every repo's state and cloud IDs
		"runnerkit-cloud-init-v2",                    // code is v3
		"scoped to RunnerKit bootstrap commands only",
		"systemctl status runnerkit-runner", // unit does not exist
		"~75",                               // there are 70 baseline packages
		"TAG=v1.0.0",                        // v1.0.0 has no GitHub Release
		"10-minute",
		"10 minutes",
		"cheaper than GitHub",
		"Recommended Cloud Runner",
		"recommended cloud",
		"recommended Hetzner",
		"disable --now 'actions.runner.*'", // systemctl disable takes no glob
	}
	for path, content := range userFacingMarkdown(t) {
		if path == "CHANGELOG.md" {
			// History may quote old wording; it is checked separately.
			continue
		}
		for _, phrase := range banned {
			if strings.Contains(content, phrase) {
				t.Errorf("%s contains stale or unsafe advice %q", path, phrase)
			}
		}
	}
	platforms := mustReadDocFile(t, "../../docs/runner-platforms.md")
	if strings.Contains(platforms, "treat as advanced BYO") || !strings.Contains(platforms, "| **Linux arm64** | **Not supported.**") || !strings.Contains(platforms, "| **macOS** | **Not supported.**") {
		t.Error("docs/runner-platforms.md must withdraw the arm64 and macOS support claims")
	}
}

// TestReadmeHonestyBanner pins the H-08 README rewrite.
func TestReadmeHonestyBanner(t *testing.T) {
	readme := mustReadDocFile(t, "../../README.md")
	for _, want := range []string{
		"**Status:** Experimental. Maintained on a capped-hours basis until a published go/kill decision on 2026-12-21.",
		"## Known issues",
		"### If you are on v1.3.3 or older",
		"## When NOT to use RunnerKit",
		"## What BYO setup installs on the host",
		"fake GitHub API",
		"**root-equivalent**",
		"`RUNNERKIT_GRANT_CI_SUDO=1`",
		"requires\n  `--experimental` and an explicit `--cloud-region`",
		"Ephemeral cloud runners are disabled.",
		"`upgrade-runner`, `doctor --fix`,\n  `recover --reinstall-service` and `recover --reregister` refuse to run",
		"Only **Ubuntu x86_64** runner hosts are supported.",
		"4.5–5 GB",
		"removed in v1.0.8",
		"TAG=vX.Y.Z",
		"(LICENSE)",
		"docs/maintainers.md",
		"docs/security-posture.md",
		"CONTRIBUTING.md",
		"SECURITY.md",
		"CHANGELOG.md",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md missing %q", want)
		}
	}
	for _, banned := range []string{
		"(D-0", "D-01", "D-02", "D-05",
		"## Maintainers: releases",
		"reliable GitHub Actions self-hosted runners",
	} {
		if strings.Contains(readme, banned) {
			t.Fatalf("README.md must not contain %q", banned)
		}
	}
}

// TestChangelogKnownIssuesMatchReadme: CLAUDE.md requires the CHANGELOG's
// Known-issues block to stay in sync with the README "Known issues"
// section, so both carry the same list, word for word. Line wrapping may
// differ; everything else, including loose-list blank lines and extra
// paragraphs, must match.
func TestChangelogKnownIssuesMatchReadme(t *testing.T) {
	readme := knownIssuesList(t, "README.md", mustReadDocFile(t, "../../README.md"), "## Known issues", "\nFull list:")
	changelog := knownIssuesList(t, "CHANGELOG.md", mustReadDocFile(t, "../../CHANGELOG.md"), "### Known issues", "")
	if n := strings.Count(readme, "- **"); n < 8 {
		t.Fatalf("README Known issues has %d bold bullets; the list was not found", n)
	}
	if readme == changelog {
		return
	}
	at := 0
	for at < len(readme) && at < len(changelog) && readme[at] == changelog[at] {
		at++
	}
	from := max(at-60, 0)
	t.Fatalf("Known issues differ at character %d:\nREADME.md:    ...%s\nCHANGELOG.md: ...%s", at, readme[from:min(at+80, len(readme))], changelog[from:min(at+80, len(changelog))])
}

// knownIssuesList returns the section under heading from its first bullet
// up to end (a marker that must follow the list) or, when end is empty, up
// to the next heading, with runs of whitespace collapsed to one space.
func knownIssuesList(t *testing.T, name, doc, heading, end string) string {
	t.Helper()
	at := strings.Index(doc, "\n"+heading+"\n")
	if at < 0 {
		t.Fatalf("%s has no %q heading", name, heading)
	}
	section := doc[at+len(heading)+2:]
	if next := strings.Index(section, "\n#"); next >= 0 {
		section = section[:next]
	}
	if end != "" {
		stop := strings.Index(section, end)
		if stop < 0 {
			t.Fatalf("%s: no %q after the Known issues list", name, strings.TrimSpace(end))
		}
		section = section[:stop]
	}
	first := strings.Index(section, "\n- ")
	if first < 0 {
		t.Fatalf("%s: no bullet list under %q", name, heading)
	}
	return strings.Join(strings.Fields(section[first:]), " ")
}

// TestLicensingAndPolicyDocs pins H-01..H-04a and H-09.
func TestLicensingAndPolicyDocs(t *testing.T) {
	license := mustReadDocFile(t, "../../LICENSE")
	for _, want := range []string{"Apache License", "Version 2.0, January 2004", "TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION"} {
		if !strings.Contains(license, want) {
			t.Fatalf("LICENSE missing %q", want)
		}
	}

	contributing := mustReadDocFile(t, "../../CONTRIBUTING.md")
	for _, want := range []string{"Developer Certificate of Origin", "Signed-off-by", "git commit -s", "2 distinct external requests", "W1", "30 build hours", "GOTOOLCHAIN=go1.26", "go generate ./...", "make generate-check"} {
		if !strings.Contains(contributing, want) {
			t.Fatalf("CONTRIBUTING.md missing %q", want)
		}
	}
	if strings.Contains(contributing, "Contributor License Agreement") {
		t.Fatal("CONTRIBUTING.md must not require a CLA")
	}

	security := mustReadDocFile(t, "../../SECURITY.md")
	for _, want := range []string{"security/advisories/new", "**14 days**", "**latest minor release**", "docs/security-posture.md", "**known and disclosed**"} {
		if !strings.Contains(security, want) {
			t.Fatalf("SECURITY.md missing %q", want)
		}
	}

	posture := mustReadDocFile(t, "../../docs/security-posture.md")
	for i := 1; i <= 13; i++ {
		id := "| SEC-" + strconv.Itoa(i) + " |"
		if !strings.Contains(posture, id) {
			t.Fatalf("docs/security-posture.md missing row %q", id)
		}
	}
	for _, want := range []string{
		"## If you already installed RunnerKit",
		"sudo rm -f /etc/sudoers.d/runnerkit-installer /etc/sudoers.d/runnerkit-runner-ci",
		"sudo visudo -c",
		// The non-recursive directory chown is what stops a job from
		// renaming svc.sh/bin and planting its own; -H follows a bin
		// symlink left by runner self-update.
		"sudo chown root:root /opt/actions-runner/runnerkit-*/\n",
		"sudo chown root:root /opt/actions-runner/runnerkit-*/svc.sh",
		"sudo chown -R -H root:root /opt/actions-runner/runnerkit-*/bin",
		// systemctl disable rejects a glob ("globs are not supported
		// for this"), so the teardown stops by pattern and disables
		// the unit names list-unit-files returns.
		"sudo systemctl stop 'actions.runner.*'",
		"systemctl list-unit-files --plain --no-legend 'actions.runner.*'",
		"Any later `runnerkit up` or `register` on this host runs",
		"id -nG runnerkit-runner",
		"sudo gpasswd -d runnerkit-runner docker",
		"runnerkit destroy --repo owner/name",
		"runnerkit=true",
		"## Ephemeral mode is not isolation",
		"GitHub-hosted runners",
		"root-equivalent",
	} {
		if !strings.Contains(posture, want) {
			t.Fatalf("docs/security-posture.md missing %q", want)
		}
	}

	changelog := mustReadDocFile(t, "../../CHANGELOG.md")
	for _, want := range []string{
		"keepachangelog.com",
		"## [Unreleased] — v1.3.4",
		"### Added", "### Changed", "### Fixed", "### Security", "### Known issues",
		"fake GitHub API",
		"(A-20)",
		"### Erratum for v1.3.3",
		"it was removed in v1.0.8",
		`source:
  "hetzner_api"`,
		"## [1.3.3] - 2026-05-18",
		"## [1.0.8] - 2026-05-11",
		"## [1.0.0]",
	} {
		if !strings.Contains(changelog, want) {
			t.Fatalf("CHANGELOG.md missing %q", want)
		}
	}
	if strings.Contains(changelog, "--mode ephemeral --cloud") {
		t.Fatal("CHANGELOG.md must not show the disabled ephemeral cloud command")
	}

	claude := mustReadDocFile(t, "../../CLAUDE.md")
	for _, want := range []string{
		"No feature without 2 distinct external requests",
		"`RenderSudoersEntry` gains no entries",
		"root-equivalent",
		"Never tag a release that claims BYO works",
		"Never pass `--disableupdate`",
		"70 apt packages",
		"**never populated**",
		"not fail-fast",
	} {
		if !strings.Contains(claude, want) {
			t.Fatalf("CLAUDE.md missing %q", want)
		}
	}
}

// TestGoModPinsPatchedToolchain guards against go.mod dropping its toolchain
// line: with only `go 1.26.0`, GOTOOLCHAIN=auto on an older local Go
// downloads go1.26.0, which lacks the stdlib security fixes of later 1.26.x
// patches. It also pins the macOS 12 floor that Go 1.26 imposes.
func TestGoModPinsPatchedToolchain(t *testing.T) {
	const minToolchain = "go1.26.8"
	var toolchain string
	for _, line := range strings.Split(mustReadDocFile(t, "../../go.mod"), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == "toolchain" {
			toolchain = f[1]
		}
	}
	if toolchain == "" {
		t.Fatalf("go.mod has no toolchain line; want toolchain %s or newer", minToolchain)
	}
	if version.Compare(toolchain, minToolchain) < 0 {
		t.Fatalf("go.mod toolchain %s is older than %s", toolchain, minToolchain)
	}
	for _, doc := range []string{"../../README.md", "../../CHANGELOG.md", "../../docs/runner-platforms.md"} {
		if !strings.Contains(mustReadDocFile(t, doc), "macOS 12") {
			t.Fatalf("%s must state the macOS 12 minimum for the CLI (Go 1.26)", doc)
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
