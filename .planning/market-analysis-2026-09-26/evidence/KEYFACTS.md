# RunnerKit strategy dossier: key facts (as of 2026-09-25/26)

Compiled from 8 code-analysis streams and 6 market-research streams. Every stream was adversarially verified, and 5 gap-fill investigations followed.
Detail files in this directory:
- code-condensed.md (code findings with verifier verdicts)
- market-brief.md (market summaries, verifier corrections and omissions)
- market-condensed.md (all market findings with sources)
- gaps.md (gap-fill deep dives)
- players.md (every competitor and tool)

The raw per-stream JSON output was not committed.

Verification caveat: an egress proxy blocked most vendor sites (hetzner.com, blacksmith.sh, ubicloud.com, docs.github.com, github.blog, HN, Reddit) during research. Facts from github.com-hosted primary sources (github/docs repo, actions/runner, vendor repos) are verified. Hetzner 2026 prices, Blacksmith funding, the BuildJet shutdown date and Ubicloud price rises rest on secondary sources and are marked as such.

## A. What RunnerKit is today (verified in code)
- Go 1.22 / Cobra CLI, about 15.4k production LOC and 13k test LOC. 548 tests pass (-race clean), with 69.5% coverage. go vet is clean. There are 14 subcommands.
- Paths:
  - BYO: the laptop SSHes into a Linux/systemd host and runs about 7 bash scripts under a "scoped" NOPASSWD sudoers fragment, installed once with `curl install.sh | sudo bash`.
  - Hetzner cloud: provision a VM with cloud-init, then run the same bootstrap.
- Modes:
  - persistent (default for trusted private repos)
  - ephemeral (a single `config.sh --ephemeral` job, then done; no autoscaling, no JIT, no scale sets, one runner per repo)
- Ops: status, logs, doctor (--deep OOM heuristics, --fix, --ignore, --json with next_actions), recover, down/unregister (BYO), destroy (cloud, verified against the provider), list/register (several repos on one BYO host), upgrade (prints instructions), upgrade-runner.
- "Image parity": about 70 apt packages plus Node 20 (EOL), Go (latest), Rust, Java 17, .NET 8, Docker CE, Chrome/Firefox and drivers, gh, cmake. Workflow YAML is regex-scanned for apt packages.
- Distribution: GoReleaser, cosign-signed checksums, 4 platforms, a Homebrew cask in its own tap.
- Engineering strengths: plan-before-mutation, dry-runs, stable RKD error codes with docs anchors, a secret-free state file (0600, atomic writes, forward migrations with backup), a redaction layer, dependency injection with fakes, and ownership tags on cloud resources.

