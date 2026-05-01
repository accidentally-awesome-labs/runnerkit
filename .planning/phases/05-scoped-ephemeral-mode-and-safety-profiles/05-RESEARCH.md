---
phase: 5
slug: scoped-ephemeral-mode-and-safety-profiles
status: complete
created: 2026-05-01
sources:
  - .planning/ROADMAP.md
  - .planning/REQUIREMENTS.md
  - .planning/STATE.md
  - .planning/PROJECT.md
  - internal/cli/up.go
  - internal/bootstrap/script.go
  - internal/labels/labels.go
  - internal/github/safety.go
  - internal/state/schema.go
  - internal/ops/status.go
  - internal/ops/logs.go
  - internal/ops/doctor.go
  - docs/byo-quickstart.md
  - docs/cloud-quickstart.md
  - https://docs.github.com/en/actions/reference/self-hosted-runners-reference
  - https://docs.github.com/actions/how-tos/managing-self-hosted-runners/monitoring-and-troubleshooting-self-hosted-runners
  - https://docs.github.com/en/actions/reference/security/secure-use
  - https://github.blog/changelog/2021-09-20-github-actions-ephemeral-self-hosted-runners-new-webhooks-for-auto-scaling
---

# Phase 5 Research: Scoped Ephemeral Mode and Safety Profiles

## Objective

Research how to plan Phase 5: **Scoped Ephemeral Mode and Safety Profiles**.

Phase goal from ROADMAP: RunnerKit offers an explicit stronger-isolation ephemeral option without pretending to be a full autoscaling fleet manager, and helps developers choose the right mode for their workload.

Requirement IDs: `RUN-02`, `RUN-04`, `DOC-03`.

No Phase 5 `CONTEXT.md` exists; user chose to continue using roadmap, requirements, research, and existing project decisions.

---

## External Facts That Matter

### GitHub ephemeral runner behavior

- GitHub supports ephemeral self-hosted runners by passing `--ephemeral` to `config.sh` during runner registration, for example `./config.sh --url https://github.com/octo-org --token example-token --ephemeral`.
- GitHub states that with ephemeral runners it guarantees only one job is assigned to a runner; after processing one job, the service automatically de-registers the runner.
- GitHub recommends ephemeral runners for autoscaling and stronger clean-environment automation. It explicitly does **not** recommend autoscaling with persistent runners because jobs can be assigned while persistent runners are shut down.
- If a job targets labels for which no matching online runner exists, the job queues until a matching runner appears or the 24-hour queue timeout expires. RunnerKit should not imply that creating one ephemeral runner is an autoscaling fleet.
- GitHub also supports just-in-time (JIT) runner configuration through the REST API, but a standard registration token plus `config.sh --ephemeral` is sufficient for this v1 scoped mode and is lower scope than implementing JIT config generation.

### Log preservation requirement

- GitHub's monitoring docs say self-hosted runner application logs live under the runner `_diag` directory and job logs use `Worker_*.log` files.
- GitHub warns that runner application log files for ephemeral runners must be forwarded and preserved externally for troubleshooting. For RunnerKit v1, the feasible scoped implementation is: preserve `_diag` and systemd/journal excerpts locally/best-effort before remote/cloud cleanup; clearly document that production-grade external log forwarding is outside v1.

### Security guidance

- GitHub recommends self-hosted runners only for private repositories because public/fork workflows can run dangerous code on the user's machine.
- GitHub's secure-use reference says self-hosted runners do not automatically provide ephemeral clean VM guarantees and can be persistently compromised by untrusted workflows.
- GitHub notes that simply destroying/restarting a persistent runner after jobs is not enough because persistent runners do not guarantee only one job and one job could observe another job's command-line secrets. Ephemeral mode matters because GitHub's service enforces the one-job assignment.
- Reusing the same hardware for JIT/ephemeral runners can still risk information exposure; clean environment automation is needed. RunnerKit should frame BYO ephemeral as **single-job registration**, not as a fully clean machine. The strongest recommendation for public/fork/untrusted workloads should be a throwaway cloud ephemeral runner plus verified destroy.

---

## Current RunnerKit Architecture Relevant to Phase 5

### Existing command and dependency seams

- `internal/cli/root.go` wires Cobra commands and shared dependencies. `up`, `status`, `logs`, `doctor`, `recover`, `down`, `destroy`, and `state` are already command-level seams.
- `internal/cli/up.go` owns setup path choice, GitHub repo/safety gates, BYO preflight/bootstrap, cloud provisioning, registration token creation, online verification, state save, and completion output.
- `internal/bootstrap/install.go` and `internal/bootstrap/script.go` own runner download, `config.sh`, `svc.sh`, and service setup scripts. Persistent mode currently uses `RenderInstallScript` and `RenderServiceScript`.
- `internal/provider/provider.go` / `internal/provider/profile.go` own cloud provisioning inputs and resource tags. `ProvisionInput` already includes runner name, labels, workflow snippet, state ID, and creation time.
- `internal/state/schema.go` already stores `Runner.Mode`, labels, workflow snippet, cleanup metadata, provider identity, and operation checkpoints.
- `internal/labels/labels.go` already supports a `Mode` label option and currently defaults to `persistent`.
- `internal/github/safety.go` currently blocks public repositories by default for persistent setup and uses copy that says to wait for ephemeral mode.
- `internal/ops/status.go`, `internal/ops/logs.go`, and `internal/ops/doctor.go` classify and explain health based on GitHub, SSH, systemd, labels, and provider facts.
- `internal/cli/destroy.go` already implements cloud billable cleanup with provider verification before local state removal, which is the right cleanup base for cloud ephemeral.

