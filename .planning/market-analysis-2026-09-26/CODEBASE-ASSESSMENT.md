# RunnerKit codebase assessment

As of 2026-09-26. Subject: `accidentally-awesome-labs/runnerkit` at HEAD `64c3003` (2026-05-18), which is also the `v1.3.3` tag. The released v1.3.3 linux_amd64 binary reports `vcs.revision=64c30030…`, `vcs.modified=false`, go1.22.12, so the release and HEAD are the same code.

## How this was produced

**Sources.** This document draws on three sources:
- the eight adversarially verified code streams (raw JSON not committed), condensed in `evidence/code-condensed.md`;
- two gap-fill investigations in `evidence/gaps.md`: `released-byo-e2e-job` and `runner-version-enforcement`;
- a spot-check of the load-bearing `file:line` references against the working tree on 2026-09-26. The spot-checked references are marked † below. A separate fact-check pass, also on 2026-09-26, re-opened about 60 cited references (in `up.go`, `install.go`, `script.go`, `image_setup.go`, `sudoers.go`, `install.sh`, `upgrade_runner.go`, `recover.go`, `destroy.go`, `down.go`, `profile.go`, `provision.go`, `system.go`, `checks.go`, `doctor.go`, `logs.go` and the docs). It corrected a few off-by-one line ranges and some wording; no defect claim was refuted.

**How verifier output was used:**
- Severities are the verifier's corrected values.
- Claims the verifier marked "partially true" carry the verifier's caveat.
- The one refuted claim (product-history-traction-11, "seed sequencing ignored") is excluded.
- Issues the verifiers found and the analysts missed are included.

**What was not done.** No tests were run for this document: a previous run showed `go test` writes files into the checkout, and the repo must not be modified. The test and coverage figures come from the engineering-quality stream. Nothing new was fetched from the web.

**Evidence tags**, used throughout:

| Tag | Meaning |
|---|---|
| **[R]** | Reproduced: with the built or released binary, in a scratch copy, or in a container run of the released artifacts |
| **[C]** | Confirmed by code reading, and by the verifier |
| **[I]** | Inferred or plausible, not live-tested |
| **[S]** | External fact from a secondary source (for example Hetzner prices, which could not be read from hetzner.com) |

**Priority rubric used in §5:**
- **P0:** a documented core path is broken with no reasonable workaround, or the defect causes a billing leak or billing mis-consent, or data or credential loss.
- **P1:** a verified high-severity defect with a workaround, or a medium-severity defect on a core path or with a cost impact.
- **P2:** polish, low severity or tech debt.

Note: KEYFACTS §B lists silent CLI errors, the missing `Input` prompter and the unset `StateBaseDir` as P0 (B.6–B.8), and STRATEGY.md lists all three in the v1.3.4 harm-reduction release, with a rule that moves them to v1.3.5 if Stage 0 runs over budget (BACKLOG.md §0 applies that rule). Under this rubric they are **P1**: none breaks the core path outright or loses data or money, and each has a workaround (`--help` and re-typing for silent errors; `--yes` for typed prompts, which the error message itself suggests; running from a scratch directory for `StateBaseDir`). Only the `Input` case prints its workaround, and that workaround (`--yes`) also auto-accepts new host keys, so it weakens safety. The cli-ux verifier downgraded silent errors from critical to high because it "causes no data loss and no unsafe action". The priority label differs from KEYFACTS; the fix order in STRATEGY.md is unaffected.

**Effort legend:**

| Size | Duration |
|---|---|
| S | 1 day or less |
| M | 1–3 days |
| L | 1–2 weeks |
| XL | more than 2 weeks, one developer |

---

## 1. Summary scorecard

Maturity scale: **Solid**, **Partial** (works with material gaps), **Fragile** (works only on narrow inputs or fails silently), **Broken** (the documented path fails), **Missing**. Three rows that are not features use descriptive labels instead: **Weak** (security), **Stale** (toolchain) and **Drifted** (documentation).

| Subsystem | Maturity | Why (one line) |
|---|---|---|
| BYO bootstrap, password-sudo host (the documented golden path) | **Broken** | The released install.sh grants 16 fewer sudoers paths than the Go template bootstrap was written against (11 commands the host scripts call are refused). The v1.3.3 binary fails at `setup_runner_image` after 39.7 s [R]. |
| BYO bootstrap, host whose SSH user already has NOPASSWD:ALL (Ubuntu x64) | Fragile | Expected to come online [I]. No successful BYO run has been recorded since image setup landed in v1.3.1. Docker access and Rust are silently missing [R]. |
| Hetzner cloud provisioning and destroy (persistent, x64) | Partial | The most mature path: the lifecycle smoke was green on 2026-05-18. But the cost shown at consent is a constant, readiness stalls for 15 min on any cloud-init error, and `down`/`--replace` can orphan VMs. |
| Ephemeral mode (BYO and cloud) | Fragile | One job per `up`. The TTL flag is ignored on the host and the finalizer cannot write its outputs. There is no auto-destroy, and the mode was never live-smoked. |
| Day-2 ops (status, doctor, logs, recover, down, destroy, upgrade-runner) | Partial | Health model and dry-runs are well designed. But `upgrade-runner` destroys a working registration, `logs` queries the wrong unit, and the recover paths are miswired. |
| CLI UX (errors, prompts, wizard) | Fragile | Cobra errors are swallowed, so the CLI fails with no output. Typed confirmations cannot work in a TTY. The wizard runs nothing. |
| JSON / agent contract | Partial | `schema_version`/`stage`/`next_actions` exist, but there are two `next_actions` shapes, stripped error envelopes, and `ok:true` alongside ERROR findings. |
| Local state store | Solid | Atomic, versioned, backed up and secret-free. No locking, and keys are case-sensitive. |
| GitHub integration | Partial | Minimal repo-scoped REST client. No pagination, timeout or retry. Token precedence is inverted relative to the tool's own advice. |
| Security model | Weak | The "scoped" sudoers is root-equivalent. The runner user can plant code that later runs as root. The host-key pin is not bound to the SSH session. |
| Workload compatibility vs GitHub-hosted | Fragile | gcc and baseline builds work. Docker, `services:`, `container:`, `sudo` steps, arm64, parallel matrix legs and macOS all fail or are absent. |
| Multi-repo BYO (SEED-002) | Partial | Works mechanically, but all repos share one Unix user, `register` dead-ends after install.sh, and the shared tarball cache is defeated by a second download. |
| Tests, CI and release gates | Partial | 548 tests are green and race-clean, but none executes the generated shell. The release workflow runs no tests. There are no lint, vuln or shellcheck gates. |
| Toolchain and dependencies | Stale | Go 1.22 (out of support), x/net from 2023, hcloud-go v1 (frozen), Node 20 (EOL). |
| Distribution | Solid | GoReleaser, 4 platforms, cosign keyless-signed checksums, Homebrew cask. But there is **no LICENSE**. |
| Documentation | Drifted | The release notes point at a deleted command, and the docs overstate arm64/macOS support, image parity and ephemeral cleanup. |

**Bottom line.** The architecture and the operator-safety scaffolding are good: plan before mutation, verified destroy, secret-free state, error codes. The shipped behaviour is not. The primary BYO path is broken for its target audience (reproduced on the released artifact). The advertised "GitHub-hosted parity" fails for Docker and sudo workloads on every fresh host. Several day-2 commands damage or misreport the runner they are meant to fix. All of these survived a green CI, because no test executes the rendered bash or dispatches a real workflow job.

---

## 2. What exists: command surface and architecture

### 2.1 Command surface

`internal/cli/root.go:163-176`† registers 14 subcommands. Cobra adds `help` and `completion`. Bare `runnerkit` with empty state starts a first-run wizard. The persistent flags are `--json`, `--no-color`, `--explain` and `--unicode`.

| Command | Purpose | Notes at HEAD |
|---|---|---|
| `up` | Create a runner: BYO (`--host user@host`) or Hetzner (`--cloud hetzner`), persistent or ephemeral | Short help says "Set up a BYO … runner" although it is also the cloud entry point. 18 local flags, including three `--allow-*` overrides (`--allow-public-repo-risk`, `--allow-ephemeral-byo-risk`, `--allow-unknown-linux`). |
| `register` | Add a repo to an already-prepared BYO host | `runUp` plus a probe for the service user. Lacks `--extra-packages` (fails silently) and `--cloud`. |
| `init` | Print the `curl … install.sh \| sudo bash` host-prep one-liner | Prints only; by default the one-liner goes to stderr as a wrapped WARNING. |
| `list` | Saved runners grouped by SSH host | Help says "reads local state", but it probes GitHub and SSH serially for each repo. |
| `status` | Per-repo health: `ready`/`busy`/`needs_attention`/`broken`/`unknown` | Exits 0 whatever the health. |
| `logs` | Bounded journal, `_diag` tail and ephemeral archive | Queries an unresolved unit name for persistent runners (P1-9). |
| `doctor` | Findings, `--deep` OOM hints, `--fix`, `--ignore`, `--json` | `ok:true` and exit 0 even with ERROR findings. `--ignore` cannot persist (P1-3). |
| `recover` | Restart, reinstall the service, or re-register | Reinstall and re-register are miswired (P1-10). |
| `down` (alias `unregister`) | BYO cleanup | No cloud guard (P0-4). |
| `destroy` | Cloud teardown with provider verification | Refuses BYO. Can wedge on an unreachable host (P1-13). |
| `state show` | Print saved state | Always prints "Will not install a runner in Phase 1". |
| `upgrade` | Print upgrade instructions for the installed channel | Print-only by design. |
| `upgrade-runner` | Re-apply bootstrap with the bundled runner pin | Deletes credentials and reconfigures with an empty token (P0-3). |
| `version` | Print version | `--version` itself is not wired (no `root.Version`). |

The following are absent, although CONCERNS or the docs sometimes imply otherwise: `byo-prepare` (deleted in f017b2c, v1.0.8), `destroy --orphans`, `state forget`, org or group runners, JIT config, scale sets, and any monitoring or watch mode.

### 2.2 Package map

Production LOC was counted at HEAD† (15,399 total; tests 13,040).

| Package | LOC | Responsibility |
|---|---:|---|
| `cmd/runnerkit` (+ `cmd/_smokebin`) | ~400 | `main` wiring (`buildDependencies`). The smoke-only orphan lister lives in `_smokebin`. |
| `internal/cli` | 6,030 | Cobra commands **and** orchestration: `up.go` alone is 2,328 LOC, holding the composition root, prompts, policy and rendering. |
| `internal/provider` (+ `hetzner`) | 1,734 | 7-method `Provider` interface, registry and fake. Hetzner client, provisioning, cloud-init renderer, readiness, destroy. |
| `internal/ops` | 1,378 | Health classifier, doctor findings, cleanup and destroy plans, log collection, OOM heuristics. |
| `internal/bootstrap` | 1,104 | Go-rendered bash: `fix_dependencies`, `setup_runner_image`, runner install and service, ephemeral unit/finalizer/timer, sudoers templates, runner pin. |
| `internal/github` | 785 | Auth discovery (`gh auth token`, then env), REST client (5 calls), safety evaluation, remote parsing. |
| `internal/remote` | 642 | `Executor` interface, `SystemExecutor` (one `ssh` subprocess per call, script over stdin), logging decorator. |
| `internal/state` | 621 | JSON state store, schema v2, migrations, `ProjectConfig` types (no loader). |
| `internal/ui` | 592 | Renderer (human/JSON, redaction), prompter, boxes, checklists. |
| `internal/testsupport` | 382 | Fixtures (a non-test package). |
| `internal/preflight` | 343 | Remote host checks: OS, arch, systemd, sudo, disk, memory, tools, egress, NTP. |
| `internal/ux` | 244 | `stage`, `nextaction`, `checkliststore`. |
| `internal/runmode`, `update`, `redact`, `errcodes`, `labels`, `workflow`, `rklog` | 94–196 each | Mode/safety profiles, update notifier, redactor, RKD registry, labels and names, plan types, slog setup. |

