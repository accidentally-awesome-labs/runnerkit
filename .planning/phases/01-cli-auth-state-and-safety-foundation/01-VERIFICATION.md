---
phase: 01-cli-auth-state-and-safety-foundation
verified: 2026-04-29T02:40:30Z
status: gaps_found
score: 10/14 must-haves verified
gaps:
  - truth: "Developer can authenticate RunnerKit for a GitHub repository using only the permissions required for runner management."
    status: failed
    reason: "The production CLI default GitHub service is fake-permitted; it does not call gh auth token, RUNNERKIT_GITHUB_TOKEN, GitHub repository metadata, or the runner registration-token permission check."
    artifacts:
      - path: "internal/cli/up.go"
        issue: "defaultGitHubService.Repository forces Private=true and VerifyAuth returns OK without any credential or API check."
      - path: "internal/cli/root.go"
        issue: "normalizeDependencies wires defaultGitHubService instead of the real internal/github auth/client/token adapter."
    missing:
      - "Wire default CLI GitHub service to internal/github.DiscoverAuth, Client.Repository, and CheckRunnerManagementPermission."
      - "Keep tests deterministic by injecting fake GitHub services only in tests, not as the production default."
  - truth: "Developer can target a GitHub repository via --repo or detected git remote and must confirm the target before auth/state actions apply."
    status: partial
    reason: "--repo works, and git-remote detection is unit-tested through an injected CommandRunner, but the actual binary does not provide an OSCommandRunner so git remote auto-detection cannot run in production."
    artifacts:
      - path: "cmd/runnerkit/main.go"
        issue: "Dependencies omit CommandRunner."
      - path: "internal/cli/root.go"
        issue: "normalizeDependencies does not default CommandRunner to github.OSCommandRunner."
    missing:
      - "Default CommandRunner to github.OSCommandRunner for the real CLI."
  - truth: "Public or fork-risk repositories are blocked or require an explicit danger override before future persistent setup can proceed."
    status: partial
    reason: "Safety evaluation itself is implemented and tested, but production CLI metadata is fake private, so actual public/fork repositories are not detected."
    artifacts:
      - path: "internal/cli/up.go"
        issue: "defaultGitHubService.Repository sets repo.Private=true instead of reading GitHub metadata."
    missing:
      - "Use real GitHub repository metadata in the default CLI path before EvaluateSafety."
human_verification: []
---

# Phase 1: CLI, Auth, State, and Safety Foundation Verification Report

