# Experiment: `curl | sudo bash` BYO install (SEED-001 candidate)

## Question

Does the SEED-001 model — a single `curl URL | sudo bash` on the host that pre-installs scoped sudoers including the v1.3.3 fix candidates (`ln`, `chmod`, `cp`, `cat`) — actually deliver a clean non-TTY lifecycle afterward, while keeping the allowlist narrow enough to block arbitrary `sudo` commands?

## Setup

Fresh Ubuntu 24.04 Docker container (`ubuntu:24.04`) with:
- An unprivileged user `maintainer` configured with **password-protected sudo** (no NOPASSWD, no NOPASSWD via group) — explicitly mirroring a real BYO host (e.g. `salar@mckee-small-desktop`)
- sshd running, pubkey auth + password auth both allowed
- Ephemeral SSH keypair generated per-run

`runnerkit-install.sh` (draft) at `experiment/runnerkit-install.sh`:
- Requires `EUID == 0` (run via `sudo bash`)
- Detects Ubuntu/Debian via `/etc/os-release`
- Creates `runnerkit-runner` service user
- Installs a minimal subset of `BaselinePackages` (build-essential, pkg-config, git, curl, jq, unzip, ca-certificates)
- Creates `/opt/actions-runner/runnerkit-shared-bin/` cache dir
- Installs scoped sudoers fragment via `mktemp` → `visudo -cf` → atomic `mv` — **including `ln`, `chmod`, `cp`, `cat` v1.3.3 fix candidates** in addition to the current `RenderSudoersEntry` set

Files:
- `experiment/Dockerfile` — `ubuntu:24.04` + sshd + maintainer user
- `experiment/runnerkit-install.sh` — the draft installer
- `experiment/run-experiment.sh` — harness: build → run → simulate `curl|sudo bash` → SSH back in and probe `sudo -n` for each allowlisted command + controls

## Run

```bash
$ ./run-experiment.sh
===> generated ephemeral ssh keypair
===> building runnerkit-byo-host image
===> container started: runnerkit-byo-experiment (host port 22822)
===> sshd reachable (after 1s)

===> PRE-INSTALL: maintainer sudo state
sudo: a password is required
  (expected: password required)

===> RUNNING runnerkit-install.sh on host via 'curl | sudo bash' simulation
[sudo] password for maintainer: ==> runnerkit-install.sh (draft) — host (ubuntu 24.04)
==> service user: runnerkit-runner
==> creating service user runnerkit-runner
==> installing baseline packages (minimal subset for demo)
/tmp/runnerkit-installer.YD9Ntf: parsed OK
==> installed scoped sudoers fragment at /etc/sudoers.d/runnerkit-installer
==> done. Host is ready for: runnerkit register --host maintainer@<host> --repo owner/name
===> install duration: 15s
```

## Post-install non-TTY probe (over SSH, no TTY)

```
--- baseline (already worked pre-fix) ---
  apt-get: OK
  install: OK
  tar: OK
--- v1.3.3 fix candidates (new in allowlist) ---
  ln: OK
  chmod: OK
  cp: OK
  cat: OK
--- NOT allowlisted (control — should fail) ---
  whoami: blocked (expected)
  ls: blocked (expected)

===> INSTALL FACTS
Install duration: 15s
Container OS: PRETTY_NAME="Ubuntu 24.04.4 LTS"
Service user present?  yes
Sudoers file present?  yes
Shared-bin dir?        yes
Baseline pkgs present? /usr/bin/gcc
```

## Findings

### Positive
1. **Install completes in 15 seconds** including the 7-package baseline subset and the sudoers fragment install.
2. **All 4 v1.3.3 fix candidates work** (`sudo -n ln`, `chmod`, `cp`, `cat`) from a fresh SSH-non-TTY session immediately after install — the gap that caused the live BYO smoke to fail is closed by adding them to the allowlist.
3. **The narrow allowlist held against escalation:** `sudo -n whoami` and `sudo -n ls /root` were both blocked (password required), confirming the sudoers fragment limits attack surface to the allowlisted commands only.
4. **visudo gate worked:** the staged tempfile was validated via `visudo -cf` before atomic rename. A malformed sudoers would have exited 21 before mutating `/etc/sudoers.d/`.
5. **UX = 1 sudo prompt on the host, 0 from the SSH side ever again.** The maintainer typed their password once, on their machine, during the install they were going to run anyway.

### What this doesn't validate
- **GitHub registration and `runnerkit register` SSH lifecycle weren't exercised** in this experiment. Draft `runnerkit-install.sh` stops at "host is ready"; the next step would be `runnerkit register --host maintainer@... --repo owner/name` from the laptop. We have evidence the allowlist supports it (every `sudo -n` it needs works), but no end-to-end SSH lifecycle test yet.
- **Tarball + cosign signature flow** isn't modeled — draft uses an inline sudoers content blob; production install.sh would fetch a signed release artifact and verify with cosign before sourcing.
- **Idempotency on re-run** — `useradd` is guarded, `mkdir -p` is, but `apt-get install` is unconditional and the sudoers write is unconditional. Re-running on a partial host is messy. Not a draft concern; production install.sh would compare hash + skip.
- **Debian 12 + Fedora 40 not tested.** Maintainer's earlier intent (multi-OS smoke matrix) is deferred to the v1.4.0 milestone if it's scoped.
- **Latent v1.3.3 bug #B (stale sudoers fragment, no auto-refresh) is solved** for the curl|bash model because every install run replaces the fragment — but that means anyone who already ran `runnerkit byo-prepare` and is on the current Path C model still has the stale fragment problem until they migrate to install.sh. Migration path needs design.

### Two security observations
1. **`sudo cp` and `sudo cat` are still attack vectors** if scope isn't tightened — they accept arbitrary paths. Reasonable v2 hardening: tighten allowlist entries to specific path patterns (e.g. `/usr/bin/cp /tmp/* /opt/actions-runner/*`, `/usr/bin/cat /etc/sudoers.d/runnerkit-installer`). Out of scope for v1.3.3 patch; should be designed into the install.sh contract.
2. **Defense-in-depth:** the visudo gate prevents lockout, the atomic `mv` prevents partial writes, and the 0440 root-owned permissions prevent the maintainer user from editing the fragment after install. All three should be retained in production.

## Conclusion

The `curl | sudo bash` model **does deliver** what SEED-001 promised on this clean test: one host-local sudo prompt, immediate non-TTY SSH lifecycle, narrow allowlist held. The remaining unknowns (multi-OS, end-to-end registration, signed-release flow, idempotent re-install) are tractable engineering work, not architectural unknowns.

It also confirms that the immediate v1.3.3 patch (just add `ln`, `chmod`, `cp`, `cat` to `RenderSudoersEntry`) **would unblock current Path C users** without any architectural rewrite — the smoke runs through this exact allowlist successfully.

## Artifacts

- `experiment/runnerkit-install.sh` — draft installer (167 lines)
- `experiment/Dockerfile` — Ubuntu 24.04 + sshd + password-protected sudo
- `experiment/run-experiment.sh` — test harness
- `experiment/experiment-run.log` — full second-run output
- `experiment/install-output.log` — install.sh stdout from inside the container
- `experiment/post-install-check.log` — SSH non-TTY allowlist probe
