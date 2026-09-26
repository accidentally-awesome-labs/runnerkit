# RunnerKit: product, market and strategy analysis

**Date:** 2026-09-26 · **Subject:** RunnerKit at HEAD `64c3003`, which matches the released binary v1.3.3 · **Status:** analysis and recommendation. Nothing in the product has been changed.

This is the entry point. It summarizes four detailed documents in this folder:

| Document | What it contains |
|---|---|
| [`STRATEGY.md`](STRATEGY.md) | The recommended strategy: stages, dated calendar, gates, metrics and kill criteria, business model, and the reasoning behind the panel's choices |
| [`BACKLOG.md`](BACKLOG.md) | An actionable backlog that implements the strategy. Each item lists files, fix approach, acceptance tests, effort and budget |
| [`CODEBASE-ASSESSMENT.md`](CODEBASE-ASSESSMENT.md) | Features, design, strengths, verified defects (P0, P1, P2 and SEC), security posture, workload compatibility and engineering quality |
| [`MARKET-RESEARCH.md`](MARKET-RESEARCH.md) | Pricing and economics, the competitive landscape, demand and user voice, trends, and go-to-market analogues. Every number carries a verification marker and a source |
| [`evidence/`](evidence/) | The verified evidence dossier the documents above cite:<br>• [`KEYFACTS.md`](evidence/KEYFACTS.md): the shared baseline<br>• [`gaps.md`](evidence/gaps.md): five deep dives — a run of the released binary against a password-sudo host, runner-version enforcement, Hetzner price and stock, agent runners on personal accounts, and launch/demand evidence<br>• [`code-condensed.md`](evidence/code-condensed.md): code findings with verifier verdicts<br>• [`market-condensed.md`](evidence/market-condensed.md) and [`market-brief.md`](evidence/market-brief.md): market findings, sources and verifier corrections<br>• [`players.md`](evidence/players.md): the competitor catalogue |

---

## TL;DR

1. **RunnerKit is well designed but works poorly as shipped.**
   - The design is good: plan before mutation, verified cloud destroy, a secret-free state file, redaction, stable error codes, and dependency injection that keeps it testable. 548 tests pass.
   - The released binary fails on the path it is built around. On a normal Ubuntu host where `sudo` asks for a password, `runnerkit up --host` fails 40 s in, at `setup_runner_image` (**P0-1**, reproduced against the released v1.3.3).
   - Even when bootstrap completes, Docker and `services:` jobs fail, because the runner user is never added to the docker group (**P0-2**).
   - `upgrade-runner` destroys the runner it is meant to upgrade (**P0-3**).
   - `down` on a cloud runner orphans a VM that keeps billing (**P0-4**).
   - The cloud price shown at consent is hard-coded at about 4× below today's reported price (**P0-6**).
   - The "scoped" sudoers grant is effectively root.
   - None of this was caught because no test executes the generated shell or runs a real workflow job.
