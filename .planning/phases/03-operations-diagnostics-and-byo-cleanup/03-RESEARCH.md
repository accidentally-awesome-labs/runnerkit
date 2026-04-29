# Phase 3 Research: Operations, Diagnostics, and BYO Cleanup

**Phase:** 03 - Operations, Diagnostics, and BYO Cleanup  
**Researched:** 2026-04-29  
**Status:** Ready for planning

## Verdict

Phase 3 should be planned as a shared BYO operations/reconciliation layer first, then thin CLI commands on top: `status` establishes the common observed-facts and health model; `logs`/`doctor` reuse it for deeper evidence; recovery and `down` reuse the same model plus explicit, checkpointed mutation plans.

## Scope to preserve

In scope for this phase:

- RunnerKit-managed **BYO persistent** repository runners created by Phase 2.
- Read-only status across local state, GitHub runner inventory, SSH reachability, systemd service health, labels, and saved machine/path facts.
- Redacted `logs` and `doctor` output that avoids manual SSH spelunking for common failures.
- Guided persistent-runner restart/re-registration/recovery for stopped/offline cases.
- `runnerkit down` for safe BYO runner cleanup and stale GitHub runner deregistration, including partial state/missing SSH cases.

Out of scope for this phase:

- Cloud provisioning/destruction and billable-resource cleanup.
- Ephemeral runners.
- Organization-level runner management.
- Automatic workflow YAML edits or read-only workflow scanning.
- Automatic `doctor --fix` repair.
- Import/adopt of arbitrary pre-existing manual runners.

## Requirements mapping

| Requirement | Planning implication |
| --- | --- |
| GH-03 | Need fresh removal-token support for host-side `config.sh remove` plus GitHub API deletion for stale/missing-host records. Existing adapter already exposes both. |
| REL-01 | Implement `runnerkit status` with local state + GitHub + SSH + systemd + labels; status must be read-only and fast. |
| REL-02 | Implement `runnerkit logs` over systemd journal and runner `_diag` files, with redaction and bounded output. |
| REL-03 | Implement `runnerkit doctor` findings with severity, evidence, and exact remediation commands; do not auto-fix. |
| REL-04 | Implement guided recovery/restart/re-register flows that mutate only after plan + confirmation/`--yes`. |
| CLEAN-02 | `down` must deregister stale GitHub runner records even when SSH/local state is incomplete. |
| CLEAN-03 | `down` must remove only recorded RunnerKit-managed BYO runner-specific service/files and avoid unrelated user data. |

## Existing implementation assets

- `internal/cli/root.go` has the injectable Cobra command tree and should add `status`, `logs`, `doctor`, recovery, and `down` as thin command handlers.
- `internal/cli/up.go` already defines the `GitHubService` interface with `CreateRemovalToken`, `ListRunners`, and `DeleteRunner`, plus helpers worth reusing/refactoring: repo resolution, host-key verification logic, `waitForRunnerOnline`, label comparison, BYO state construction, and state-save confirmation patterns.
- `internal/github/runners.go` exposes runner `ID`, `Name`, `OS`, `Status`, `Busy`, and `Labels`, enough for status source facts, stale detection, and deletion.
- `internal/state/schema.go` already stores repo scope, runner name/labels/snippet/mode/OS/arch, BYO host/user/port/key/fingerprint/install path/work dir/service name, provider kind, GitHub runner ID, and cleanup metadata.
- `internal/state/store.go` supports load/save/get/upsert with atomic, secret-free JSON persistence. Phase 3 needs remove/update helpers and partial-cleanup state, but the base is sound.
- `internal/remote/executor.go` + `internal/remote/system.go` isolate SSH execution. Phase 3 can layer service/log/remove helpers over `Run` without immediately changing the interface.
- `internal/preflight/checks.go` has useful deep checks for `doctor`; do not call full preflight from default `status` because it is too broad/slow for D-05.
- `internal/bootstrap/install.go` and `internal/bootstrap/script.go` define installed service/user/path assumptions and can be split for recovery/cleanup scripts.
- `internal/workflow/plan.go` gives plan/checkpoint types suitable for cleanup/recovery plans.
- `internal/ui/output.go` centralizes human/JSON output and redaction flags.

