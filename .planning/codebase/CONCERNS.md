# CONCERNS — RunnerKit tech debt, risks, and fragile areas

> Reference for planning. Honest, not marketing. Every claim cites a file path. Active deferred work lives in `.planning/seeds/`.

**Analysis date:** 2026-05-17

## Scope

Bootstrap/sudoers, SSH wrapping, Hetzner cloud, secrets, large CLI functions, deferred work in `CLAUDE.md` + seeds + Phase 06 bug history.

**Code volume:** ~15.4k production LOC / ~13.0k test LOC. Test/production ratio is healthy.

**Notable:** **zero** `TODO`/`FIXME`/`HACK`/`XXX` markers anywhere in `internal/`, `cmd/`, `scripts/` (only `mktemp XXXXXX` template strings and `RKD-XXX-NNN` doc placeholders). Real debt lives in planning docs, seeds, and load-bearing code comments — not as inline markers.

## Tech debt

### Deferred BYO bootstrap architecture (SEED-001)

- **Issue:** `runnerkit up` / `register` conflate one-time privileged install (Path A/B/C) with repeated unprivileged lifecycle ops. Every BYO operation negotiates sudo from the maintainer's machine over SSH — Path B (TTY prompt) or Path C (scoped sudoers). Phase 06 attempts 2–19+ patched this surface (Bugs 1–32).
- **Files:** `internal/cli/up.go` (2328 LOC), `internal/bootstrap/sudoers.go`, `internal/preflight/checks.go` lines 160–218, `install.sh`, `internal/bootstrap/ci_sudoers.go`.
- **Impact:** Bug 31 (preflight `sudo -n true` not in byo-prepare allowlist) blocked the v1.0.0 tag for an entire attempt cycle. Non-interactive consumers (CI, agents, MCP) are locked out of Path B and forced through Path C, but Path C has had its own drift (Bug 27: hardcoded `/opt/runnerkit-runner/svc.sh` never matched real install dir `/opt/actions-runner/runnerkit-*/`).
- **Fix:** Land SEED-001 (`v1.1-01-bootstrap-lifecycle-split`). Net delta is **~−200 LOC** because it deletes `byo_prepare.go`, `promptSudoPasswordForPathB`, `RUNNERKIT_SUDO_PASSWORD` threading, and all Path B/C docs. See `.planning/seeds/SEED-001-bootstrap-lifecycle-split.md`.

### `runnerkit byo-prepare` command exists in comments only

- `internal/bootstrap/sudoers.go:52` says *"so Path C (`runnerkit byo-prepare`) works end-to-end"*; `install_integration_test.go:160` mentions it; but `internal/cli/root.go:163-176` does NOT register `newByoPrepareCommand` anywhere (`grep "Use:.*byo"` returns zero hits). The real Path C entry point is `curl -fsSL <install.sh> | sudo bash`.
- **Files:** `internal/cli/root.go` lines 163–176, `internal/bootstrap/sudoers.go` lines 30–53.
- **Impact:** Docs/comments trail leads users to a phantom CLI command.
- **Fix:** Either implement `newByoPrepareCommand` wrapping `RenderSudoersEntry` + `RemoteVisudoCheckScript` over SSH, or remove the comments and point everywhere at `install.sh`. SEED-001 obsoletes both.

### `install.sh` sudoers allowlist drifts from `RenderSudoersEntry`

- `install.sh` lines 34–46 list a smaller allowlist than `internal/bootstrap/sudoers.go:59-77` produces. Missing from `install.sh`: `/usr/bin/tee`, `/usr/bin/gpg`, `/bin/mkdir`, `/usr/bin/mkdir`, `/usr/bin/unzip`, `/usr/sbin/usermod`, `/usr/bin/dpkg`, `/usr/bin/add-apt-repository`. Those were added to the Go renderer (Plan 06-14) but never back-ported to the bash installer.
- **Impact:** A host prepared via `curl ... install.sh | sudo bash` may fail `SudoersIsPrepared` byte-comparison and silently break `setup_runner_image` steps that use `tee` / `gpg` / `add-apt-repository` (NodeSource, Docker, Chrome keys — `internal/bootstrap/image_setup.go` lines 43–155). Cloud path is safe because `cloudInitUserData` calls `bootstrap.RenderSudoersEntry` directly.
- **Fix:** Generate `install.sh` from `RenderSudoersEntry` via `go:embed` / `go generate`, or add `TestInstallShellMatchesRenderSudoersEntry` that diffs the two.

