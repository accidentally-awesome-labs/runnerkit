# Phase 2: BYO Persistent Runner Happy Path - Research

**Researched:** 2026-04-29
**Status:** Complete
**Confidence:** MEDIUM-HIGH

## Research Question

What do we need to know to plan Phase 2 well: SSH preflight, safe non-root Linux/systemd bootstrap, repository-scoped GitHub runner registration, stable labels, completion guidance, warnings, smoke validation, and BYO quickstart.

## Phase Requirements Covered

- `CLI-03`: complete the supported BYO happy path in about 10 minutes.
- `CLI-04`: show runner name, labels, machine target, and next workflow step.
- `GH-02`: register repository-scoped runner without manual GitHub copy/paste commands.
- `GH-04`: predictable RunnerKit labels.
- `GH-05`: copy-paste `runs-on` guidance without editing workflow YAML.
- `MACH-01`: SSH to existing Linux machine and run preflight before install.
- `MACH-02`: bootstrap dedicated non-root runner user, dependencies, and managed service.
- `RUN-01`: persistent runner as default for trusted private solo repos.
- `RUN-03`: warnings for public/fork/untrusted persistent runner workloads.
- `DOC-01`: concise BYO quickstart.

## Locked Context Constraints

Phase 2 must extend `runnerkit up`, not create a separate primary command. The flow should remain wizard-first interactively, but automation-friendly via flags such as `--repo owner/name` and `--host user@host`. It must preserve Phase 1's fail-closed GitHub/public-repo safety behavior, secret-free state, stable labels, human/JSON output, and centralized redaction.

Host-key trust is a product requirement: unknown hosts should show a fingerprint and require acceptance, accepted fingerprints should be stored in RunnerKit state, and future mismatches must fail closed. Remote mutation must only happen after preflight and an explicit fix/install plan. The runner service must not run as root by default.

## Existing Code Facts

### Current foundation to reuse

- `internal/cli/up.go` already owns the `runnerkit up` flow: repo resolution, auth verification, public/fork safety gate, dry-run, `--yes`, `--non-interactive`, `--json`, state save/replace, labels, and output rendering.
- `internal/github/client.go` already implements repository metadata and short-lived registration/removal token creation with redaction registration.
- `internal/github/service.go` currently exposes only `Repository` and `VerifyAuth` through `cli.GitHubService`; Phase 2 needs a broader RunnerKit-owned interface for actual runner lifecycle methods.
- `internal/labels/labels.go` already generates `self-hosted`, `runnerkit`, repo-specific, OS, arch, and mode labels plus a `runs-on: [...]` snippet and the warning not to use `runs-on: self-hosted` alone.
- `internal/state/schema.go` already has `RunnerIdentity`, `MachineRef`, `ProviderRef`, and `CleanupMetadata`. `MachineRef` needs SSH identity fields (port, fingerprint, fingerprint algorithm, maybe accepted time) and `CleanupMetadata.GitHubRunnerID` should be filled after registration.
- `internal/workflow/plan.go` provides lightweight plan/checkpoint primitives suitable for preflight/fix/install/registration steps.
- Existing validation command is `go test ./...`; it is green before Phase 2 planning.

### Current gaps Phase 2 must introduce

- No SSH/remote executor abstraction exists yet.
- No host-key verifier or fingerprint persistence exists yet.
- No runner inventory/list/delete/status GitHub API wrapper exists yet.
- No bootstrap installer/service script generation exists yet.
- `runnerkit up` copy still says Phase 1 does not install a runner; Phase 2 must replace this with real BYO setup progress and completion summary.
- No docs/README exist yet; Phase 2 should add a concise BYO quickstart file.

## External Technical Findings

### GitHub runner APIs and tokens

GitHub's REST self-hosted runner API supports repository runner registration/removal token creation, runner listing, runner lookup/delete, and label management. RunnerKit already uses the registration-token endpoint as a least-privilege permission check. Phase 2 should add typed wrappers for at least:

- `POST /repos/{owner}/{repo}/actions/runners/registration-token`
- `POST /repos/{owner}/{repo}/actions/runners/remove-token`
- `GET /repos/{owner}/{repo}/actions/runners`
- `GET /repos/{owner}/{repo}/actions/runners/{runner_id}` (or list filtering if simpler)
- `DELETE /repos/{owner}/{repo}/actions/runners/{runner_id}` for stale duplicate handling later; Phase 2 may only need list/status and a clear duplicate error if full cleanup is deferred.

Registration/removal tokens are short-lived secrets. Request them just in time, use immediately, register with the redactor, never persist, and re-request on retry rather than replaying old scripts.