## Key planning findings

### 1. Build a shared operations model before individual commands

Plan 03-01 should create the common model used by every Phase 3 command. Suggested package names: `internal/ops`, `internal/reconcile`, or `internal/health`.

Core types to plan:

- `TargetScope`: one repo, inferred current repo, all local repos, or explicit stale-GitHub target.
- `ObservedRunner`: local state facts, GitHub runner facts, SSH facts, systemd facts, label facts, and non-fatal collection errors.
- `HealthState`: e.g. `ready`, `busy`, `needs_attention`, `broken`, `unknown`.
- `Reason` / `Finding`: stable IDs, severity, source, evidence, and remediation.
- `NextAction`: command string plus why it is safest.
- `ArtifactPlan`: cleanup/recovery artifact, observed state, action, confirmation default, result/checkpoint.

This avoids four commands each inventing their own runner lookup, label comparison, SSH probing, state handling, and JSON shape.

### 2. `status` needs a fast probe path, not full preflight

`remote.SystemExecutor.Probe` currently performs many checks: `uname`, `/etc/os-release`, systemd discovery, required tools, sudo, disk, time sync, and host-key scan. That is appropriate for `up`/`doctor`, but `status` should only collect fast health facts:

- local state exists and has runner/machine metadata;
- GitHub runner inventory/listing for the repo;
- SSH reachable and host key matches saved fingerprint;
- systemd unit load/active/substate for the recorded service;
- label set from GitHub vs saved expected labels;
- saved install/work paths/snippet.

Add focused helpers over `remote.Executor.Run`, for example:

- `ProbeSSHReachable(ctx, target)` or a bounded `true` command;
- `ObservedHostKey(ctx, target)` or reuse a trimmed host-key scan;
- `ServiceStatus(ctx, target, serviceName)` using `systemctl show <unit> --property=LoadState,ActiveState,SubState,UnitFileState,ExecMainStatus --no-pager`;
- optional path existence checks only if still fast.

### 3. Status health should be derived but transparent

Suggested minimum derived states:

| State | Typical condition | Next action |
| --- | --- | --- |
| `ready` | GitHub online, not busy, SSH reachable, service active, labels match | None or show workflow snippet. |
| `busy` | Same as ready but GitHub `busy=true` | Wait or check GitHub Actions jobs. |
| `needs_attention` | Recoverable drift: GitHub offline while SSH works, service inactive, label drift, stale saved GitHub ID fallback by name | `runnerkit doctor`, `runnerkit logs`, or `runnerkit recover`. |
| `broken` | Host-key mismatch, service failed/missing, GitHub runner missing while remote install exists, duplicate candidates, unsafe cleanup ambiguity | `doctor`, explicit recovery, or `down` depending on source. |
| `unknown` | Cannot collect enough evidence: missing state, GitHub auth/API failure, SSH unreachable with inconclusive GitHub state | Fix auth/SSH, run `down` for stale GitHub only, or provide `--repo`. |

Human output should include top-line health, compact source matrix, saved `runs-on` snippet, label drift warning, and one safest next command. JSON should include the same derived state plus raw source facts and errors.

Suggested JSON shape:

```json
{
  "ok": true,
  "command": "status",
  "scope": "repo",
  "state_path": ".../state.json",
  "repo": "owner/name",
  "health": {
    "state": "needs_attention",
    "summary": "GitHub reports the runner offline while SSH is reachable and the service is failed.",
    "reasons": [{"id":"service_failed","severity":"error","source":"systemd"}],
    "next_actions": [{"command":"runnerkit doctor --repo owner/name","why":"Inspect service logs before recovery."}]
  },
  "runner": {"name":"runnerkit-owner-name-local","labels":["self-hosted","runnerkit"],"workflow_snippet":"runs-on: [...]"},
  "sources": {
    "state": {"present":true},
    "github": {"found":true,"id":123,"status":"offline","busy":false,"labels":[...]},
    "ssh": {"reachable":true,"host_key":"matched"},
    "systemd": {"service":"actions.runner...service","active_state":"failed","sub_state":"failed"},
    "labels": {"match":false,"missing":["persistent"],"extra":[]}
  },
  "redactions_applied": true
}
```

