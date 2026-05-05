package ops

import (
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/preflight"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

func TestBuildDoctorReportFindingsAndRemediations(t *testing.T) {
	repo := testsupport.PartialCleanupRepositoryState()
	observed := ObservedRunner{Repo: repo.Repo.FullName, StatePath: "/state.json", StatePresent: true, State: &repo, GitHub: GitHubFact{Found: true, ID: 123, Name: repo.Runner.Name, Status: "offline", Labels: []string{"self-hosted", "runnerkit", "runnerkit-owner-repo"}}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repo.Machine.ServiceName, ActiveState: "failed", SubState: "failed"}, Labels: CompareLabels(repo.Runner.Labels, []string{"self-hosted", "runnerkit", "runnerkit-owner-repo"})}
	report := BuildDoctorReport(repo, observed, DeepChecks{InstallPathOK: false, WorkDirOK: false, InstallPathError: "config.sh missing", WorkDirError: "work dir missing", Preflight: preflight.Report{Results: []preflight.Result{{ID: preflight.CheckNetworkGitHub, Severity: preflight.SeverityFailure, Message: "network failed"}}}})
	text := ""
	for _, finding := range report.Findings {
		text += finding.ID + " " + finding.Remediation + "\n"
	}
	for _, want := range []string{"service_failed", "ssh_unreachable", "label_drift", "install_path_missing", "network_github_failed", "cleanup_pending", "runnerkit logs --repo owner/repo --since 30m", "runnerkit recover --repo owner/repo --reregister --dry-run", "runnerkit down --repo owner/repo --dry-run"} {
		if want == "ssh_unreachable" {
			continue
		}
		if !strings.Contains(text, want) {
			t.Fatalf("doctor report missing %q in\n%s", want, text)
		}
	}
}

func TestBuildDoctorReportSSHUnreachable(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	observed := ObservedRunner{Repo: repo.Repo.FullName, StatePath: "/state.json", StatePresent: true, State: &repo, GitHub: GitHubFact{Found: true, ID: 123, Name: repo.Runner.Name, Status: "online", Labels: repo.Runner.Labels}, SSH: SSHFact{Reachable: false, HostKey: "not_checked", Error: "SSH unreachable"}, Service: ServiceFact{Service: repo.Machine.ServiceName}, Labels: CompareLabels(repo.Runner.Labels, repo.Runner.Labels)}
	report := BuildDoctorReport(repo, observed, DeepChecks{InstallPathOK: true, WorkDirOK: true})
	found := false
	for _, finding := range report.Findings {
		if finding.ID == "ssh_unreachable" && strings.Contains(finding.Remediation, "Verify SSH access to alice@example.com:22") {
			found = true
		}
	}
	if !found {
		t.Fatalf("ssh_unreachable finding missing: %#v", report.Findings)
	}
}

// TestDoctor_ByoHostPreparedFinding asserts that when DeepChecks
// reports the remote host has /etc/sudoers.d/runnerkit-installer
// (i.e. `runnerkit byo-prepare` was previously applied), the doctor
// report emits a `byo_host_prepared` finding with informational
// severity. When BYOHostPrepared is false, no such finding is added.
func TestDoctor_ByoHostPreparedFinding(t *testing.T) {
	repo := testsupport.HealthyRepositoryState()
	observed := ObservedRunner{Repo: repo.Repo.FullName, StatePath: "/state.json", StatePresent: true, State: &repo, GitHub: GitHubFact{Found: true, ID: 123, Name: repo.Runner.Name, Status: "online", Labels: repo.Runner.Labels}, SSH: SSHFact{Reachable: true, HostKey: "matched"}, Service: ServiceFact{Service: repo.Machine.ServiceName, ActiveState: "active"}, Labels: CompareLabels(repo.Runner.Labels, repo.Runner.Labels)}

	preparedReport := BuildDoctorReport(repo, observed, DeepChecks{InstallPathOK: true, WorkDirOK: true, BYOHostPrepared: true})
	foundPrepared := false
	for _, finding := range preparedReport.Findings {
		if finding.ID == "byo_host_prepared" {
			foundPrepared = true
			if finding.Severity != string(SeverityPass) {
				t.Fatalf("byo_host_prepared severity = %q, want pass (informational)", finding.Severity)
			}
			if !strings.Contains(finding.Evidence, "/etc/sudoers.d/runnerkit-installer") {
				t.Fatalf("byo_host_prepared evidence missing canonical path: %s", finding.Evidence)
			}
		}
	}
	if !foundPrepared {
		t.Fatal("byo_host_prepared finding missing when BYOHostPrepared=true")
	}

	unpreparedReport := BuildDoctorReport(repo, observed, DeepChecks{InstallPathOK: true, WorkDirOK: true, BYOHostPrepared: false})
	for _, finding := range unpreparedReport.Findings {
		if finding.ID == "byo_host_prepared" {
			t.Fatalf("byo_host_prepared finding emitted when BYOHostPrepared=false: %#v", finding)
		}
	}
}
