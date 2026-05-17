# ARCHITECTURE — RunnerKit system design

> Reference for planning and execution. Every claim cites a file path.

**Analysis date:** 2026-05-17

## Pattern Overview

**Overall:** Single-binary Cobra CLI orchestrating a layered Go pipeline. Module is `github.com/accidentally-awesome-labs/runnerkit` (Go 1.22). Each subcommand registered in `internal/cli/root.go` delegates through layers: `cli` → `ops` (read-only analysis) → `bootstrap` / `preflight` / `remote` (host I/O) → `provider` (cloud lifecycle) → `state` (user-local JSON inventory).

### Key characteristics

- **Dependency injection at the root.** `cli.Dependencies` (`internal/cli/root.go:23`) bundles writers, TTY caps, clock, `Prompter`, `CommandRunner`, `GitHubService`, `provider.Registry`, `remote.Executor`, `*slog.Logger`, `PollInterval/Timeout`, `Sleep`, `Explain`, `UnicodeBox`. `normalizeDependencies` (`root.go:52`) wires production defaults; tests inject fakes.
- **Plan-before-mutation.** Every billable or remote-modifying flow renders a plan, requires `--yes` (or interactive confirm), and supports `--dry-run`. See `runUp` / `runCloudUp` (`internal/cli/up.go:99,659`), `runDown`, `runDestroy`, `runRecover`, `runUpgradeRunner`.
- **Stable error codes (RKD-*).** `internal/errcodes/codes.go` is the single registry mapping every user-facing failure / doctor finding to an `RKD-<COMPONENT>-NNN` ID + URL into `docs/troubleshooting/<component>.md` (D-15). Components: `AUTH`, `SSH`, `BOOT`, `GH`, `PROV`, `CLEAN`, `STATE`, `CORE`.
- **Two output modes.** `--json` for machine output (`ui.Renderer.JSON` in `internal/ui/output.go`), human stepwise blocks for default. `internal/ux/nextaction/nextaction.go` and `internal/ux/stage/stage.go` produce typed, schema-versioned JSON envelopes (`schema_version`, `stage`, `next_actions`, `host_incident_hints`).
- **Secrets never persisted.** `internal/redact/redact.go` redacts tokens/SSH keys/passwords in all output; `state.ValidatePersistedJSON` (`internal/state/store.go:226`) refuses to write state files with raw secret keys.
- **Idempotent SEED-002 per-repo isolation.** Each saved repository has unique `InstallPath`, `RunnerName`, `WorkDir`, systemd unit so multiple `runnerkit-*` runners share a BYO host. Runner tarballs cache once per version under `/opt/actions-runner/runnerkit-shared-bin/<version>/` (`bootstrap.SharedRunnerCacheRoot`, `internal/bootstrap/install.go:202`).

## Layers

### `cmd/runnerkit` (entry point)

- Purpose: Build production `cli.Dependencies` and execute the root Cobra command.
- Location: `cmd/runnerkit/main.go`
- Contains: `buildDependencies()` wiring `github.OSCommandRunner{}`, `ui.NewCLIPrompter`, `time.Now`, stdio with `isTerminal` probe; `main()` calls `cli.NewRootCommand(...).Execute()` then `os.Exit(cli.ExitCode(err))`.
- Depends on: `internal/cli`, `internal/github`, `internal/ui`.

### `internal/cli` (presentation + flow control)

- Purpose: Cobra definitions, flag parsing, output rendering, prompts, state save/load orchestration. Holds the bulk of `runUp` / `runCloudUp` / `runDown` / `runDestroy` / `runDoctor` / `runRecover` glue.
- Contains: one file per subcommand (`up.go` 2328 LOC, `register.go`, `down.go`, `destroy.go`, `doctor.go`, `recover.go`, `status.go`, `list.go`, `logs.go`, `init.go`, `wizard.go`, `upgrade.go`, `upgrade_runner.go`, `state.go`); cross-cutting helpers `root.go`, `exit.go`, `userconfig.go`, `byo_checklist.go`, `doctor_fix.go`, `doctor_shared_host.go`, `installhint.go`, `explain.go`, `quote.go`, `workflow_packages.go`, `update_notice.go`.
- Depends on: `bootstrap`, `errcodes`, `github`, `labels`, `ops`, `preflight`, `provider`, `provider/hetzner`, `redact`, `remote`, `runmode`, `state`, `ui`, `ux/nextaction`, `ux/stage`, `ux/checkliststore`, `workflow`.
- Used by: `cmd/runnerkit`.

### `internal/ops` (read-only analysis)

