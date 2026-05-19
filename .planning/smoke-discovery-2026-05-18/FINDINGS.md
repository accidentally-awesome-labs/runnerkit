---
discovery_date: 2026-05-18
runnerkit_version_under_test: HEAD (v1.3.2 + 1 docs-only commit, equivalent to v1.3.2 code)
binary_resolution: scripts use `go run ./cmd/runnerkit` (built from source HEAD), not installed binary
smoke_repo: accidentally-awesome-labs/dat0
byo_host: salar@mckee-small-desktop (Ubuntu 24.04.4 LTS, password-protected sudo, Path C scoped sudoers from 2026-05-10)
cloud_provider: Hetzner fsn1 cpx22 ubuntu-24.04
---

# RunnerKit v1.3.2 — Live Smoke Discovery (2026-05-18)

Triggered by maintainer pivoting away from speculative milestone scoping (architecture cleanup + stabilize): *"actually let's properly test the BYO and cloud features of runnerkit to make sure what work actually needs to be done or if it's functioning properly as it is."*

## Verdict

| Path | Result | Wall-clock |
|---|---|---|
| **Cloud (Hetzner)** | ✅ GREEN end-to-end | 498s (~8.3min) including 300s destroy-verify poll |
| **BYO (Path C, scoped sudoers)** | ❌ RED at `setup_runner_image` step | Failed in ~45s |
| **Destroy on long-running runner** | ✅ GREEN | 8 artifacts removed cleanly via `auto_delete` cascade |

## Bugs found

### Bug A — Scoped sudoers allowlist missing FOUR commands (P0, expanded after static analysis)

**Location:** `internal/bootstrap/sudoers.go::RenderSudoersEntry`