### `.runnerkit/config.yaml` loader missing

- `internal/state/config.go` lines 5–17 defines `ProjectConfig` / `ProjectDefaults` with `yaml` struct tags. No YAML library in `go.mod`. `internal/cli/up.go:189` and `:682` hardcode `resolveExtraPackages(opts.extraPackages, nil, autoDetected)` — project-defaults parameter is always `nil`.
- **Impact:** Docs promise project-level config; users get no behavior change.
- **Fix:** Add `gopkg.in/yaml.v3` + a `LoadProjectConfig(cwd)` loader, or strip the type.

### `runUp` (283 LOC) and `runCloudUp` (193 LOC) are too large

- `internal/cli/up.go` lines 99–381 and 659–851. Both interleave preflight, GitHub permission checks, mode-decision rendering, token creation, state checkpointing, host-key probing, `bootstrap.Apply` / `ApplyEphemeral`, and online verification.
- **Impact:** Every Phase 06 gap-closure (Bugs 4–31) had to thread through one of these. Coverage exists across 7 test files but the bodies resist line-level reasoning.
- **Fix:** SEED-001 splits `up` into thin dispatcher (~100 LOC). Interim: extract the BYO bootstrap branch into `runBYOBootstrap(...)`.

### Cloud-init YAML built via `fmt.Sprintf` string concatenation

- `internal/provider/hetzner/provision.go` lines 351–405. `sudoersBlock` is two `strings.Builder` loops; apt packages list is hand-rolled `"  - " + pkg + "\n"`. No structural escape for `publicKey`, baseline-package names, or anything from `bootstrap.RenderSudoersEntry`.
- **Impact:** Cloud-init failure is the most expensive RunnerKit failure mode — billable Hetzner resources stay up while readiness fails. Bug 29 (Plan 06-12) was a related class. The constant `CloudInitUserDataVersion = "runnerkit-cloud-init-v3"` implies v1/v2 had user-visible problems.
- **Fix:** Use `gopkg.in/yaml.v3` for structure-level escaping; minimum, add a smoke test that runs the rendered user-data through `python3 -c "import yaml; yaml.safe_load(...)"` and fails on malformed output.

## Known bugs

None open in source. Bugs 1–34 from `06-GAP-byo-sudo-handling.md` and Plans 06-13/14/16 are fixed per `.planning/STATE.md` + `.planning/ROADMAP.md`. Bug 31 (preflight allowlist) is most recent — fix landed in Plan 06-13; smoke-green pending in Plan 06-07 attempt-20.

### Latent bug — no signal handling

- `internal/cli/up.go:102` uses `context.Background()` for the entire `runUp`; same in `runCloudUp`, `runDown`, `runDoctor`, `runDestroy`, `runRecover`, `runLogs`, `runStatus`, `runList`, `runUpgradeRunner`. **No `signal.NotifyContext`** anywhere in `cmd/runnerkit/main.go` or `internal/cli/root.go`. Ctrl-C mid-`runnerkit up --cloud hetzner` will NOT cancel `client.CreateServer` or the cloud-init wait — billable resources keep provisioning after CLI exits.
- **Files:** `cmd/runnerkit/main.go:44`, `internal/cli/root.go` lines 123–179.
- **Fix:** Wire `ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` in `main.go` and thread it down.

## Security considerations

### SSH host-key verification disabled in `SystemExecutor`

- `internal/remote/system.go` lines 206–226: every SSH invocation sets `StrictHostKeyChecking=no` + `UserKnownHostsFile=/dev/null`. Verification is re-implemented via `ssh-keyscan` (lines 94–115) and persisted as `state.MachineRef.HostKeyFingerprint`. Comment at lines 207–213 explains the choice (Plan 06-16 / Bug 34: stale known_hosts entries on recycled cloud IPs).
- **Mitigation:** `ProbeHostKey` + state persistence + `host_key_match` check; `selectHostKeyLine` (Plan 06-11 Bug 24) ensures deterministic fingerprints across calls.
- **Recommendation:** First connection is TOFU — there's no out-of-band fingerprint verification (Hetzner Console host-key display, DNS SSHFP). Document TOFU window explicitly in `docs/troubleshooting/ssh.md`. For cloud, consider reading host fingerprint from Hetzner server console output. Better: once state has a recorded fingerprint, write a per-target known_hosts under `$RUNNERKIT_STATE_DIR/known_hosts/<repo>` and switch lifecycle ops to `-o UserKnownHostsFile=<that> -o StrictHostKeyChecking=yes`.

