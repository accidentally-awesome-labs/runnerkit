# Recommendation — what to do about RunnerKit BYO after the 2026-05-18 discovery

This document synthesizes the four research streams + the live-smoke evidence + the curl|bash Docker experiment. Read it last; it cites the other docs.

## TL;DR

- **v1.3.2 cloud BYO is solid.** Cloud smoke green end-to-end (498s); destroy on a 4-day-old VM also clean.
- **v1.3.2 BYO host install is broken.** `runnerkit up --host` fails at the `setup_runner_image` step because `sudo ln` (+ `chmod`, `cp`, `cat`) are not in the scoped sudoers allowlist.
- **A 4-line allowlist patch (v1.3.3) restores BYO end-to-end** with no architecture change. Confirmed by Docker experiment.
- **Every other major runner/agent tool in the market uses one of two patterns RunnerKit doesn't: (a) `curl|sudo bash` host-local install, or (b) agent-dials-home outbound HTTPS.** RunnerKit's "control plane SSHs in with scoped sudoers" model is unique — and is the root cause of the bug class.
- **Three real options for the user**, with concrete sizing below.

## Cross-research consensus

| Question | Answer |
|---|---|
| Who else uses scoped-sudoers allowlist? | **Nobody.** [RESEARCH-competitors.md] confirms no surveyed competitor (Actuated, RunsOn, WarpBuild, Ubicloud, Cirun, Namespace, BuildJet, Blacksmith, ARC, Fireactions) uses this pattern. Path C is a RunnerKit invention. |
| How does the rest of the industry communicate with BYO hosts? | **Agent dials home over outbound HTTPS.** Buildkite, GitLab, CircleCI, Drone, Datadog, AWS SSM, Tailscale, Twingate, k3s, Boundary, GitHub Actions itself. [RESEARCH-adjacent.md] |
| What's the canonical BYO install command shape? | **`curl URL \| sudo bash`** with a distro-detecting installer that adds a GPG-signed apt/dnf repo. Tailscale, Docker, k3s, Datadog, NetBird. [RESEARCH-patterns.md] |
| Does the canonical model deliver clean non-TTY lifecycle? | **Yes, confirmed by the Docker experiment.** 15s install, all 4 v1.3.3 fix candidates work non-interactively, narrow allowlist held against escalation. [EXPERIMENT-curl-bash.md] |
| Do existing GHA runner installers handle dependencies correctly? | **No** — official `installdependencies.sh` covers only runtime libs (libssl/libicu/liblttng), no `build-essential`. RunnerKit's `BaselinePackages` (~75 pkgs) is the right deviation. [RESEARCH-github-installers.md] |
| Is the registration token API still using deprecated headers? | **Yes, in the official `actions/runner` create-latest-svc.sh** (open issue #3915). RunnerKit should use current headers (`Authorization: Bearer`, `X-GitHub-Api-Version: 2022-11-28`). |
| Token rotation? | **Best in class: Boundary** auto-rotates worker credentials every 14 days. AWS SSM and Tailscale also TTL-bound bootstrap creds. RunnerKit has no rotation today. [RESEARCH-adjacent.md] |

## What the bugs actually justify

| Bug | Severity | Fix size | Architecture-implicating? |
|---|---|---|---|
| A: `ln`/`chmod`/`cp`/`cat` missing from sudoers allowlist | P0 (smoke red) | 4 lines | Yes — symptom of a non-scaling model, but the immediate fix is tiny |
| B: Stale sudoers fragment not auto-refreshed | P1 (silent regression after upgrade) | Add hash compare to `upgrade-runner` | Partial — disappears entirely if install.sh replaces `byo-prepare` |
| C: `doctor --json` error envelope missing 4 contract fields | P2 (breaks agent/MCP integrations) | ~15 lines + test | No — orthogonal |

Bug A is patchable in <1 hour. Bug C in ~1 hour. Bug B is 2-3 hours.

The architectural decision is **not** "should we fix these bugs" (yes, obviously) but **"should we keep growing the sudoers allowlist forever or change the install model".**

## Three options

### Option A — Patch-only (v1.3.3, ~3-4 hours)
**Scope:**
- Add `ln`, `chmod`, `cp`, `cat` to `RenderSudoersEntry`
- Add error-envelope contract fields to `doctor --json`
- Extend `assert-doctor-json-contract.sh` to cover error path
- Document `byo-prepare --refresh` as the way to update stale sudoers (or run `byo-prepare` again — same effect)
- Tag v1.3.3, push to upstream

**What this gets you:**
- Today's failure on `salar@mckee-small-desktop` would pass after `runnerkit byo-prepare` re-runs against the host (which rewrites the now-current allowlist)
- Cloud smoke continues to work as it already does
- No user-visible API or UX changes

**What this doesn't address:**
- Every future bootstrap step that needs a new binary still requires another allowlist entry (`mv`? `dd`? `sed`? `wget`?)
- `sudo cp` and `sudo cat` of arbitrary paths remain latent privilege-escalation footguns (the allowlist matches binary + any args, not binary + path pattern)
- The maintainer keeps fighting the model

**Best fit if:** maintainer is time-constrained and wants to ship a fix now; happy to keep playing whack-a-mole on the allowlist; not yet ready to commit to architectural work.

### Option B — install.sh pivot (v1.4.0 milestone, ~2 GSD phases)
**Scope:**
- Ship a real `runnerkit-install.sh` (signed via cosign, fetched from GitHub release artifact, distro-detecting Ubuntu/Debian/Fedora)
- Tailscale-style: install.sh adds GPG-signed apt/dnf repo for `runnerkit-runner` package; package install creates service user, sudoers, cache dir via standard postinst hooks
- `runnerkit init --print-install-command` emits the one-liner the maintainer pastes on their host
- **Path-scoped sudoers entries** for `cp`/`cat`/`ln`/`chmod` (e.g. `/usr/bin/ln -sf /usr/local/go/bin/* /usr/local/bin/*`) so the allowlist tightens at the same time it expands
- `runnerkit byo-prepare` kept as deprecated alias for v1.4 cycle, removed in v1.5
- Lifecycle (`up`/`register`/`down`/`destroy`) still SSH-from-laptop — unchanged
- Integration test matrix: Ubuntu 24.04, Debian 12 (skip Fedora 40 — `setup_runner_image` is Ubuntu-gated anyway)
- v1.3.3 patches included as Phase 1 work so smoke is green throughout

**What this gets you:**
- Host-side privileged setup happens once, with one sudo prompt the maintainer was going to type anyway
- `runnerkit-install.sh` becomes the canonical install path; apt/dnf upgrades become free
- Latent attack vectors closed (path-scoped sudoers)
- Bug class (allowlist gap) goes away — `runnerkit-install.sh` is responsible for *everything privileged*; lifecycle ops never need a new sudo entry
- Matches industry shape (Tailscale, Buildkite, Datadog, k3s, GitLab)
- SEED-001 mostly delivered

**What this doesn't address:**
- "Control plane SSHs into host" remains; lifecycle still uses non-root SSH + scoped sudo. That's *fine* once install.sh has set up the allowlist correctly, but the SSH-from-laptop direction is still industry-unusual.

**Best fit if:** maintainer wants to close the class of bugs permanently and is willing to spend ~2 phases on it. SEED-002 (cloud multi-repo) and SEED-003 (agent plugin) become tractable on top of this.

### Option C — Full agent-dials-home (v1.5.0+, ~3-4 GSD phases)
**Scope:**
- Everything in Option B
- Plus: a `runnerkit-agent` daemon on the host that polls GitHub Actions for jobs (already what the GHA runner does internally), polls its own state for lifecycle ops (status/upgrade/cleanup directives), and exposes a Unix socket for local CLI
- The maintainer's laptop talks to GitHub directly (not the host) for issue/runner state; the agent on the host is the source of truth for everything host-side
- SSH from laptop becomes optional / debugging-only — no longer the lifecycle pathway
- Two-stage tokens with rotation (Boundary 14d)
- GitHub App auth alongside PAT

**What this gets you:**
- Industry-aligned (every surveyed tool works this way)
- Behind-NAT BYO works (no inbound reach needed)
- Sudoers allowlist disappears entirely (no ad-hoc sudo over SSH; the agent owns its own files)
- Path opens for SEED-003 (Claude Code plugin / MCP) since the agent surface is a clean API

**What this costs:**
- Substantial code — a new daemon, IPC contract, state machine, rotation
- v1.x users get a forced re-install (deb/rpm migration story is non-trivial)
- Probably v1.5 minimum, possibly v2.0

**Best fit if:** maintainer wants the architectural endpoint and is okay with longer timeline. Aligns with where the industry is.

## What I'd pick if forced

**v1.3.3 patch + v1.4.0 = Option B.** Reasoning:
1. The patch unblocks users today (Hetzner cloud users aren't even affected, and BYO Path C users get unstuck after re-running `byo-prepare`).
2. `runnerkit-install.sh` is the smallest possible architecture step that closes the bug class permanently — and the Docker experiment already shows the shape works.
3. Option C is correct as a long-term endpoint but doesn't *need* to happen in v1.4. It's a clean extension on top of Option B.
4. Option A alone doesn't close the issue — it just moves the next bug into the future.

## Concrete patterns to borrow during v1.4.0 (if Option B picked)

From the research (with attribution):

| Pattern | Source | Where to apply in RunnerKit |
|---|---|---|
| `main() { ... }; main "$@"` truncation-safe wrapper | Tailscale install.sh, k3s, Homebrew | `runnerkit-install.sh` |
| EUID-0 precondition check, exit non-zero with actionable message | actions/runner `installdependencies.sh` | `runnerkit-install.sh` first executable line |
| Distro detection via `/etc/os-release` ID + ID_LIKE | Tailscale install.sh | `runnerkit-install.sh` distro branch |
| GPG-signed apt/dnf repo + signed-by keyring | Buildkite, Tailscale, NetBird, Datadog | RunnerKit publishes a `runnerkit` repo alongside the existing Homebrew tap |
| Env-prefix one-liner UX (`DD_API_KEY=... bash -c "$(curl ...)"`) | Datadog | `RUNNERKIT_REPO=... RUNNERKIT_TOKEN=... bash -c "$(curl ...)"` |
| Two-token model: 1h registration → long-lived per-host credential | GitHub Actions runner itself + GitLab Runner | RunnerKit already does PAT → registration token; add long-lived per-host credential with rotation |
| `#cloud-boothook` (init stage) instead of cloud_final_modules | machulav/ec2-github-runner | RunnerKit Hetzner cloud-init v3 hardening |
| Refuse-if-exists, `-f` to force | actions/runner `create-latest-svc.sh` | `runnerkit-install.sh` and `runnerkit register` |
| `--dry-run` mode | Docker, k3s, Homebrew | `runnerkit-install.sh --dry-run` (cheap audit lever) |
| Try-multiple-library-versions in apt-get | actions/runner `installdependencies.sh` | RunnerKit `BaselinePackages` should accept `libssl3 libssl3t64` and similar |
| Path-scoped sudoers entries instead of bare commands | (own design) | New `RenderSudoersEntry` model: `/usr/bin/cp /tmp/* /opt/actions-runner/*` not `/usr/bin/cp` |
| Resource-scoped tokens | CircleCI, Coder | Long-lived RunnerKit credentials limited to (org/repo, host_id) scope |
| `operator credentials rotate/list/revoke` CLI | Nomad, Tailscale | RunnerKit credential CLI surface |
| `runner_state: started\|stopped\|absent` lifecycle vocabulary | Ansible role | RunnerKit CLI verb consistency audit |
| `no_log: true` discipline on PAT-touching code | Ansible role | Audit redact patterns in `internal/redact` + `internal/provider/github` |
| Current registration-token API headers (`Bearer`, `vnd.github+json`, `X-GitHub-Api-Version`) | GitHub docs (vs. issue #3915) | Audit `internal/provider/github` |

## What this discovery did NOT validate

- Multi-OS smoke matrix (Ubuntu 22.04, Debian 12, Fedora 40) — we have evidence on Ubuntu 24.04 only. If maintainer wants confidence on Debian/Fedora before committing to Option B, add a 1-day Docker matrix run before phase planning.
- `runnerkit-agent` viability — Option C is well-supported by research but no experiment yet. If maintainer leans toward C, a 2-3 day spike to prototype the daemon + Unix socket would de-risk it.
- Existing user migration path — if v1.4.0 ships install.sh as the new BYO model, users with `byo-prepare`-installed sudoers need a clean migration. Migration design is part of v1.4.0 phase planning.

## Side-effects from today's discovery

- **Production CI runner for `accidentally-awesome-labs/dat0` was destroyed.** Maintainer indicated they'll restore manually. Hetzner is currently empty; `runnerkit list` is clean.
- 3 smoke logs + 4 research docs + 1 experiment doc + this recommendation committed under `.planning/smoke-discovery-2026-05-18/`.
- `salar@mckee-small-desktop` still has the stale-allowlist sudoers fragments from 2026-05-10. Will fix itself on first new `runnerkit byo-prepare` run after v1.3.3.
- 2 new memory entries: `feedback_test_reality_before_scoping_work.md` (test before scoping speculative work), `feedback_dont_assume_orphan_infra.md` (verify before labeling infra "orphan").

## Next decision

The maintainer should pick A / B / C and what to do about the dat0 production runner. After that, scoping work continues from a known evidence base instead of from the stale memory of the v1.0.0 milestone smoke-red.