### Current assumptions that Phase 5 must alter

- Runner mode is hard-coded as `labels.DefaultMode` / `persistent` in BYO and cloud setup paths.
- Runner name generation does not currently distinguish persistent vs ephemeral. Repeated ephemeral runs need a name that avoids stale GitHub conflicts. Persistent compatibility should be preserved for existing state/tests.
- Persistent bootstrap uses `svc.sh install/start/status`; ephemeral should not blindly reuse a restartable persistent service. A custom `systemd` unit with `Restart=no`, `ExecStart=.../run.sh`, finalizer, and TTL timer is a safer one-job lifecycle.
- `status` treats a missing GitHub runner as recovery-needed. For ephemeral, GitHub runner absence after one job can be an expected terminal state if the finalizer/log preservation completed.
- `destroy` currently handles cloud cleanup; it should remain the billable cleanup command and work for cloud ephemeral state.
- `down` currently handles BYO cleanup; it should remain the BYO cleanup command and work for BYO ephemeral state.

---

## Recommended Product Shape

### User-facing mode selection

Add an explicit mode/profile choice to `runnerkit up`:

```text
--mode persistent|ephemeral
--ephemeral-ttl 24h
```

Recommended behavior:

- Default remains `persistent` for trusted private, non-fork repositories.
- Interactive setup should explain the tradeoff before mutation:
  - `Persistent trusted runner` — lower ongoing friction, reused across jobs, cheapest for repeated trusted private workflows, requires cleanup with `down`/`destroy`, not safe for public/fork/untrusted workflows.
  - `Ephemeral one-job runner` — stronger isolation boundary, GitHub assigns one job then deregisters, higher setup/cost per run, requires log preservation and cleanup; not a full autoscaling fleet.
- Non-interactive setup requires `--mode ephemeral` for ephemeral. Missing mode keeps persistent default only if the repository passes the existing trusted-private safety gate.
- For public/fork repositories, persistent should still block by default. The error copy should now point to `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` as the recommended safer path, while keeping `--allow-public-repo-risk` as an explicit dangerous persistent override.

### Safety profiles

Represent safety as an internal profile decision, not necessarily a separate top-level command:

| Profile              | Applies to                                        | Allowed default                                                    | Required copy                                                                                   |
| -------------------- | ------------------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------- |
| `persistent-trusted` | private non-fork repos                            | yes                                                                | trusted-private, reusable runner, cleanup command                                               |
| `persistent-risky`   | public/fork/untrusted repo with persistent mode   | no unless explicit override                                        | public/fork warning, can run untrusted code repeatedly, prefer ephemeral cloud                  |
| `ephemeral-byo`      | BYO machine, `--mode ephemeral`                   | yes for trusted private; risky repos require typed acknowledgement | one-job GitHub registration, not a clean VM, machine/network/secrets remain user responsibility |
| `ephemeral-cloud`    | cloud machine, `--mode ephemeral --cloud hetzner` | recommended for stronger isolation                                 | one job, TTL, logs, destroy command, not autoscaling                                            |

### Lifecycle implementation

For `--mode ephemeral`, use a separate bootstrap path:

1. Build labels with mode `ephemeral`, e.g. `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]`.
2. Generate an ephemeral runner name such as `runnerkit-owner-repo-ephemeral-<short-id>` while preserving the existing persistent name `runnerkit-owner-repo-local`.
3. Run preflight/readiness before any registration token just like persistent.
4. Create registration token just-in-time, register redaction, then configure runner with:
   ```text
   ./config.sh --unattended --url https://github.com/owner/name --token "$RUNNERKIT_REGISTRATION_TOKEN" --name <runner> --labels <labels> --work <workdir> --replace --ephemeral
   ```
5. Install a RunnerKit-managed systemd unit for one-job execution with `Restart=no` and `ExecStart=<installPath>/run.sh` rather than `svc.sh install`.
6. Install a finalizer script and TTL timer:
   - finalizer copies `_diag/Runner_*.log`, `_diag/Worker_*.log`, and bounded journal excerpts into `/var/lib/runnerkit/ephemeral/<runner>/logs`.
   - finalizer removes local runner credentials/config best-effort and writes a sentinel/status file such as `/var/lib/runnerkit/ephemeral/<runner>/state.json`.
   - TTL timer defaults to 24h and stops/finalizes if no job completes.
