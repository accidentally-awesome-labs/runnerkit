# Research: real-world GitHub Actions self-hosted runner installers (GitHub search)

## Methodology

- **Date:** 2026-05-18
- **Tools used:** WebFetch on github.com / API URLs (gh CLI Bash access was blocked), supplemented by WebSearch for surfacing high-signal links (issues, deprecation notices). Raw `raw.githubusercontent.com` URLs were blocked, but `github.com/.../blob/...` rendered files successfully and the official script's content was also reconstructable via deprecation issue #3915 which quotes it verbatim.
- **Search angles run:**
  - `topic:github-actions-runner stars:>10` (repo discovery)
  - `actions-runner-installer`, `self-hosted-runner-install` (code search behind auth wall; used WebSearch)
  - `"create-latest-svc.sh" "RUNNER_CFG_PAT"` (official script lineage + open issues)
  - `installdependencies.sh actions/runner` (built-in deps script)
  - `self-hosted runner "Ubuntu 24" OR "noble"` (failure modes)
  - Each candidate repo's `issues?q=is:issue+...` page for sudo/permission/OS failures.
- **Scope:** GitHub Actions-specific installers (not Buildkite/GitLab — covered in parallel research). Skipped repos with last commit > 2 yr.

## Top installers found

### actions/runner (official) — scripts/create-latest-svc.sh
- **What it does:** Canonical one-shot bash installer — downloads latest runner tarball, exchanges a PAT for a registration token, runs `config.sh`, installs+starts systemd service.
- **Install command (verbatim):**
  ```
  export RUNNER_CFG_PAT=ghp_xxx
  curl -s https://raw.githubusercontent.com/actions/runner/main/scripts/create-latest-svc.sh | bash -s -- -s myorg/myrepo -n myname -l label1,label2
  ```
- **Install architecture:** detect platform/arch → fetch latest version tag from GitHub releases API → curl tarball (skip if file exists) → `mkdir runner && tar xzf` (refuse if `runner/` exists, `-f` to force replace) → POST `/repos/<o>/<r>/actions/runners/registration-token` with PAT → `./config.sh --unattended --replace` → `${prefix}./svc.sh install ${svc_user}` and `${prefix}./svc.sh start`.
- **Privilege model:** `prefix="sudo "` set only when `runner_plat == linux`; `sudo -u ${svc_user}` to drop privs for `config.sh`; `sudo -E` to preserve env. **No sudoers pre-flight** — assumes interactive password or NOPASSWD already present.
- **Notable details / clever choices:**
  - Single source of truth for "what's latest" via the GitHub releases API (no hardcoded versions).
  - Tarball download skip-if-exists for cheap idempotency on re-run.
  - Refuses to clobber `./runner` unless `-f` — explicit foot-gun guard.
  - Service-user-aware: `sudo -u $svc_user` so `.runner`/`.credentials` end up owned correctly.
- **Notable failure modes / open issues:**
  - **Issue #3915 (open):** registration-token API call uses deprecated `application/vnd.github.everest-preview+json` + `Authorization: token`. New API requires `application/vnd.github+json`, `Bearer`, and `X-GitHub-Api-Version: 2022-11-28`. Script in `main` still on old headers.
  - **Issue #3160 (closed not-planned):** script starts with `#/bin/bash` instead of `#!/bin/bash` — works via `bash script.sh` or `curl | bash` but not `chmod +x` execution under dash.
  - `docs/automate.md` flags only one caveat ("no Docker"); no checksum, no rollback, no prereq doc.
  - No checksum or signature verification of the tarball.
- **Lessons for RunnerKit:**
  - The "blessed" path is `curl | bash` with PAT in env — adopting the same UX doesn't fight muscle memory.
  - Both maintenance issues are still live in 2026; an installer that uses current API headers + verifies the tarball checksum already exceeds the official UX.
  - "Refuse if dir exists, `-f` to force" is small but high-signal.
- **Source:**
  - https://github.com/actions/runner/blob/main/scripts/create-latest-svc.sh
  - https://github.com/actions/runner/blob/main/docs/automate.md
  - https://github.com/actions/runner/issues/3915, https://github.com/actions/runner/issues/3160

