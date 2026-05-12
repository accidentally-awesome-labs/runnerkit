package cli

import (
	"fmt"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

func appendSharedHostDoctorFinding(report *ops.DoctorReport, store rkstate.Store, current rkstate.RepositoryState) {
	if strings.TrimSpace(current.Machine.HostRef) == "" {
		return
	}
	repos, err := store.ListRepositories()
	if err != nil {
		return
	}
	port := current.Machine.Port
	if port == 0 {
		port = 22
	}
	key, err := remote.CanonicalHostKey(current.Machine.HostRef, port)
	if err != nil {
		return
	}
	var siblings []string
	for _, r := range repos {
		if r.Repo.FullName == current.Repo.FullName {
			continue
		}
		if strings.TrimSpace(r.Machine.HostRef) == "" {
			continue
		}
		p := r.Machine.Port
		if p == 0 {
			p = 22
		}
		k, err := remote.CanonicalHostKey(r.Machine.HostRef, p)
		if err != nil || k != key {
			continue
		}
		siblings = append(siblings, r.Repo.FullName)
	}
	if len(siblings) == 0 {
		return
	}
	report.Findings = append(report.Findings, ops.Finding{
		ID:          "byo.multi_repo_shared_host",
		Severity:    string(ops.SeverityPass),
		Source:      "inventory",
		Evidence:    fmt.Sprintf("This SSH host (%s) also has local RunnerKit state for: %s", key, strings.Join(siblings, ", ")),
		Remediation: "Jobs on self-hosted runners share disk and environment. Use `runnerkit list --host` for an overview. See docs/troubleshooting/multi-repo.md for PAT scope and isolation notes.",
	})
}