### Bare `HCLOUD_TOKEN` value not registered with redactor

- `internal/redact/redact.go:53` pattern is `regexp.MustCompile(\`\bHCLOUD_TOKEN=[^<\s][^\s]*\`)` — only matches when literal `HCLOUD_TOKEN=` precedes the value. The token VALUE itself is never registered via `Redactor.Register(redact.ProviderCredential, source.Token)`. `internal/provider/hetzner/credentials.go` lines 26–43 resolves the token but doesn't redact it.
- **Mitigation:** hcloud-go errors typically include the env-var form.
- **Recommendation:** In `Provider.client()` (`provision.go:265-278`), call `redactor.Register(redact.ProviderCredential, source.Token)`. Same for `RUNNERKIT_GITHUB_TOKEN` from `internal/github/service.go:43` (currently only the `gh[pousr]_…` / `github_pat_…` shapes are pattern-matched; any other GitHub token format slips through).

### Sudoers wildcard `/opt/actions-runner/runnerkit-*/svc.sh`

- `internal/bootstrap/sudoers.go:75` grants NOPASSWD on this glob. Sudoers `*` doesn't match `/`, so it's bounded to a single directory level (comment at lines 41–42). Anyone with write access to `/opt/actions-runner/runnerkit-FOO/svc.sh` can escalate to passwordless root. The dir is created by `sudo install -d -o root -g root` in `install.go:235`, so production is safe.
- **Recommendation:** Document the trust boundary in `docs/troubleshooting/bootstrap.md`. Add a `sudoers_test.go` invariant that the rendered entry NEVER contains a `*` matching anything other than svc.sh under that one directory level.

### Package-name validator allows `+` and `:` but is fed auto-detected workflow input

- `internal/cli/up.go` lines 2251–2263: `isValidPackageName` allows `[a-zA-Z0-9-._:+]`. `internal/cli/workflow_packages.go` lines 50, 83 lets these flow into `apt-get install $pkg`. A `apt-get install foo:armhf` from a workflow gets installed on the runner host.
- **Mitigation:** `extractAptPackages` rejects shell metacharacters (`|>&;$()`); it only reads files in the repo's own `.github/workflows/` (already trusted to run CI). Trust boundary matches.
- **Recommendation:** The CLI prints "Auto-detected N workflow package(s)" only in non-JSON mode (`up.go:2298`). Make it always print, including in `--yes` and JSON modes, so users see what gets installed.

## Performance bottlenecks

### Cloud-init host-key probe budget hardcoded at 5 minutes

- `internal/provider/hetzner/provision.go` lines 593–596: `defaultHostKeyProbeAttempts=60` × `defaultHostKeyProbeInterval=5s` = 300s. Hetzner cloud-init is 30–90s typical, but `setup_runner_image` (`internal/bootstrap/image_setup.go`) installs Node + Python + Go + Rust + Java + .NET + Docker + Chrome + Firefox + geckodriver + chromedriver + cmake + ninja + zstd + gh — all serially in `runcmd`. Cold boot can exceed 5 minutes on cpx22.
- **Fix:** Bug 29 added `RUNNERKIT_CLOUD_INIT_TIMEOUT` for the wait, but not the host-key probe. Either expose `RUNNERKIT_HOST_KEY_PROBE_TIMEOUT`, or split into two phases (short for ssh-keyscan, long for full cloud-init).

### `setup_runner_image` is one giant single-shot script

- `internal/bootstrap/image_setup.go` lines 21–170: ~150 lines installing ~12 tools serially. Partial failures abort the whole script; per-tool idempotency exists but isn't surfaced. Atomic marker (`/var/lib/runnerkit/image-setup.json`) treats setup as binary.
- **Fix:** Break into separate `remote.Command` IDs (`setup_node`, `setup_docker`, ...) so `bootstrap.Result.Commands` shows which step failed; or emit per-tool status JSON for `doctor` to read.

