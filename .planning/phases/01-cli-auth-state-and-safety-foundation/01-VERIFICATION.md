---
phase: 01-cli-auth-state-and-safety-foundation
verified: 2026-04-29T17:02:00Z
status: passed
score: 14/14 must-haves verified
gaps: []
human_verification: []
reverification: true
source:
  - 01-01-SUMMARY.md
  - 01-02-SUMMARY.md
  - 01-03-SUMMARY.md
  - 01-04-SUMMARY.md
---

# Phase 1: CLI, Auth, State, and Safety Foundation Verification Report

**Phase Goal:** RunnerKit has a runnable CLI foundation that can safely authenticate to GitHub, explain setup prerequisites, persist non-secret state, and redact sensitive data before any real runner install flow depends on it.
**Verified:** 2026-04-29T17:02:00Z
**Status:** passed
**Re-verification:** Yes — after Plan 01-04 closed the production GitHub auth/safety wiring gaps.

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                                                      | Status     | Evidence                                                                                                                                                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Developer can run the local runnerkit binary and see stable help/version output.                                                           | ✓ VERIFIED | `go run ./cmd/runnerkit --help` lists `up`, `state`, and `version`; `go run ./cmd/runnerkit version --json` emits JSON with `redactions_applied:true`.                                                      |
| 2   | Developer can start `runnerkit up` and see prerequisites before state writes.                                                              | ✓ VERIFIED | `TestUpDryRunDisplaysPhaseOneWizard` covers Welcome → Prerequisites → Repo/auth → Safety checks → State preview → Next steps, and dry-run/state tests prove no state writes before save confirmation.       |
| 3   | Automation callers can use `--json`, `--non-interactive`, `--dry-run`, `--yes`, and `--no-color` with deterministic output and exit codes. | ✓ VERIFIED | CLI tests cover JSON-only output, no-TTY input-required exit 6, dry-run, `--yes`, invalid flag exit 2, GitHub auth exit 3, safety exit 4, and state replacement prompts.                                    |
| 4   | Registered secrets never appear raw in human output, JSON output, or logs.                                                                 | ✓ VERIFIED | `internal/redact` masks GitHub, runner, SSH, provider, and machine patterns; renderer JSON sanitizes outputs; new service/default-path tests assert raw GitHub and registration tokens do not appear.       |
| 5   | Developer can target a repo via `--repo` or detected git remote before auth/state.                                                         | ✓ VERIFIED | `internal/github/remote.go` parses explicit and remote targets; `cmd/runnerkit/main.go` injects `github.OSCommandRunner{}`; `normalizeDependencies` defaults nil command runners to `gh.OSCommandRunner{}`. |
| 6   | Developer can reuse existing `gh` authentication with runner-management permission.                                                        | ✓ VERIFIED | `internal/github/service.go` calls `DiscoverAuth` and `CheckRunnerManagementPermission`; `internal/github/auth.go` tries `gh auth token`; registration-token creation is the permission probe.              |
| 7   | Developer receives exact fine-grained-token remediation for missing/insufficient credentials.                                              | ✓ VERIFIED | `TestDefaultGitHubServiceMissingCredentialsFailsClosed` proves default missing auth returns exit 3, JSON code `github_permission_denied`, selected-repo remediation, and no state write.                    |
| 8   | Public/fork-risk repositories are blocked or explicitly overridden.                                                                        | ✓ VERIFIED | `TestDefaultGitHubServiceUsesRealMetadataAndBlocksPublicRepo` proves the default service reads public metadata and blocks with exit 4 plus `WARNING: Public repository risk`.                               |
| 9   | Runner registration/removal tokens are ephemeral secrets and redacted.                                                                     | ✓ VERIFIED | `Client.CreateRegistrationToken`, `Client.CreateRemovalToken`, and `Service.VerifyAuth` tests register fixture tokens with the redactor and never persist/render them raw.                                  |
| 10  | Developer can preview exactly what RunnerKit will save and where.                                                                          | ✓ VERIFIED | `runnerkit up` preview/state tests cover state path, labels, workflow snippet, auth source reference, safety status, and `runner_installed:false`/Phase 1 no-install copy.                                  |
| 11  | Developer can save versioned non-secret foundation state for a confirmed repo.                                                             | ✓ VERIFIED | State integration tests save schema version 1 repository state through the atomic store using explicit fake GitHub dependencies.                                                                            |
| 12  | Developer can inspect saved foundation state without exposing secrets.                                                                     | ✓ VERIFIED | `state show --json` tests assert no forbidden token/private-key/provider-credential terms appear and `redactions_applied:true` is present.                                                                  |
| 13  | Developer sees stable RunnerKit labels and copy-paste `runs-on` guidance.                                                                  | ✓ VERIFIED | Label tests assert `[self-hosted runnerkit runnerkit-owner-repo linux x64 persistent]` and output warns not to use `runs-on: self-hosted` alone.                                                            |
| 14  | Foundation workflows use reusable plan/confirm/apply primitives.                                                                           | ✓ VERIFIED | `internal/workflow` provides stable step IDs, statuses, checkpoints, checklist rendering, and apply behavior.                                                                                               |

