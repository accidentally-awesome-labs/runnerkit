# RunnerKit — maintainer and agent notes

## Rules (read first)

- **No feature without 2 distinct external requests.** The only exemption is the Stage 1 validation work W1–W3 (`runnerkit-watch`, `runnerkit checkup`, recipes), capped at 30 build hours in total. Bug, security and docs fixes are always in scope. See [`CONTRIBUTING.md`](CONTRIBUTING.md).
- **`RenderSudoersEntry` gains no entries.** The installer sudoers fragment (`internal/bootstrap/sudoers.go`) is **root-equivalent** (`su`, `tee`, `cp`, `apt-get`, `systemctl` with any arguments give a root shell); never call it "scoped". `install.sh`'s block between `# BEGIN runnerkit-sudoers` / `# END runnerkit-sudoers` is **generated** from it (`go generate ./...`, `internal/bootstrap/gen_installsh.go`); never hand-edit it. `TestInstallShSudoersMatchesTemplate` and `make generate-check` enforce this.
- **Never tag a release that claims BYO works** without a real GitHub job on a fresh password-sudo host prepared only by that release's `install.sh` (A-20 manual run for v1.3.4; a CI canary later). Otherwise the release says BYO is not supported. Runbook and scripts: [`docs/testkit/`](docs/testkit/README.md) (`scripts/testkit/gate.sh`; also the V-1 to V-4 validations). Kit workflows only run on `workflow_dispatch` (or on a schedule on GitHub-hosted runners) in a private repository (`testkit_test.go`).
- **Never pass `--disableupdate`** to the runner's `config.sh`; runners must keep updating themselves (`TestRenderedScriptsNeverDisableUpdate`).
- Never print a cloud price that was not fetched from the Hetzner API at plan time.
- Record user-visible changes in [`CHANGELOG.md`](CHANGELOG.md) (Keep a Changelog) and keep its Known-issues block in sync with the README "Known issues" section. Security weaknesses and their fix stages live in [`docs/security-posture.md`](docs/security-posture.md).

## Disabled and refused commands (v1.3.4)

`upgrade-runner`, `doctor --fix`, `recover --reinstall-service` and `recover --reregister` refuse with `command_disabled` (exit 2) before any remote call; `lifecycle_invariants_test.go` fails if any command other than `up`/`register` sends `config.sh --unattended`, deletes `.credentials` or runs `svc.sh install`. BYO `up`/`register` refuse with `byo_unsupported_release` (exit 2) unless `--accept-known-issues` is passed (A-21: v1.3.4 shipped without a passing real-job gate; `Dependencies.BYOUnsupportedRelease`, set in `cmd/runnerkit/main.go`). `down`/`unregister` refuse cloud state (`wrong_cleanup_command`); `up` refuses to replace live cloud state (`cloud_state_exists`). `--cloud` and BYO `--mode ephemeral` need `--experimental`; `--cloud` needs `--cloud-region` (no default); ephemeral cloud is disabled. Gates live in `internal/cli/experimental_gates.go`. `byo-prepare` is a hidden tombstone (`command_removed`); it was removed in v1.0.8.

## Shipping changes to end users

**Merging to `main` does not update Homebrew or GitHub Releases.** Install docs point users at those channels; they advance only when a **`v*`** tag is pushed on the **upstream** repo (`accidentally-awesome-labs/runnerkit`), which triggers `.github/workflows/release.yml` (GoReleaser).

**Rough sequence**

1. Land work on `main` (PR merge or direct push).
2. Run the **pre-tag checklist** in [`docs/release-process.md`](docs/release-process.md): CI green, `make generate-check`, govulncheck result read, the real-job BYO gate (or the "BYO not supported" fallback), Homebrew tap PAT verified, CHANGELOG section final.
3. Choose the next **SemVer** tag (`v1.0.x` patch for fixes/small additive CLI; bump minor/major when warranted).
4. Create an **annotated** tag and push **only the tag** (or push tag after verifying commit):

   ```bash
   git fetch origin && git checkout main && git pull origin main
   git tag -a vX.Y.Z -m "RunnerKit vX.Y.Z — short summary"
   git push origin vX.Y.Z
   ```

5. Confirm in GitHub **Actions** that the release workflow succeeded.
6. Confirm **GitHub Releases** has assets for `vX.Y.Z` and **`accidentally-awesome-labs/homebrew-tap`** received the cask bump (GoReleaser commit, e.g. `runnerkit: bump cask to vX.Y.Z`).

**Release gate:** `release.yml` runs `go vet`, a `go generate` drift check and `go test -race` before GoReleaser, and GoReleaser runs only when `github.repository` is upstream. **Fork caveat:** tag pushes from forks only run the test job — always release from the upstream repository.

Full prerequisites (Homebrew PAT, optional Apple notarization), failure modes, and verification commands: **`docs/release-process.md`**.

