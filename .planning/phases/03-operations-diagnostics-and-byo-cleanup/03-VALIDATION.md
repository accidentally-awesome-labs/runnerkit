---
phase: 3
slug: operations-diagnostics-and-byo-cleanup
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-29
---

# Phase 3 - Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property               | Value                                        |
| ---------------------- | -------------------------------------------- |
| **Framework**          | Go standard `testing` package with `go test` |
| **Config file**        | `go.mod`; no separate test runner config     |
| **Quick run command**  | `go test ./...`                              |
| **Full suite command** | `go test ./... && go vet ./...`              |
| **Estimated runtime**  | ~15-45 seconds locally                       |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...` plus any task-specific focused command in the plan.
- **After every plan wave:** Run `go test ./... && go vet ./...`.
- **Before `/gsd-verify-work`:** Run `go test ./... && go vet ./...` and docs greps for `runnerkit status`, `runnerkit logs`, `runnerkit doctor`, `runnerkit recover`, and `runnerkit down` once docs tasks land.
- **Max feedback latency:** one task; no three consecutive Phase 3 tasks may rely only on manual checks.

---

## Per-Task Verification Map

| Task ID  | Plan | Wave | Requirement               | Test Type     | Automated Command                                                                                       | File Exists | Status     |
| -------- | ---- | ---- | ------------------------- | ------------- | ------------------------------------------------------------------------------------------------------- | ----------- | ---------- |
| 03-01-01 | 01   | 1    | REL-01                    | unit/scaffold | `go test ./internal/testsupport ./internal/state ./...`                                                 | ❌ W0       | ⬜ pending |
| 03-01-02 | 01   | 1    | REL-01                    | unit/cli      | `go test ./internal/cli ./internal/state ./internal/github ./internal/remote ./internal/ui ./...`       | ❌ W0       | ⬜ pending |
| 03-02-01 | 02   | 2    | REL-02, REL-03            | unit/cli      | `go test ./internal/cli ./internal/redact ./internal/preflight ./internal/remote ./internal/ui ./...`   | ❌ W0       | ⬜ pending |
| 03-03-01 | 03   | 3    | REL-04, GH-03             | unit/workflow | `go test ./internal/cli ./internal/github ./internal/state ./internal/workflow ./internal/redact ./...` | ❌ W0       | ⬜ pending |
| 03-04-01 | 04   | 4    | GH-03, CLEAN-02, CLEAN-03 | unit/workflow | `go test ./internal/cli ./internal/state ./internal/github ./internal/remote ./internal/workflow ./...` | ❌ W0       | ⬜ pending |

_Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky_

---

## Wave 0 Requirements

- [ ] Extend `internal/testsupport` with a reusable fake GitHub service satisfying `internal/cli.GitHubService`, including token creation, runner listing, deletion, call recording, injected errors, duplicate candidates, and mutable status/labels/busy fields.
- [ ] Add or centralize a reusable remote fake for scripted command results covering fast probes, `systemctl show`, `journalctl`, `_diag` reads, path checks, restart/reinstall, `config.sh remove`, re-registration, and cleanup commands.
- [ ] Add state fixture builders for healthy BYO runner, GitHub offline, busy runner, label drift, SSH unreachable, host-key mismatch, missing GitHub runner, missing service, partial cleanup pending, and local-state-missing stale GitHub cases.
- [ ] Add output assertion helpers for human text, JSON decoding, stable finding IDs, `redactions_applied: true`, and absence of raw tokens/private keys/provider-like secrets.
- [ ] Add state store tests for `ListRepositories`, `UpdateRepository`, `RemoveRepository`, and partial cleanup/recovery checkpoint persistence before commands depend on those helpers.

---

## Manual-Only Verifications

| Behavior                                | Requirement               | Why Manual                                                                   | Test Instructions                                                                                                                                            |
| --------------------------------------- | ------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Real GitHub runner UI status agreement  | REL-01                    | Requires disposable GitHub repo and registered self-hosted runner            | Run `runnerkit status --repo owner/name` and compare online/busy/labels with GitHub Actions runner UI.                                                       |
| Real SSH/systemd/journal log collection | REL-02, REL-03            | Requires disposable Linux host with systemd journal and runner `_diag` files | Run `runnerkit logs --repo owner/name --lines 50` and `runnerkit doctor --repo owner/name`; confirm output avoids manual SSH spelunking and redacts secrets. |
| Real recovery of stopped service        | REL-04, GH-03             | Requires controlled service stop on disposable host                          | Stop the runner service, run `runnerkit recover --repo owner/name --yes`, then confirm GitHub reports runner online.                                         |
| Real BYO cleanup side effects           | GH-03, CLEAN-02, CLEAN-03 | Requires disposable BYO install to safely delete files/services              | Run `runnerkit down --repo owner/name --yes`; confirm runner is deregistered and only recorded runner-specific install/work/service artifacts are removed.   |

---

## Validation Sign-Off

- [x] All implementation plans must include automated verification or depend on Wave 0 scaffolding.
- [x] Sampling continuity: no 3 consecutive tasks without automated verification.
- [x] Wave 0 covers missing reusable fakes/fixtures/assertion helpers.
- [x] No watch-mode flags.
- [x] Feedback latency target defined.
- [x] `nyquist_compliant: true` set in frontmatter.

**Approval:** pending