### 4. Repo targeting must be shared and command-specific

Decisions require:

- `runnerkit status` defaults to the current git repository when inferrable, matching `up` repo detection.
- `runnerkit status --all` shows all locally managed repositories.
- `--repo owner/name` remains explicit.

Current `resolveUpRepo` is setup-oriented: it can prompt for confirmation and is coupled to `up` error copy. Plan a shared repo resolver with modes:

- setup mode: current `up` behavior with confirmation;
- read-only mode: infer current repo without mutation/prompt, or return actionable `input_required` if no repo/all scope;
- cleanup mode: allow explicit stale-GitHub targeting even if local state is missing.

### 5. State storage needs Phase 3 helpers

Add to `internal/state`:

- `ListRepositories()` or use `Load()` through a helper for `status --all`.
- `RemoveRepository(fullName)` for successful cleanup.
- `UpdateRepository(repoState)` for recovery changing GitHub runner ID, cleanup notes, or partial checkpoints.
- Optional append-only cleanup/recovery notes with timestamps.

Current `CleanupMetadata.ManagedPaths` is risky for deletion: Phase 2 stored `[]string{installPath, "/var/lib/runnerkit"}`. The latter is a shared parent and should **not** be blindly deleted by `down --yes`. For safe cleanup, prefer:

- remove `Machine.InstallPath` exactly;
- remove `Machine.WorkDir` exactly;
- remove the recorded service only;
- remove empty parent directories only when provably empty and RunnerKit-owned;
- update future state to include the work dir in managed paths and avoid broad shared paths.

Partial cleanup state is required: do not remove local state until all selected/required artifacts have succeeded or been intentionally skipped. If SSH is unreachable but GitHub deletion succeeds, keep/update state with `remote_cleanup_pending` and explicit notes.

### 6. Logs must be bounded, sectioned, and redacted by default

`runnerkit logs` should be read-only and default to the current repo. Suggested flags:

- `--repo owner/name`, `--all` optional only if useful;
- `--since 1h` and/or `--lines 200`;
- `--service` / `--runner` selectors if planner wants granular output;
- `--json` for tests and automation.

Sources to collect for BYO persistent runners:

- `journalctl -u <recorded-service-name> --since ... -n ... --no-pager`;
- runner `_diag` files under `<install_path>/_diag`, newest `Runner_*.log` and `Worker_*.log` files;
- limited RunnerKit bootstrap/service facts if present (currently there is no durable bootstrap log file, so do not promise one unless implementation adds it);
- local state path and runner metadata without secrets.

Important redaction note: Phase 2 mostly avoids printing remote command output. Phase 3 will print logs, so route every collected string through `ui.Renderer`/`redact.Redactor` and avoid commands that dump environment variables. Include a warning that redaction is best-effort for workflow-produced secrets and users should review logs before sharing.

### 7. `doctor` should share status observations but go deeper

`doctor` should be read-only in Phase 3 and produce actionable findings, not automatic fixes.

Recommended finding categories:

- Local state: schema readable, repo entry present, state path permissions, cleanup pending notes.
- GitHub auth/API: can list runners, runner ID/name exists, duplicate RunnerKit candidates, status/busy, label drift.
- SSH identity/reachability: host reachable, host key match/mismatch.
- Service: unit loaded, active/substate, user is not root if inspectable, restart policy if available.
- Installation: install path exists, `config.sh`/`run.sh`/`.runner` present, work dir exists/owned by service user.
- Host health: reuse `preflight.Run` or selected checks for disk, tools, time sync, outbound GitHub/API network, systemd, sudo.
- Persistent-runner hygiene: old `_work` size, disk pressure, Docker group/socket warning if visible.
- Logs: summarize recent service/runner errors and point to `runnerkit logs`.

Output contract should include stable finding IDs, severity (`pass`, `warning`, `error`), evidence, and remediation. Example next actions:

- `runnerkit logs --repo owner/name --since 30m`
- `runnerkit recover --repo owner/name --restart-service`
- `runnerkit recover --repo owner/name --reregister`
- `runnerkit down --repo owner/name`

### 8. Recovery should be guided, targeted, and distinct from `up`

The requirement is restart/recover through documented or guided CLI steps. A minimal robust command can be `runnerkit recover` with planner-chosen flags/submodes. Avoid telling users to blindly rerun `up` for Phase 3 recovery.

Recovery cases to plan:

1. **Stopped/inactive/failed service, GitHub record exists:** show plan, confirm, run `systemctl restart <service>` or recorded install-path `svc.sh start`, then wait for GitHub online with expected labels.
2. **Service missing but install path exists:** reinstall/start service using recorded `svc.sh install runnerkit-runner`, then verify.
3. **GitHub runner record missing/stale but SSH/install exists:** stop service, use fresh removal token with `config.sh remove` when possible, request fresh registration token, run `config.sh --unattended --replace` with saved labels/name/work dir, start service, wait online, update `Cleanup.GitHubRunnerID`.
4. **SSH unreachable:** do not mutate; report SSH remediation or suggest `runnerkit down` for stale GitHub deregistration.
5. **Host-key mismatch:** fail closed; do not recover until user verifies host identity.

All registration/removal tokens remain just-in-time, registered with the redactor, and never persisted. Recovery should use the saved runner name, labels, install path, work dir, service user, and service name from state.

### 9. `down` is a reconciliation problem, not just a delete script

`runnerkit down` should build and show an artifact-by-artifact plan before mutation. In interactive mode, ask before each artifact. With `--yes`, apply the safe default plan without per-artifact prompts.

Recommended artifacts:

| Artifact | Safe action |
| --- | --- |
| GitHub runner record | Prefer exact saved `Cleanup.GitHubRunnerID`; fallback to saved runner name + RunnerKit repo label. Delete stale record via GitHub API if host removal cannot do it. |
| Host-side runner registration | If SSH/install exists, request fresh removal token and run `config.sh remove --token` from the recorded install path. Treat already-removed as skipped/done. |
| systemd service | Stop/disable/uninstall only the recorded `Machine.ServiceName` / install-path `svc.sh` service. |
| install path | Remove exactly recorded `Machine.InstallPath` if it is an expected RunnerKit runner path. |
| work dir | Remove exactly recorded `Machine.WorkDir`; avoid deleting parent `/var/lib/runnerkit` unless empty/exclusive. |
| local state | Remove only after successful selected cleanup, otherwise update pending notes/checkpoints. |

Partial-state behavior:

- SSH unreachable + GitHub stale: allow GitHub API deletion, keep remote cleanup pending in state, print what may remain on the BYO host.
- GitHub missing + SSH reachable: remove host service/files and state; record GitHub artifact as already absent.
- Local state missing + GitHub stale: support an explicit stale-GitHub flow. Safest shape is `runnerkit down --repo owner/name --github-runner-id <id> --yes`, or interactive listing of candidates with RunnerKit labels. Non-interactive deletion without exact ID/name should be blocked.
- Missing service/path: mark artifact skipped/already absent; cleanup remains idempotent.
- GitHub auth failure: do not remove state as if complete; if remote cleanup was selected and succeeds, keep pending GitHub cleanup note.

### 10. Test fakes should be upgraded early

Current fakes are embedded in `internal/cli/*_test.go` and `internal/testsupport/github.go` only covers Phase 1 auth. Phase 3 needs richer reusable fakes:

- GitHub fake with runners, list/delete/create-token call recording, configurable errors, duplicate candidates, stale/missing IDs.
- Remote fake with per-command outputs/errors for service status, logs, path exists, remove/restart/re-register scripts.
- State fixture builders for healthy, offline, missing GitHub ID, missing SSH, host-key mismatch, partial cleanup.

These fakes will keep plan slices small and make JSON contracts testable.

## Plan-by-plan implementation guidance

### Plan 03-01: Multi-source status reconciliation

Deliverables to plan:

