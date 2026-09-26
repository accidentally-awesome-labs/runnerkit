# RunnerKit: final recommended strategy

*Date: 2026-09-26. Revised after the red-team review and editorial rulings R1–R10 (made during this analysis and open to maintainer override) (revision log at the end). Built from the evidence dossier in `evidence/` (KEYFACTS.md, gaps.md, market-brief.md, market-condensed.md, players.md, code-condensed.md), five lens proposals, three judge evaluations and two direct repo checks. Evidence references use dossier section IDs; defect IDs (P0-n, P1-n, P2-n, SEC-n) are CODEBASE-ASSESSMENT.md §5–6. All dates come from the calendar in section 10. Anything not in the dossier is marked **[inference]** or **[new]**.*

---

## 1. Verdict on market need

**Rating: narrow but real for day-2 upkeep of runners people already have. Weak to none for RunnerKit's founding thesis ("a cheaper self-hosted runner in 10 minutes for solo devs"). Weak for anything individuals would pay for.**

**Evidence that the need is real**

- **Who self-hosts.** Roughly 10^4–10^5 individuals, mostly copy-pasting `config.sh`/`svc.sh` onto a VPS or home box, often the machine that deploys their app (gaps.md solo-selfhost-demand). This is an order-of-magnitude inference from a commit proxy (95,266 commits in 2026 mention "self-hosted runner", 62% of a sample user-owned), not a measured market.
- **Pain 1: runners go offline or stop picking up jobs.** 1,714 commits in 2026 mention "self-hosted runner" with "offline"; 12 of 13 sampled were user-owned. #120813 (online but not picking up jobs) is unanswered. GitHub auto-removes a runner offline for 14 days.
- **Pain 2: version deprecation.** #4442 is the 2nd most-reacted of 137 actions/runner issues filed in 2026. Full enforcement for GHEC on github.com started 2026-09-25; personal accounts are presumed to follow the same date (MARKET-RESEARCH §2.5, inference). It hits mostly **pinned or container** runners, because `svc.sh` runners update themselves (gaps.md runner-version-enforcement; medium confidence, not live-tested).
- **Pain 3: disk and Docker rot.** #434 has been open since 2020. The maintainer's own RunnerKit host hit `df=0` on 2026-05-26.
- **Pain 4: no fallback to hosted runners.** #20019 has **107** upvotes (players.md's "197" is refuted; MARKET-RESEARCH §9.1); GitHub has "no plans" to add it.

**Evidence that the need is narrow**

