# CONVENTIONS — RunnerKit code style and patterns

> Reference for planning and execution. Every claim cites a file path. Use these conventions when adding code; deviate only with explicit justification.

## Language & toolchain

- **Go 1.22** — declared in `go.mod`; pinned in both CI workflows via `actions/setup-go@v5` `go-version: '1.22'`.
- **No `.golangci.yml` / `.golangci.yaml`** anywhere in the tree. Lint enforcement is `go vet ./...` via `Makefile` `vet:`. No `gofumpt`, `staticcheck`, or `golangci-lint` config files exist.
- Build/test commands (in `Makefile`):
  - `make test` → `go test ./... -count=1`
  - `make test-race` → `go test ./... -count=1 -race`
  - `make test-integration` → `RUNNERKIT_INTEGRATION=1 go test -tags=integration ./internal/bootstrap/... -count=1 -v`
  - `make vet`, `make release-snapshot`
- Direct dependencies in `go.mod`:
  - `github.com/spf13/cobra` v1.10.1
  - `github.com/hetznercloud/hcloud-go` v1.59.2
  - `github.com/hashicorp/go-version` v1.9.0
  - `golang.org/x/term` v0.10.0
- `github.com/stretchr/testify` is in `go.sum` as a transitive dep but **NOT imported anywhere** — every test uses stdlib `testing` only.

## Package layout

- Classic Go layout: production code lives entirely under `internal/` (import-private to the module). No `pkg/`.
- Entry point `cmd/runnerkit/main.go`. Smoke helper binaries under `cmd/_smokebin/{empty_precheck,destroy_verify}`.
- Packages under `internal/` with role:
  - `cli` — Cobra command tree, `Dependencies` struct, exit-code mapping
  - `bootstrap` — remote install plan rendering + `Apply`/`ApplyEphemeral`; `BaselinePackages`; sudoers template
  - `errcodes` — stable `RKD-<COMPONENT>-NNN` registry + docs URL builder
  - `github` — API client + service interface
  - `labels` — RunnerKit runner labels and workflow snippet
  - `ops` — status/doctor/probes/logs/recovery above `remote` + `github`
  - `preflight` — SSH preflight checks producing `Report.Results`
  - `provider`, `provider/hetzner` — provider interface + Hetzner implementation
  - `redact` — secret masking (strings + JSON paths)
  - `remote` — SSH `Executor` interface, `SystemExecutor`, `LoggingExecutor`, `UnavailableExecutor`
  - `rklog` — slog factory
  - `runmode` — persistent vs ephemeral
  - `state` — on-disk schema, store, migrations
  - `testsupport` — cross-package test fakes + fixtures
  - `ui` — renderer, prompter, box, checklist
  - `update` — version check / notifier
  - `ux/checkliststore`, `ux/nextaction`, `ux/stage` — resumable BYO checklists; versioned `next_actions` JSON contract; lifecycle stage inference
  - `workflow` — bootstrap step plan
- File naming: snake_case Go style, one file per concept. Tests sit next to subject: `up.go` → `up_test.go`. Multi-aspect coverage uses topic suffixes: `up_byo_test.go`, `up_cloud_test.go`, `up_ephemeral_test.go`, `up_integration_test.go`, `bootstrap/install_integration_test.go`.

## Import grouping and aliases

Verified in `internal/cli/up.go`, `internal/cli/root.go`, `internal/cli/doctor.go`:

```go
import (
    // stdlib
    "context"
    "errors"
    "fmt"

    // module-internal
    "github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
    "github.com/accidentally-awesome-labs/runnerkit/internal/errcodes"
    gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
    rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"

    // third-party
    "github.com/spf13/cobra"
)
```

Standard aliases:
- `gh "internal/github"` (frees `github`)
- `rkstate "internal/state"` (frees `state`)
- `hcloud "github.com/hetznercloud/hcloud-go/hcloud"`

`import .` is never used.

## Naming patterns