**Live smoke (`make smoke-live`, D-11):** After interactive `runnerkit doctor`, BYO and cloud scripts run **`scripts/smoke/assert-doctor-json-contract.sh`** to assert **`doctor --json`** includes **`schema_version`**, **`stage`**, **`host_incident_hints`** and **`next_actions`** as JSON arrays (never `null`) and **`doctor --deep --json`** exits 0. They also run **`scripts/smoke/assert-list-json-contract.sh`** on **`list --json`** (SEED-002). Requires **`python3`**. Override **`RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1`** to skip the deep pass. The cloud leg needs **`RUNNERKIT_SMOKE_CLOUD_REGION`** (it passes `--experimental --cloud-region`). These smokes never dispatch a workflow job, so they do not satisfy the BYO release rule above.

**BYO multi-repo smoke (optional):** Set **`RUNNERKIT_SMOKE_MULTI_REPO=1`** and **`RUNNERKIT_SMOKE_REPO2=owner/other`** (second trusted private repo, different from **`RUNNERKIT_SMOKE_REPO`**) before **`make smoke-live-byo`** / **`make smoke-live`**. The BYO script then **`register`**s the second repo on the same host, asserts two repos via **`scripts/smoke/assert-list-host-repo-count.sh`**, runs the doctor JSON contract for repo2, then **`down`** repo2 then the primary.

## Hetzner cloud provisioning (cloud-init v3) — experimental, frozen

Cloud is behind `--experimental` and frozen except for safety fixes. Plans are priced live from the Hetzner API (`internal/provider/hetzner/pricing.go`: server type price for the location + primary IPv4; JSON `estimated_*_cost` objects with `source: "hetzner_api"`); an unpriced location is refused (`cloud_location_unpriced`). There are no hard-coded prices.

When RunnerKit creates the VM (`runnerkit up --repo … --experimental --cloud hetzner --cloud-region <loc>`), **user-data** applies the same root-equivalent `/etc/sudoers.d/runnerkit-installer` rules as `install.sh` (`internal/bootstrap/sudoers.go`), validated with **`visudo`** before SSH bootstrap runs; the admin user also has `NOPASSWD:ALL`, and SSH is open to `0.0.0.0/0` by default. Readiness uses **`cloud-init status --wait`** and does not accept **`status: error`** as ready, but it is **not fail-fast**: every failure (including a deterministic `status: error`) is retried until the 15-minute deadline (`defaultCloudInitTimeout`, P1-4). **`waitCloudTargetReady`** runs preflight with **`RequirePasswordlessSudo`** so missing NOPASSWD surfaces as **`host.privilege.cloud_bootstrap`** before bootstrap. Inventory records **`runnerkit-cloud-init-v3`** (constant **`hetzner.CloudInitUserDataVersion`**); the host also writes **`/var/lib/runnerkit/cloud-init.json`**. Generic **`--host`** machines are unchanged: they still need the one-time host install when sudo is password-protected.

## Extra packages (`--extra-packages` + auto-detection)

CI workflows often need OS-level dependencies (native libraries, GUI test infrastructure, build tools) that the base Ubuntu image does not include. RunnerKit resolves extra packages from three sources (merged, deduplicated, CLI wins):

1. **Auto-detection** — `scanWorkflowExtraPackages` (`internal/cli/workflow_packages.go`) scans `.github/workflows/*.{yml,yaml}` in CWD for `apt-get install` / `apt install` commands (handles `sudo`, flags, backslash continuations) and extracts package names. Runs automatically during `runnerkit up` for both BYO and cloud paths.
2. **`--extra-packages "pkg1,pkg2"`** — explicit CLI flag, merged on top of auto-detected.
3. **`.runnerkit/config.yaml` `defaults.extra_packages`** — project-level defaults (plumbing ready, YAML loader not yet integrated).

**Baseline packages** (`bootstrap.BaselinePackages`): 70 apt packages matching the GitHub-hosted Ubuntu 24.04 runner image are always installed — `build-essential`, `pkg-config`, `gcc`, `g++`, `make`, `curl`, `jq`, `unzip`, and many more. Without them, compiled-language CI fails with "linker cc not found" or missing pkg-config probes. On cloud-provisioned hosts, baseline packages are installed via cloud-init; the `fix_dependencies` bootstrap step skips them (deduplication via `CloudProvisioned` flag in `bootstrap.Options`).

**Runner image setup** (`bootstrap.RenderImageSetupScript`): Step order is `fix_dependencies` → `create_runner_user` → `setup_runner_image` (shared `prepareHostCommands`), so `usermod -aG docker` and the per-user rustup install find the user. The Ubuntu/Debian-gated `setup_runner_image` step installs language runtimes (Node.js 20 LTS, Python pip/venv, Go, Rust, Java 17, .NET 8), Docker CE, browser testing tools (Chrome, ChromeDriver, Firefox, Geckodriver), and CLI tools (gh, cmake, ninja, zstd). The script is idempotent (each tool section checks `command -v`) and writes a marker at `/var/lib/runnerkit/image-setup.json` with `ImageSetupVersion` (now `"2"`; bumping it makes hosts re-run the script). The host marker is what gates re-runs: `RepositoryState.ImageSetupVersion` exists but is **never populated**. The `docker` group is root-equivalent for every job (disclosed, SEC-5). OS gating: `isUbuntuLike(opts.OSReleaseID)` skips image setup on other distros, but non-Ubuntu hosts are **unsupported**: Ubuntu package names still go to dnf/yum in `fix_dependencies` and fail. arm64 is also unsupported (the image script downloads amd64 artifacts). Every bootstrap script starts with an `RKFAIL` ERR trap so failures name the step and command (`bootstrap.FailedCommand`).