- **Individuals rarely need to self-host.** An inferred 97% or more fit inside the free minutes (derived from GitHub's figure that 2.8% of individuals would have seen a decrease under the fee; GitHub never states the 97% directly, MARKET-RESEARCH §1). Public repos get free, unlimited runners. 575 commits in 2026 say they went "back to GitHub-hosted".
- **The payment statistic is weak evidence.** "0.09% would have paid more, median < $2" measures exposure to the proposed fee, not willingness to pay (user-voice-demand-16, low confidence).
- **Setup is not the pain.** About 8 genuine setup issues drew about 6 reactions (gaps.md).
- **Day-2 tools draw little interest.** soulteary/runner-fleet (22★), mikehardy/runner-fallback-action (15★), gh-runnerctl (2★).
- **RunnerKit has no signal of its own.** It was never launched; 0 stars and 35 downloads say nothing either way.

| Has the need | Does not |
|---|---|
| Owners of a VPS that doubles as a deploy agent: offline stops deploys, a full disk takes production down | Low-volume solo developers chasing savings (free minutes; cpx22 beats GitHub-hosted only above about 5,900–6,900 min/month at secondary-source prices) |
| Homelab and own-hardware self-hosters with private repos | Public repos: hosted runners are free, and GitHub says self-hosted runners should "almost never" run public code |
| Pinned or container runners (myoung34 at 66M pulls, `--disableupdate`) facing the 30-day rule | Anyone below roughly 9–21k min/month who would rent an always-on cpx22 (about 5–8k for a cx23, if orderable): managed runners are cheaper there at secondary-source Hetzner prices (MARKET-RESEARCH §3.3: cpx22 vs Blacksmith $0.004/min about 8.8k, vs Ubicloud premium $0.002 about 12.9k, vs Ubicloud standard $0.00125 about 20.6k). Blacksmith is org-only (and its price is secondary-source), and Ubicloud's personal-account support is not stated, so for a personal account the managed option is not confirmed. Above that, or on owned hardware, self-hosting wins on cost |
| Owners of hardware needing more than 2 cores / 8 GB (Free and Pro cannot buy larger runners) | Ephemeral pools for agents or untrusted code (ARC, scaleset, Zoomies, Actuated; Copilot code review is ARC-only) |

## 2. Recommended strategy and positioning

**Strategy in one paragraph.** Treat RunnerKit as a bounded probe, not a relaunch. By 2026-10-10, make the published artifact legal, honest and harmless: add a LICENSE, disable the destructive commands, guard against billing leaks, make the documented BYO install path work (a repair time-boxed to 6 hours, or refuse BYO if the box overruns), and tell the truth about security, including how to revoke what earlier versions installed. Then, under hard hour caps, test the one hypothesis the evidence supports (people who already run a self-hosted runner will adopt free tooling that stops CI from dying silently) with three low-trust artifacts, in order: standalone **recipes** plus thread replies (a zero-code pre-gate), a zero-install **runner-watch** Action, and, only if people ask for host-side tooling, a read-only **`runnerkit checkup user@host`** reusing the working doctor, preflight and OOM code. Launch once, in the week of 2026-11-09; publish a pre-committed go/kill memo on **2026-12-21**, with archiving as the default. Only on a pass, grow into a "pet-runner caretaker": first an honest privilege model, then heartbeat, hygiene and `adopt`. Freeze cloud, ephemeral, agent pools and monetization.

**Positioning statement.** *For developers who already run a GitHub Actions runner on a box they own (the VPS that deploys their app, a homelab server, a dedicated machine), RunnerKit is the free, open-source runner caretaker. It tells you before CI silently stops: offline, idle while jobs queue, auto-removed, too old for GitHub to route jobs to, or out of disk. For build and test jobs it can route new jobs to GitHub-hosted runners while your box is down; deploy jobs stay on your box. Unlike copy-paste `config.sh`, a Docker restart policy, or a Kubernetes or SaaS fleet manager, it works with the runners you already have, needs no control plane, and never puts your GitHub admin credentials on the host. For public repos and untrusted code, it tells you to use GitHub-hosted runners.*

We are the "pet" tool; Zoomies, ARC and Actuated are the "cattle" tools, and we say so.

## 3. Target segments (ranked)

| # | Segment | Why | Evidence |
|---|---|---|---|
| 1 | **Deploy-agent VPS owners** whose runner lives on the box that runs their app | Failure is most expensive: offline stops deploys, a full disk takes production down. They already chose to self-host. Hosted fallback cannot run their deploy jobs; the value is early warning | gaps.md solo-selfhost-demand; 1,714 "offline" commits, 12 of 13 user-owned; the df=0 incident |
| 2 | **Pinned or container runner users** (mostly myoung34, `--disableupdate`, custom images). ARC/GKE fleet operators are out of scope | The dossier reads every 2026 "deprecated and cannot receive messages" report as coming from a non-updating runner (mostly container, ARC or GKE images); this is explicit only in #4203 and #4392, and #4613 is unexplained (MARKET-RESEARCH §2.5). They are the runners most exposed to the 30-day rule. For myoung34 the fix is often a newer image, so the value is the warning. Willingness to pay is untested **[inference]** | gaps.md runner-version-enforcement (#4203, #4305, #4392, #4442, #4613); 66M myoung34 pulls |
| 3 | **Homelab and own-hardware users with private repos** | 14-day auto-removal when the box sleeps, silent outages, disk rot. They already run Uptime Kuma, ntfy or healthchecks | "homelab"/"LXC" labels in the commit sample; #179202 |
| 4 | **Heavy-CI indies and 2–5 person teams on owned boxes** (deferred to Stage 2–3) | More than 2 cores, warm caches, N runners per host. The only plausible payers | Team overage $42/$102/$282 per month at 10k/20k/50k min; payment unproven |
| — | *Not targeted:* cost-driven solo developers, public repos, agent-pool seekers, GHES or air-gapped enterprises | | KEYFACTS D; gaps.md agent-runner-personal-accounts |

## 4. What to stop, de-emphasize and delete

**Stop now**

1. **The "10-minute setup" and "cheaper than GitHub" messaging.** The evidence contradicts both.
2. **Widening the sudoers allowlist, and calling it scoped.** It has been root-equivalent since the first version (SEC-1); 62% of fix commits touched this surface and Phase 6 needed 21 live-smoke attempts. Fix the comment at `internal/bootstrap/sudoers.go:65-66`. v1.3.4 makes `install.sh` carry the list `RenderSudoersEntry` already defines: more lines in `install.sh`, no new capability, since the list was already root-equivalent. It is disclosed, not narrowed, until Stage 2 item 0; `RenderSudoersEntry` gains no entries.
3. **Recommending Hetzner cloud.** Change the wizard label "Hetzner cloud (recommended default)" (`internal/cli/wizard.go:67`). Retract `docs/safety.md:9-12`, which calls ephemeral BYO "stronger isolation per job" and routes untrusted work to ephemeral cloud VMs that are never auto-destroyed, keep billing and expose SSH to 0.0.0.0/0 (P1-12, SEC-11).
4. **AI-planned feature work.** 265 commits in 21 days produced 15.4k LOC nobody asked for. Rule: **no feature without 2 distinct external requests**, and no public `.planning` corpus. The one exemption is W1–W3, the validation instrument, capped at 30 build hours (section 6).
5. **Tagging releases that claim BYO works** unless a real job has run on a fresh password-sudo host prepared only by `install.sh`. v1.3.4 meets this with one container run, or ships with BYO refused (section 5).
6. **The agent-pool, scale-set, autoscaling, arm64, non-Ubuntu, macOS, multi-provider, MCP/SEED-003 and UX-polish roadmaps.**

**De-emphasize (freeze as experimental)**

- **Hetzner provider:** behind `--experimental` from v1.3.4. Still on hcloud-go v1 after the API removed `server.datacenter`; RunnerKit nil-guards it, so the `/v1/datacenters` 410 after 2026-10-01 is not an emergency (gaps.md). The hard-coded €4.90 (P0-6) gives way to the live API price.
- **Ephemeral mode:** one job per manual `up`, TTL ignored, never live-smoked (P1-12).
- **Full "image parity":** about 4.5–5 GB, Node 20 (EOL), five or more third-party apt repos (P1-14).
- **Workflow apt auto-detection:** off by default; it turns comment words into package names (P1-5).

**Delete**

- **Now:** `.planning/` (147 files, 28 mentioning the maintainer's hosts), `smoke-output.log`, `GEMINI.md`, and the `RELEASE-NOTES-*.md` files (fold into `CHANGELOG.md`).
- **Git history, before announcing:** search for hostnames, IPs and secrets. If hostnames or IPs appear, rotate keys or decommission those hosts rather than rewrite history, which would break the SHAs that the 18 tags and the cosign-signed release checksums reference. Confirm which hosts still exist: the dossier says the dat0 runner was destroyed in the 2026-05-18 smoke test, but a RunnerKit host running dat0 CI hit df=0 on 2026-05-26. Rewrite only if actual secrets appear; then publish a notice and re-tag.
- **In Stage 2, unless 5+ users ask:** the Hetzner provider (about 1.7k prod LOC), ephemeral paths, the image-parity default and the wizard.

## 5. Stage 0: stabilize and make it real (2026-09-26 to 2026-10-17; ≤25 maintainer-hours, including the 6h repair box)

**Capacity check first (by 2026-10-03).** Capacity means availability, not spend: at least 2 work sessions a week and issue responses within 72 hours through 2026-12-21. If not, do only 0.1–0.3 and archive with an honest notice; a public artifact that grants root and makes false safety claims is itself harmful. The section 10 hour caps are the only spend limits.

**Budget** (BACKLOG.md §0 committed figures): 0.1 + 0.2 about 6.5h; 0.3 about 9h including the release run; 0.3b ≤6h; 0.4 about 3.5h. BACKLOG.md's re-baselined full scope for Stage 0 is about 35h. If Stage 0 projects over 25h, P1-1, P1-2 and P1-3 (which have workarounds) move to v1.3.5. BACKLOG.md §0 also defers the toolchain bump (back in v1.3.4 only if govulncheck finds a reachable issue), the `recover` refusals, the name and git-history checks (the history check still precedes any outreach) and part of the CHANGELOG. The toolchain deferral departs from R2 and needs the maintainer's sign-off.

**0.1 Licensing and hygiene (S).** Apache-2.0 (patent grant; contrasts with Zoomies' AGPL), CONTRIBUTING with a DCO (no CLA), SECURITY.md with a 14-day response target, a CHANGELOG, the section 4 clean-up and history check. About an hour of name checks (srz-zumix/gh-runner-kit, EUIPO/USPTO); the .com/.dev/.app domains are taken, so buy none.

**0.2 Honesty pass (S).**
- **README banner:** BYO on password-sudo hosts (P0-1) and Docker on fresh hosts (P0-2), "fixed in v1.3.4" only if 0.3b passes, otherwise "BYO is not supported in this release"; job `sudo apt-get` needs `RUNNERKIT_GRANT_CI_SUDO=1` (P1-11), a root-equivalent grant (SEC-12); ephemeral BYO is broken on install.sh hosts (P0-1, P1-12); cloud and ephemeral require `--experimental`. Correct the v1.3.3 notes, which cite `byo-prepare` (deleted in v1.0.8).
- **Security-posture page listing every known issue with its fix stage:** SEC-1 (root-equivalent sudoers), SEC-2 (root runs the runner-writable `svc.sh`) and SEC-3 → Stage 2 item 0; SEC-5 (docker group is root) and SEC-12 → disclosed now, opt-in in Stage 2; SEC-4 (host keys) → strict `known_hosts` in `checkup` (Stage 1), CLI in Stage 2; SEC-6 (tokens in argv), SEC-7 (shared Unix user), SEC-8 to SEC-10 → Stage 2; SEC-11 and cloud-init `NOPASSWD:ALL` → cloud frozen, fixed only if revived (Stage 3); SEC-13 → test gate in v1.3.4, action SHA-pinning in Stage 1. Ephemeral BYO is not isolation; untrusted or public code belongs on GitHub-hosted runners.
- **"If you already installed RunnerKit"** on the same page: (1) `sudo rm /etc/sudoers.d/runnerkit-installer /etc/sudoers.d/runnerkit-runner-ci`, then `sudo visudo -c`; (2) make `svc.sh` and `bin/` under `/opt/actions-runner/runnerkit-*/` root-owned, or reinstall the service from a root-owned unit; (3) review docker-group membership; (4) delete RunnerKit-created Hetzner resources with `runnerkit destroy` or the Hetzner console.
- **"When NOT to use RunnerKit"** (commercial lens).

**0.3 v1.3.4 harm-reduction items (all S).**

| Defect | Fix |
|---|---|
| P0-4 `down`/`unregister` orphans billing VMs; P0-5 `up --cloud --replace` overwrites state | Refuse both on cloud state; point to `destroy` |
| P0-3 `upgrade-runner` (and `doctor --fix`) deletes credentials, then runs `config.sh --token ""` | Disable both, with the manual steps |
| P0-6 hard-coded "approx €4.90/month" | Live monthly price from the Hetzner API (`serverType` pricing for the chosen location plus primary IPv4; hcloud-go v1.59.2 exposes both), labelled "reported by the Hetzner API" |
| Defaults fsn1 (reportedly nothing orderable, 2026-08-25) and cpx22 (reportedly €19.49 + €0.50 IPv4) | `--cloud` and ephemeral BYO behind `--experimental`; require explicit `--cloud-region` (no fsn1 default) |
| P1-12 ephemeral cloud keeps billing | Block `--mode ephemeral --cloud` |
| P1-1 / P1-2 / P1-3 | Print Cobra errors (incl. `--version`); implement prompter `Input`; set `StateBaseDir` |
| Runner pin 2.334.0 (P2-9) | Bump to 2.337.0 with release-body SHA-256s; unit test that rendered scripts never contain `--disableupdate` |
| Go 1.22 unsupported; x/net from 2023 | Supported Go release, `go mod tidy`, govulncheck, GoReleaser `--snapshot` dry run |
| `release.yml` runs no tests (SEC-13) | GoReleaser depends on `go test` |

Cloud stays unsupported beyond these guards; P1-8 (duplicate SSH-key upload) is fixed only if hours remain. Before tagging, verify the Homebrew tap PAT and OIDC signing (idle 130 days) per the CLAUDE.md checklist; sign with cosign as today. If v1.3.4 is not tagged by 2026-10-10, deprecate the Homebrew cask instead.

**0.3b Minimal core-path repair (≤6 maintainer-hours).** The launch sends strangers to this repo, and a documented-broken core command destroys trust; the defects are reproduced and small (gaps.md released-byo-e2e). The judges objected to an L-sized correctness release, not a 6h fix.
- (a) Generate `install.sh`'s sudoers block from `RenderSudoersEntry`, with a full-body equality test replacing today's header-only test (P0-1).
- (b) Run `create_runner_user` before `setup_runner_image`, and bump `ImageSetupVersion` so existing hosts re-run (P0-2).
- (c) Wrap SSH exec errors so failures name the failing step, not "(unknown)" (P1-15).
- (d) Validate the release candidate on **one** containerized password-sudo Ubuntu 24.04 host prepared only by `install.sh`: a runner comes online and runs a real job (gcc plus `docker run hello-world` as the runner user).

**If the box is exceeded, fall back to option A:** BYO `up`/`register` refuse to run without `--accept-known-issues`, and the banner says BYO is not supported in this release. Neither path narrows the root-equivalent sudoers.

**0.4 Day-1 validations (≤4h of work).**
1. **Does GitHub already notify owners** about offline, auto-removed or deprecated runners? Grep github/docs by 2026-10-17. Stop a throwaway runner (from validation 3) by 2026-10-12 for 15 days, past auto-removal; record any email or UI notice; publish on 2026-10-27. If GitHub notifies, drop those Watch rules.
2. **Minimal token** for listing a personal repo's runners: `GITHUB_TOKEN`, then a fine-grained PAT with Administration:read; confirm `version`, `status` and `busy` come back populated [new per the skeptic, schema-verified].
3. **Runner auto-update:** on a disposable host or container, install 2.334.0 manually with `config.sh`/`svc.sh` (not RunnerKit `up`), run one job, confirm the self-update to 2.337.0, and check the #4421 0-byte `runsvc.sh` risk.
4. **Fallback mechanics:** `runs-on: ${{ fromJSON(vars.RUNS_ON) }}` routes new jobs; a Variables:write PAT can flip it (docs-verified, never tested live); already-queued jobs keep their target.

## 6. Stage 1: validate (pre-gate from 2026-10-17, launch week of 2026-11-09, decide 2026-12-21)

**Caps.** W1–W3 build ≤30h, as the explicit exemption to the two-request rule. Cumulative ≤55h at launch, ≤85h at the memo.

**W3. Recipes plus thread replies: the zero-code pre-gate (about 6h, by 2026-10-24).** Human-written guides, useful without RunnerKit, and the fallback deliverable if the probe is killed:
1. "Runner online but not picking up jobs."
2. "Disk full and root-owned workspace files": an `ACTIONS_RUNNER_HOOK_JOB_COMPLETED` chown/cleanup script, a prune timer, and a `JOB_STARTED` per-job HOME reset for claude-code-action #1688.
3. "Fall back to GitHub-hosted when your runner is down" via `vars` in `runs-on`, **for build and test jobs only**. Deploy jobs stay pinned (or fall back to an SSH-based deploy step). Flipping the variable does not rescue already-queued jobs; cancel and re-run them.

Post one helpful reply each on #120813, #4442, #20019 and #434, recipe first, authorship disclosed, mentioning that a read-only host checkup is being considered.

**W1. `runnerkit-watch` (by 2026-11-02).** A scheduled composite Action in its own repo, ≤800 LOC, Marketplace-listable; one issue per runner, optional ntfy or webhook. Rules:
- **Offline and disappearance**, warning before 14-day auto-removal. The REST object has no last-seen field, so Watch stores the first-seen-offline time in its issue and documents the 6h granularity.
- **Idle-while-queued:** a job queued more than N minutes while a matching runner is *online and not busy* (#120813).
- **Version.** The API cannot distinguish pinned from auto-updating runners, and healthy ones lag until a job arrives. Warn when installed < latest **and** the next-newer release is more than 10–14 days old; escalate beyond 25 days; error below the 2.329.0 floor. Pinned status is labelled "suspected". #4613 (2.336.0 refused while latest) is a documented unexplained case.
- **Unknown runner registrations:** a diff against the last snapshot.
- **Token expiry:** warn 14 days before a fine-grained PAT expires [new; confirm how the API exposes expiry during the build].
- **Optional fallback, build/test jobs only:** flip `RUNS_ON` between the self-hosted labels and `ubuntu-latest`. Watch already holds a token on hosted infrastructure, so no third-party monitor sends an (unverified) authenticated PATCH. Queued jobs are not rescued, and at 6h cadence the flip is hours late.
- **Opt-in `report_usage: true`** (off by default) opens one issue on the Watch repo.

Default cadence 6h: about 120 billed min/month on a private repo, since each job rounds up to a minute (hourly ≈ 720) [new, per skeptic]. Document that public-repo schedules stop after 60 days of inactivity, that Watch is slow (the Stage 2 heartbeat is the fast detector), and that it is blind during GitHub Actions outages.

**W2. `runnerkit checkup user@host` (gated).** Build only if the pre-gate or W1 yields **at least 2 external responses asking for host-side tooling**, checked on 2026-11-02 and 2026-11-23. Read-only, no install, no sudo where avoidable.
- Reuses `ops.Classify`, preflight and `hostkillhint` over `remote.Executor`; discovers `actions.runner.*` units (hand- or RunnerKit-installed) and myoung34 containers.
- Reports service state, listener staleness, installed vs latest version, "deprecated" journal lines, a 0-byte `runsvc.sh` (#4421), leftover `bin.<ver>` dirs, disk/inode and `docker system df`, root-owned `_work` files (#434), OOM kills, and leftover RunnerKit sudoers files or a root-run, runner-owned `svc.sh`.
- Prints an anonymized summary for a pinned Discussion: prevalence data no dossier source could measure.
- Required fixes: `Version` in `github.Runner` (`internal/github/runners.go:9-16` drops it); resolved unit names in `logs` and OOM heuristics (P1-9); `--fail-on` exit codes (P1-19); host-read installed version (P2-9); strict `known_hosts`, never `StrictHostKeyChecking=no` (SEC-4).
- Over budget: drop container discovery and the anonymized report first.

**No-code discovery (parallel, ≤5h).** 10 short interviews from #120813, #4442, #20019, #434, r/selfhosted and myoung34 issues: what broke, how they found out, and whether they would run a read-only checkup, a root heartbeat agent, or neither. No price questions unless org adopters appear.

**Launch (week of 2026-11-09).**
- **Hook:** "Is GitHub about to stop sending jobs to your self-hosted runner?", anchored on the 2026-09-25 enforcement and 14-day auto-removal, not cost.
- **Channels, same week, human-written** (awesome-selfhosted forbids AI-written submissions): Show HN; r/selfhosted, r/homelab, r/github; a PR to jonico/awesome-runners (895★, no BYO/day-2 row); a Marketplace listing. Follow up in the pre-gate threads only where useful.
- **Dogfood post:** the dat0 df=0 story and the recipe-2 fix.
- **Weekly public table:** opt-in Watch users, public dependents (separately), external authors, checkup reports, real-problem reports, billed minutes, hours.
- **No spend:** no ads, no domain, no Hetzner tutorial.

**Exit gate (memo on 2026-12-21).** Without default-on telemetry, adoption counts are opt-in and reported as **lower bounds**: +1s on a pinned "I'm running Watch/checkup" Discussion (linked from the READMEs and Watch's first issue), `report_usage` issues, and public-repo dependents counted separately. "Real problems caught" = Watch catches + user-confirmed actionable checkup findings.
- **PASS**, all of: ≥10 external opt-in Watch users *or* ≥15 external checkup reports; ≥5 distinct external issue or discussion authors; ≥3 real problems caught. With ≥8 checkup reports, ≥30% must show an actionable day-2 problem.
- **KILL**, any of: <5 opt-in Watch users *and* <8 checkup reports *and* <3 external authors; with ≥8 reports, actionable problems on <20% of hosts (fewer reports means grey zone); more than 85 hours spent.
- **Grey zone:** one 6-week extension to **2027-02-01**, no new features, then a binary decision defaulting to archive.

Stars are reported but never decide the gate.

## 7. Stage 2: extend (2027-01-04, or 2027-02-02 after a grey-zone pass, to 2027-03-26; only if Stage 1 passed)

In order, moving on only when earlier items are used:

0. **Honest privilege model**, required before any root-installed component and independent of setup demand: a one-time root install; a root-owned fixed-verb helper; root-owned systemd units rendered by RunnerKit (never `sudo ./svc.sh`); **root never executes runner-writable files**, while the install dir stays runner-writable so self-update keeps satisfying the 30-day rule (per gaps.md, the self-update writes into the install dir, so a root-owned one would break it; inferred from the update mechanics, not live-tested). Closes SEC-1 to SEC-3 and retires the sudoers v1.3.4 only discloses.
1. **Heartbeat / dead-man's switch** pinging a user-owned healthchecks.io, Uptime Kuma or ntfy URL; **no GitHub token on the host (Tier 0)**; catches a dead host in minutes. It may ship before item 0 **only** as a no-root `systemd --user` unit or crontab.
2. **Hygiene pack** (after item 0): the recipes as managed hooks and timers: `_work` chown (#434), stray-container kill, tiered docker prune with a build-cache cap, old `bin.<ver>` cleanup, per-job HOME (#1688), log rotation.
3. **`runnerkit adopt user@host`** (after item 0): take over hand-installed or older RunnerKit runners **without re-registration**, with host-resident state under `/var/lib/runnerkit`. Record whether each beta host arrived via `adopt` or `up`.
4. **Rebuild BYO `up` beyond the v1.3.4 repair, only with ≥5 concrete setup requests:** enforced host-key pin, stdin tokens, opt-in Docker, per-repo users, minimal image profile, resolve-latest runner with release-body SHA-256 and an auto-bumped fallback pin, a working `upgrade-runner`. Gate every release on a **real-job canary** (gcc, docker build, `services:`, `sudo apt-get`) on password-sudo Ubuntu 22.04 and 24.04 containers prepared only by `install.sh`; add govulncheck and shellcheck as CI gates.
5. **Scope deletion:** cloud, ephemeral, the wizard and apt auto-detection, unless each has ≥5 requests.

**Pivots.** *Adopt-only:* if `adopt` is ≥50% of beta hosts, or the `up` rebuild exceeds 3 weeks or 15 live-smoke attempts, freeze `up` and point users to `config.sh` or myoung34. *Upstream:* if fallback dominates, contribute to runner-fallback-action; for a watchdog UI, point to runner-fleet.

**Stage 2 gate (2027-03-26), all required:** ≥25 weekly-active watched repos or hosts (opt-in counts); ≥10 external hosts on heartbeat/hygiene for 30+ days with ≥60% 30-day retention; ≥1 user-confirmed prevented incident per 3 active hosts per month; zero outages or data loss caused by RunnerKit. On failure: maintenance mode (Watch and checkup kept current).

## 8. Stage 3: expand (2027-04 to 2027-09-26), trigger-gated bets

| Bet | Trigger | Notes |
|---|---|---|
| **N runners per host; org runners and groups** | ≥5 team/org requests, or a top-3 request | Matrix legs serialize today (P1-22); prerequisite for any team offer |
| **`audit` / `doctor --security`** | ≥1 true positive from Watch's unknown-runner rule, or a new rogue-runner wave | Delegate workflow linting to zizmor [new, per security-first] |
| **Agent pools** (JIT or scale set, sandbox per job, egress allowlist) | ≥5 requests to run agent jobs on owned hardware, *and* a 2-day spike on a *personal* repo, *and* ≥3 committed users | Breaks the deploy-agent use case. actions/scaleset is in preview and churning (breaking listener change 2026-09-15, #113; open listener-stall bug #131). Copilot code review is ARC-only. Until then: the #1688 HOME hook and a compatibility matrix |
| **Second provider / reviving cloud** | ≥5 explicit requests *and* a live authenticated check of `/v1/pricing` and `/v1/server_types` | Otherwise cloud stays behind `--experimental`; "provision any box, then `adopt`" |
| **macOS** | ≥10 requests | Tart is OpenAI-owned and FSL-licensed; costly to support |
| **Forgejo adapter** | ≥10 requests or a partner | Codeberg data was unverifiable |
| **Commercial layer** | ≥100 active hosts *and* ≥5 team requests, then 10+ interviews and a priced waitlist with ≥10 org sign-ups | Section 9 |

## 9. Business model (staged) and what never to do

- **Stages 0–2:** $0 revenue by design, no monetization code. After launch, GitHub Sponsors (prolific maintainers have about 31–32 sponsors). Disclosed referral links only after traction.
- **Stage 3, if triggered:** a **RunsOn-style commercial licence** for org features (multi-host, org runners, fleet reports, audit export), as an offline signed licence file with no phone-home, in a separate source-available module. Single-host stays free forever. Verified comparables: RunsOn €300/yr (README) and Actuated $150/month for the first server plus $125 for each additional one (MARKET-RESEARCH §7.1). The lens's €29/org/month (€290/yr) price, its finding that such a licence eats 60–100% of savings at 10–20k min/month, and its €2–9k ARR base case with under 15% odds of a salary are **[commercial-lens estimates, not dossier-sourced]**. Since managed runners beat a rented cpx22 below roughly 9–21k min/month (section 1), a buyer must be at roughly 30k+ min/month or own hardware, and buy on capability, not price.
- **Rejected: a Coolify-style $3–5/host hosted plane.** $5k MRR needs about 1,250 paying hosts, i.e. 25–42k active hosts, close to the whole estimated population. SEED-001 is a bootstrap/lifecycle split, not an outbound agent (confirmed in the repo).

**Never:** resell per-minute compute (BuildJet shut down); run a hosted service holding users' admin tokens or reaching their hosts; paywall security defaults; publish cost claims from unverified prices (live API prices are labelled as such); ship default-on telemetry; raise money; operate root-privileged CI for customers.

## 10. Calendar, success metrics and kill/pivot criteria

**10.1 Calendar (the only schedule).**

| # | Date | Milestone | Cumulative hour cap |
|---|---|---|---|
| C1 | 2026-10-03 | Capacity check; hour log and weekly table started | — |
| C2 | 2026-10-10 | LICENSE, banner, notes, security page with revocation section; v1.3.4 tagged (0.3b repair or option-A refusal), or cask deprecated | — |
| C3 | 2026-10-12 | Latest start of the 15-day offline test | — |
| C4 | 2026-10-17 | Stage 0 closes: validations 2–4 and the docs grep published | ≤25h |
| C5 | 2026-10-24 | W3 recipes and the four thread replies posted | — |
| C6 | 2026-10-27 | Offline-test readout published | — |
| C7 | 2026-11-02 | W1 released; W2 go/no-go #1 (if go, W2 before launch only if the logged hours plus W2 fit the ≤55h cap; otherwise after launch, by 2026-12-07) | — |
| C8 | Week of 2026-11-09 (latest 2026-11-16) | Launch on 4+ channels plus the awesome-runners PR | ≤55h |
| C9 | 2026-11-23 | W2 go/no-go #2 (if go, W2 by 2026-12-07) | W1–W3 build ≤30h |
| C10 | 2026-12-21 | Decision memo | ≤85h |
| C11 | 2027-02-01 | Grey-zone end; binary decision | no new features |
| C12 | 2027-01-04 (2027-02-02 after a grey-zone pass) → 2027-03-26 | Stage 2; gate memo 2027-03-26 | — |
| C13 | 2027-04 → 2027-09-26 | Stage 3, trigger-gated | — |

**10.2 Metrics** (deliverable dates and hour caps are in 10.1).

| Metric | Target | When |
|---|---|---|
| Watch billed minutes at default cadence | ≤150 min/month | A week after C8 |
| Opt-in Watch users or checkup reports (lower bounds) | ≥10 or ≥15 | C10 |
| Distinct external authors | ≥5 | C10 |
| Real problems caught (Watch + confirmed checkup findings) | ≥3 | C10 |
| Hosts with an actionable day-2 problem (only with ≥8 reports) | ≥30% | C10 |
| Concrete setup requests (gates the `up` rebuild) | ≥5 | C10 |
| Real-job canary green on every tag claiming BYO works | 100% | From v1.3.4 (one host); full matrix in Stage 2 |
| Heartbeat/hygiene hosts at 30+ days; retention | ≥10; ≥60% | 2027-03-26 |
| Prevented incidents | ≥1 per 3 hosts per month | 2027-03-26 |
| Outages or data loss caused by RunnerKit | 0 | Ongoing |

**Kill / pivot criteria**
1. No logged progress for 2 consecutive weeks, or any calendar milestone slipping more than 7 days: finish the harm patch if unfinished, then archive with a notice.
2. Launch not done by 2026-11-16, or more than 55 hours spent before launch: archive.
3. GitHub already sends native offline or deprecation alerts: drop those Watch rules; continue with checkup and recipes only if they still fill a gap.
4. The 2026-12-21 KILL thresholds: publish the recipes as gists or a small composite action, archive the CLI, hand off or archive Watch.
5. With ≥8 checkup reports, actionable problems on <20% of hosts: the day-2 hypothesis is false; stop.
6. Adopt-only or upstream pivots (section 7).
7. Stage 2 retention below 40%, or fewer than 10 retained hosts by 2027-03-26: maintenance mode.
8. GitHub ships first-party fallback, runner health alerts or a user-level runner scope: drop the overlapping features.
9. A **newly reported** vulnerability, not already disclosed on the security-posture page, that cannot be fixed within 14 days: freeze distribution and all security messaging. Disclosed issues with a planned fix stage do not trigger this.

## 11. Key risks and mitigations

| Risk | Mitigation |
|---|---|
| **Demand is too thin** (comparables at 15–22★; our prior is that kill criteria trigger) | Hour-capped, dated, archive by default; recipes ship first and stay useful |
| **Maintainer dormancy** (a 21-day burst, then 130 days of silence) | Availability check, weekly log, archive on 2 weeks without progress or a 7-day slip; resolve-latest runner (Stage 2) removes forced release cadence |
| **The 6h repair regresses BYO** (the surface behind 62% of fix commits) | Hard time box, full-body test, one real-job container run before tagging, option-A fallback |
| **Opt-in counts understate adoption** | Reported as lower bounds; public dependents separate; near-threshold results fall in the grey zone |
| **Watch token friction and exposure** (listing needs admin read; flipping needs Variables:write) | Least privilege validated on day 1; opt-in flipping; Administration:write never on hosts |
| **Fine-grained PAT expiry** makes Watch fail silently | Warning 14 days before expiry |
| **GitHub Actions outages**: self-hosting does not isolate from them, and Watch runs on Actions | Document the blind spot; the Stage 2 heartbeat reports to an external monitor |
| **The $0.002/min self-hosted fee returns** (postponed, not cancelled) | Re-run the decision memo and messaging if announced |
| **Trust barrier** (SSH checkup into production; later root components) | Checkup is read-only with strict `known_hosts`; Watch needs no host access; root components only after Stage 2 item 0, with cosign and SECURITY.md |
| **Reputational harm from v1.3.3** | v1.3.4 (repair or BYO refused), banner and revocation guide before outreach, or a deprecated cask |
| **Unverified premises and platform churn** (auto-update, population, Hetzner prices, registration floor, scaleset preview) | Manual auto-update test; API-reported prices only; 2.337.0 pin for margin above 2.329.0; config-driven Watch rules; no scaleset dependency before Stage 3 |
| **AI-assisted scope creep; faster rivals** (Zoomies, runner-fleet) | 30h W1–W3 cap, 800-LOC Watch, W2 gated on requests, two-request rule; contribute upstream if a rival covers the pull |

## 12. Scoreboard and how the panel's views were combined

| Lens | Avg /50 | Judge totals (J1 investor / J2 staff engineer / J3 persona) | Judge ranks |
|---|---|---|---|
| skeptic | **35.3** | 35 / 36 / 35 | 1 / 2 / 2 |
| reliability-wedge | 32.3 | 30 / 34 / 33 | 2 / 1 / 1 |
| security-first | 27.0 | 23 / 30 / 28 | 4 / 3 / 3 |
| commercial | 23.0 | 23 / 23 / 23 | 3 / 4 / 4 |
| agent-runners | 18.7 | 17 / 18 / 21 | 5 / 5 / 5 |

**How we combined them.** The skeptic led every judge's average on feasibility and time-to-value, so it is the **spine**: sequencing, hour caps, a dated memo, archive by default, no large BYO rework before demand. Two judges ranked reliability-wedge first on thesis, so its **product direction** (recipes, vars fallback, checkup, heartbeat, hygiene, `adopt`, pet-vs-cattle) is the continuation path, plus, per R1, a 6h slice of its correctness work; its L-sized correctness release and root-owned install dir are rejected. Security-first supplies the **privilege model** (now Stage 2 item 0) and honesty items, but not a security-led launch or 10–13 weeks of pre-value hardening. Commercial supplies **arithmetic guardrails** (marked as estimates), Apache-2.0 + DCO and "When NOT to use", but not the XL Team edition, concierge pools or month-1 hcloud-go v2 work. Agent-runners supplies only the #1688 hook, a compatibility matrix and a trigger-gated spike; its preview-API rewrite puts admin credentials beside prompt-injectable agents.

**Contradictions resolved from the evidence:** root never executes runner-writable files, but the install dir stays writable (gaps.md); Watch first, checkup only on request; Watch, not third-party monitors, flips variables, for build/test jobs only; version drift is a headline only for pinned and container runners; #20019 is 107, not 197; "outbound-only" is not a differentiator, since Zoomies has it.

## Revision log (versus the panel draft)

- **R1:** added 0.3b, a 6h minimal BYO repair (P0-1, P0-2, P1-15, one real-job container check) with an option-A refusal fallback; Stop #2 and #5 reworded; sudoers stays disclosed and root-equivalent. Red-team must-fix on shipping a broken core command.
- **R2:** v1.3.4 gains the 2.337.0 pin plus a `--disableupdate` test, a supported Go with govulncheck and a snapshot dry run, a tap-PAT check, `--experimental` cloud with the live API price and a required `--cloud-region`, and blocked ephemeral cloud.
- **R3:** kill criterion 9 covers only newly reported issues; the security page lists every SEC item with its fix stage and adds a revocation section.
- **R4:** Stage 1 runs W3 plus replies (pre-gate), then W1, then W2 only on ≥2 requests; W1–W3 exempted and capped at 30h.
- **R5:** opt-in, lower-bound adoption counts; checkup findings count as real problems; ≥8-report minimum on the 20% KILL.
- **R6:** privilege model is Stage 2 item 0; heartbeat may precede it only as a no-root user unit; the ≥5-request gate covers only the `up` rebuild.
- **R7:** availability replaces the 8 h/week rule; kill on 2 weeks without progress or a 7-day slip; caps 25h, 55h, 85h.
- **R8:** fallback limited to build/test jobs; queued jobs not rescued; positioning and segment 1 updated.
- **R9:** release-age version rule, "suspected" pinned label, persisted offline time, #4613 caveat; risks for fee reinstatement, Actions outages and PAT expiry.
- **R10:** Ubicloud "every volume" replaced by cpx22 break-evens of about 9–21k min/month (MARKET-RESEARCH §3.3; originally 9–19k from the dossier's figures, which excluded IPv4); €29 and €2–9k ARR marked as lens estimates beside verified comparables; git-history rule; 15-day offline test; manual 2.334.0 auto-update test.
- **Other red-team items:** segment-2 willingness-to-pay claim dropped (ARC/GKE out of scope); checkup flags leftover sudoers files; P1-8 noted for cloud.
- **Housekeeping:** B.1–B.8 replaced by CODEBASE-ASSESSMENT IDs; one calendar (10.1) replaces conflicting dates, and the 2026-10-26 Stage 0 end is gone; per-lens summaries condensed. BACKLOG.md was re-baselined against this document on 2026-09-26 (scope decision, G-0 capacity rule and budget ledger). Section 5's budget line now uses its committed figures, and C7 now says W2 is built after launch unless it fits the ≤55h cap, because with BACKLOG's estimates the old "W2 by 2026-11-16" date and the cap could not both hold. BACKLOG's deferral of the toolchain bump to v1.3.5 still needs a maintainer ruling.