- Purpose: Build typed `DoctorReport` / `Health` / `CleanupPlan` from saved state + observed GitHub/SSH/systemd facts. Pure with a few helpers that fan out to `RemoteExecutor` / `GitHubService` (no mutation).
- Contains: `doctor.go` (`BuildDoctorReport`, `Finding`, `DeepChecks`), `status.go` (`Health`, `Classify`, `ObservedRunner`, `EphemeralFact`, `HealthState` enum), `probes.go` (SSH/systemd probes), `logs.go` (`CollectBoundedJournalsForHints`), `hostkillhint.go` (`AnalyzeJournalForOOMHints`, `HostIncidentHint`, `ShouldCollectHostIncidentJournals`), `cleanup.go` (`BuildCleanupPlan`), `recovery.go`, `cloud_destroy.go`.
- Used by: `cli/doctor.go`, `cli/status.go`, `cli/down.go`, `cli/destroy.go`, `cli/recover.go`, `cli/list.go`, `cli/logs.go`.

### `internal/bootstrap` (remote install pipeline)

- Purpose: Render and execute the BYO/cloud install pipeline over SSH. Two entry points: `bootstrap.Apply` (persistent) and `bootstrap.ApplyEphemeral` (one-shot), both in `internal/bootstrap/install.go`.
- Contains: `install.go` (`Options` struct, `Apply`/`ApplyEphemeral`, `BaselinePackages` ~75 apt packages matching the GitHub-hosted Ubuntu 24.04 image, `mergePackages`, `isUbuntuLike`, `downloadRunnerCommand`, `SharedRunnerCacheRoot`, `ServiceNotActiveError`), `script.go` (`RenderDependencyFixScript`, `RenderInstallScript`, `RenderServiceScript`, `RenderEphemeralInstallScript`, `RenderEphemeralServiceScript`, `RenderEphemeralFinalizerScript`, `RenderEphemeralTTLTimerScript`), `image_setup.go` (`RenderImageSetupScript`, `ImageSetupVersion`), `package.go` (`PackageFor`, `RunnerVersion="2.334.0"`, pinned SHA256), `sudoers.go` (`SudoersFilePath="/etc/sudoers.d/runnerkit-installer"`, `RenderSudoersEntry`, `RemoteVisudoCheckScript`, `FoundationUserProbeScript`), `ci_sudoers.go` (runner service-user passwordless sudo for `apt-get`/`dnf`/`yum`).
- Used by: `cli/up.go`, `cli/upgrade_runner.go`, `cli/doctor.go`, `cli/recover.go`, `ops/doctor.go`.

### `internal/preflight` (host probe → typed findings)

- Purpose: Fixed catalog of host checks (SSH, OS release, arch, systemd, sudo non-interactive, disk ≥ 2 GiB, MemAvailable, swap, tools, network egress, time sync, runner conflict) returning a `Report` with per-check `Severity` (pass/warning/failure).
- Contains: `Run` entry point, `Report.Passed()`, `Report.Result(id)`, `CheckXxx` ID constants (`CheckSSHConnectivity`, `CheckSSHHostKey`, `CheckOSRelease`, `CheckArch`, `CheckSystemd`, `CheckPrivilege`, `CheckPrivilegePasswordReq`, `CheckPrivilegeCloudBootstrap`, `CheckPrivilegeNoSudo`, `CheckDisk`, `CheckHostMemAvailable`, `CheckHostSwap`, `CheckTools`, `CheckNetworkGitHub`, `CheckTime`, `CheckRunnerConflict`), `Options{AllowUnknownLinux, RequirePasswordlessSudo, RunnerName}`, `RequiredTools()`, `NormalizeArch`, `memWarnThresholdBytes` (overridable via `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES`).
- Used by: `cli/up.go`, `cli/upgrade_runner.go`, `cli/doctor.go`, `ops/doctor.go`.

### `internal/remote` (SSH executor + host key handling)

- Purpose: Abstract SSH transport behind the `Executor` interface so production uses local `ssh` tooling and tests use fakes.
- Contains: `executor.go` (`Executor`, `HostKeyProber` interfaces; `Command`, `Result`, `ProbeResult`, `RemoteError`, `UnavailableExecutor`), `system.go` (`SystemExecutor` shelling to `ssh`/`ssh-keyscan`, `selectHostKeyLine` deterministic key picker for ed25519 > ECDSA > RSA preference, `parseProcMeminfoViaAwk`, `sshArgs` always sets `StrictHostKeyChecking=no` + `UserKnownHostsFile=/dev/null`), `target.go` (`Target` + `ParseTarget` for `user@host[:port]` / `ssh://...`, `CanonicalHostKey`), `hostkey.go` (`HostKey`, `NormalizeHostKey`, `FingerprintSHA256`), `meminfo.go` (`ParseMeminfoAwkOutput`), `logging_executor.go` (`WrapWithLogging` decorator emitting slog).
- Used by: `bootstrap`, `preflight`, `ops`, `cli`, `provider/hetzner`.