## B. Shipped defects (verified; several reproduced against the RELEASED v1.3.3 binary)
P0 (the product does not work for its core path):
1. The documented BYO path is broken on password-sudo hosts. install.sh's sudoers is missing 16 command paths that bootstrap needs. `byo-prepare` was deleted in v1.0.8, yet the v1.3.3 release notes tell users to run it. The v1.3.3 "Bug A fix" changed only the Go renderer, which only cloud-init uses. Reproduced: the released binary fails at setup_runner_image (`sudo: a terminal is required`) after 39.7s and names the failing step "(unknown)".
2. setup_runner_image runs before create_runner_user. On every fresh host the runner user is never added to the docker group and Rust is skipped, both silently (`|| true`), and the marker file blocks any retry. So `docker build`, `services:` and `container:` jobs fail out of the box (reproduced: docker.sock permission denied).
3. `upgrade-runner` (also called by `doctor --fix`) deletes the credentials and then runs `config.sh --token ""`. The runner is left unconfigured. `tar --skip-old-files` also means binaries are never replaced.
4. `down`/`unregister` has no cloud guard. On a Hetzner runner it deletes local state, the only record of the resource IDs, and never touches Hetzner, so the VM is orphaned and keeps billing. `up --cloud --replace` overwrites cloud state the same way.
5. Cloud cost shown at the consent screen is hard-coded "approx €4.90/month" for every type and region. The default cpx22 is reportedly €19.49 (+€0.50 IPv4) for new orders since 2026-06-15 (secondary source). The default location fsn1 reportedly had no orderable types on 2026-08-25.
6. Silent CLI errors. Unknown commands, unknown flags, bad flag values and --version all print nothing and exit non-zero, because Cobra errors are swallowed.
7. The production prompter has no Input method, so every typed confirmation fails in a real TTY. That covers destroy, the public-repo ack, state replace and the BYO host prompt. Users are forced onto --yes.
8. StateBaseDir is never set in main.go. config.json is read from the current directory, `doctor --ignore` always fails, and sessions/*.json files are written into the user's repo checkout.

P1:
- `logs` and the OOM heuristics query the wrong systemd unit name, so the runner journal comes back empty.
- The ephemeral TTL flag is ignored on the host (hard-coded 24h). The finalizer runs as the service user and fails its writes. The cloud ephemeral VM is never auto-destroyed and keeps billing.
- The cloud-init "recoverable error" tolerance is dead code under `set -e`. Real errors retry for 15 minutes.
- The same local SSH public key is uploaded to Hetzner on every provision, and Hetzner rejects duplicate fingerprints (inferred, not live-tested).
- arm64 is advertised but broken: amd64 is hard-coded in 6 places, and the image lookup ignores architecture.
- Non-Ubuntu distros pass preflight but fail bootstrap (dnf is handed Ubuntu package names; Debian/Mint get Ubuntu-only repos).
- Package auto-detection is on by default and turns comment words and option arguments into apt package names.
- down/destroy against a dead host can never clear state. A partial cleanup exits 0.
- Health commands exit 0 even when they report ERROR. The JSON contract is heterogeneous across commands.
- The runner tarball cache is verified only on first download. configure re-downloads the 225MB tarball, which defeats the shared cache.
- No state-file locking.
- No signal handling (Ctrl-C during provisioning).
- hcloud-go is still v1.59.2 (v2.49.0 is current). server.datacenter was removed from the API on 2026-07-01, and /v1/datacenters returns 410 after 2026-10-01.
- Go 1.22 is out of support. x/net is from 2023. CI has no lint, vuln or shellcheck gates, and the release workflow runs no tests.

Security (verified):
- The "scoped" sudoers allowlist is effectively NOPASSWD ALL (su, tee, cp, dpkg, apt-get and systemctl with any arguments), and has been since the first version.
- Cloud-init also grants the admin user NOPASSWD:ALL.
- The runner install dir, including svc.sh, is owned by the runner user, and RunnerKit later runs `sudo ./svc.sh`. Any job can plant root code.
- Docker group membership (once the ordering bug is fixed) is root-equivalent.
- All repos on a multi-repo host share one Unix user.
- The host-key pin is checked against a separate ssh-keyscan connection. Real sessions use StrictHostKeyChecking=no, and the fingerprint format is non-standard.
- Registration tokens end up in argv and in sudo's auth logs.
- Cloud SSH is open to 0.0.0.0/0 by default.
- Tests assert rendered bash strings against fake executors and never execute the generated shell. No smoke test ever sends a workflow job to the runner.

Workload fit:
- One runner per repo means matrix legs run one at a time.
- Cloud runners give the job user no sudo at all; hosted runners have passwordless sudo, and many workflows use it.
- There is no workspace or Docker hygiene.
- macOS, Windows, GPU and Android emulators are unsupported.
- Measured install footprint is about 4.5–5GB of disk and about 1.4GB of downloads, but preflight only requires 2GiB free.
- Projected first run is about 3.5–5 minutes on a fast host and 8–12 minutes on a 2 vCPU host or a 25Mbps link.
- The cloud smoke took 498s in total, including destroy.

Runner version:
- RunnerKit pins actions/runner 2.334.0; the latest is 2.337.0.
- RunnerKit never passes --disableupdate, so runners self-update and still work. Inferred with medium confidence, not live-tested.
- GitHub now enforces a registration floor of 2.329.0 and a 30-day update rule (full enforcement on GHEC/github.com from 2026-09-25).
- doctor never reads the version actually on the host.

## C. Traction and history (verified via the GitHub API)
- One author made 265 commits in 21 days (2026-04-28 to 2026-05-18), with 18 tags, v1.0.0 to v1.3.3. At least 88 commits carry AI co-author trailers.
- 62% of fix commits touched the bootstrap/sudo surface. Phase 6 needed 21 live-smoke attempts.
- Traction: 0 stars, 0 forks, 0 watchers, and no issues or PRs ever. There have been 35 total downloads across 16 releases, 14 of which got 2 or fewer (the maintainer's own pattern).
- There is no LICENSE, so the project is legally not open source.
- It was never launched: no Show HN, Reddit or announcement.
- Dormant: no commits since 2026-05-19 (about 130 days).
- The maintainer's own host hit df=0 (disk exhaustion) on 2026-05-26 while running CI through RunnerKit.
- The repo root presents as an AI-planning workspace: .planning has 147 files, plus smoke logs, CLAUDE.md and GEMINI.md.
- The original research rated demand "low-to-medium confidence, validate with early users", and that validation was never done.

## D. Market facts
Pricing (verified from the github/docs source unless noted):
- GitHub-hosted Linux 2-core costs $0.006/min, down from $0.008 on 2026-01-01 ("up to 39%" cut). arm64 is $0.005, ubuntu-slim 1-core $0.002 (15-minute limit, no Docker), Windows $0.010, macOS $0.062.
- Included minutes: Free 2,000, Pro 3,000, Team 3,000.
- Public repos get free, unlimited standard runners.
- Larger runners are available only to Team and Enterprise Cloud orgs; Free and Pro individuals cannot buy them.
- Concurrency caps: Free 20, Pro 40, Team 60. Job timeout is 6h hosted versus 5 days self-hosted.
- The self-hosted $0.002/min "platform charge" was announced 2025-12-16 for 2026-03-01 and postponed within days. The docs as of 2026-09-25 still say self-hosted runners are free. It is postponed, not cancelled.
- GitHub's own data: only 0.09% of individual Free/Pro users with private-repo Actions usage would have paid more under the fee. Only 2.8% of individuals pay any hosted overage, so at least 97% fit inside the free minutes.

Hetzner (SECONDARY sources, unverified by primary):
- Prices rose 30–37% on 2026-04-01.
- On 2026-06-15, new-order prices rose again: CX23 €5.49, CAX11 €5.99, CX33 €8.49, CPX22 €19.49, CPX32 €35.49 (net, excluding €0.50 IPv4).
- CX/CAX lines are frequently out of stock (early Sep 2026).
- Break-even for cpx22 against GitHub-hosted is about 5,900 min/month on Free and about 6,900 on Pro. For cx23 (if orderable) it is about 3,200 on Free and 4,200 on Pro.

Managed runners:
- Crowded and consolidating. Blacksmith: $0.004/min, a reported $45M Series B (unverified), about 6,000 customers.
- Depot: $0.006/min (verified from its docs repo), $20/month for 2,000 minutes.
- Namespace: $23M. Ubicloud: standard $0.00125/min, premium $0.002/min, $2.50/month credit (verified from its docs repo).
- RunsOn: AWS-only, €300/yr commercial license, claims $0.0009/min (verified from README).
- WarpBuild, Cirun and CRACI (EU, 2025) are also active.
- BuildJet shut down (2026-03-31, secondary source). Cirrus Runners closed to new customers after Cirrus Labs joined OpenAI. Shipfox pivoted.
- Blacksmith and GitHub larger runners require an organization.
- Vendors marketed the proposed fee as a "self-hosted runner tax".

OSS / self-managed substitutes (star counts verified):
- ARC: 6.5k stars, Kubernetes. Its legacy persistent modes are community-maintained only.
- actions/scaleset: GitHub's official Go client for building autoscalers without Kubernetes. MIT, public preview since 2026-02-03, v0.4.0, about 193 stars, API churn (a breaking listener change on 2026-09-15). Repo-level scale sets work with a PAT in code, but this has not been proven on personal accounts. Needs Go 1.26.
- terraform-aws-github-runner: 3.1k stars. GARM: 407 stars.
- myoung34/docker-github-actions-runner: 2.5k stars, 66M Docker Hub pulls. This is the de facto packaged substitute.
- Hetzner-specific: Cyclenerd hcloud-github-runner (182 stars, one VM per job, runs inside the workflow) and TestFlows Hetzner runners (101 stars, always-on autoscaler, cache volumes, recycling).
- Zoomies (eyupio/zoomies): v1.0.0 on 2026-09-13, 51 stars, AGPL. A single Go binary with no Kubernetes, ephemeral containers, outbound-only agents and a web UI. It is the closest new rival.
- Others: soulteary/runner-fleet (22 stars, a watchdog for several runners per machine) and solutionforest/EPAR (a warm pool with a microVM per job on your own hosts).
- Ansible roles (macunha1, 38 stars).
- macOS: Tart (6.9k stars, now owned by OpenAI and FSL-licensed), Cilicon (1.2k) and Tartelet (797).
- Forgejo and Gitea runners exist.

User voice (verified mostly from github.com community discussions and issues):
- Individuals' pain is day-2 upkeep, not setup:
  - runners that show online but pick up no jobs (#120813, 103 comments; #3609, 141 comments);
  - runner-version deprecation silently breaking pinned runners (#4203, #4442, and more);
  - disk and Docker rot, and root-owned workspace files (#434, open since 2020);
  - no fallback to hosted runners when self-hosted is down (#20019, 107 upvotes; GitHub has "no plans");
  - fear of persistent runners after the Shai-Hulud rogue-runner incidents.
- Setup complaints are few and draw few reactions.
- The fee backlash was about principle and org bills, not solo developers.
- A commit-message proxy suggests roughly 10^4 to 10^5 individual self-hosters, typically using copy-paste config.sh/svc.sh on a VPS or home box, often as a deploy agent.
- 575 commits in 2026 say they went "back to GitHub-hosted"; some users simply make the repo public.

Trends:
- AI agents drive huge PR volume: agent-authored PRs went from about 63k (May 2025) to about 11.45M (Sep 2026), per a community tracker.
- Copilot cloud agent can use self-hosted Ubuntu x64/Windows runners on personal repos. GitHub RECOMMENDS ephemeral single-use runners (via ARC or the Scale Set Client), the firewall must be disabled, and the timeout is capped at 59 minutes.
- Copilot code review is ARC-only for self-hosting.
- claude-code-action works on self-hosted runners and is the best fit (22.9k dependent repos), but it needs a separate HOME per job.
- gh-aw needs Linux, Docker and egress.
- codex-action needs disposable hosts with sudo.
- Supply-chain attacks moved into the CI runtime: tj-actions; Shai-Hulud 1 and 2 (2025, including rogue self-hosted runners as backdoors); 2026 waves via community trackers (TanStack cache poisoning plus OIDC token from runner memory, Megalodon across 5,561 repos, Miasma reading Runner.Worker memory, ChainDrop in 444 packages).
- GitHub's response: SHA pinning policy, immutable releases, and blocking pull_request_target by default in public repos from 2026-11-02. Its guidance says self-hosted runners should "almost never" be used on public repos.
- GitHub Actions reliability complaints are rising (many 2026 outages; the figures are unverified). Self-hosting does not isolate you from control-plane failures.
- A small, values-driven migration to Codeberg/Forgejo is under way (Zig), and those users self-host runners, often on Hetzner.
- Ubuntu 26.04 hosted images exist. Node 20 is deprecated in the runner.

GTM analogues:
- Coolify: $5/month for 2 servers plus $3 per server, with Hetzner referral and sponsorship. Dokploy: $4.50/server. RunsOn: €300/yr license, free for non-commercial use. Actuated: $150/month per server.
- Donation-only models raise little. Earthly stopped maintenance after its cloud shut down.
- Channels:
  - jonico/awesome-runners (895 stars) lists no Hetzner/BYO tool, a gap RunnerKit could fill.
  - awesome-selfhosted requires a human-written submission.
  - Hetzner Community tutorials require traction first.
  - homebrew-core requires a license.
- The name "RunnerKit" is free on package registries, but the .com, .dev and .app domains are taken. Also check "gh-runner-kit".