- Add read-only repo-scope resolution (`--repo`, inferred current repo, `--all`).
- Add state listing and state-not-found handling.
- Add operations model and health classifier.
- Add GitHub runner lookup by saved ID, fallback name, and label candidate detection.
- Add fast SSH/host-key/service status helpers.
- Add label drift comparison and saved `runs-on` snippet output.
- Add `runnerkit status` human + JSON contracts and tests.

Definition of done:

- Healthy runner renders `ready` plus source matrix and snippet.
- Busy runner renders `busy` not failure.
- GitHub offline + service failed points to `doctor`/recovery.
- Label drift is visible in human output and JSON.
- Host-key mismatch is read-only, fail-closed in health, and does not prompt.
- `status --all` lists all local repo states.
- No status path mutates GitHub, remote, or local state.

### Plan 03-02: Logs and doctor diagnostics

Deliverables to plan:

- `runnerkit logs` with bounded journal/runner `_diag` collection.
- Redacted sectioned human output and JSON.
- `runnerkit doctor` with stable findings, severity, evidence, remediation.
- Reuse status observations; add deeper preflight/host checks only in doctor.
- Redaction tests with fake GitHub tokens, registration/removal tokens, SSH private keys, provider-looking secrets, and synthetic runner log secrets.
- Update BYO quickstart troubleshooting to point to status/logs/doctor instead of manual SSH first.

Definition of done:

- Developer can inspect service/runner logs without manually SSHing.
- Doctor explains common states: offline, service stopped/failed, SSH unreachable, label drift, missing GitHub runner, disk/network/tool issue.
- JSON output is stable and redacted.
- No `doctor --fix` mutation is introduced.

### Plan 03-03: Guided persistent-runner recovery

Deliverables to plan:

- Command shape (recommended: `runnerkit recover`) and/or explicit `restart` alias if desired.
- Recovery plan builder from status/doctor findings.
- Service restart flow with confirmation/`--yes`.
- Service reinstall/start flow if unit missing but install path exists.
- Re-registration flow with fresh registration/removal tokens and saved labels/name/work dir.
- Update state with new GitHub runner ID after successful re-registration.
- Wait/verify using `waitForRunnerOnline` or a shared equivalent.
- Documentation snippets for common recovery cases.

Definition of done:

- Stopped/failed service can be restarted and verified online.
- Missing/stale GitHub registration can be re-registered without creating duplicates.
- Host-key mismatch and SSH unreachable do not attempt unsafe recovery.
- Tokens are redacted and not persisted.
- Recovery does not blindly run the whole `up` flow.

### Plan 03-04: BYO cleanup and stale deregistration

Deliverables to plan:

- `runnerkit down` with `--repo`, inferred repo, `--yes`, `--dry-run`, JSON, and explicit stale-GitHub targeting if local state is missing.
- Cleanup artifact plan and per-artifact interactive confirmations.
- Non-interactive safe default plan.
- GitHub runner deletion by saved ID/name/labels.
- Host-side removal token + `config.sh remove` when possible; API delete fallback for stale records.
- Service stop/uninstall and exact install/work path removal.
- State remove/update helpers and partial cleanup checkpoints.
- Docs update: cleanup command, stale GitHub flow, what BYO host data is/is not removed.

Definition of done:

- Reachable BYO cleanup removes only RunnerKit runner-specific artifacts and local state.
- SSH-unreachable stale GitHub cleanup deregisters GitHub and leaves remote cleanup pending in state/output.
- Missing GitHub runner does not block local BYO file/service removal.
- Missing service/path is idempotent skip, not a hard failure.
- `--yes` does not delete shared user/shared `/var/lib/runnerkit` data blindly.

## UX and command contract recommendations

- Keep `status`, `logs`, and `doctor` read-only.
- Use `down` for BYO runner removal; reserve `destroy` for future cloud resources.
- Mutating commands (`recover`, `down`) should support `--dry-run` and `--yes` where practical.
- All commands should support global `--json`; JSON must include `redactions_applied: true` through the existing renderer.
- Human output should always end with exact next commands when unhealthy or partially complete.
- Status should always show the saved workflow snippet and warn not to use `runs-on: self-hosted` alone.

