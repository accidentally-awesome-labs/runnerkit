# Milestones

## v1.3.2 Self-hosted GitHub Actions runner v1 (Shipped: 2026-05-18)

**Phases completed:** 6 phases, 35 plans, 79 tasks

**Key accomplishments:**

- Go/Cobra RunnerKit CLI with deterministic version/up output, automation flags, typed exit codes, and redacted human/JSON rendering
- Repository-scoped GitHub auth foundation with gh-first discovery, runner-token permission fixtures, redaction, and public/fork safety gates in `runnerkit up`
- Versioned secret-free RunnerKit foundation state with atomic saves, deterministic labels, reusable workflow plans, `runnerkit up` persistence, and redacted `state show` inspection
- Production `runnerkit up` now performs real GitHub credential discovery, registration-token permission checks, repository metadata reads, and public-repo safety blocking by default.
- RunnerKit now has a BYO SSH target contract, host-key trust fields, and a remote preflight report before runner installation.
- RunnerKit can render and apply a deterministic BYO Linux bootstrap plan for a non-root persistent runner service.
- RunnerKit now registers and verifies a repository-scoped persistent runner, persists secret-free BYO state, and prints copy-paste workflow guidance.
- Phase 2 now has user-facing safety boundaries, a fake end-to-end BYO smoke test, and concise quickstart documentation.
- Read-only runner status with shared health classification, fast SSH/systemd probes, reusable Phase 3 fakes, and JSON/human output parity
- Bounded redacted log collection plus read-only doctor findings with exact remediation commands and troubleshooting docs
- Guided recover command for service restart, service reinstall, and runner re-registration with fresh redacted GitHub tokens
- Safe `runnerkit down` cleanup with artifact planning, stale GitHub deletion, exact host artifact removal, and partial-state checkpoints
- Hetzner cloud-provider planning foundation with explicit `runnerkit up --cloud hetzner` intent, non-token GitHub read checks, and cost-aware human/JSON provisioning plans
- Hetzner VM, SSH-key, firewall, cloud inventory, pending cleanup checkpoints, and readiness gates before GitHub runner registration
- Provisioned Hetzner machines now become RunnerKit runners through the shared BYO bootstrap path, with final cloud state and provider-aware status/logs/doctor output
- Cloud destroy now plans, applies, verifies, and documents billable Hetzner cleanup before local state removal
- runnerkit up now explains and gates persistent vs ephemeral runner mode tradeoffs before any GitHub, remote, provider, or state side effect.
- `runnerkit up --mode ephemeral` now creates a real RunnerKit-managed one-job GitHub Actions runner with a one-shot systemd unit, 24h TTL safeguard, finalizer-preserved logs, and mode-aware status/logs/doctor/down/destroy semantics for both BYO and Hetzner cloud.
- `docs/safety.md`, README, BYO quickstart, and cloud quickstart now explain exactly when persistent runners are unsafe and when ephemeral mode is recommended; fake end-to-end CLI tests plus lifecycle regression tests prove RunnerKit's UX and one-job semantics for trusted persistent, public-blocked persistent, public ephemeral cloud, and BYO ephemeral scenarios without live GitHub or Hetzner calls.
- Tag-triggered GoReleaser pipeline producing 4-platform signed binaries (cosign keyless via GitHub Actions OIDC), Homebrew Cask publishing to a separate tap repo, plus install/verify documentation and a maintainer-only release process — closing REL-05 distribution requirement and Phase 6 success criterion 1.
- Lazy 24h-cached update notifier, channel-detecting `runnerkit upgrade`, idempotent `upgrade-runner` re-bootstrap, forward-only state migration framework with side-by-side backup, and a stale-runner doctor warning — wired through to docs/upgrade.md.
- Stable `RKD-<COMPONENT>-NNN` error code registry, 6-component troubleshooting docs with explicit HTML anchors, CLI emit-site wiring across 18 doctor findings + 5 user-facing failure paths, and cross-linked README/quickstart/safety navigation — closing DOC-04 and the Phase 6 troubleshooting success criterion.
- Live smoke harness + verification baseline + first release notes — closing Phase 1 outstanding GitHub permission smoke and Phase 4 outstanding Hetzner billable smoke STATE.md notes; D-10/D-11/D-12-gates-1-and-2/D-13 implemented; final v1 sign-off pending the maintainer human-action checkpoint that runs `make smoke-live` against real billable Hetzner + GitHub credentials.
- Real `sudo -n true` preflight probe + sudo-prefixed download_runner + redacted remote stderr surfacing closes the two BLOCKER bugs that made BYO bootstrap unusable in v1.
- Path B (interactive sudo password fallback in `runnerkit up`) + Path C (`runnerkit byo-prepare` scoped sudoers installer with visudo lockout-prevention) close the BYO gap-doc 2026-05-04 user decision so a fresh user can complete BYO setup against a sudo-with-password host without manually editing /etc/sudoers.d/.
- Replaced `sudo -u runnerkit-runner ./config.sh` with `sudo su -s /bin/bash - runnerkit-runner -c "..."` in both bootstrap renderers to close Bug 3 (register_runner runas mismatch), restoring BYO functionality for v1.0.0 against hosts whose sudoers grants only `(root) NOPASSWD: ALL`.
- Created:
- Modified:
- Closes 4 surgical bugs (Bugs 24-27) that block Plan 06-07 attempt-17+ smoke-green: SSH host-key fingerprint determinism for status, sudo-prompt gating in down, Hetzner auto_delete cascade in cloud destroy, and the scoped sudoers svc.sh glob path.
- 1. [Rule 3 - Blocking] `deps.Env` field does not exist; used `os.Getenv` to match codebase precedent

---
