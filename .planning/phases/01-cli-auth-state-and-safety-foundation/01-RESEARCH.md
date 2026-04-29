# Phase 1 Research: CLI, Auth, State, and Safety Foundation

**Researched:** 2026-04-29  
**Status:** Ready for planning  
**Confidence:** Medium-high  
**Scope:** Phase 1 only - CLI foundation, GitHub auth foundation, versioned non-secret state/config, redaction, labels/naming, and idempotent workflow primitives. No BYO runner installation, cloud provisioning, diagnostics/cleanup implementation, or ephemeral execution.

## Verdict

Plan Phase 1 as a greenfield Go/Cobra foundation with redaction, output contracts, auth boundaries, and versioned state established before any later runner install path depends on them. The safest execution order is:

1. Create the CLI/module skeleton, output/prompt abstractions, `runnerkit up` guided foundation wizard, automation flags, exit codes, and minimal redacted logging first.
2. Add GitHub repo resolution and least-privilege auth/permission checks behind a dedicated adapter that never stores durable tokens and discards runner registration/removal tokens immediately.
3. Add versioned JSON state/project config, runner name/label conventions, plan/confirm/apply primitives, fake adapters, and end-to-end validation harnesses.

This preserves the Phase 1 promise: a developer can run `runnerkit`, understand prerequisites, verify GitHub auth for a selected repo, preview/save foundation state, and trust that sensitive values are redacted, while the CLI remains honest that no runner is installed until Phase 2.

## Findings

### 1. Greenfield Go CLI shape

- No application source tree exists yet (`go.mod`, Go packages, app README, and tests are absent), so Phase 1 must include module initialization, package boundaries, command routing, test conventions, and local-run/install acceptance checks for `runnerkit`.
- Use Go as the default implementation language based on project research and `GEMINI.md`: single static-ish binary distribution, fast startup, strong HTTP/SSH/process support, and straightforward CI/release path.
- Use Cobra for command routing unless implementation discovers a blocking issue. Keep the command layer thin: parse flags, render prompts/output, and call core services.
- Establish an executable layout suitable for later growth:
  - `cmd/runnerkit/main.go` for binary entrypoint.
  - `internal/cli` for Cobra commands, flags, command dependencies, exit handling.
  - `internal/ui` or `internal/output` for terminal renderer, prompt abstraction, JSON renderer, color/ASCII modes.
  - `internal/redact` for sensitive value registration and sanitization.
  - `internal/github` for auth/repo/API adapter interfaces and implementation.
  - `internal/state` for project config, local state, migrations, atomic writes.
  - `internal/labels` for runner labels and workflow hint formatting.
  - `internal/workflow` for plan/checkpoint/apply primitives.
  - `internal/testsupport` for fake adapters and golden output helpers.

### 2. Terminal UX must be a contract, not ad hoc output

- The approved UI-SPEC is mandatory for all Phase 1 human and JSON output.
- `runnerkit up` should implement the first-run guided order: Welcome -> prerequisites -> repo/auth -> safety checks -> state preview -> next steps.
- Human output should use static, log-friendly lines rather than mandatory full-screen TUI redraws. A richer TUI/prompt library can be used, but automation and tests require a prompt/output abstraction.
- Default prompt behavior:
  - Interactive only when stdin and stdout are TTYs.
  - Auto-detected repo requires explicit confirmation before auth/state actions apply to it.
  - Any state write shows a plan/checklist and asks `Save this foundation state? [y/N]`; default no.
  - No-TTY prompts exit code `6` unless all required flags and `--yes` are supplied.
- Phase 1 should support `--repo owner/name`, `--yes`, `--json`, `--non-interactive`, `--dry-run`, and `--no-color` from the beginning.
- Exit codes should follow UI-SPEC: `0` success, `1` unexpected error, `2` invalid input/flags, `3` GitHub auth/permission failure, `4` safety gate blocked, `5` state read/write failure, `6` interactive input required, `130` canceled.

### 3. Redaction needs to exist before auth work

- Redaction cannot wait until the final state plan if GitHub auth work happens earlier. Plan 01-01 should introduce the central redaction package/logger/output sanitizer or explicitly make it a prerequisite for 01-02.
- Sensitive value types to support now:
  - GitHub durable token/PAT -> `<redacted:github-token>`
  - Runner registration token -> `<redacted:runner-registration-token>`
  - Runner removal token -> `<redacted:runner-removal-token>`
  - SSH private key material -> `<redacted:ssh-private-key>`
  - Provider/cloud credential -> `<redacted:provider-credential>`
  - Sensitive machine/provider reference -> `<redacted:machine-ref>` when rendered in diagnostics.
