# STRUCTURE — RunnerKit directory layout

> Reference for planning and execution. Every claim cites a file path.

**Analysis date:** 2026-05-17

## Directory layout

```
spool/                                  # Repo root (module: github.com/accidentally-awesome-labs/runnerkit)
├── cmd/                                # Binary main packages
│   ├── runnerkit/                      # Production CLI entry point
│   └── _smokebin/                      # Maintainer-only smoke harness binaries
│       ├── destroy_verify/             # D-12 gate 2: post-destroy resource verification
│       └── empty_precheck/             # D-12 gate 1: pre-test cleanliness check
├── internal/                           # All non-public Go packages
│   ├── bootstrap/                      # Remote install pipeline + Apply/ApplyEphemeral
│   ├── cli/                            # Cobra subcommands, flow control, output rendering
│   │   └── sessions/                   # Test fixtures for resumable BYO checklist
│   ├── errcodes/                       # Stable RKD-* code registry + docs URL resolver
│   ├── github/                         # GitHub API client, auth, safety evaluation
│   ├── labels/                         # Runner label / name / workflow-snippet computation
│   ├── ops/                            # Doctor/Status/Cleanup analysis (read-only)
│   ├── preflight/                      # Host probe → typed Findings catalog
│   ├── provider/                       # Cloud Provider interface + Registry
│   │   └── hetzner/                    # Hetzner implementation (hcloud-go)
│   ├── redact/                         # Output redactor (tokens, keys, passwords)
│   ├── remote/                         # SSH Executor + Target + HostKey + meminfo parse
│   ├── rklog/                          # slog wiring from RUNNERKIT_LOG/RUNNERKIT_LOG_DEST
│   ├── runmode/                        # persistent vs ephemeral mode + safety profiles
│   ├── state/                          # User-local versioned JSON inventory + migrations
│   ├── testsupport/                    # Cross-package test fakes (importable, not _test.go-gated)
│   ├── ui/                             # Renderer, TerminalCapabilities, prompts, boxes, checklists
│   ├── update/                         # GitHub Releases notice + cache
│   ├── ux/                             # Schema-versioned UX value objects
│   │   ├── checkliststore/             # Persisted BYO progress under <state>/sessions/
│   │   ├── nextaction/                 # `next_actions` JSON envelope helpers
│   │   └── stage/                      # Coarse lifecycle Stage enum + InferFromDoctor
│   └── workflow/                       # Plan/Step typed value objects (BootstrapPlan, FoundationUpPlan)
├── scripts/                            # Bash test/release scripts (not a Go package)
│   └── smoke/                          # Live smoke scripts + JSON contract asserters
├── docs/                               # End-user + maintainer markdown
│   └── troubleshooting/                # Per-component RKD-* anchor targets
├── .github/                            # CI configuration
│   └── workflows/                      # release.yml (GoReleaser), pr-checks.yml
├── .planning/                          # GSD planning artifacts
│   ├── codebase/                       # Codebase mapping output (this directory)
│   ├── phases/01..08/                  # Per-phase plan/verification docs
│   ├── research/                       # Research notes
│   └── seeds/                          # SEED-001..004 specs
├── sessions/                           # Runtime-created BYO checklist state (gitignored)
├── bin/                                # Local-build output (gitignored)
├── dist/                               # GoReleaser output (gitignored)
├── go.mod / go.sum                     # Go module manifest + checksums
├── Makefile                            # test, vet, smoke-live-*, release-snapshot
├── install.sh                          # One-time BYO host installer (curl | sudo bash)
├── install_sh_test.go                  # Tests for install.sh (rendered sudoers, ordering)
├── CLAUDE.md / GEMINI.md               # Agent-facing maintainer notes
├── README.md                           # Public README
└── RELEASE-NOTES-v*.md                 # Per-release notes (annotated tags)
```

## Directory purposes

### `cmd/runnerkit/`

- Purpose: Production CLI binary main package.
- Contains: `main.go` (35 LOC; builds `cli.Dependencies`, calls `cli.NewRootCommand`), `main_test.go` (regression guard for `buildDependencies` wiring per Bug 4 / Task G).
- Key files: `main.go`.

### `cmd/_smokebin/{destroy_verify,empty_precheck}/`