### Preflight runs ~12 SSH round-trips serially

- `internal/remote/system.go` lines 18–51: separate SSH invocations for `uname -s`, `uname -m`, `cat /etc/os-release`, `test -d /run/systemd/system`, one `command -v` per tool, `sudo -n true`, `df`, meminfo, time-sync. Each has ConnectTimeout=10s. No SSH multiplexing (`exec.Command("ssh", …)` fresh each time).
- **Fix:** Build a single shell script that emits structured KEY=VALUE output, or enable `-o ControlMaster=auto -o ControlPath=...`. Preflight currently consumes 5–15s wall-clock per BYO `up`.

## Fragile areas

### Path C scoped sudoers must stay in lockstep with `RenderInstallScript`

- **Files:** `internal/bootstrap/sudoers.go` lines 59–77, `internal/bootstrap/install.go` lines 217–250 (`downloadRunnerCommand`), `internal/bootstrap/script.go` lines 30–110.
- **Why fragile:** Bug 32 (Plan 06-14) — preflight passed because the probe was in the allowlist, but bootstrap failed because `sudo curl`, `sudo sha256sum -c -`, `sudo chown`, `sudo rm`, `sudo su -s /bin/bash -` were missing from the allowlist. Every new `sudo <cmd>` in any `Render*Script` needs a matching sudoers entry; no compile-time check binds them.
- **Safe-modification rule:** Adding a new `sudo <cmd>` requires updating (a) `RenderSudoersEntry`, (b) `install.sh` lines 34–46, (c) cloud-init (indirect via `RenderSudoersEntry`), (d) bumping `CloudInitUserDataVersion`.
- **Gap:** No test enforces this binding.

### Cloud destroy depends on `AutoDelete=true` cascade

- **Files:** `internal/provider/hetzner/provision.go` lines 142–186 (`PublicNet: &hcloud.ServerCreatePublicNet{EnableIPv4: true, EnableIPv6: true}` with `IPv4=nil, IPv6=nil`), lines 473–500 (state records `PrimaryIPv4AutoDelete`), `internal/provider/hetzner/destroy.go`.
- **Why fragile:** Bugs 26 + 30. Anyone who adds explicit `IPv4: &hcloud.PrimaryIP{...}` to `ServerCreateOpts` will lose `AutoDelete=true`, `destroy.go` skips `DeletePrimaryIP` based on stale state, resources leak. Comment at `provision.go:144-153` is load-bearing.
- **Gap:** No integration test asserts `PrimaryIPv4AutoDelete=true` when `IPv4=nil` AND `false` when explicit `*PrimaryIP` is passed.

### Probe-classify split in `preflight.Run`

- `internal/preflight/checks.go` lines 160–218. Bug 28 (Plan 06-12): the err-guard for `probe_sudo_n` was doing `if err != nil { return }` and missing `*exec.ExitError`-wrapped `rc=1 + stderr="sudo: a password is required"`. Current code at lines 194–217 inspects stderr regardless of err. Adding a new stderr marker (e.g. localized non-English sudo) needs a new `case strings.Contains(...)`.
- **Coverage:** `checks_bugfix_test.go` (144 LOC) is explicitly named after regressions — suggests this path keeps re-breaking.

### ssh-keyscan algorithm precedence

- `internal/remote/system.go` lines 137–174 (`selectHostKeyLine`). Bug 24 (Plan 06-11) — picking different host-key lines across two scans produced different fingerprints. Fixed with algorithm preference map. Changing existing ordering would break every persisted `MachineRef.HostKeyFingerprint` (false `host key mismatch`).

## Scaling limits

### State file is a single JSON document

- `internal/state/store.go` lines 14–112: reads/writes entire `State.Repositories` slice as one file (`$XDG_STATE_HOME/runnerkit/state.json`). Multi-repo (SEED-002, v1.2.0) means N repos × M checkpoints × ephemeral metadata in one file. Sub-MB at ~50 repos; noticeable at ~500. Not urgent — product targets solo developers.

### Hetzner provisioning is one VM per `runnerkit up --cloud`

- Per `CLAUDE.md`: cloud is one-server-per-up unless user manually points a second repo at an existing machine. Cost scales linearly. Scaling path = revision of SEED-002 for cloud lifecycle.