### `internal/provider` + `internal/provider/hetzner` (cloud lifecycle)

- Purpose: Single `Provider` interface (`internal/provider/provider.go:13`) covering Validate → Plan → Provision → WaitReady → Describe → Destroy → VerifyDestroyed. `Registry` maps provider names to implementations; only Hetzner registered in v1.x.
- Contains: `provider.go` (interface + `ProvisionInput`, `ProvisionPlan`, `ProvisionResult`, `Machine`, `ProviderStatus`, `DestroyResult`, `VerificationResult`, `Registry`, `ProvisionError`), `profile.go` (Hetzner defaults `HetznerDefaultRegion="fsn1"`, `HetznerDefaultServerType="cpx22"`, `HetznerDefaultImage="ubuntu-24.04"`, `HetznerDefaultSSHUser="runnerkit-admin"`, `HetznerProvisionPlan`, `HetznerResourceNames`, `HetznerOwnershipTags`), `fake.go` (test double), `hetzner/client.go` (hcloud-go wrapper interface), `hetzner/credentials.go` (`HCLOUD_TOKEN`/`HETZNER_CLOUD_TOKEN` discovery + `MissingTokenError`), `hetzner/provision.go` (`Provider`, cloud-init user-data with `CloudInitUserDataVersion="runnerkit-cloud-init-v3"`), `hetzner/readiness.go` (`WaitReady` 150 × 2s = 5-min poll), `hetzner/describe.go`, `hetzner/destroy.go` (cascade-delete with 409 `must_be_unassigned` retry).
- Used by: `cli/up.go`, `cli/destroy.go`, `cli/status.go`, `cli/recover.go`, `ops/cloud_destroy.go`.

### `internal/state` (user-local JSON inventory)

- Purpose: Read/write the versioned, secret-free `state.json` under `~/.local/state/runnerkit/` (override via `$RUNNERKIT_STATE_DIR` or `$XDG_STATE_HOME`).
- Contains: `schema.go` (`State`, `RepositoryState`, `MachineRef`, `ProviderRef`, `CloudInventory`, `CleanupMetadata`, `SafetyMetadata`, `EphemeralMetadata`, `OperationCheckpoint`, `RunnerIdentity`, `AuthReference`, `CostProfileRef`, `SchemaVersion="2"`, `ErrRepositoryExists`), `store.go` (`Store`, atomic write via tmp+rename+fsync, mode 0600/0700, `Load` with side-by-side `*.backup-v<old>-<ts>` before migration, `ValidatePersistedJSON` rejecting raw secret keys), `migrations.go` (`Migrate`, `cmpVersion`, `migrateV1ToV2`, `ErrSchemaTooNew`), `config.go` (env-driven base dir).
- Used by: `cli`, `ops`, `provider/hetzner`.

### `internal/github` (GitHub API + auth + safety)

- Purpose: Resolve repo metadata, mint registration/removal tokens, list/delete runners, evaluate public-repo safety. Defaults to `gh` CLI for credential discovery; supports `RUNNERKIT_GITHUB_TOKEN` override.
- Contains: `service.go` (`Service`, `ServiceOptions`, `NewService`), `client.go` (HTTP wrapper), `auth.go` (credential discovery), `tokens.go` (registration/removal tokens), `runners.go` (`ListRunners`, `DeleteRunner`, `FindRunnerByName`), `safety.go` (`EvaluateSafety`, `SafetyDecision`, `SafetyOptions`, `SafetyCodePublicRisk`, `AllowPublicRepoRiskFlag`, `FineGrainedTokenRemediation`), `remote.go` (`ResolveTarget`, repo URL parse), `types.go` (`Repo`, `Runner`, `RunnerToken`, `PermissionStatus`, `AuthSource`, `CommandRunner`, `OSCommandRunner`).
- Used by: `cli` via the `GitHubService` interface declared in `internal/cli/up.go:58`.

### `internal/labels`, `internal/runmode`, `internal/workflow` (typed value objects)

- `internal/labels/labels.go`: `Build`, `LabelSet`, `WorkflowSnippet`, `RepoScopedLabel`, `EphemeralRunnerName`, `DefaultOS="linux"`, `DefaultArch="x64"`, `ModePersistent`/`ModeEphemeral`.
- `internal/runmode/mode.go`: `Normalize`, `Evaluate`, `Decision`, `Options`, `Tradeoffs`, `ProfilePersistentTrusted`/`ProfilePersistentRisky`/`ProfileEphemeralBYO`/`ProfileEphemeralCloud`, `WarningPublicForkPersistent`, `WarningEphemeralCloudBillable`, `DefaultEphemeralTTL=24h`.
- `internal/workflow/plan.go`: `Plan`, `Step`, `StepStatus`, `BootstrapPlan()`, `FoundationUpPlan()`, `Apply(ctx, plan, runner)`, `Checkpoint`, `StepRunner` interface, step ID constants.