### actions/runner — src/Misc/layoutbin/installdependencies.sh
- **What it does:** Bundled dep installer — detects OS family, installs `liblttng-ust*`, `libkrb5-3`, `zlib1g`, `libssl*`, `libicu*` and distro equivalents.
- **Install command:** `sudo ./bin/installdependencies.sh` (run from runner dir after extract).
- **Install architecture:** branches on `/etc/debian_version` (apt-get with apt fallback), `/etc/redhat-release` (dnf vs yum + RHEL 6 special-case), `ID_LIKE=suse` in `/etc/os-release` (zypper).
- **Privilege model:** **explicit `EUID == 0` check** — exits 1 with "Need to run with sudo privilege" if not root. Cleanest privilege precondition in the ecosystem.
- **Notable details / clever choices:**
  - Tries multiple library version names in a single apt-get (`libssl3t64 libssl3 libssl1.1 libssl1.0.2`) so it survives Ubuntu LTS jumps without code change.
  - RHEL 6 only installs libs it has and links to docs for the rest.
- **Notable failure modes:**
  - Doesn't install `build-essential`/`gcc`/`pkg-config` — compiled-language CI fails after a "successful" run. Matches the gap RunnerKit's `BaselinePackages` already fills.
  - Pure runtime libs only — no Docker / Node / Python.
  - Ubuntu 24.04 has been recurring pain because the libssl/libicu version probe lags real LTS releases.
- **Lessons for RunnerKit:** EUID-0 check is a one-liner worth stealing for any `runnerkit install`. "Try multiple lib version names" is an explicit policy worth documenting.
- **Source:** https://github.com/actions/runner/blob/main/src/Misc/layoutbin/installdependencies.sh

### myoung34/docker-github-actions-runner (★2.3k, last release 2.334.0 on 2026-04-21)
- **What it does:** Most-downloaded community runner image; registers/deregisters at start/stop using PAT, runner token, or GitHub App.
- **Install architecture:** `entrypoint.sh` → `token.sh` (PAT→reg-token API exchange) → `config.sh --unattended --replace` → exec runner. Deregister on SIGTERM/SIGINT/SIGQUIT.
- **Token flow (verbatim from token.sh):** POST `/{orgs|repos|enterprises}/{scope}/actions/runners/registration-token` with `Authorization: token ${ACCESS_TOKEN}`, returns JSON `{token, full_url}`. **Same deprecated header set as official `create-latest-svc.sh`** — ecosystem-wide lag.
- **Privilege model:** dedicated `runner` user pre-created in image; runs unprivileged unless `RUN_AS_ROOT=true`; uses `gosu`; sudoers preserve proxy env, NOPASSWD for sudo group.
- **Notable details / clever choices:**
  - Three orthogonal auth modes: PAT, ephemeral runner-token, GitHub App (`APP_ID + APP_PRIVATE_KEY`). App auth refreshes token before deregister so cleanup works on long-lived runners.
  - `CONFIGURED_ACTIONS_RUNNER_FILES_DIR` persists `.runner`/`.credentials` so restart doesn't re-register.
  - `UNSET_CONFIG_VARS=true` scrubs the PAT from env after registration (defense-in-depth).
- **Notable failure modes / open issues:**
  - **#527 (closed):** `lsb-release` removed in 2.329.0 broke `actions/setup-python@v5`. Workaround: pin to 2.328.0. Cautionary tale against trimming "unnecessary" packages.
  - **#225 (closed):** "Sudo No Longer working" after non-root migration — classic.
  - **#587 (closed 2026-05-12):** "How do I use a specific GitHub base image like `ubuntu-24`?" — explicit demand for 24.04 parity.
- **Lessons for RunnerKit:** Three-auth-mode UX is what advanced users expect (currently mostly PAT). Scrubbing PAT after registration is a small win. The `lsb-release` incident validates "match the GitHub-hosted runner image" — which `BaselinePackages` already does.
- **Source:**
  - https://github.com/myoung34/docker-github-actions-runner
  - https://github.com/myoung34/docker-github-actions-runner/blob/master/entrypoint.sh
  - https://github.com/myoung34/docker-github-actions-runner/blob/master/token.sh

### MonolithProjects/ansible-github_actions_runner (★242, updated 2026-04-08)
- **What it does:** Ansible role to deploy/redeploy/uninstall the runner on Linux & macOS; repo-, org-, and enterprise-scoped.
- **Install architecture:** `tasks/install.yml` → pre-create dedicated OS user → download official tarball → extract to `/opt/actions-runner` (Linux) or `C:\actions-runner` → register with PAT (`PERSONAL_ACCESS_TOKEN` env) → install systemd service.
- **Privilege model:** Ansible `become: yes` — privilege handling delegated to Ansible inventory/SSH config, not embedded.
- **Notable details:**
  - `runner_state` accepts `started | stopped | absent` — full lifecycle via one var, including deregistration.
  - `reinstall_runner: true` triggers full teardown without manual cleanup.
  - `no_log: true` on every PAT-touching task prevents leak into Ansible logs.
  - PAT only via env var, never inlined.
