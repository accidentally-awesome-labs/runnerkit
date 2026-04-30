---
phase: 4
slug: recommended-cloud-path-and-billable-cleanup
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-30
---

# Phase 4 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property               | Value                                                                                        |
| ---------------------- | -------------------------------------------------------------------------------------------- |
| **Framework**          | Go `testing`                                                                                 |
| **Config file**        | `go.mod`                                                                                     |
| **Quick run command**  | `go test ./internal/provider/... ./internal/cli/... ./internal/ops/... ./internal/state/...` |
| **Full suite command** | `go test ./...`                                                                              |
| **Estimated runtime**  | ~30 seconds                                                                                  |

---

## Sampling Rate

- **After every task commit:** Run the task's focused package command from the map below.
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green.
- **Max feedback latency:** 30 seconds for focused tests; 90 seconds for the full suite.

---

## Per-Task Verification Map

| Task ID  | Plan | Wave | Requirement          | Test Type            | Automated Command                                                                                                                                                                 | File Exists | Status     |
| -------- | ---- | ---- | -------------------- | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ---------- |
| 04-01-01 | 01   | 1    | MACH-03              | unit                 | `go test ./internal/provider/...`                                                                                                                                                 | ❌ W0       | ⬜ pending |
| 04-01-02 | 01   | 1    | MACH-03              | CLI/unit             | `go test ./internal/cli/... ./internal/provider/...`                                                                                                                              | ✅          | ⬜ pending |
| 04-01-03 | 01   | 1    | CLEAN-01             | CLI/golden           | `go test ./internal/cli/...`                                                                                                                                                      | ✅          | ⬜ pending |
| 04-02-01 | 02   | 2    | MACH-03              | unit                 | `go test ./internal/provider/... ./internal/state/...`                                                                                                                            | ❌ W0       | ⬜ pending |
| 04-02-02 | 02   | 2    | MACH-05              | unit                 | `go test ./internal/provider/... ./internal/state/...`                                                                                                                            | ✅          | ⬜ pending |
| 04-02-03 | 02   | 2    | MACH-03              | CLI/integration fake | `go test ./internal/cli/... ./internal/provider/...`                                                                                                                              | ✅          | ⬜ pending |
| 04-03-01 | 03   | 3    | MACH-04              | CLI/integration fake | `go test ./internal/cli/... ./internal/bootstrap/...`                                                                                                                             | ✅          | ⬜ pending |
| 04-03-02 | 03   | 3    | MACH-05              | unit                 | `go test ./internal/ops/... ./internal/cli/...`                                                                                                                                   | ✅          | ⬜ pending |
| 04-03-03 | 03   | 3    | REL-01/REL-02/REL-03 | unit                 | `go test ./internal/ops/... ./internal/cli/...`                                                                                                                                   | ✅          | ⬜ pending |
| 04-04-01 | 04   | 4    | CLEAN-01/CLEAN-04    | unit                 | `go test ./internal/ops/... ./internal/provider/...`                                                                                                                              | ✅          | ⬜ pending |
| 04-04-02 | 04   | 4    | CLEAN-04             | CLI/integration fake | `go test ./internal/cli/... ./internal/provider/...`                                                                                                                              | ✅          | ⬜ pending |
| 04-04-03 | 04   | 4    | DOC-02               | docs/grep            | `grep -R "runnerkit up --repo owner/name --cloud hetzner" README.md docs/cloud-quickstart.md && grep -R "runnerkit destroy --repo owner/name" README.md docs/cloud-quickstart.md` | ❌ W0       | ⬜ pending |

_Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky_

---

## Wave 0 Requirements

- [ ] `internal/provider/` - provider interfaces, Hetzner profile constants, fake client scaffolding, and package tests for MACH-03/CLEAN-04.
- [ ] `internal/provider/hetzner/` - adapter skeleton that can be tested without live credentials.
- [ ] `docs/cloud-quickstart.md` - documentation file stub for DOC-02.

---

## Manual-Only Verifications

| Behavior                                        | Requirement                | Why Manual                                                        | Test Instructions                                                                                                                                                                                                                                                                    |
| ----------------------------------------------- | -------------------------- | ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Live Hetzner cloud provisioning and destruction | MACH-03, MACH-04, CLEAN-04 | Requires real provider credentials and creates billable resources | In a controlled private repo with `HCLOUD_TOKEN`, run `runnerkit up --repo owner/private-repo --cloud hetzner`, confirm runner online, then run `runnerkit destroy --repo owner/private-repo` and verify no RunnerKit-created server/firewall/SSH key/primary IP remains in Hetzner. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