- Redaction must apply to human output, JSON, structured logs, errors, API fixture failures, state previews, and debug/test output. There is no unredacted debug mode in Phase 1.
- Tests should table-drive exact input/output pairs and include edge cases where sensitive values appear inside URLs, command strings, JSON payloads, and multi-line output.

### 4. GitHub auth and repo targeting should fail closed

- Preferred auth sequence:
  1. Resolve target repo from `--repo owner/name` or parse local git remote.
  2. Confirm detected repo interactively before auth/state actions.
  3. Try to reuse existing `gh` auth (`gh auth token`) without persisting the token in RunnerKit state.
  4. If `gh` is missing/insufficient, guide the user to a selected-repository fine-grained token path with exact permission instructions.
  5. Optionally accept `RUNNERKIT_GITHUB_TOKEN`/explicit token input only as a reference/input path, not durable saved state.
- Implementation-time docs verification is still required for exact GitHub fine-grained permissions. Current best planning assumption: selected repository access plus repository Administration read/write and Metadata read is required for repository self-hosted runner registration/removal endpoints.
- Permission checks should call the same capability needed later: repository runner registration token creation, plus repo metadata/visibility. If the token lacks permission, return exit code `3` with exact remediation.
- Registration/removal token adapter should model short-lived token responses but must discard raw token material after use/checks. Never save tokens in local state or write them to fixtures/logs.
- Use explicit GitHub REST API headers (`Accept: application/vnd.github+json`, API version header) and wrap go-github/direct REST behind RunnerKit's own interfaces so endpoint behavior changes are localized.
- GitHub auth should disclose when a reused `gh` credential is broader than minimum scope and offer fine-grained PAT guidance for minimum-permission users.

### 5. Public/fork-risk safety gate belongs in Phase 1

- Phase 1 does not install a persistent runner, but later persistent setup must not start without a foundation safety model.
- Fetch repo visibility/fork metadata and surface a safety status in the `runnerkit up` preview.
- For public repositories or obvious fork-risk contexts, fail closed or require an explicit danger override such as `--allow-public-repo-risk` with typed confirmation in interactive mode.
- Copy should be precise: persistent self-hosted runners are intended for trusted private repos; public/fork/untrusted workflows need GitHub-hosted runners or a future safer ephemeral profile.
- Deeper workflow file inspection can be deferred; obvious repo visibility gating should ship now.

### 6. State/config should be versioned, human-debuggable, and secret-free

- Recommended Phase 1 storage:
  - Optional project config: `.runnerkit/config.yaml` or `.runnerkit/config.json` containing safe repeatable defaults only.
  - Mandatory user-local state: OS app state dir, for example `~/.local/state/runnerkit/state.json` on Linux/macOS-compatible systems, with platform abstraction.
- Start with versioned JSON state for simplicity and inspectability. SQLite can be deferred until multiple runners/events/concurrency justify it.
- State schema must include `schema_version` from day one plus migration hooks/tests.
- Fields to support or reserve now:
  - Repo scope: owner, name, host, full name.
  - Auth reference only: `gh`, `fine-grained-token`, or `env`; no durable token material.
  - Runner identity: planned runner name, labels, mode (`persistent` default/preview), OS, arch, capability labels.
  - Machine path: provider type, SSH/user-owned placeholder refs, install path placeholders.
  - Provider IDs/cleanup metadata: optional future fields so cloud/BYO cleanup can be modeled without state migration shock.
  - Safety status and accepted risk override metadata.
  - Version metadata: RunnerKit CLI version, state schema version, future runner/service template versions.
- Writes should be atomic: create parent dir with `0700`, write temp file with `0600`, fsync where practical, and rename. Read/write failures should produce exit code `5`.
- Replacement of existing state for the same repo requires an explicit destructive confirmation: `Type replace {owner/repo} to overwrite the existing RunnerKit state for this repository.`

### 7. Labels and runner names should be stable now

- Do not ever recommend `runs-on: self-hosted` alone.
- Phase 1 should establish a stable recommended label set and snippet even before registration exists:
  ```yaml
  runs-on:
    [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]
  ```
- Labels should be deterministic and slugged:
  - `self-hosted` because GitHub requires/uses it for self-hosted routing.
  - `runnerkit` to distinguish managed runners.
  - `runnerkit-<owner>-<repo>` or another stable repo-scoped slug to avoid cross-repo confusion.
  - OS/arch labels (`linux`, `x64` or `arm64`) based on future machine target/default.
  - Mode label (`persistent` for default future path, later `ephemeral`).