- Purpose: Maintainer-only smoke binaries (D-12 gates). Underscore prefix excludes them from `go build ./...` discovery by tooling that respects it.
- Contains: `main.go` + `main_test.go` per binary.
- Generated: No. Committed: Yes.

### `internal/cli/`

- Purpose: All Cobra command definitions + flow orchestration.
- Contains: One file per subcommand (`up.go`, `register.go`, `down.go`, `destroy.go`, `doctor.go`, `recover.go`, `status.go`, `list.go`, `logs.go`, `init.go`, `wizard.go`, `upgrade.go`, `upgrade_runner.go`, `state.go`); cross-cutting helpers (`root.go`, `exit.go`, `userconfig.go`, `byo_checklist.go`, `doctor_fix.go`, `doctor_shared_host.go`, `installhint.go`, `explain.go`, `quote.go`, `workflow_packages.go`, `update_notice.go`).
- Key files: `root.go` (~216 LOC; `NewRootCommand`, `Dependencies`, `normalizeDependencies`, `newRenderer`), `up.go` (2328 LOC, the orchestrator for both BYO and cloud paths), `exit.go` (typed exit codes), `wizard.go` (first-run TTY wizard).

### `internal/cli/sessions/`

- Purpose: Test fixture for SEED-002 resumable BYO checklist (`byo-owner_repo__alice_example_com.json`).
- Generated: Hand-authored fixture. Committed: Yes.

### `internal/bootstrap/`

- Purpose: Remote install pipeline. Output of `Render*Script` is shell that the SSH executor pipes to `bash -s` on the host.
- Contains: `install.go` (`Options`, `Apply`, `ApplyEphemeral`, `BaselinePackages`, `mergePackages`, `isUbuntuLike`, `SharedRunnerCacheRoot`, `ServiceNotActiveError`, `downloadRunnerCommand`), `script.go` (render scripts for configure/service/ephemeral steps), `image_setup.go` (`RenderImageSetupScript`, `ImageSetupVersion`), `package.go` (`PackageFor`, `RunnerVersion`, pinned SHA256), `sudoers.go` (`SudoersFilePath`, `RenderSudoersEntry`, `RemoteVisudoCheckScript`, `FoundationUserProbeScript`), `ci_sudoers.go` (runner service-user passwordless sudo).
- Tests: `*_test.go` siblings + `install_integration_test.go` (gated on `RUNNERKIT_INTEGRATION=1` with build tag `integration`).

### `internal/ops/`

- Purpose: Read-only correlation of state + GitHub + SSH + systemd + preflight + journal heuristics into typed reports.
- Contains: `doctor.go` (`BuildDoctorReport`, `Finding`, `DeepChecks`), `status.go` (`Health`, `Classify`, `ObservedRunner`, `EphemeralFact`, severity/reason constants), `probes.go`, `logs.go` (`CollectBoundedJournalsForHints`), `hostkillhint.go` (`AnalyzeJournalForOOMHints`, `HostIncidentHint`, `ShouldCollectHostIncidentJournals`), `cleanup.go`, `recovery.go`, `cloud_destroy.go`.

### `internal/preflight/`

- Purpose: Fixed catalog of host checks. Single file.
- Key file: `checks.go` (343 LOC; `Run`, `Report`, `Result`, `Severity`, `Options`, `CheckXxx` ID constants, `RequiredTools`, `NormalizeArch`, `memWarnThresholdBytes`).

### `internal/provider/`

- Purpose: Cloud Provider boundary.
- Contains: `provider.go` (interface + DTOs + `Registry`), `profile.go` (`DefaultHetznerProfile`, `HetznerProvisionPlan`, `HetznerResourceNames`, `HetznerOwnershipTags`, `sanitizeName`), `fake.go` (test double).

### `internal/provider/hetzner/`

- Purpose: Hetzner implementation of `Provider`.
- Contains: `client.go` (hcloud-go wrapper interface + `NewClient`), `credentials.go` (`HCLOUD_TOKEN` / `HETZNER_CLOUD_TOKEN` discovery + `MissingTokenError`), `provision.go` (`Provider`, `NewProvider`, `Validate`, `Plan`, `Provision`, `Describe`, `CloudInitUserDataVersion`, cloud-init user-data renderer, ownership tag helpers), `readiness.go` (`WaitReady` action+server poll), `describe.go`, `destroy.go` (cascade-delete with 409 retry loop).

