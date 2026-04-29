---
phase: 2
slug: byo-persistent-runner-happy-path
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-29
---

# Phase 2 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property               | Value           |
| ---------------------- | --------------- |
| **Framework**          | Go `testing`    |
| **Config file**        | `go.mod`        |
| **Quick run command**  | `go test ./...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime**  | ~5 seconds      |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID  | Plan | Wave | Requirement    | Test Type                   | Automated Command | File Exists | Status     |
| -------- | ---- | ---- | -------------- | --------------------------- | ----------------- | ----------- | ---------- |
| 02-01-01 | 01   | 1    | MACH-01        | unit/integration            | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-01-02 | 01   | 1    | MACH-01        | unit                        | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-01-03 | 01   | 1    | MACH-01        | CLI integration             | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-02-01 | 02   | 2    | MACH-02        | unit                        | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-02-02 | 02   | 2    | MACH-02        | unit/integration            | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-02-03 | 02   | 2    | MACH-02        | CLI integration             | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-03-01 | 03   | 3    | GH-02          | unit                        | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-03-02 | 03   | 3    | GH-04, GH-05   | unit/CLI integration        | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-03-03 | 03   | 3    | CLI-04, RUN-01 | CLI integration             | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-04-01 | 04   | 4    | RUN-03         | CLI integration             | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-04-02 | 04   | 4    | DOC-01         | docs grep + go test         | `go test ./...`   | ✅ existing | ⬜ pending |
| 02-04-03 | 04   | 4    | CLI-03         | fake e2e + manual smoke doc | `go test ./...`   | ✅ existing | ⬜ pending |

_Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky_

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements:

- [x] `go.mod` - Go module and Cobra CLI dependency already exist.
- [x] `internal/cli/*_test.go` - CLI integration test pattern exists.
- [x] `internal/github/*_test.go` - fake GitHub/API test pattern exists.
- [x] `internal/state/*_test.go` - state persistence/migration test pattern exists.
- [x] `internal/labels/*_test.go` - label/snippet test pattern exists.
- [x] `internal/redact/*_test.go` - redaction test pattern exists.
- [x] `go test ./...` - current baseline is green.

---

## Manual-Only Verifications

| Behavior                                           | Requirement                     | Why Manual                                                                    | Test Instructions                                                                                                                                                                            |
| -------------------------------------------------- | ------------------------------- | ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Real BYO Linux host registration and online status | CLI-03, GH-02, MACH-01, MACH-02 | Requires disposable Linux systemd host and real GitHub repo token permissions | Run `runnerkit up --repo owner/repo --host user@host --yes`; confirm GitHub runner online, service active as `runnerkit-runner`, labels match snippet, and no token appears in state/output. |
| Copy-pasted workflow actually routes to the runner | GH-04, GH-05                    | Requires a real repository workflow run                                       | Paste the completion `runs-on: [...]` array into a minimal workflow and confirm the job is picked up by the RunnerKit runner.                                                                |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-29
