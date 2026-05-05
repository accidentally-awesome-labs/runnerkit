---
gsd_state_version: 1.0
milestone: v1.0.0
milestone_name: milestone
status: executing
stopped_at: Completed 06-08-byo-register-runner-runas-fix-PLAN.md
last_updated: "2026-05-05T16:31:34.438Z"
last_activity: 2026-05-05
progress:
  total_phases: 6
  completed_phases: 5
  total_plans: 27
  completed_plans: 26
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-29)

**Core value:** A solo developer can get a reliable, cost-effective GitHub Actions self-hosted runner online and usable in a project in about 10 minutes, without manual GitHub runner setup headaches.
**Current focus:** Phase 06 — release-upgrade-docs-and-v1-validation

## Current Position

Phase: 06 (release-upgrade-docs-and-v1-validation) — EXECUTING
Plan: 2 of 8
Status: Ready to execute
Last activity: 2026-05-05

Milestone Progress: [███████░░░] 67%

## Performance Metrics

**Velocity:**

- Total plans completed: 16
- Average duration: 27 min
- Total execution time: 7.2 hours

**By Phase:**

| Phase | Plans | Total   | Avg/Plan |
| ----- | ----- | ------- | -------- |
| 01    | 4/4   | 71 min  | 18 min   |
| 02    | 4/4   | 95 min  | 24 min   |
| 03    | 4/4   | 176 min | 44 min   |
| 04    | 4/4   | 90 min  | 23 min   |

**Recent Trend:**

- Last 5 plans: 03-04 (52 min), 04-01 (9 min), 04-02 (15 min), 04-03 (27 min), 04-04 (39 min)
- Trend: Phase 4 is complete: RunnerKit now has Hetzner provisioning, shared cloud runner bootstrap, final cloud state, provider-aware status/logs/doctor, billable destroy verification, and cloud quickstart documentation. Phase 5 planning is complete with 3 plans for scoped ephemeral mode and safety profiles.