### Official runner package and service behavior

GitHub's Linux self-hosted runner flow uses release tarballs from `actions/runner` and `config.sh`. The official releases publish Linux `x64` and `arm64` tarballs plus SHA-256 checksums in release notes. The releases page notes progressive rollout, so a pinned known-good version is safer than always using latest in Phase 2.

For Linux systems using systemd, GitHub's generated `svc.sh` can install and manage the runner as a service after `config.sh` succeeds. Use `sudo ./svc.sh install runnerkit-runner`, then `sudo ./svc.sh start`, and verify with `sudo ./svc.sh status` or `systemctl is-active`. The service user argument is the key guardrail: install/setup may need elevated privileges, but the service user must be the dedicated unprivileged runner user.

### SSH host-key handling

`golang.org/x/crypto/ssh/knownhosts` provides an OpenSSH-compatible host-key callback for known-hosts files. Phase 2 should wrap it behind a RunnerKit `HostKeyVerifier` so tests can simulate unknown, accepted, and mismatch states. For the product requirement, RunnerKit state should store an explicit accepted host fingerprint separate from the user's global `~/.ssh/known_hosts` behavior, because the state is used to fail closed on later mismatches even if system SSH config changes.

Recommended fingerprint display format: OpenSSH-style `SHA256:<base64>` for the presented host public key, plus algorithm (`ssh-ed25519`, `ecdsa-sha2-*`, `rsa-sha2-*`) and host/port. Store the exact accepted fingerprint string and algorithm in state.

### System SSH vs Go SSH

Use a RunnerKit remote abstraction, not direct SSH calls in `cli`. The implementation can start with system `ssh`/`scp` for developer-friendly behavior with existing SSH config/agent, or `x/crypto/ssh` for direct host-key control. Because explicit host-key trust is a hard product requirement, the plan should either:

1. implement Go SSH first with a testable host-key callback; or
2. implement a hybrid where a Go probe captures/verifies the host key before all subsequent system `ssh` commands run with `StrictHostKeyChecking=yes` and controlled known-host settings.

The plan should avoid baking shell strings into `up.go`; put SSH concerns in `internal/remote` with fakes.

## Recommended Phase 2 Architecture

### New internal boundaries

1. `internal/remote`
   - `Target` parser for `user@host[:port]` plus key path and optional alias.
   - `Executor` interface: `Probe`, `Run`, `Upload`, maybe `RunScript`.
   - `HostKeyVerifier` interface: classify unknown/accepted/mismatch, return display fingerprint.
   - Redacted command output and structured remote errors.

2. `internal/preflight`
   - Remote checks with typed IDs and fixability: SSH connectivity, host-key trust, OS/arch, systemd, sudo/install privileges, disk, required tools, time sync, outbound HTTPS to GitHub/runner downloads, existing service/dir conflict.
   - `FixPlan` for installable missing dependencies. The flow should show fixes before mutation.

3. `internal/bootstrap`
   - Version-pinned runner package metadata for Linux x64/arm64.
   - Script/rendered steps for creating `runnerkit-runner`, install dirs, work dir, metadata dirs, package download/checksum/extract, `config.sh --unattended`, `svc.sh install runnerkit-runner`, service start/status.
   - Idempotent rerun behavior: skip existing user/dirs, verify ownership, do not create duplicate GitHub registrations without explicit handling.

4. `internal/github` runner lifecycle additions
   - `Runner` struct with ID, name, OS, status, busy, labels.
   - `ListRunners`, `GetRunner`, maybe `DeleteRunner`.
   - `CreateRegistrationToken` already exists at client level; expose through the service/interface used by CLI workflows.

5. `internal/cli` orchestration
   - Keep Cobra thin; `runUp` should branch from foundation-only preview into BYO flow when host/profile info is provided or collected.
   - Expand `upOptions` with host/user/port/key flags, best-effort distro override, install path, runner version if needed, and dry-run/no-mutate behavior.
   - Human output should show progress categories; JSON output should include machine, runner, labels, service, and next steps with `redactions_applied: true`.

### Remote host defaults

- Runner user: `runnerkit-runner`.
- Install root: `/opt/actions-runner/<runner-name>`.
- Work directory: `/var/lib/runnerkit/work/<runner-name>` or GitHub runner `_work` under the install root if using official defaults; choose one and persist it.
- RunnerKit metadata: `/var/lib/runnerkit/metadata.json` and logs/checkpoints under `/var/lib/runnerkit`.
- Service: generated by `svc.sh` with a predictable systemd unit name discoverable from the runner directory; persist the service name in state.
- Supported Phase 2 platforms: Linux `x64` and `arm64` with systemd. Common distros are best-effort if they have compatible glibc and package tooling; unknown distros warn and require an override.