2. **Nobody uses it yet, and that says little about demand.** 0 stars, 0 forks, 35 downloads (a pattern consistent with the maintainer's own use), no LICENSE, never launched, and no commits since 2026-05-19.
3. **The founding thesis, "a cheaper self-hosted runner in 10 minutes for solo developers", is weak.**
   - By inference from GitHub's own data, about 97% or more of individuals fit inside GitHub's free minutes.
   - GitHub cut hosted prices by up to 39% on 2026-01-01; Linux 2-core is now $0.006/min.
   - Hetzner's new-order prices reportedly rose sharply in 2026 (secondary source).
   - Setup is not where users complain.
   - The managed-runner market is crowded, well funded, consolidating and aimed at organizations.
4. **A narrow but real need exists: keeping the runners people already self-host healthy.**
   - The evidenced pains:
     - runners that show online but never pick up jobs;
     - version deprecation, now enforced (2.329.0 floor plus a 30-day update rule);
     - disk and Docker rot;
     - silent 14-day auto-removal;
     - no fallback to hosted runners (a request GitHub says it has "no plans" to build).
   - The people affected: owners of a VPS that doubles as a deploy agent, homelab and own-hardware users, and teams running pinned or container runners.
   - No open-source tool owns this "pet runner caretaker" niche. Comparable tools have 15–22 stars, which is also a warning that pull may be modest.
5. **Recommendation: make it honest and working, then run a small, time-boxed probe of the day-2 wedge before building anything bigger.**
   - **Stage 0** (≤25 hours; v1.3.4 by 2026-10-10, stage closes 2026-10-17): add a license, disclose the known issues honestly, and ship v1.3.4, which fixes the harmful paths and makes the core BYO path work.
   - **Stage 1** (launch the week of 2026-11-09): publish recipes, then a zero-install `runnerkit-watch` GitHub Action, then a read-only `runnerkit checkup user@host`.
   - **Decision memo on 2026-12-21**, with pre-committed pass and kill thresholds. Archiving is the default if the signal is not there.
   - Cloud, ephemeral pools, AI-agent runners, a second provider, macOS, Forgejo and monetization are **gated expansions**. Each has a specific trigger signal and none is built by default.

---

## 1. What RunnerKit is today: features, functionality and design

**Shape.** A Go 1.22 / Cobra CLI of about 15.4k production LOC and 13k test LOC. It has two provisioning paths:
- **BYO.** The laptop SSHes into a Linux/systemd host and runs about 7 generated bash scripts under a NOPASSWD sudoers fragment. The fragment is installed once with `curl install.sh | sudo bash`.
- **Hetzner Cloud.** RunnerKit provisions a VM with cloud-init, then runs the same bootstrap.

There are two runner modes: **persistent**, the default for trusted private repos, and **ephemeral**, a single job with a TTL. Day-2 commands are `status`, `logs`, `doctor` (`--deep` OOM heuristics, `--fix`, `--ignore`, `--json` with `next_actions`), `recover`, `down`/`unregister`, `destroy`, `list`/`register` (several repos on one BYO host), `upgrade` and `upgrade-runner`. An "image parity" layer installs about 70 apt packages plus Node, Go, Rust, Java, .NET, Docker, browsers and gh, and scans workflow YAML for apt packages. Releases go out through a GoReleaser pipeline with cosign signatures and a Homebrew cask.

**Scorecard** (the full matrix is in `CODEBASE-ASSESSMENT.md` §1 and §3):

| Area | Maturity | Headline |
|---|---|---|
| BYO on a password-sudo host (the documented golden path) | **Broken** | The install.sh sudoers is missing 16 paths the bootstrap needs (P0-1). Reproduced on the released binary |
| BYO where the SSH user already has NOPASSWD:ALL | Fragile | Expected to reach online, but Docker and Rust are silently missing (P0-2) |
| Hetzner cloud, persistent x64 | Partial | The most mature path; its live smoke was green on 2026-05-18. The consent price is wrong (P0-6), and `down`/`--replace` can orphan VMs (P0-4, P0-5) |
| Ephemeral mode | Fragile | One job per manual `up`. The TTL flag is ignored on the host, the finalizer cannot write, and the cloud VM is never auto-destroyed (P1-12) |
| Day-2 ops | Partial | The health model is good, but `upgrade-runner` is destructive (P0-3), `logs` queries the wrong unit (P1-9) and `recover` is miswired (P1-10) |
| CLI UX | Fragile | Errors are silent (P1-1), typed confirmations cannot be answered (P1-2), config is written into the working directory (P1-3), and the wizard runs nothing (P2-1) |
| State store / distribution | Solid | Atomic, versioned, secret-free; cosign-signed releases. But there is **no LICENSE** |
| Security model | Weak | The sudoers grant is root-equivalent. Root later runs a runner-writable `svc.sh`. The host-key pin is not bound to the real session. All repos on a host share one Unix user |
| Workload fit vs GitHub-hosted | Fragile | gcc builds work. Docker, `services:`, `sudo` steps, arm64, parallel matrix legs, macOS and Windows fail or are absent |
| Tests and CI | Partial | 548 tests are green and race-clean, but the rendered bash is never executed. The release job runs no tests. There are no lint, vuln or shellcheck gates |
| Toolchain | Stale | Go 1.22 is out of support, hcloud-go is v1 (the Hetzner API dropped `server.datacenter`), and Node 20 is EOL |

**Design strengths worth keeping:**
- plan-before-mutation with billing consent;
- checkpointed, provider-verified cloud cleanup;
- a secret-free, migration-safe state store;
- durable credentials never leave the workstation;
- the safety gate runs before any side effect;
- an injectable, testable architecture;
- a multi-source health classifier;
- stable RKD error codes;
- runner self-update left on, which is load-bearing under GitHub's 30-day rule.

The architecture is sound. The failures come from execution and verification.

**How the project got here.** One author made 265 commits in 21 days, 18 tags, heavily AI-assisted. 62% of fix commits touched the bootstrap/sudo surface, and Phase 6 needed 21 live-smoke attempts. Each fix widened the sudoers allowlist. The v1.3.3 fix changed the Go renderer, which only cloud-init uses, and never reached `install.sh`. The planning corpus drifted from the code. That drift is what sent the fix to the wrong place.

---

## 2. Is there a market need?

**Verdict: narrow but real for day-2 upkeep of self-hosted runners. Weak to none for "cheaper runners for solo developers".** The evidence is in `MARKET-RESEARCH.md` §§1–5 and `evidence/gaps.md`.

**Economics** (MARKET-RESEARCH §3; Hetzner inputs are secondary-sourced):

| Who | Cheapest sensible option |
|---|---|
| Solo developer under about 3k private min/month (the large majority) | GitHub-hosted free minutes. Self-hosting saves nothing |
| Public repos | GitHub-hosted: free, unlimited, 4 vCPU / 16 GB. GitHub also says self-hosted runners should "almost never" run public code |
| An always-on rented cpx22, against GitHub-hosted | Breaks even around 5,900 min/month on Free and 6,900 on Pro |
| Managed runners (Ubicloud from $0.00125/min; Blacksmith $0.004 and Depot $0.006, both org-oriented) against an always-on rented box | Managed is cheaper below roughly 9–21k min/month. Above that, or on hardware you already own, self-hosting wins on cost |
| Individuals needing more than 2 cores / 8 GB | Must self-host: Free and Pro accounts cannot buy larger runners |

**Competition** (MARKET-RESEARCH §4):
- **Managed runners.** Blacksmith, Depot, Namespace, WarpBuild, Ubicloud and RunsOn are funded and org-focused. BuildJet shut down, and Cirrus left the market.
- **Open source.** ARC is Kubernetes-only. GitHub's new **actions/scaleset** Go client, in preview since Feb 2026, removes the Kubernetes barrier for autoscalers. terraform-aws-github-runner and GARM also exist, and **myoung34/docker-github-actions-runner** has 66M pulls.
- **Hetzner-specific tools.** Cyclenerd hcloud-github-runner and TestFlows.
- **Closest new rival: Zoomies** (single Go binary, ephemeral containers, outbound-only agents, v1.0 on 2026-09-13).
- **The gap.** No tool combines "no control plane, works on the box you already own, persistent and warm, with host diagnostics and day-2 care". That is the niche.

**Demand signals** (MARKET-RESEARCH §5, `evidence/gaps.md`):
- **Population.** About 10⁴–10⁵ individual self-hosters (an order-of-magnitude inference). Most use copy-paste `config.sh`/`svc.sh`, often on the same VPS that deploys their app.
- **Top pains:**
  - "online but not picking up jobs" (#120813, 103 comments, unanswered);
  - version deprecation (#4442 is among the most-reacted runner issues of 2026);
  - disk and root-owned workspace rot (#434, open since 2020);
  - no fallback to hosted (#20019, 107 upvotes, GitHub has "no plans").
- **Setup complaints are rare.**
- **Trends:**
  - AI-agent PR volume grew about 180× in 16 months. But GitHub steers agent workloads to ephemeral, scale-set runners, and Copilot code review is ARC-only.
  - Supply-chain attacks moved into CI runtimes, including rogue self-hosted runners in Shai-Hulud. This raises the bar for persistent runners.
  - The $0.002/min self-hosted fee is postponed, not cancelled.

**Why RunnerKit has no users today:**
1. It was never launched.
2. Its core path does not work on the host shape most people have.
3. It has no license, so channels such as homebrew-core, awesome-selfhosted and Hetzner Community tutorials are closed.
4. It sells day-1 setup and cost savings, which the evidence says are not the pain.
5. It has been dormant for 130 days, while runner-version enforcement, Hetzner price and API changes, and the Node 20 EOL moved underneath it.

---

## 3. How to improve, build, optimize, extend and expand it

Everything below is sequenced and gated in `STRATEGY.md` and itemized in `BACKLOG.md`. The rule is to **improve and stabilize before building, and to build only what a measured signal justifies.**

### Improve: make it honest and working (Stage 0, v1.3.4, ≤25 h)
- **License and hygiene:**
  - Apache-2.0, a DCO, SECURITY.md and a CHANGELOG.
  - Move `.planning/`, smoke logs and agent memory files out of the public tree. Do not rewrite git history unless actual secrets are found.
  - Remove planning jargon from user-facing output.
- **Honesty pass:**
  - A known-issues banner.
  - Corrected v1.3.3 release notes (they point at the deleted `byo-prepare`).
  - A security-posture page listing every known issue and its fix stage, with **revocation steps for anyone who already ran `install.sh`**.
  - Retract the claims that ephemeral BYO gives "isolation" and that Hetzner is the "recommended" path.
- **Harm fixes:**
  - a cloud guard on `down` and `--replace` (P0-4, P0-5);
  - disable the destructive `upgrade-runner` and `doctor --fix` until they are fixed (P0-3);
  - live Hetzner pricing from the API, an explicit `--cloud-region`, and `--cloud` and ephemeral BYO behind `--experimental` (P0-6);
  - block ephemeral cloud;
  - print CLI errors (P1-1), implement prompt input (P1-2) and set the state directory (P1-3). These move to v1.3.5 if Stage 0 projects over its 25-hour cap; `BACKLOG.md` §0's ledger says it will.
- **Minimal core-path repair, time-boxed to 6 hours:**
  - generate `install.sh`'s sudoers from the one Go source, with a full-body equality test (P0-1);
  - create the runner user before image setup and bump the marker (P0-2);
  - name the failing step in errors (P1-15);
  - prove it on a containerized password-sudo Ubuntu 24.04 host that runs a real job (gcc plus `docker run`).
  - If this overruns, BYO `up` refuses to run without `--accept-known-issues`.
- **Freshness:**
  - bump the runner pin to 2.337.0, with a test that forbids `--disableupdate`;
  - run govulncheck, and move off Go 1.22. `BACKLOG.md` defers the toolchain bump to v1.3.5 unless govulncheck finds a reachable issue. That deferral needs the maintainer's sign-off (see §6);
  - make the release workflow run the tests.

### Build: the day-2 wedge, as a probe (Stage 1)
- **W3, three recipes first** (zero code):
  - "runner online but not picking up jobs";
  - "disk full and root-owned workspace files", using job hooks and a prune timer;
  - "fall back to GitHub-hosted for build/test jobs", by putting a repo variable in `runs-on`.
  - Post helpful replies on #120813, #4442, #434 and #20019.
- **W1, `runnerkit-watch`.** A zero-install scheduled GitHub Action that alerts on:
  - offline runners and 14-day disappearance;
  - jobs queued while a matching runner sits idle;
  - version lag against the 30-day window and the 2.329.0 floor;
  - unknown runner registrations (post-Shai-Hulud).
  - It can optionally flip build/test jobs to hosted runners.
- **W2, `runnerkit checkup user@host`.** Built only if the recipes or Watch draw at least 2 external requests for host-side tooling (checked on 2026-11-02 and 2026-11-23). A read-only host check that reuses the existing doctor, preflight and OOM code on *any* runner, however it was installed. It covers service state, installed version, disk and Docker usage, root-owned `_work` files, OOM kills and leftover update directories.

### Optimize: performance and UX, as the wedge touches each area
- **SSH.** Multiplex connections (ControlMaster) and batch probes; today about 40 handshakes happen before bootstrap. Honor `~/.ssh/config`. Use standard fingerprints and strict host-key checking bound to the real session.
- **Image profiles.** `minimal` is the default on personal machines; `full` parity is opt-in. The full image costs about 4.5–5 GB of disk and about 1.4 GB of downloads, while preflight checks for only 2 GiB. Show the plan before consent.
- **Runner version.** Resolve the latest release at install time, verified by the release-body SHA-256, with an auto-bumped fallback pin. Doctor should read the version actually on the host.
- **Automation contract.** One JSON envelope and one `next_actions` shape. Health-aware exit codes (`--fail-on`). A `code` field on every finding.
- **Verification.** A hermetic sshd-container end-to-end suite that runs the real `SystemExecutor` and a real job. shellcheck on rendered scripts. A lint that checks every `sudo` command against the allowlist.

### Extend: only if the Stage 1 gate passes (Stage 2)
0. **An honest privilege model first:**
   - a one-time root install;
   - a root-owned helper with fixed verbs and root-owned units;
   - root never executes files the runner can write;
   - the install directory stays runner-writable, so self-update keeps satisfying the 30-day rule;
   - tokens passed over stdin;
   - per-repo Unix users;
   - Docker as a disclosed opt-in.
1. **Heartbeat / dead-man's switch** to a monitor the user already runs (healthchecks, Uptime Kuma, ntfy). No GitHub token on the host.
2. **Hygiene pack.** The Stage 1 recipes installed as managed hooks and timers.
3. **`runnerkit adopt user@host`.** Take over hand-installed runners without re-registering them.
4. **BYO `up` rebuild** on the new privilege model, only if at least 5 concrete setup requests arrive, gated on a real-job canary.
5. **Delete scope nobody asked for:** cloud, ephemeral, the wizard and apt auto-detection unless each has at least 5 requests.

### Expand: optional bets, each with a trigger (Stage 3)

| Bet | Build it only when |
|---|---|
| **N runners per host; org runners and runner groups** (matrix legs currently run one at a time) | 5 or more team or org requests, or it is a top-3 request |
| **Security audit** (`doctor --security`: unknown runners, risky triggers, delegated to zizmor) | Watch's unknown-runner rule records a true positive, or a new rogue-runner wave creates pull |
| **Agent-ready ephemeral pools** (actions/scaleset or a JIT loop, sandbox per job, egress allowlist, per-job HOME) | 5 or more requests to run agents on owned hardware, **and** a 2-day spike proves scale sets on a *personal* repo, **and** 3 users commit. The upside is highest here, but so are the risk and the churn (the preview API broke on 2026-09-15; code review is ARC-only) |
| **Reviving cloud, or a second provider** (hcloud-go v2, arch-aware images, an SSH key per runner) | 5 or more explicit requests and a live authenticated price and stock check. Until then: "provision any box, then `adopt`" |
| **macOS (Tart) / Forgejo runner adapter** | 10 or more requests, or a partner |
| **Commercial layer** (a RunsOn-style signed licence for org features; single-host use free forever) | 100 or more active hosts and 5 or more inbound team requests, then a priced waitlist |

**Never:**
- resell compute;
- run a hosted service that holds users' GitHub admin tokens or can reach their hosts;
- paywall security defaults;
- publish unverified prices;
- turn telemetry on by default;
- raise money for this.

---

## 4. Calendar, gates and how we will know

| Date | Milestone |
|---|---|
| **2026-10-03** | Capacity check: at least 2 work sessions a week and issue responses within 72 h through 2026-12-21. Otherwise ship only the honesty and harm fixes, then archive |
| **2026-10-10** | v1.3.4 tagged (with the 6 h repair, or the `--accept-known-issues` fallback), or the Homebrew cask deprecated; LICENSE, banner, security page with revocation steps |
| 2026-10-12 | Latest start of the 15-day offline test (does GitHub notify owners before auto-removal?) |
| **2026-10-17** | Stage 0 closes (≤25 h): day-1 validations answered — minimal token, auto-update test via a manual `config.sh` install, `vars`-driven `runs-on` |
| 2026-10-24 | W3 recipes and the four thread replies posted (the zero-code pre-gate) |
| **2026-11-02** | W1 `runnerkit-watch` released; first W2 go/no-go |
| **Week of 2026-11-09** | One human-written launch (latest 2026-11-16): Show HN; r/selfhosted, r/homelab, r/github; a jonico/awesome-runners PR; a Marketplace listing. ≤55 h cumulative |
| 2026-11-23 | Second W2 go/no-go; if go, W2 ships by 2026-12-07 |
| **2026-12-21** | Published go/kill memo. ≤85 h cumulative |
| 2027-02-01 | End of the grey-zone extension, if used: binary decision, archive by default |
| 2027-01-04 → 2027-03-26 | Stage 2 (from 2027-02-02 after a grey-zone pass). Exit gate: 25 or more weekly-active watched repos or hosts, 10 or more external hosts retained 30 or more days (≥60%), zero incidents caused by RunnerKit |

**The Stage 1 pass gate.** All counts are opt-in lower bounds: a pinned Discussion, an off-by-default `report_usage` input, and public dependents counted separately. There is no default-on telemetry. Pass requires all of:
- 10 or more external opt-in Watch users, or 15 or more checkup reports;
- 5 or more distinct external authors;
- 3 or more real problems caught (Watch catches plus user-confirmed checkup findings).

With 8 or more checkup reports, at least 30% must show an actionable day-2 problem.

**Kill** on any of:
- fewer than 5 Watch users, fewer than 8 checkup reports and fewer than 3 authors, all at once;
- with 8 or more reports, fewer than 20% showing an actionable problem;
- more than 85 hours spent.

Stars are reported but never decide anything. The exact rules and the capacity/availability kill rule are in `STRATEGY.md` §6 and §10.

---

## 5. Key risks and what is still unverified

| Risk / uncertainty | Handling |
|---|---|
| Demand may simply be too thin (comparables have 15–22★) | Hours are capped, dates are fixed, archive is the default, and the recipes are useful regardless |
| Maintainer capacity (a 21-day burst followed by 130 days of silence) | An availability rule, a weekly log, and early archive after 2 missed weeks |
| **Unverified premises:** Hetzner 2026 prices and stock (secondary sources only), whether runners auto-update (medium confidence), the population estimate, Blacksmith funding and the BuildJet shutdown date | Check live before use. Publish no unverified prices. The Stage 0 validation installs 2.334.0 by hand and watches it self-update |
| Platform churn: registration floor, 30-day rule, scaleset preview, Hetzner API removals | Resolve the latest runner at install time. Take no scaleset dependency before Stage 3. hcloud-go v2 only if cloud is revived |
| The self-hosted fee returns, or GitHub ships native health alerts or fallback | Re-run the decision memo, and drop the overlapping features |
| Trust: strangers must SSH checkup into production boxes | Checkup is read-only and uses strict host keys. Watch needs no host access. Cosign-signed releases and SECURITY.md |

**Research limitations.** An egress proxy blocked most vendor sites (hetzner.com, blacksmith.sh, ubicloud.com, docs.github.com, github.blog), HN and Reddit, and the web-search budget ran out partway through. Facts read from GitHub-hosted primary sources are marked ✅ in `MARKET-RESEARCH.md`: the `github/docs` source repo, actions/runner, and vendor repos and config. Everything else is ⚠️. **Run an authenticated Hetzner `/v1/pricing` and `/v1/server_types` call before changing any default or publishing any cost claim.**

---

## 6. Decisions the maintainer should make now

1. **Accept, amend or reject rulings R1–R10.** They are listed in `STRATEGY.md`'s revision log. They were made during this analysis to resolve red-team findings, for example putting the minimal BYO repair into v1.3.4. They are not the maintainer's decisions.
2. **The Go toolchain bump:** keep it in v1.3.4, as R2 says, or defer it to v1.3.5 to fit the 25-hour cap, as `BACKLOG.md` §0 does.
3. **Capacity:** can the 2-sessions-a-week availability rule be met through 2026-12-21? If not, the plan is: honesty and harm fixes, then archive.
4. **Where planning material lives.** The strategy recommends moving `.planning/` out of the public tree, and this folder with it.
5. **Host inventory.** The evidence disagrees on whether the `dat0` runner host still exists. Confirm which RunnerKit-managed hosts and Hetzner resources are live, and revoke the sudoers fragments on any you keep (see the revocation steps planned for the security page).
6. **Verify before publishing.** Run a live, authenticated Hetzner `/v1/pricing` and `/v1/server_types` call before any cost claim or default change.

## 7. How this analysis was produced

Three orchestrated multi-agent workflows:
1. **Analysis.** 8 code-analysis lenses: CLI/UX, BYO bootstrap, cloud and ephemeral, GitHub/state/security, ops/diagnostics, engineering quality, workload compatibility, and history/traction. 6 market lenses: pricing, managed competitors, OSS tools, user voice, agents and security trends, and go-to-market.
   - Every lens was checked by an **adversarial verifier**. Code verifiers re-opened the cited file:line references and ran reproductions. Market verifiers re-checked prices, dates and star counts against primary sources.
   - A completeness critic then commissioned 5 gap investigations, including a run of the **released v1.3.3 binary** against a password-sudo Ubuntu 24.04 container.
2. **Strategy.** 5 strategists with distinct lenses (reliability wedge, agent-era runners, security-first, commercial, honest skeptic) were each scored by 3 independent judges: a market realist, a solo-maintainer engineering realist, and target-user personas.
   - Final scores out of 50: skeptic 35.3, reliability-wedge 32.3, security-first 27.0, commercial 23.0, agent-runners 18.7.
   - The synthesis uses the skeptic's discipline as the spine and the reliability wedge as the product direction.
3. **Hardening.** A red-team review raised 21 amendments. Editorial rulings on those amendments, made in this analysis and open to the maintainer's review, were applied in `STRATEGY.md`. Separate fact-checkers edited each appendix, and a final cross-document consistency pass reconciled them.

About 50 agents ran in total. Nothing in the product code was modified.