- Files: lower_snake_case (`workflow_packages.go`, `doctor_shared_host.go`).
- Exported identifiers PascalCase (`bootstrap.Apply`, `preflight.Run`, `state.NewStore`). Unexported camelCase (`runUp`, `resolveBYOTarget`, `buildBootstrapOptions`).
- Constant groups use a shared prefix:

  ```go
  // internal/cli/exit.go
  const (
      ExitSuccess           = 0
      ExitUnexpected        = 1
      ExitInvalidInput      = 2
      ExitGitHubAuth        = 3
      ExitSafetyGate        = 4
      ExitStateIO           = 5
      ExitInputRequired     = 6
      ExitStateSchemaTooNew = 7
      ExitCanceled          = 130
  )
  ```

- **Check IDs / finding IDs / step IDs** are stable lowercase `dot.separated` strings declared as exported constants. From `internal/preflight/checks.go`:
  ```go
  CheckSSHConnectivity         = "ssh.connectivity"
  CheckPrivilege               = "host.privilege"
  CheckPrivilegePasswordReq    = "host.privilege.password_required"
  CheckPrivilegeCloudBootstrap = "host.privilege.cloud_bootstrap"
  CheckHostMemAvailable        = "host.mem_available"
  ```
- Bootstrap remote command IDs are `lower_snake_case`: `fix_dependencies`, `setup_runner_image`, `create_runner_user`, `download_runner`, `configure_runner`, `install_service`, `verify_service`, `configure_ephemeral_runner`, `install_ephemeral_service`, `install_ephemeral_ttl_timer`, `verify_ephemeral_service`. Tests treat these as join keys.
- **Error codes** follow `RKD-<COMPONENT>-NNN` (components `AUTH`, `SSH`, `BOOT`, `GH`, `PROV`, `CLEAN`, `STATE`, `CORE`). Stable across renames, monotonic per component, **never renumbered**. Single source of truth: `internal/errcodes/codes.go`. Emitted via `errcodes.FormatLine(code)` (→ `RKD-XXX-NNN: <Title>\nSee: <URL>`).

## Options struct pattern

Three idioms in use:

1. **`Options` struct passed by value, normalized internally** — `bootstrap.Options` + `normalizeOptions(opts *Options)` in `internal/bootstrap/install.go`. `Apply(ctx, exec, target, opts)` accepts the struct, fills defaults (e.g. `ServiceUser`, `InstallPath`, `WorkDir`, `EphemeralTTL=24h`), then iterates the `[]remote.Command` slice.
2. **CLI command `*Options` pointer carrying Cobra flag bindings** — `internal/cli/up.go` `upOptions{}` with `cmd.Flags().StringVar(&opts.repo, "repo", ...)`. The runner is a top-level `runUp(deps, jsonOutput, noColor, opts)` returning `error`.
3. **Functional options for providers** — `hetzner.Option = func(*Provider)` with `WithClient`, `WithClientFactory`, `WithSleep`, `WithLogger` (`internal/provider/hetzner/provision.go`).

**Dependency injection** is centralized in `cli.Dependencies` (`internal/cli/root.go` lines 23-50). `normalizeDependencies` (lines 52-112) fills defaults: `rklog.NewFromEnv(deps.Err)` for the logger, `time.Now` for the clock, `gh.OSCommandRunner{}`, `gh.NewService(...)`, `provider.NewRegistry(hetzner.NewProvider(...))`, `remote.NewSystemExecutor()` wrapped by `remote.WrapWithLogging`, default poll interval/timeout, default `Sleep` honoring `ctx.Done()`, default `Explain`/`UnicodeBox` returning false.

## Error handling — three layered surfaces

### 1. `preflight.Result` (structured, non-fatal)

`internal/preflight/checks.go`:

```go
type Result struct {
    Check       Check
    ID          string
    Severity    Severity   // pass / warning / failure
    Message     string
    Remediation string
    Fixable     bool
}
func (r Report) Passed() bool {
    for _, result := range r.Results {
        if result.Severity == SeverityFailure { return false }
    }
    return true
}
```