The dependency graph is acyclic, with `redact` and `errcodes` as leaves [C]. Two layering faults stand out. First, the composition root is `cli.normalizeDependencies`, not `main`. Second, `state`, `labels` and `runmode` import `internal/github` for the `gh.Repo` domain type [C].

### 2.3 Data flow: `up` on BYO (`runUp`, `internal/cli/up.go`)†

1. Resolve the repo from `--repo` or the git remote. **GitHub metadata call** (`:109`). This happens before any local flag validation, so typos surface as auth errors and `--dry-run` needs credentials.
2. `resolveModeDecision` (`:115`), then `gh.EvaluateSafety` (`:119`): the public/fork gate. It runs before any side effect, which is a strength.
3. `resolveSetupPath` (`:124`), then `verifyTargetHostKey` (`:164`: ssh-keyscan, TOFU prompt, per-repo pin), then `preflight.Run` (`:169`). The Probe runs twice; each Probe is 17 serial ssh calls.
4. `autoDetectExtraPackages` (`:188`): regex-scans `.github/workflows` **in the current directory**.
5. `bootstrap.Plan` (`:194`) returns a static 6-step plan that omits `setup_runner_image`. `confirmBootstrapPlan` (`:229`) prompts without rendering it.
6. Mint the registration token (`:235`), then run `bootstrap.Apply` / `ApplyEphemeral` (`:246/:264`). Each step is a separate `ssh … bash -s` with inline `sudo`, in this order (`internal/bootstrap/install.go:113-127`†):
   `fix_dependencies` → `setup_runner_image` (Ubuntu/Debian/Mint only) → `create_runner_user` → `download_runner` (shared cache) → `configure_runner` (`config.sh --replace`) → `install_service` (`svc.sh`) → `verify_service`.
7. `waitForRunnerOnline` (`:287`), then `saveRepositoryState` (`:311`). The state-replace gate is checked **here**, after the host has been mutated (P1-21).

### 2.4 Data flow: `up --cloud hetzner` (`runCloudUp`, `up.go:659`)†

