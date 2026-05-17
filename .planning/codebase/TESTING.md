# TESTING — RunnerKit test framework and practices

> Reference for planning and execution. Every claim cites a file path.

## Test framework

- **stdlib `testing` only.** `testify` is in `go.sum` as a transitive dep but `grep -rln 'github.com/stretchr/testify' internal/ cmd/` returns no matches.
- Assertions are plain `if got != want { t.Fatalf("..."); }`. Sub-cases use `t.Run(tc.name, ...)`.
- `~32` test files use `t.Parallel()`; majority of tests are serial because they mutate `t.Setenv` or fake state.
- Run commands:
  - `make test` → `go test ./... -count=1`
  - `make test-race` → `go test ./... -count=1 -race` (CI default in `.github/workflows/pr-checks.yml`)
  - `make test-integration` → `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -v`
- No `TestMain` anywhere; no benchmark files.

## Test file organization

- **Co-located, white-box** (same package): every `xxx.go` ships with a sibling `xxx_test.go` in the same package.
- Topic suffixes for multi-aspect coverage: `internal/cli/up_test.go`, `up_byo_test.go`, `up_cloud_test.go`, `up_ephemeral_test.go`, `up_ephemeral_e2e_test.go`, `up_integration_test.go`, `up_modes_test.go`.
- One repo-root test file: `install_sh_test.go` (package `main_test`) cross-checks `install.sh` against the Go-rendered sudoers template.
- **Build-tagged tests:** only `//go:build integration` — `internal/bootstrap/install_integration_test.go`. Doubly-gated: requires the tag *and* `RUNNERKIT_INTEGRATION=1` env var, with `t.Skip("set RUNNERKIT_INTEGRATION=1 to run; requires NOPASSWD sudo on the test machine")` inside each test.
- **No `testdata/`, `fixtures/`, or `golden/` directories anywhere** (`find internal -type d -name testdata` returns empty). Fixtures are built inline by Go factory functions in `internal/testsupport/state.go`.

## Table-driven tests

Heavily used in pure-logic packages (`internal/remote/target_test.go`, `internal/runmode/mode_test.go`, `internal/rklog/rklog_test.go`, `internal/ui/cli_prompter_test.go`, `internal/ops/status_test.go`, `internal/provider/hetzner/destroy_test.go`, `internal/cli/up_cloud_test.go`). Canonical shape:

```go
// internal/remote/target_test.go
func TestParseTargetAcceptedForms(t *testing.T) {
    cases := []struct {
        raw  string
        user string
        host string
        port int
    }{
        {raw: "alice@example.com", user: "alice", host: "example.com", port: 22},
        {raw: "alice@example.com:2222", user: "alice", host: "example.com", port: 2222},
        {raw: "ssh://alice@example.com:2222", user: "alice", host: "example.com", port: 2222},
    }
    for _, tc := range cases {
        t.Run(tc.raw, func(t *testing.T) {
            target, err := ParseTarget(tc.raw, 22)
            // ...
        })
    }
}
```

Loop variable is `tc` or `tt`. Sub-test name is a stable identifier (the input string or a hand-named slug). Negative cases use a flat `[]string` slice when the only assertion is "should fail."

## How SSH is faked/mocked

Three layers, no real SSH in unit tests:

1. **Shared fake in `internal/testsupport/remote.go`** — `testsupport.RemoteExecutor` implements `remote.Executor`, captures every `Command` into `r.Commands`, returns canned `Results`/`Errors` keyed by `command.ID`. Also has `ProbeResult`, `ProbeErr`, `ProbeHostKey*` fields. Exposes `CommandIDs() []string` for join-key assertions.
2. **Per-package private fakes** — small structs like `internal/preflight/checks_test.go` `fakePreflightExecutor` (probe + ID-keyed `runResults`) and `internal/cli/root_test.go` `fakeRemoteExecutor` (probe + ID-keyed run results + errors). Same shape, scoped to the test file.
3. **`remote.UnavailableExecutor`** in `internal/remote/executor.go` returns "remote SSH executor is not configured" — fallback when `Apply`/`Probe` is called with nil executor; lets tests assert "this code path should never reach SSH."