### `setup_runner_image` cold boot adds ~3–5 minutes per cloud `up`

- Hits the 10-minute stopwatch budget (`docs/release-process.md` lines 181–223). Repeated `destroy → up` cycles repay the cost each time.
- **Scaling path:** Build a custom Hetzner image (Packer) with `setup_runner_image` pre-baked, select via `--cloud-image`. Out of scope for v1.

## Dependencies at risk

### `hcloud-go v1.59.2` (NOT v2)

- `go.mod:6`. Per Plan 04-02 decision in `STATE.md`. v1 is in maintenance; v2 is upstream current. Future Hetzner API changes may land in v2 only. Migration touches `internal/provider/hetzner/{client,provision,destroy,readiness,describe,credentials}.go`.

### Bundled runner pinned to `2.334.0`

- `internal/bootstrap/package.go:5` + SHA256s inline at lines 25, 34. GitHub deprecates old runners. Plan 06-02 added `runnerkit upgrade-runner` but users must run it manually. Per `docs/release-process.md:117`, bumps are a "separate PR." Recommend: scheduled CI job that opens a PR on new actions/runner releases.

### `golang.org/x/term v0.10.0` is old

- `go.mod:9`. Blocks future Windows host support.

## Missing critical features

- **No signal handling / graceful cancellation** (see Latent bugs above). Ctrl-C during cloud provision = orphan billable resource.
- **No `runnerkit destroy --orphans` / discovery mode.** `cmd/_smokebin/empty_precheck/main.go` lists `runnerkit-*` Hetzner resources but is smoke-only, not wired into production CLI. If `state.json` is lost, the user has no CLI command to discover and destroy orphans.
- **No `runnerkit byo-prepare`** despite extensive references (see Tech Debt).

## Test coverage gaps (ordered by priority)

1. **Cloud-init user-data YAML schema validation** (**High** — billable orphan risk). `internal/provider/hetzner/provision_test.go` lines 140–170 only asserts substrings. No YAML parse or `cloud-init schema --config-file -` check.
2. **Sudoers ↔ install-script command-set drift** (**High** — proximate cause of v1.0.0 release blocker). No test enforces every `sudo <cmd>` in `script.go` + `downloadRunnerCommand` is covered by `RenderSudoersEntry`. Bug 32 exactly this gap.
3. **`install.sh` ↔ `RenderSudoersEntry` byte equality** (**High** — confirmed drift today).
4. **Multi-line / continuation handling in `extractAptPackages`** (Medium). 167 LOC of tests cover basics; mixed `&&`-chained and `RUN:`-block parsing less covered.
5. **Long-running cloud destroy with retry budget** (Medium). `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT` default 30s may be aggressive in cold Hetzner DCs.
6. **`ApplyEphemeral` failure recovery** (Medium). 8 sequential commands; mid-sequence failure leaves partial install with no test asserting recoverable state.

## Release-process foot-guns

From `docs/release-process.md` + `CLAUDE.md`:

- **Tag pushes from forks silently produce broken releases:** GitHub strips OIDC `id-token: write` on fork PRs; cosign signing fails. Workflow runs but cannot sign. Add a tag-validation step that fails fast if `github.repository != 'accidentally-awesome-labs/runnerkit'`.
- **HOMEBREW_TAP_GITHUB_TOKEN PAT was pasted into chat during Plan 06-01 closure** (`STATE.md` 2026-05-08). Maintainer was advised to rotate before public v1.0.0 tag. Status of rotation unknown from this analysis — must be confirmed before tagging. **High priority before any public release.**
- **`make smoke-live` requires real PAT + real HCLOUD_TOKEN + real billable resources.** D-11 enforced by absence — targets are NOT in any `.github/workflows/*.yml`. Maintainer discipline alone gates the v1.0.0 promise. Consider a release-workflow `before:` hook that greps `RELEASE-NOTES-vX.Y.Z.md` for a non-empty stopwatch table.
- **Live BYO smoke needs a hardware TTY until Plan 06-13 ships** (per `MEMORY.md`). Code landed in `ROADMAP.md` line 153; smoke-green confirmation pending Plan 06-07 attempt-20. Affects v1.0.0 tag timing.