**Score:** 14/14 truths verified

### Required Artifacts

| Artifact                     | Expected                                     | Status     | Details                                                                                                              |
| ---------------------------- | -------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                     | Go module definition                         | ✓ VERIFIED | Module exists with Cobra dependency.                                                                                 |
| `cmd/runnerkit/main.go`      | Binary entrypoint                            | ✓ VERIFIED | Executes `cli.NewRootCommand` and injects `github.OSCommandRunner{}` for production subprocesses.                    |
| `internal/cli/root.go`       | Cobra root/version/default dependency wiring | ✓ VERIFIED | Root, flags, version, state/up commands, `gh.OSCommandRunner{}`, and `gh.NewService` default wiring are present.     |
| `internal/cli/up.go`         | Guided setup, auth/safety/state flow         | ✓ VERIFIED | No `defaultGitHubService`, no fake-private metadata assignment, uses injected GitHub service plus `EvaluateSafety`.  |
| `internal/github/service.go` | Real CLI-facing GitHub service               | ✓ VERIFIED | Calls `DiscoverAuth`, `Client.Repository`, and `CheckRunnerManagementPermission`; caches credentials in memory only. |
| `internal/ui/output.go`      | Human/JSON renderer                          | ✓ VERIFIED | Adds `redactions_applied` and sanitizes JSON/human strings.                                                          |
| `internal/redact/redact.go`  | Central redactor                             | ✓ VERIFIED | Contains exact redaction replacements and JSON field sanitization.                                                   |
| `internal/github/remote.go`  | Repo/remote parser                           | ✓ VERIFIED | Parses owner/name, HTTPS, SCP SSH, and SSH URLs.                                                                     |
| `internal/github/auth.go`    | gh/env auth discovery                        | ✓ VERIFIED | Calls `gh auth token`, registers token, supports `RUNNERKIT_GITHUB_TOKEN`.                                           |
| `internal/github/client.go`  | REST client wrapper                          | ✓ VERIFIED | Sends `Accept` and `X-GitHub-Api-Version`; supports repo metadata and runner token endpoints.                        |
| `internal/github/tokens.go`  | Runner token permission check                | ✓ VERIFIED | Provides registration/removal token wrappers and permission-check helper.                                            |
| `internal/github/safety.go`  | Public/fork safety gate                      | ✓ VERIFIED | Implements `public_repo_risk`, `fork_risk`, and `--allow-public-repo-risk` decisions.                                |
| `internal/state/*`           | Versioned state/config/store                 | ✓ VERIFIED | Schema v1, migrations, atomic 0600 writes, `.runnerkit/config.yaml`, raw-secret validation.                          |
| `internal/labels/labels.go`  | Stable labels/snippet                        | ✓ VERIFIED | Produces runner name, labels, snippet, and warning.                                                                  |
| `internal/workflow/plan.go`  | Plan/checkpoint/apply primitives             | ✓ VERIFIED | Stable IDs and checklist/apply behavior.                                                                             |
| `internal/cli/state.go`      | `state show` command                         | ✓ VERIFIED | Reads saved state and renders redacted human/JSON output.                                                            |