`internal/remote/fake_test.go` `TestFakeExecutorSatisfiesExecutor` is a compile-time guard that pins the `Executor` interface shape.

Real SSH only runs in `internal/bootstrap/install_integration_test.go` via a `shellExecutor` that shells out to `bash -c c.Script` against a `httptest.NewServer` serving a synthetic runner tarball — gated as above.

## Fixtures

- Built inline by exported Go factories in **`internal/testsupport/state.go`**:
  - Constants: `TestRepoFullName`, `TestRunnerName`, `TestHostRef`, `TestServiceName`, `TestInstallPath`, `TestWorkDir`, `TestGitHubRunnerID`, `TestLabels`.
  - Functions: `HealthyRepositoryState()`, `BusyRepositoryState()`, `GitHubOfflineRepositoryState()`, `LabelDriftRepositoryState()`, `SSHUnreachableRepositoryState()`, `HostKeyMismatchRepositoryState()`, `MissingGitHubRunnerRepositoryState()`, `MissingServiceRepositoryState()`, `PartialCleanupRepositoryState()`, `EphemeralBYORepositoryState()`, `EphemeralCloudRepositoryState()`, `CloudRepositoryState()`, `HealthyRunner()`, `StateWithRepository(repo)`.
  - Pattern: each derived fixture calls `HealthyRepositoryState()` and mutates only the fields under test.
- Shared GitHub fake: `internal/testsupport/github.go` `GitHubService` — counts every call (`*Calls int`), records last input (`Last*In`), supports per-method error injection (`*Err`). Each method has sensible defaults so most tests construct it empty.
- Output helpers: `internal/testsupport/output.go` (`AssertEqual`, `AssertNoANSI`) and `internal/testsupport/golden.go` (`DecodeJSON[T any]`, `RequireContains`, `RequireNotContains`, `RequireRedactionsApplied` — asserts `"redactions_applied":true` is present). Despite the filename, no actual golden-file pattern is used; everything is in-memory.
- CLI test harness: `internal/cli/root_test.go` `executeForTest(t, args...) (out, err string, err error)` is the standard entry point. Builds a `NewRootCommand(Dependencies{...})` with a temp `StateBaseDir: t.TempDir()`, a `fakePermittedGitHubService`, a `fakeRemoteExecutor`, `Sleep: noSleep`, and `ui.TerminalCapabilities{StdinTTY: false, StdoutTTY: false, Width: 80}`. Two `bytes.Buffer`s capture stdout/stderr.

## Ephemeral test stubbing

Package-level `cli.shortEphemeralIDFn` (a `func() string`) is overridden in tests via a `t.Cleanup`-paired helper (`internal/cli/up_ephemeral_test.go`):

```go
func withDeterministicEphemeralID(t *testing.T) {
    t.Helper()
    prev := shortEphemeralIDFn
    shortEphemeralIDFn = func() string { return "fake1" }
    t.Cleanup(func() { shortEphemeralIDFn = prev })
}
```

## Env-driven tests

`t.Setenv` is the standard pattern (auto-cleans on test exit). Heaviest user is `internal/update/check_test.go` (sets `CI`, `RUNNERKIT_NO_UPDATE_NOTIFIER`). Also in `internal/errcodes/codes_test.go` (`RUNNERKIT_DOCS_BASE` override), `internal/rklog/rklog_test.go` (`RUNNERKIT_LOG`).

## Doctor/list JSON contract assertions