- Runner names should be predictable and collision-aware, e.g. `runnerkit-<owner>-<repo>-<short-suffix>` or `runnerkit-<repo-slug>-<machine-slug>`. Exact scheme can be decided during planning but must support cleanup/status and duplicate detection later.

### 8. Idempotent primitives prevent later BYO/cloud scripts from becoming fragile

- Phase 1 should define reusable workflow primitives even if only used for foundation state:
  - Plan/checklist preview with typed actions and warnings.
  - Confirm/apply separation.
  - Step IDs, statuses, retryability, and checkpoint metadata.
  - Typed user-fixable errors with codes/remediation.
  - Dry-run and JSON rendering of plans/results.
  - Fake GitHub/state/prompt adapters for tests.
- This keeps later BYO/cloud work from embedding ad hoc shell and API logic directly in CLI commands.

### 9. Testing strategy must be part of each plan

- Phase 1 cannot rely on manual checks only. It should establish Go test infrastructure immediately.
- Required automated test categories:
  - CLI command routing/help/version and binary local run.
  - Wizard step ordering and prerequisite copy golden tests.
  - No-TTY behavior and exit code `6` for missing flags.
  - JSON success/error shape with no ANSI/prose/secrets.
  - Git remote parsing for SSH/HTTPS GitHub URLs and invalid remotes.
  - Repo confirmation flow via fake prompt adapter.
  - `gh` auth discovery via fake command runner and fallback instructions.
  - GitHub permission success/denial with `httptest` fixtures.
  - Registration/removal token response redaction/discard tests.
  - State schema migration, atomic write, permissions, replacement confirmation, and no-token persistence tests.
  - Label/name builder output tests.
  - Redaction table tests across human/JSON/log/error paths.

## Risks and Planning Implications

| Risk                                                                          | Planning implication                                                                                                  |
| ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| Exact GitHub fine-grained PAT permissions may differ from planning assumption | Make a task explicitly verify docs/API behavior and encode permission copy/tests.                                     |
| Reused `gh` auth may be broader than minimum                                  | Disclose auth source and provide fine-grained token alternative instead of pretending it is least-privilege.          |
| Registration token check creates a sensitive token                            | Treat token creation as ephemeral; tests must prove it is never persisted or printed.                                 |
| Rich terminal UX can break automation                                         | Abstract prompts/output; make `--json`, `--non-interactive`, `--dry-run`, `NO_COLOR`, `TERM=dumb` first-class.        |
| State schema can overfit future phases                                        | Include reserved structured fields for future BYO/cloud cleanup, but keep Phase 1 writes minimal and migration-ready. |
| Safety gate may miss workflow-level fork risk                                 | Gate public repo/metadata now; leave deep workflow validation for later roadmap phases.                               |

## Recommended Plan Split

### Plan 01-01: CLI skeleton, output, prompts, wizard scaffold, redaction minimum

Build the executable foundation:

- `go.mod`, `cmd/runnerkit/main.go`, package skeleton, tests, and local run/install docs.
- Cobra root command and `up`, `version`, `state show` placeholder if useful.
- `runnerkit up` guided scaffold with Welcome -> prerequisites -> repo/auth -> safety checks -> state preview -> next steps, using fakes where GitHub/state are not yet implemented.
- Output renderer for human/JSON, terminal width/color/ASCII, exit-code mapping, `--repo`, `--yes`, `--json`, `--non-interactive`, `--dry-run`, `--no-color` flags.
- Prompt abstraction and no-TTY behavior.
- Minimal central redaction package/logger used by all output paths before auth work begins.

### Plan 01-02: GitHub repo resolution, auth, permission checks, token adapter fixtures

Build the GitHub foundation:

- Git remote parser and explicit repo confirmation.
- `gh` token discovery via command runner without persistence.
- Fine-grained token fallback instructions and env/input auth source handling.
- GitHub API adapter with explicit REST headers and `httptest` fixtures.
- Repo metadata visibility/fork-risk check.
- Runner registration/removal token adapter for permission checks, with token discard and redaction tests.
- Fail-closed typed errors and exact remediation copy.

### Plan 01-03: Versioned state/config, labels/names, plan/checkpoint primitives, E2E dry-run/save

Build durable foundation primitives:

- Optional project config schema and mandatory user-local state schema.
- Versioned JSON state, migrations, atomic writes, permissions, existing-state replacement safety.
- Label/name builder and workflow snippet formatter.
- Plan/checklist/confirm/apply primitives with checkpoint metadata and fake adapters.
- End-to-end tests for dry-run/no-write, save-with-yes, state show, redaction across persisted previews/output, and Phase 1 completion copy that says `runner_installed: false`.