### `internal/ui` + `internal/ux/*` (output + persisted UX)

- `internal/ui/output.go`: `Renderer`, `Format` (`FormatHuman`/`FormatJSON`), `TerminalCapabilities`, `Line`/`LineKind`, `Step`, `Warning`, `Error`, `JSON`, helpers `Success`/`WarningLine`/`ErrorLine`/`PromptLine`/`Next`/`Bullet`.
- `internal/ui/box.go`, `checklist.go`: ASCII/UTF-8 panels and progress checklists (`RenderBoxed`, `RenderChecklist`, `ChecklistStep`, `ChecklistStepStatus`).
- `internal/ui/prompt.go`, `cli_prompter.go`: `Prompt`, `Option`, `Prompter` (`Confirm`/`Select`), optional `PasswordPrompter`.
- `internal/ux/nextaction/nextaction.go`: schema-versioned `Action`/`MergePayload`/`ApplySchemaAndStage` for `next_actions` envelopes; `SeverityBlocking`/`Info`.
- `internal/ux/stage/stage.go`: `Stage` enum (`NoLocalState`, `Unknown`, `Error`, `Uninstalled`, `Installed`, `Registered`, `Running`); `InferFromObserved`, `InferFromDoctor`, `ActionsFromOpsNext`.
- `internal/ux/checkliststore/store.go`: persisted BYO `up`/`register` checklist under `<stateBase>/sessions/<id>.json` (resumable cross-run); `BYORegisterSessionID`.

### Cross-cutting infrastructure

`internal/errcodes`, `internal/redact`, `internal/rklog`, `internal/update`, `internal/testsupport` — see Cross-Cutting Concerns below.

## Data flow

### `runnerkit up --repo owner/name --host user@host` (BYO persistent)

1. `cmd/runnerkit/main.go` builds `cli.Dependencies` and calls `cli.NewRootCommand(...).Execute()`.
2. Cobra dispatches to `newUpCommand` / `runUp` (`internal/cli/up.go:68,99`).
3. `runUp` opens `rkstate.NewStore(deps.StateBaseDir)`.
4. `resolveUpRepo` → `deps.GitHub.Repository(ctx, repo)` validates repo metadata; failure → `ExitGitHubAuth=3`.
5. `resolveModeDecision` (`up.go:382`) optionally prompts setup-path (BYO/cloud) + mode (persistent/ephemeral) via `deps.Prompts.Select`; `runmode.Evaluate` returns typed `Decision` with safety profile + tradeoffs.
6. `gh.EvaluateSafety(repo, SafetyOptions{AllowPublicRepoRisk})` runs the public-repo persistent-runner guard; `enforceModeSafetyDecision` blocks or requires `--allow-public-repo-risk` / `--allow-ephemeral-byo-risk`.
7. `resolveSetupPath` (`up.go:622`) returns `setupPathBYO` when `--host` set.
8. `deps.GitHub.VerifyAuth` confirms Administration read/write + Metadata read (else `RKD-AUTH-004` `github_permission_denied`, exit 3).
9. `resolveBYOTarget` parses `--host` via `remote.ParseTarget`; `verifyTargetHostKey` runs `ssh-keyscan` (`remote.SystemExecutor.ProbeHostKey`) and compares against saved fingerprint (mismatch → `RKD-SSH-001`).
10. `preflight.Run` (`internal/preflight/checks.go:114`) executes the full host check catalog via `deps.RemoteExecutor`; `report.Passed()` gates progress (failure → `RKD-BOOT-011`).
11. When sudo requires a password and `RequirePasswordlessSudo=false`, the warning `host.privilege.password_required` triggers `RenderHostInstallRequired` (point user at `install.sh` / `runnerkit init`).
12. `autoDetectExtraPackages` calls `scanWorkflowExtraPackages` (`internal/cli/workflow_packages.go`) to scrape `apt-get install` lines from `.github/workflows/*.yml`. `resolveExtraPackages` merges CLI `--extra-packages` over auto-detected.
13. `buildBootstrapOptions` (`up.go:1599`) constructs `bootstrap.Options` with deterministic `InstallPath=/opt/actions-runner/<runnerName>`, `WorkDir=/var/lib/runnerkit/work/<runnerName>`, `ServiceUser="runnerkit-runner"`, `Package=bootstrap.PackageFor("linux", arch)`, `MissingTools=report.FixableTools`, `ExtraPackages`, `OSReleaseID`.
14. `--dry-run` short-circuits via `renderDryRun`.
15. `deps.GitHub.ListRunners` checks for runner-name collision; `isRunnerKitManagedRunner` lets re-runs keep their own registration (`config.sh --replace`).
16. `confirmBootstrapPlan` requires `--yes` or interactive confirm; else `RKD-CORE-001` (`input_required`, exit 6).
17. `deps.GitHub.CreateRegistrationToken` mints a single-use token; `renderer.Redactor().Register(redact.RunnerRegistrationToken, token.Token)` ensures it never leaks.
18. `bootstrap.Apply` (`internal/bootstrap/install.go:108`) runs the persistent step sequence via `remote.Executor.Run`:
    - `fix_dependencies` — `RenderDependencyFixScript(mergePackages(missingTools, BaselinePackages, extraPackages))`; skips `BaselinePackages` when `CloudProvisioned=true`.
    - `setup_runner_image` (Ubuntu/Debian only) — installs Node 20, Python, Go, Rust, Java 17, .NET 8, Docker CE, Chrome/Firefox/drivers, gh, cmake, ninja, zstd; writes `/var/lib/runnerkit/image-setup.json` with `ImageSetupVersion`.
    - `create_runner_user` — idempotent `useradd --system --create-home --shell /usr/sbin/nologin runnerkit-runner`.
    - `download_runner` — shared cache at `/opt/actions-runner/runnerkit-shared-bin/<version>/<filename>`, SHA256-verified, `tar xzf --skip-old-files` into the per-repo InstallPath.
    - `configure_runner` — runs `RenderInstallScript` via `sudo su -s /bin/bash - runnerkit-runner -c "...config.sh --unattended --url <repo> --token $RUNNERKIT_REGISTRATION_TOKEN --name <runnerName> --labels <csv> --work <workDir> --replace"`. Token flows via `Command.Env` / `RedactArgs`.
    - `install_service` — `RenderServiceScript`: stop+uninstall existing unit, `svc.sh install runnerkit-runner && svc.sh start && svc.sh status`.
    - `verify_service` — `cd <InstallPath> && sudo ./svc.sh status` (non-zero → `bootstrap.ServiceNotActiveError` → `runner_service_not_active`).