Warnings deliberately keep `Passed()` true so CLI can branch on specific IDs (e.g. `CheckPrivilegePasswordReq` triggers a "run one-time host install" copy without failing preflight).

### 2. `ops.Finding` (doctor structured findings)

`internal/ops/doctor.go`. Findings tied to a registered error code go through `addWithCode`, which appends `"\n\nSee: " + errcodes.URL(code)` to remediation. Pass-only findings use `add` without a code.

### 3. Typed `error` wrappers for command exit

- **`cli.ExitError`** (`internal/cli/exit.go`) — carries the process exit code through Cobra; build with `NewExitError(code, err)`; `cli.ExitCode(err)` resolves via `errors.As`, returning `ExitCanceled` (130) for `context.Canceled` and `ExitUnexpected` (1) for anything else.
- **`bootstrap.ServiceNotActiveError`** — `{Err, CommandID, Stderr}` so callers can surface the failing step's `CommandID` and remote stderr; discriminated with `errors.As` in `internal/cli/up.go`.
- **`remote.RemoteError`** — `{CommandID, ExitCode, Message}` for non-zero remote exit.
- **Sentinels** — `state.ErrSchemaTooNew` (its `.Error()` embeds `RKD-STATE-004` + URL inline, guarded by `TestErrcodesEmit_StateSchemaTooNew_IncludesRKDCode`), `state.ErrRepositoryExists`.

Error wrapping uses stdlib only: `fmt.Errorf("...: %w", err)`, `errors.Is`, `errors.As`. No third-party error library.

**Standard CLI emit shape:**

```go
_ = renderer.Error("code_slug", "User-facing sentence.", []string{
    "Remediation step 1.",
    errcodes.FormatLine(errcodes.AuthRunnerManagementPermissionDenied),
})
return NewExitError(ExitGitHubAuth, err)
```

The `_ =` on `renderer.Error` is intentional — output failure is best-effort.

## Preflight vs errors — at a glance

| Where | Type | Severity values | Stops execution? |
|---|---|---|---|
| `preflight.Run` | `preflight.Result` | `pass` / `warning` / `failure` | Only `failure` (via `Passed()`) |
| `ops.BuildDoctorReport` | `ops.Finding` | `SeverityPass` / `SeverityWarning` / `SeverityError` | Never (informational) |
| Cobra `RunE` returns | `error` (often `*ExitError`) | n/a | Yes — `os.Exit(cli.ExitCode(err))` in `cmd/runnerkit/main.go` |

CLI inspects preflight by **stable ID** before deciding to escalate (`internal/cli/up.go`):

```go
if r, ok := report.Result(preflight.CheckPrivilegePasswordReq); ok && r.Severity == preflight.SeverityWarning {
    return RenderHostInstallRequired(renderer, jsonOutput, deps.Version)
}
```

`preflight.Options.RequirePasswordlessSudo` flips the same scenario from warning → `CheckPrivilegeCloudBootstrap` failure for the cloud path.

## Logging

- Library: stdlib `log/slog` only.
- Builder: `rklog.NewFromEnv(w io.Writer) *slog.Logger` in `internal/rklog/rklog.go`.
- Env-driven:
  - `RUNNERKIT_LOG` — `off`/`0`/`false` discards (default); `debug`/`info`/`warn`/`error` sets level; unknown non-empty → `info`. Debug enables `AddSource: true`.
  - `RUNNERKIT_LOG_DEST` — `stderr` (default), `stdout`, or `file:<path>` (parent dir `0o755`, file `0o644`).