### Key Link Verification

| From                         | To                           | Via                                                       | Status  | Details                                                                                        |
| ---------------------------- | ---------------------------- | --------------------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------- |
| `cmd/runnerkit/main.go`      | `internal/github/auth.go`    | `Dependencies.CommandRunner` = `github.OSCommandRunner{}` | ✓ WIRED | Production binary can run `gh` and `git remote` subprocesses.                                  |
| `internal/cli/root.go`       | `internal/github/service.go` | `normalizeDependencies` uses `gh.NewService`              | ✓ WIRED | Default CLI path uses the real GitHub service when tests do not inject fakes.                  |
| `internal/github/service.go` | `internal/github/auth.go`    | `DiscoverAuth`                                            | ✓ WIRED | Service discovers `gh` or `RUNNERKIT_GITHUB_TOKEN` credentials and stores them in memory only. |
| `internal/github/service.go` | `internal/github/client.go`  | `Client.Repository(ctx, repo)`                            | ✓ WIRED | Service reads GitHub metadata instead of fabricating `Private=true`.                           |
| `internal/github/service.go` | `internal/github/tokens.go`  | `CheckRunnerManagementPermission`                         | ✓ WIRED | Service uses registration-token creation as the runner-management permission check.            |
| `internal/cli/up.go`         | `internal/github/safety.go`  | `gh.EvaluateSafety(repo, ...)`                            | ✓ WIRED | Safety decisions use metadata returned by the GitHub service.                                  |
| `internal/ui/output.go`      | `internal/redact/redact.go`  | sanitize before write                                     | ✓ WIRED | Renderer redacts human and JSON output.                                                        |
| `internal/cli/up.go`         | `internal/state/store.go`    | state save                                                | ✓ WIRED | State is saved only after auth, metadata, safety, and confirmation gates pass.                 |
| `internal/cli/up.go`         | `internal/labels/labels.go`  | `labels.Build`                                            | ✓ WIRED | Preview and saved state include stable labels and workflow snippet.                            |

### Data-Flow Trace