## Validation Architecture

Phase 1 validation should use Go's native test stack plus golden fixtures and `httptest`. The validation goal is to sample every critical contract before later runner-install phases depend on it.

### Test infrastructure

- Framework: Go `testing` package with standard `go test ./...`.
- HTTP fixtures: `net/http/httptest` for GitHub API success, permission denial, public repo, private repo, token response, and API error cases.
- Golden fixtures: `testdata/*.golden` for human wizard output, JSON success, JSON error, fine-grained token instructions, and safety warnings.
- Fake adapters: prompt adapter, command runner for `gh`, GitHub client interface, state store filesystem rooted at `t.TempDir()`, terminal capability detector.
- CI-compatible commands:
  - Quick: `go test ./...`
  - Full: `go test ./... && go vet ./...`
- Max expected feedback latency during Phase 1: under 60 seconds after Go module dependencies are downloaded; under 15 seconds for warm local runs.

### Requirement-to-validation mapping

| Requirement | Validation signals                                                                                                                                                                                                                                              |
| ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CLI-01      | `go build ./cmd/runnerkit` succeeds; `go run ./cmd/runnerkit --help` exits 0 and contains `runnerkit`; `go run ./cmd/runnerkit version --json` emits JSON only.                                                                                                 |
| CLI-02      | Wizard golden test proves the order `Welcome`, `Prerequisites`, `Repo/auth`, `Safety checks`, `State preview`, `Next steps`; no-TTY missing flags exits 6; state write prompt defaults no.                                                                      |
| GH-01       | GitHub adapter tests prove `gh` auth source works, insufficient permission returns exit code 3/remediation, fine-grained token instructions mention selected repo and required runner-management permission, registration token is not persisted/printed.       |
| STATE-01    | State schema tests prove `schema_version`, repo scope, runner name, labels, machine path placeholders, provider IDs/cleanup metadata, and atomic write permissions are present; migration hook test exists.                                                     |
| STATE-02    | Redaction table tests prove GitHub tokens, runner registration/removal tokens, SSH private keys, provider credentials, and machine refs are replaced in human output, JSON, logs, errors, and state previews; persisted state contains no raw `token` material. |

### Sampling plan

- After every task commit: run `go test ./...` once Go module exists. Before module exists, run `test -f go.mod && test -f cmd/runnerkit/main.go` for Wave 0 setup tasks.
- After each plan: run `go test ./... && go vet ./...`.
- Before phase verification: run `go test ./... && go vet ./...` plus grep checks for no raw token fields in committed state fixtures.
- No plan should contain three consecutive implementation tasks without an automated `go test ./...` or targeted `go test ./internal/<pkg>` verification.

### Manual-only checks

Manual checks should be minimal:

- Interactive terminal feel: run `go run ./cmd/runnerkit up` in a TTY and confirm the flow is readable at 80 columns, prompts default safely, Ctrl-C prints `Canceled; no changes made.`
- Real GitHub permission behavior may require one controlled manual smoke test after automated fixtures pass, because exact GitHub token permissions can differ from docs. This should not store or print tokens and should use a test repository.

## Evidence

- `GEMINI.md` directs implementation toward a Go CLI, likely Cobra, with GitHub REST/go-github, local versioned state, and strict redaction.
- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-CONTEXT.md` records decisions for richer CLI wizard, `runnerkit up`, plan/confirm before mutation, `gh` auth first, fine-grained token fallback, repo confirmation, fail-closed permissions, public repo safety gate, and optional project config plus mandatory local state.
- `.planning/phases/01-cli-auth-state-and-safety-foundation/01-UI-SPEC.md` defines terminal UX, output, JSON, no-TTY, redaction, accessibility, and copywriting contracts.
- `.planning/ROADMAP.md` maps Phase 1 to three plans and excludes BYO install/cloud/diagnostics/cleanup/ephemeral implementation.
- `.planning/REQUIREMENTS.md` maps Phase 1 to CLI-01, CLI-02, GH-01, STATE-01, and STATE-02.
- `.planning/research/SUMMARY.md`, `STACK.md`, and `ARCHITECTURE.md` recommend local-first Go CLI, thin command layer, adapters, short-lived tokens, fake/test tooling, and versioned state.
- `.planning/research/PITFALLS.md` identifies Phase 1 pitfalls: registration token lifecycle, over-broad GitHub auth, secret leakage, unsafe generic labels, and public/untrusted persistent runner risk.
- Repository inspection found no existing source tree or project skill files to preserve.

## RESEARCH COMPLETE