## Risks and uncertainties

- **Remote output redaction risk:** Phase 3 prints logs for the first time. Existing redactor is good for known RunnerKit/GitHub/provider patterns, but arbitrary workflow secrets may appear in runner logs. Keep log collection bounded and warn before sharing.
- **`ManagedPaths` over-delete risk:** Existing state records `/var/lib/runnerkit` as managed. `down --yes` must not blindly remove it; use exact install/work paths and only remove shared parents if empty/proven safe.
- **Status latency risk:** Reusing full `remote.Probe` could make status slow and violate the fast-probe decision. Add focused status probes.
- **Service name semantics:** Use the saved `Machine.ServiceName` from state rather than recomputing. Official runner `svc.sh` naming can vary; state is the source of truth.
- **Local-state-missing cleanup ambiguity:** Without state, only delete GitHub runners that are explicit by ID/name or clearly selected interactively from RunnerKit-labeled candidates.
- **State schema evolution:** Optional fields can be added under schema `1`, but partial cleanup/recovery checkpoints should be planned carefully before implementation so failed `down` runs are resumable.
- **Auth failure handling:** Read-only commands can render partial/unknown status when GitHub auth fails; mutating cleanup/recovery must keep state if any required artifact could not be reconciled.

## Verification strategy

Run `go test ./...` after each plan. Add targeted tests for:

- Status health matrix: ready, busy, label drift, GitHub offline, service failed, SSH unreachable, host-key mismatch, missing state, GitHub auth/list failure.
- `status --all` inventory output and JSON list.
- Logs collection with redaction and bounded lines/sections.
- Doctor findings and remediation commands for common failures.
- Recovery service restart and re-registration paths, including fresh token creation and state update.
- Down dry-run, interactive per-artifact prompting, `--yes` safe defaults, partial cleanup checkpoints, stale GitHub with SSH unreachable, local-state-missing explicit runner deletion, and idempotent missing service/path behavior.
- State store remove/update helpers and redaction invariants.
- Docs greps for `status`, `logs`, `doctor`, `recover`, and `down` replacing manual SSH troubleshooting as first-line guidance.

## Evidence read

- Required phase context: `.planning/phases/03-operations-diagnostics-and-byo-cleanup/03-CONTEXT.md`
- Requirements/state: `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/ROADMAP.md`, `.planning/PROJECT.md`
- Prior phase docs: `.planning/phases/02-byo-persistent-runner-happy-path/02-CONTEXT.md`, `.planning/phases/02-byo-persistent-runner-happy-path/02-VERIFICATION.md`
- Research docs: `.planning/research/SUMMARY.md`, `.planning/research/ARCHITECTURE.md`, `.planning/research/FEATURES.md`, `.planning/research/PITFALLS.md`, `.planning/research/STACK.md`
- User docs: `docs/byo-quickstart.md`, `README.md`
- Code: `internal/cli/root.go`, `internal/cli/up.go`, `internal/cli/state.go`, `internal/github/runners.go`, `internal/github/service.go`, `internal/state/schema.go`, `internal/state/store.go`, `internal/remote/executor.go`, `internal/remote/system.go`, `internal/preflight/checks.go`, `internal/bootstrap/install.go`, `internal/bootstrap/script.go`, `internal/workflow/plan.go`, `internal/ui/output.go`, `internal/redact/redact.go`, existing tests and fakes.
- Validation command: `go test ./...` passed on 2026-04-29 during research.

## Validation Architecture

### Test infrastructure

| Property | Phase 3 value |
| --- | --- |
| Framework | Go standard `testing` package with `go test` |
| Test config | `go.mod` only; no separate test runner config or Makefile exists |
| Existing test roots | `internal/*/*_test.go`, command tests under `internal/cli`, adapter tests under `internal/github`, `internal/remote`, `internal/state`, `internal/redact`, `internal/ui`, and `internal/workflow` |
| Existing fakes | `internal/cli/root_test.go` has `fakePermittedGitHubService` and `fakeRemoteExecutor`; `internal/testsupport/github.go` has an auth/repo fake that should be extended for runner operations |
| Quick command | `go test ./...` |
| Full suite command | `go test ./... && go vet ./...` |
| Baseline observed | `go test ./... && go vet ./...` passed on 2026-04-29 before planning |

