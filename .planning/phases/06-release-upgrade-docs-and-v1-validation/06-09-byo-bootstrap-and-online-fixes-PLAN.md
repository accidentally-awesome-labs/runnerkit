---
phase: 06-release-upgrade-docs-and-v1-validation
plan: 09
type: execute
wave: 1
depends_on: [05, 06, 08]
files_modified:
  - internal/ui/cli_prompter.go
  - internal/ui/cli_prompter_test.go
  - cmd/runnerkit/main.go
  - cmd/runnerkit/main_test.go
  - go.mod
  - go.sum
  - internal/bootstrap/sudoers.go
  - internal/bootstrap/sudoers_test.go
  - internal/bootstrap/sudo_rewrite.go
  - internal/bootstrap/sudo_rewrite_test.go
  - internal/bootstrap/install.go
  - internal/bootstrap/install_test.go
  - internal/bootstrap/script.go
  - internal/bootstrap/script_test.go
  - internal/cli/byo_prepare.go
  - internal/cli/up.go
  - internal/cli/runner_online_test.go
  - internal/preflight/checks.go
  - internal/preflight/checks_bugfix_test.go
  - scripts/smoke/byo-permission.sh
autonomous: true
gap_closure: true
requirements: [REL-05]
must_haves:
  truths:
    - "`runnerkit byo-prepare --host user@host` against an Ubuntu 24.04 LTS host with password-protected sudo lands `/etc/sudoers.d/runnerkit-installer` end-to-end after a single sudo password prompt."
    - "`runnerkit up --host user@host --mode persistent` against the same host succeeds end-to-end through preflight, Path B sudo password prompt, bootstrap (download_runner + configure_runner + install_service + verify_service), runner registration, and online-check; lands a GitHub runner ID."
    - "Bootstrap is idempotent against re-runs: stale `.runner` / `.credentials` files (Bug 13) and stale `/etc/systemd/system/actions.runner.<...>.service` units (Bug 14) are pre-cleaned before re-installing."
    - "Pre-bootstrap runner-name conflict check (Bug 17) skips the error when the existing runner has the canonical `runnerkit` label; only foreign-runner collisions still error."
    - "Cloud path is verifiably unchanged: `internal/provider/hetzner/provision.go` is NOT modified; existing hetzner package tests stay green."
    - "Plan 06-07 BYO smoke against `salar@mckee-small-desktop` lands `BYO_DURATION_SECONDS=125` with a successful `.runner` sentinel assertion (smoke harness Bug 18 fix)."
  artifacts:
    - path: "internal/ui/cli_prompter.go"
      provides: "Concrete CLIPrompter (Confirm/Select/Password) wired in cmd/runnerkit/main.go's Dependencies. Closes Bug 4 (Prompts == nil in production binary)."
      contains: "NewCLIPrompter"
      contains_also: "term.ReadPassword"
    - path: "internal/bootstrap/sudoers.go"
      provides: "RemoteVisudoCheckScript creates staging tempfile via `sudo mktemp` to bypass Ubuntu 24.04 fs.protected_regular=2 (Bug 5)."
      contains: "TMP=$(sudo mktemp"
    - path: "internal/bootstrap/sudo_rewrite.go"
      provides: "RewriteSudoForPasswordPipe (word-boundary regex) — preserves `visudo ` and `sudoers` while rewriting standalone `sudo ` to `sudo -S `. Closes Bug 6."
      contains: "RewriteSudoForPasswordPipe"
      contains_also: "regexp.MustCompile"
    - path: "internal/preflight/checks.go"
      provides: "Stderr-based sudo classification (Bug 7) + curl probes drop -f flag for HTTP-rate-limit tolerance (Bug 8)."
      contains: "strings.Contains(probeResult.Stderr"
      contains_also: "curl -sS --connect-timeout"
    - path: "internal/bootstrap/install.go"
      provides: "configure_runner + configure_ephemeral_runner have Sudo: true (Bug 9). wrapSudoCommand uses cred-prime structure (Bug 10). verify_service has cd <installPath> (Bug 15). ServiceNotActiveError carries CommandID + Stderr (Bug 12)."
      contains: "configure_runner"
      contains_also: "sudo -S -v"
      contains_also2: "ServiceNotActiveError"
    - path: "internal/bootstrap/script.go"
      provides: "register_runner -c arg cd's into installPath (Bug 11). Pre-config rm of stale .runner/.credentials (Bug 13). Service script pre-stop+uninstall before install (Bug 14)."
      contains: "cd %[2]s &&"
      contains_also: "sudo rm -f .runner"
      contains_also2: "sudo ./svc.sh stop || true"
    - path: "internal/cli/up.go"
      provides: "ServiceNotActiveError remediation surfaces remote stderr (Bug 12). isRunnerKitManagedRunner skips self-collision in pre-check (Bug 17). runnerOnlineWithLabels case-insensitive (Bug 16)."
      contains: "isRunnerKitManagedRunner"
      contains_also: "strings.ToLower(label)"
    - path: "scripts/smoke/byo-permission.sh"
      provides: "test -f assertions without sudo (Bug 18) — config.sh + .runner are world-readable."
      contains: "ssh \"${HOST}\" 'test -f /opt"
  key_links:
    - from: "Plan 06-07 BYO smoke against salar@mckee-small-desktop"
      to: "BYO_DURATION_SECONDS=NNN with .runner sentinel asserted"
      via: "All 15 in-tree fixes (Bugs 4-17) + smoke harness fix (Bug 18) lined up so attempt-15 cleared all gates"
      pattern: "BYO_DURATION_SECONDS="

