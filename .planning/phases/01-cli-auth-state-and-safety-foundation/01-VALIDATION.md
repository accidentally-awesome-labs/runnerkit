---
phase: 1
slug: cli-auth-state-and-safety-foundation
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-29
---

# Phase 1 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property               | Value                                                |
| ---------------------- | ---------------------------------------------------- |
| **Framework**          | Go `testing` package + `httptest` + golden fixtures  |
| **Config file**        | none - Wave 0 initializes `go.mod` and test packages |
| **Quick run command**  | `go test ./...`                                      |
| **Full suite command** | `go test ./... && go vet ./...`                      |
| **Estimated runtime**  | ~60 seconds cold, ~15 seconds warm                   |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...` once `go.mod` exists; before `go.mod`, run `test -f go.mod && test -f cmd/runnerkit/main.go`.
- **After every plan wave:** Run `go test ./... && go vet ./...`.
- **Before `/gsd-verify-work`:** Full suite must be green.
- **Max feedback latency:** 60 seconds.

---

## Per-Task Verification Map

| Task ID | Plan  | Wave | Requirement                        | Test Type                  | Automated Command                 | File Exists | Status     |
| ------- | ----- | ---- | ---------------------------------- | -------------------------- | --------------------------------- | ----------- | ---------- |
| 01-01   | 01-01 | 1    | CLI-01                             | build/unit/golden          | `go test ./...`                   | ❌ W0       | ⬜ pending |
| 01-02   | 01-01 | 1    | CLI-02, STATE-02                   | unit/golden/no-TTY         | `go test ./...`                   | ❌ W0       | ⬜ pending |
| 01-03   | 01-01 | 1    | CLI-01, CLI-02, STATE-02           | build/unit/golden          | `go test ./... && go vet ./...`   | ❌ W0       | ⬜ pending |
| 01-04   | 01-02 | 2    | GH-01                              | unit/httptest/fake command | `go test ./internal/github ./...` | ❌ W0       | ⬜ pending |
| 01-05   | 01-02 | 2    | GH-01, STATE-02                    | unit/httptest/redaction    | `go test ./... && go vet ./...`   | ❌ W0       | ⬜ pending |
| 01-06   | 01-03 | 3    | STATE-01                           | unit/filesystem/migration  | `go test ./internal/state ./...`  | ❌ W0       | ⬜ pending |
| 01-07   | 01-03 | 3    | STATE-01, STATE-02                 | unit/golden/filesystem     | `go test ./...`                   | ❌ W0       | ⬜ pending |
| 01-08   | 01-03 | 3    | CLI-01, CLI-02, GH-01, STATE-01/02 | end-to-end fake adapters   | `go test ./... && go vet ./...`   | ❌ W0       | ⬜ pending |

_Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky_

---

## Wave 0 Requirements

- [ ] `go.mod` - Go module initialized for RunnerKit.
- [ ] `cmd/runnerkit/main.go` - runnable binary entrypoint.
- [ ] `internal/testsupport/` - fake prompt, fake command runner, golden output helpers, temp state filesystem helpers.
- [ ] `internal/redact/` - central redaction package with table tests before GitHub token work.
- [ ] `testdata/` fixtures under package-specific test directories for wizard output, JSON output, GitHub API fixtures, and fine-grained token instructions.

---

## Manual-Only Verifications

| Behavior                     | Requirement | Why Manual                                                                           | Test Instructions                                                                                                 |
| ---------------------------- | ----------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| Interactive terminal feel    | CLI-02      | Prompt readability, Ctrl-C behavior, and 80-column wrapping need one real TTY check. | Run `go run ./cmd/runnerkit up` in a terminal; verify six-step order, safe defaults, and Ctrl-C copy.             |
| Real GitHub permission smoke | GH-01       | Official permission behavior may differ from fixtures/docs.                          | Use a controlled test repo and least-privilege credential; verify denied vs allowed paths without logging tokens. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies.
- [x] Sampling continuity: no 3 consecutive tasks without automated verify.
- [x] Wave 0 covers all MISSING references.
- [x] No watch-mode flags.
- [x] Feedback latency < 60s.
- [x] `nyquist_compliant: true` set in frontmatter.

**Approval:** approved 2026-04-29