**Phase Goal:** RunnerKit has a runnable CLI foundation that can safely authenticate to GitHub, explain setup prerequisites, persist non-secret state, and redact sensitive data before any real runner install flow depends on it.  
**Verified:** 2026-04-29T02:40:30Z  
**Status:** gaps_found  
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                                                      | Status     | Evidence                                                                                                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Developer can run the local runnerkit binary and see stable help/version output.                                                           | ✓ VERIFIED | `go run ./cmd/runnerkit --help` lists `up`; `version --json` emits JSON with `redactions_applied:true`.                                                     |
| 2   | Developer can start `runnerkit up` and see prerequisites before state writes.                                                              | ✓ VERIFIED | `up --dry-run --repo owner/repo --yes --no-color` renders Welcome/Prerequisites/Repo/auth/Safety/State/Next steps and writes no state file.                 |
| 3   | Automation callers can use `--json`, `--non-interactive`, `--dry-run`, `--yes`, and `--no-color` with deterministic output and exit codes. | ✓ VERIFIED | CLI tests cover JSON-only output, no-TTY input-required exit 6, dry-run, and `--yes`; spot checks passed.                                                   |
| 4   | Registered secrets never appear raw in human output, JSON output, or logs.                                                                 | ✓ VERIFIED | `internal/redact` masks GitHub, runner, SSH, provider, and machine patterns; renderer JSON sanitizes outputs.                                               |
| 5   | Developer can target a repo via `--repo` or detected git remote before auth/state.                                                         | ✗ PARTIAL  | `--repo` works; git remote parsing and confirmation are tested, but the binary has no default `CommandRunner`, so production auto-detection cannot execute. |
| 6   | Developer can reuse existing `gh` authentication with runner-management permission.                                                        | ✗ FAILED   | `DiscoverAuth` and token-check adapters exist, but `runnerkit up` production default uses fake `defaultGitHubService` and never calls them.                 |
| 7   | Developer receives exact fine-grained-token remediation for missing/insufficient credentials.                                              | ✗ PARTIAL  | Permission-denied test with injected service returns exit 3 and remediation; production default never detects missing/insufficient auth.                    |
| 8   | Public/fork-risk repositories are blocked or explicitly overridden.                                                                        | ✗ PARTIAL  | `EvaluateSafety` and CLI tests pass with injected metadata; production default forces `Private=true`, bypassing real metadata.                              |
| 9   | Runner registration/removal tokens are ephemeral secrets and redacted.                                                                     | ✓ VERIFIED | Client token tests register fixture registration/removal tokens with the redactor; rendered output removes fixture token text.                              |
| 10  | Developer can preview exactly what RunnerKit will save and where.                                                                          | ✓ VERIFIED | `runnerkit up --dry-run` shows state path, labels, workflow snippet, auth source, safety status, and no-install copy.                                       |
| 11  | Developer can save versioned non-secret foundation state for a confirmed repo.                                                             | ✓ VERIFIED | `runnerkit up --repo owner/repo --yes --json` with temp `RUNNERKIT_STATE_DIR` writes `state.json` schema version 1.                                         |
| 12  | Developer can inspect saved foundation state without exposing secrets.                                                                     | ✓ VERIFIED | `state show --json` emits no raw `token`, `registration_token`, `remove_token`, `private_key`, or `provider_credential` terms.                              |
| 13  | Developer sees stable RunnerKit labels and copy-paste `runs-on` guidance.                                                                  | ✓ VERIFIED | Labels equal `[self-hosted runnerkit runnerkit-owner-repo linux x64 persistent]`; output includes the self-hosted-alone warning.                            |
| 14  | Foundation workflows use reusable plan/confirm/apply primitives.                                                                           | ✓ VERIFIED | `internal/workflow` provides stable step IDs, statuses, checkpoints, checklist, and `Apply`.                                                                |

**Score:** 10/14 truths verified

### Required Artifacts

| Artifact                    | Expected                             | Status     | Details                                                                                           |
| --------------------------- | ------------------------------------ | ---------- | ------------------------------------------------------------------------------------------------- |
| `go.mod`                    | Go module definition                 | ✓ VERIFIED | Module exists with Cobra dependency.                                                              |
| `cmd/runnerkit/main.go`     | Binary entrypoint                    | ⚠ PARTIAL  | Executes `cli.NewRootCommand`; does not inject `github.OSCommandRunner` for git remote detection. |
| `internal/cli/root.go`      | Cobra root/version wiring            | ⚠ PARTIAL  | Root, flags, version, and state commands wired; defaults fake GitHub service.                     |
| `internal/cli/up.go`        | Guided setup, auth/safety/state flow | ⚠ PARTIAL  | CLI flow works with fake default and injected tests; production auth/metadata are not wired.      |
| `internal/ui/output.go`     | Human/JSON renderer                  | ✓ VERIFIED | Adds `redactions_applied` and sanitizes JSON/human strings.                                       |
| `internal/redact/redact.go` | Central redactor                     | ✓ VERIFIED | Contains exact redaction replacements and JSON field sanitization.                                |
| `internal/github/remote.go` | Repo/remote parser                   | ✓ VERIFIED | Parses owner/name, HTTPS, SCP SSH, and SSH URLs.                                                  |
| `internal/github/auth.go`   | gh/env auth discovery                | ✓ VERIFIED | Calls `gh auth token`, registers token, supports `RUNNERKIT_GITHUB_TOKEN`.                        |
| `internal/github/client.go` | REST client wrapper                  | ✓ VERIFIED | Sends `Accept` and `X-GitHub-Api-Version`; supports repo metadata and runner token endpoints.     |
| `internal/github/tokens.go` | Runner token permission check        | ✓ VERIFIED | Provides registration/removal token wrappers and permission-check helper.                         |
| `internal/github/safety.go` | Public/fork safety gate              | ✓ VERIFIED | Implements `public_repo_risk`, `fork_risk`, and `--allow-public-repo-risk` decisions.             |
| `internal/state/*`          | Versioned state/config/store         | ✓ VERIFIED | Schema v1, migrations, atomic 0600 writes, `.runnerkit/config.yaml`, raw-secret validation.       |
| `internal/labels/labels.go` | Stable labels/snippet                | ✓ VERIFIED | Produces runner name, labels, snippet, and warning.                                               |
| `internal/workflow/plan.go` | Plan/checkpoint/apply primitives     | ✓ VERIFIED | Stable IDs and checklist/apply behavior.                                                          |
| `internal/cli/state.go`     | `state show` command                 | ✓ VERIFIED | Reads saved state and renders redacted human/JSON output.                                         |

