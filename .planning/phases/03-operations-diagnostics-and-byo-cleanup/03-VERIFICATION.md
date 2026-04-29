---
phase: 03-operations-diagnostics-and-byo-cleanup
verified: 2026-04-30T02:08:00Z
status: passed
score: 4/4 must-haves verified
requirements:
  [GH-03, REL-01, REL-02, REL-03, REL-04, CLEAN-02, CLEAN-03, STATE-02]
---

# Phase 03: Operations, Diagnostics, and BYO Cleanup Verification Report

**Phase Goal:** RunnerKit reduces self-hosted-runner fragility by reconciling state across GitHub, SSH, and systemd, exposing logs, diagnosing common failures, recovering persistent runners, and cleaning up BYO installs safely.
**Verified:** 2026-04-30T02:08:00Z
**Status:** passed

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                                                                            | Status     | Evidence                                                                                                                                                                                           |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Developer can run `runnerkit status` and see GitHub runner status, local service status, labels, and machine reachability for managed runners.                   | ✓ VERIFIED | `internal/cli/status.go`, `internal/ops/status.go`, `internal/ops/probes.go`, `TestStatusRepoHumanOutputAndReadOnly`, `TestStatusInferredGitRemoteAllJSONLabelDriftAndHostKeyMismatch`             |
| 2   | Developer can run `runnerkit logs` and `runnerkit doctor` to inspect relevant logs and receive actionable remediation guidance without manual SSH spelunking.    | ✓ VERIFIED | `internal/cli/logs.go`, `internal/ops/logs.go`, `internal/cli/doctor.go`, `internal/ops/doctor.go`, `TestLogsDefaultsAndRedactsHumanOutput`, `TestDoctorHidesPassFindingsUnlessVerboseAndReadOnly` |
| 3   | Developer can restart or recover a stopped/offline RunnerKit-managed persistent runner through documented or guided CLI steps.                                   | ✓ VERIFIED | `internal/ops/recovery.go`, `internal/cli/recover.go`, `TestRecoverDryRunRestartReinstallMissingYesAndHostKeyBlock`, `TestRecoverReregisterUpdatesGitHubRunnerID`, docs recovery section           |
| 4   | Developer can deregister stale GitHub runner records and remove BYO runner files/services without deleting unrelated user data, even when some state is missing. | ✓ VERIFIED | `internal/ops/cleanup.go`, `internal/cli/down.go`, `TestDownYesCompleteCleanupDeletesStateAndRedactsToken`, `TestDownPartialAndStaleGitHubOnlyFlows`, safe path tests, docs cleanup section        |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact                                               | Expected                   | Status                 | Details                                                                             |
| ------------------------------------------------------ | -------------------------- | ---------------------- | ----------------------------------------------------------------------------------- |
| `internal/cli/status.go`                               | `runnerkit status` command | ✓ EXISTS + SUBSTANTIVE | Wires `Use: "status"`, `--repo`, `--all`, source facts, snippets, and JSON.         |
| `internal/ops/status.go`                               | Shared health model        | ✓ EXISTS + SUBSTANTIVE | Defines ready/busy/needs_attention/broken/unknown and reason IDs.                   |
| `internal/ops/probes.go`                               | Fast status probes         | ✓ EXISTS + SUBSTANTIVE | Uses `status.ssh.reachable` and `status.systemd.show`, host-key first.              |
| `internal/cli/logs.go` / `internal/ops/logs.go`        | Bounded logs               | ✓ EXISTS + SUBSTANTIVE | Implements journal and `_diag` commands plus redacted output warning.               |
| `internal/cli/doctor.go` / `internal/ops/doctor.go`    | Read-only diagnostics      | ✓ EXISTS + SUBSTANTIVE | Stable finding IDs, pass-finding collapse, remediation commands; no `doctor --fix`. |
| `internal/cli/recover.go` / `internal/ops/recovery.go` | Guided recovery            | ✓ EXISTS + SUBSTANTIVE | Dry-run, `--yes`, restart/reinstall/reregister, token redaction, state update.      |
| `internal/cli/down.go` / `internal/ops/cleanup.go`     | Safe cleanup               | ✓ EXISTS + SUBSTANTIVE | Dry-run, prompts, stale deletion, partial checkpoints, safe path validation.        |
| `docs/byo-quickstart.md` / `README.md`                 | User guidance              | ✓ EXISTS + SUBSTANTIVE | Status/logs/doctor/recover/down commands and cleanup boundaries documented.         |

**Artifacts:** 8/8 verified

### Key Link Verification