19. `waitForRunnerOnline` polls `deps.GitHub.ListRunners` with case-insensitive label match (Bug 16) until `PollTimeout` (60s default).
20. `buildBYORepositoryState` constructs `RepositoryState{Repo, Auth, Runner, Machine{Kind:"byo-ssh", HostKeyFingerprint, InstallPath, WorkDir, ServiceName}, Provider{Kind:"byo"}, Cleanup{ManagedPaths,GitHubRunnerID}, Safety, ExtraPackages, ImageSetupVersion, RunnerKitVersion}`.
21. `saveRepositoryState` → `store.SaveRepository(repoState, replace)` atomically writes `~/.local/state/runnerkit/state.json`; refuses raw-secret keys via `ValidatePersistedJSON`.
22. `renderCompletionHuman` / `upCompletionJSON` print the workflow `runs-on:` snippet and runner identity.

### `runnerkit up --repo owner/name --cloud hetzner` (Hetzner cloud)

1–6. Same as BYO through `resolveModeDecision`.
7. `resolveSetupPath` returns `setupPathCloud`; `runUp` delegates to `runCloudUp` (`internal/cli/up.go:659`).
8. `deps.GitHub.VerifyRunnerManagementRead` (read-only — no token minted yet).
9. `autoDetectExtraPackages` + `buildCloudProvisionInput` (`up.go:858`) construct `provider.ProvisionInput{RepoFullName, RunnerName, Labels, WorkflowSnippet, Profile=DefaultHetznerProfile(), SSHAllowedCIDR, PublicKey=resolveCloudPublicKey(opts), StateID, CreatedAt, Mode, ExtraPackages}`.
10. `registerKnownCloudProviderSecrets` registers `HCLOUD_TOKEN` / `HETZNER_CLOUD_TOKEN` with the redactor.
11. `cloudProvider.Validate` (`internal/provider/hetzner/provision.go:76`) → `lookupProfile` to verify region/server-type/image (else `RKD-PROV-004`).
12. `cloudProvider.Plan` returns `ProvisionPlan` with Resources (server billable, ssh_key, firewall, public IP), tags `runnerkit=true managed=true repo=... runner=... mode=... created_at=...`, `FutureDestroyCommand=runnerkit destroy --repo ...`.
13. `--dry-run` short-circuits via `renderCloudProvisionPlan`.
14. `confirmCloudStateReplaceBeforeProvision` + `confirmCloudProvisionPlan` require `--yes` (or interactive type-to-confirm).
15. `cloudProvider.Provision`:
    - `CreateSSHKey`, `CreateFirewall` (rules from `firewallRules(SSHAllowedCIDR)`), `CreateServer` with `PublicNet{EnableIPv4: true, EnableIPv6: true}` (auto-allocated PrimaryIPs carry `AutoDelete: true` for cascade-delete; Bug 30).
    - `UserData=cloudInitUserData(profile.SSHUser, publicKey, extraPackages)` writes scoped sudoers + installs `BaselinePackages` + extras + writes `/var/lib/runnerkit/cloud-init.json` with `CloudInitUserDataVersion="runnerkit-cloud-init-v3"`.
    - On error after billable resources exist: wraps `*ProvisionError`; caller `saveCloudPendingRepository` (status `cloud_provision_pending`) so `runnerkit destroy` can clean up.