### `internal/remote/`

- Purpose: SSH transport abstraction.
- Contains: `executor.go` (`Executor`, `HostKeyProber`, `Command`, `Result`, `ProbeResult`, `RemoteError`, `UnavailableExecutor`), `system.go` (`SystemExecutor`, `sshOutput`, `sshArgs`, `selectHostKeyLine`, `parseProcMeminfoViaAwk`), `target.go` (`Target`, `ParseTarget`, `CanonicalHostKey`), `hostkey.go` (`HostKey`, `NormalizeHostKey`, `FingerprintSHA256`), `meminfo.go` (`ParseMeminfoAwkOutput`), `logging_executor.go` (`WrapWithLogging` slog decorator).

### `internal/state/`

- Purpose: User-local JSON inventory + migrations.
- Contains: `schema.go` (`State`, `RepositoryState`, `MachineRef`, `ProviderRef`, `CloudInventory`, `CleanupMetadata`, `SafetyMetadata`, `EphemeralMetadata`, `OperationCheckpoint`, `RunnerIdentity`, `AuthReference`, `CostProfileRef`, `SchemaVersion="2"`, `ErrRepositoryExists`), `store.go` (`Store`, `NewStore`, `NewStoreAtPath`, `DefaultBaseDir`, atomic write, side-by-side backup, `ValidatePersistedJSON`), `migrations.go` (`Migrate`, `cmpVersion`, `migrateV1ToV2`, `ErrSchemaTooNew`), `config.go` (env helpers).

### `internal/errcodes/`

- Purpose: RKD code registry. Zero-dep on `internal/ops` to avoid import cycles.
- Contains: `codes.go` (`Code`, `Severity` enum, ~52 codes in `Registry`), `url.go` (`URL`, `FormatLine` for `See: <url>` suffixes).

### `internal/github/`

- Purpose: GitHub HTTP + auth + safety.
- Contains: `service.go` (`Service`, `ServiceOptions`, `NewService`), `client.go` (HTTP `Client`), `auth.go` (`Credential`, gh CLI discovery), `tokens.go` (registration/removal token mint), `runners.go` (`ListRunners`, `DeleteRunner`, `FindRunnerByName`), `safety.go` (`EvaluateSafety`, `SafetyDecision`, `SafetyOptions`, `SafetyCodePublicRisk`, `AllowPublicRepoRiskFlag`, `FineGrainedTokenRemediation`), `remote.go` (`ResolveTarget`, URL parse), `types.go` (`Repo`, `Runner`, `RunnerToken`, `PermissionStatus`, `AuthSource`, `CommandRunner`, `OSCommandRunner`).

### `internal/ui/`

- Purpose: Output rendering + prompts.
- Contains: `output.go` (`Renderer`, `Format`, `TerminalCapabilities`, `Line`/`LineKind`, `Step`/`Warning`/`Error`/`JSON`), `box.go` (`RenderBoxed`), `checklist.go` (`RenderChecklist`, `ChecklistStep`), `prompt.go` (`Prompter`, `PasswordPrompter`), `cli_prompter.go` (`CLIPrompter` reading from `*os.File`).

### `internal/ux/`

Three small sub-packages — `checkliststore` (persisted BYO progress), `nextaction` (JSON envelope helpers + `SchemaVersion`), `stage` (`Stage` enum + `InferFromObserved` / `InferFromDoctor`).

### `internal/testsupport/`

Cross-package test helpers. Notably NOT `_test.go`-gated — these files can be imported from any test package. Contains `github.go`, `golden.go`, `output.go`, `remote.go`, `state.go`.

### `scripts/smoke/`

- Purpose: Live smoke entrypoints (maintainer-only per D-11; never wired into CI).
- Contains: `byo-permission.sh`, `cloud-end-to-end.sh`, `hetzner-empty-precheck.sh`, `hetzner-destroy-verify.sh`, `install-sh-matrix.sh`, plus JSON-contract asserters: `assert-doctor-json-contract.sh`, `assert-list-json-contract.sh`, `assert-list-host-repo-count.sh`.
- Invoked from: `Makefile` targets `smoke-live-byo`, `smoke-live-cloud`, `smoke-live`.