- Handler: always `slog.NewJSONHandler`. Discard path returns a no-op `discardHandler` so `log.Enabled(...)` is cheap.
- Event-name convention: `dot.separated` lower_snake — `runnerkit.cli.begin`, `remote.probe`, `remote.probe.error`.
- Always use typed `slog.<Type>(key, value)` constructors:
  ```go
  // internal/cli/root.go PersistentPreRun
  deps.Logger.InfoContext(ctx, "runnerkit.cli.begin",
      slog.String("command", cmd.CommandPath()),
      slog.String("version", deps.Version),
  )
  ```
- `remote.LoggingExecutor` (`internal/remote/logging_executor.go`) wraps `Executor` only when the logger is enabled at info; stdout/stderr bodies are debug-only and routed through `redact.Redactor`.

## User-facing output (`internal/ui/`, `internal/ux/`)

- One renderer, two formats: `ui.Renderer` (`internal/ui/output.go`) handles both `FormatHuman` and `FormatJSON`. Built by `cli.newRenderer` (`internal/cli/root.go`), which forces:
  - `caps.Color=false` on `--no-color`, `NO_COLOR`, `CLICOLOR=0`, `TERM=dumb`
  - `caps.ASCII=true` on `TERM=dumb` or non-TTY stdout
  - a fresh `redact.New()` redactor
- Line kinds and glyphs (`internal/ui/output.go`):

  | Kind | Unicode | ASCII |
  |---|---|---|
  | `LineSuccess` | `✓` | `OK` |
  | `LineWarning` | `!` | `WARNING` |
  | `LineError` | `✗` | `ERROR` |
  | `LinePrompt` | `?` | `PROMPT` |
  | `LineNext` | `→` | `NEXT` |
  | `LineBullet` | `•` | `-` |

  Helpers `ui.Success`, `ui.WarningLine`, `ui.ErrorLine`, `ui.PromptLine`, `ui.Next`, `ui.Bullet`.
- JSON invariants (asserted by `TestJSONOutputIsMachineOnlyAndRedacted` in `internal/ui/output_test.go`):
  - always wraps with `redactions_applied: true`
  - HTML escaping disabled (`encoder.SetEscapeHTML(false)`)
  - passed through `redact.Redactor.JSONBytes`
  - written to stdout only — never stderr
- Versioned JSON contract (`internal/ux/nextaction/nextaction.go`): `SchemaVersion = 1`. `MergePayload(base, stage, actions)` adds `schema_version`, optional `stage`, and `next_actions` array. `Action` is `{id, severity, title, command?, kind?}` where `severity ∈ {info, warning, blocking}` and `kind ∈ {run_on_host, run_local}`.
- Checklist rendering (`internal/ui/checklist.go`): `ChecklistStep{ID,Title,Status,Duration}` with `Status ∈ {done, active, pending}`; persisted to `internal/ux/checkliststore` so BYO `up`/`register` is resumable.
- Boxed command rendering (`internal/ui/box.go`): `RenderBoxed(host, cmd, why, useUnicode, width)` draws copy-paste boxes with Unicode (`┌─┐│└─┘`) or ASCII borders.
- Lifecycle stage (`internal/ux/stage/stage.go`): `Stage ∈ {no_local_state, unknown, error, uninstalled, installed, registered, running}`. `InferFromObserved` / `InferFromDoctor` use `ops.ObservedRunner`, `ops.Health`, `ops.DeepChecks`.

## Feature gating