### Preflight checklist minimum

Each check needs an ID, user-facing description, machine-readable result, remediation, and whether RunnerKit can fix it.

- `ssh.connectivity`: can authenticate to target within timeout.
- `ssh.host_key`: fingerprint is accepted or promptable; mismatch fails closed.
- `host.os_release`: reads `/etc/os-release` and `uname -s`.
- `host.arch`: maps `x86_64|amd64 -> x64`, `aarch64|arm64 -> arm64`; unsupported architectures fail.
- `host.systemd`: `systemctl --version` and `/run/systemd/system` present.
- `host.privilege`: can run required sudo commands non-interactively or has an interactive escalation plan.
- `host.disk`: enough free space in install/work roots (start with >= 2 GiB).
- `host.tools`: `curl`, `tar`, `gzip`, `sha256sum`, `id`, `useradd`, `install` present or installable.
- `host.network.github`: outbound HTTPS to `github.com` and `api.github.com`; runner release download reachable.
- `host.time`: time sync roughly sane (`timedatectl` if available) to avoid TLS/token errors.
- `runner.conflict`: runner dir/service/name not already active unless the flow is an explicit replace/repair.

## Implementation Risks and Planning Guidance

### Token passage to `config.sh`

The official `config.sh` flow expects a registration token. Passing a token as a command argument can appear in process listings briefly. If the official runner does not provide a safer stdin/env-only interface, Phase 2 should minimize exposure by running the configuration step immediately on a dedicated host, disabling shell tracing, not logging command lines, using protected temp files only when necessary, deleting temp files immediately, registering the token with the redactor, and proving state/logs do not persist it. Do not copy durable GitHub auth to the remote host.

### Duplicate/stale runners

Before registration, list repository runners and check for the intended runner name. In Phase 2 happy path, recommended behavior is:

- If the same name is online/busy: stop with a clear duplicate message unless an explicit replace path is planned.
- If the same name is offline: show a warning and require typed/flagged replacement before deleting or reconfiguring.
- If delete/reconcile is too much for Phase 2, fail with exact GitHub UI/API cleanup instructions and defer automated cleanup to Phase 3. However, the plan should at least prevent silent duplicate names.

### Root and Docker risk

The runner service must not run as root. Preflight/bootstrap tests should grep generated service/install content for `svc.sh install runnerkit-runner` or equivalent `User=runnerkit-runner`. Docker is not a Phase 2 must-have; if Docker is detected or the user asks for Docker labels/groups, warn that Docker group access is root-equivalent and defer full Docker profile support.

### Output and redaction

Remote command output can include paths, hosts, command args, package-manager details, and token-adjacent text. All user-facing output must pass through the existing renderer/redactor. Tests should include fake remote output containing a registration token and assert the token does not appear in stdout/stderr/JSON/state.

## Suggested Plan Slices

The roadmap's four planned work slices are sound. Recommended boundaries:

1. **02-01 SSH + preflight**: add `internal/remote`, host flags/prompts, host-key verifier/state fields, platform/preflight checks, dry-run/preflight output, fake executor tests. No runner installation yet.
2. **02-02 bootstrap installer/service**: add runner package metadata, dependency fix plan, dedicated user, directories, checksum download/extract, `config.sh`/`svc.sh` script generation/execution abstractions, idempotency tests with fake remote. Registration token may be a fake input at this stage.
3. **02-03 GitHub registration + completion**: add runner inventory API, real token-to-bootstrap orchestration, duplicate detection, state update with runner ID/machine metadata, service/GitHub online verification, final human/JSON summary and snippet.
4. **02-04 safety/profile/docs/smoke**: persist persistent default profile, strengthen public/fork/untrusted warnings in the real BYO path, add BYO quickstart docs, add fake end-to-end smoke tests and optional manual smoke instructions.

## Testing Strategy

### Automated

- Unit tests for SSH target parsing, host-key classification, fingerprint persistence/mismatch, platform mapping, preflight result aggregation, bootstrap script rendering, GitHub runner API parsing, duplicate detection, state migration/defaults, and completion summary payloads.
- CLI integration tests with fake GitHub and fake remote executor for:
  - missing `--host` in `--non-interactive` returns input-required remediation;
  - unknown host-key requires prompt/acceptance;
  - host-key mismatch fails closed before remote mutation;
  - preflight failure prints fix plan and does not request registration token;
  - successful BYO setup saves machine/runner/service metadata and prints exact `runs-on` snippet;
  - public repo without override remains blocked before SSH mutation;
  - `--json` output is machine-only and includes `redactions_applied: true`.
