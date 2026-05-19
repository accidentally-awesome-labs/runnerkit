# Research: install patterns for CLI tools that need privileged host setup

## Methodology
- **Date:** 2026-05-18
- Pulled real install scripts directly via curl (Tailscale, Docker, k3s, Rustup, Homebrew, NetBird, Sentry CLI, GitHub Actions runner, GitLab Runner) and cross-referenced with vendor docs. Reflects what `main` actually ships in May 2026.

## Per-pattern analysis

### Pattern: `curl URL | sudo bash` (pipe-to-shell)
- **Technical model:** Vendor hosts a script at a stable URL; user pipes into `sh` with root. Production scripts wrap everything in `main()` called from the last line so truncated downloads fail safe.
- **Examples:** Tailscale (https://tailscale.com/install.sh — 800+ lines, 20+ distros), Docker (https://get.docker.com — tracks upstream SHA, ships `--dry-run`/`--channel`/`--version`/`--mirror`/`--no-autostart`), k3s (https://get.k3s.io — env-var contract `INSTALL_K3S_VERSION`, `K3S_TOKEN`, `INSTALL_K3S_EXEC`), Rustup (https://sh.rustup.rs — runs WITHOUT sudo, $HOME/.cargo), Homebrew (`NONINTERACTIVE=1`, refuses root).
- **Friction:** 1 command, 1 sudo prompt; TTY not required with env vars.
- **Security:** TLS+DNS only for script; packages themselves GPG-signed via apt/yum keyrings. No SBOM/signed manifest of the script itself.
- **Fit for RunnerKit:** Strong. Audience tolerates it. Eliminates NOPASSWD-sudoers fragility entirely.

### Pattern: `curl > install.sh; sudo bash install.sh` (download + inspect + run)
- **Technical model:** Same script, but the documented runbook is save → cat → `--dry-run` → run.
- **Examples:** Docker's header lists this exact 4-step recipe; k3s docs encourage saving for re-run with new env vars; Homebrew docs ditto.
- **Friction:** 3-4 commands; TTY not required; dry-run is opt-in (skipped in practice).
- **Security:** Same as pipe-to-shell but `--dry-run` adds a real audit lever.
- **Fit for RunnerKit:** Use as the DOCUMENTED form (with `--dry-run` from day one) even if the pipe form is what users actually type.

### Pattern: Cloud-init / user-data injection
- **Technical model:** `#cloud-config` YAML or shell-script user-data executed as root on first boot. Handlers for `packages:`, `runcmd:`, `users:`, `write_files:`. `cloud-init status --wait` observable.
- **Examples:** RunnerKit's own Hetzner path (`runnerkit-cloud-init-v3`); AWS EC2, Hetzner, DigitalOcean, Linode, OpenStack; k0s/k3s cloud-init recipes.
- **Friction:** Zero host-side (no host yet).
- **Security:** User-data is plaintext in the metadata service (169.254.169.254) — secrets visible to all VM processes. `/var/log/cloud-init.log` is the audit.
- **Fit for RunnerKit:** Already correct for the cloud path. Useless for BYO.

### Pattern: Signed binary distribution (apt/yum repo, Homebrew cask)
- **Technical model:** GPG-signed packages in a repo; user adds repo + key once; package manager handles install/upgrade and post-install hooks (systemd unit registration etc.).
- **Examples:** Tailscale's install.sh is mostly an apt-repo bootstrapper (`/usr/share/keyrings/tailscale-archive-keyring.gpg`, `signed-by=...`); GitLab Runner `.deb` has a post-install hook that creates `gitlab-runner` user and configures systemd; NetBird publishes both apt+yum signed repos; Homebrew Cask + macOS notarization.
- **Friction:** 1 command after repo-add; TTY not required; unattended upgrades free.
- **Security:** Best in class — signed metadata, signed packages, distro-audited hooks. `/var/log/apt/history.log`.
- **Fit for RunnerKit:** Right for the RunnerKit binary (Homebrew cask exists). Wrong for the per-host bootstrap (token-shaped, not package-shaped).

### Pattern: Install token with TTL
- **Technical model:** Control plane issues a short-lived token; agent exchanges it for long-lived credentials at install time and discards the token.
- **Examples:** GitHub Actions runner `config.sh --token` (~1h TTL, single-use, fetched via PAT); Buildkite cluster tokens + ephemeral hosted agents; GitLab Runner `--registration-token` (modern versions short-lived); Sentry DSN.
- **Friction:** 2 steps (fetch token, run installer); TTY optional with env var.
- **Security:** Bearer token; `ps aux` footgun if passed on CLI. TTL bounds blast radius. Control-plane audit.
- **Fit for RunnerKit:** Partial — already used for GitHub runner registration. Good pattern, but orthogonal to host bootstrap.

### Pattern: Container-based agent (DaemonSet / docker-compose pull)
- **Technical model:** Ship agent as container image; host setup reduced to "install Docker once".
- **Examples:** GitLab Runner Docker executor, Buildkite agent Helm chart, GitHub ARC (Actions Runner Controller for k8s), Dagger Engine.
- **Friction:** 1 command after Docker install (Docker install is its own pipe-to-shell).
- **Security:** Strong isolation; image digest pinning; Docker socket mount is the escape vector.
- **Fit for RunnerKit:** Poor. GitHub Actions runner is explicitly unsupported in containers (no systemd — see https://raw.githubusercontent.com/actions/runner/main/docs/automate.md). Also defeats RunnerKit's value prop ("runs your real workflow on a real host").

### Pattern: Device enrollment via mTLS / SSH cert / OIDC
- **Technical model:** Each host gets a unique identity (TPM, SSH host key, X.509). Short-lived bootstrap credential exchanged for long-lived device cert signed by control-plane CA. No shared secrets on disk afterwards.
- **Examples:** Tailscale auth keys → node keys (ephemeral + pre-approved); HashiCorp Boundary worker registration; Teleport node joins; AWS SSM IAM-role identity.
- **Friction:** 1 step (e.g. `tailscale up --authkey=...`); TTY not required.
- **Security:** Excellent — no long-lived shared secret on disk; revocation; mutual auth.
- **Fit for RunnerKit:** Aspirational. Requires building a RunnerKit control plane or piggybacking on GitHub OIDC. Out of scope for SEED-001.

### Pattern: WireGuard mesh enrollment
- **Technical model:** Network = control plane. Each host gets a WireGuard keypair; control plane assembles peer graph.
- **Examples:** Tailscale, Headscale, NetBird (https://github.com/netbirdio/netbird/blob/main/release_files/install.sh), Twingate, Innernet.
- **Fit for RunnerKit:** Interesting transport-layer addition (could replace SSH from laptop to host), but orthogonal to install pattern. v2 territory.

### Pattern: Magic-link / web UI to local installer
- **Technical model:** Web app generates one-time deep link or shell snippet with embedded token; user pastes into terminal.
- **Examples:** Tailscale "Add device" copyable snippet; CircleCI self-hosted runner UI; Sentry Install Wizard; Cursor/Linear/Vercel CLI auth deep links.
- **Friction:** 1 click + 1 paste; TTY required.
- **Security:** Token in URL/clipboard — both leak; TTL critical.
- **Fit for RunnerKit:** Poor (no web UI). Closest local analog is the next pattern.

### Pattern: `runnerkit init --print-install-command` (the SEED-001 candidate)
- **Technical model:** Maintainer runs RunnerKit on laptop; it emits a copy-paste line targeted at the host (`curl -fsSL https://install.runnerkit.dev | RUNNERKIT_HOST_TOKEN=... sudo sh`). Host fetches signed script, runs privileged setup once, prints host-id + non-root SSH user. All subsequent `runnerkit up/down/doctor` use non-root SSH.
- **Closest analogs:** `cloudflared tunnel install` (run on host, token-driven); `flyctl launch`; `vc link`.
- **Friction:** 2 commands; sudo TTY on the host where the user is already sitting — same friction as pipe-to-shell, which all the industry leaders ship.
- **Security:** Token TTL bounds blast radius; install script over TLS, GPG-signable; **the entire `runnerkit byo-prepare` sudoers fragment disappears**.
- **Fit for RunnerKit:** This is the proposal, and the survey supports it — Buildkite, GitHub Actions, GitLab Runner, CircleCI all ship approximately this shape. RunnerKit's current design is the outlier.

### Bonus: Configuration management (Ansible/Salt/Chef)
- Ansible playbooks for self-hosted runners are widely published. Overkill for solo developers; but RunnerKit's `byo-prepare` is essentially a bad embedded version of one. A small idempotent runner inside install.sh would close the gap.

## Cross-pattern observations
1. **Every successful CI-agent vendor requires the user to run something on the host with sudo at least once.** None tries to escalate sudo over SSH from a remote machine. RunnerKit is the outlier today.
2. **The `main() { ... }; main "$@"` trailing-call pattern is universal** for truncation safety (Tailscale, k3s, Homebrew). Any RunnerKit install.sh should follow.
3. **Two-tier auth is standard:** short-lived install/registration token → long-lived device credential (GH runner token → runner credentials; Tailscale auth key → node key; GitLab registration → runner token). RunnerKit can adopt this shape for host enrollment.
4. **`--dry-run` is standard** (Docker, k3s, Homebrew). Cheap to add, big trust dividend.
5. **Cloud-init and pipe-to-shell are complements, not competitors.** RunnerKit already uses cloud-init for cloud; SEED-001 is the BYO equivalent.

## Recommendation matrix for RunnerKit

Scoring 0-5 per dimension (5 = ideal). Score = mean of the 5 dimensions.

| Pattern | CLI-only | BYO | Cloud | Agent-driveable | Persistent+Ephemeral | Score |
|---|---|---|---|---|---|---|
| `curl \| sudo bash` (pipe) | 5 | 5 | 3 | 3 | 4 | **4.0** |
| `curl > install.sh; sudo bash` | 5 | 5 | 3 | 4 | 4 | **4.2** |
| Cloud-init / user-data | 5 | 0 | 5 | 5 | 3 | **3.6** |
| Signed binary repo | 5 | 4 | 3 | 4 | 3 | **3.8** |
| Install token w/ TTL | 4 | 4 | 4 | 5 | 5 | **4.4** |
| Container-based agent | 3 | 2 | 3 | 4 | 5 | **3.4** |
| Device enrollment (mTLS/cert) | 3 | 3 | 3 | 5 | 5 | **3.8** |
| WireGuard mesh | 3 | 3 | 4 | 4 | 4 | **3.6** |
| Magic-link / web UI | 2 | 2 | 2 | 1 | 2 | **1.8** |
| `runnerkit init --print-install-command` | 5 | 5 | 4 | 5 | 4 | **4.6** |
| Config mgmt (Ansible/Salt) | 2 | 4 | 4 | 3 | 3 | **3.2** |

Top 3 for RunnerKit's constraints: (1) `runnerkit init --print-install-command` (composes proven patterns), (2) install token with TTL (already half-used), (3) download-and-inspect form with `--dry-run`.

What to deprecate: the `byo-prepare` sudoers-fragment approach — it scores ~0 on "agent-driveable" because every bootstrap step needing a new binary becomes a new allowlist entry, which is exactly the bug class (ln/chmod/cp/cat) that motivated this survey.

## Sources
- https://tailscale.com/install.sh
- https://tailscale.com/docs/features/access-control/auth-keys
- https://get.docker.com (source repo: https://github.com/docker/docker-install)
- https://sh.rustup.rs
- https://get.k3s.io
- https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh
- https://raw.githubusercontent.com/netbirdio/netbird/main/release_files/install.sh
- https://sentry.io/get-cli/
- https://raw.githubusercontent.com/actions/runner/main/scripts/create-latest-svc.sh
- https://raw.githubusercontent.com/actions/runner/main/docs/automate.md
- https://gitlab.com/gitlab-org/gitlab-runner/-/raw/main/packaging/root/usr/share/gitlab-runner/post-install
- https://buildkite.com/docs/agent/v3/tokens
- https://github.com/macstadium/orka-integrations/blob/master/Buildkite/ephemeral-agent.md
- https://cloudinit.readthedocs.io/en/latest/explanation/format.html
- https://raw.githubusercontent.com/Homebrew/brew/master/Library/Homebrew/cask/installer.rb
