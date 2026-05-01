---
phase: 5
slug: scoped-ephemeral-mode-and-safety-profiles
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-05-01
---

# Phase 5 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property               | Value                                                                                         |
| ---------------------- | --------------------------------------------------------------------------------------------- |
| **Framework**          | Go `testing`                                                                                  |
| **Config file**        | `go.mod`                                                                                      |
| **Quick run command**  | `go test ./internal/cli/... ./internal/bootstrap/... ./internal/ops/... ./internal/state/...` |
| **Full suite command** | `go test ./...`                                                                               |
| **Estimated runtime**  | ~90 seconds                                                                                   |

---

## Sampling Rate

- **After every task commit:** Run the task's focused package command from the map below.
- **After every plan wave:** Run `go test ./...`.
- **Before `/gsd-verify-work`:** Full suite plus docs greps must be green.
- **Max feedback latency:** 90 seconds focused; 180 seconds full suite on slower machines.

---

## Per-Task Verification Map

| Task ID  | Plan | Wave | Requirement   | Test Type            | Automated Command                                                                                                                            | File Exists | Status     |
| -------- | ---- | ---- | ------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ---------- |
| 05-01-01 | 01   | 1    | RUN-04        | CLI/unit             | `go test ./internal/cli/... ./internal/github/... ./internal/labels/...`                                                                     | ✅          | ⬜ pending |
| 05-01-02 | 01   | 1    | RUN-02/RUN-04 | CLI/unit             | `go test ./internal/cli/... ./internal/labels/... ./internal/state/...`                                                                      | ✅          | ⬜ pending |
| 05-01-03 | 01   | 1    | RUN-04        | CLI/unit             | `go test ./internal/cli/... ./internal/github/...`                                                                                           | ✅          | ⬜ pending |
| 05-02-01 | 02   | 2    | RUN-02        | bootstrap/unit       | `go test ./internal/bootstrap/... ./internal/state/...`                                                                                      | ✅          | ⬜ pending |
| 05-02-02 | 02   | 2    | RUN-02        | CLI/integration fake | `go test ./internal/cli/... ./internal/bootstrap/... ./internal/provider/...`                                                                | ✅          | ⬜ pending |
| 05-02-03 | 02   | 2    | RUN-02        | ops/unit             | `go test ./internal/ops/... ./internal/cli/... ./internal/state/...`                                                                         | ✅          | ⬜ pending |
| 05-03-01 | 03   | 3    | DOC-03/RUN-04 | docs/grep            | `go test ./internal/cli/... && grep -R "runnerkit up --repo owner/name --mode ephemeral --cloud hetzner" README.md docs`                     | ✅          | ⬜ pending |
| 05-03-02 | 03   | 3    | RUN-02/RUN-04 | CLI/integration fake | `go test ./internal/cli/... ./internal/bootstrap/... ./internal/ops/...`                                                                     | ✅          | ⬜ pending |
| 05-03-03 | 03   | 3    | DOC-03        | full regression      | `go test ./... && grep -R "persistent self-hosted runners" README.md docs && grep -R "Ephemeral mode is not a fleet manager" README.md docs` | ✅          | ⬜ pending |

_Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky_

---

## Wave 0 Requirements

Existing infrastructure covers the phase requirements, but the first implementation tasks should introduce these test fixtures before production behavior:

- [ ] `internal/cli/up_test.go` / `internal/cli/up_byo_test.go` / `internal/cli/up_cloud_test.go` - mode selection, public/fork safety, and persistent compatibility assertions.
- [ ] `internal/bootstrap/script_test.go` - ephemeral `config.sh --ephemeral`, one-shot service, finalizer, and TTL script assertions.
- [ ] `internal/ops/status_test.go` / `internal/ops/logs_test.go` / `internal/ops/doctor_test.go` - mode-aware ephemeral completed/expired/log-preserved states.
- [ ] `internal/cli/docs_test.go` - safety guide and README command/copy assertions.

---

## Manual-Only Verifications

| Behavior                               | Requirement            | Why Manual                                                                                       | Test Instructions                                                                                                                                                                                                                                                                                                                                                                                                  |
| -------------------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Live cloud ephemeral one-job lifecycle | RUN-02, RUN-04, DOC-03 | Requires real GitHub repo, real Hetzner credentials, queued workflow job, and billable resources | In a private test repo with `HCLOUD_TOKEN`, run `runnerkit up --repo owner/private-repo --mode ephemeral --cloud hetzner`, trigger a workflow on the printed ephemeral labels, verify GitHub auto-deregisters after one job, run `runnerkit logs --repo owner/private-repo --since 30m`, then run `runnerkit destroy --repo owner/private-repo` and verify no RunnerKit-created Hetzner resources remain billable. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies.
- [x] Sampling continuity: no 3 consecutive tasks without automated verify.
- [x] Wave 0 covers all MISSING references.
- [x] No watch-mode flags.
- [x] Feedback latency target documented.
- [x] `nyquist_compliant: true` set in frontmatter.

**Approval:** pending