### Key Link Verification

| From                        | To                                      | Via                    | Status      | Details                                                                                                          |
| --------------------------- | --------------------------------------- | ---------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------- |
| `cmd/runnerkit/main.go`     | `internal/cli/root.go`                  | `cli.NewRootCommand`   | ✓ WIRED     | Entrypoint executes root command.                                                                                |
| `internal/cli/up.go`        | `internal/ui/output.go`                 | shared renderer        | ✓ WIRED     | Up flow uses renderer for human/JSON/error/warning output.                                                       |
| `internal/ui/output.go`     | `internal/redact/redact.go`             | sanitize before write  | ✓ WIRED     | Renderer calls redactor for strings and JSON bytes.                                                              |
| `internal/cli/up.go`        | `internal/github/remote.go`             | target resolution      | ⚠ PARTIAL   | Calls `github.ResolveTarget`, but production dependencies omit OS command runner for auto-detection.             |
| `internal/cli/up.go`        | `internal/github/auth.go` / `tokens.go` | auth permission check  | ✗ NOT WIRED | Default CLI path calls fake `VerifyAuth`; real DiscoverAuth/Client/token check are unused by production default. |
| `internal/github/tokens.go` | `internal/redact/redact.go`             | token registration     | ✓ WIRED     | Client registers registration/removal tokens with redactor.                                                      |
| `internal/github/safety.go` | `internal/cli/up.go`                    | safety gate            | ⚠ PARTIAL   | Up calls `EvaluateSafety`, but default metadata is fake private.                                                 |
| `internal/cli/up.go`        | `internal/state/store.go`               | state save             | ✓ WIRED     | Saves/replaces repository state through Store.                                                                   |
| `internal/cli/up.go`        | `internal/labels/labels.go`             | labels/snippet preview | ✓ WIRED     | Builds labels and includes snippet in state preview and JSON.                                                    |
| `internal/workflow/plan.go` | `internal/cli/up.go`                    | checklist rendering    | ✓ WIRED     | Up renders `FoundationUpPlan().Checklist()`.                                                                     |
| `internal/cli/state.go`     | `internal/redact/redact.go`             | redacted state display | ✓ WIRED     | Uses shared renderer for state show output.                                                                      |

### Data-Flow Trace (Level 4)