### `docs/`

- Purpose: End-user + maintainer documentation.
- Top-level: `byo-quickstart.md`, `cloud-quickstart.md`, `release-process.md` (pre-tag checklist), `runner-platforms.md`, `safety.md`, `upgrade.md`.
- `docs/troubleshooting/`: One markdown file per RKD component (`auth.md`, `bootstrap.md`, `cleanup.md`, `doctor-ux.md`, `github.md`, `host-resources.md`, `multi-repo.md`, `provider.md`, `ssh.md`) + `README.md` index. Each anchor matches an `errcodes.Code{Anchor}` value; tests walk the registry to enforce coverage.

### `.github/workflows/`

`release.yml` (GoReleaser; triggered by `v*` tag push on upstream repo only; signs + publishes Releases + bumps the Homebrew tap), `pr-checks.yml` (CI test/vet/lint).

## Key file locations

**Entry points:**
- `cmd/runnerkit/main.go` — Production binary main.
- `internal/cli/root.go` — Subcommand registration in `NewRootCommand` (`root.go:163-176`).
- `install.sh` — One-time BYO host installer (curl | sudo bash).

**Configuration:**
- `go.mod` — Module manifest (`go 1.22`, hcloud-go, cobra, hashicorp/go-version, golang.org/x/term).
- `Makefile` — `test`, `test-race`, `test-integration` (gated `RUNNERKIT_INTEGRATION=1`), `vet`, `release-snapshot`, `smoke-live-*` (maintainer-only).
- `CLAUDE.md` — Agent-facing maintainer notes (release process, SEED notes).

**Core orchestration:**
- `internal/cli/up.go` — `runUp` (BYO) + `runCloudUp` (Hetzner). Single largest file (2328 LOC) holds setup-path resolution, mode prompts, preflight, bootstrap dispatch, online wait, state persist for both paths.
- `internal/bootstrap/install.go` — `Apply` + `ApplyEphemeral` pipelines.
- `internal/preflight/checks.go` — Host check catalog.
- `internal/provider/hetzner/provision.go` — Hetzner Provision / Validate / Plan / Describe.
- `internal/state/schema.go` + `store.go` — Persisted state.

**Testing:**
- Tests live next to source as `*_test.go` files (Go convention).
- Build-tag-gated integration tests: `internal/bootstrap/install_integration_test.go` requires `-tags=integration` + `RUNNERKIT_INTEGRATION=1` + NOPASSWD sudo on the local machine.
- Cross-package test helpers: `internal/testsupport/` (regular `.go` files, importable).

## Naming conventions

### Files

- Lowercase, underscore-separated (Go standard): `install.go`, `image_setup.go`, `hostkillhint.go`, `workflow_packages.go`.
- One subcommand per `internal/cli/<subcommand>.go`; multi-aspect subcommands split (`doctor.go`, `doctor_fix.go`, `doctor_shared_host.go`; `up.go` + `up_byo_test.go`, `up_cloud_test.go`, `up_ephemeral_test.go`).
- Tests as `<file>_test.go` siblings; integration tests as `<file>_integration_test.go`.
- Maintainer-only binaries prefixed `_`: `cmd/_smokebin/`.

### Directories

- Lowercase, single-word where possible: `bootstrap`, `cli`, `preflight`, `remote`, `state`, `ui`.
- Multi-word as compounds, no separator: `errcodes`, `testsupport`, `runmode`, `checkliststore`, `nextaction`.
- Provider implementations as subdirectories: `provider/hetzner/`.

### Packages

- Package name matches directory name (Go convention).
- Imports use full module path: `github.com/accidentally-awesome-labs/runnerkit/internal/<pkg>`.
- Common aliases: `gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"` (used because the stdlib-ish name `github` conflicts with the org name); `rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"` (avoids shadow with local vars named `state`); `hcloud "github.com/hetznercloud/hcloud-go/hcloud"`.

### Identifiers