- **Lessons for RunnerKit:** `state: present/absent/restarted` semantics generalize to a CLI verb set. "Never log the PAT" is a one-line redact discipline worth auditing end-to-end.
- **Source:**
  - https://github.com/MonolithProjects/ansible-github_actions_runner
  - https://github.com/MonolithProjects/ansible-github_actions_runner/blob/master/tasks/install.yml

### github-aws-runners/terraform-aws-github-runner (★3.1k, v7.6.0 on 2026-04-01)
- **What it does:** Largest community Terraform module for auto-scaling ephemeral EC2 runners (formerly philips-labs).
- **Install architecture:** Webhook → Lambda (scale-up) → EC2 RunInstance with user-data → either (a) download+configure from scratch or (b) use a pre-baked AMI (recommended). Lambda also handles scale-down + spot termination.
- **Privilege model:** Cloud-only; user-data runs as root via cloud-init; **GitHub App credentials in SSM Parameter Store** instead of PATs.
- **Notable details:**
  - Steers users toward pre-baked AMI rather than first-boot install — faster cold start, less can fail at boot.
  - Ephemeral by default: one job, one instance, terminate. Eliminates a whole class of state-drift bugs.
- **Lessons for RunnerKit:** Pre-baked image vs first-boot install is a real architectural choice — RunnerKit's cloud path could expose a "snapshot this VM as a template" mode. GitHub App > PAT for any fleet >1 host (roadmap item for SEED-001+).
- **Source:** https://github.com/github-aws-runners/terraform-aws-github-runner

### machulav/ec2-github-runner (★849, updated 2026-05-08)
- **What it does:** GitHub Action that spins up EC2, registers a runner, runs your job, terminates.
- **Install architecture:** user-data uses `#cloud-boothook` (init stage, not final modules) — deliberately works around AMIs with broken `cloud_final_modules`. Detects arch, downloads runner, runs `config.sh` (or JIT for single-use), then `run.sh` or systemd.
- **Notable details / clever choices:**
  - `#cloud-boothook` is a directly transferable trick — RunnerKit's Hetzner cloud-init payload could be hardened the same way.
  - Supports `pre-runner-script` (arbitrary bash) and `packages` JSON array (passed to yum/apt-get) — two-knob extra-deps UX similar to RunnerKit's `--extra-packages`.
  - Polls EC2 serial console in debug mode to surface cloud-init failures.
- **Source:** https://github.com/machulav/ec2-github-runner

### cloudbase/garm (★326, updated 2026-05-18; Go — closest peer to RunnerKit)
- **What it does:** Controller daemon that creates/destroys runners across multiple clouds via pluggable provider model. Supports GitHub Actions and Gitea.
- **Install architecture:** Daemon + provider plugins (each implements a small interface). Runs as systemd service or Docker container.
- **Notable:** Pluggable provider model in Go is the strongest existing pattern for what RunnerKit's SEED-001 architecture rewrite is targeting. Worth a direct read before that work starts.
- **Source:** https://github.com/cloudbase/garm

### whywaita/myshoes (★169, updated 2026-03-04)
- Smaller-scope version of GARM (webhook → provider → ephemeral VM). Confirms "controller + provider plugin" is the convergent design at scale.
- **Source:** https://github.com/whywaita/myshoes

## Cross-installer patterns

1. **PAT in env → registration-token via API → `config.sh --unattended --replace`** is universal across every reviewed installer. RunnerKit already does this — no deviation needed.
2. **Tarball-from-releases, not from a package manager.** No-one packages the runner as deb/rpm; everyone downloads the official tarball with the version pinned to a string fetched from the GitHub releases API.
3. **`config.sh --replace` is the only idempotency lever for re-registration.** Combined with deleting `.runner`/`.credentials` on uninstall, this is the entire state-cleanup story.
4. **Dependency installation is split from runner installation.** Official `installdependencies.sh` is runtime libs only; Docker images bake deps into base; Ansible role only installs runner deps. **No installer in this set automatically installs the GitHub-hosted-runner package surface** — exactly the gap RunnerKit's `BaselinePackages` (~75 packages) already fills.
5. **Systemd via `svc.sh`** is the only Linux service story; nobody hand-writes unit files. `svc.sh install <user>` + `svc.sh start` closes every Linux installer.
6. **Auth modernization is real:** GARM, terraform-aws-github-runner, and myoung34 all support GitHub App auth. PAT-only is increasingly entry-level.
7. **Privilege model is hand-waved.** Only `installdependencies.sh` does an explicit EUID-0 check. Everyone else assumes "you ran with sudo or you're already root" — directly explains the volume of sudo/permission issues across all installers.