1. Auto-detect packages (`:681`), then `Provider.Validate` (`:693`). Live Hetzner lookups run **before** the dry-run branch, so the dry-run needs a token. Any failure is reported as "credentials missing".
2. Check state replace up front (`:722`, correctly before provisioning). Then the plan and cost with billing consent: `--yes`, or an interactive confirm. The cost comes from `internal/provider/profile.go:27-28`† and is a constant.
3. `Provision` (`:729`): SSH key (the user's own public key), a firewall (TCP 22 from `0.0.0.0/0` by default), and a server with cloud-init user-data. The cloud-init creates `runnerkit-admin` with `NOPASSWD:ALL` (`provision.go:385`†), stages the scoped sudoers through visudo, and installs about 70 baseline packages plus the auto-detected ones. A **pending checkpoint** with the resource IDs is saved immediately after Provision returns.
4. Readiness (`waitCloudTargetReady`, `:754`, `:922`):
   - provider WaitReady (150×2 s);
   - host-key probe (60×5 s, TOFU);
   - root-run `cloud-init status --wait`, retried for up to **15 min** on any non-zero exit (`:1027-1056`†);
   - preflight with `RequirePasswordlessSudo`.
5. Token, then the same `bootstrap.Apply` with `CloudProvisioned=true` (baseline packages skipped), then wait for online, then save state.
6. `destroy` removes things in dependency order: GitHub runner, remote runner, detach firewall, server (primary IPs cascade through `auto_delete`), SSH key, firewall. It then runs `VerifyDestroyed` (12 tries) and drops state only after verification passes.

### 2.5 State model

**Local state.** One file, `state.json`, under `$RUNNERKIT_STATE_DIR`, then `$XDG_STATE_HOME/runnerkit`, then `~/.local/state/runnerkit`. It uses `schema_version: "2"` (`internal/state/schema.go:10`). It holds `repositories[]` keyed by a **case-sensitive `full_name`** with no GitHub repo ID, so there is **one runner per repo**.

Each entry holds:
- repo (`gh.Repo`, persisted with PascalCase keys because the type has no json tags);
- auth *source* (never the value);
- runner name and labels;
- a machine ref (host, port, user, install/work paths, service name, host-key fingerprint);
- the provider ref (a Hetzner-shaped inventory: server, firewall and key IDs, primary IPs, cost profile);
- cleanup metadata and operation checkpoints;
- safety decision and overrides;
- ephemeral metadata (TTL, expires_at, archive path, finalizer status);
- `extra_packages`;
- `image_setup_version` and `runner_template_version`, which are **never populated by `up`** [C].

Writes are atomic (O_EXCL temp file, fsync, rename, dir fsync, 0600/0700). Keys that look like secrets are rejected. The raw file is backed up before a migration, and newer schemas are refused. There is **no lock**.

**Sidecar files.** `config.json` (doctor ignores) and `sessions/*.json` (BYO checklists) are meant to sit next to `state.json`. In the production binary they resolve against the **current directory** (P1-3).

**Host-side state:**

| Path | Contents | Owner |
|---|---|---|
| `/opt/actions-runner/<runner-name>/` | Install dir, including `svc.sh` | Runner user |
| `/opt/actions-runner/runnerkit-shared-bin/<ver>/` | Tarball cache | root |
| `/var/lib/runnerkit/` | Work dirs, `image-setup.json` marker, `cloud-init.json`, ephemeral state and log archives | Runner user |
| `/etc/sudoers.d/runnerkit-installer` | SSH-user allowlist | root |
| `/etc/sudoers.d/runnerkit-runner-ci` | Opt-in runner apt sudo | root |
| `/etc/systemd/system/…` | Units: stock `svc.sh` unit for persistent; RunnerKit-rendered unit, finalizer and TTL timer for ephemeral | root |

The runner-user ownership of the install dir and of `/var/lib/runnerkit` is a security issue (§6).

---

## 3. Feature maturity matrix

| Feature | Maturity | Evidence |
|---|---|---|
| Command tree plus dependency injection | Solid | `cli.Dependencies` injects GitHub, SSH, provider, prompts, clock and TTY (`root.go:21-113`). Gap: `main.go:23-45`† never sets `StateBaseDir`. |
| First-run wizard | Fragile | Executes nothing, says to paste a workstation command "on" the host, shows "Step 2 of 2" with no step 1, and exits silently on Enter (`wizard.go:66-95`, `ui/box.go:31-32`) [R] |
| `init` / install.sh host prep | **Broken** | `install.sh:33-46`† lacks tee, gpg, mkdir, unzip, usermod, dpkg, add-apt-repository, chmod, cp, cat and ln (16 paths), which `sudoers.go:70-91`† grants [R] |
| BYO `up` persistent, password-sudo host | **Broken** | Released binary: preflight PASS, fix_dependencies PASS, `setup_runner_image` FAIL at `sudo mkdir -p /etc/apt/keyrings` (`image_setup.go:43`†), exit 4 [R] |
| BYO `up` persistent, NOPASSWD host | Fragile | Runner should come online [I]. No docker group and no Rust (ordering bug) [R] |
| `register` (multi-repo) | Fragile | Dead end after install.sh: the service user is missing, and the remediation says to rerun install.sh (`up.go:1586-1597`, `installhint.go:60-83`) [C] |
| Preflight | Partial | Useful OS, arch, systemd, disk and memory checks, but: NTP check always warns, curl required before it can be installed, `runner.conflict` never populated, sudo probe tests only `install`, `no_sudo` branch unreachable [C/R] |
| SSH transport | Partial | One ssh process per call, script on stdin, `BatchMode`. No multiplexing, keepalive or command timeouts, and output is buffered [C] |
| Host-key TOFU pinning | Fragile | Separate keyscan, sessions run with `StrictHostKeyChecking=no` (`remote/system.go:214-220`†), non-standard fingerprint [R] |
| Runner download, shared cache, SHA-256 | Partial | Pinned 2.334.0 with per-arch SHA. The cache is verified only on first download, and the configure step downloads the 225 MB tarball again [R] |
| Service install and registration | Partial | Idempotent through `--replace` and svc stop/uninstall/install. Stock svc.sh unit with no restart or OOM drop-in [C] |
| Baseline apt packages (70) | Partial | Fixes "linker cc not found" on Ubuntu x64. Ubuntu names are sent to dnf/yum; Arch and openSUSE exit 20 [C] |
| Image parity (`setup_runner_image`) | Fragile | Installs Node 20 (EOL), Go "latest", Java 17, .NET 8, Docker CE, Chrome, Firefox and gh. Runs before the service user exists; amd64 hard-coded 6×; marker freezes partial failures [R/C] |
| Workflow apt auto-detection | Fragile | Unanchored regex: comment words become packages and `apt-get -y install` is missed. Scans the CWD, fails closed, has no opt-out (`workflow_packages.go:14-17`†) [R] |
| Hetzner provisioning | Partial | Consent, labels, checkpoints [C]. SSH key re-uploaded every time [I]. Deprecated arch-agnostic image lookup (`client.go:63`†) [C] |
| Cloud-init v3 | Partial | visudo-gated sudoers, baseline packages, version marker. Admin gets `NOPASSWD:ALL` anyway. User-data built with Sprintf [C] |
| Readiness gate | Partial | Multi-stage, but the "recoverable error" tolerance is dead code under `set -e`, and every failure retries for 15 min (`up.go:994-1011`†) [R] |
| Cloud plan and billing consent | Partial | Clear resource plan and future destroy command [C]. Cost is a constant for every type and region [R]. Dry-run needs a live token [C] |
| Cloud destroy with verification | Partial | Ordered, 404-tolerant, verifies before dropping state (live green) [R]. Can never converge when SSH is unreachable; partial destroy exits 0 [R] |
| Ownership labels on cloud resources | Solid | `runnerkit=true`, `managed=true`, repo, runner, state_id, mode, created_at [C] |
| Orphan discovery | Missing | Exists only as a smoke-harness binary (`cmd/_smokebin/empty_precheck`) [C] |
| Ephemeral mode | Fragile | One job per `up`. TTL hard-coded (`script.go:227`†). Finalizer runs unprivileged. No VM teardown. Never live-smoked [R/C] |
| Autoscaling, JIT, scale sets, pools | Missing | No webhook, polling, JIT or scaleset code; stated non-goal (`docs/safety.md:121-125`) [C] |
| `status` / `list` | Partial | Ordered multi-source classifier (`ops/status.go:186-251`). Exit 0 regardless. `list` does serial network I/O [R] |
| `doctor` | Partial | Rich findings with doc links. False positives when facts are missing. `ok:true` hard-coded (`cli/doctor.go:145`†) [R] |
| `doctor --deep` OOM hints | Fragile | Wrong unit for persistent runners, current boot only, linker regex scans a journal that never holds job output, warnings discarded [C] |
| `doctor --fix` / `--ignore` | Missing in practice | `--fix` handles only `runner_version_stale`, which `up` never arms. `--ignore` cannot write config [R] |
| `logs` | Fragile | Empty systemd section for persistent runners (`ops/logs.go:55`†). `--service` override exists [C] |
| `recover` | Fragile | Restart works. Reinstall and re-register are miswired (`recover.go:170,233`†) [C/I] |
| `down` / `unregister` | Partial (BYO); **Broken** for cloud | BYO plan is careful (path allowlist before `rm -rf`). On a cloud runner it drops state and orphans the VM [R] |
| `upgrade-runner` | **Broken** | Empty registration token after deleting credentials; binaries never replaced (`upgrade_runner.go:110-122,150`†) [R] |
| `upgrade` plus update notifier | Solid | Print-only. 24 h cache, ETag, silent in CI or JSON mode [C] |
| State store | Solid | See §2.5. No lock [R: 20 concurrent saves leave 1 entry] |
| JSON contract | Partial | Heterogeneous shapes. The smoke contract checks only non-null arrays and is never run in CI [R] |
| RKD error codes | Partial | 51 codes, each with a docs anchor. 19 never emitted. Findings have no `code` field [C] |
| Redaction | Partial | Centralized in the renderer, but split across ≥5 `redact.New()` instances. Runner-token regex matches only fixtures. `RedactArgs` never read [C] |
| Public/fork safety gate | Partial | Runs before side effects [C]. Evaluated only at `up`/`register`. Private forks misclassified [R] |
| GitHub auth and API client | Partial | gh token first, env fallback. No pagination, timeout or retry [C] |
| `.runnerkit/config.yaml` | Missing | Types only, no YAML library, callers pass `nil`. The docs tell users to use it [C] |
| Resumable checklist sessions | Missing in practice | Write-only (`checkliststore.Load` has no caller), and written to the CWD [C] |
| `--explain` / `--unicode` / `--no-color` | Missing in practice | `--no-color` is vestigial (no ANSI output), `--explain` is read at 2 sites [C] |
| Monitoring and alerting | Missing | No timer, agent, webhook, metrics or jobs-API use [C] |
| arm64 | **Broken** | Preflight accepts it; image setup is amd64-only [C] |
| Non-Ubuntu Linux | **Broken** | 11 distros pass preflight; only Ubuntu is provisioned correctly [C] |
| macOS / Windows / GPU / Android | Missing | macOS is wrongly listed as supported in `docs/runner-platforms.md:12` [C] |
| Release and distribution | Solid (signing) / Partial (gates) | cosign over checksums including install.sh. No test gate in `release.yml` [C] |

---

## 4. Genuine strengths (keep these)

Each item carries the verifier's caveat where one applies.

1. **Plan-before-mutation and billing consent.** Every mutating command has `--dry-run`. The cloud plan lists each resource and marks what is billable, with names, tags, CIDR, the "not created" list and the future destroy command. Billable provisioning requires `--yes` or an interactive confirm, and refuses under JSON or non-TTY (`up.go:1175-1197`). *Caveats:* the cost figure itself is wrong (P0-6), and the BYO plan omits the heaviest step (P1-14).
2. **Cloud cleanup discipline.** A pending checkpoint with resource IDs is saved right after `Provision` returns, and `ProvisionError` carries partially created IDs. Destroy deletes in dependency order, tolerates 404s, and removes local state only after provider verification. There is a regression test for the `AutoDelete` primary-IP cascade (`provision_test.go:203`). The live Hetzner destroy of a 4-day-old VM removed 8 artifacts.
3. **Secret-free, durable local state.** Writes are atomic with fsync. Files are 0600 in 0700 directories. Secret-named keys are denied. The raw file is backed up before migration. A newer schema is refused with RKD-STATE-004, exit 7.
4. **Durable credentials never leave the workstation.** Only short-lived registration and removal tokens reach hosts, and they travel over SSH stdin. The GitHub token is not in cloud-init. The Hetzner token is read from env and never persisted. *Caveat:* on the host the tokens leak into argv and sudo logs (SEC-6).
5. **Safety gate before side effects.** A blocked public repo mints no token, runs no probe and saves no state; this is regression-tested (`up_test.go:254-267`). Persistent mode on public repos needs a typed acknowledgement. Every completion prints the warning against using `runs-on: self-hosted` alone.
6. **Testable architecture.** Every external effect goes through injected interfaces (`Executor`, `Provider`, GitHub service, prompts, clock, sleep, TTY), and the package graph is acyclic. This makes the proposed host-e2e test harness cheap to add.
7. **Operator-grade diagnostics design.** The health classifier is ordered and multi-source, and is reused by status, doctor, list, recover and stage. There are stable RKD codes with docs anchors (tested) and a docs-base override. Before `rm -rf`, the cleanup path checks an allowlist (`SafeRunnerPaths` refuses `/`, `/opt`, `/var/lib/runnerkit` and unrecorded paths). Runner deletion is conservative: it needs an ID match, or both labels, and ambiguity is refused. *Caveat:* the exit-code taxonomy is weaker than advertised, because `ExitSafetyGate` (4) is reused for infrastructure failures.
8. **Runner self-update left enabled.** `--disableupdate` is never passed (grep of the Go code† finds none). This is load-bearing. Under GitHub's 30-day update rule, auto-updating runners stay routable, so the 2.334.0 pin and the 4-month release gap do **not** by themselves break new installs [I, medium confidence, `gaps.md`]. It should be recorded as a documented invariant.
9. **Supply-chain basics.** Runner tarballs are pinned by SHA-256 per arch. The GoReleaser version is pinned. cosign keyless signatures cover the checksums, including `install.sh`, and the verify identity is documented. Upgrades are print-only instead of self-replacing, and the update notifier is polite: cached, ETag, silent in CI.
10. **Shell hygiene at lint level.** shellcheck reports zero findings on `install.sh` and the smoke scripts. The Go-rendered host scripts yield one SC1091 info note. The defects are behavioural, not lint-level.
11. **The hosted-image-parity idea.** The baseline list tracks actions/runner-images and fixes the classic DIY-runner failure. The per-tool `command -v` guards make re-runs cheap once the ordering and marker issues are fixed.
12. **Institutional memory.** Bug-numbered rationale comments and regression tests are named after bugs, and the 2026-05-18 live smoke published honest red results. *Caveat:* the planning corpus drifted from the code, and that drift is what sent the v1.3.3 fix to the wrong allowlist (§8.5).

---

## 5. Verified defects

Each entry cross-references its dossier stream ID in parentheses. Security-specific defects are consolidated in §6 as SEC-n.

### P0: broken core paths, billing leaks or mis-consent, data or credential loss

**P0-1: The documented BYO host prep cannot bootstrap a password-sudo host.**
(byo-bootstrap-1, engineering-quality-1, product-history-traction-3, workload-compatibility-6, M1; critical; [R] on the released artifact)
- **Evidence:**
  - `install.sh:33-46`† grants apt-get/dnf/yum, useradd, install, curl, sha256sum, chown, rm, su, tar, systemctl and svc.sh.
  - `internal/bootstrap/sudoers.go:70-91`† additionally grants tee, gpg, mkdir, unzip, usermod, dpkg, add-apt-repository, chmod, cp, cat and ln (16 paths).
  - The only production caller of `RenderSudoersEntry` is cloud-init (`provision.go:360`), where the admin already has `NOPASSWD:ALL` (`provision.go:385`†). So both the v1.3.1 and v1.3.3 allowlist fixes only ever reached cloud.
  - `install_sh_test.go:24-38` compares only the header line.
  - The preflight probe `sudo -n install --version` (`checks.go:185`) passes on an install.sh host, and doctor's `byo_host_prepared` is only `test -f` (`cli/doctor.go:181`†). Both give false greens.
  - `RELEASE-NOTES-v1.3.3.md:7,12,26` tell users to rerun `runnerkit byo-prepare`, which was deleted in f017b2c and exits 1 silently.
- **Reproduction** (`gaps.md`): released v1.3.3 binary with the released install.sh on Ubuntu 24.04. `setup_runner_image` fails at `sudo mkdir -p /etc/apt/keyrings` (`image_setup.go:43`†) with "a terminal is required", exit 4 after 39.7 s, failing step reported as "(unknown)". 11 commands and 16 paths are refused.
- **Ephemeral BYO:** on install.sh-prepared hosts it fails on every distro, not only Ubuntu, because its unit, finalizer and timer need `sudo tee` and `sudo chmod`.
- **Scope:** hosts whose SSH user already has NOPASSWD:ALL are unaffected, but password-sudo hosts are the documented golden path.
- **User impact:** a new user following README → `init` → install.sh → `up` never gets a runner. The documented fix command does not exist, and rerunning install.sh reinstalls the same stale list.
- **Fix:** generate install.sh's fragment from `RenderSudoersEntry` (`go generate` or embed), add a full-body equality test, and add a lint that every `sudo <cmd>` in the rendered scripts is allowlisted. Wire `SudoersIsPrepared` into doctor and preflight. Correct the release notes. Strategically, have install.sh perform the privileged user and image setup as root once, or replace the allowlist with a root-owned helper (SEC-1).
- **Effort:** S for sync, tests and notes; L for the helper redesign.

**P0-2: `setup_runner_image` runs before `create_runner_user`, so the runner never gets Docker access or Rust, and never self-heals.**
(workload-compatibility-1 critical; byo-bootstrap-3, engineering-quality-2, cloud missed-1 high; [R])
- **Evidence:**
  - `install.go:119→123` (Apply) and `:170→174` (ApplyEphemeral)† run the steps in that order.
  - `image_setup.go:101`† `sudo usermod -aG docker <svc> 2>/dev/null || true` and `:71` (rustup via `su - <svc> … || true`) both fail silently for a user that does not exist yet.
  - The marker written at `:167`† gates every later run (`:30-35`†).
  - Cloud-init creates only the admin user, so cloud is affected too.
  - The ordering tests never set `OSReleaseID`, so this path is untested.
- **Reproduction** (`gaps.md`): `id runnerkit-runner` shows `groups=runnerkit-runner` only. `docker run hello-world` as the runner user gives "permission denied … /var/run/docker.sock". Adding the group after the user exists fixes it.
- **User impact:** every `docker build`/`run`, `services:`, `container:` and Docker-based action job fails on every fresh BYO and cloud host, although Docker is advertised. Rust is missing. Rerunning `up` or `upgrade-runner` does not repair it.
- **Fix:** create the user first. Make the usermod and Rust steps fatal, or verify `id -nG` plus `docker info` and `cargo -V` as the runner user in `verify_service`. Restart the service after the group change. Bump `ImageSetupVersion` so existing hosts rerun. **Decide SEC-5 at the same time**, because making the group work makes every job root-equivalent.
- **Effort:** S.

**P0-3: `upgrade-runner` destroys a working registration and never upgrades binaries.**
(github-state-security-1, ops-diagnostics-1, workload-compatibility-4, WC-M1, M3; high; [R] for the token, [I] for config.sh's exact failure)
- **Evidence:**
  - `upgrade_runner.go:110-122`† builds `bootstrap.Options` with no `RunnerToken` and never calls GitHub.
  - `configure_runner` runs `sudo rm -f .runner .credentials .credentials_rsaparams` and then `config.sh --token ""` (`script.go:55-56`†). A scratch run logged `RUNNERKIT_REGISTRATION_TOKEN=""`.
  - Both extractions use `tar --skip-old-files` (`install.go:241`, `script.go:52`†), so `bin/` is never replaced.
  - The service is not stopped first and busy runners are not gated.
  - State records the bundled pin anyway (`:150`†).
  - `docs/upgrade.md:47-48` calls the command "idempotent — safe to re-run".
  - `doctor --fix` invokes it with `--yes`, but that trigger (`runner_version_stale`) essentially never fires, because `up` never writes `RunnerTemplateVersion` (the only writer is `upgrade_runner.go:150`†).
- **User impact:** running the documented command on a healthy persistent runner deletes its credentials. The runner goes offline at its next restart, and the user must find `recover --reregister`, which is itself miswired (P1-10) and so most likely fails too.
- **Fix:** short term, mint a token, or skip reconfigure when `.runner` exists. Real fix: stop the service, extract into a versioned directory and swap, keep `.runner`/`.credentials`/`_work`, restart, read the real version back, and gate on busy. Pull `doctor --fix` until this is fixed.
- **Effort:** S (token); M (real upgrade).

**P0-4: `down` / `unregister` on a cloud runner deletes the only record of billable resources.**
(cloud-ephemeral-1, cli-ux-4; high after the verifier lowered it from critical; [R])
- **Evidence:**
  - `internal/cli/down.go` contains no provider or cloud check (grep† finds zero matches); `destroy.go:77-81`† has the reverse guard.
  - `ops/cleanup.go:39-53` builds only BYO artifacts.
  - A scratch run of `down --repo owner/repo --yes --json` on seeded Hetzner state returned `ok:true, state_removed:true`, removed `srv-123` from state, and made no provider call.
  - Doctor suggests `down --dry-run` for `cleanup_pending` (`ops/doctor.go:142`), which also fires on pending cloud states.
- **User impact:** "unregister" is a natural verb for someone finished with a runner. Using it leaves a VM billing indefinitely, and `destroy` then says nothing is saved. Recovery is manual, in the Hetzner console, via the `runnerkit=true` labels.
- **Fix:** refuse cloud state in `down` with `wrong_cleanup_command` pointing to `destroy`, or make `down` provider-aware. Point doctor and status `next_actions` at `destroy` for cloud. Add `destroy --orphans` using the existing labels.
- **Effort:** S (guard); M (orphan sweep).

**P0-5: `up --cloud --replace` overwrites cloud state without destroying the previous VM.**
(cloud missed-3; high; [C])
- **Evidence:** `up.go:722-726` and `:1199-1210` return "replace" with no provider check. `:733-746` saves the new state. Ephemeral names are random (`:684-688`), so no name collision stops a second VM. The prompt mentions only "overwrite the existing RunnerKit state".
- **User impact:** this is the natural route to "one more ephemeral job" (see P1-12). The first VM keeps billing, and the CLI can no longer see it. Only Hetzner's key-fingerprint uniqueness (P1-8) may block it, and `--ssh-key` bypasses that.
- **Fix:** for cloud state, refuse `--replace` until `destroy` succeeds, or run destroy as part of the replace. At minimum, list the old IDs in the prompt and keep them in a "superseded" inventory for GC.
- **Effort:** S.

**P0-6: Billing consent shows a hard-coded cost for every server type and region.**
(cloud-ephemeral-7; high; [R] for the constant, [S] for current prices)
- **Evidence:** `internal/provider/profile.go:27-28`† hard-codes "approx €0.0081/hour" and "approx €4.90/month", and these are applied to every type and region (`:113-118`). A scratch plan for `ccx63` in `sin` printed the same figures. The two numbers disagree with each other (0.0081 × 730 ≈ €5.91), and IPv4 is excluded. Live per-location pricing (`ServerType.Pricings`) is already fetched and discarded.
- **Secondary-source context [S]:** the default `cpx22` reportedly costs €19.49/month net (€19.99 with IPv4) for new orders since 2026-06-15, about 4× the displayed figure. The default location `fsn1` reportedly had no orderable types in a 2026-08-25 probe.
- **User impact:** the billing-consent screen, which is central to the "safe billing" positioning, understates cost by about 4× for the default and by orders of magnitude for larger types. The default may not be orderable at all.
- **Fix:** compute hourly and monthly cost from `serverType.Pricings` for the chosen location plus primary IPv4, and show the source and fetch time. Check `ServerType.Locations[].Available`, which needs hcloud-go v2. Reconsider the default type and location against live data.
- **Effort:** S for v1 pricing; M together with the v2 migration.

### P1: high-severity or core-path defects with workarounds

| ID | Defect (stream refs, verified severity, tag) | Evidence | User impact | Fix | Effort |
|---|---|---|---|---|---|
| P1-1 | Cobra errors are swallowed: unknown commands, bad flags, bad values and `--version` print **nothing** and exit 1 or 2 (cli-ux-1 high, missed-5; [R]) | `root.go:126-127`† SilenceErrors/SilenceUsage; `main.go:42-45`† only calls `os.Exit`; no `root.Version` | Typos and the v1.3.3 upgrade note (`byo-prepare`) look like crashes; support cannot triage | Print any error not already rendered (keep Cobra suggestions); set `root.Version`; binary test for non-empty stderr | S |
| P1-2 | The production prompter has no `Input`, so typed confirmations and the interactive BYO host prompt fail in a TTY (cli-ux-2 high, cli-ux-m3; [R]) | `ui/cli_prompter.go` implements Confirm:44, Select:70, Password:106 only†; call sites `up.go:1447,2088,2112,2188`, `destroy.go:124` | Interactive `destroy`, public-repo ack, ephemeral ack and state replace exit 6. Guided `up --repo X` asks BYO vs cloud twice, then dead-ends. Users learn `--yes`, which also auto-accepts new host keys (`up.go:1482-1485`) | Implement `Input`; test the production prompter against every optional interface; PTY smoke for destroy | S |
| P1-3 | `StateBaseDir` is never set, so `config.json` and `sessions/` resolve against the CWD (cli-ux-3 and ops-diagnostics-3 high; engineering-quality-3 medium; [R]) | `main.go:23-40`†; `userconfig.go:16-40`; `checkliststore/store.go:31-32` (only `rkstate.NewStore` falls back) | `doctor --ignore` always fails (exit 5). A non-object `./config.json` breaks doctor. An object-shaped one with `doctor_ignore_finding_ids` silently hides ERROR findings. `up --host` writes `sessions/*.json` (containing user@host) into the user's repo checkout | Default to `state.DefaultBaseDir()` in `normalizeDependencies`; test the production wiring | S |
| P1-4 | Cloud-init readiness: the "recoverable error" tolerance is dead code under `set -e`, every failure (including deterministic `status: error`) retries until the 15-min deadline, and there is no rollback or resume (engineering-quality-4 high, eq-missed-5 medium, cloud-ephemeral-12 and -13 medium; [R] with a stub) | `up.go:994-1011`† (`cloud-init status --wait; rc=$?` under `set -euo pipefail`); `:1036-1049`† retries any error; `:1056`† 15 min; budgets add up to about 25 min | One bad package (for example a misdetected one, P1-5) stalls about 15 min on a billed VM, then fails. The user must `destroy` and re-provision from scratch | `rc=0; … \|\| rc=$?`; retry only transport errors (ssh 255); fail fast on exit codes 1/2 with `status --long`; `--destroy-on-failure` plus resume from pending | S / M |
| P1-5 | Workflow apt auto-detection is on by default with no opt-out, mis-parses, scans the CWD rather than `--repo`, and fails closed (engineering-quality-5 high, workload-compatibility-8, WC-M3; [R]) | `workflow_packages.go:14-17`† unanchored regex; `-\S+\s+` eats `-t` but not its argument; `:75-86` tokenizer; `up.go:2291-2297` (CWD; printed only without `--json`); one `apt-get install` (`script.go:19`) or cloud-init `packages:` (`provision.go:374-378`) | `# needed for the build step` becomes packages "needed", "for", "the"; `-t bookworm-backports` becomes a package; `apt-get -y install x` is missed. BYO `up` fails with "Unable to locate package"; cloud fails at readiness after billing starts | YAML plus shell-aware parse restricted to jobs targeting RunnerKit labels; print and confirm the list in all modes; install extras per package, non-fatal, after readiness. Short term, turn detection off by default behind an opt-in flag (as STRATEGY.md's de-emphasis list proposes) | M |
| P1-6 | arm64 is advertised but broken (workload-compatibility-3 high; engineering-quality-11 medium; cloud-ephemeral-8 partially true; [C]) | `image_setup.go:60,97,107,119,141,153`† hard-code amd64; `hetzner/client.go:63`† deprecated arch-agnostic `Image.GetByName`; `docs/runner-platforms.md:11` | BYO arm64 Ubuntu (Pi, Ampere, CAX) aborts at `docker-ce` after installing an amd64 Go. Cloud CAX likely fails at `CreateServer` [I] | `dpkg --print-architecture`-driven script; `GetByNameAndArchitecture`; arm64 CI job; or withdraw the claim | M |
| P1-7 | Non-Ubuntu distros pass preflight, then fail bootstrap (byo-bootstrap-4 high; workload-compatibility-7, eq-missed-1/2 medium; WC-M5 low; [C]) | `checks.go:313-321`† accepts fedora…opensuse; `install.go:313-329` always adds the Ubuntu-named baseline; `script.go:20-26` (dnf/yum get apt names; others exit 20); Debian/Mint get the Ubuntu Docker repo and codename (`image_setup.go:97`); `ID_LIKE` ignored | Fedora/RHEL/Debian/Mint home servers fail minutes in. Pop!_OS and similar are rejected as unknown | Narrow preflight to a tested matrix with a clear message, or per-family package maps; container bootstrap matrix | M |
| P1-8 | Every provision uploads the user's own public key as a new Hetzner key; no reuse path (cloud-ephemeral-6 high; engineering-quality-15 partially true; [I]: plausible, not live-tested) | `provision.go:126`† `CreateSSHKey`; `up.go:880-910`; `GetByFingerprint` unused | Likely `uniqueness_error` when the key is already in the project or when a second repo's cloud runner is created | Reuse by fingerprint (mark not-owned) or a per-runner generated key; map the error to an RKD code | S–M |
| P1-9 | `logs` and the doctor OOM heuristics query an unresolved unit name; cloud admin has no journal access; evidence sources are wrong (ops-diagnostics-2 high, partially true; ops missed-2 medium; [R]) | `ops/logs.go:55,152`† use `Machine.ServiceName` (`actions.runner.<runner>.service`); the real unit `actions.runner.<owner-repo>.<runner>.service` is resolved only in `probes.go:89-150`; `journalctl -k` is current boot only; `cli/doctor.go:160` discards warnings | Persistent runners: empty journal section with no warning, exactly when a crash loop is the question. The OOM hint feature effectively never fires on cloud. Ephemeral is unaffected; `--service` override exists | Resolve and persist the real unit; journal group or sudo for the admin; `-b -1`; surface collection warnings; get linker kills from job logs via the API | S |
| P1-10 | `recover` reinstall and re-register are miswired (M2 high, [I] from svc.sh semantics; ops-diagnostics-13 medium, [C]) | `recover.go:200-205` stops and uninstalls the service (`:204`†), then `:233`† only runs `svc.sh start`; `:170`† runs `svc.sh install` without uninstalling (fails when the unit exists) and verifies a stale pre-resolved name (`:147,:174`) | The natural fix for P0-3 or stale credentials most likely leaves the runner registered but serviceless. The documented RKD-BOOT-003 remedy fails | Reuse `RenderServiceScript` (stop, uninstall, install, start); re-resolve the unit afterwards; command-sequence test | S |
| P1-11 | The job user gets no sudo on cloud, and only opt-in package-manager sudo on BYO (workload-compatibility-2 high; [R]) | cloud-init `provision.go:379-404` has no runner CI sudoers; `install.sh:65-93` grants it only with `RUNNERKIT_GRANT_CI_SUDO=1`; the default one-liner omits it | Workflows green on `ubuntu-latest` fail at their first `sudo apt-get`, `tee`, `systemctl`, disk cleanup or `playwright --with-deps` | `--ci-sudo none\|packages\|full`; full by default on single-tenant cloud VMs, documented as root-equivalent | S–M |
| P1-12 | Ephemeral mode: TTL flag ignored on the host, finalizer cannot write, cloud VM never auto-destroyed, cloud logs "preserved" onto the VM being deleted, one job per `up`, never live-smoked (cloud-ephemeral-2/3/4/10 medium, -5 high; ops-diagnostics-11; product-history-traction-15; [R]/[C]) | `script.go:227`† `OnActiveSec=24h`; `up.go:1910-1914` persists `"24h"` while `ExpiresAt` uses the flag; `script.go:182-187`† `User=` with `ExecStopPost` and no `+`, into root-owned dirs (simulation: exit 1); `destroy.go:165-179` then `:218-226`; `KillMode=process` | `--ephemeral-ttl 2h`: status says "expired" while the runner still takes jobs for 24 h. No archive and no "completed" sentinel, so status recommends recovery. Cloud VMs bill until a manual `destroy`; the completion output says so, but the docs say "cleaned up automatically" | Render the TTL; `ExecStopPost=+…`; disarm the timer on completion; fetch the archive before destroy; fix the docs; longer term, a JIT or scale-set refill loop (XL) | S / XL |
| P1-13 | Cleanup cannot converge, and exit codes hide partial cleanup (ops-diagnostics-4/5, cloud missed-2/-5 medium; ops missed-3 low; [R]) | `destroy.go:199-205,253-267` keeps state whenever SSH is unreachable, even after verified provider deletion; partial destroy exits 0 (`:92-99`); `down.go:343-352,376-382` drops state when file removal failed; `down.go:441-447` ends in `\|\| true`; no `state forget` (`state.go` has `show` only); `docs/troubleshooting/cleanup.md:111-121` recommends `rm -rf /opt/actions-runner/runnerkit-* /var/lib/runnerkit` | Dead-host entries can never be cleared except by deleting `state.json`, which drops every repo. Scripts see exit 0. Following the docs on a multi-repo host wipes every repo's runner and the shared cache | Treat provider-confirmed deletion as satisfying remote cleanup; `state forget --repo`; dedicated partial-cleanup exit code; per-runner doc steps | S–M |
| P1-14 | The BYO install plan is not disclosed, and the heavy image install is forced (cli-ux-13, cli-ux-m2, eq-missed-3 medium; byo-bootstrap-11 partially true; [R]) | `bootstrap.Plan` ignores its options (`install.go:100`), so the static 6-step plan (`workflow/plan.go:79-92`) omits `setup_runner_image`; `confirmBootstrapPlan` (`up.go:1617-1636`) shows nothing; no image flag; preflight requires 2 GiB (`checks.go:48`†) vs a measured 3.16 GiB and a projected 4.5–5 GB with about 1.4 GB of downloads (`gaps.md`) | Users approve "Apply BYO runner install plan?" and get Docker CE, Chrome, Firefox, JDK, .NET, Node 20 (EOL), Go "latest" and 5+ third-party apt sources on a personal machine. Small disks fill. `docs/byo-quickstart.md:75` does disclose this; the CLI does not | Derive the plan from Apply's step list including packages; `--image minimal\|standard\|full` (minimal default for BYO); disk check per profile | M |
| P1-15 | Bootstrap failures lose the failing step (byo-bootstrap-7 medium; [R]) | `install.go:133-140` returns the raw `*exec.ExitError`, so `up.go:2314-2327` prints "(unknown)"; stderr starts with ssh "Permanently added" noise | Every bootstrap failure needs manual SSH forensics; this is how Bug A was investigated | Always wrap as `RemoteError{CommandID, ExitCode}`; `trap 'echo RKFAIL:$BASH_COMMAND' ERR`; `-o LogLevel=ERROR` | S |
| P1-16 | Image-setup marker freezes partial failures, and network lookups abort the whole script (WC-M4 medium; byo-bootstrap-9c/d; `gaps.md` [R]) | Marker gate `image_setup.go:30-35`, written unconditionally at `:167`; gpg key fetches are `\|\| true` while `.list` files are written anyway; Go and geckodriver version lookups under `set -e` abort (rc=1 reproduced) | A transient failure permanently leaves a tool missing, or leaves a broken apt source that breaks every later `apt-get update` | Per-section markers written only after verification; per-tool status in doctor; skip and remove the `.list` on key failure | M |
| P1-17 | Runner tarball cache is verified only on first download; configure downloads it again (engineering-quality-8 partially true medium; WC-M2, M4; [R]) | `install.go:235-241` (verify only inside `if [ ! -f ]`; curl writes to the final path); `script.go:48-52`† second 225 MB download per install dir | An interrupted download causes a sticky `tar` failure on every later run, across all repos on a multi-repo host. The shared cache is defeated | Temp file, verify, `mv`; verify before every extract; remove the duplicate download | S |
| P1-18 | `register` is a dead end after the documented install (byo-bootstrap-6 medium; [C]) | install.sh creates no service user; `up.go:1586-1597` requires one; `installhint.go:60-83` says to rerun install.sh | Users and agents following `next_actions` loop | install.sh creates the user and `/var/lib/runnerkit` layout, or register falls back to `create_runner_user` | S |
| P1-19 | Health exit codes and the JSON contract are unreliable for automation (cli-ux-5/6, ops-diagnostics-8 medium; [R]) | `cli/doctor.go:145`† hard-codes `"ok": true`; status and doctor exit 0 on broken or unknown; two `next_actions` shapes (even inside one `status --json` payload); `status --all --json` lacks `schema_version`; error envelopes stripped; `redactions_applied` constant | `runnerkit doctor \|\| notify` cannot work, and agents must special-case each command, undercutting the SEED-003 agent story | One envelope helper and a published JSON Schema; exit codes 0/degraded/broken/partial with `--exit-zero`; contract test over success and error paths | M |
| P1-20 | Doctor misdiagnoses: false positives when facts are missing, blind to stale sudoers (ops-diagnostics-6 medium; product-history missed-3 high; cli-ux-9 medium; cli-ux-m4 low; [R]) | `ops/doctor.go:71-75,90-92,105-112,175`; stage returns "uninstalled" whenever the install probe fails (`ux/stage/stage.go:46-52`); `byo_host_prepared` is `test -f` (`cli/doctor.go:181`†) while `SudoersIsPrepared` is unused | With no auth or no ssh binary, doctor says the install is missing and suggests re-registering. On the product's most common failure (P0-1), it reports the host install PASS | Explicit collection findings that suppress dependent checks; tri-state probes; content-level sudoers check | M |
| P1-21 | BYO rerun mutates the host before the state-replace gate (cli-ux-12 medium; [C]) | Token, Apply and online wait (`up.go:235-296`) happen before `saveRepositoryState`/`confirmStateReplace` (`:1707-1717`); cloud checks first (`:722`) | "Just rerun `up`" re-registers and restarts the runner, then exits 6 with stale state | Move the gate before any remote mutation; same-host reruns become idempotent updates | S |
| P1-22 | One runner per repo; runner identity is per repo, not per host (github-state-security-12 medium; workload-compatibility-5 medium; [R]) | `labels.go:73-77` name is the label plus `-local`; `--replace` (`script.go:56`); no `--runners`/`--labels` flags (`ExtraLabels` unused); state keyed by repo | Matrix legs serialize (a 6-leg matrix takes about 6× as long). A second machine silently takes over the first's registration. Relocating fails with a misleading "fingerprint changed" | Host discriminator in names; N runners per repo; `--labels`; check `HostRef` before the fingerprint | M |
| P1-23 | Mutating day-2 commands ignore `busy` (ops-diagnostics-14 medium; [C]) | `recover.go:151-160,201`; upgrade-runner gates only ephemeral runners (`upgrade_runner.go:64-78`) | `recover` or `upgrade-runner` during CI silently kills an in-flight job | Refuse or `--drain` when GitHub reports busy, unless `--force` | S |
| P1-24 | apt robustness: no lock timeout, `DEBIAN_FRONTEND` dropped by sudo, no command timeouts, buffered output (byo missed-4 medium; byo-bootstrap-8 partially true; byo-bootstrap-14, workload-compatibility-14 low; [C]) | `script.go:15-19`; `image_setup.go:38` export lost to `env_reset`; no `Timeout` on bootstrap commands (`install.go:113-127`); `system.go:61-91` buffered | Immediate "Could not get lock" on freshly booted Ubuntu (unattended-upgrades). Minutes of silence. Rare debconf prompts can consume the piped script | `-o DPkg::Lock::Timeout=600`; `sudo DEBIAN_FRONTEND=… apt-get`; stdin from `/dev/null`; per-step timeouts and streaming | S |
| P1-25 | No state lock; no signal handling (github-state-security-7 medium; engineering-quality-19 low; cloud-ephemeral-9 partially true medium; [R]) | `state/store.go:133-172`; grep of the Go code† finds no `signal.Notify` and no `flock`; 11 `context.Background()` calls in `internal/cli` | Concurrent `up` or agent-driven commands can drop an entry (millisecond window; 20 concurrent saves kept 1). Ctrl-C mid-bootstrap leaves no guidance. The cloud orphan window is small because of the checkpoint | `flock` on `state.json.lock`; `signal.NotifyContext` plus a cleanup or resume hint | S |

### P2: polish, low severity, tech debt

| ID | Defect (refs, severity) | Evidence | Fix | Effort |
|---|---|---|---|---|
| P2-1 | First-run wizard misdirects and runs nothing (cli-ux-7 medium) | `wizard.go:66-95`; "paste this on <host>"; "Step 2 of 2"; Enter on "(recommended default)" exits 2 silently | Make the wizard call `runUp` in-process; re-prompt; honor defaults; URLs rather than repo paths | M |
| P2-2 | Verb and flag model inconsistent (cli-ux-14 medium) | `up`/`register` near-duplicates; `down` vs `destroy` split by provider; `logs --runner` vs `down --runner-name`; `--yes` overloaded | One create verb, one provider-aware teardown verb; separate `--accept-host-key` | M |
| P2-3 | Planning jargon and false copy in user output (cli-ux-15 medium; cli-ux-m5 low) | "Phase 4" (`up.go:87,631,662`), "SEED-002" (`list.go:25`), "future runnerkit down flow" (`up.go:1788`), "Will not install a runner in Phase 1" (`state.go:82`), README D-01 and fixture sentence | String sweep; move doc-test anchors into fixtures | S |
| P2-4 | `init` one-liner printed as a wrapped WARNING on stderr; fixed width 80 (cli-ux-16 medium) | `init.go:98-99` uses `renderer.Warning`; `main.go:33`† `Width: 80` | Commands unwrapped on stdout; detect terminal width | S |
| P2-5 | `status`/`list` empty states contradictory; `list` does hidden serial network I/O (cli-ux-17 medium; ops-diagnostics-16 low) | `status.go:71,96-104`; `list.go:82` `collectStatus(…, true)` serially | Single empty-state message; `list` local by default, `--probe` parallel | S |
| P2-6 | Preflight correctness gaps (byo-bootstrap-13 medium; byo missed-3 low) | NTP greps "true" but `timedatectl` prints yes/no (`system.go:47`); curl required before it can be installed; `RunnerConflict` never set; `no_sudo` unreachable | Fix the checks; `sudo -n -l` per required command | S |
| P2-7 | Global flags inert; progress UX poor; sessions write-only (cli-ux-18/19 low) | No ANSI anywhere; `Explain()` read at 2 sites; checklist reprinted 5×; `checkliststore.Load` has no caller | Remove or implement; one line per transition with durations | S |
| P2-8 | `doctor --ignore`/`--fix` design thin; version-stale finding never armed (cli-ux-20 low; ops missed-1 medium; ops missed-4 low) | Global, unvalidated ignores; `--fix` silently no-ops; `up` never writes `RunnerTemplateVersion` | Per-repo validated ignores, "N ignored" notice; read the real runner version from the host (§8.3) | S |
| P2-9 | Runner version model not grounded in the host (ops-diagnostics-10, refuted in part; `gaps.md` runner-version-enforcement) | `ops/doctor.go:149`† compares saved pin with bundled pin using `!=` and "older than" wording; never reads `bin/Runner.Listener --version` | Read the actual version; resolve latest with SHA from release-body markers; bump pin 2.334.0 → 2.337.0 | S–M |
| P2-10 | SSH config partially honored; about 40 serial handshakes before bootstrap (byo-bootstrap-17 low; engineering-quality-17 low) | `-p` always passed; keyscan ignores aliases and ProxyJump; no `IdentitiesOnly`; no ControlMaster; Probe runs twice | `ssh -G` resolution; ControlMaster; one batched probe script | M |
| P2-11 | Ephemeral BYO leaves TTL timer and units behind (cloud-ephemeral-17 low) | `down.go:444-446` disables only the main unit | Remove timer, service, unit files and finalizer dir; `daemon-reload` | S |
| P2-12 | Provider diagnostics dead or misleading (cloud-ephemeral-18 low; cloud missed-4 low) | `Drift: nil` (`provision.go:244`); quota code never emitted; any validation failure reported as "credentials missing" (`up.go:693-708`); power state ignored | Typed `hcloud.IsError` mapping; compute drift | M |
| P2-13 | Default cloud profile trips RunnerKit's own warnings; no swap (cloud-ephemeral-11 low; workload-compatibility-10 medium; ops-diagnostics-18 low) | `profile.go:11`† cpx22 (4 GB); smoke doctor shows `host_mem_low`, `host_swap_constrained` and `time_unsynchronized` on a fresh VM | Swapfile in cloud-init; presets; profile-aware thresholds | S |
| P2-14 | Identity and slug fragility (github-state-security-13/17 low) | Lossy slug plus 63-rune cap means `my.repo`/`my-repo` collide on install path; case-sensitive keys; additive fields dropped by older binaries | Hash suffix; repo ID; case-insensitive keys | S |
| P2-15 | GitHub client brittle (github-state-security-16 low) | No pagination (30 runners), `http.DefaultClient` with no timeout, no retry; the permission probe mints a throwaway token | Timeout, backoff, Link pagination; read-only probe | S |
| P2-16 | Error-code registry diverges from emit sites (ops-diagnostics-12 partially true low) | 19 of 51 codes have no emit site; `Finding` has no `code` field; docs show uppercase RKD lines doctor does not print (codes appear only as lowercase URL anchors) | `code` field everywhere; registry-to-emit-site test | S |
| P2-17 | Cloud-init YAML built with Sprintf (CONCERNS; engineering-quality-15 partially true) | `provision.go:351-405`; a key comment containing `": "` becomes a YAML mapping (PyYAML) | Marshal with yaml.v3; `cloud-init schema` in CI | S |
| P2-18 | Output polish (ops-diagnostics-19 partially true; cli-ux-m6/m7) | Placeholder "runner id 123" (`ops/cleanup.go:43`); hard-coded `owner/repo` (`recover.go:269`); circular next steps; raw Select errors | Golden-output tests for human renderers | S |
| P2-19 | Dead or misleading plumbing (byo-bootstrap-18 low; engineering-quality-14 medium) | `SudoersIsPrepared`, `RemoteVisudoCheckScript`, `RemoteSudoersRemoveScript`, `RunnerCI*` helpers, `Command.RedactArgs` (never read), `checkliststore.Load`, `ImageSetupVersion` never persisted; about 30 `byo-prepare` references | Delete or rewire (for example the sudoers check into doctor) | S |
| P2-20 | Local validation after network calls; `--dry-run` not offline (cli-ux-11 medium) | `up.go:109` GitHub call before mode and cloud validation; cloud Validate before dry-run (`:693` before `:715`) | Validate enums in `PreRunE`; offline plan mode | S |

---

## 6. Security posture

### 6.1 Honest privilege model

| Actor | What the code and docs imply | Effective privilege at HEAD | Mechanism and evidence |
|---|---|---|---|
| BYO SSH user (after install.sh) | "Scoped" NOPASSWD allowlist, "NOT a blanket NOPASSWD ALL" (`sudoers.go:65-66`†) | **Passwordless root** | Unrestricted `/bin/su` (`sudo su -`), `tar` (checkpoint-action), `apt-get` (Pre-Invoke hooks), `systemctl`, `install`, `chown`, `curl`. The Go template adds `tee`, `cp`, `chmod`, `ln`, `dpkg`, `usermod`. The **first** install.sh (v1.0.8) was already root-equivalent; later widening did not change that (verifier correction to product-history-traction-6). Mitigating context: the grantee already had password sudo, so the change removes the password requirement rather than adding privilege. |
| Cloud admin (`runnerkit-admin`) | Scoped sudoers applied through cloud-init | `NOPASSWD:ALL` | `provision.go:385`†. Documented in `cloud-quickstart.md:20`, and intentional. The staged scoped fragment adds nothing on cloud. |
| Runner service user (runs workflow code) | Non-root, isolated | **Can escalate to root on the next lifecycle command** | Install dir, including `svc.sh`, is `chown -R` to the runner user (`script.go:44,53`†). RunnerKit later runs `sudo ./svc.sh …` from it in up, recover, down, destroy and upgrade-runner, and the sudoers glob allows that. This is the stock upstream layout, but CONCERNS.md:81 ("production is safe") is wrong. |
| Runner service user, continued | — | Can tamper with evidence and root-written paths | `/var/lib/runnerkit` is owned by the runner user (`script.go:45`†), while root writes markers, TTL finalizer output and log copies into it. That allows symlink or rename clobbering and forging or emptying the image marker (byo missed-1, M5). |
| Runner service user, docker | "Added to docker group" (docs) | Not in the group on fresh hosts today (P0-2); **root-equivalent once P0-2 is fixed** | `image_setup.go:101`†. Also applies to ephemeral BYO, the mode recommended for untrusted code. Not mentioned in `docs/safety.md`. |
| Runner service user, opt-in CI sudo | Package managers only | Root-equivalent | `apt-get -o APT::Update::Pre-Invoke::=…`. `install.sh:65-93`, `ci_sudoers.go:24-35`. |

**Consequence.** A persistent BYO runner on a personal machine should be treated as "CI owns the host". That is consistent with GitHub's own guidance, but it is not how the product describes itself.

### 6.2 Isolation

- **Multi-repo hosts:** every repo's runner runs as the single `runnerkit-runner` user (`up.go:1608`, `recover.go:212,228`, `upgrade_runner.go:116`, `down.go:291`). A job in repo A can read repo B's `.credentials_rsaparams` (runner identity) and workspace. Mitigation: `docs/troubleshooting/multi-repo.md:24-26` says to treat the host as trusted across repos, and doctor emits an informational `byo.multi_repo_shared_host` finding. There is no trust-tier check when mixing public or ephemeral repos with private persistent ones.
- **Ephemeral:** it limits the GitHub registration to one job but is not a fresh environment:
  - BYO ephemeral reuses the host;
  - `KillMode=process` lets escaped processes survive the service stop;
  - the cloud VM outlives its job until a manual `destroy`.
  `runmode` copy is honest ("BYO ephemeral is not a clean VM"). The docs' "stronger isolation per job" framing is not.
- **No per-job hygiene:** no workspace wipe, Docker prune or job hooks (see §7).

### 6.3 Host keys

- **TOFU against a separate connection.** The pin comes from a separate `ssh-keyscan`. Every real session runs with `StrictHostKeyChecking=no` and `UserKnownHostsFile=/dev/null` (`remote/system.go:214-220`†). An active on-path attacker can pass the keyscan through and terminate the session, which carries the registration token and sudo commands. The verifier rates this medium, because exploitation needs an active MITM.
- **Unverifiable fingerprint.** The displayed `SHA256:` is a hash of the whole keyscan line, including host and port (`system.go:114`), so it never matches `ssh-keygen -lf`, although the docs tell users to verify it. It also changes between IP and DNS addressing.
- **Weak spots in how the pin is used:**
  - `logs` and `upgrade-runner`, which runs the full sudo bootstrap, skip the check entirely;
  - `--yes` auto-accepts unknown keys;
  - the pins are per repo, not per host;
  - on cloud there is no pre-generated host-key injection, so every VM starts with a TOFU window.

### 6.4 Secrets

- **Good:** durable GitHub and Hetzner credentials stay on the workstation. State is secret-free by construction. JSON output is sanitized by key name and by value.
- **Tokens on the host:** registration and removal tokens are expanded by the outer shell into the argv of `sudo su … -c "… --token <value>"` (`script.go:56,109,274,287`). They are therefore visible in `/proc/*/cmdline` and **logged by sudo** (`COMMAND=`) to auth.log and journald, readable by the `adm` group, and they stay there for as long as logs are kept. Tokens are valid for about 1 h. The code comment at `script.go:82` ("never interpolates a token value") is true of the template, not of runtime.
- **Redaction gaps:**
  - at least 5 independent `redact.New()` instances;
  - the runner-token regex matches only the test-fixture shape;
  - `RedactArgs` is never read;
  - debug log files are created 0644;
  - no patterns for 40-hex classic PATs, `Bearer` headers or URL credentials;
  - the `HCLOUD_TOKEN` value is registered in `up`/`destroy` only.
  Leakage needs `RUNNERKIT_LOG=debug` plus echoing output, so the verifier rates this low.
- **Auth precedence:** an ambient `gh auth token` wins over an explicit `RUNNERKIT_GITHUB_TOKEN` (`github/auth.go:42-58`), while every remediation tells users to set the latter. The troubleshooting docs recommend adding the unneeded `workflow` scope (`auth.md:148`, `github.md:137`), which is the scope supply-chain worms use to plant workflows.
- **Unknown:** CONCERNS.md records that a `HOMEBREW_TAP_GITHUB_TOKEN` was pasted into chat during development, and whether it was rotated is not known. This is unverified here and should be confirmed by the maintainer.

### 6.5 Network and supply chain

- **Cloud firewall:** SSH defaults to `0.0.0.0/0` (`profile.go:14`†), root login with the owner key stays enabled (`provision.go:158`), and the host key is TOFU. The firewall otherwise admits only TCP 22, which is a minimal surface.
- **Image setup:** it fetches Go "latest" at run time and adds 5+ third-party apt sources. Key fetches end in `|| true` while `.list` files are written anyway. Artifacts go to predictable `/tmp` paths as the SSH user and are then consumed by root. On Ubuntu defaults (`fs.protected_regular`) the practical effect is mostly denial of service.
- **Runner tarball:** pinned by SHA-256, but verified only on first download (P1-17).
- **Release workflow:** actions pinned to mutable major tags with `contents: write` and `id-token: write`, and no `github.repository` guard (fork tags break OIDC signing; see CONCERNS).

### 6.6 Security defect summary

| ID | Defect | Verified severity | Priority | Fix | Effort |
|---|---|---|---|---|---|
| SEC-1 | "Scoped" sudoers is root-equivalent while code and docs claim otherwise | high (byo-bootstrap-2) / medium (github-state-security-3) | P1 | Say plainly that RunnerKit requires root, **or** a root-owned helper with fixed verbs behind one sudoers line | S (docs) / L (helper) |
| SEC-2 | Runner-writable install dir holds `svc.sh`, which is later run as root | medium | P1 | Root-owned install dir; only `_work`, `_diag` and credentials writable; run a RunnerKit-rendered unit | M |
| SEC-3 | `/var/lib/runnerkit` owned by the runner user while root writes into it | medium | P1 | Root-owned 0755; runner owns only per-runner work dirs | S |
| SEC-4 | Host-key pin not bound to the session; non-standard fingerprint; `logs` and `upgrade-runner` skip the check | medium (cli-ux-m1 rated high) | P1 | Managed `known_hosts` plus `StrictHostKeyChecking=yes` plus `HostKeyAlias`; `x/crypto/ssh` fingerprints; cloud-init-injected host keys | S–M |
| SEC-5 | Docker group (root-equivalent) given silently by design, including to ephemeral runners; undocumented | high (github-state-security-5) | P1, decide together with P0-2 | Opt-in `--docker` or rootless Docker; document; require acknowledgement for persistent public use | S–M |
| SEC-6 | Tokens in argv and sudo logs | medium / low | P2 | Single-quoted `-c` plus a 0600 token file, or JIT config | S |
| SEC-7 | Multi-repo shared UID | medium | P2 (P1 if mixed trust tiers) | Per-repo users, 0700 dirs; trust-tier check in `register` | M–L |
| SEC-8 | Trust gate evaluated only at `up`/`register`; not rechecked in status or doctor; `recover` and `upgrade-runner` re-register without it; private forks mislabeled and override flag leaks through | medium / low | P2 | Recheck visibility in status and doctor; gate re-registration | S |
| SEC-9 | Auth precedence inverted; docs push the `workflow` scope | medium | P2 | Env first; `gh auth token --hostname github.com`; drop the scope advice | S |
| SEC-10 | Redaction split; 0644 logs; fixture-only token regex | low | P2 | One process-wide redactor; 0600 logs; more patterns | S |
| SEC-11 | Cloud SSH `0.0.0.0/0` by default | low | P2 | Default to the caller's IP/32 | S |
| SEC-12 | Opt-in CI apt sudo is root-equivalent and undocumented as such | medium | P2 | Document; or `--ci-sudo` with explicit semantics (P1-11) | S |
| SEC-13 | Release workflow on mutable action tags with no repo guard | low | P2 | Pin actions by SHA; repository guard; test job as `needs:` | S |

---

## 7. Workload compatibility vs GitHub-hosted `ubuntu-24.04`

This reflects a fresh host at v1.3.3/HEAD, on the path where bootstrap completes: cloud, or a NOPASSWD BYO host. On password-sudo BYO hosts nothing runs at all (P0-1). "Hosted" behaviour is from the dossier's workload stream and KEYFACTS.

| Workload | GitHub-hosted | RunnerKit today | Evidence |
|---|---|---|---|
| C/C++ builds, `gcc`, `pkg-config`, `make` | Works | **Works** (Ubuntu x64): 70-package baseline | `install.go:289-308`; `gaps.md` job simulation: gcc passes [R] |
| `docker build` / `docker run` / buildx / compose | Works | **Fails**: `permission denied … docker.sock` | P0-2 [R] |
| `services:` (for example postgres) and `container:` jobs | Works | **Fails** (same cause) | P0-2 [C] |
| `sudo apt-get install …` in a job | Passwordless sudo | **Fails** by default. BYO works only with `RUNNERKIT_GRANT_CI_SUDO=1` at install time; cloud has no job sudo | P1-11 [R] |
| Other `sudo` (tee, mkdir, systemctl, disk-cleanup actions) | Works | **Fails** (CI sudo covers package managers only) | `ci_sudoers.go:24-32` [C] |
| Preinstalled Node / Python / Go / Java / .NET | Current images | Installed on Ubuntu x64: Node **20 (EOL 2026-04-30)**, Go "latest" at install day then frozen, Java 17, .NET 8 | `image_setup.go` [C] |
| Rust / cargo | Preinstalled | **Missing** (rustup runs as a user that does not exist yet) | P0-2 [R] |
| Ruby | Preinstalled on hosted | **Missing**, although `cloud-quickstart.md` lists "Ruby (system)" | workload-compatibility-13 [C] |
| `actions/setup-*` and tool cache | Works | Works: downloads into `<work>/_tool`. No shared `/opt/hostedtoolcache` and no `ImageOS` env | workload-compatibility features [C] |
| `actions/cache`, artifacts | Works | Works (same GitHub service; zstd installed) | [C] |
| Browsers / E2E (Chrome, Firefox, drivers, xvfb) | Works | Partial: x64 only; Firefox is the snap transitional package; no WebKit/GStreamer deps; `playwright install --with-deps` fails (needs general sudo) | workload-compatibility-15 [C] |
| Matrix legs and parallel jobs | Up to 20 concurrent (Free) / 40 (Pro) | **Serialized**: one runner per repo | P1-22 [C] |
| Clean environment per job | Fresh VM per job | Persistent: none (no workspace wipe, Docker prune or job hooks; root-owned files from container jobs can break the next checkout). Ephemeral: one job, then a manual re-`up` | workload-compatibility-9/11 [C] |
| Disk | Hosted image disk | Image needs about 4.5–5 GB and about 1.4 GB of downloads; preflight requires 2 GiB; doctor warns only below 2 GiB | `gaps.md` (projection) [I] |
| Memory | Managed sizing (not quantified in the dossier) | Default cpx22 = 2 vCPU / 4 GB, no swap; RunnerKit's own memory and swap warnings fire on a fresh VM | P2-13 [R] |
| Job timeout | 6 h | Up to 5 days (self-hosted limit) | KEYFACTS §D |
| arm64 | arm64 hosted runners exist | **Broken** (image setup amd64-only) | P1-6 [C] |
| macOS / Windows / GPU / Android emulator / iOS | macOS and Windows hosted available | **Unsupported**. Docs wrongly list macOS as supported | workload-compatibility-13 [C] |
| First runner online | Instant | Cloud: 498 s for up + status + doctor×3 + list + destroy on v1.3.2 code, of which about 300 s was the destroy-verify poll; it also includes `go run` compilation, so `up` alone was well under 498 s but was not measured separately. BYO projected 3.5–5 min on a fast host, 8–12 min on 2 vCPU or 25 Mbps; no BYO success recorded since image setup landed | `03-smoke-cloud.log`; `gaps.md` [R]/[I] |
| Runner version currency | Managed | Pin 2.334.0 (latest 2.337.0). Auto-update left on, so it should stay routable under the 30-day rule. Each fresh install pays a self-update before its first job | `gaps.md` runner-version-enforcement [I, medium] |
| Agent workloads (secondary; `gaps.md`) | — | claude-code-action: likely works, but the shared HOME across jobs is a known leak (#1688), and its optional bubblewrap `sudo apt-get` step works only on BYO hosts installed with `RUNNERKIT_GRANT_CI_SUDO=1` (never on cloud). Copilot cloud agent: probably routable on Ubuntu x64, though GitHub recommends single-use runners. gh-aw: needs the runner user in the docker group, so it is blocked on fresh hosts by P0-2 (the agent-runner answer in `gaps.md` assumed the group was applied; the released-binary reproduction in the same file shows it is not). Copilot code review: not possible (ARC-only). codex-action: no (needs `sudo chmod/chown` and disposable hosts) | [I] |

Nothing in the smoke suite would catch these gaps: the smokes check that the runner is online and idle and **never dispatch a workflow job** (`scripts/smoke/cloud-end-to-end.sh:41-63`; no `workflow_dispatch` or `gh workflow run` anywhere).

---

## 8. Engineering quality

### 8.1 Tests: why a green suite missed these bugs

**Headline numbers** (engineering-quality stream):
- 548 tests pass in about 18.8 s, clean under `-race` (52 s).
- 13,040 test LOC under `internal/` and `cmd/`† (13,095 including the 55-line root `install_sh_test.go`) against 15,399 production LOC†.
- Coverage 69.5% overall. By package: errcodes 100, runmode 97.4, redact 86.2, bootstrap 81.8, update 80.3, labels 78.0, preflight 76.1, ops 73.4, provider 73.3, hetzner 72.6, cli 72.4, github 66.3, state 65.6, cmd/runnerkit 50.0, remote 43.8.
- `go vet` clean; `go mod tidy -diff` clean.

**Why the suite missed the bugs:**

1. **Rendered bash is asserted as strings, never executed.** Bootstrap and cloud-init tests check substrings through fake executors that always succeed. Every P0/P1 shell defect lives in rendered bash that no test runs: sudoers drift, step order, `set -e` in the cloud-init wait, cache verification, auto-detect parsing, the finalizer's privileges and `--skip-old-files`.
2. **Real adapters have 0% coverage.** That covers `remote/system.go` (`Run`, `Probe`, `sshArgs`, `scanHostKey`), `hetzner/client.go`, the GitHub service's token and runner methods, `runRegister`, `runInit`, `applyDoctorFixes` and `CollectBoundedJournalsForHints`.
3. **Parity tests check too little.** `install_sh_test.go:24-38` compares only the header line of the two sudoers sources. `preflight/checks_test.go:276-289` greps the source for a literal. `ui/cli_prompter_test.go:18` compares an interface to nil and can never fail (staticcheck SA4023).
4. **Test wiring differs from production wiring.** Tests always inject `StateBaseDir`, so P1-3 is invisible. `cmd/runnerkit/main_test.go` checks that `Prompts` is wired, not that it implements `Input` (P1-2). Order tests never set `OSReleaseID`, so `setup_runner_image` never appears in the tested order (P0-2).
5. **Integration and live coverage are thin:**
   - The only real-shell test (`install_integration_test.go`, `//go:build integration`) is never run in CI.
   - Live smokes are manual only; `Makefile` forbids CI invocation (D-11).
   - The smokes never run `logs`, `recover` or `upgrade-runner`, never exercise ephemeral mode, and never dispatch a job.
6. **Tests are not hermetic.** `go test ./...` writes `internal/cli/sessions/byo-owner_repo__alice_example_com.json` into the checkout; the file is present, and `.gitignore` entries were added rather than fixing the cause. Tests also call the live GitHub releases API, and some fall back to the real state dir and system ssh.

**What would have caught them.** A hermetic host-e2e job: sshd in `ubuntu:24.04` and `debian:12` containers, arm64 via QEMU, an httptest fake GitHub API, and the real `SystemExecutor` running `Apply`/`ApplyEphemeral`/`down`/`doctor`. It would assert the host end state: files, unit, `id -nG`, sudoers coverage and `docker info` as the runner user. The `released-byo-e2e-job` run in `gaps.md` is effectively a prototype of this. Add a canary workflow dispatch before tagging.

### 8.2 CI and release gates

- **`pr-checks.yml`:** GoReleaser check and snapshot build with archive assertions, plus `go test ./... -count=1 -race` on Ubuntu only. There is **no** gofmt check (9 files unformatted), no staticcheck (46 findings, including dead functions and a never-failing test), no golangci-lint, no govulncheck, no shellcheck of rendered scripts, and no macOS job although darwin binaries ship. `make lint` is a declared phony target with no recipe.
- **`release.yml`:** a single GoReleaser job with **no test step** and no `needs:`, triggered by any `v*` tag. It has no `github.repository` guard, and actions are on mutable major tags. In practice release and PR checks started about 5 s apart on the same SHA for v1.3.1–v1.3.3. That timing claim was not independently verified; the structural absence of a gate was. Verifier caveat: `pr-checks.yml` also runs the tests on pushes to `main`, so a tag cut from `main` is normally tested indirectly. What is missing is a gate that blocks the release when those tests fail.
- **Human release gate:** the pre-tag checklist (`docs/release-process.md`: `make smoke-live` plus a stopwatch, "do NOT tag if > 10 min") was skipped in the final burst:
  - v1.3.0, v1.3.1 and v1.3.2 were tagged within about 4 h on 2026-05-13;
  - v1.3.1 introduced `setup_runner_image` and broke BYO, which was found 5 days later;
  - v1.3.3 was tagged without a BYO re-smoke, and its fix never reached BYO.
- **Distribution supply chain is good.** 4-platform archives, cosign keyless signature over checksums (including `install.sh`), Homebrew cask in a separate tap, and the last 12 release runs succeeded.
- **No Dependabot or Renovate.** `.github/` contains only `workflows/`.

### 8.3 Toolchain and dependency staleness

| Component | Pinned | Current (per dossier) | Note |
|---|---|---|---|
| Go (`go.mod`†, CI†) | `go 1.22`; released binary built with go1.22.12 | 1.22 out of support since Feb 2025. "1.27.1 current" was **not** independently confirmed by the verifier | The pin is self-imposed and changes in one line. A scratch bump to 1.26.3 built and passed all tests (`gaps.md`) |
| `golang.org/x/net` | v0.12.0 (2023-07-05) | v0.59.0 | Indirect. govulncheck could not run (vuln.go.dev blocked), so CVE exposure is **unmeasured**. For a client CLI most stdlib CVEs are server-side, which is why the verifier rated this medium |
| `x/sys`, `x/term`, `x/text` | v0.10.0, v0.10.0, v0.11.0 | v0.48, v0.46, v0.42 | Latest versions need Go ≥1.26 |
| `hcloud-go` | v1.59.2 (2024-11-22, last v1 release) | v2.49.0 (2026-09-22) | The API removed `server.datacenter` (2026-07-01) and deprecates `/v1/datacenters` (410 after 2026-10-01). These dates come from hcloud-go release notes and the v2.49.0 schema (verifier-confirmed), not from the live Hetzner API, which was blocked. RunnerKit only reads `server.Datacenter` and degrades to the saved region, so the impact is low. The typed `StatusCode()` error branch is dead (`hcloud.Error` lacks it). Deprecated `Image.GetByName`/`Server.Delete` (SA1019) |
| actions/runner | 2.334.0 (`bootstrap/package.go:5`†) | 2.337.0 (2026-08-26) | Kept routable by auto-update [I]. The registration floor is 2.329.0 |
| Node (image) | 20.x | EOL 2026-04-30 | |
| Go (image) | "latest" at install time, then frozen by the marker | — | Unpinned, not reproducible |

### 8.4 Architecture debt

- **God package:**
  - `internal/cli` is 6,030 LOC, 39% of production code; `up.go` is 2,328 LOC.
  - `runUp` is about 221 lines and `runCloudUp` about 192. Each interleaves policy, prompts, provisioning, bootstrap, state and rendering.
  - The composition root lives in `cli.normalizeDependencies`, not `main`. That is why the `StateBaseDir` default was never applied.
- **Duplicated persistent and ephemeral paths:**
  - `RenderInstallScript` vs `RenderEphemeralInstallScript` (`script.go:30-58` vs `80-110`);
  - `Apply` vs `ApplyEphemeral` step lists (`install.go:108-140` vs `158-185`).
  Every fix must be applied twice, and the ordering bug is in both.
- **Two sources of truth for privileges:** `install.sh` and `RenderSudoersEntry`, with no binding test. The Go-side installer is dead code.
- **Domain types in the API-client package:**
  - `gh.Repo` has no json tags, so `state.json` mixes `FullName`/`Private` with snake_case keys, and renaming needs a migration;
  - additive state fields deliberately do not bump the schema, so older binaries drop them.
- **Provider abstraction leaks Hetzner.** The 7-method interface is clean, but the CLI lookups, `isCloudProvider`, destroy copy, state schema (`PrimaryIPv4AutoDelete`, `FirewallID`), the cloud-init renderer (in the `hetzner` package), root-SSH readiness and the fake's default name are all Hetzner-shaped. A second provider would touch 6–8 files plus the schema.
- **Static plan vs executed plan.** `bootstrap.Plan` ignores its options and returns a fixed list, so the plan users approve can never match what runs.
- **Laptop-driven long privileged sessions.** About 40 serial SSH handshakes, no multiplexing, no host-side execution, no command timeouts, buffered output. The strategic alternative in the dossier is to upload the plan and run it as a transient systemd unit on the host with per-step markers.
- **Dead code:** `deadcode` reported 33 unreachable functions (the verifier could not rerun it but confirmed the key ones by grep), plus unused types (`ProjectConfig`, `checkliststore.Load`, `ui.PasswordPrompter`).

### 8.5 Docs drift

| Location | Claim | Reality at HEAD |
|---|---|---|
| `RELEASE-NOTES-v1.3.3.md:7,12,26` | Rerun `runnerkit byo-prepare` to get the fix | The command was deleted in v1.0.8 and exits 1 silently. The fix only reached cloud-init |
| `docs/byo-quickstart.md:35` | install.sh grants tee, gpg, mkdir, unzip, usermod, dpkg, add-apt-repository | It does not (P0-1) |
| `docs/byo-quickstart.md:75`; CLAUDE.md | Non-Ubuntu hosts "receive only fix_dependencies with baseline tools" | Ubuntu package names go to dnf/yum and fail; Arch/openSUSE exit 20 |
| `docs/cloud-quickstart.md:23-32` | "The same software as GitHub-hosted Ubuntu 24.04", Ruby, runner "added to the docker group" | No Ruby; no docker group (P0-2); no Rust |
| `docs/cloud-quickstart.md:70`, `docs/safety.md:88` | Ephemeral TTL means the runner is "cleaned up automatically" | The TTL only stops the service. The VM bills until `destroy` (the same pages also say so, which is contradictory) |
| `docs/cloud-quickstart.md:93-102` | Use `.runnerkit/config.yaml` | No loader exists |
| `docs/runner-platforms.md:11-12` | arm64 on both paths; macOS "Supported" (advanced BYO) | arm64 broken; preflight rejects non-Linux and non-systemd hosts |
| `docs/troubleshooting/bootstrap.md:430,442,66,386` | cloud-init v2; sudoers "scoped to RunnerKit bootstrap commands only"; `systemctl status runnerkit-runner` | Code is v3; the fragment is root-equivalent; that unit does not exist |
| `docs/troubleshooting/cleanup.md:111-121,242,343` | `rm -rf /opt/actions-runner/runnerkit-* /var/lib/runnerkit`; `rm state.json` | Destroys every repo's runner and all state on multi-repo hosts |
| `docs/troubleshooting/github.md:17-21` | Doctor prints `RKD-GH-001: …` | Codes appear only as lowercase URL anchors |
| `docs/troubleshooting/auth.md:148` | Add the `workflow` scope | Not needed; widens exposure |
| `docs/upgrade.md:47-48` | `upgrade-runner` is "idempotent — safe to re-run" | Destroys registration (P0-3) |
| `CLAUDE.md` | Sessions saved "inside the state directory"; readiness "rejects status: error" (implying fail-fast); state tracks `ImageSetupVersion` for re-run decisions; "~75" baseline packages | CWD; 15-min retry; never populated; 70 |
| `README.md` | `TAG=v1.0.0` snippets (annotated "replace"); D-01/D-02/D-05 decision IDs; a test-fixture sentence at `:90`; a maintainer section before the pitch | v1.0.0 has no GitHub Release; internal jargon |
| Repo root | `GEMINI.md`, `smoke-output.log`, 8 release-notes files (no v1.3.0–v1.3.2), no CHANGELOG, no LICENSE, no SECURITY/CONTRIBUTING; `.planning/` has 147 files, including the maintainer's personal host string in 28 files | Presents as an AI-planning workspace. Legally not open source |

Planning-corpus drift has caused real defects:
- all four seeds say `status: dormant`, although SEED-001, SEED-002 and SEED-004 shipped;
- FINDINGS.md, RECOMMENDATION.md and the v1.3.3 notes all assumed `byo-prepare` still existed, 7 days after it was deleted. That is why the v1.3.3 fix was applied to the wrong allowlist (product-history missed-6).

---

## 9. Status of `.planning/codebase/CONCERNS.md` items at HEAD

CONCERNS.md is dated 2026-05-17, the day before v1.3.3. Status legend:
- **Open**: still true.
- **Open, worse**: true and understated.
- **Stale**: resolved, or wrong when written.
- **Partly**: partly addressed.

| CONCERNS item | Status at HEAD | Evidence and notes |
|---|---|---|
| Deferred BYO bootstrap architecture (SEED-001) | **Stale** (as "deferred") | SEED-001's deliverables landed in f017b2c (v1.0.8, 2026-05-11): `byo_prepare.go` and `sudo_rewrite.go` deleted, install.sh/`init`/`register` added. The seed file still says "dormant". The underlying fragility (allowlist growth) remains; see the next two rows. |
| `byo-prepare` exists in comments only | **Open, worse** | About 30 references, now including user-facing `RELEASE-NOTES-v1.3.3.md:12,26` and CLAUDE.md. `sudoers.go:52` still cites it. |
| install.sh allowlist drifts from `RenderSudoersEntry` | **Open, worse** | CONCERNS listed 8 missing paths; v1.3.3 added 8 more to Go only, so 16 are missing. The test still checks only the header. Now reproduced as a total BYO failure (P0-1). |
| `.runnerkit/config.yaml` loader missing | **Open** | `up.go:189,682` pass `nil`. The docs still advertise it. |
| `runUp` (283 LOC) / `runCloudUp` (193 LOC) too large | **Open** | About 221 and 192 lines now; `up.go` is 2,328 LOC. |
| Cloud-init YAML via Sprintf | **Open** | Valid YAML for normal input (PyYAML), but a key comment containing `": "` becomes a mapping. |
| "Known bugs: none open in source" | **Stale / wrong** | See §5. At least 6 P0 and 25 P1 defects are open. |
| Latent bug: no signal handling | **Open** | No `signal.Notify` anywhere†. The verifier notes the cloud orphan window is small because of the checkpoint. |
| SSH host-key verification disabled | **Open, worse** | Also: the pin is not bound to the session, the fingerprint is non-standard, and `logs`/`upgrade-runner` skip the check (SEC-4). |
| Bare `HCLOUD_TOKEN` not registered with redactor | **Partly** / stale as written | Registered in `up` (`registerKnownCloudProviderSecrets`, `up.go:852-856`) and `destroy` (`:152`) since 2026-05-06. Still not in status, doctor or the provider client, and the GitHub token is registered only in the service's private redactor. |
| Sudoers wildcard `…/runnerkit-*/svc.sh`: "production is safe" because the dir is root-owned | **Wrong** | The install dir is `chown -R` to the runner user (`script.go:44,53`†), so any job can rewrite `svc.sh` (SEC-2). The concern is also moot, because the fragment is root-equivalent anyway (SEC-1). |
| Package-name validator allows `+`/`:`; auto-detect notice hidden in JSON | **Open, worse** | Still hidden under `--json` (`up.go:2297`). The bigger problem is mis-parsing and no opt-out (P1-5). |
| Host-key probe budget hard-coded at 5 min | **Open**, premise wrong | Still hard-coded (`provision.go:593-596`). But `setup_runner_image` runs over SSH after readiness, not in cloud-init `runcmd`. |
| `setup_runner_image` is one giant script | **Open** | Plus: marker freezes partial failures (P1-16); ordering bug (P0-2). |
| Preflight runs about 12 SSH round-trips | **Open, worse** | 17 ssh calls plus a keyscan per Probe; `up` probes twice (about 40 handshakes). |
| Path C sudoers must stay in lockstep; no binding test | **Open** | The drift materialized (P0-1). Still no test. |
| Cloud destroy depends on `AutoDelete`; no test | **Partly stale** | A regression test exists (`provision_test.go:203`, since 2026-05-06). The dependency itself remains. |
| Probe-classify split in `preflight.Run`; English-only stderr matching | **Open** | Plus: the `no_sudo` branch is unreachable, and a user not in sudoers is sent to install.sh. |
| ssh-keyscan algorithm precedence | **Open** | Unchanged (`system.go:137-174`). |
| State file is a single JSON document | **Open** | CONCERNS omits the missing lock (P1-25). |
| One Hetzner VM per `up` | **Open** | Plus: `--replace` can orphan the previous VM (P0-5). |
| `setup_runner_image` cold boot adds 3–5 min per cloud `up` | **Open** | No pre-baked image. |
| hcloud-go v1.59.2 | **Open, worse** | v2.49.0 is current; the API dropped `server.datacenter`; SA1019 deprecations. |
| Bundled runner pinned to 2.334.0; no bump automation | **Open**, mitigated | Latest is 2.337.0. Auto-update keeps runners routable [I]. |
| `x/term v0.10.0` old | **Open** | And x/net, x/sys and x/text are equally old; Go 1.22 is out of support. |
| Missing: `destroy --orphans` | **Open** | Smoke-only lister. |
| Missing: `runnerkit byo-prepare` | **Stale framing** | The right fix is to remove the references and point at install.sh, not to implement the command. |
| Test gaps 1–6 (cloud-init schema, sudoers-to-script binding, install.sh byte equality, apt-parse edge cases, destroy retry budget, `ApplyEphemeral` recovery) | **All open** | Gap 3 is now proven to have shipped a P0. |
| Release foot-gun: no fork guard in `release.yml` | **Open** | |
| Release foot-gun: `HOMEBREW_TAP_GITHUB_TOKEN` rotation | **Unknown** | Not verifiable from the repo. |
| Release foot-gun: `smoke-live` is manual-only | **Open** | And it was skipped for v1.3.0–v1.3.3. |
| Release foot-gun: BYO smoke needs a TTY | **Stale** | Obsolete since the install.sh pivot. |

**Serious defects CONCERNS.md never listed:**

| Defect | Reference |
|---|---|
| Step ordering, which breaks Docker and Rust | P0-2 |
| Unset `StateBaseDir` | P1-3 |
| cloud-init `set -e` dead code and the 15-min retry-all | P1-4 |
| Auto-detect mis-parsing | P1-5 |
| Root-equivalent "scoped" sudoers | SEC-1 |
| Non-hermetic tests | §8.1 |
| `upgrade-runner` empty token | P0-3 |
| `down` with no cloud guard | P0-4 |
| `--replace` orphaning | P0-5 |
| Hard-coded cost | P0-6 |
| Missing `Input` prompter | P1-2 |
| Silent CLI errors | P1-1 |
| Wrong journal unit | P1-9 |
| Miswired recover | P1-10 |
| Ephemeral TTL and finalizer | P1-12 |
| Non-converging cleanup | P1-13 |
| arm64 and non-Ubuntu breakage | P1-6, P1-7 |