- **`scripts/smoke/assert-doctor-json-contract.sh`** — wraps `python3` and invokes `go run ./cmd/runnerkit doctor --repo <repo> --json --no-color [--deep]`. Required keys: `ok` (must be `True`), `command` (must be `"doctor"`), `repo`, `state_path`, `health`, `findings` (list), `next_actions`, `host_incident_hints` (list, never null), `redactions_applied` (must be `True`), `schema_version`, `stage`. Runs **twice** — baseline then `--deep` — unless `RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1`.
- **`scripts/smoke/assert-list-json-contract.sh`** — runs `go run ./cmd/runnerkit list --json --no-color`. Required keys: `ok`, `command` (must be `"list"`), `schema_version`, `state_path`, `hosts` (list), `next_actions` (list).
- **`scripts/smoke/assert-list-host-repo-count.sh <expected_count> <user@host[:port]>`** — canonicalizes the host arg (adding `:22` default), looks up the host bucket in `hosts[]`, asserts `len(repos) == expected_count`. Used in multi-repo BYO mode.
- All three require `python3`, run from a git checkout, honor `RUNNERKIT_STATE_DIR`.

## Smoke layers (`make smoke-*`, `scripts/smoke/*`)

- **D-11 contract:** live smokes are **maintainer-only**, never wired into CI, never run from forks, never given real `HCLOUD_TOKEN`/`GITHUB_PAT` in CI secrets.
- `make smoke-live` = `smoke-live-byo` + `smoke-live-cloud` + `smoke-stopwatch`.

### `smoke-live-byo` (`scripts/smoke/byo-permission.sh`)

- Required env: `RUNNERKIT_SMOKE_BYO_HOST=user@host`, `RUNNERKIT_SMOKE_REPO=owner/name`. Optional: `RUNNERKIT_SMOKE_MULTI_REPO=1` + `RUNNERKIT_SMOKE_REPO2=owner/other` for SEED-002 dual-repo smoke.
- Creates isolated `mktemp -d` state dir, exports `RUNNERKIT_STATE_DIR`, traps cleanup.
- Runs `runnerkit up --mode persistent --yes`, asserts `/opt/actions-runner/runnerkit-*/config.sh` and `.runner` sentinel exist on the remote host via plain `test -f` (no sudo — these are mode 0755/0664).
- Runs `runnerkit status`, `runnerkit doctor`, both JSON-contract assertions, then `runnerkit down --yes`.
- In multi-repo mode: registers second repo, asserts list shows 2 repos via `assert-list-host-repo-count.sh`, runs doctor JSON contract for the second repo, downs both.

### `smoke-live-cloud` (`scripts/smoke/cloud-end-to-end.sh`)

- Required env: `HCLOUD_TOKEN`, `RUNNERKIT_SMOKE_REPO`. Honors maintainer-set `RUNNERKIT_SMOKE_STATE_DIR` (the Makefile creates it via `mktemp`).
- **Trap-on-exit** runs `runnerkit destroy --yes` before removing the state dir — guarantees a Ctrl-C mid-smoke does not leak billable Hetzner resources.
- Sequence: `hetzner-empty-precheck.sh` → `runnerkit up --cloud hetzner --mode persistent --yes` → status → doctor → doctor JSON contract → list JSON contract → snapshot `state.json` to `state-after-destroy.json` → `runnerkit destroy --yes` → `hetzner-destroy-verify.sh <timeout=300s>`.

### `smoke-stopwatch`

Prints instructions only; maintainer manually records BYO + Hetzner end-to-end durations into `RELEASE-NOTES-vX.Y.Z.md` and the verification doc.

All smoke shell scripts use `set -euo pipefail`, `: "${VAR:?msg}"` for required env assertions, `mktemp -d` for state dirs, and quote every env-substituted variable.

## CI workflows under `.github/workflows/`

### `pr-checks.yml`

Triggers: `pull_request` and `push` to `main`. Two jobs:

- `goreleaser-validate` — setup-go 1.22 + cosign installer + goreleaser v2.15.4 (`install-only`); runs `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean --skip=sign`, then `ls dist/` + asserts the four platform archives (`darwin_amd64`, `darwin_arm64`, `linux_amd64`, `linux_arm64`) + `*_checksums.txt` exist.
- `go-test` — setup-go 1.22 + `go test ./... -count=1 -race` (**race detector always on in CI**).
- `permissions: contents: read` only.