16. `cloudProvider.WaitReady` (`internal/provider/hetzner/readiness.go:22`) polls Hetzner action → server running with public IP (150 × 2s = 5 min).
17. `waitCloudTargetReady` (`up.go:922`) probes SSH host key as `root` (Hetzner injects root SSH key at provision), runs `runCloudInitWaitWithRetry` (`up.go:1027`) executing `cloud-init status --wait` — tolerates exit 2 (recoverable error), rejects `status: error`. Then `preflight.Run` with `RequirePasswordlessSudo=true` (else `host.privilege.cloud_bootstrap` failure).
18. `buildBootstrapOptions(..., cloudProvisioned=true)` — same `bootstrap.Apply` pipeline as BYO but `BaselinePackages` is skipped (cloud-init already installed them).
19. `bootstrap.Apply` runs against `readyMachine.Target`; registration token minted via `deps.GitHub.CreateRegistrationToken` after readiness so its 1-hour TTL covers install.
20. `waitForRunnerOnline` → `buildCloudRepositoryState` (`Machine.Kind="cloud-ssh"`, `Provider.Kind="hetzner"` + `Cloud{ServerID, ServerName, Region, ServerType, Image, PublicIPv4/IPv6, PrimaryIP*ID, PrimaryIP*AutoDelete, SSHKey*, Firewall*, CostProfile, CloudInitVersion}`) → `store.SaveRepository(..., replace=true)`.
21. `renderCloudCompletionHuman` / `cloudCompletionJSON` print `runs-on:` snippet, future destroy command, cost estimate caveat.

### State management

- Single user-local JSON at `~/.local/state/runnerkit/state.json` (override via `$RUNNERKIT_STATE_DIR` or `$XDG_STATE_HOME`).
- `SchemaVersion = "2"`. Loads older versions through `Migrate` after writing a `*.backup-v<old>-<ts>` snapshot; refuses to mutate newer schemas with `ErrSchemaTooNew` (`ExitStateSchemaTooNew=7`).
- Atomic write: tmp file → fsync → rename → fsync parent dir; mode 0600 (file) / 0700 (dir).
- `<stateBase>/sessions/<id>.json` (`internal/ux/checkliststore`) persists per-(repo,host) BYO progress for resumable `up`/`register`.
- `<stateBase>/config.json` (`internal/cli/userconfig.go`) persists `DoctorIgnoreFindingIDs` and other user preferences.

## Key abstractions

**`Provider` interface** (`internal/provider/provider.go:13`) — Cloud lifecycle boundary independent of vendor. Methods: `Name`, `Validate`, `Plan`, `Provision`, `WaitReady`, `Describe`, `Destroy`, `VerifyDestroyed`. Implementations: `internal/provider/hetzner/provision.go` (production), `internal/provider/fake.go` (tests). Registered via `provider.NewRegistry(hetzner.NewProvider(nil, hetzner.WithLogger(logger)))` in `normalizeDependencies` (`internal/cli/root.go:80`).

**`remote.Executor` interface** (`internal/remote/executor.go:9`) — SSH transport abstraction. Methods: `Probe(ctx, target) ProbeResult`, `Run(ctx, target, cmd) Result`. Implementations: `SystemExecutor` (production, shells to `ssh`/`ssh-keyscan`), `UnavailableExecutor` (zero-value safety), test fakes in `internal/remote/fake_test.go`, `internal/testsupport/remote.go`. Optional `HostKeyProber` interface for `ssh-keyscan` probes. Wrapped by `remote.WrapWithLogging` in `normalizeDependencies` to emit slog `remote.command.*` events.

**`bootstrap.Options`** (`internal/bootstrap/install.go:14`) — Single struct carrying everything to install a runner — `RunnerName`, `RepoURL`, `Labels`, `InstallPath`, `WorkDir`, `ServiceUser`, `RunnerToken`, `Package`, `MissingTools`, `ExtraPackages`, `CloudProvisioned`, `OSReleaseID`, `ImageSetupVersion`, `RunnerCacheRoot`, `Mode`, `EphemeralTTL`, `LogArchivePath`, `FinalizerPath`, `EphemeralServiceName`, `EphemeralTTLServiceName`, `EphemeralTTLTimerName`. `normalizeOptions` populates defaults. Persisted into `state.RepositoryState.Machine` so `runnerkit upgrade-runner` can re-render the same pipeline.