Cloud path: baseline + extra packages injected into cloud-init `packages:` (installed at first boot). BYO path: baseline packages installed alongside missing tools during the `fix_dependencies` bootstrap step. Both paths run `setup_runner_image` after `fix_dependencies` on Ubuntu/Debian. Recorded in `RepositoryState.ExtraPackages` (`upgrade-runner`, which used to re-install them, is disabled). Auto-detection always runs (there is no switch to turn it off) and can turn comment words into package names (P1-5). Package names are validated: only alphanumerics, hyphens, dots, colons, underscores, and `+` are accepted.

Implementation: `internal/cli/workflow_packages.go` (`scanWorkflowExtraPackages`, `extractAptPackages`), `internal/cli/up.go` (`autoDetectExtraPackages`, `resolveExtraPackages`, `parseExtraPackages`), `internal/bootstrap/install.go` (`BaselinePackages`, `mergePackages`, `isUbuntuLike`), `internal/bootstrap/image_setup.go` (`RenderImageSetupScript`, `ImageSetupVersion`), `internal/provider/hetzner/provision.go` (`cloudInitUserData`), `internal/bootstrap/sudoers.go` (`RenderSudoersEntry`; root-equivalent, no new entries), `internal/bootstrap/installsh.go` + `gen_installsh.go` (generates `install.sh`'s block).

## Multi-repo BYO (SEED-002, v1.2+)

**v1.2 scope:** multi-repo on a **single BYO SSH host** (same `user@host` for each `runnerkit up` / `register`). **Cloud** remains one provisioned server per `runnerkit up --cloud` unless you manually point a second repo at an existing machine’s SSH address. Tarballs cache under **`/opt/actions-runner/runnerkit-shared-bin/<runner-version>/`**. All repos on a host share one Unix user (`runnerkit-runner`), so they must share a trust level (SEC-7). Narrative: [`docs/troubleshooting/multi-repo.md`](docs/troubleshooting/multi-repo.md).

## UX polish layer (SEED-004, v1.1+)

Line-oriented CLI only (no full-screen TUI). **`runnerkit`** with no subcommand runs a **first-run wizard** when there are no saved repos; **`--explain`** / **`--unicode`** are root persistent flags; **`doctor --ignore`** persists in **`config.json`** (`doctor --fix` is disabled). BYO **`up`**/**`register`** prints **checklists** and saves progress under **`sessions/`** inside the state directory — true only since v1.3.4 (A-03); v1.1.0–v1.3.3 wrote `config.json` and `sessions/` into the CWD because `StateBaseDir` was unset. Tests use `internal/testsupport/statedir.go` so nothing lands in the CWD; delete any stray `internal/cli/sessions/`.

Implementation touchpoints: `internal/ui/box.go`, `internal/ui/checklist.go`, `internal/ux/stage/`, `internal/ux/checkliststore/`, `internal/cli/wizard.go`, `internal/cli/byo_checklist.go`, `internal/cli/explain.go`, `internal/cli/doctor_fix.go`, `internal/cli/userconfig.go`; JSON helpers in `internal/ux/nextaction/nextaction.go`.

## Host capacity, OOM, and `runnerkit doctor` (Phase 7)

When users hit **runner offline**, **systemd failed**, or **CI OOM / `ld` signal 9** on small self-hosted VMs:

- **Preflight** reads `MemAvailable` / `SwapFree` from `/proc/meminfo` over SSH. Below **4 GiB** MemAvailable → warning `host.mem_available` (**RKD-BOOT-016**). No swap and MemAvailable **&lt; 8 GiB** → **RKD-BOOT-017**. Warnings do **not** fail `preflight.Passed()`. Override threshold with **`RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES`** (positive integer, bytes).
- **`runnerkit doctor`**: same preflight findings via `host_mem_low` / `host_swap_constrained`. When SSH works and the runner looks **sick** (service failed, GitHub offline/missing runner), or when the user passes **`--deep`**, RunnerKit pulls **bounded** `journalctl` excerpts and runs **heuristic** OOM/SIGKILL detection → finding **`host_incident_hints`** (**RKD-BOOT-018**), JSON field **`host_incident_hints`**. **`--with-log-snippets`** adds short **redacted** matching lines.
- **Narrative doc:** [`docs/troubleshooting/host-resources.md`](docs/troubleshooting/host-resources.md) (index: [`docs/troubleshooting/README.md`](docs/troubleshooting/README.md)).

Implementation touchpoints: `internal/preflight/checks.go`, `internal/remote/system.go` + `meminfo.go`, `internal/ops/hostkillhint.go`, `internal/ops/logs.go` (`CollectBoundedJournalsForHints`), `internal/cli/doctor.go`, `internal/ops/doctor.go`, `internal/errcodes/codes.go`.