- Redaction tests with fake registration tokens and remote output.
- `go test ./...` remains the full phase validation command.

### Manual/controlled smoke

A real BYO smoke needs a disposable Linux systemd host and a test GitHub repository. It should verify:

1. `runnerkit up --repo owner/repo --host user@host --yes` reaches completion.
2. GitHub shows the runner online with RunnerKit labels.
3. `systemctl` shows the service active and running as `runnerkit-runner`, not root.
4. A copy-pasted `runs-on` snippet can route a simple workflow job.
5. No registration token appears in local state, CLI output, or obvious remote metadata files.

This can be documented in Phase 2 but may remain manual until later cloud automation exists.

## Documentation Requirements

Phase 2 should create an initial quickstart because no README/docs exist yet. Keep it concise and concrete:

- Prerequisites: RunnerKit binary/source checkout, `gh` auth or `RUNNERKIT_GITHUB_TOKEN`, private/trusted GitHub repo, SSH access to a Linux systemd host, sudo ability.
- Safety note: persistent runners are for trusted private workloads; public/fork/untrusted workloads should use GitHub-hosted or future ephemeral mode.
- Command: `runnerkit up --repo owner/name --host user@host`.
- What RunnerKit does: preflight, host-key prompt, dedicated user, official runner download/checksum, repo runner registration, systemd service start.
- What RunnerKit does not do: edit workflow YAML, destroy BYO host, make public PRs safe.
- Copy-paste workflow snippet pattern using RunnerKit labels.
- Basic troubleshooting: SSH fails, unsupported OS/arch, sudo missing, service not active, GitHub runner not online.

## Validation Architecture

### Test infrastructure

- Framework: Go `testing` with `go test`.
- Current config: `go.mod`, no separate test config.
- Quick command: `go test ./...` (currently green and fast enough for every task).
- Full suite: `go test ./...`.
- Optional manual smoke: disposable Linux systemd host + test GitHub repo after implementation.

### Feedback sampling contract

- Run `go test ./...` after every task and every plan wave.
- For tasks that create generated shell scripts, add tests that inspect rendered script strings; add ShellCheck later only if the project intentionally adds that dev dependency.
- Every plan task should include grep-verifiable acceptance criteria for exact field names, commands, labels, docs headings, or test names.
- No three consecutive tasks should be docs-only/manual-only; each implementation task needs an automated test or fake adapter test.

### Requirement-to-validation map

| Requirement | Automated validation                                                                           | Manual/controlled validation               |
| ----------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `MACH-01`   | fake remote preflight tests for SSH, host-key, OS/arch, systemd, sudo, disk, tools, network    | disposable Linux host preflight            |
| `MACH-02`   | bootstrap renderer/executor tests assert dedicated user, dirs, checksum, service user          | systemd service active as non-root         |
| `GH-02`     | fake GitHub API tests for token/list/status and no manual command output                       | GitHub UI runner registered                |
| `GH-04`     | label unit tests with detected OS/arch/mode                                                    | GitHub runner labels match snippet         |
| `GH-05`     | CLI output tests for exact `runs-on: [...]`; no workflow file edits                            | copy snippet into workflow                 |
| `CLI-03`    | fake happy-path integration test finishes without unsupported prompts in non-interactive flags | timed real host smoke                      |
| `CLI-04`    | human/JSON summary tests include runner name, labels, machine target, next workflow step       | completion output review                   |
| `RUN-01`    | state/default tests assert mode `persistent`                                                   | service remains active after command exits |
| `RUN-03`    | public/fork safety tests remain blocking before SSH mutation                                   | user-facing warning review                 |
| `DOC-01`    | docs file exists and contains quickstart headings/commands/safety copy                         | fresh-user read-through                    |

## Open Questions for Planner Discretion

- Whether Phase 2 should add `golang.org/x/crypto/ssh` immediately or start with system `ssh` plus a Go host-key probe.
- Exact flag names beyond `--host`, such as `--ssh-key`, `--ssh-port`, `--accept-host-key`, `--allow-unknown-linux`, and `--runner-version`.
- Whether duplicate offline runner deletion is implemented in 02-03 or deferred with an explicit fail/remediation path.
- Whether docs should live in `README.md`, `docs/byo-quickstart.md`, or both. Because no README exists, the plan should probably add both a minimal README and a docs quickstart.

## RESEARCH COMPLETE