**Bootstrap Apply pipelines**:
- `Apply` (persistent): `fix_dependencies` → `setup_runner_image`? → `create_runner_user` → `download_runner` → `configure_runner` → `install_service` → `verify_service`.
- `ApplyEphemeral`: `fix_dependencies` → `setup_runner_image`? → `create_runner_user` → `download_runner` → `configure_ephemeral_runner` → `install_ephemeral_finalizer` → `install_ephemeral_service` → `install_ephemeral_ttl_timer` → `verify_ephemeral_service`.
- Both serialize as `remote.Command{ID, Script, Sudo, Env, RedactArgs, Timeout}`. Service-step failures surface as `bootstrap.ServiceNotActiveError{Err, CommandID, Stderr}` so CLI attaches failing remote stderr.

**`workflow.Plan`** (`internal/workflow/plan.go:48`) — Static step list used for `--dry-run` previews and JSON envelopes. `BootstrapPlan()` returns the persistent step names; `FoundationUpPlan()` returns `resolve_repo, verify_auth, check_safety, preview_state, save_state` for the BYO foundation flow.

**`state.RepositoryState`** (`internal/state/schema.go:20`) — Canonical persisted record per repo. Embeds `Repo`, `Auth`, `Runner`, `Machine`, `Provider`, `Cleanup`, `Safety`, `Ephemeral`, `Operations`, `ExtraPackages`, `ImageSetupVersion`, `RunnerKitVersion`, `RunnerTemplateVersion`, `ServiceTemplateVersion`, `CreatedAt`, `UpdatedAt`. Read by `doctor` / `status` / `down` / `destroy` / `recover` / `upgrade-runner` / `logs`.

**`preflight.Report` + `preflight.Result`** (`internal/preflight/checks.go:69,87`) — Typed catalog of host findings. `Result{Check, ID, Severity, Message, Remediation, Fixable}`; `Report.Passed()` returns false only on `SeverityFailure`; warnings (e.g. `host.privilege.password_required`, `host.mem_available`, `host.swap`, `host.tools` with `Fixable=true`) allow progress. `Report.FixableTools` is consumed by `bootstrap.Options.MissingTools`.

**`ops.DoctorReport` + `ops.Finding`** (`internal/ops/doctor.go:13,21`) — Doctor command output. `BuildDoctorReport(repoState, observed, checks, hostHints)` correlates state + GitHub + SSH + systemd + preflight + journal heuristics into typed findings with stable IDs (`state_present`, `github_runner_offline`, `service_failed`, `service_missing`, `ssh_unreachable`, `ssh_host_key_mismatch`, `label_drift`, `provider_drift`, `install_path_missing`, `work_dir_missing`, `disk_low`, `tools_missing`, `network_github_failed`, `time_unsynchronized`, `host_mem_low`, `host_swap_constrained`, `cleanup_pending`, `runner_version_stale`, `ephemeral_{waiting,busy,completed,ttl_expired,cleanup_pending}`, `byo_host_prepared`, `host_incident_hints`). Each routes to an `errcodes.Code` so remediation appends `See: <URL>`.

**`errcodes.Code`** (`internal/errcodes/codes.go:36`) — Stable RKD identifier with docs anchor. Numbers never reused or renumbered. `URL(code)` resolves the absolute docs URL via `internal/errcodes/url.go`; `FormatLine(code)` returns `code.ID + " See: " + URL(code)` for appending to remediation slices. `Registry` slice walked by tests for uniqueness / docs anchor coverage.

## Entry points

**Production binary** — `cmd/runnerkit/main.go`. Triggers: `runnerkit <subcommand>` from a user shell or the installed Homebrew cask. Responsibilities: Build `cli.Dependencies`, execute root Cobra command, map errors to typed exit codes (`internal/cli/exit.go`).

**Subcommand dispatch** — `internal/cli/root.go:115` (`NewRootCommand`). All subcommands registered in `root.AddCommand(...)` block (`root.go:163-176`): `version`, `init`, `register`, `up`, `list`, `status`, `logs`, `doctor`, `recover`, `down`, `destroy`, `state`, `upgrade`, `upgrade-runner`. Running `runnerkit` with no subcommand → `runFirstRunWizard` (`internal/cli/wizard.go:38`) when state is empty, else `cmd.Help()`.

**Maintainer smoke binaries** — `cmd/_smokebin/destroy_verify/main.go` (D-12 gate 2 post-`destroy` resource-404 verification) and `cmd/_smokebin/empty_precheck/main.go` (D-12 gate 1 refusal if Hetzner project already holds `runnerkit-*` resources). Invoked from `scripts/smoke/cloud-end-to-end.sh` / `hetzner-destroy-verify.sh` / `hetzner-empty-precheck.sh`; never wired into CI (`Makefile` `smoke-live-*` targets are maintainer-only).