| From                      | To                             | Via                                                    | Status  | Details                                                                                                         |
| ------------------------- | ------------------------------ | ------------------------------------------------------ | ------- | --------------------------------------------------------------------------------------------------------------- |
| `internal/cli/root.go`    | command files                  | Cobra wiring                                           | ✓ WIRED | Root wires `newStatusCommand`, `newLogsCommand`, `newDoctorCommand`, `newRecoverCommand`, and `newDownCommand`. |
| `internal/cli/status.go`  | `internal/ops/status.go`       | `ops.Classify`                                         | ✓ WIRED | CLI builds observed facts and delegates health classification.                                                  |
| `internal/ops/probes.go`  | `internal/remote/executor.go`  | `remote.Executor` / `remote.HostKeyProber`             | ✓ WIRED | Status probes use remote boundary and host-key prober before SSH commands.                                      |
| `internal/cli/doctor.go`  | `internal/ops/doctor.go`       | `BuildDoctorReport`                                    | ✓ WIRED | CLI collects status/deep checks and renders report findings.                                                    |
| `internal/cli/recover.go` | `internal/bootstrap/script.go` | `RenderRemoveConfigScript` / `RenderReconfigureScript` | ✓ WIRED | Re-registration uses fresh token env vars and script helpers.                                                   |
| `internal/cli/down.go`    | `internal/ops/cleanup.go`      | `ops.BuildCleanupPlan` / `ops.SafeRunnerPaths`         | ✓ WIRED | Cleanup builds a plan and validates exact removal paths.                                                        |
| `internal/cli/down.go`    | GitHub runner API              | `DeleteRunner`                                         | ✓ WIRED | Stale runner records are deleted by saved ID or safe fallback.                                                  |
| `internal/cli/down.go`    | state store                    | `UpdateRepository` / `RemoveRepository`                | ✓ WIRED | Complete cleanup removes state; partial cleanup stores checkpoints.                                             |

**Wiring:** 8/8 connections verified

## Requirements Coverage

| Requirement | Status      | Evidence                                                                                                                 |
| ----------- | ----------- | ------------------------------------------------------------------------------------------------------------------------ |
| REL-01      | ✓ SATISFIED | `runnerkit status` tests cover repo, inference, `--all`, JSON, labels, host-key mismatch, and read-only counters.        |
| REL-02      | ✓ SATISFIED | `runnerkit logs` tests cover defaults, journal/diag command IDs, redaction, JSON, and docs.                              |
| REL-03      | ✓ SATISFIED | Doctor tests cover stable findings, pass visibility, JSON, redacted machine refs, and no mutation counters.              |
| REL-04      | ✓ SATISFIED | Recover tests cover dry-run, restart, reinstall, `--yes`, unsafe blocks, reregister token/state update.                  |
| GH-03       | ✓ SATISFIED | Recover/down use removal tokens, re-registration, `DeleteRunner`, and stale GitHub runner deletion tests.                |
| CLEAN-02    | ✓ SATISFIED | Down tests cover local-state-missing explicit ID deletion, SSH-unreachable partial GitHub cleanup, and ambiguity blocks. |
| CLEAN-03    | ✓ SATISFIED | Cleanup plan tests block unsafe/shared paths and down removes only exact install/work paths.                             |
| STATE-02    | ✓ SATISFIED | Redaction tests cover diagnostic logs, tokens, private keys, provider credentials, and machine refs.                     |

**Coverage:** 8/8 requirements satisfied

## Anti-Patterns Found

| File | Pattern | Severity | Impact                                                                      |
| ---- | ------- | -------- | --------------------------------------------------------------------------- |
| None | -       | -        | No placeholder/stub or forbidden `doctor --fix` / BYO `destroy` docs found. |

**Anti-patterns:** 0 found

## Human Verification Required

None - all phase-level requirements are verified programmatically against fake GitHub/remote adapters and state files.

A controlled disposable GitHub repo + Linux systemd host smoke remains recommended before public release, matching prior phase practice, but it is not a blocker for Phase 3 completion.

## Gaps Summary

**No gaps found.** Phase goal achieved. Ready to proceed.

## Automated Checks

- `go test ./...` — passed
- `go vet ./...` — passed
- Acceptance greps for `status`, `logs`, `doctor`, `recover`, `down`, `status.systemd.show`, `RenderReconfigureScript`, `remote_cleanup_pending`, cleanup docs, and forbidden `doctor --fix` / BYO `destroy` — passed
- Focused suites: `go test ./internal/cli -run 'TestStatus|TestLogs|TestDoctor|TestRecover|TestDown|TestDocs'` — passed via full suite

## Verification Metadata

**Verification approach:** Goal-backward from ROADMAP Phase 3 success criteria and plan must-haves.
**Must-haves source:** Phase 3 ROADMAP plus 03-01 through 03-04 PLAN frontmatter.
**Automated checks:** 4 suites passed, 0 failed.
**Human checks required:** 0 blocking.
**Total verification time:** 8 min

---

_Verified: 2026-04-30T02:08:00Z_
_Verifier: Pi executor_