| Missing command | Where used | Smoke surface |
|---|---|---|
| `sudo ln` | `internal/bootstrap/image_setup.go:64, 65, 121` — Go binary + chromedriver symlinks | **Confirmed RED in BYO smoke** at `setup_runner_image` |
| `sudo chmod` | `internal/bootstrap/image_setup.go:143` — `chmod +x /usr/local/bin/geckodriver` | Latent — would fail the same way if smoke reached it (it doesn't because `ln` fails first) |
| `sudo cp` | `internal/bootstrap/script.go:258, 259` — ephemeral runner log preservation during `down`/`destroy` | Latent for **ephemeral mode** (today's smoke uses persistent) |
| `sudo cat` | `internal/bootstrap/sudoers.go:120` — `RemoteSudoersReadScript` reads installed fragment for verification | Latent — runs post-install during readback |

**Direct evidence (`ln`):** `ssh salar@mckee-small-desktop 'sudo -n ln -sf /tmp/a /tmp/b'` → `password required` exit 1. Compare `sudo -n apt-get -h` → exit 0 (allowlisted) and `sudo -n install --version` → exit 0 (preflight probe passes).

**Static analysis** (`/tmp/sudo-static-check.sh` — cross-references all `\bsudo <cmd>` invocations in `internal/bootstrap/*.go` against `RenderSudoersEntry`):
- Allowlist (basenames): `add-apt-repository apt-get chown curl dnf dpkg gpg install mkdir rm sha256sum su systemctl tar tee unzip useradd usermod yum`
- Missing (filtering comment-text noise): `ln`, `chmod`, `cp`, `cat`
- Sudo commands inside `RemoteVisudoCheckScript` (`mktemp`, `mv`, `visudo`, `tee`, `chmod`) run *during* `byo-prepare` with Path-B password and so don't need allowlist coverage.

**Smoke surface (today):** `make smoke-live-byo` fails at "Remote bootstrap install" step with `Remote stderr: ... sudo: a terminal is required to read the password`. The error doesn't name the command — it's `ln` inside `setup_runner_image` after `fix_dependencies`.

**Impact:**
- **Every Ubuntu/Debian BYO host using Path C scoped sudoers cannot complete `runnerkit up`** since `setup_runner_image` is gated only by `isUbuntuLike(opts.OSReleaseID)`.
- Ephemeral-mode `down`/`destroy` log preservation latently broken (`sudo cp`).
- Path B (interactive password) works but requires a TTY — non-TTY automation can't proceed.

**Fix sketch (immediate v1.3.3 patch):**
```go
// internal/bootstrap/sudoers.go::RenderSudoersEntry
return fmt.Sprintf(`# /etc/sudoers.d/runnerkit-installer (managed by runnerkit install.sh)
%s ALL=(root) NOPASSWD: \
  ... existing entries ...
  /bin/ln, /usr/bin/ln, \
  /bin/chmod, /usr/bin/chmod, \
  /bin/cp, /usr/bin/cp, \
  /bin/cat, /usr/bin/cat, \
  ... existing entries ...
`, user)
```

**Security caveat:** unconditional `sudo cp` and `sudo cat` are latent privilege escalation vectors (read/write any file as root). Naively widening the allowlist trades reliability for attack surface. A safer expansion uses path-scoped wildcards similar to the existing `/opt/actions-runner/runnerkit-*/svc.sh` pattern:
```
  /usr/local/bin/ln, /usr/local/bin/chmod, \
  /var/lib/runnerkit/ephemeral/logs/*, \           // for sudo cp targets
  /etc/sudoers.d/runnerkit-installer, \            // for sudo cat readback
```
This requires deeper review and may be the right v1.4.0 work, not v1.3.3 quick-fix.

**SEED-001 validation:** The allowlist grows every time bootstrap gains a step. This is the architectural problem SEED-001 calls out — the scoped-sudoers approach doesn't scale. Either we keep expanding the allowlist (and the attack surface), or we move the privileged setup to `install.sh` running once on the host with a single sudo prompt.

### Bug B — Stale on-disk sudoers fragment not auto-refreshed (P1)

**Location:** no mechanism. `runnerkit byo-prepare` writes the fragment once; `runnerkit upgrade-runner` re-bootstraps the runner but does NOT re-install the sudoers fragment.

**Direct evidence:** host fragment at `/etc/sudoers.d/runnerkit-installer` is dated `2026-05-10 17:36:11`. Current source `RenderSudoersEntry` includes `mkdir` entries, but `ssh $HOST 'sudo -n mkdir /tmp/x'` → `password required`. So the on-disk fragment predates the `mkdir` expansion (CLAUDE.md: *"expanded allowlist for `tee`, `gpg`, `mkdir`, `unzip`, `usermod`, `dpkg`, `add-apt-repository`"*).

**Impact:** Silently bricks BYO after `runnerkit upgrade` whenever `RenderSudoersEntry` gains new commands. The user sees "sudo password required" with no hint that their sudoers fragment is stale.

**Fix sketch:**
- `upgrade-runner` (or new `byo-prepare --refresh`) hashes current `RenderSudoersEntry` content and compares to on-disk; if different, re-runs the visudo-validated install.
- Add a new `RKD-BOOT-NNN` doctor finding for "sudoers fragment is stale: expected N commands allowlisted, found M" so users can self-diagnose.

### Bug C — `doctor --json` no-state error envelope violates JSON contract (P2)

**Location:** error path in `cmd/runnerkit/doctor.go` (or equivalent) when no `--repo` is provided and inferred repo has no state.

**Direct evidence:**
```bash
$ runnerkit doctor --json
{
    "error": {
        "code": "state_not_found",
        "message": "...",
        "remediation": [...]
    },
    "ok": false,
    "redactions_applied": true
}
```

Missing per CLAUDE.md contract: `schema_version`, `stage`, `next_actions[]` (array, never null), `host_incident_hints[]` (array, never null).

Compare happy path `doctor --repo X --json` which DOES include all four.

**Smoke surface:** `scripts/smoke/assert-doctor-json-contract.sh` only asserts the happy path, so this contract violation is invisible to the existing smoke.

**Impact:** agent / MCP callers reading the JSON envelope will fail to parse `next_actions` (gets undefined / null where it expects an array). Bad UX for SEED-003 plugin work.

**Fix sketch:**
- Error envelope should always emit `schema_version: 1, stage: <inferred or "no_local_state">, next_actions: [], host_incident_hints: []` alongside `error` and `ok: false`.
- Extend `assert-doctor-json-contract.sh` with an error-path case (run against a non-existent repo, assert contract holds).

## Positive evidence (don't rebuild what works)

- **Cloud-init v3** — `hetzner.CloudInitUserDataVersion = runnerkit-cloud-init-v3`, scoped sudoers up front + `cloud-init status --wait` rejecting `status: error`. Works.
- **Plan 06-13 preflight probe** — `sudo -n install --version` correctly distinguishes prepared vs unprepared Path C hosts. Probe passed during the failed BYO smoke, confirming the issue is downstream of preflight.
- **Plans 06-14..16** — they did what they claimed (sudo-prefixed `download_runner`, redacted remote stderr, cloud readiness retry, SSH auth convergence). They just didn't anticipate `setup_runner_image` needing `ln`.
- **Destroy lifecycle on long-running runner** — first time the smoke exercised destroy on a multi-day-old VM. 8 artifacts cleanly removed, `auto_delete` cascade for primary IPs per Plan 06-11 Bug 26.
- **JSON contracts (happy path)** — `doctor --json` baseline + `--deep --json` + `list --json` all pass `python3` assertions.
- **GitHub issue tracker** — zero open issues. No external triage backlog.

## SEED-001 reassessment

The original SEED-001 framing ("conflate one-time install with repeated lifecycle ops, force everything through Path B/C") is still valid as a *direction*, but the immediate pain is smaller and patchable:

- Bug A is a 14-byte sudoers fix → v1.3.3 patch, not a milestone
- Bug C is small and isolated → v1.3.3 patch
- Bug B is a real architectural gap (no upgrade path for sudoers fragment) but can be addressed by `byo-prepare --refresh` rather than a full SEED-001 rewrite

**SEED-001 still has value** for SEED-003 (agent integration / MCP / plugin) because the `next_actions` JSON contract needs to be universal + canonical. But the BYO bootstrap surface is patchable without the full rewrite.

## Recommended next moves

| Priority | Item | Scope |
|---|---|---|
| P0 | Patch Bug A: add `ln` to `RenderSudoersEntry` allowlist | 14-byte change + test. Ship v1.3.3. |
| P0 | Patch Bug C: error-envelope JSON contract | Add 4 contract fields to error path + extend `assert-doctor-json-contract.sh`. Ship in v1.3.3. |
| P1 | Implement `byo-prepare --refresh` (Bug B mitigation) | New flag; compares hash, re-installs with visudo. Could ship v1.3.3 or v1.4.0. |
| P2 | Add Docker-based BYO smoke matrix (Ubuntu 24, Debian 12, Fedora 40) | Prevents Bug A re-regression. v1.4.0 milestone material. |
| P2 | Static analysis: grep all `sudo ` invocations in `internal/bootstrap/*.go` vs `RenderSudoersEntry` allowlist | Surfaces remaining gaps before they cause runtime failures. CI lint candidate. |
| Defer | SEED-001 full split | Still candidate for v1.5+ alongside SEED-003 agent integration. Not blocking. |

## Side-effect inventory

- **Destroyed:** `accidentally-awesome-labs/dat0` production Hetzner runner (server 130878819, ssh-key 112260615, firewall 10971942, primary-ip 130708654/655). Confirmed deliberate by maintainer at the time of consent.
- **Cleaned:** smoke-created `dat0` cloud runner (server 131688953, etc.) — provisioned + destroyed within the cloud smoke run.
- **Untouched:** `salar@mckee-small-desktop` retains pre-existing `/etc/sudoers.d/runnerkit-installer` + `/etc/sudoers.d/runnerkit-runner-ci` and `/opt/actions-runner/runnerkit-shared-bin/` (SEED-002 cache). The failed BYO smoke did not progress past `setup_runner_image`, so no new per-repo directories on the host.

## Artifacts captured

- `01-destroy-orphan-human.log` — runnerkit destroy on the originally-misnamed-as-orphan dat0 production runner
- `02-smoke-byo.log` — failed BYO smoke (Bug A surface)
- `03-smoke-cloud.log` — full green cloud smoke (498s)