**One-time host install** — `install.sh` — `curl -fsSL <url> | sudo bash`; installs `/etc/sudoers.d/runnerkit-installer` validated by `visudo`. Mirrors `bootstrap.RenderSudoersEntry` so BYO `up` / `register` can run non-interactively.

## Error handling

**Strategy:** Every user-facing failure path resolves to either (1) `*cli.ExitError{Code, Err}` returned from a `runE` function and walked by `cli.ExitCode(err)` (`internal/cli/exit.go:49`), or (2) a pre-render `renderer.Error(code, message, remediation)` block followed by `NewExitError(...)`.

**Exit code contract** (`internal/cli/exit.go:9`):
- `0` success, `1` unexpected, `2` invalid input/flag, `3` GitHub auth/permission, `4` safety gate (preflight failure, bootstrap failure, runner timeout), `5` state I/O, `6` input required, `7` state schema too new, `130` canceled (`context.Canceled`).

**Patterns:**
- Remote command failures surface as `remote.RemoteError{CommandID, ExitCode, Message}` from `SystemExecutor.Run`.
- Bootstrap service failures wrap as `bootstrap.ServiceNotActiveError{Err, CommandID, Stderr}` so the CLI attaches actual remote stderr to remediation (Bug 12, Plan 06-07 attempt-9).
- Provider failures after billable resources exist wrap as `*provider.ProvisionError{Stage, Result, Err}`; the CLI persists a pending-cleanup `RepositoryState` so `runnerkit destroy` can clean up.
- All long-form remediation appends `errcodes.FormatLine(code)` so the user has both the RKD ID and a docs URL.
- `--json` errors emit `{"ok": false, "error": {"code": "<snake>", "message": "..."}}` plus optional fields (`ssh-preflight`, `findings`, `next_actions`); `next_actions` / `host_incident_hints` / `findings` are always JSON arrays (never `null`) per the smoke contract (`scripts/smoke/assert-doctor-json-contract.sh`).

## Cross-cutting concerns

**Logging** — `internal/rklog/rklog.go` — `NewFromEnv(w)` returns a `*slog.Logger`. Driven by `RUNNERKIT_LOG=debug|info|warn|error|off` (default off → discard handler) and `RUNNERKIT_LOG_DEST=stderr|stdout|file:/path` (default stderr). JSON handler always. `AddSource=true` at debug. `PersistentPreRun` in `root.go:134` emits `runnerkit.cli.begin{command, version}`. `remote.WrapWithLogging` emits `remote.command.*` events per SSH call. Hetzner provider emits `hetzner.provision.{begin,end}` and `hetzner.destroy.{begin,summary}` with bounded fields.

**Validation** — `runmode.Normalize` accepts only `persistent` / `ephemeral`. `remote.ParseTarget` accepts `user@host`, `user@host:port`, `ssh://...`; rejects empty user, port outside `[1,65535]`. `preflight.NormalizeArch` accepts `x86_64`/`amd64`/`x64` → `x64` and `aarch64`/`arm64` → `arm64`. `bootstrap.PackageFor("linux", arch)` accepts only `linux/x64` and `linux/arm64`. `cli.isValidPackageName` (`workflow_packages.go`) accepts only alphanumeric + `.+-_:` to harden `--extra-packages`. `state.ValidatePersistedJSON` (`store.go:226`) walks marshaled JSON and refuses raw `token` / `registration_token` / `private_key` / `provider_credential` keys.

**Authentication** — GitHub via `internal/github/service.go` — `Service.client` reads `RUNNERKIT_GITHUB_TOKEN`, falls back to `gh auth token` subprocess via `CommandRunner`. Cloud via `internal/provider/hetzner/credentials.go` — `HCLOUD_TOKEN` or `HETZNER_CLOUD_TOKEN`; both registered with redactor. SSH defers to local user agent + `~/.ssh/id_ed25519` / `~/.ssh/id_rsa` discovered by `resolveCloudPublicKey` (`internal/cli/up.go:881`); explicit `--ssh-key` overrides. `sshArgs` always sets `StrictHostKeyChecking=no` + `UserKnownHostsFile=/dev/null` because RunnerKit verifies host keys explicitly via `ssh-keyscan` → state fingerprint.

**Update notifications** — `internal/update/check.go` polls GitHub Releases (cached under state dir) and prints a non-blocking notice via `maybeShowUpdateNotice` on every long-running command. Disabled in `--json`.

**Test scaffolding** — `internal/testsupport/` — `golden.go` (golden-file helpers), `output.go` (capture stdout/stderr), `state.go` (in-memory state), `github.go` (fake GitHub service), `remote.go` (fake executor).