## Recurring failure modes from issue trackers

- **Ubuntu 24.04 (Noble) lag** — multiple reports through late 2025/early 2026; `setup-ruby` failures, MS Edge install breakage, mismatched `ImageOS` env. Sources: https://github.com/orgs/community/discussions/173414 (runner offline on 24.04 in Docker), https://github.com/actions/runner-images/issues/12626 (install-microsoft-edge.sh failing), https://github.com/orgs/community/discussions/160468 (24.04 docker image discussion).
- **Missing `lsb_release` after image trim** — https://github.com/myoung34/docker-github-actions-runner/issues/527 — broke `actions/setup-python@v5`.
- **Deprecated registration-token API headers in official script** — https://github.com/actions/runner/issues/3915 still open. Anyone copy-pasting `create-latest-svc.sh` inherits the debt.
- **Service install fails on RHEL 9.2 (V8 memory protection)** — https://github.com/actions/runner/issues/3222 — runner works under `run.sh` but `svc.sh install` triggers Node 16 V8 crash. Closed not-planned.
- **Shebang missing on official scripts** — https://github.com/actions/runner/issues/3160.
- **Sudo regressions on non-root container user migrations** — https://github.com/myoung34/docker-github-actions-runner/issues/225.

## Concrete patterns RunnerKit should adopt

1. **EUID-0 (or NOPASSWD-sudo) precondition check at script start**, exit non-zero with actionable message — copied directly from `installdependencies.sh`. Matches RunnerKit's existing `RequirePasswordlessSudo` preflight; if RunnerKit ships an `install.sh`, this must be the first executable line. Source: https://github.com/actions/runner/blob/main/src/Misc/layoutbin/installdependencies.sh
2. **Refuse-if-exists with explicit `-f` flag** for the runner directory, mirroring `create-latest-svc.sh`. Prevents the silent "second runner overwrote first" footgun. Source: https://github.com/actions/runner/blob/main/scripts/create-latest-svc.sh
3. **Try-multiple-package-versions in a single apt-get line** for libssl/libicu — survives Ubuntu LTS bumps. Audit `BaselinePackages` for any pinned `libssl3` (should be `libssl3 libssl3t64`). Source: same `installdependencies.sh`.
4. **Use current registration-token API headers** (`Authorization: Bearer`, `Accept: application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28`). Official script is still on deprecated set per #3915. Verify `internal/provider/github/` already uses current set. Source: https://github.com/actions/runner/issues/3915
5. **`UNSET_CONFIG_VARS`-style PAT scrubbing** after registration succeeds — myoung34's pattern, applied to bash env on the bootstrap host so PAT isn't sitting in process env or shell history. Source: https://github.com/myoung34/docker-github-actions-runner/blob/master/entrypoint.sh
6. **`#cloud-boothook` for cloud-init payloads** instead of default modules-final stage — protects against images with truncated `cloud_final_modules`. Apply to RunnerKit's Hetzner cloud-init v3. Source: https://github.com/machulav/ec2-github-runner
7. **GitHub App auth as a roadmap item** — PAT-only is entry-level in 2026; the three highest-star community projects all support GitHub App. Aligns with SEED-001+ multi-host trajectory.
8. **`runner_state: started|stopped|absent`-style verb set** generalized to the CLI — Ansible role's lifecycle vocabulary clarifies deregistration semantics. Source: https://github.com/MonolithProjects/ansible-github_actions_runner
9. **No-log discipline on PAT-touching code paths** — `no_log: true` equivalent; audit RunnerKit's `redact` patterns end-to-end against any code that constructs Authorization headers.
10. **Treat `installdependencies.sh` as the floor, not the ceiling.** Every issue surveyed agrees: official deps cover only the runner binary itself. `BaselinePackages` (~75 packages, GitHub-hosted runner image parity) is the right model — install at least what github-hosted runners ship with, gated by Ubuntu/Debian detection.