tasks:
  - id: bug-4-cli-prompter
    name: "Wire concrete ui.Prompter in main.go (Bug 4)"
    autonomous: true
    commits: [3813a8e, 1d1888e]
  - id: bug-5-sudo-mktemp
    name: "sudo mktemp for fs.protected_regular=2 (Bug 5)"
    autonomous: true
    commits: [cc44067, 62cdd2a]
  - id: bug-6-word-boundary
    name: "Word-boundary sudo rewrite (Bug 6)"
    autonomous: true
    commits: [3c9bf59, b1ce1c1]
  - id: bug-7-8-preflight
    name: "Preflight stderr-based privilege + curl -f drop (Bugs 7+8)"
    autonomous: true
    commits: [e7e2cfb, 9a08b01]
  - id: bug-9-configure-sudo
    name: "configure_runner Sudo: true (Bug 9)"
    autonomous: true
    commits: [f5e35ee, f195a83]
  - id: bug-10-cred-prime
    name: "wrapSudoCommand cred-prime structure (Bug 10)"
    autonomous: true
    commits: [a176a82, 281966d]
  - id: bug-11-su-cwd
    name: "cd inside sudo su -c arg (Bug 11)"
    autonomous: true
    commits: [64b826c, beef841]
  - id: bug-12-service-stderr
    name: "ServiceNotActiveError surfaces stderr (Bug 12)"
    autonomous: true
    commits: [76f0a14, 248c68b]
  - id: bug-13-stale-runner
    name: "Pre-rm .runner/.credentials before config.sh (Bug 13)"
    autonomous: true
    commits: [4d76d14, ae1a702]
  - id: bug-14-systemd-unit
    name: "Pre-stop+uninstall before svc.sh install (Bug 14)"
    autonomous: true
    commits: [da3ad73, 7bc9b25]
  - id: bug-15-verify-cd
    name: "verify_service cd into installPath (Bug 15)"
    autonomous: true
    commits: [ec19486]
  - id: bug-16-label-case
    name: "Case-insensitive label match (Bug 16)"
    autonomous: true
    commits: [91c45ff]
  - id: bug-17-self-collision
    name: "Skip self-collision in runnerNameConflict (Bug 17)"
    autonomous: true
    commits: [485daa3]
  - id: bug-18-smoke-sudo
    name: "Drop sudo from smoke harness test -f (Bug 18)"
    autonomous: true
    commits: [c7ede69]