### `release.yml`

Fires on tags matching `v*`. Single `goreleaser` job runs `goreleaser release --clean` with secrets: `GITHUB_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`, `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`. `permissions: contents: write` + `id-token: write` (for OIDC signing).

Per `CLAUDE.md`, the release workflow runs only from the upstream repo (`accidentally-awesome-labs/runnerkit`); fork tag pushes do not trigger upstream releases and may break OIDC signing.

## `make` targets summary

- `make help` — auto-generated from `## ...` comments on PHONY targets.
- `make test` — full unit/integration `go test ./... -count=1`.
- `make test-race` — same with `-race`.
- `make test-integration` — `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -v` (real bash + NOPASSWD sudo required).
- `make vet` — `go vet ./...`.
- `make release-snapshot` — local GoReleaser dry-run.
- `make smoke-live`, `make smoke-live-byo`, `make smoke-live-cloud`, `make smoke-stopwatch` — see "Smoke layers" above.

## Race detector

- CI runs **every PR/push** with `-race` (`.github/workflows/pr-checks.yml` `go-test` job line 48: `go test ./... -count=1 -race`).
- Local `make test` does not pass `-race` (only `-count=1`); maintainer must use `make test-race` to mirror CI locally.
- `internal/redact/redact.go` uses `sync.RWMutex` around its `values []registeredValue` and `patterns []patternFilter`; the redactor is the most likely race target and is exercised in parallel CLI tests.

## Common patterns

**Async/sleep injection.** `cli.Dependencies.Sleep func(ctx, d) error` defaults to a `time.NewTimer`-backed wait that respects `ctx.Done()`. Tests set `Sleep: noSleep` to fast-forward. Same pattern for `hetzner.Provider.Sleep func(time.Duration)` injected via `WithSleep(...)` for retry loops.

**Clock injection.** `cli.Dependencies.Clock func() time.Time` defaults to `time.Now`. Tests pass `Clock: func() time.Time { return time.Date(2026, 5, 2, 18, 30, 0, 0, time.UTC) }` for deterministic ephemeral runner names and `created_at`/`updated_at` timestamps.

**Error assertion shape.** Standard:

```go
if err == nil { t.Fatal("expected error") }
if got := cli.ExitCode(err); got != cli.ExitInputRequired {
    t.Fatalf("ExitCode() = %d, want %d", got, cli.ExitInputRequired)
}
if !strings.Contains(errOut, "--repo owner/name") {
    t.Fatalf("missing remediation in stderr: %q", errOut)
}
```

**JSON output assertion shape.** Standard:

```go
var payload map[string]any
if err := json.Unmarshal([]byte(out), &payload); err != nil {
    t.Fatalf("json output invalid: %v\n%s", err, out)
}
if payload["runner_installed"] != false || payload["redactions_applied"] != true {
    t.Fatalf("unexpected up payload: %#v", payload)
}
```

**Visudo-gated tests.** `internal/bootstrap/sudoers_test.go` skips with `t.Skipf("visudo not available, skipping: %v", err)` when `exec.LookPath("visudo")` fails — keeps macOS dev fast while preserving the safety property on Linux CI.

**Redaction assertion.** `testsupport.RequireRedactionsApplied(t, got)` asserts the string contains `"redactions_applied":true`. Plus `TestJSONOutputIsMachineOnlyAndRedacted` in `internal/ui/output_test.go` asserts (1) output starts with `{`, (2) no ANSI escape `\x1b[`, (3) the registered secret `secret-token` is absent, (4) `redactions_applied:true` is present, (5) stderr is empty.

**Docs-anchor regression.** `internal/errcodes/codes_test.go` walks `Registry`, verifies each `Code` has a `<a name="<anchor>"></a>` in its `File`, asserts `(File, Anchor)` pairs unique, asserts all 6 component files exist on disk and have at least one entry, and asserts each docs section contains `### Symptom`, `### Diagnosis`, `### Fix` (D-17). Uses `runtime.Caller(0)` to find the docs root robustly.