| Artifact                     | Data Variable         | Source                                    | Produces Real Data | Status    |
| ---------------------------- | --------------------- | ----------------------------------------- | ------------------ | --------- |
| `internal/cli/up.go`         | repo target           | `github.ParseRepo` / `ResolveTarget`      | Yes                | ✓ FLOWING |
| `internal/github/service.go` | auth source           | `DiscoverAuth`                            | Yes                | ✓ FLOWING |
| `internal/github/service.go` | runner permission     | `CheckRunnerManagementPermission`         | Yes                | ✓ FLOWING |
| `internal/github/service.go` | repo metadata safety  | `Client.Repository`                       | Yes                | ✓ FLOWING |
| `internal/cli/up.go`         | public/fork gate      | `EvaluateSafety` on real service metadata | Yes                | ✓ FLOWING |
| `internal/cli/up.go`         | state path/state save | `state.NewStore` / `SaveRepository`       | Yes                | ✓ FLOWING |
| `internal/cli/state.go`      | saved state           | `state.Store.GetRepository`               | Yes                | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior                            | Command                                                                                      | Result                                             | Status |
| ----------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------- | ------ |
| Full suite                          | `go test ./... && go vet ./...`                                                              | Passed                                             | ✓ PASS |
| Help output                         | `go run ./cmd/runnerkit --help`                                                              | Listed `up`, `state`, and `version`                | ✓ PASS |
| Version JSON                        | `go run ./cmd/runnerkit version --json`                                                      | JSON only, `redactions_applied:true`               | ✓ PASS |
| Default missing auth fails closed   | `go test ./internal/cli -run TestDefaultGitHubServiceMissingCredentialsFailsClosed -v`       | Passed; exit code 3 and no state write asserted    | ✓ PASS |
| Default real metadata safety gate   | `go test ./internal/cli -run TestDefaultGitHubServiceUsesRealMetadataAndBlocksPublicRepo -v` | Passed; registration + metadata endpoints asserted | ✓ PASS |
| Real GitHub service wiring          | `go test ./internal/github -run TestService -v`                                              | Passed; auth header, metadata, redaction asserted  | ✓ PASS |
| No fake CLI default                 | `grep -R "defaultGitHubService" internal/cli/up.go`                                          | No matches                                         | ✓ PASS |
| No fake-private production metadata | `grep -R "repo.Private = true" internal/cli/up.go`                                           | No matches                                         | ✓ PASS |
| Real default service                | `grep -R "gh.NewService" internal/cli/root.go`                                               | Match                                              | ✓ PASS |
| Real OS command runner              | `grep -R "github.OSCommandRunner{}" cmd/runnerkit/main.go`                                   | Match                                              | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan                | Description                                                      | Status      | Evidence                                                                                                                     |
| ----------- | -------------------------- | ---------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------- |
| CLI-01      | 01-01, 01-03, 01-04        | Install/run CLI binary                                           | ✓ SATISFIED | Runnable Go/Cobra CLI, help/version/state/up commands pass; production command runner exists.                                |
| CLI-02      | 01-01, 01-02, 01-03, 01-04 | Guided setup explains choices/prereqs before changes             | ✓ SATISFIED | Up flow previews steps/state with deterministic tests and now fails closed through real auth/safety defaults.                |
| GH-01       | 01-02, 01-04               | Authenticate for repo with minimum runner-management permissions | ✓ SATISFIED | Default CLI path uses `gh.NewService`, `DiscoverAuth`, `Client.Repository`, and registration-token permission checks.        |
| STATE-01    | 01-03, 01-04               | Versioned local state/config                                     | ✓ SATISFIED | Schema v1, atomic store, state show, labels, cleanup/provider/machine placeholders; missing auth regression writes no state. |
| STATE-02    | 01-01, 01-02, 01-03, 01-04 | Redact secrets/tokens from state/logs/output                     | ✓ SATISFIED | Redactor, renderer, client token registration, service redaction tests, state validation, and output leak checks pass.       |

### Resolved Gaps

| Previous Gap                                                                                            | Resolution                                                                                                                           | Evidence                                                                                   |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| Production `runnerkit up` used fake-permitted GitHub auth/metadata.                                     | Removed `defaultGitHubService`; `normalizeDependencies` now defaults to `gh.NewService`.                                             | `internal/cli/up.go` has no fake service; `internal/cli/root.go` contains `gh.NewService`. |
| Production git remote auto-detection could not run because no OS command runner was injected/defaulted. | `cmd/runnerkit/main.go` injects `github.OSCommandRunner{}` and normalization defaults nil command runners to `gh.OSCommandRunner{}`. | Grep and `TestNormalizeDependenciesDefaultsToRealGitHubAndOSCommandRunner`.                |
| Public/fork safety used fake private metadata.                                                          | Real service reads metadata via `Client.Repository`; default-path regression tests return public metadata and assert safety exit 4.  | `TestDefaultGitHubServiceUsesRealMetadataAndBlocksPublicRepo`.                             |

### Human Verification Required

None. The previous blockers were code wiring gaps and are covered by automated tests and code inspection. A controlled live GitHub permission smoke remains useful before a public release, but it does not block Phase 1 because the production wiring and permission-check behavior are now verified with deterministic API fixtures.

### Final Assessment

Phase 1 now satisfies its goal. RunnerKit has a runnable CLI foundation, real fail-closed GitHub authentication and runner-management permission checks in the production default path, real repository metadata safety gating, versioned non-secret state, stable labels, and centralized redaction. No phase-level gaps remain.

---

_Verified: 2026-04-29T17:02:00Z_
_Verifier: Pi inline verifier (gsd-verifier fallback)_