| Artifact                | Data Variable         | Source                               | Produces Real Data                                | Status    |
| ----------------------- | --------------------- | ------------------------------------ | ------------------------------------------------- | --------- |
| `internal/cli/up.go`    | repo target           | `github.ParseRepo` / `ResolveTarget` | Yes for `--repo`; no for default remote detection | ⚠ PARTIAL |
| `internal/cli/up.go`    | auth source           | `defaultGitHubService.VerifyAuth`    | No — hardcoded `gh` OK                            | ✗ HOLLOW  |
| `internal/cli/up.go`    | repo metadata safety  | `defaultGitHubService.Repository`    | No — hardcoded `Private=true`                     | ✗ HOLLOW  |
| `internal/cli/up.go`    | state path/state save | `state.NewStore` / `SaveRepository`  | Yes                                               | ✓ FLOWING |
| `internal/cli/state.go` | saved state           | `state.Store.GetRepository`          | Yes                                               | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior                       | Command                                                                                                   | Result                                                                           | Status |
| ------------------------------ | --------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ------ |
| Test suite                     | `go test ./...`                                                                                           | Passed                                                                           | ✓ PASS |
| Vet suite                      | `go vet ./...`                                                                                            | Passed                                                                           | ✓ PASS |
| Help output                    | `go run ./cmd/runnerkit --help`                                                                           | Listed `up`                                                                      | ✓ PASS |
| Version JSON                   | `go run ./cmd/runnerkit version --json`                                                                   | JSON only, `redactions_applied:true`                                             | ✓ PASS |
| Dry-run/no-write               | `RUNNERKIT_STATE_DIR=$(mktemp -d) go run ./cmd/runnerkit up --repo owner/repo --dry-run --yes --no-color` | Printed expected labels/snippet; no state file written                           | ✓ PASS |
| Save state                     | `RUNNERKIT_STATE_DIR=$(mktemp -d) go run ./cmd/runnerkit up --repo owner/repo --yes --json`               | JSON contained `runner_installed:false`, `state_path`, `redactions_applied:true` | ✓ PASS |
| State show redaction           | `go run ./cmd/runnerkit state show --repo owner/repo --json` in temp state dir                            | No forbidden secret key terms in output                                          | ✓ PASS |
| Real GitHub auth               | Code inspection of default CLI path                                                                       | Default path is fake-permitted                                                   | ✗ FAIL |
| Real public/fork metadata gate | Code inspection of default CLI path                                                                       | Default path forces private repo metadata                                        | ✗ FAIL |

### Requirements Coverage

| Requirement | Source Plan         | Description                                                      | Status      | Evidence                                                                                                       |
| ----------- | ------------------- | ---------------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------- |
| CLI-01      | 01-01, 01-03        | Install/run CLI binary                                           | ✓ SATISFIED | Runnable Go/Cobra CLI, help/version/state/up commands pass.                                                    |
| CLI-02      | 01-01, 01-02, 01-03 | Guided setup explains choices/prereqs before changes             | ✓ SATISFIED | Up flow previews steps, state path, labels, safety, and no-install message before writes.                      |
| GH-01       | 01-02               | Authenticate for repo with minimum runner-management permissions | ✗ BLOCKED   | Auth/client/token adapters exist, but default CLI does not use them and can save state without any credential. |
| STATE-01    | 01-03               | Versioned local state/config                                     | ✓ SATISFIED | Schema v1, atomic store, state show, labels, cleanup/provider/machine placeholders.                            |
| STATE-02    | 01-01, 01-02, 01-03 | Redact secrets/tokens from state/logs/output                     | ✓ SATISFIED | Redactor, renderer, client token registration, state validation, and spot checks pass.                         |

### Anti-Patterns Found

| File                                              | Line                                | Pattern                                | Severity                                                       | Impact                                                                               |
| ------------------------------------------------- | ----------------------------------- | -------------------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `internal/cli/up.go`                              | 32-44                               | Fake production `defaultGitHubService` | 🛑 Blocker                                                     | CLI reports auth/safety success without real GitHub auth or metadata.                |
| `cmd/runnerkit/main.go`                           | 17-31                               | Missing `CommandRunner` injection      | ⚠ Warning                                                      | Git remote auto-detection is only test-injected, not available in the actual binary. |
| `internal/labels/labels.go`                       | 44                                  | Future collision TODO comment          | ℹ Info                                                         | Acceptable future hook; does not block Phase 1 foundation.                           |
| `internal/cli/up.go` / `internal/state/schema.go` | placeholder machine/provider values | ℹ Info                                 | Intentional Phase 1 placeholders for Phase 2/4; not a blocker. |

### Human Verification Required

None. The blocking issues are code wiring gaps, not human-only checks.

### Gaps Summary

Phase 1 delivered a strong CLI/state/redaction foundation, but it did **not** achieve the GitHub authentication/safety part of the phase goal in the actual binary. The real auth and GitHub REST adapter code exists and is tested in isolation, but `runnerkit up` defaults to a fake service that always approves auth and marks repositories private. As a result, GH-01 is not satisfied and public/fork safety cannot be trusted in production.

---

_Verified: 2026-04-29T02:40:30Z_  
_Verifier: Claude (gsd-verifier)_