7. State records `Runner.Mode = "ephemeral"`, TTL expiry, cleanup command (`runnerkit destroy --repo owner/name` for cloud; `runnerkit down --repo owner/name` for BYO), log archive path, and finalizer status.
8. `status`/`doctor` understand expected ephemeral terminal states: waiting for job, busy, completed-needs-cleanup, ttl-expired, cleanup-pending.
9. `logs` can collect preserved ephemeral logs from the saved archive path in addition to current `_diag` and journal.
10. Cloud `destroy` remains the billable final cleanup for cloud ephemeral; BYO `down` remains the host cleanup for BYO ephemeral.

### Scope boundaries

Do **not** implement in Phase 5 unless a plan explicitly says otherwise:

- No hosted control plane.
- No webhook listener or true autoscaling fleet.
- No Actions Runner Controller (ARC), Kubernetes, scale sets, organization-level runner management, or JIT runner API.
- No automatic workflow YAML edits.
- No guarantee that BYO ephemeral is a clean VM; it is single-job GitHub assignment plus local cleanup, not hardware isolation.

---

## Codebase Planning Risks

1. **Mode should not be sprinkled through `up.go`:** create small helpers/types for mode/profile decisions and ephemeral options so `runUp` remains testable.
2. **GitHub token side effects must remain late:** keep registration-token creation after repo resolution, safety, setup path, preflight/readiness, provider plan/confirmation, and duplicate runner checks.
3. **Persistent compatibility:** existing tests expect persistent labels/snippets and runner name `runnerkit-owner-repo-local`. Do not change persistent defaults without updating backwards compatibility deliberately.
4. **Systemd service semantics:** using `svc.sh install` for ephemeral may restart or leave confusing service state. Prefer a custom unit with `Restart=no` and a RunnerKit finalizer.
5. **Cloud billing:** ephemeral cloud can still bill after the one job. Completion output and docs must put `runnerkit destroy --repo owner/name` near the workflow snippet.
6. **Logs before deletion:** cloud destroy can make logs unrecoverable. Preserve best-effort logs before remote/provider cleanup where SSH is reachable.
7. **Status semantics:** GitHub runner missing is failure for persistent but can be success/completed for ephemeral. Health classification needs mode-aware branches.
8. **Safety copy accuracy:** ephemeral is stronger isolation, not absolute safety. BYO ephemeral and cloud ephemeral need different warnings.

---

## Suggested Phase Decomposition

1. **05-01 Mode/profile selection and safety policy**
   - Add mode flag/options, tradeoff prompt/output, mode labels/snippets, safety profile evaluation, and persistent-vs-ephemeral copy.
   - Address `RUN-04` and the selection half of `RUN-02`.

2. **05-02 Ephemeral lifecycle and operations**
   - Add ephemeral bootstrap scripts, custom one-shot systemd service, finalizer, TTL timer, state metadata, cloud/BYO integration, logs/status/doctor semantics, and cleanup command wiring.
   - Address the implementation half of `RUN-02`.

3. **05-03 Safety guide and risky-workload validation**
   - Add `docs/safety.md`, README/docs updates, CLI docs assertions, and fake E2E tests for private persistent, public/fork blocked persistent, cloud ephemeral recommended, TTL/log preservation, and cleanup paths.
   - Address `DOC-03` and reinforce `RUN-04`.

---

## Validation Architecture

Framework: Go `testing` with focused package runs and grep/docs assertions.

Recommended commands:

- Focused CLI/mode tests: `go test ./internal/cli/... ./internal/github/... ./internal/labels/... ./internal/state/...`
- Ephemeral lifecycle tests: `go test ./internal/bootstrap/... ./internal/cli/... ./internal/ops/... ./internal/state/...`
- Docs and full regression: `go test ./... && grep -R "runnerkit up --repo owner/name --mode ephemeral --cloud hetzner" README.md docs && grep -R "persistent self-hosted runners" README.md docs`

Required fake coverage:

- `--mode ephemeral` renders `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]` and configures `config.sh` with `--ephemeral`.
- Persistent default still renders `runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]` and runner name `runnerkit-owner-repo-local`.
- Public/fork persistent setup blocks before registration token and recommends ephemeral cloud.
- Cloud/BYO ephemeral create no registration token before preflight/readiness and confirmation gates.
- Ephemeral finalizer command includes `_diag`, `Runner_*.log`, `Worker_*.log`, TTL sentinel, and no raw registration/removal tokens in output/state.
- Mode-aware status treats completed/auto-deregistered ephemeral runner as a cleanup-needed terminal state, not persistent recovery.
- `runnerkit destroy` supports cloud ephemeral state and preserves/fetches logs before deleting cloud resources when SSH is reachable.

Manual validation:

- A controlled live cloud ephemeral smoke remains recommended before public release: create a private test repo, run one ephemeral Hetzner runner, trigger one job on the ephemeral labels, verify GitHub auto-deregistration, inspect preserved logs, and run `runnerkit destroy` to stop billing.

---

## Research Complete

Phase 5 is plan-ready. The main technical decision is to implement scoped `config.sh --ephemeral` plus one-shot service/finalizer/TTL/log preservation, while keeping autoscaling/JIT/ARC out of scope and making cloud ephemeral the recommended safer path for public/fork/untrusted workloads.

## RESEARCH COMPLETE