### Sampling cadence

- **After every task commit:** run the quick command `go test ./...` and include any task-specific focused command from that task's `<automated>` block.
- **After every plan wave:** run the full suite command `go test ./... && go vet ./...`.
- **Before verification:** run `go test ./... && go vet ./...` plus the Phase 3 docs/grep checks added by plans, such as greps for `runnerkit status`, `runnerkit logs`, `runnerkit doctor`, `runnerkit recover`, and `runnerkit down` in `README.md` and `docs/byo-quickstart.md` after docs are updated.
- **Sampling rule:** no three consecutive Phase 3 tasks may rely only on manual checks; every implementation task must add or update an automated Go test that fails before the behavior exists.

### Requirement-to-validation map

| Requirement | Automated validation required | Manual/controlled validation |
| --- | --- | --- |
| `GH-03` | Fake GitHub service tests prove fresh removal-token use for host-side `config.sh remove`, API `DeleteRunner` fallback for stale records, token redaction, and no persisted registration/removal tokens. | Disposable repo smoke confirms removed/recreated runner disappears from the GitHub Actions runners UI. |
| `REL-01` | `runnerkit status` CLI tests cover local state lookup, inferred `--repo`, `--all`, GitHub runner ID/name lookup, SSH reachability, host-key mismatch, systemd active/failed/missing states, label drift, saved `runs-on` snippet, human output, JSON output, and zero mutation calls. | Optional real BYO host smoke confirms status agrees with GitHub UI and `systemctl` for an online and stopped service. |
| `REL-02` | Remote fake tests cover bounded `journalctl` and runner `_diag` collection, `--since`/`--lines`, sectioned human output, JSON output, redaction of tokens/private keys/provider-looking secrets, and graceful partial log failures. | Disposable host smoke confirms collected journal and `_diag` snippets match real service files without requiring manual SSH spelunking. |
| `REL-03` | Doctor tests cover stable finding IDs, severity, evidence, exact remediation commands, local state problems, GitHub auth/list failures, offline/missing/duplicate runners, SSH unreachable, host-key mismatch, service failed/missing, install path/work dir checks, disk/tool/network/time findings, log summaries, JSON output, and read-only behavior. | Optional controlled failure matrix on a disposable host validates that common real failures produce the expected finding categories. |
| `REL-04` | Recovery tests cover confirmed/`--yes` service restart, service reinstall/start when unit is missing, re-registration with fresh tokens and saved labels/name/work dir, GitHub online wait, state update of `Cleanup.GitHubRunnerID`, dry-run/no-confirm no-op, and fail-closed behavior for SSH unreachable or host-key mismatch. | Disposable host smoke stops a service, runs recovery, and verifies the runner returns online in GitHub. |
| `CLEAN-02` | Down tests cover stale GitHub deletion by saved runner ID, fallback by saved name + RunnerKit labels, explicit local-state-missing deletion by runner ID, SSH-unreachable GitHub-only cleanup, GitHub auth failure preserving pending state, and idempotent already-missing runners. | Disposable repo smoke confirms stale records can be removed even when the host is unavailable. |
| `CLEAN-03` | Cleanup plan tests assert only recorded RunnerKit-managed runner-specific artifacts are targeted: `Machine.ServiceName`, `Machine.InstallPath`, `Machine.WorkDir`, and the matching local state record; tests must reject broad deletion of shared `/var/lib/runnerkit` unless proven empty/exclusive. | Disposable host smoke inspects that unrelated user data and shared parent directories remain after `runnerkit down --yes`. |

### Per-plan validation focus

