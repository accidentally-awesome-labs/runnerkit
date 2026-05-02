package ops

import (
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/bootstrap"
	"github.com/salar/runnerkit/internal/state"
	"github.com/salar/runnerkit/internal/testsupport"
)

// TestDoctor_StaleRunnerVersion: when RunnerTemplateVersion is older than
// bootstrap.RunnerVersion, BuildDoctorReport emits exactly one
// runner_version_stale finding with severity=warning and a remediation
// referencing `runnerkit upgrade-runner`. When the template matches the
// pin, no stale finding is emitted.
func TestDoctor_StaleRunnerVersion(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	repo.RunnerTemplateVersion = "2.330.0" // older than bootstrap.RunnerVersion ("2.334.0")
	observed := ObservedRunner{
		Repo:         repo.Repo.FullName,
		StatePath:    "/state.json",
		StatePresent: true,
		State:        &repo,
		GitHub:       GitHubFact{Found: true, ID: 123, Name: repo.Runner.Name, Status: "online", Labels: repo.Runner.Labels},
		SSH:          SSHFact{Reachable: true, HostKey: "matched"},
		Service:      ServiceFact{Service: repo.Machine.ServiceName, ActiveState: "active"},
		Labels:       CompareLabels(repo.Runner.Labels, repo.Runner.Labels),
	}
	report := BuildDoctorReport(repo, observed, DeepChecks{InstallPathOK: true, WorkDirOK: true})

	count := 0
	var got Finding
	for _, f := range report.Findings {
		if f.ID == "runner_version_stale" {
			count++
			got = f
		}
	}
	if count != 1 {
		t.Fatalf("runner_version_stale findings = %d, want 1; findings=%+v", count, report.Findings)
	}
	if got.Severity != string(SeverityWarning) {
		t.Fatalf("severity = %q, want %q", got.Severity, SeverityWarning)
	}
	if !strings.Contains(got.Evidence, "2.330.0") || !strings.Contains(got.Evidence, bootstrap.RunnerVersion) {
		t.Fatalf("evidence missing version strings: %q", got.Evidence)
	}
	if !strings.Contains(got.Remediation, "runnerkit upgrade-runner") {
		t.Fatalf("remediation missing `runnerkit upgrade-runner`: %q", got.Remediation)
	}

	// Same scenario with RunnerTemplateVersion == bootstrap.RunnerVersion -> no finding.
	matched := testsupport.HealthyRepositoryState()
	matched.RunnerTemplateVersion = bootstrap.RunnerVersion
	matchedObs := observed
	matchedObs.State = &matched
	matchedReport := BuildDoctorReport(matched, matchedObs, DeepChecks{InstallPathOK: true, WorkDirOK: true})
	for _, f := range matchedReport.Findings {
		if f.ID == "runner_version_stale" {
			t.Fatalf("expected no runner_version_stale when template matches pin; got %+v", f)
		}
	}

	// Sanity: empty RunnerTemplateVersion (legacy state) should also skip the finding.
	legacy := testsupport.HealthyRepositoryState()
	legacy.RunnerTemplateVersion = ""
	legacyObs := observed
	legacyObs.State = &legacy
	legacyReport := BuildDoctorReport(legacy, legacyObs, DeepChecks{InstallPathOK: true, WorkDirOK: true})
	for _, f := range legacyReport.Findings {
		if f.ID == "runner_version_stale" {
			t.Fatalf("expected no runner_version_stale for empty template version; got %+v", f)
		}
	}

	// Ensure compile reference to state.RepositoryState lives in this file.
	_ = state.RepositoryState{}
}