_Updated after each plan completion_
| Phase 05 P01 | 16 | 3 tasks | 11 files |
| Phase 05-scoped-ephemeral-mode-and-safety-profiles P02 | 23 min | 3 tasks | 24 files |
| Phase 05-scoped-ephemeral-mode-and-safety-profiles P03 | 16 min | 3 tasks | 13 files |
| Phase 06-release-upgrade-docs-and-v1-validation P02 | 12 min | 3 tasks tasks | 12 files files |
| Phase 06-release-upgrade-docs-and-v1-validation P03 | 14m | 4 tasks tasks | 20 files files |
| Phase 06-release-upgrade-docs-and-v1-validation P01 | 5m | 4 tasks (5 planned; Task 5 is human-action checkpoint) tasks | 6 files files |
| Phase 06-release-upgrade-docs-and-v1-validation P04 | 8m | 3 of 4 tasks (Task 4 is checkpoint:human-action awaiting maintainer) tasks | 11 created + 2 modified files |
| Phase 06-release-upgrade-docs-and-v1-validation P05 | 8m | 2 tasks | 12 files |
| Phase 06-release-upgrade-docs-and-v1-validation P06 | 11m | 3 tasks tasks | 15 files (4 created, 11 modified) files |
| Phase 06-release-upgrade-docs-and-v1-validation P08 | 5m | 2 tasks tasks | 4 files (1 production, 2 test, 1 smoke) files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Initialization: GitHub Actions only for v1; CLI-only interface; solo developers first.
- Initialization: Support both BYO machines and one recommended low-cost cloud provisioning path.
- Initialization: Persistent runners are the default for trusted private repos; ephemeral mode is explicit for stronger isolation.
- Roadmap: Build BYO persistent runner before cloud provisioning; harden diagnostics/cleanup before adding billable cloud resources.
- Phase 1 context: Prefer richer CLI wizard/TUI setup, `gh` auth first with fine-grained token fallback, git remote repo detection with confirmation, fail-closed permission handling, and optional project config plus user-local state.
- Plan 01-01: Use Cobra v1.10.1 for the RunnerKit CLI command tree.
- Plan 01-01: Centralize current human and JSON output through a renderer that adds `redactions_applied: true`.
- Plan 01-02: Use runner registration token creation as the runner-management permission check and immediately register returned tokens with the redactor.
- Plan 01-02: Block public repositories by default with `public_repo_risk`; require explicit `--allow-public-repo-risk` for future persistent setup.
- Plan 01-03: Persist Phase 1 foundation state as versioned, secret-free JSON with atomic writes and migration hooks.
- Plan 01-03: Use stable `runnerkit-owner-repo` labels plus explicit `runs-on` guidance; never recommend `self-hosted` alone.
- Plan 01-03: Require typed `replace owner/repo` or `--yes --replace` before replacing existing repo state.
- Plan 01-04: Production `runnerkit up` now defaults to `gh.NewService` plus `github.OSCommandRunner{}`; fake-permitted GitHub behavior is test-only.
- Plan 01-04: The real GitHub service caches credentials in memory only and uses registration-token creation plus repository metadata for permission/safety checks.
- Phase 2 context: Extend `runnerkit up` for BYO with wizard-first SSH detail collection and automation-friendly host flags.
- Phase 2 context: Prompt with SSH host fingerprint, record accepted fingerprint in state, and fail closed on mismatch.
- Phase 2 context: Support common systemd Linux best-effort; unknown or unverified distros warn, require explicit override, then try.
- Phase 2 context: Run full preflight before remote mutation, show fix plans before applying changes, and report remote progress with redacted actionable failure copy.
- Phase 2 context: Exact privilege flow is planner discretion, but the persistent runner service must not run as root by default.
- Plan 02-01: BYO SSH target parsing, host-key trust, and preflight checks are separate packages behind `remote.Executor`.
- Plan 02-02: Bootstrap uses pinned GitHub Actions runner 2.334.0 packages and installs the service as `runnerkit-runner`.
- Plan 02-03: Runner registration is just-in-time after preflight and duplicate checks; state is saved only after GitHub reports the runner online with RunnerKit labels.
- Plan 02-04: Persistent BYO is documented and warned as trusted-private-repository only; RunnerKit prints snippets and does not edit workflow YAML.
- Phase 3 context: `runnerkit status` should default to the current repo, stay read-only, use fast health probes, show derived health plus source facts, include the saved `runs-on` snippet, flag label drift, and expose the same model in JSON.
- Phase 3 context: BYO cleanup should use `runnerkit down`; interactive cleanup asks artifact-by-artifact, while `down --yes` applies a safe default plan limited to RunnerKit-managed runner-specific artifacts.
- Phase 4 context: Cloud provider/profile is planner discretion after research, optimized for smooth setup/reliability over absolute lowest cost; cloud auth should reuse provider CLI/env credentials; interactive `runnerkit up` offers cloud vs BYO when no host is provided; non-interactive cloud requires explicit cloud flags; provisioning plans show cost, resources, identity/tags, labels, and exact `runnerkit destroy`; state stores full cloud resource inventory; `runnerkit destroy` verifies GitHub removal plus provider resources absent/non-billable and keeps pending checkpoints on partial failure.
- Plan 04-01: Hetzner is the first registered cloud provider path, with default `fsn1`/`cpx22`/`ubuntu-24.04`/`runnerkit-admin` planning profile and env-only token discovery.
- Plan 04-01: Non-interactive cloud setup requires explicit `--cloud hetzner --yes`; missing `--host` plus `--yes` fails before provider, remote, state, or registration-token side effects.
- Plan 04-01: Cloud pre-provisioning uses non-token runner-management read checks and renders plan-before-mutation output with cost caveat, resource names/tags, labels, and future destroy command.
- Plan 04-02: Use hcloud-go v1.59.2 (not /v2) for the Hetzner adapter while the module targets Go 1.22.
- Plan 04-02: Store full Hetzner cloud inventory under ProviderRef.Cloud while preserving existing provider kind/ids/region compatibility.
- Plan 04-02: Persist cloud_provision_pending immediately after billable resources exist and cloud_readiness_pending if provider, SSH, cloud-init, or preflight readiness fails.
- Plan 04-02: Keep provider-only readiness in internal/provider/hetzner and CLI-owned SSH/cloud-init/BYO preflight readiness before any registration-token creation.
- Plan 04-03: Cloud runner installation reuses the BYO bootstrap.Apply/service/online-verification path after provider and SSH readiness pass.
- Plan 04-03: Successful cloud setup replaces pending checkpoints with final cloud-ssh state, GitHub runner ID, provider inventory, and deterministic cleanup IDs.
- Plan 04-03: Status and doctor use provider Describe for cloud facts while logs use saved provider metadata; operations never call provider mutation methods.
- Plan 04-04: Cloud cleanup uses `runnerkit destroy`; it plans GitHub, remote, provider, and local-state cleanup before mutation, verifies Hetzner resources absent/non-billable before state removal, and keeps pending checkpoints for partial cleanup.
- [Phase 05]: Mode and safety-profile decisions live in a new internal/runmode package so the CLI, labels, state, and tests share one typed Decision.
- [Phase 05]: The runner-mode prompt replaces the previous setup-path prompt; selecting persistent-byo or ephemeral-byo lets resolveBYOTarget collect a host while ephemeral-cloud selects --cloud hetzner.
- [Phase 05]: Public/fork persistent setup blocks with the new UI-SPEC body and DangerousPersistentOverrideCopy before any GitHub auth, registration token, remote, provider, or state mutation; ephemeral cloud is the recommended public/fork path; ephemeral BYO on public/fork requires typed ack or --allow-ephemeral-byo-risk --yes.
- [Phase 05-scoped-ephemeral-mode-and-safety-profiles]: Plan 05-02: Ephemeral lifecycle classification (waiting/busy/completed/ttl_expired/cleanup_pending) runs before persistent github_runner_missing in ops.Classify so completed auto-deregistration is reported as terminal progress.
- [Phase 05-scoped-ephemeral-mode-and-safety-profiles]: Plan 05-02: bootstrap.ApplyEphemeral mirrors Apply's command shape but replaces svc.sh install/start with finalizer/service/TTL-timer/verify steps; ServiceNotActiveError surfaces only for ephemeral install/verify failures.
- [Phase 05-scoped-ephemeral-mode-and-safety-profiles]: Plan 05-02: ephemeral.logs.preserve runs before file/provider mutation in down/destroy and never blocks cleanup; failures record ephemeral_log_preservation_pending and surface via ephemeral_cleanup_pending classification.
- [Phase 05-scoped-ephemeral-mode-and-safety-profiles]: Plan 05-02: provider.ProvisionInput.Mode drives HetznerOwnershipTags mode=ephemeral and ephemeral cloud resource names; persistent default unchanged.
- [Phase 05]: Plan 05-03: docs/safety.md owns canonical Phase 5 safety copy; README/BYO/cloud quickstarts link to it and only repeat sentences required by docs grep contract.
- [Phase 05]: Plan 05-03: Mode-decision warnings (notably the public/fork ephemeral cloud recommendation) merge into ephemeral BYO/cloud completion via mergeWarnings with de-duplication so safety guidance flows to user-visible output.
- [Phase 05]: Plan 05-03: classifyEphemeral prefers observed remote sentinel finalizer status over saved RepositoryState so freshly-completed/TTL-expired ephemeral runners classify as terminal even when state on disk records 'pending'.
- [Phase 05]: Plan 05-03: EphemeralBYO/Cloud RepositoryState fixtures share the deterministic ephemeral runner name runnerkit-owner-repo-ephemeral-20260501t183000 so status/logs/doctor/down/destroy regressions assert exact ephemeral artifact paths.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-02: Side-by-side state backup is taken in Store.Load (not Migrate) so it captures the ORIGINAL raw bytes byte-for-byte before any parsing.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-02: ErrSchemaTooNew refuses to mutate (no backup, no rewrite) when on-disk schema_version is newer than the binary; maps to ExitStateSchemaTooNew = 7.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-02: Lazy update-check honors six silent paths (jsonOutput, $CI, $RUNNERKIT_NO_UPDATE_NOTIFIER, network error, fresh cache, same-version response); uses conditional GET via ETag to avoid re-downloading payloads.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-02: runnerkit upgrade is print-only (channel-detect Homebrew/binary/unknown); reads latest from cache so the command is instantaneous and deterministic in CI.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-02: upgrade-runner refuses without --force when ephemeral FinalizerStatus is waiting/busy/empty; no-ops on completed/ttl_expired; State.RunnerTemplateVersion is bumped only after Apply returns nil.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-03: STATE and CORE codes share docs/troubleshooting/cleanup.md so the file matrix stays at exactly 6 (auth, ssh, bootstrap, github, provider, cleanup) per D-14.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-03: RKD-BOOT-001 reserved; RKD-BOOT-002 maps to runner_version_stale (Plan 06-02), the most-likely runtime warning. Reserved-numbering note in README components-table footnote prevents the gap from looking like a docs bug.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-03: Append (not prepend) RKD codes to remediation arrays so existing tests indexing remediation[0] keep passing; the stable code lives at remediation[len-1].
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-03: errcodes.URL detection rule — keep .md suffix iff base contains /blob/ (GitHub blob URLs) and strip it otherwise (static-site URLs). No separate config flag required.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-03: ErrSchemaTooNew embeds RKD-STATE-004 + default URL via errcodes.FormatLine so err.Error() carries the stable code without wrapper at every call site; errors.Is wiring preserved.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01: Single-tag release pipeline (vX.Y.Z push from upstream) produces 4-platform binaries + sha256 checksums + cosign keyless sigstore bundle + Homebrew Cask formula update via GoReleaser v2 in one workflow run.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01: Cosign signs ONLY checksums.txt (signs[].artifacts: checksum) per D-04, using bundle output (.sigstore.json) over the deprecated v1/v2 .sig+.crt format; bundle is verifiable via cosign verify-blob with cert-identity pinned to the workflow path + tag ref.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01: Tag-only trigger (on: push: tags: ['v*']) with no pull_request/workflow_dispatch sidesteps fork-PR OIDC strip (Pitfall 1). PR validation runs goreleaser check + snapshot build + 4-arch dist assertion + go test ./... on a separate workflow with no signing.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01: README install verification cert-identity URL is string-equal to the workflow file path (.github/workflows/release.yml@refs/tags/${TAG}) embedded in cosign keyless cert; OIDC issuer is locked to https://token.actions.githubusercontent.com (NOT the user-OAuth issuer).
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01: Homebrew Cask publishing requires cross-repo PAT (HOMEBREW_TAP_GITHUB_TOKEN) because default GITHUB_TOKEN cannot push outside the workflow's own repo. Tap repo (salar/homebrew-runnerkit) and PAT secret are documented in docs/release-process.md as one-time maintainer prerequisites; Task 5 is a human-action checkpoint pending external resolution.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01 Task 5 closure: org migrated from 'salar' to 'accidentally-awesome-labs' (commit c359831). All release artifacts, Go module path, 79 source files, troubleshooting docs, and update/errcodes URL constants renamed in one atomic commit. Tap repo accidentally-awesome-labs/homebrew-runnerkit created (public, main, Casks/.gitkeep) and HOMEBREW_TAP_GITHUB_TOKEN PAT secret stored in accidentally-awesome-labs/runnerkit settings. Task 5 resolved 'tap-ready'.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-01 Task 5 closure: HOMEBREW_TAP_GITHUB_TOKEN PAT was pasted into chat during external resolution. Maintainer advised to rotate the token (revoke pasted PAT, generate new one, replace secret) before the public v1.0.0 tag push. PAT value is NOT recorded in any committed file or planning doc.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-04: cmd/_smokebin/ pattern — Go binaries excluded from default builds via _ directory prefix; invoked by shell scripts with go run. Tests still execute when targeted directly.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-04: D-12 gate 1 (empty-project precheck) is a HARD gate before any runnerkit up; refuses if any Server/SSHKey/PrimaryIP/Firewall has a runnerkit- Name prefix. D-12 gate 2 (destroy-verify) polls every 500ms via hcloud.IsError(err, hcloud.ErrorCodeNotFound) until every saved cloud ID returns 404 or RUNNERKIT_SMOKE_TIMEOUT (default 300s) elapses.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-04: scripts/smoke/cloud-end-to-end.sh installs trap cleanup EXIT INT TERM that runs runnerkit destroy --yes if state.json exists (Pitfall 7). State.json is snapshotted to state-after-destroy.json before destroy executes so the verifier can recover IDs even after destroy removes the repo entry. EXIT trap is disarmed on success but INT/TERM stays armed for the verify phase.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-04: hcloud-go v1.59.2 IDs are int (not int64). cloudIDs struct + verifierClient interface use plain int, matching internal/provider/hetzner/client.go and the upstream module surface. Plan body suggested int64; auto-fixed to int.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-04: D-11 enforced by absence — make smoke-live targets are NOT referenced by any .github/workflows/*.yml file. Maintainer must run them manually before tagging; CI environment never holds real PAT or HCLOUD_TOKEN secrets.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-05: host.privilege.password_required preflight result is SeverityWarning (not Failure) so report.Passed() stays true; this preserves Path B fallback reachability (Plan 06-06) without short-circuiting the existing happy path.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-05: download_runner is factored into a shared downloadRunnerCommand(opts) helper consumed by Apply, ApplyEphemeral, AND the integration test, so production drift physically cannot bypass the real-shell test.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-05: bootstrap_failed surfaces 'Remote stderr (commandID): ...' through renderer.Redactor().String(); CommandID extracted from remote.RemoteError via errors.As, fallback 'unknown' when error path predates wrapping.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-05: integration test double-gated by //go:build integration AND RUNNERKIT_INTEGRATION=1 env var so go test ./... and CI ignore it; only make test-integration runs the real-shell path.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-06: sudo password flows via remote.Command.Env['RUNNERKIT_SUDO_PASSWORD'] + RedactArgs append, NOT a new SudoPassword field on remote.Command. Script wrapper reads env via printf|sudo -S so the literal never appears in rendered Script string.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-06: ui.PasswordPrompter is an OPTIONAL capability interface, not a required Prompter method. Callers type-assert via deps.Prompts.(ui.PasswordPrompter), so all existing fake Prompter implementations remain compatible.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-06: byo_host_prepared doctor finding uses ops.SeverityPass (informational/positive) since ops.Severity has only pass/warning/error — no SeverityInfo. Pass is the closest analog and renders correctly via ui.Success in renderDoctorHuman.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-06: visudo runs on the REMOTE host inside RemoteVisudoCheckScript, NOT locally. Atomic-rename pattern (tmp → visudo -cf → mv on success only, exit 21 otherwise) guarantees /etc/sudoers.d/runnerkit-installer is never persisted with malformed content — locks out are physically impossible.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-06: --yes does NOT imply --non-interactive for sudo password input. --yes accepts safe defaults; sudo password is a separate human-input concern that always requires a TTY prompt unless --non-interactive is explicitly set. Locked in by TestUp_SudoPasswordPrompt_YesDoesNotImplyNonInteractive.
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-08: register_runner uses 'sudo su -s /bin/bash - <user> -c "..."' instead of 'sudo -u <user> ./config.sh ...' so a host with only (root) NOPASSWD: ALL or only the byo-prepare scoped sudoers entry is sufficient — closes Bug 3 from 06-GAP-byo-sudo-handling.md (Task F).
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-08: outer 'bash -c "..."' uses double quotes so the OUTER SSH-user shell expands \"$RUNNERKIT_REGISTRATION_TOKEN\" before su invokes the inner shell — preserves env-var indirection, no token leak in rendered Script string. RenderRemoveConfigScript and RenderReconfigureScript intentionally retain 'sudo -u %s' (out of scope per gap doc Task F bounds; recovery flows run after byo-prepare/Path B has established a working sudo path).
- [Phase 06-release-upgrade-docs-and-v1-validation]: Plan 06-08: smoke harness exit codes 3=config.sh missing (Plan 06-05) vs 4=.runner sentinel missing (Plan 06-08) so Plan 06-07 attempt-2 re-smoke can branch on cause. Cloud path verifiably unchanged: internal/provider/hetzner not modified, hetzner package tests stay green.

### Pending Todos

[From .planning/todos/pending/ - ideas captured during sessions]

None yet.

### Blockers/Concerns

[Issues that affect future work]

- Plan 01-02/01-04 validation note: automated fixtures cover GitHub success/denial/redaction/default-path behavior; a controlled live GitHub permission smoke remains recommended before public release.
- Phase 4 validation note: a controlled live Hetzner smoke remains recommended before public release because it creates billable resources and needs real credentials.
- Plan 01-01 validation note: `go run` wraps non-zero binary exits as process exit 1 while printing `exit status 6`; the direct built binary exits 6 for input-required paths.

## Session Continuity

Last session: 2026-05-05T16:31:21.055Z
Stopped at: Completed 06-08-byo-register-runner-runas-fix-PLAN.md
Resume file: None
