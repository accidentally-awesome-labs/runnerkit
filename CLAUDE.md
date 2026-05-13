# RunnerKit — maintainer and agent notes

## Shipping changes to end users

**Merging to `main` does not update Homebrew or GitHub Releases.** Install docs point users at those channels; they advance only when a **`v*`** tag is pushed on the **upstream** repo (`accidentally-awesome-labs/runnerkit`), which triggers `.github/workflows/release.yml` (GoReleaser).

**Rough sequence**

1. Land work on `main` (PR merge or direct push).
2. Run the **pre-tag checklist** in [`docs/release-process.md`](docs/release-process.md) (CI green, smoke/stopwatch expectations as applicable).
3. Choose the next **SemVer** tag (`v1.0.x` patch for fixes/small additive CLI; bump minor/major when warranted).
4. Create an **annotated** tag and push **only the tag** (or push tag after verifying commit):

   ```bash
   git fetch origin && git checkout main && git pull origin main
   git tag -a vX.Y.Z -m "RunnerKit vX.Y.Z — short summary"
   git push origin vX.Y.Z
   ```

5. Confirm in GitHub **Actions** that the release workflow succeeded.
6. Confirm **GitHub Releases** has assets for `vX.Y.Z` and **`accidentally-awesome-labs/homebrew-tap`** received the cask bump (GoReleaser commit, e.g. `runnerkit: bump cask to vX.Y.Z`).

**Fork caveat:** Tag pushes from forks do not run upstream releases and may break OIDC signing — always release from the upstream repository.

Full prerequisites (Homebrew PAT, optional Apple notarization), failure modes, and verification commands: **`docs/release-process.md`**.

**Live smoke (`make smoke-live`, D-11):** After interactive `runnerkit doctor`, BYO and cloud scripts run **`scripts/smoke/assert-doctor-json-contract.sh`** to assert **`doctor --json`** includes **`schema_version`**, **`stage`**, **`host_incident_hints`** and **`next_actions`** as JSON arrays (never `null`) and **`doctor --deep --json`** exits 0. They also run **`scripts/smoke/assert-list-json-contract.sh`** on **`list --json`** (SEED-002). Requires **`python3`**. Override **`RUNNERKIT_SMOKE_SKIP_DOCTOR_DEEP=1`** to skip the deep pass.

**BYO multi-repo smoke (optional):** Set **`RUNNERKIT_SMOKE_MULTI_REPO=1`** and **`RUNNERKIT_SMOKE_REPO2=owner/other`** (second trusted private repo, different from **`RUNNERKIT_SMOKE_REPO`**) before **`make smoke-live-byo`** / **`make smoke-live`**. The BYO script then **`register`**s the second repo on the same host, asserts two repos via **`scripts/smoke/assert-list-host-repo-count.sh`**, runs the doctor JSON contract for repo2, then **`down`** repo2 then the primary.

## Hetzner cloud provisioning (cloud-init v3)

When RunnerKit creates the VM (`runnerkit up --repo … --cloud hetzner`), **user-data** applies the same **scoped** `/etc/sudoers.d/runnerkit-installer` rules as `install.sh` / `byo-prepare` (`internal/bootstrap/sudoers.go`), validated with **`visudo`** before SSH bootstrap runs — so non-interactive `sudo apt-get` / install steps do not depend on a fragile `users[].sudo` stanza alone. Readiness uses **`cloud-init status --wait`** and rejects **`status: error`** (older builds incorrectly treated some error states as ready because **`boot-finished`** existed). **`waitCloudTargetReady`** runs preflight with **`RequirePasswordlessSudo`** so missing NOPASSWD surfaces as **`host.privilege.cloud_bootstrap`** before bootstrap. Inventory records **`runnerkit-cloud-init-v3`** (constant **`hetzner.CloudInitUserDataVersion`**); the host also writes **`/var/lib/runnerkit/cloud-init.json`**. Generic **`--host`** machines are unchanged: they still need the one-time host install when sudo is password-protected.

## Extra packages (`--extra-packages` + auto-detection)

CI workflows often need OS-level dependencies (native libraries, GUI test infrastructure, build tools) that the base Ubuntu image does not include. RunnerKit resolves extra packages from three sources (merged, deduplicated, CLI wins):

1. **Auto-detection** — `scanWorkflowExtraPackages` (`internal/cli/workflow_packages.go`) scans `.github/workflows/*.{yml,yaml}` in CWD for `apt-get install` / `apt install` commands (handles `sudo`, flags, backslash continuations) and extracts package names. Runs automatically during `runnerkit up` for both BYO and cloud paths.
2. **`--extra-packages "pkg1,pkg2"`** — explicit CLI flag, merged on top of auto-detected.
3. **`.runnerkit/config.yaml` `defaults.extra_packages`** — project-level defaults (plumbing ready, YAML loader not yet integrated).

**Baseline packages** (`bootstrap.BaselinePackages`): ~75 apt packages matching the GitHub-hosted Ubuntu 24.04 runner image are always installed on Ubuntu/Debian hosts — `build-essential`, `pkg-config`, `gcc`, `g++`, `make`, `curl`, `jq`, `unzip`, and many more. Without them, compiled-language CI fails with "linker cc not found" or missing pkg-config probes. On cloud-provisioned hosts, baseline packages are installed via cloud-init; the `fix_dependencies` bootstrap step skips them (deduplication via `CloudProvisioned` flag in `bootstrap.Options`).