- Exported `CamelCase`, unexported `camelCase` (Go standard).
- Doctor finding IDs in snake_case (`github_runner_offline`, `host_mem_low`, `byo_host_prepared`) so they map cleanly to JSON keys.
- Error codes `RKD-<COMPONENT>-NNN` with `Code{ID, Severity, Title, File, Anchor}` and `Anchor` mirroring lowercase ID (`rkd-auth-001`).
- Cobra subcommand `Use:` strings always single lowercase words (`up`, `down`, `destroy`, `doctor`, `recover`, `upgrade-runner` is the lone hyphen).
- Preflight check IDs use dotted hierarchy (`ssh.connectivity`, `host.privilege.password_required`, `host.network.github`).
- Bootstrap step IDs use snake_case (`fix_dependencies`, `create_runner_user`, `download_runner`, `configure_runner`, `install_service`, `verify_service`, `setup_runner_image`).

## Where to add new code

### New subcommand

1. Create `internal/cli/<name>.go` with a `new<Name>Command(deps Dependencies, jsonOutput, noColor *bool) *cobra.Command` constructor and `run<Name>(deps, jsonOutput, noColor, opts)` body.
2. Define `<name>Options` struct holding all flag values; bind in the constructor.
3. Wire into `internal/cli/root.go:163` block with `root.AddCommand(new<Name>Command(deps, &jsonOutput, &noColor))`.
4. Use `newRenderer(deps, jsonOutput, noColor)` for output, `NewExitError(ExitXxx, err)` for typed exits, and `errcodes.FormatLine(<Code>)` to append docs links to remediation slices.
5. Tests in `internal/cli/<name>_test.go` using `testsupport.NewFakeGitHubService` / `testsupport.NewFakeRemoteExecutor`.

### New bootstrap step

1. Add a `Render<StepName>Script(opts Options) string` to `internal/bootstrap/script.go` (or a new sibling file if substantial, e.g. `image_setup.go`).
2. Append a `remote.Command{ID: "<step>", Script: Render<StepName>Script(opts), Sudo: true}` into the `commands` slice in `Apply` (and `ApplyEphemeral` if applicable) in `internal/bootstrap/install.go:108,159`. Place it in canonical order (e.g. dependency-related steps before `create_runner_user`).
3. If the step is OS-gated, wrap in `if isUbuntuLike(opts.OSReleaseID) { ... }` (`install.go:117`).
4. If failures should surface as `ServiceNotActiveError`, add the step ID to the switch in the loop (`install.go:134-136,187-189`).
5. Update the static `workflow.BootstrapPlan` in `internal/workflow/plan.go:79` so dry-run previews match.
6. If new sudo commands are required, extend `RenderSudoersEntry` (`internal/bootstrap/sudoers.go:59`) and bump `install.sh` in lockstep.
7. Add tests in `internal/bootstrap/script_test.go` (golden) + `install_test.go` (fake executor verifies ordering).

### New provider (e.g. `digitalocean`)

1. Add `internal/provider/<name>/` mirroring the hetzner structure: `client.go`, `credentials.go`, `provision.go` (with `Provider` struct implementing `provider.Provider`), `readiness.go`, `describe.go`, `destroy.go`.
2. Add provider constants to `internal/provider/profile.go` (`<Name>Provider`, `<Name>DefaultRegion`, etc.) and a `Default<Name>Profile()`.
3. Add `<Name>ProvisionPlan(input)`, `<Name>ResourceNames(input)`, `<Name>OwnershipTags(input)` in `profile.go`.
4. Register in `internal/cli/root.go:80` via `provider.NewRegistry(hetzner.NewProvider(...), <name>.NewProvider(...))`.
5. Accept the new value in `resolveSetupPath` (`internal/cli/up.go:622`) and the `--cloud` flag help text.
6. Add destroy/describe wiring in `internal/ops/cloud_destroy.go` and any cloud-aware status code paths in `internal/cli/status.go`.
7. Add a fake in `internal/provider/fake.go` if tests need provider-specific behavior, and new RKD-PROV-* codes in `internal/errcodes/codes.go` plus matching anchors in `docs/troubleshooting/provider.md`.

### New preflight check