---

# Plan 06-09: BYO Bootstrap + Online-Check Gap-Closure Fixes

## Context

Plan 06-07 attempt-1 (Bug 3, closed in Plan 06-08) revealed that the
BYO bootstrap path had never been live-validated end-to-end. Each
subsequent re-attempt (2 through 15) surfaced a new layer of bugs as
prior failures unblocked deeper code paths. This plan retroactively
captures the 15 bugs that were closed in sequence to land a working
BYO smoke against `salar@mckee-small-desktop` (Ubuntu 24.04 LTS,
password-protected sudo, residential IP with anonymous GitHub API
rate-limit).

## Bug Summary

| Bug | Description | Surface |
|-----|-------------|---------|
| 4 | `ui.Prompter` interface had no concrete implementation; production binary's `deps.Prompts` was always nil | byo-prepare + Path B prompt |
| 5 | `mktemp` staging file owned by SSH user → root EACCES under `fs.protected_regular=2` (Ubuntu 24.04) | byo-prepare visudo gate |
| 6 | Naive `strings.ReplaceAll("sudo ", "sudo -S ")` mangled `visudo ` → `visudo -S ` | byo-prepare wrapper + install.go wrapper |
| 7 | Preflight `sudo -n true` switch ignored `*exec.ExitError` (real executor returns one for non-zero exits) | preflight |
| 8 | `curl -fsS https://api.github.com` failed on HTTP 403 from anonymous rate limit | preflight |
| 9 | `configure_runner` + `configure_ephemeral_runner` Commands missing `Sudo: true` | install.go bootstrap |
| 10 | `wrapSudoCommand` outer brace-pipe broke inner `printf | sudo X` patterns (sha256sum) | install.go bootstrap |
| 11 | `sudo su -s /bin/bash -` login shell dropped cwd; `./config.sh` not found | script.go register_runner |
| 12 | `ServiceNotActiveError` swallowed remote stderr; users couldn't diagnose service failures | install.go + up.go |
| 13 | Stale `.runner` / `.credentials` blocked re-registration (config.sh refused) | script.go register_runner |
| 14 | Stale systemd unit blocked `svc.sh install` ("Failed: error: exists ...") | script.go service script |
| 15 | `verify_service` Command lacked `cd <installPath>` before `./svc.sh status` | install.go verify |
| 16 | `runnerOnlineWithLabels` case-sensitive match against GitHub auto-labels (`Linux`/`X64`) | up.go online check |
| 17 | `runnerNameConflict` pre-check refused to proceed when our own deterministic-name runner existed | up.go pre-bootstrap |
| 18 | Smoke harness `ssh ... 'sudo test -f'` over non-tty session needed password (config.sh + .runner are world-readable) | scripts/smoke/byo-permission.sh |

## Outcome

Plan 06-07 attempt-15 (after Bugs 4-17 in-tree + Bug 18 smoke harness)
completed BYO smoke in 125s, registered runner id 24 on
`accidentally-awesome-labs/dat0`, asserted `.runner` sentinel,
recorded `BYO_DURATION_SECONDS=125`. Cloud smoke surfaced 5 new bugs
(19-23: status/doctor/down + cloud SSH-readiness + cloud destroy
ordering) — those land in Plan 06-10.

## Verification

- Full repo `go test ./... -count=1 -race` passes (17/17 packages).
- BYO live smoke completed end-to-end through `runnerkit up` →
  `runnerkit status` → `runnerkit doctor` → `runnerkit down`.
- 18 RED test commits + 16 GREEN fix commits (Bugs 11+15+16+17+18
  bundled RED+GREEN); ~25 commits total under `(06-09)` prefix.