**Runner image setup** (`bootstrap.RenderImageSetupScript`): After `fix_dependencies`, an Ubuntu/Debian-gated `setup_runner_image` bootstrap step installs language runtimes (Node.js 20 LTS, Python pip/venv, Go, Rust, Java 17, .NET 8), Docker CE, browser testing tools (Chrome, ChromeDriver, Firefox, Geckodriver), and CLI tools (gh, cmake, ninja, zstd). The script is idempotent (each tool section checks `command -v`) and writes a marker at `/var/lib/runnerkit/image-setup.json` with `ImageSetupVersion`. The `RepositoryState.ImageSetupVersion` field tracks this for `upgrade-runner` re-run decisions. OS gating: `isUbuntuLike(opts.OSReleaseID)` — non-Ubuntu BYO hosts skip with a log message and receive only `fix_dependencies`.

Cloud path: baseline + extra packages injected into cloud-init `packages:` (installed at first boot). BYO path: baseline packages installed alongside missing tools during the `fix_dependencies` bootstrap step. Both paths run `setup_runner_image` after `fix_dependencies` on Ubuntu/Debian. Persisted in `RepositoryState.ExtraPackages` so **`runnerkit upgrade-runner`** re-installs them. Package names are validated: only alphanumerics, hyphens, dots, colons, underscores, and `+` are accepted.

Implementation: `internal/cli/workflow_packages.go` (`scanWorkflowExtraPackages`, `extractAptPackages`), `internal/cli/up.go` (`autoDetectExtraPackages`, `resolveExtraPackages`, `parseExtraPackages`), `internal/bootstrap/install.go` (`BaselinePackages`, `mergePackages`, `isUbuntuLike`), `internal/bootstrap/image_setup.go` (`RenderImageSetupScript`, `ImageSetupVersion`), `internal/provider/hetzner/provision.go` (`cloudInitUserData`), `internal/bootstrap/sudoers.go` (expanded allowlist for `tee`, `gpg`, `mkdir`, `unzip`, `usermod`, `dpkg`, `add-apt-repository`).

## Multi-repo BYO (SEED-002, v1.2+)

**v1.2 scope:** multi-repo on a **single BYO SSH host** (same `user@host` for each `runnerkit up` / `register`). **Cloud** remains one provisioned server per `runnerkit up --cloud` unless you manually point a second repo at an existing machine’s SSH address. Tarballs cache under **`/opt/actions-runner/runnerkit-shared-bin/<runner-version>/`**. Narrative: [`docs/troubleshooting/multi-repo.md`](docs/troubleshooting/multi-repo.md).

## UX polish layer (SEED-004, v1.1+)

Line-oriented CLI only (no full-screen TUI). **`runnerkit`** with no subcommand runs a **first-run wizard** when there are no saved repos; **`--explain`** / **`--unicode`** are root persistent flags; **`doctor --fix`** / **`--ignore`** persist in **`config.json`**. BYO **`up`**/**`register`** prints **checklists** and saves progress under **`sessions/`** inside the state directory.

Implementation touchpoints: `internal/ui/box.go`, `internal/ui/checklist.go`, `internal/ux/stage/`, `internal/ux/checkliststore/`, `internal/cli/wizard.go`, `internal/cli/byo_checklist.go`, `internal/cli/explain.go`, `internal/cli/doctor_fix.go`, `internal/cli/userconfig.go`; JSON helpers in `internal/ux/nextaction/nextaction.go`.

## Host capacity, OOM, and `runnerkit doctor` (Phase 7)

When users hit **runner offline**, **systemd failed**, or **CI OOM / `ld` signal 9** on small self-hosted VMs:

- **Preflight** reads `MemAvailable` / `SwapFree` from `/proc/meminfo` over SSH. Below **4 GiB** MemAvailable → warning `host.mem_available` (**RKD-BOOT-016**). No swap and MemAvailable **&lt; 8 GiB** → **RKD-BOOT-017**. Warnings do **not** fail `preflight.Passed()`. Override threshold with **`RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES`** (positive integer, bytes).
- **`runnerkit doctor`**: same preflight findings via `host_mem_low` / `host_swap_constrained`. When SSH works and the runner looks **sick** (service failed, GitHub offline/missing runner), or when the user passes **`--deep`**, RunnerKit pulls **bounded** `journalctl` excerpts and runs **heuristic** OOM/SIGKILL detection → finding **`host_incident_hints`** (**RKD-BOOT-018**), JSON field **`host_incident_hints`**. **`--with-log-snippets`** adds short **redacted** matching lines.
- **Narrative doc:** [`docs/troubleshooting/host-resources.md`](docs/troubleshooting/host-resources.md) (index: [`docs/troubleshooting/README.md`](docs/troubleshooting/README.md)).

Implementation touchpoints: `internal/preflight/checks.go`, `internal/remote/system.go` + `meminfo.go`, `internal/ops/hostkillhint.go`, `internal/ops/logs.go` (`CollectBoundedJournalsForHints`), `internal/cli/doctor.go`, `internal/ops/doctor.go`, `internal/errcodes/codes.go`.