1. Add a `CheckXxx` constant to `internal/preflight/checks.go:13` block.
2. Inside `Run` (`checks.go:114`), emit `pass()`, `warning()`, or `failure()` using the new ID. If you need new probe data, extend `remote.ProbeResult` (`internal/remote/executor.go:33`) and the `SystemExecutor.Probe` implementation (`internal/remote/system.go:18`).
3. If the check is fixable (i.e. `bootstrap.Apply` will install something), set `Result{Fixable: true}` and append to `report.FixableTools` so `buildBootstrapOptions` picks it up.
4. Allocate a new `errcodes.BootXxx` code (next available number) in `internal/errcodes/codes.go` and add an anchored entry in `docs/troubleshooting/bootstrap.md`.
5. Map the new check in `internal/ops/doctor.go:113` switch so `doctor` reports it with the new code.
6. Tests: `internal/preflight/checks_test.go` + (if relevant) `internal/preflight/checks_bugfix_test.go`.

### New doctor finding (without a new preflight check)

1. Pick an `errcodes.Code` (existing or new in `internal/errcodes/codes.go`).
2. Add the `addWithCode("<snake_id>", errcodes.<Code>, Severity, "<source>", evidence, remediation)` call in `BuildDoctorReport` (`internal/ops/doctor.go:46`). Keep source values in the existing vocabulary (`state`, `github`, `ssh`, `systemd`, `labels`, `provider`, `remote`, `preflight`, `bootstrap`, `logs`, `ephemeral`, `host`).
3. If the finding requires new evidence collection, add a helper next to existing probes (`internal/ops/probes.go`) and call it from `internal/cli/doctor.go:144` (`collectDoctorChecks`) or `collectDoctorHostHints`.
4. Add anchor + Symptom/Diagnosis/Fix section to the appropriate `docs/troubleshooting/<component>.md`; tests in `errcodes` walk the registry to verify the docs anchor exists.

### New errcode / RKD ID

1. Append a `var <Name> = Code{ID: "RKD-<COMP>-NNN", Severity: ..., Title: "...", File: "<comp>.md", Anchor: "rkd-<comp>-nnn"}` in `internal/errcodes/codes.go`. Number monotonically per component; never renumber.
2. Add to the `Registry` slice (`codes.go:119`).
3. Add a matching `rkd-<comp>-NNN` anchored section in `docs/troubleshooting/<comp>.md` with Symptom/Diagnosis/Fix.
4. Emit at every relevant call site via `errcodes.FormatLine(<Name>)` appended to the remediation slice passed to `renderer.Error(...)`.

### New state field

1. Add the field as `omitempty` to the appropriate struct in `internal/state/schema.go` (additive, no `SchemaVersion` bump needed if old states still load).
2. If schema-breaking, bump `SchemaVersion` (`schema.go:10`) and add a `migrate<V>To<V+1>` function in `internal/state/migrations.go` registered in `Migrate`.
3. Update `build<XYZ>RepositoryState` constructors in `internal/cli/up.go` (and `down.go` / `destroy.go` for cleanup paths) so the field is populated.
4. If the field is a secret, ensure the key is in `isDeniedSecretKey` (`internal/state/store.go:266`) so a bug that ever assigned a raw value would fail `ValidatePersistedJSON`.

## Special directories

- **`internal/`** — Standard Go `internal/` — packages here cannot be imported by code outside this module. All non-`main` code lives here.
- **`internal/testsupport/`** — Importable from any test package. Generated: No. Committed: Yes.
- **`cmd/_smokebin/`** — Maintainer-only binaries. Generated: No. Committed: Yes. Underscore prefix is a convention to signal "not part of the public CLI surface."
- **`.cache/`** — Local Go build/module cache. Generated: Yes. Committed: No (per `.gitignore`).
- **`bin/`, `dist/`** — `go build` / GoReleaser output. Generated: Yes. Committed: No.
- **`sessions/`** — Runtime BYO checklist sessions written by `internal/ux/checkliststore`. Generated: Yes. Committed: No.
- **`.planning/`** — GSD planning artifacts. Generated: By GSD commands. Committed: Yes for plans/seeds/research; this `codebase/` directory is regenerated on demand.
- **`.pi/`, `.pi-lens/`, `.claude/`** — Tooling configs. Mixed gen/commit status; not load-bearing for the CLI itself.