- **OS/distro:** `bootstrap.isUbuntuLike(opts.OSReleaseID)` gates the `setup_runner_image` step (`internal/bootstrap/install.go` lines 117-121, 168-172); non-Ubuntu-like hosts get only `fix_dependencies`. `preflight.isRecognizedLinux` lists supported IDs (`ubuntu, debian, linuxmint, fedora, centos, rhel, rocky, almalinux, arch, opensuse-leap, opensuse-tumbleweed`).
- **Cloud vs BYO:** `resolveSetupPath` returns `setupPathCloud` or BYO in `internal/cli/up.go`. Further branches via `bootstrap.Options.CloudProvisioned` (skips `BaselinePackages` because cloud-init installed them) and `preflight.Options.RequirePasswordlessSudo` (turns the password-required warning into a `CheckPrivilegeCloudBootstrap` failure).
- **Persistent vs ephemeral:** branch on `modeDecision.Mode == runmode.ModeEphemeral` to call `bootstrap.ApplyEphemeral` vs `bootstrap.Apply` (different remote command set + different cleanup remediation).
- **Build tags:** only `//go:build integration` is used — gates `internal/bootstrap/install_integration_test.go` (single file). Run via `make test-integration`.
- **Runtime env flags driving behavior:** `RUNNERKIT_INTEGRATION`, `RUNNERKIT_STATE_DIR`, `XDG_STATE_HOME`, `RUNNERKIT_LOG`, `RUNNERKIT_LOG_DEST`, `RUNNERKIT_DOCS_BASE`, `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES`, `RUNNERKIT_NO_UPDATE_NOTIFIER`, `CI`, `NO_COLOR`, `CLICOLOR`, `TERM`.

## Redaction (`internal/redact/redact.go`)

- `Redactor.Register(kind, value)` masks literal values (sorted longest-first to avoid partial matches).
- Built-in regex patterns: `gh[pousr]_*`, `github_pat_*`, `registration-token-*`, `(remove|removal)-token-*`, PEM private keys, `HCLOUD_TOKEN=...`.
- JSON-key-aware (`sensitiveKindForKey`): masks values whose key names contain `token`, `password`, `secret`, `credential`, `private_key`, `hcloud`, etc.
- `redactions_applied: true` is always set into machine output (`objectWithRedactionsFlag`) as a tripwire.
- Kinds: `GitHubToken`, `RunnerRegistrationToken`, `RunnerRemovalToken`, `SSHPrivateKey`, `ProviderCredential`, `MachineRef`, `SudoPassword`.
- Registration pattern at the secret mint site:
  ```go
  // internal/cli/up.go
  token, err := deps.GitHub.CreateRegistrationToken(ctx, repo)
  // ...
  renderer.Redactor().Register(redact.RunnerRegistrationToken, token.Token)
  bootstrapOpts.RunnerToken = token.Token
  ```
  Same pattern for `redact.MachineRef` on host references in `internal/cli/doctor.go`.

## Comments

- Every exported package/type/function gets a doc comment. Package comments live in the lead file (e.g. `internal/errcodes/codes.go` opens with `// Package errcodes is the single source of truth ...`).
- Recurring **bug trail** convention: in-code comments cite `Bug N (Plan ID, YYYY-MM-DD)` plus the regression test that locks the fix in. Example from `internal/preflight/checks.go`:
  ```
  // Bug 31 (Plan 06-13, 2026-05-08): the probe Script MUST be a
  // command that is inside `runnerkit byo-prepare`'s scoped sudoers
  // allowlist, ... defeating the entire one-time-prepare purpose. ...
  // Regression test: TestCheckPrivilege_AllowsScopedSudoers.
  ```
  Found across `internal/bootstrap`, `internal/preflight`, `internal/cli`. New non-obvious changes are expected to carry the same trail.

## Function shape

- Cobra commands always come in pairs: `newXxxCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command` + `runXxx(deps Dependencies, jsonOutput bool, noColor bool, opts *xxxOptions) error`.
- Persistent flags `--json`, `--no-color`, `--explain`, `--unicode` live on the root (`internal/cli/root.go`).
- No global mutable state except `cli.shortEphemeralIDFn` (stubbed in tests via `withDeterministicEphemeralID`) and named defaults like `bootstrap.SharedRunnerCacheRoot`.

## Notable gaps & maintainer notes

- **No `.golangci.yml`** despite `vet` being the only lint guard — adding `golangci-lint` would likely surface a few unhandled `_ =` returns and the package-level mutable `shortEphemeralIDFn`.
- **`stretchr/testify` is in `go.sum` but unused** — could be pruned with `go mod tidy` if a dep no longer transitively requires it.