| Plan | Requirement focus | Automated focus | Minimum command |
| --- | --- | --- | --- |
| `03-01` Multi-source status reconciliation | `REL-01` | Unit/CLI tests for repo resolution, state listing, GitHub runner lookup, SSH/service probes, health classifier states (`ready`, `busy`, `needs_attention`, `broken`, `unknown`), label drift, snippets, human/JSON contracts, and read-only call counts. | `go test ./internal/cli ./internal/state ./internal/github ./internal/remote ./internal/ui ./...` |
| `03-02` Logs and doctor diagnostics | `REL-02`, `REL-03`, `STATE-02` carry-forward | Unit/CLI tests for bounded logs, redaction, partial failures, doctor finding IDs/severity/evidence/remediation, selected preflight reuse, docs troubleshooting greps, and no mutation calls. | `go test ./internal/cli ./internal/redact ./internal/preflight ./internal/remote ./internal/ui ./...` |
| `03-03` Guided persistent-runner recovery | `REL-04`, `GH-03` | Recovery plan tests with fake remote/GitHub/state for restart, service reinstall, re-registration, token lifecycle, online wait, state update, dry-run, confirmation, and fail-closed unsafe cases. | `go test ./internal/cli ./internal/github ./internal/state ./internal/workflow ./internal/redact ./...` |
| `03-04` BYO cleanup and stale deregistration | `GH-03`, `CLEAN-02`, `CLEAN-03` | Cleanup plan and command tests for `down --dry-run`, interactive per-artifact prompts, `--yes` safe defaults, stale GitHub deletion, host-side removal token path, API fallback, exact service/path removal, partial cleanup checkpoints, and state remove/update helpers. | `go test ./internal/cli ./internal/state ./internal/github ./internal/remote ./internal/workflow ./...` |

### Automated versus manual-only checks

- **Automated by default:** all decision logic, command output, JSON contracts, state transitions, GitHub API calls, remote command selection, redaction, confirmation gates, dry-run behavior, partial failure behavior, and docs presence checks must be covered by Go tests or deterministic grep/test commands.
- **Manual/controlled only:** real GitHub Actions runner UI status, actual SSH/systemd/journal behavior on a disposable Linux host, and real cleanup side effects on a disposable BYO install. These require external credentials/hosts and should be final smoke checks, not task-level gates.
- **Manual check command sequence after implementation, if a disposable host/repo is available:** `runnerkit up --repo owner/name --host user@host --yes`, `runnerkit status --repo owner/name`, `runnerkit logs --repo owner/name --lines 50`, `runnerkit doctor --repo owner/name`, stop the service on the host, `runnerkit recover --repo owner/name --yes`, then `runnerkit down --repo owner/name --yes`.
- **Manual safety constraint:** never run cleanup smoke against a host with unrelated important data in the recorded install/work paths; use a disposable test repo and host only.

### Wave 0 test scaffolding needs

Wave 0 is applicable for Phase 3 because current runner-operation fakes are mostly embedded in `internal/cli/root_test.go` and are not rich enough for status/logs/doctor/recover/down matrices.

- Extend `internal/testsupport` with a reusable GitHub fake that satisfies `internal/cli.GitHubService`, including `CreateRegistrationToken`, `CreateRemovalToken`, `ListRunners`, `DeleteRunner`, call recording, injected errors, duplicate runner candidates, and mutable runner status/labels/busy fields.
- Add or centralize a reusable remote fake with scripted `remote.Command.ID` results for fast status probes, `systemctl show`, `journalctl`, `_diag` reads, path existence, service restart/reinstall, `config.sh remove`, re-registration, and cleanup commands.
- Add state fixture builders for healthy BYO runner, GitHub offline, busy runner, label drift, SSH unreachable, host-key mismatch, missing GitHub runner, missing service, partial cleanup pending, and local-state-missing stale GitHub cases.
- Add output assertion helpers for human text, JSON decoding, stable finding IDs, `redactions_applied: true`, and absence of raw tokens/private keys/provider-looking secrets.
- Add state store tests for `ListRepositories`, `UpdateRepository`, `RemoveRepository`, and partial cleanup/recovery checkpoint persistence before commands depend on those helpers.
- Wave 0 is test scaffolding only; product behavior can remain red until the corresponding plan tasks implement it, but scaffolding must compile under `go test ./...`.

## RESEARCH COMPLETE

