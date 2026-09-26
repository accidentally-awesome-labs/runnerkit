# RunnerKit market research: self-hosted GitHub Actions runners for solo developers

**As of:** 2026-09-26 · **Subject:** RunnerKit (Go CLI that sets up GitHub Actions self-hosted runners on a BYO Linux host over SSH or on a Hetzner Cloud VM) · **Basis:** the adversarially verified research dossier in `evidence/` (KEYFACTS, market-condensed, market-brief, players, gaps). No new web facts were fetched for this write-up. Every figure below comes from the dossier, and every verifier correction has been applied.

## How to read this report

**Verification markers** (attached to every number):

| Marker | Meaning |
|---|---|
| ✅ | **Verified, primary source.** Read from a primary source (the `github/docs` source repo, a GitHub page or discussion, the GitHub or Docker Hub API, or a vendor's own GitHub repo or config file), and confirmed, or at least not contradicted, by the adversarial verifier. |
| ⚠️ | **Secondary or unverified.** The figure comes from a search snippet, a third-party blog or tracker, a vendor website the proxy blocked, a single-source claim, or the researchers' own inference or estimate. |
| ❌ | **Refuted or outdated.** A primary source contradicted the claim. The corrected value is given next to it. |

- **(calc)** marks arithmetic done in this report or in the dossier. A calculated number is only as reliable as its weakest input, so it carries that input's marker.
- **Source keys** such as `[G1]` or `[H3]` resolve to URLs in §10.
- **"Solo developer"** means a GitHub Free or Pro personal account whose private repos are not in an organization.

---

## 1. Key takeaways

1. **For a typical solo developer, cost is a weak reason to self-host.**
   - GitHub-hosted Linux 2-core costs **$0.006/min** ✅ [G1]. That price follows a cut of **up to 39%** on 2026-01-01; the old rate was $0.008 ✅ [G5][G6].
   - Free includes **2,000** minutes a month and Pro **3,000** ✅ [G3]. Public repos use standard runners for free with no limit ✅ [G3].
   - GitHub's own data: only **0.09%** of individual Free/Pro users with private-repo Actions usage would have paid more under the proposed self-hosted fee, with a **median increase under $2/month** ✅ [G5].
   - An inferred **≈97% or more** of individuals stay within their free minutes ⚠️. This comes from GitHub's "2.8% would see a decrease" figure; GitHub never stated it directly.
2. **Hetzner, RunnerKit's cloud backend, got much more expensive and harder to buy in 2026** (every Hetzner figure here is ⚠️ secondary).
   - A new cpx22 order costs **€19.49/month** net (+€0.50 IPv4) since 2026-06-15 [H1][H3]. RunnerKit's consent screen still shows a hard-coded **"approx €4.90/month"** ✅ [R2], so the real cost is about 4× what the screen says.
   - The cheap CX and CAX lines were often **out of stock** in August and September 2026 [H2][H4].
   - An always-on cpx22 only beats GitHub-hosted above roughly **5,900 min/month on Free** and **6,900 on Pro** (calc ⚠️).
3. **Self-hosted runners are still free, but the fee risk has not gone away.**
   - GitHub announced a **$0.002/min** "Actions cloud platform charge" on 2025-12-16 and postponed it on 2025-12-17 ✅ [G5][G7].
   - GitHub's billing docs still say usage is "free for self-hosted runners" as of 2026-09-25 ✅ [G4].
   - GitHub said *postponed*, not cancelled, and has published no new date ⚠️. github.blog could not be checked.
4. **Runner-version enforcement is now live, and it only breaks runners that do not update themselves.**
   - Runners must be at least **2.329.0** to register, and must install each new release **within 30 days** or GitHub stops queuing jobs to them ✅ [G13][G15].
   - GitHub Enterprise Cloud (GHEC) reached full enforcement on **2026-09-25** ✅ [G13]. Personal accounts are presumed to follow the same github.com date ⚠️ (inference).
   - Runners with auto-update enabled comply automatically ✅ [G13]. The 2026 refusal reports exist ✅ [G18]; that every one involved a pinned or non-updating runner is the dossier's reading (explicit only for #4203's `disableUpdate=true` and #4392's container image; #4613 on 2.336.0 is unexplained) ⚠️.
5. **The managed-runner market is a crowded commodity, and it is consolidating.**
   - A 2-vCPU Linux minute costs **$0.00125–$0.006** across vendors (Ubicloud ✅, Depot ✅, others ⚠️).
   - Money is concentrating in a few players. Blacksmith reportedly raised **$45M at a $550M valuation** ⚠️.
   - Weaker players are leaving: BuildJet shut down ⚠️, and Cirrus Labs joined OpenAI, with Cirrus CI closed ✅ [M10].
   - The funded vendors serve **GitHub organizations**, not personal accounts:
     - Blacksmith is org-only ⚠️.
     - Depot's setup requires the org owner role ✅ [M1].
     - GitHub larger runners are Team or Enterprise only ✅ [G10].
6. **The open-source side moved fast in 2026.**
   - GitHub released **actions/scaleset**, an MIT-licensed Go client for building autoscalers without Kubernetes (public preview, 193★) ✅ [O1]. It makes building autoscaling runners much easier.
   - **Zoomies** is the closest new rival: a single Go binary, no Kubernetes, ephemeral-only, AGPL. v1.0.0 shipped 2026-09-13 and it has 51★ ✅ [O9].
   - The Hetzner niche already has **Cyclenerd** (182★, one VM per job) and **TestFlows** (101★, autoscaler with cache volumes) ✅ [O7][O8].
   - No open-source tool combines RunnerKit's exact mix: no control plane, a laptop CLI over SSH, persistent by default, and host diagnostics. Several tools cover parts of it.
7. **Demand from individuals exists, but it is niche, and the pain is day-2 upkeep, not setup.**
   - An order-of-magnitude estimate puts individual self-hosters at **~10⁴ to 10⁵** ⚠️ (inference from a commit-message proxy).
   - The loudest pains:
     - runners that show online but pick up no jobs;
     - forced version deprecation;
     - disk and workspace rot;
     - no fallback to hosted runners. Discussion #20019 has **107 upvotes** ✅; the dossier's earlier ❌ "197" is refuted.
   - Setup complaints are few and draw few reactions ✅ [U7].
8. **AI agents are a real volume trend, but GitHub's official guidance points agent workloads to ephemeral runners.**
   - Agent-authored PRs grew from **~63k (2025-05-26) to ~11.45M (2026-09-25)** ✅ [T1], per a community tracker of public GitHub search counts.
   - For the Copilot cloud agent, GitHub recommends **ephemeral single-use runners via ARC or the Scale Set Client** ✅ [T2].
   - Copilot code review is **ARC-only**: "do not use non-ARC self-hosted runners" ✅ [T3].
   - Three agent workloads can target a plain repo-scoped runner on a personal account, according to their docs ✅: claude-code-action, gh-aw and the Copilot cloud agent. That RunnerKit's runners actually work with them is untested ⚠️.
9. **Supply-chain attacks moved into the CI runtime, and that raises the stakes for persistent self-hosted runners.**
   - Examples: tj-actions (23,000+ repos ✅), the TanStack OIDC-from-runner-memory chain ✅, and several 2026 worm waves ⚠️.
   - GitHub says self-hosted runners should **"almost never"** be used on public repos ✅ [G21].
   - Free runtime hardening for self-hosted runners is missing: Harden-Runner requires an Enterprise subscription for self-hosted ✅ [T14].
10. **Every proven business model here is priced for teams or per server.**
    - RunsOn: €300/yr license for commercial use ✅.
    - Actuated: $150/month for the first server and $125 for each additional one ✅.
    - Coolify: $5/month for 2 servers plus $3 per extra server ✅.
    - Dokploy: $4.50 per server per month ✅.
    - Donations raise little: prolific infrastructure maintainers have about 30 sponsors ✅.
    - **No evidence was found of solo developers paying** for runner-management tools ⚠️.
11. **A license gates distribution.**
    - RunnerKit has **no LICENSE** ✅ [R1]. homebrew-core, awesome-selfhosted, Hetzner Community tutorials and the GitHub Secure Open Source Fund all require FOSS ✅ [B7][B9][B10][B13].
    - The most accessible channel is **jonico/awesome-runners** (895★). It lists 2-star projects and has **no Hetzner or BYO entry** ✅ [B8].
12. **RunnerKit itself has no market signal, and its released binary does not work on the documented path.**
    - Adoption: **0 stars, 0 forks, 35 total downloads** across 16 releases. It was never launched and has been dormant since **2026-05-19** ✅ [R1]. Because it was never launched, that tells us nothing about demand either way.
    - The released v1.3.3 fails on the documented BYO path (password-sudo hosts). This was reproduced against the released binary ✅ [R4].
    - Any market opportunity below depends on fixing that first.

---

## 2. GitHub Actions pricing and policy (September 2026)

### 2.1 Standard GitHub-hosted runners (private repos)

| Runner | Spec | Price/min now | Before 2026-01-01 | Notes | Verif. / Source |
|---|---|---|---|---|---|
| Linux 1-core x64 (`ubuntu-slim`) | 1 vCPU / 5 GB, container on shared VM | **$0.002** | $0.002 | Hard **15-minute** job timeout, unprivileged, no Docker. GitHub says it is "not suitable for typical heavyweight CI/CD builds", so it is not a CI substitute | ✅ [G1][G10][G6] |
| Linux 2-core x64 | **2 vCPU / 8 GB** (private repos) | **$0.006** | $0.008 | The baseline for every comparison in this report | ✅ [G1][G6] |
| Linux 2-core arm64 | 2 vCPU | **$0.005** | $0.005 | Labels ubuntu-24.04-arm, 22.04-arm, 26.04-arm. Private-repo availability date (2026-01-29) is ⚠️ | ✅ [G1][G11]; date ⚠️ [G24] |
| Linux "Advanced" 2-core | n/a | **$0.006** | new SKU | `linux_2_core_advanced` | ✅ [G2] |
| Windows 2-core x64 / arm64 | 2 vCPU | **$0.010** | $0.016 (x64) | n/a | ✅ [G1][G6] |
| macOS 3/4-core | n/a | **$0.062** | $0.080 | n/a | ✅ [G1][G6] |
| Public repos | Linux **4 vCPU / 16 GB** | **free, unlimited** (standard runners) | same | Larger runners are *not* free on public repos | ✅ [G3][G2][G11] |

- **Billing granularity:** each job is rounded up to a whole minute ✅ [G1].
- **Framing:** GitHub describes the cut as "up to 39%", with the $0.002 platform charge "already included in the reduced meter price" for hosted runners ✅ [G5][G6].
- **2025 users:** GitHub says public projects used 11.5 billion free Actions minutes in 2025 ⚠️ (not re-checked).

### 2.2 Larger runners: prices, and who can buy them

| Larger runner | Price/min (before 2026-01-01) | Verif. |
|---|---|---|
| Linux x64 4 / 8 / 16 / 32 / 64 / 96-core | $0.012 ($0.016) / $0.022 ($0.032) / $0.042 ($0.064) / $0.082 / $0.162 / $0.252 | ✅ [G2][G6] |
| Linux arm64 4 / 8 / 16-core | $0.008 ($0.010) / $0.014 / $0.026 | ✅ [G2][G6] |
| Windows 4-core | $0.022 | ✅ [G2] |
| macOS 12-core / M2 Pro 5-core | $0.077 / $0.102 | ✅ [G2] |
| GPU Linux 4-core | $0.052 ($0.070) | ✅ [G2][G6] |

**Restrictions:**
- Larger runners are available **only to organizations and enterprises on the Team and Enterprise Cloud plans**. Free and Pro individuals cannot buy them at any price ✅ [G10].
- Included minutes do not apply to larger runners, and they are not free on public repos ✅ [G2].
- Static IPs and Azure private networking are also org-only ⚠️ (docs.github.com was blocked).
- **Implication:** for a solo developer, "more than 2 vCPU / 8 GB" is a *capability* only self-hosting provides. It is not a price trade-off.

### 2.3 Included quotas, concurrency and limits

| Plan | Included private-repo minutes/mo | Artifact storage | Standard-runner concurrency | Verif. |
|---|---|---|---|---|
| Free (personal) | **2,000** | 500 MB | **20** | ✅ [G3][G9] |
| Pro (personal) | **3,000** | 1 GB | **40** | ✅ [G3][G9] |
| Free for orgs | 2,000 | 500 MB | n/a | ✅ [G3] |
| Team | **3,000** | 2 GB | **60** | ✅ [G3][G9] |
| Enterprise Cloud | **50,000** | 50 GB | **500** | ✅ [G3][G9] |

**Limits and plan pricing:**
- **Cache:** 10 GB per repo ✅ [G3].
- **Concurrency:** GitHub Support can raise the limits ✅ [G9]. So "more concurrency than hosted runners" only holds for the Free, Pro and Team plans.
- **Job limits:**
  - GitHub-hosted: **6 h** per job ✅ [G9].
  - Self-hosted: **5 days** per job, **24 h** in the queue, and 35 days per workflow run ✅ [G9].
- **Plan prices:** Team is currently "$4 per user/month (first 12 months)", so its price is promotional and may rise after a year; Enterprise starts at $21 ✅ [G12].
- **Value of the Free quota:** about **$12/month** of Linux 2-core time (2,000 × $0.006) (calc ✅).

### 2.4 The postponed self-hosted "platform charge"

| Date | Event | Verif. / Source |
|---|---|---|
| 2025-12-16 | GitHub announces a **$0.002/min** "Actions cloud platform charge" on self-hosted usage in private and internal repos, starting **2026-03-01**. Public repos and GHES are exempt. Under the design, self-hosted minutes would "consume available usage based on list price", drawing down the free quota first. | ✅ [G5]; changelog ⚠️ [G24] |
| 2025-12-17 | Postponed "to take time to re-evaluate our approach". GitHub admitted it "missed the mark". The postponement thread has **263 👍, 52 ❤️, 67 comments and 71 replies**. | ✅ [G7] |
| 2026-01-01 | Hosted-runner price cut takes effect. | ✅ [G5][G6] |
| 2026-07-08 | A third-party post says the fee "has never taken effect" and no reinstatement is announced. | ⚠️ (bex.co snippet) |
| 2026-09-25 | GitHub's billing docs: Actions usage "is **free** for **self-hosted runners** and for public repositories". | ✅ [G4] |

- **Date correction ❌:** The resources page header reads "Dec 15, 2025". That is probably the page's original date (the page now also carries the postponement text), *not* the postponement date (inference ⚠️). The fee was announced on 2025-12-16 and postponed on 2025-12-17 ✅ [G5][G7].
- **GitHub's impact data** ✅ [G5]:
  - 96% of customers would see no change.
  - Of the 4% affected, 85% would see a decrease and 15% a median increase of about $13.
  - Individuals: 0.09% would see an increase (median under $2/month) and 2.8% a decrease.
- **Scenario if the fee returns as designed** (calc ⚠️, low confidence; assumes a dollar-denominated quota):
  - The Free plan's ~$12 quota covers ~6,000 self-hosted minutes, and Pro's ~$18 covers ~9,000.
  - At 20,000 self-hosted minutes a month, the fee would add about **$28/month on Free** and **$22 on Pro**.
  - At 5,000 minutes or fewer it would add **$0**.

### 2.5 Self-hosted runner lifecycle rules and version enforcement

| Rule | Detail | Verif. / Source |
|---|---|---|
| **Registration floor ("Config")** | A runner must be ≥ **2.329.0** to register or re-register. | ✅ [G13] |
| **30-day update rule ("Runtime")** | "If the runner is not updated within 30 days of an update being available, the GitHub Actions service will stop queuing jobs to it." Critical security updates pause queuing until they are applied. | ✅ [G13][G15] |
| **Auto-update satisfies the rule** | "Runners with auto-update enabled meet the 30-day requirement automatically, as long as they can reach the update service." On github.com, the runner self-updates when a job is assigned, or within a week of a release if it is idle. | ✅ [G13][G14] |
| **Enforcement dates** | GHEC with data residency: full enforcement **2026-07-31**. GHEC: full enforcement **2026-09-25**, after brownouts on 2026-08-24 through 2026-09-18. The change "applies to github.com". GHES "isn't impacted". | ✅ [G13] |
| Free / Pro / Team date | No separate row is published. The same github.com date is presumed. | ⚠️ inference |
| Observed lag before refusal | About 63–66 days after the next release, for 2.330.0 through 2.332.0 (dossier reading of issue dates). A third-party post says the API currently allows roughly 63–71 days. | ⚠️ inference; [U8] (third-party post) |
| **2026 refusal reports** | #4203 (2.329.0; logs show disableUpdate=true; 16 reactions), #4305 (2.330.0), #4392 (2.331.0), #4442 (2.332.0; 27 reactions), #4613 (2.336.0, while it was still the latest, unexplained). Per the dossier, all involve pinned or non-updating runners, mostly container, ARC or GKE images; only some reports show this explicitly. | ✅ issues [G18]; "all pinned" ⚠️ inference |
| Auto-update failure mode | #4421: an auto-update can leave a 0-byte `runsvc.sh`, so the next restart fails with 203/EXEC. | ✅ [G18] |
| Release cadence | Roughly monthly: 2.331.0 (2026-01-09) through **2.337.0 (2026-08-26)**. | ✅ [G17] |
| RunnerKit's pin | **2.334.0** (2026-04-21), three releases behind. RunnerKit never passes `--disableupdate`, so its runners should self-update. It is **probably still functional** (inference, medium confidence, not live-tested). | ✅ code [R2]; functionality ⚠️ |
| Offline auto-removal | GitHub deletes a runner that has not connected for **14 days** (**1 day** for ephemeral runners). | ✅ [G16] |
| Registration token lifetime | 1 hour. | ⚠️ (docs.github.com blocked) |
| Scope | One runner instance registers to **one repo, org or enterprise at a time**. There is **no user-level scope**, so personal accounts need one runner per repo. | ✅ [G14][G19] |
| Runner-side platform changes | The runner deprecates Node.js 20 for actions (date **2026-06-02**) and makes Node 24 the default (**2026-06-16**). Linux ARM32 is deprecated. | ✅ [G17] |
| Hosted images | `ubuntu-26.04` and `ubuntu-26.04-arm` labels exist. `ubuntu-latest` still maps to 24.04. | ✅ [G11] |

**What this means for the market.** Tools that keep auto-update on are fine. Tools and images that pin a runner version and disable auto-update (container images, some Ansible roles, ARC images) now carry a recurring 30-day maintenance burden. That burden is itself one of the measurable user pains (§5.3).

---

## 3. Self-hosting economics

### 3.1 Assumptions

- **Workload:** Linux 2-vCPU **private-repo** minutes. Public repos are free on GitHub-hosted runners, so self-hosting them saves nothing.
- **FX and tax:** 1 EUR = **1.1622 USD** (the dated ECB rate used in the dossier [H10]). VAT is excluded. Hetzner prices **include €0.50/month for primary IPv4** ⚠️ [H1][H2].
- **Hetzner new-order prices after 2026-06-15** ⚠️ [H1][H3] (net; incl. IPv4; USD):

  | Type | Spec | Net price | Incl. IPv4 | USD |
  |---|---|---|---|---|
  | CX23 | 2c / 4 GB | €5.49 | €5.99 | **$6.96** |
  | CAX11 | 2c / 4 GB, arm | €5.99 | €6.49 | $7.54 |
  | CX33 | 4c / 8 GB | €8.49 | €8.99 | $10.45 |
  | CPX22 | 2c / 4 GB AMD; RunnerKit's default | €19.49 | €19.99 | **$23.23** |
  | CPX32 | 4c / 8 GB | €35.49 | €35.99 | **$41.83** |

- **Ubicloud** ✅ [M2][M3]:
  - New accounts default to **premium-2 at $0.002/min**. **Standard-2 at $0.00125** is available on request.
  - Every account gets a **$2.50/month credit** ("equivalent to 1,250 minutes" at the premium rate).
  - 2-vCPU runners come with 8 GB of RAM.
- **Blacksmith:** $0.004/min with **3,000 free minutes per organization** ⚠️ [M7]. Organizations only.
- **Depot Developer:** $20/month including 2,000 minutes, at $0.006/min list ✅ [M1]. That overage minutes bill at $0.006 is our assumption ⚠️.
- **RunsOn:** claims about $0.0009/min of AWS compute on c8a.large ✅ (a vendor claim in its README) [M5]. A €300/year license applies to commercial use; non-commercial use is free ✅.
- **BYO existing hardware:** marginal cash cost is taken as about **$0**. Electricity, bandwidth, disk and owner time are **not quantified** ⚠️.
- **Excluded everywhere:** GitHub plan subscription fees and the time spent maintaining the runner.

### 3.2 Monthly cost by usage tier (USD, private-repo Linux 2-vCPU minutes)

| Option | 500 | 2,000 | 3,000 | 5,000 | 10,000 | 20,000 | Verif. |
|---|---|---|---|---|---|---|---|
| **GitHub-hosted, Free plan** | 0 | 0 | 6.00 | 18.00 | 48.00 | 108.00 | ✅ (calc) |
| **GitHub-hosted, Pro plan** | 0 | 0 | 0 | 12.00 | 42.00 | 102.00 | ✅ (calc) |
| GitHub-hosted arm64, Free plan | 0 | 0 | 5.00 | 15.00 | 40.00 | 90.00 | ✅ (calc) |
| **Always-on Hetzner CX23** (4 GB; *if orderable*) | 6.96 | 6.96 | 6.96 | 6.96 | 6.96 | 6.96 | ⚠️ |
| Always-on Hetzner CX33 (8 GB; *if orderable*) | 10.45 | 10.45 | 10.45 | 10.45 | 10.45 | 10.45 | ⚠️ |
| **Always-on Hetzner CPX22** (4 GB; RunnerKit's default; orderable in nbg1/hel1 but not in fsn1, RunnerKit's default location, on 2026-08-25) | 23.23 | 23.23 | 23.23 | 23.23 | 23.23 | 23.23 | ⚠️ |
| Always-on Hetzner CPX32 (8 GB; RAM parity with GitHub) | 41.83 | 41.83 | 41.83 | 41.83 | 41.83 | 41.83 | ⚠️ |
| Hetzner VM per job (CPX22 ≈ €0.032/h incl. IPv4 ≈ $0.037 per job; 10-min jobs; each job billed 1 h) | 1.85 | 7.42 | 11.13 | 18.55 | 37.10 | 74.19 | ⚠️ (calc) |
| **BYO existing hardware** (marginal) | ~0 | ~0 | ~0 | ~0 | ~0 | ~0 | ⚠️ (inference) |
| Ubicloud premium-2 (default for new accounts) | 0 | 1.50 | 3.50 | 7.50 | 17.50 | 37.50 | ✅ (calc) |
| Ubicloud standard-2 (opt-in) | 0 | 0 | 1.25 | 3.75 | 10.00 | 22.50 | ✅ (calc) |
| Blacksmith (orgs only) | 0 | 0 | 0 | 8.00 | 28.00 | 68.00 | ⚠️ (calc) |
| Depot Developer | 20.00 | 20.00 | 26.00 | 38.00 | 68.00 | 128.00 | ✅ rate / ⚠️ overage assumption |
| RunsOn (AWS compute only; + €300/yr if commercial) | 0.45 | 1.80 | 2.70 | 4.50 | 9.00 | 18.00 | ⚠️ (vendor claim) |

### 3.3 Break-even: how many minutes a month before an always-on box is cheaper

Formula: included minutes + (box cost ÷ per-minute rate). For Ubicloud, the $2.50 credit is added to the box cost.

| Always-on box ($/mo incl. IPv4) | vs GitHub Free | vs GitHub Pro | vs Ubicloud premium-2 | vs Ubicloud standard-2 | vs Blacksmith | Verif. |
|---|---|---|---|---|---|---|
| CX23 ($6.96) | **3,160** | **4,160** | 4,731 | 7,569 | 4,740 | ⚠️ (calc) |
| CAX11 ($7.54) vs GitHub arm64 ($0.005) | 3,509 | 4,509 | n/a | n/a | n/a | ⚠️ (calc) |
| CX33 ($10.45) | 3,741 | 4,741 | 6,474 | 10,359 | 5,612 | ⚠️ (calc) |
| **CPX22 ($23.23)** | **5,872** | **6,872** | 12,866 | 20,586 | 8,808 | ⚠️ (calc) |
| CPX32 ($41.83) | 8,971 | 9,971 | 22,164 | 35,462 | 13,457 | ⚠️ (calc) |

**Cross-checks and sensitivities:**
- **Other dossier estimates.** The dossier's own figures used 1 EUR ≈ 1.17 USD, excluded IPv4, and in one case omitted Ubicloud's credit: CPX22 ≈ 5,800 (Free) / 6,800 (Pro), CPX32 ≈ 8,900, ≈ 8,700 vs Blacksmith, ≈ 12,600 vs Ubicloud premium, and 18,586 vs Ubicloud standard. They agree with the table above within about ±10%, and the conclusions are the same.
- **Sensitivity to Hetzner's 2026 price rises.** CPX22 at its April 2026 price (€7.99) would break even with GitHub Free at about **3,600** min/month. At the €4.90 RunnerKit still displays, the figure is about **2,950**, or about 3,050 with IPv4 (calc ⚠️). The June repricing raised the break-even usage by about **60%** (≈3,600 → ≈5,900 min/month on Free); the paid-minute component above the free quota more than doubled (≈1,650 → ≈3,870).
- **Teams on larger runners** (Team or Enterprise orgs only). A CPX32 (4 vCPU) against GitHub's 4-core runner at $0.012/min breaks even at about **3,490** minutes, with no included minutes to subtract (calc ⚠️). This comparison does not exist for Free and Pro users, because they cannot buy larger runners ✅ [G10].

### 3.4 Caveats

1. **RAM is not like-for-like.**
   - GitHub's private 2-vCPU runner has **8 GB** ✅ [G1], and so do Ubicloud's 2-vCPU runners ✅ [M3].
   - CX23, CAX11 and CPX22 have **4 GB** ⚠️. RunnerKit's own preflight warns when MemAvailable is under **4 GiB** ✅ [R2].
   - The fair comparisons are therefore **CX33** (often out of stock) or **CPX32** ($41.83).
2. **Hetzner availability decides which box you can actually get** ⚠️ [H2][H4][H5]:
   - hetzner.com showed every CX and CAX plan as "not available" on 2026-09-03/04. The cheapest orderable plan was CPX12 at €11.99 including IPv4.
   - On 2026-08-25, **fsn1 (RunnerKit's default location) had no orderable server types**.
   - On 2026-09-24, cx33 was orderable in none of the six datacenters. The same API probe found cax11, cax21, cpx22 and cpx32 orderable in nbg1, so the hetzner.com "every CX/CAX unavailable" reading (09-04) is not uniform across datacenters.
   - One conflicting report (2026-09-25) had cx23 through cx53 provisioning fine while cpx nodes never became ready.
   - Stock varies by datacenter and possibly by account.
3. **Capacity is limited.**
   - One always-on runner can deliver at most about **43,200 min/month** (calc).
   - RunnerKit registers **one runner per repo**, so matrix legs run one after another ✅ [R2].
   - GitHub-hosted runs 20, 40 or 60 jobs in parallel (Free, Pro, Team) ✅ [G9].
   - At 20,000 min/month, a single runner is busy about 46% of the time, so peaks will queue.
4. **Performance.**
   - GitHub standard runners score about **2,269** PassMark single-thread, versus **4,299** for an m8azn instance. Both figures are ✅ as quoted in RunsOn's README, but they are a vendor's benchmark [M5].
   - Namespace (4,454), Blacksmith (4,406) and a claim that Hetzner-hosted runners have "abysmal" bandwidth to GitHub's Azure-backed cache are all ⚠️ vendor benchmarks that could not be fetched [M15].
5. **Ephemeral one-VM-per-job on Hetzner is structurally costly.**
   - Hetzner "always round[s] up the hourly usage" (a quote verified in Cyclenerd's README; ⚠️ as a third-party statement of Hetzner policy) [O7].
   - Above about **626 jobs/month**, per-job VMs cost more than an always-on CPX22 (calc ⚠️).
   - RunnerKit's 498-second cloud smoke test is mostly a 300-second destroy-verify poll ✅ [R2]. So "minutes of provisioning per job" is overstated as evidence.
6. **"BYO costs about $0" hides real costs** ✅ [R4]:
   - Owner time.
   - Disk: RunnerKit's full image needs about **4.5–5 GB** of disk and about **1.4 GB** of downloads (measured and projected from a sandbox run of the released binary).
   - Update breakage and security exposure (§6.2).
7. **A rented dedicated box is a BYO option for heavy users.** A Hetzner Server Auction AX41-NVMe at about **€37/month** (⚠️ low confidence [H9]) gives 6+ cores and 64 GB, which Free and Pro users cannot get from GitHub at any price.

### 3.5 Verdict by segment

| Segment | Cheapest sensible option | Does self-hosting win? | Basis |
|---|---|---|---|
| Solo dev, private repos, under 3k min/mo | GitHub's free quota ($0–6 on Free, $0 on Pro). Ubicloud costs $0–3.50 at its default premium tier. Blacksmith is org-only, so it is not an option for a personal account | **No.** An always-on cloud box costs about $7–23/month against $0–6 hosted | §3.2 (calc ⚠️) |
| OSS on public repos | GitHub-hosted: free, unlimited, 4 vCPU / 16 GB | **No.** GitHub also says "almost never" self-host public repos | ✅ [G3][G21] |
| Solo dev who already owns hardware (homelab, Mac mini, paid VPS) | BYO | **Yes.** Every minute above the quota is saved | inference ⚠️ |
| Needs more than 2 vCPU / 8 GB, jobs over 6 h, or private-network access | BYO or self-hosted | **Yes, on capability.** Free and Pro users cannot buy larger runners | ✅ [G10][G9] |
| Indie or small team on Team plan, 6–20k+ min/mo | Managed runner or self-hosted | **Sometimes.** Hosted costs about $18 at 6k and $102 at 20k min/mo on Team (3,000 included), so a $7–42 box ranges from a small loss or saving at 6k to about $60–95/month saved at 20k. Managed vendors compete with no ops burden | calc ⚠️ |
| Heavy user wanting a cloud box (≈100+ CI hours/month) | CPX22 or CPX32, or a dedicated box | **Yes, above about 6–9k min/mo against GitHub-hosted.** Against Ubicloud a CPX22 breaks even only at about 12.9k (premium) to 20.6k (standard) min/mo (§3.3), and Ubicloud's personal-account support is not stated | calc ⚠️ |

---

## 4. Competitive landscape

### 4.1 (a) Managed runner vendors

| Vendor | Model | 2-vCPU Linux price | Free allowance | Personal-account repos? | Funding / traction | Status | Verif. / Source |
|---|---|---|---|---|---|---|---|
| **GitHub-hosted** | First-party | $0.006 (arm $0.005) | 2,000 / 3,000 min | Yes | n/a | Incumbent | ✅ [G1][G3] |
| **Blacksmith** | Bare-metal microVMs | $0.004 x64; $0.0025 arm; $0.008 Windows; $0.08 macOS M4 | 3,000 min per org | **No** (org-only; jobs sat queued until the repo moved to an org) | $10M Series A (GV, 2025-09-18); **$45M Series B at $550M** (announced 2026-08-12); ~800 → **6,000+ customers** | Active; pitches to AI coding agents | ⚠️ all [M7][M8] |
| **Depot** | Managed runners + own CI engine | **$0.006** x64 or arm (2 vCPU / 8 GB); Windows $0.008; macOS $0.08 (Business plan only) | Developer $20/mo incl. 2,000 min; Startup $200/mo incl. 20,000 min | Setup "requires the organization owner role" | $10M Series A (Mar 2026) ⚠️; Depot CI GA 2026-03-24 ⚠️ | Active | ✅ prices [M1]; ⚠️ funding [M8] |
| **Ubicloud** | AGPL cloud + managed runners on Hetzner/Leaseweb bare metal | premium-2 **$0.002** (default); standard-2 **$0.00125**; standard-4 $0.0025; standard-8 $0.005 | $2.50/mo credit | Onboarding is a GitHub App + label change; personal-account support not stated | 12.3k★ AGPL ✅; $16M seed (Jan 2024) ⚠️ | Standard-2 rose **$0.0008 → $0.0010 (2026-05-01) → $0.00125 (2026-09-01)**; premium-2 **$0.0016 → $0.0020** (2026-09-01) | ✅ [M2][M3][M4] |
| **WarpBuild** | Managed + BYOC on hyperscalers | $0.004 x86; $0.003 arm; BYOC $0.002 + your cloud bill | n/a | Unknown | YC; ~$2M raised | Active; courted BuildJet users | ⚠️ [M13] |
| **Namespace** | Managed compute "for code and agents" | ~$0.002 in plan, $0.003 overage; Team $100/mo (100k unit-min), Business $250/mo | n/a | Unknown | **$23M** led by NEA (2026-03-23) | Active | ⚠️ [M8][M13] |
| **Cirun** | SaaS orchestration on your cloud (incl. **Hetzner**) | Per private repo: $29/mo (3 repos), $79 (10), $99 (20), $499 (100) + your cloud bill | Free for public repos | n/a | n/a | Active | ⚠️ [M13] |
| **RunsOn** | Source-available control plane in *your* AWS account | ~$0.0009/min compute (vendor claim) | Free for non-commercial use; 15-day trial | n/a (AWS account) | **1.3k★**; "2.13M jobs in one day" | Active (v3.3.2, 2026-09-16) | ✅ [M5] |
| **Actuated** | SaaS control plane + Firecracker microVMs on *your* hosts | **$150/mo first server + $125/mo each additional** (since 2026-05-07); Enterprise custom, paid annually | None | Org-focused | actuated repo 183★ | Active | ✅ [M6]; ❌ "from $250/mo" superseded (see §9.1) |
| **Tenki** | Runners + AI review + sandboxes | $0.002/core-min ($0.004 for 2-core) | $10 credit | n/a | n/a | Active | ⚠️ [M13] |
| **Latchkey** | "Self-healing" runners | $0.0025 (2 vCPU / 8 GB) | Free minutes on every plan | n/a | New in 2026 | Active | ⚠️ [M13] |
| **StarSling** | "Self-driving CI" | $0.004 | 2,000 min one-time | n/a | YC X25 | Active | ⚠️ [M13] |
| **CRACI** | EU-resident runners, SBOM per build | €0.002 per vCPU-min | Startup programme ≥100k min | n/a | Founded 2025, Helsinki | Active | Existence ✅ [M12]; price and "EU residency only on Enterprise" ⚠️ |
| **machine.dev** | GPU/CPU runners | GPU from $0.003/min (spot) | $10 credit | n/a | n/a | Active | ⚠️ [M13] |
| **Bitrise Build Hub** | Mobile/macOS runners | macOS from $0.0072/min | n/a | n/a | Established | Active | ⚠️ [M13] |
| **Buildkite hosted agents** | Not GitHub Actions runners | $0.004/vCPU-min + $30/user | n/a | n/a | Established | Active | ⚠️ [M13] |
| BuildJet / Cirrus Runners / Shipfox | n/a | n/a | n/a | n/a | n/a | **Exited** (§4.4) | mixed |

**Reading the table:**
- **Price is a commodity.** Vendors now compete on caching, observability, agent features and compliance ⚠️ (inference).
- **Two segments are crowded:** managed fast Linux runners for **organizations**, and runners in **your own AWS/GCP/Azure account** (RunsOn, WarpBuild BYOC, Sprinters, Cirun) ⚠️ (inference).
- **Vendors turned the proposed fee into marketing.** They called it a "self-hosted runner tax" (WarpBuild) and said "the GitHub Actions control plane is no longer free" (Blacksmith) ⚠️ [M14].
- **Most "best runners" comparison pages are run by vendors.** One site, runnerbench, says it grades "including the one the owner founded" ⚠️ [M14].

### 4.2 (b) Open-source and self-managed tools

| Tool | Architecture | Always-on control plane? | Persistent runners? | Hetzner / BYO fit | License | Stars / traction | Latest activity | Verif. |
|---|---|---|---|---|---|---|---|---|
| **actions/runner** (manual `config.sh` + `svc.sh`) | The official agent | No | Yes | Any Linux | MIT | 6.3k★; 430 open issues; "not taking contributions" | v2.337.0 (2026-08-26) | ✅ [O18][G17] |
| **ARC** (actions-runner-controller) | Kubernetes operator, now built on scaleset | Yes (Kubernetes) | Legacy modes "maintained by the community only" | Via kube-hetzner (3.9k★) | Apache-2.0 | **6.5k★**, 1.5k forks | 0.14.2 (2026-05-22); near-daily commits | ✅ [O2][O16] |
| **actions/scaleset** | Official Go library for building autoscalers | You build one | Ephemeral / JIT | Any VM or container | MIT | **193★** | Public preview; v0.4.0 (2026-05-05); main needs **Go 1.26.3**; breaking listener change 2026-09-15; open listener-stall bug #131 | ✅ [O1] |
| **terraform-aws-github-runner** | Lambda-based autoscaler on AWS | Yes | Ephemeral + persistent | AWS only | MIT | **3.1k★**, 746 forks | v7.11.0 (2026-08-17) | ✅ [O3] |
| **RunsOn** | ECS/Lambda on AWS | Yes | Ephemeral | AWS only | Commercial (templates MIT) | 1.3k★ | v3.3.2 (2026-09-16) | ✅ [M5] |
| **GARM** | Multi-provider server with web UI | Yes | Both; also supports Gitea | No Hetzner provider | Apache-2.0 | **407★** | v0.2.1 (2026-05-08); commits to 2026-09-21 | ✅ [O4] |
| **myoung34/docker-github-actions-runner** | Docker image | No | Both | Any Docker host (the homelab default) | GPL-3.0 | **2.5k★** (2,467), 480 forks, **66,040,018 Docker Hub pulls** | Tags track upstream (2.337.0) | ✅ [O5] |
| **MonolithProjects Ansible role** | Ansible | No | Yes | Any SSH host | MIT | **248★** | 1.28.1 (2026-07-03) | ✅ [O6] |
| macunha1 Ansible role | Ansible | No | Yes | Any SSH host | n/a | **38★** | n/a | ✅ [O6] |
| **machulav/ec2-github-runner** | In-workflow action | No | Ephemeral | AWS only | MIT | 861★, 388 forks | v2.6.1 (2026-04-10) | ✅ [O12] |
| **Fireactions** (Hostinger) | Firecracker microVM pools | Yes | Ephemeral | Bare metal, needs KVM | Apache-2.0 | 190★ | v2.0.0 | ✅ [O13] |
| ForgeMT (Cisco) | Multi-tenant AWS platform | Yes | Ephemeral | AWS only | Apache-2.0 | 212★ | Active | ✅ [O13] |
| **Tart** (now openai/tart) | Apple Silicon VMs | n/a | n/a | Mac minis | FSL-1.1-ALv2 | **6.9k★** | 2.38.0 (2026-09-23) | ✅ [O14] |
| Cilicon / Tartelet | macOS ephemeral runner apps | n/a | Ephemeral | Mac | MIT | **1.2k★** / **797★** | Last tags 2025-12-12 / 2025-06-13 (stale) | ✅ [O14] |
| gh extensions (gh-runnerctl, srz-zumix/gh-runner-kit, madkoo/gh-runner) | CLI helpers | No | Yes | Local or SSH | MIT (runnerctl) | 2★ / 0★ / 2★ | Active in 2026 | ✅ [O15] |
| mikehardy/runner-fallback-action | Picks hosted runners when self-hosted is down | No | n/a | n/a | OSS | **15★** | Active | ✅ [O17] |
| Gitea Runner / Forgejo Runner | Runners for other forges | No | Yes | Any | FOSS | Gitea Runner 1.0.0 (2026-05-05) ⚠️; Forgejo Runner v13.0.0 (2026-08-03) ⚠️ | Active | Rename act_runner → gitea-runner ✅ [T20] |

**Structural observations:**
- **Every autoscaler needs an always-on controller somewhere.** ARC runs on Kubernetes, terraform-aws on Lambda, RunsOn on ECS/Lambda, GARM on its own server, TestFlows as a service, and Zoomies as a controller with TLS and a GitHub App.
- **The in-workflow actions** (Cyclenerd, machulav) avoid a control plane. The trade-off: they are ephemeral-only, and every run pays VM boot time plus the provider's minimum billing unit.
- **Persistent installs** (raw config.sh, the Ansible roles, the Docker image) come with no diagnostics, no hygiene and no provisioning.
- These observations are inferences drawn from the ✅ facts in the table.

### 4.3 (c) Hetzner-specific tools and closest rivals, compared with RunnerKit

| | **Zoomies** (eyupio/zoomies) | **TestFlows Hetzner Runners** | **Cyclenerd hcloud-github-runner** | **soulteary/runner-fleet** | **solutionforest/EPAR** | Flinner/gh-runnerctl | **RunnerKit** (for reference) |
|---|---|---|---|---|---|---|---|
| Pitch | "Lightweight, self-hosted GitHub Actions runner fleet controller" | Always-on autoscaler for Hetzner | "New Workflow, New Server" | Web UI managing several runners on one machine | Warm pool of disposable runners on hosts you own | Declarative multi-host runner pools over SSH | Laptop CLI that sets up a BYO SSH host or one Hetzner VM |
| Runner model | **Ephemeral by default; persistent not supported** | Ephemeral, one server per job; recycling (opt-in no-rebuild mode, May 2026) | Ephemeral, one VM per workflow; "a pool of persistent runners is outside the scope" | Persistent, several per host | Ephemeral, each job in a Docker Sandboxes microVM | Persistent (systemd) | Persistent default; single-job ephemeral |
| Control plane | Controller + SQLite + web UI + GitHub App + TLS; agents are **outbound-only** | Python service (systemd or on a Hetzner VM); "no Webhooks … no need to setup any GitHub application" | None; runs inside the workflow | On-host service | On-host | None | None |
| Provisioning | Your hosts; also ships cloud-marketplace deploy templates | Hetzner (x64 + arm64; label-driven type, image and location) | Hetzner; defaults cx23 / nbg1 / ubuntu-24.04; needs an admin PAT stored as a repo secret | None | None (Linux/macOS/Windows hosts) | None | Hetzner (x64 only in practice) + BYO |
| Isolation | Container per job (Docker or Podman required) | VM per job | VM per workflow | None | microVM per job | None | None on BYO |
| Health / self-heal | Web UI | Prometheus, dashboards, cost estimates | n/a | **Self-healing auto-start, config-drift repair, Prometheus, health checks** | Warm pool | n/a | `doctor` / `recover` on demand (no watchdog) |
| Caching | n/a | **Persistent cache volumes (up to 16 × 10 TB)** | n/a | n/a | n/a | n/a | Warm host (persistent) |
| License | AGPL-3.0-only | Apache-2.0 | Apache-2.0 | MIT | n/a | MIT | **None** |
| Traction | **51★, 43 forks, 1,785 commits** | **101★**, 15 forks; used by Altinity for ClickHouse; pip-distributed | **182★**, 23 forks; **16 dependent repos** (6 of 12 visible are user-owned); on the Marketplace | **22★**; v1.9.0; created 2026-02-17 | **13★** | **2★**; listed on awesome-runners | **0★**, 35 downloads |
| Release pace | **6 releases in 13 days**, v1.0.0 (2026-09-13) → v1.3.2 (2026-09-25), after 3 RC tags from 2026-09-10 | Runner bumped to 2.337.0 on 2026-09-21; datacenter-deprecation fix on 2026-07-02 | v1.4.1 (2026-06-11) | Active | n/a | Active | Last release 2026-05-19 |
| Stated savings | n/a | "Up to 33× cheaper" | 97.16–98.75% vs GitHub (vendor claim) | n/a | n/a | n/a | n/a |
| Threat to RunnerKit | **High on BYO**: same "one Go binary, no K8s, one-line installer" pitch | **High on Hetzner**: already has autoscaling and warm caches | **High on Hetzner ephemeral**: zero install | **Medium**: already covers the "watchdog" and "several runners per host" gaps | **Medium**: covers "clean ephemeral on BYO" | Low (tiny), but near-identical architecture | n/a |
| Verif. | ✅ [O9] | ✅ [O8] | ✅ [O7] | ✅ [O10] | ✅ [O11] | ✅ [O15] | ✅ [R1] |

**Corrections applied:**
- ❌ "Persistent warm-cache runners are uncontested on Hetzner." TestFlows already offers persistent cache volumes and a no-rebuild recycle mode ✅ [O8]. What RunnerKit still has that TestFlows lacks: **no always-on service**, plus BYO diagnostics.
- ❌ "Zoomies has no cloud path." It ships cloud-marketplace deploy templates ✅ [O9].
- The other Zoomies facts (ephemeral-only, no built-in VM provisioning) are ✅.

**Smaller Hetzner tools** ✅ [O20]:
- stonemaster/hetzner-github-runner: 14★, a Marketplace action.
- louisgundelwein/runner-autoscaler: 0★. Webhook + JIT; reuses a VM within its billed hour; needs an HTTPS endpoint.

### 4.4 (d) Market consolidation and structural events

| Date | Event | Why it matters | Verif. / Source |
|---|---|---|---|
| 2025-09-18 | Blacksmith raises a $10M Series A | Capital enters managed runners | ⚠️ [M8] |
| 2025-10-28 | Copilot coding agent starts supporting self-hosted runners | Opens agent workloads to self-hosting | ⚠️ [G24] (cited in #182419 [U16]) |
| 2025-11-26 | Zig moves to Codeberg, citing Actions "vibe-scheduling" | Values-driven exits from GitHub | ✅ [T16] |
| 2025-12-16 / 17 | $0.002/min self-hosted fee announced, then postponed | Fee risk is still live | ✅ [G5][G7] |
| 2026-01-01 | GitHub-hosted prices cut by up to 39% | Shrinks the cost case for self-hosting | ✅ [G5][G6] |
| 2026-01-29 | arm64 standard runners reach private repos | Removes one reason for third-party runners | ⚠️ date [G24]; capability ✅ [G11] |
| 2026-02-03 / 05 | actions/scaleset open-sourced (first commit 2026-02-03; v0.1.0 tagged 2026-02-05) | Autoscaling without Kubernetes becomes commoditized | ✅ [O1][G25] |
| 2026-02-06 → 03-31 | BuildJet announces its shutdown and stops running jobs, saying GitHub "largely closed" the gap | Standalone runner vendors are fragile | ⚠️ [M9] |
| 2026-03-19 | ARC 0.14.0 moves onto the scaleset client and adds multi-label scale sets | Official tooling converges on scale sets | ✅ [O2] |
| 2026-03 | Namespace raises $23M (2026-03-23); Depot raises a $10M Series A; Depot CI GA (2026-03-24) | Money moves to "compute for agents" | ⚠️ [M8] |
| 2026-03-26 | Custom images for GitHub-hosted runners reach GA | Weakens the image-parity argument for self-hosting | ⚠️ date [G24] |
| 2026-04-01 | Hetzner raises cloud prices about 30–37% for new and existing servers | Cloud cost thesis erodes | ⚠️ [H1][H9] |
| 2026-04-07 | Cirrus Labs joins OpenAI; Tart moves to openai/tart under FSL | macOS tooling changes owners | ⚠️ date; Tart ✅ [O14] |
| 2026-05-01 | Ubicloud standard-2: $0.0008 → $0.0010 | Supply-side cost pressure | ✅ [M2] |
| 2026-05-07 | Actuated switches to per-server pricing | Price ceiling for a managed control plane over BYO hosts | ✅ [M6] |
| by 2026-05 | Shipfox pivots to an "AI Software Factory" | Another runner vendor exits | ✅ repo [M11]; timing ⚠️ |
| 2026-06-01 | Cirrus CI shuts down; Cirrus Runners stop taking new customers | Leaves a gap in managed macOS | ✅ [M10] |
| 2026-06-01 | Copilot code review starts consuming Actions minutes on private repos | New demand for runner minutes | Consumption ✅ [T17]; date ⚠️ [G24] |
| 2026-06 | Daytona moves core development to a private codebase | Agent-sandbox OSS closes up | ✅ [T19] |
| 2026-06-15 | Hetzner reprices new orders and rescales (CPX22 €7.99 → €19.49) | RunnerKit's default becomes poor value | ⚠️ [H1][H3] |
| 2026-07-01 | Hetzner API removes `server.datacenter` | API churn for Hetzner tooling (TestFlows had to ship a fix) | ✅ [H6][O8] |
| 2026-08-12 | Blacksmith announces a $45M Series B at a $550M valuation | Capital concentrates | ⚠️ [M8] |
| 2026-09-01 | Ubicloud standard-2 → $0.00125; premium-2 → $0.0020 | Second +25% rise | ✅ [M2] |
| 2026-09-13 | Zoomies v1.0.0 | Direct OSS rival appears | ✅ [O9] |
| 2026-09-25 | GHEC runner-version enforcement reaches full strength | Pinned runners break | ✅ [G13] |
| 2026-10-01 (upcoming) | Hetzner `/v1/datacenters` starts returning HTTP 410 | Breaks old hcloud-go clients | ✅ [H6] |
| 2026-11-02 (upcoming) | Default policy blocking `pull_request_target` enforced for affected public repos | Security tightening | ✅ [G22] |
| n/a (date not captured) | Earthly (12.1k★) stops maintenance after shutting down its cloud | Stars do not make a CI business viable | ✅ [B5] |

**Reading (inference ⚠️):**
- Venture money is going to **agent and sandbox compute for teams**.
- Standalone runner businesses are **exiting**. "Your CI shouldn't depend on a startup's runway" is a credible message for self-owned tools.
- GitHub is steadily removing the reasons third-party runners exist: arm64 standard runners, custom images, cheaper minutes and scaleset.

A note on Hetzner's price rises: two in 2026 (April 1 and June 15) are corroborated by several secondary sources. A third rise, claimed by one research stream, remains ⚠️ unconfirmed.

---

## 5. Demand and the voice of the user

### 5.1 How many individuals self-host?

**Estimate: about 10⁴ individual self-hosters, possibly up to 10⁵** ⚠️. This is an order-of-magnitude inference, medium confidence, not a measurement. The evidence behind it:

| Signal | Value | Caveat | Verif. / Source |
|---|---|---|---|
| Individuals who would pay more under the self-hosted fee | 0.09% of Free/Pro users with private-repo usage | Measures who would pay more, not who self-hosts | ✅ [G5] |
| Public commits in 2026 mentioning "self-hosted runner" | **95,266** | Noisy (tutorials, bots) | ✅ [U6] |
| Owner type in a sample of 87 of those commits | **54 user-owned (62%)** / 33 org-owned; about 9 of the 54 are tutorial or course repos | Best-match order, small sample | ✅ [U6] |
| Environment labels seen in that sample | VPS, home-server, talos-homelab, LXC, Windows, Debian, Azure, AWS; **no Hetzner or arm64** | Qualitative | ✅ [U6] |
| myoung34 Docker image pulls | **66,040,018** lifetime (since 2019) | Includes CI and automated pulls; not unique users | ✅ [O5] |
| Repos depending on the Cyclenerd Hetzner action | **16** (6 of the 12 visible are user-owned) | Niche | ✅ [O7] |
| 2026 commits saying they went "back to GitHub-hosted" | **575** | Churn signal | ✅ [U6] |

### 5.2 Who self-hosts, and why (ranked by evidence)

| # | Reason | Evidence | Verif. |
|---|---|---|---|
| 1 | **Ran out of free minutes on a private repo** | 4,612 commits in 2026 mention "self-hosted runner" plus "minutes" [U6]. But there is an easy way out: one user moved to self-hosted, then made the repo public **9 minutes later**, because "public repositories get free, unlimited GitHub-hosted Actions" [U12]. | ✅ |
| 2 | **Deploy agent or private-network access** (runner on the same VPS or home server as the app) | Commit sample labels; commits such as "Make CD manual while the self-hosted runner is offline" | ✅ (qualitative) [U6] |
| 3 | **Special hardware** (Apple Silicon, ARM, bigger machines) | The top repos under the `self-hosted-runner` topic are Cilicon (1.2k★) and Tartelet (797★) [O19]. Correction: ❌ issue #805 "Add ARM + MacOS target" (253 reactions) is **closed**. That demand was met years ago, so it is not current demand [U15]. | ✅ |
| 4 | **Principle, compliance and privacy** | "How can they justify per-minute charges when it's MY machine that runs it?" [G8]. Compliance and privacy needs, and volunteer projects on donated hardware, appear in [G7][G8]. | ✅ quotes; counts vary between page renders (65+ vs 72 upvotes) |
| 5 | **Speed or cost versus hosted** | "Insanely overpriced and underpowered" [G8]. The Cyclenerd README claims 97–98.75% savings (a vendor claim) [O7]. | ✅ quotes |
| 6 | **Platform discontent and reliability** | Zig's exit ✅ [T16]. The ARC incident thread #204152 (24 upvotes, 15 comments + 36 replies) shows ARC pods stuck idle for about 4 hours ✅ [U9]. Actions uptime of 99.33% over 90 days and "57 Actions outages in 12 months" are ⚠️ unverifiable [U10]. | mixed |

**Caveat: self-hosting does not isolate you from GitHub's control plane** ✅ [U9]. Billing locks and control-plane failures in August and September 2026 blocked Actions whatever the runner type:
- #208693, "Private repository workflows fail before any job or runner is created" (2026-09-24);
- #208250, "Billing Lock not removed";
- #205214.

**The fee backlash was about principle and org bills, not solo developers:**
- The postponement thread had 263 👍 ✅ [G7].
- Reported bills of about $140, $1.8k and about $3.5k per month were all org-scale ⚠️ (quotes not re-verified individually).
- At least 7 Hacker News submissions ⚠️ (HN was blocked).

### 5.3 Top pains, ranked, with evidence counts

**Ranking method:** evidence volume weighted by fit for solo developers. The weighting is the researchers' judgement ⚠️. The "RunnerKit today" column comes from the code dossier.

| Rank | Pain | Evidence (counts) | Verif. | RunnerKit today |
|---|---|---|---|---|
| **1** | **Runner shows online or idle but picks up no jobs, loses contact, or stays offline for weeks** | #120813 (2024-04-23; still unanswered): **33 comments + 70 replies**; its 46 upvotes are unconfirmed ⚠️ [U1].<br>#3609: **141 comments**, but it is a Windows runner on EC2; its 92 👍 are ⚠️ [U2].<br>#3857 "Runner not found": comes from the Linux-kernel BPF CI running at org scale; its 116 👍 are ⚠️ [U2].<br>**16** title-matched actions/runner issues in 2026 [U7].<br>**19** "offline" and **53** "queued" community discussions in 2026 (full-text, noisy) [U14].<br>**1,714** commits in 2026 mention "self-hosted runner" plus "offline"; **12 of 13** sampled were user-owned [U6]. | ✅ except where marked | Diagnosis and fixes run on demand only (`status`, `doctor`, `recover`). No watchdog. `logs` queries the wrong systemd unit (a P1 defect). |
| **2** | **Forced version deprecation and update drift** | 2026: **4** issues with "deprecated" in the title, **45** reactions in total. #4442 (**27**) is the 2nd most-reacted of **137** issues filed in 2026; #4203 has **16** [U7][G18].<br>#3767 (2025; **22** 👍) [G18].<br>#2708: update leftovers over 1 GB, **22** 👍, open [U3].<br>Community #206494: "There is no published deprecation schedule" [U8].<br>Correction: "often without warning" is overstated. GitHub did announce minimum versions (e.g., 2025-12-12), but notices are inconsistent ✅. | ✅ | The `doctor` check `runner_version_stale` compares against the **bundled pin** only, never the real version on the host or upstream. Auto-update stays on. |
| **3** | **Disk, Docker and workspace rot; root-owned files** | #434 (open since **2020-04-17**; root-owned files from Docker steps break checkout) ✅. Its "132 👍 / 62 comments" did not render ⚠️ [U3].<br>**11** disk- or workspace-titled actions/runner issues in 2026 (about 9 genuine, about 2 reactions) [U7].<br>**3,432** commits mention "self-hosted runner" plus "disk" (noisy) [U6].<br>**RunnerKit's own maintainer's host hit df=0 on 2026-05-26** [R3]. | ✅ except where marked | Preflight checks for 2 GiB free once; there is no prune, cleanup or disk-pressure finding. |
| **4** | **Runner auto-deleted after 14 days offline** (laptops, homelabs) | Docs rule ✅ [G16]. #58146, "Stop automatically deleting inactive runners": **43** upvotes; a GitHub staff reply says it "still can't be forever" ✅ [U5]. | ✅ | `recover --reregister` exists but is manual. |
| **5** | **No automatic fallback to hosted runners when self-hosted is down** | #20019 (2022-07-05): **107 upvotes, 24 comments + 25 replies** ✅. ❌ The dossier's earlier "197 upvotes / 49 comments" is refuted. GitHub (2024-08-16): "we do not currently have plans to support this as a first-party feature" ✅ [U4]. Workaround: runner-fallback-action (15★) ✅ [O17]. #50926 (29 votes) ⚠️. | ✅ | Not covered. |
| **6** | **Security of persistent or public-repo runners** | GitHub: "We recommend that you only use self-hosted runners with private repositories" ✅ [G23]. Shai-Hulud installed rogue runners as backdoors (2025-11-24) ⚠️ [T13][T21]. Gato-X offers runner-takeover tooling ✅ [T15]. | mixed | Persistent vs ephemeral gating plus a public-repo acknowledgement exist, but the typed acknowledgement fails in a real TTY (P1-2 in CODEBASE-ASSESSMENT), so users are pushed to `--yes`. The "scoped" sudoers is effectively NOPASSWD ALL. No audit for unknown runners. |
| **7** | **Setup and registration complexity** | About **8** genuine install/registration issues on actions/runner in 2026, with about **6** reactions [U7].<br>About **8** auth/registration issues on myoung34 (2025–26), with about **1** reaction [U13].<br>A Hacker News comment calls the process "unnecessarily byzantine" ⚠️ [U11]. | ✅ counts | This is RunnerKit's core job, but the **released binary fails on password-sudo hosts** [R4]. |
| **8** | **One machine for all my personal repos** | #179202 ("i want the little machine on my desk to be handle all of my personal projects"): **2 upvotes, 7 comments + 6 replies**. A precise need, not a mass complaint ✅ [G19]. | ✅ | Designed for it (multi-repo on one BYO host), but it inherits the broken BYO path on password-sudo hosts, and all repos share one Unix user. |
| n/a | OOM on small VMs | ❌ The earlier evidence (#3724) was a GitHub-hosted **32-core larger runner**, closed as stale. **No verified user-voice evidence was found.** The pain is still plausible for 4 GB boxes ⚠️ [U15]. | ❌ / ⚠️ | Heuristics exist (`doctor --deep`). |
| n/a | Zig-cited `safe_sleep.sh` hang (#3792) | Now **closed and fixed**, so this pain is resolved ✅ [U15]. | ✅ | n/a |

**Bottom line.** For individuals, the pain is **keeping a runner alive**, not installing one. Setup draws the fewest reactions of all the pain categories ✅ [U7][U13].

### 5.4 What individuals use today

| Approach | Evidence | Typical user | Verif. |
|---|---|---|---|
| GitHub's copy-paste `config.sh` / `svc.sh` on a VPS, home server or Windows box, often the app's own host acting as a deploy agent | Commit sample | Most individual self-hosters | ⚠️ (inference from sample) [U6] |
| myoung34 Docker image | 2.5k★, 66M pulls | Homelab and Docker users | ✅ [O5] |
| Ansible roles | 248★ / 38★ | Ops-savvy users | ✅ [O6] |
| Cyclenerd action (Hetzner, per job) | 182★, 16 dependents | Hetzner users | ✅ [O7] |
| ARC on k3s | GitHub community answers point Copilot-agent self-hosters to ARC ("k3s works too") | Kubernetes-literate users | ✅ [U16] |
| **Exit: make the repo public, or go back to hosted** | 575 "back to GitHub-hosted" commits; the 9-minute public switch | Anyone who hits the minutes limit | ✅ [U6][U12] |

### 5.5 Willingness to pay

- **Proven willingness to pay sits with teams.** RunsOn (€300/yr, commercial use) ✅ and Actuated ($150+/server/month) ✅.
- **No evidence was found of solo developers paying** for runner-management tools ⚠️ (an absence finding from the user-voice stream). The one "pay extra" complaint in #20019 comes from a team context [U4].
- Individuals are highly price-sensitive. A median increase under $2/month for 0.09% of them still drew heavy outrage ✅ [G5][G7].

---

## 6. Trends

### 6.1 AI agents and CI

**Volume** (community tracker counting public GitHub search results by branch prefix or bot author; cumulative) ✅ [T1]:

| Date | Agent-authored PRs (cumulative) |
|---|---|
| 2025-05-26 | 62,940 |
| 2026-01-01 | 3,998,020 |
| 2026-08-01 | 9,274,136 |
| 2026-09-01 | 10,368,626 (≈1.09M new in August) |
| 2026-09-25 | **11,453,997** |

By agent, on 2026-09-25:

| Agent | PRs | Note |
|---|---|---|
| Codex | 7,312,216 | Includes non-Action usage |
| Copilot | 2,145,465 | 1,550,965 merged |
| Cursor | 1,395,788 | Up from 527,204 on 2026-07-01 |
| Jules | 328,636 | n/a |
| Devin | 264,020 | n/a |

**What GitHub officially supports** when an agent workload runs on a personal-account repo with a **repo-scoped, non-ARC** runner (the kind RunnerKit creates):

| Workload | Works on a personal repo? | Runner requirements (official) | Fit with RunnerKit today | Verif. / Source |
|---|---|---|---|---|
| **Copilot cloud agent** (renamed from coding agent) | **Yes.** Available on all paid Copilot plans (incl. Pro, Pro+, Max) in all repos except EMU; `runs-on` is set in `copilot-setup-steps.yml` | Ubuntu x64 or Windows x64 only (**no ARM**). GitHub **recommends** "ephemeral, single-use runners… Most customers set this up using ARC or the Runner Scale Set Client"; this is a recommendation, not a requirement. The **built-in firewall must be disabled** on self-hosted runners, or the agent is blocked. `timeout-minutes` ≤ **59**. Pro users need an extra egress host, `api.individual.githubcopilot.com`. Org owners can set a default runner. | Probably routable on Ubuntu x64 (untested ⚠️). A persistent runner goes against GitHub's guidance; single-job ephemeral dies after one session. | ✅ [T2][T4][T5] |
| **Copilot code review** | **No** (not supported for non-ARC runners) | "ARC is the only officially supported solution for self-hosting Copilot code review. For security reasons, do not use non-ARC self-hosted runners." Ubuntu x64 only. Consumes **Actions minutes on private repos** plus AI credits. If hosted runners are disabled, review falls back to a limited non-agentic mode. | Not possible short of a GitHub policy change (a scaleset-based runner still counts as non-ARC). | ✅ [T3][T17]; start date 2026-06-01 ⚠️ |
| **GitHub Agentic Workflows (gh-aw)**, preview | **Yes.** `runs-on` accepts a string, an array or a group | **Linux with Docker only**. The runner user must be in the docker group. Egress needed to api.githubcopilot.com, github.com and ghcr.io. The Copilot CLI install uses sudo unless `--rootless`. Engines: Copilot, Claude Code, Codex, Gemini, Pi. Versions ≥ 0.83.3 and < 0.85.4 were retired after a security issue. | Likely on Ubuntu x64, *if* the docker-group bug is fixed and the Copilot CLI is installed with `--rootless` (RunnerKit's job-user sudo is opt-in and limited to package managers). gh-aw's docs conflict on whether its egress firewall needs sudo ⚠️. | ✅ [T6]; 5.2k★ ✅ |
| **anthropics/claude-code-action** | **Yes**, and the best fit | A composite action that installs Bun. It "executes entirely on your own GitHub runner" and publishes no self-hosted requirements. Optional `sudo apt-get` step. **Separate HOME per job needed**: #1688 (2026-08-17) shows settings leaking between jobs on shared-HOME runners. | Likely works; shared HOME across repos is a risk. | ✅ [T7]; **8.9k★, 22,928 dependent repos** ✅ |
| **openai/codex-action** | Only on **disposable** hosts | Needs passwordless sudo, `/usr/bin/setpriv` and unprivileged user namespaces. The default drop-sudo mode is irreversible and "can outlive a job on reused self-hosted runners". #160: drop-sudo breaks D-Bus and systemd-resolved. | **No.** It runs `sudo chmod`/`chown` unconditionally, and it would damage a persistent host. | ✅ [T8]; 1.2k★ ✅ |
| **Scale-set runners on personal accounts (actions/scaleset)** | Supported **in code** (owner/repo scope, PAT sent as a Bearer token) | Unproven on user-owned repos: upstream e2e tests run only against an org-owned repo. Labels conflict: the docs say scale sets "can only have one label", while the README and the self-hosted-runners page allow several. Main requires Go 1.26.3. | A spike is needed. | ✅ [O1][G20][G14] |

**GitHub's own direction:**
- The scale-set client roadmap item explicitly targets **self-hosted Dependabot and Copilot coding agent** workloads ✅ [T10].
- GitHub recommends autoscaling with ephemeral runners: "autoscaling with persistent self-hosted runners is not recommended" ✅ [G21].

**Agent compute is also moving outside Actions:**
- **Copilot cloud sandboxes** (public preview): $0.000024 per compute-second, $0.000003 per GiB-second and $0.005 per GiB-month ✅ [T9]. ❌ The "$10/month entitlement" ended with July 2026, so all usage is now billed ✅.
- **Cursor** has "private workers" ✅ [T19].
- **E2B** (14k★) ✅. **Daytona** (71.7k★) went private in June 2026 ✅ [T19].
- Blacksmith, Namespace and Depot all market "compute for agents" ⚠️ [M8].

**Distribution through agents** ✅ [T18]:
- Agent Skills are an open standard, and `gh skill` (public preview; gh ≥ 2.90.0) installs them from GitHub repos.
- GitHub's official MCP server has **no** tools for managing self-hosted runners.
- No runner-management MCP server was found ⚠️ (an absence claim).
- Dokploy's MCP server has 386★ ✅ [B2]. Actuated's agent-skills repo has 1★ ✅ [B14], so it is a vendor experiment, not proof that the channel works.

### 6.2 Supply-chain security

| Date | Incident | Mechanism | Scale | Verif. / Source |
|---|---|---|---|---|
| 2025-03-14/15 | tj-actions/changed-files (CVE-2025-30066, CVSS 8.6) | Moved tags; memory scan of Runner.Worker; secrets dumped into logs | **23,000+ repos** | ✅ [T11] |
| 2025-09-14 / 2025-11-24 | Shai-Hulud 1 and "Second Coming" | npm worm; **rogue self-hosted runners as persistent backdoors** | 517+ / 1,100+ packages | ⚠️ [T13] |
| 2026-02-17 → 04-22 | SANDWORM_MODE; axios RAT; Bitwarden CLI via the Checkmarx action | Workflow poisoning; a compromised action | n/a | ⚠️ [T13] |
| **2026-05-11** | **TanStack** (GHSA-g7cv-rxg3-hmpx, CVSS 9.6) | `pull_request_target` Pwn Request + fork↔base cache poisoning + **OIDC token pulled from runner memory** | 84 versions across 42 packages | ✅ [T12] |
| 2026-05-18 | Nx Console (CVE-2026-48027) | Cascade from the TanStack breach | Live about 18–36 minutes | ✅ [T12] |
| 2026-05-18 | Megalodon | Malicious workflows injected using stolen PATs | **5,561 repos** in 6 hours | ⚠️ [T13] |
| 2026-05-22 → 06-17 | TrapDoor/TeamPCP (drops CLAUDE.md and .cursorrules droppers); Laravel-Lang (700+ tags rewritten); Mastra (141 packages) | Poisoning of agent config files and dependencies | n/a | ⚠️ [T13] |
| 2026-06-01 | Miasma | Reads `/proc/<pid>/mem` of Runner.Worker | n/a | ⚠️ [T13] |
| 2026-08-04 | keyv/ChainDrop | Poisoned releases **with valid npm provenance** | **444 packages** / 2,236 versions | ⚠️ [T13] |

The tracker reported no new wave between Aug 4 and 2026-08-21. September 2026 could not be checked ⚠️.

**GitHub's response** ✅ [G21][G22]:
- A policy option requiring actions to be **pinned to a full commit SHA**.
- **Immutable releases** for action publishers.
- A **read-only cache** for `pull_request_target`. Workflows can opt out with a write-capable `cache-mode`.
- A **default policy blocking `pull_request_target`** in public repos, enforced on **2026-11-02** for affected repos that used the default policy before GA.
- Guidance: self-hosted runners "should almost never be used for public repositories" and "can be persistently compromised".

**Tooling gap** ✅ [T14][T2][T15]:
- Harden-Runner: "Self-hosted runners require an Enterprise subscription, even for public repositories."
- Copilot's firewall "is not compatible with self-hosted runners".
- Gato-X offers runner-on-runner takeover and can hold persistent runners for up to 5 days.
- Docker group membership is root-equivalent ✅ (standard Linux knowledge). RunnerKit's image setup *tries* to add the runner user to the docker group, but an ordering bug currently prevents it [R2][R4]. Fixing that bug will create this exposure.
- **Inference ⚠️:** persistent hosts with Docker access are prime persistence targets. Free egress allowlisting and rogue-runner auditing for solo self-hosters is an **unserved** need.

### 6.3 Platform shifts

- **Forgejo and Codeberg:**
  - Zig migrated on 2025-11-26 ✅ [T16]. Its GitHub repo's 43.3k★ is ⚠️ not re-checked.
  - Codeberg had over 300k repos and over 200k accounts as of Nov 2025 ⚠️ [T20].
  - Bloggers in 2026 describe running a Forgejo runner on Hetzner, and Codeberg reportedly requires approval for Actions ⚠️.
  - Gitea's act_runner was renamed gitea-runner ✅ (via GARM's release notes). Gitea Runner 1.0.0 (2026-05-05) and Forgejo Runner v13.0.0 (2026-08-03) are ⚠️.
  - **Size:** small and values-driven. No usage statistics could be retrieved.
- **arm64:**
  - GitHub arm64 runners cost $0.005 vs $0.006 for x64 ✅ [G1].
  - GitHub's AI agents are **x64-only** ✅ [T2][T3].
  - Hetzner CAX stock is volatile, and Arm prices rose more than Intel in June (CAX21 €10.49 vs CX33 €8.49) ⚠️ [H10].
  - RunnerKit's arm64 support is broken in code: amd64 is hard-coded in 6 places ✅ [R2].
- **Ubuntu 26.04 and Node:**
  - Hosted `ubuntu-26.04` and `-arm` labels exist; `ubuntu-latest` still maps to 24.04 ✅ [G11]. Zoomies' runner images already include 26.04 ✅ [O9].
  - The runner deprecates Node 20 for actions and made Node 24 the default in June 2026 ✅ [G17]. RunnerKit's image setup still installs Node 20 ✅ [R2].
- **Hetzner API churn:** `server.datacenter` removed on 2026-07-01; `/v1/datacenters` returns 410 after 2026-10-01 ✅ [H6].
- **GitHub-hosted images get better:** custom images reached GA and can now be layered ⚠️ [G24]. That erodes "image parity" as a reason to self-host.

### 6.4 EU sovereignty (low confidence)

**What exists:**
- **CRACI** (Helsinki, 2025): EU-resident managed runners, a VM per build and an SBOM per build ✅ [M12]. Whether EU residency is Enterprise-only is ⚠️ unconfirmed, and the one source presents it as a core feature.
- **GitHub's own EU data residency** is limited to Enterprise Cloud on ghe.com, expanded to EFTA countries from 2026-05-01 ⚠️ [G24].
- Blacksmith reportedly has an EU fleet (per a competitor's page), Cirun offers GCP europe-west1, and Ubicloud runs in Germany and Finland ⚠️.

**What is missing:**
- **No data on market size** was obtainable. hetzner.com, EU news sites and codeberg.org were blocked.
- **Inference ⚠️:** nobody offers a cheap, self-serve "runner on your own EU box with no third-party control plane" for individuals. The demand for it is unmeasured.

---

## 7. Go-to-market and business-model analogues

### 7.1 Analogues

| Project | Category | License | How it makes money | Price points | Traction | Lesson | Verif. / Source |
|---|---|---|---|---|---|---|---|
| **Coolify** | Self-host PaaS | Apache-2.0, "no feature behind the paywall" | Hosted control plane over your own servers, plus sponsorships from infrastructure providers, including a **co-branded Hetzner referral link** (htznr.li/CoolifyXHetzner) | **$5/mo incl. 2 servers + $3/mo per extra server** | **62.3k★**, 5.5k forks; sponsors include Hetzner, Hostinger, Ubicloud, Blacksmith, Contabo; founder's GitHub Sponsors page: 31 sponsors; 5+ Hetzner Community tutorials | The closest analogue for the audience: free OSS, a cheap per-server hosted add-on, and provider partnerships | ✅ [B1] |
| **Dokploy** | Self-host PaaS (open core) | OSS core + `/proprietary` under the Dokploy Source Available License | Per-server cloud plans + an enterprise upsell (SSO/SAML, SCIM, audit logs, RBAC) | **Hobby $4.50/server/mo** ($3.60 annual); Startup from $15/mo (3 servers) + $4.50 per server; Enterprise custom | **37.5k★**, 3.0k forks; MCP server **386★** with 508 tools | Open core with enterprise features | ✅ [B2] |
| **Dokku** | Self-host PaaS | MIT | Dokku Pro: commercial web UI/API with an online license check (price undisclosed); OpenCollective and Patreon | n/a | **32.2k★**; available as a DigitalOcean 1-Click | "Free CLI, paid UI/API add-on" | ✅ [B3] |
| CapRover | Self-host PaaS | OSS | Donations only (Open Collective) | n/a | 15.2k★ ⚠️ | Donations alone rarely fund a team | ⚠️ [B4] |
| Kamal | SSH deploy CLI | MIT | None; backed by its employer (37signals) | n/a | 14.6k★ ⚠️; a Hetzner tutorial exists | A laptop CLI that SSHes into servers grew through a strong sponsor and audience | ⚠️ stars [B4] |
| **RunsOn** | Runners in your AWS account | Source-available commercial (templates MIT) | Flat license for commercial use; sponsorship buys access to the source | **€300/yr** commercial; free for non-commercial use; 15-day trial; Sponsorship €1,500/yr ⚠️; tier-based licensing coming (renewals from 2027) ⚠️ | **1.3k★**; "2.13M jobs in one day" | A license for commercial use works for teams | ✅ [M5] |
| **Actuated** | SaaS control plane over customer hosts | Commercial | Per-server subscription | **$150/mo first server + $125/mo each additional** (1–10 servers); Enterprise custom, paid annually | 183★; >1.5M CI minutes donated to CNCF by May 2024 ⚠️; agent-skills repo 1★ | The price ceiling for a managed control plane over BYO hosts. Team-only. | ✅ [M6][B14] |
| **Ubicloud** | AGPL cloud + managed runners | AGPL-3.0 | Managed per-minute service | $0.002 / $0.00125 per minute; $2.50 credit | 12.3k★; sponsors Coolify | Open source + managed service on Hetzner-class hardware | ✅ [M2][M3][M4] |
| **Earthly** | Build tool + CI cloud | MPL-2.0 | The cloud was discontinued | n/a | 12.1k★; "no longer actively maintained" | Stars do not make a CI business viable | ✅ [B5] |
| Donation baselines | n/a | n/a | GitHub Sponsors | Alex Ellis: tiers from $10 to $1,000 | **32 sponsors against a goal of 100**; Coolify founder: 31 | Sponsorware yields tens of sponsors, not a salary | ✅ [B6] |
| Hostinger / Fireactions | Hosting provider building runner tooling | Apache-2.0 | Strategic, for the provider | n/a | 190★ | Providers invest in runner tooling, which makes them potential partners | ✅ [O13] |

**Patterns (inference ⚠️):**
- **(A)** Free OSS CLI, grown through provider partnerships (referral links, sponsorship, tutorials).
- **(B)** A hosted control plane at about **$3–5 per host per month**.
- **(C)** A license for commercial or team use.
- **(D)** A paid UI or API add-on.
- Donations alone are not enough.
- **Every observed revenue stream comes from teams, or is priced per server.** None comes from solo developers.

### 7.2 Distribution channels and what it takes to get in

| Channel | Entry requirements | RunnerKit today | Verif. / Source |
|---|---|---|---|
| **jonico/awesome-runners** (895★, Apache-2.0) | "PRs Welcome". Lists very small projects (gh-runnerctl 2★, Sprinters 9★, Zoomies 51★). **No Hetzner-specific or BYO tool listed**; Cyclenerd, TestFlows and Ubicloud are all absent. | **The most open channel.** Has room for a Hetzner/BYO entry. | ✅ [B8] |
| neysofu/awesome-github-actions-runners | Stale; still lists BuildJet | Low value | ⚠️ |
| **awesome-selfhosted** (321.8k★) | Must be FOSS. First release more than 4 months old. Active maintenance. Excludes software "depending on specific cloud providers" and "generic deployment/virtualization/container automation tools". **Submissions must be written by a human, not an AI agent.** | **Blocked** (no license, dormant), and RunnerKit may fall under the excluded categories | ✅ [B9] |
| **homebrew-core** | A DFSG-compatible license. At least **75★ or 30 forks or 30 watchers** (any one) if a third party submits it, or **225★ or 90 forks or 90 watchers** if the owner self-submits. Repo at least 30 days old. CLI-only open-source software belongs in core as a formula, not as a cask. | **Blocked.** RunnerKit ships a cask from its own tap. | ✅ [B7] |
| **Hetzner Community tutorials** | Up to **€50 credit** per new tutorial, about €10 for major updates. Needs an existing Hetzner invoice. External resources must be FOSS. "Should not include links to or content of **small external repositories**." No references to Hetzner's competitors. Review backlog. | **Blocked** until RunnerKit has a license *and* traction | ✅ [B10] |
| Hetzner Cloud Apps (one-click) | "We cannot accept … suggestions for new apps as of now" | **Closed** | ✅ [B11] |
| Hetzner partner / referral link | A co-branded link exists for Coolify | Terms could not be checked | Link ✅ [B1]; terms ⚠️ |
| DigitalOcean 1-Clicks | Community Packer templates (MIT; 160★; 100+ templates incl. Dokku) | Possible later | ✅ [B11] |
| **GitHub Marketplace** | Actions can be listed for free but **cannot be sold**. Paid listings are Apps only: an org publisher, publisher verification, **≥ 100 installations**, monthly and annual billing. | A discovery channel only (Cyclenerd and machulav are listed there) | ✅ [B12] |
| GitHub Secure Open Source Fund | An open-source license plus community traction; **$10,000 per project** | **Blocked** | ✅ [B13] |
| Agent Skills (`gh skill`), Claude Code plugin marketplace, MCP | The open Agent Skills spec; gh ≥ 2.90.0 for `gh skill install` | Cheap to try; demand unproven (Dokploy MCP 386★; Actuated's skill 1★) | ✅ [T18][B2][B14] |
| Show HN, r/selfhosted, r/github, lobste.rs | No formal gate | **Never used** ✅ [R1]; response could not be measured (blocked) ⚠️ | n/a |
| Pain-specific answers in GitHub community threads (#120813, #20019, #179202) | Must actually help | Unused | inference ⚠️ |

**Name availability** ✅ [B15]:
- "RunnerKit" is free on GitHub (org/user), npm, PyPI, crates.io, RubyGems and Docker Hub.
- **runnerkit.com, .dev and .app are registered.** runnerkit.io and .sh did not resolve ⚠️.
- **srz-zumix/gh-runner-kit** is an active gh extension with a similar name (0★, active Sep 2026) ✅ [O15].
- Trademark status is unknown ⚠️.

---

## 8. Where RunnerKit could win, and where it cannot

**Preconditions.** Everything under "could win" depends on these; they come from the code dossier and are all ✅.
- **The product has shipped P0 defects on its core paths:**
  - BYO fails on password-sudo hosts.
  - The runner user is never added to the docker group.
  - `upgrade-runner` destroys the runner's credentials.
  - `down` orphans billed Hetzner VMs.
  - The consent screen shows a wrong price.
- **It also has P1 defects with workarounds** (CODEBASE-ASSESSMENT §5):
  - CLI errors are silent.
  - Typed confirmations fail.
- **There is no LICENSE.**
- **It has been dormant for about 130 days, with no known external users** (0 stars; the 35 downloads fit the maintainer's own pattern).

None of the opportunities below can be tested until a license and a working release exist.

### 8.1 Could win (niches, each conditional)

| # | Opportunity | Why it looks open | Evidence strength | What must be true | Nearest competitors |
|---|---|---|---|---|---|
| 1 | **"Keep my runner alive": day-2 operations for owner-operated persistent runners.** A watchdog for "online but idle", auto re-register after the 14-day removal, version-drift checks, disk and Docker hygiene, and a snippet that falls back to hosted runners. | These are the top 5 measured pains (§5.3). Upstream is "not taking contributions". GitHub has "no plans" for fallback. | **Medium.** Pains ✅; willingness to pay ⚠️ none seen | The tool must itself be reliable (its own maintainer's host ran out of disk). Needs a host-side agent or timer. | runner-fleet (22★), runner-fallback-action (15★), GitHub docs; nothing solo-focused with traction |
| 2 | **One box for all my personal repos** | A runner registers to one repo at a time, with no user scope ✅. Managed vendors are org-only (✅/⚠️). | **Low–medium.** A real need, but low engagement (#179202: 2 upvotes) | Bulk or automatic registration; no race conditions on shared state | Separate runner directories by hand; Docker images; "create an org" |
| 3 | **BYO existing or rented-dedicated hardware** (homelab, Mac mini, paid VPS, Hetzner auction box) | The only case with near-zero marginal cost. Free and Pro users cannot buy more than 2 vCPU / 8 GB from GitHub ✅. | **Medium** on economics; demand size ⚠️ | Must support non-Ubuntu and arm64 hosts properly (both broken today ✅) | myoung34 Docker image, Ansible roles, raw config.sh |
| 4 | **A neutral, honest cost and break-even tool** (e.g. `runnerkit cost`) | Comparison content is dominated by vendors ⚠️. The honest answer is often "stay on hosted" (§3). | **Low–medium** (an SEO and trust play) | Live Hetzner and GitHub prices; say plainly "hosted is cheaper for you" when that is true | Vendor calculators |
| 5 | **Agent-ready self-hosted runners on personal repos** (claude-code-action, gh-aw, Copilot cloud agent) | Agent volume is exploding ✅. These three are not restricted to orgs or ARC ✅. | **Low** for self-hosting demand specifically (unmeasured) | A **refilling ephemeral pool** (scaleset or JIT), a separate HOME per job, an egress allowlist, and an x64 check. Copilot code review stays out of reach. | ARC/k3s, Zoomies, managed "agent compute" vendors |
| 6 | **Free security hardening for self-hosters** (egress allowlist, audit for unknown runners) | Harden-Runner self-hosted is Enterprise-only ✅. Rogue runners are a known worm technique ⚠️. | **Medium** on the gap; demand ⚠️ | Must not repeat RunnerKit's own security issues (a sudoers file that is effectively NOPASSWD ALL; docker group) | None free at the solo level |
| 7 | **EU/own-box with no third-party control plane** | No cheap self-serve option found ⚠️ | **Low** (no size data) | Stable Hetzner pricing, or a BYO focus | CRACI (teams), Ubicloud (managed) |
| 8 | **Listing gap** | awesome-runners has no Hetzner/BYO row ✅ | **High** that the gap exists, low impact on its own | A license and a working release | Cyclenerd and TestFlows could claim it first |

### 8.2 Cannot win, or should not try

| Area | Why not | Verif. |
|---|---|---|
| **"Cheaper than GitHub" for a typical solo dev** (under 3k min/mo) | GitHub's free quota costs $0 (up to $6 at 3k on Free). Ubicloud's credit covers 1,250 premium minutes, so it costs $0–3.50. Blacksmith's 3,000 free minutes apply only to organizations. A Hetzner box is a net loss (§3). | ✅ / ⚠️ calc |
| **Public or untrusted repos** | Hosted is free and unlimited (4 vCPU / 16 GB), and GitHub says "almost never" self-host public repos. | ✅ [G3][G21] |
| **Managed fast runners for organizations** | Crowded, well funded, a commodity at $0.00125–0.006/min. | mixed |
| **Kubernetes autoscaling, AWS-account runners** | ARC (6.5k★, official); RunsOn and terraform-aws (1.3k★ / 3.1k★). | ✅ |
| **Copilot code review self-hosting** | ARC-only by policy. | ✅ [T3] |
| **Per-job ephemeral Hetzner VMs** | Cyclenerd needs no install; TestFlows autoscales with caches; hourly billing makes it costly. | ✅ / ⚠️ |
| **Hetzner cloud as the default cost story** | Price roughly 4× RunnerKit's assumption; stock is volatile; API churn. | ⚠️ / ✅ |
| **macOS, Windows, GPU** | Unsupported. The macOS space belongs to Tart (OpenAI, FSL) and managed vendors. | ✅ |
| **Shielding users from GitHub outages** | Self-hosted runners depend on the same control plane. | ✅ [U9] |
| **Out-shipping Zoomies on fleet features** | 6 releases in 13 days, a web UI, multi-host agents, AGPL. | ✅ [O9] |

### 8.3 Honest positioning statement

**What the evidence supports:** a small, free, open-source tool for **individuals and small teams who already run, or want to run, a persistent runner on hardware they control**. It would differentiate on **day-2 reliability and safety**, not on setup speed or price.

**Size of the prize:** tens of thousands of potential users at most ⚠️, with **no evidence of willingness to pay** among individuals ⚠️.

**What would change the picture** (not yet validated):
- **A measured response to a single low-cost launch test** built around the day-2 message.
- **A working scale-set spike** that makes agent-ready ephemeral pools credible.

---

## 9. Research limitations

### 9.1 Corrections applied from adversarial verification

| Claim in an earlier dossier stream | Status | Corrected value used here |
|---|---|---|
| #20019 hosted-fallback request: "197 upvotes, 49 comments" | ❌ refuted | **107 upvotes; 24 comments + 25 replies**. A cited workaround is still working, not broken. |
| #3724 as evidence of OOM on small self-hosted VMs | ❌ refuted | It was a GitHub-hosted 32-core larger runner, closed as stale. **No verified OOM user-voice evidence.** |
| Ubicloud "premium only for new customers; standard closed; credit unverified" | ❌ refuted | New accounts **default** to premium ($0.002) and **can opt into** standard ($0.00125). The **$2.50/mo credit (1,250 premium minutes) is documented.** Premium rose only once ($0.0016 → $0.0020, 2026-09-01). |
| Ubicloud cost for 5,000 min "$3.75" | ❌ (default tier) | **$7.50** at the default premium rate; $3.75 applies only after switching to standard |
| #805 "Add ARM + MacOS target" (253 reactions) as current unmet demand | ❌ outdated | Closed; the demand was met years ago |
| Copilot cloud sandbox "$10/month entitlement" | ❌ outdated | Ended with July 2026; all usage is now billed |
| Actuated "from $250/mo" | ❌ superseded | **$150 first server + $125 each additional** (per-server plans since 2026-05-07) |
| Depot "$0.004/min" (third-party sites) | ❌ stale | **$0.006/min** (Depot docs). Tracked by the second and billed as whole minutes at month end. macOS and GPU on the Business plan only. |
| Break-even assuming solo devs get 2,000 free minutes | ❌ incomplete | Pro includes **3,000**, which raises break-even by about 1,000 minutes |
| Break-even against larger runners (2–3.5k min) as a solo-dev argument | ❌ inapplicable | Larger runners are **Team/Enterprise orgs only** |
| Postponement dated "Dec 15, 2025" | ❌ | Announced **2025-12-16**, postponed **2025-12-17** |
| actions/scaleset "announced 2026-02-05" | ❌ minor | Community announcement and first commit **2026-02-03**; v0.1.0 tagged 2026-02-05 |
| RunsOn "10× cheaper" | ⚠️ conflicting | The README's current wording is "7× lower 2-vCPU Linux cost" ($0.0009 vs $0.0060); one verifier also saw "10x" |
| "RunnerKit doctor has no version check" | ❌ | It has `runner_version_stale`, but it compares against the bundled pin only |
| "Persistent warm cache is uncontested on Hetzner" | ❌ | TestFlows has persistent cache volumes and a recycle mode |
| #120813 "103 comments" | clarified | 33 comments + 70 replies; the 46 upvotes are unconfirmed |
| #3609 as evidence from Linux solo users | clarified | A Windows runner on EC2; reaction counts unverified |
| #3857 "Runner not found" as a solo pain | clarified | Org-scale Linux-kernel BPF CI; counts unverified |
| #3792 `safe_sleep.sh` (cited by Zig) | clarified | Closed and fixed |
| Forced deprecation "often without warning" | overstated | GitHub did announce minimum versions (e.g., 2025-12-12), but notices are inconsistent |
| `pull_request_target` block "for all public repos" | clarified | Applies to **affected repos** that used the default policy before GA |
| gh-aw engines "four" | clarified | Copilot, Claude Code, Codex, Gemini, **Pi** |
| Discussion #182089 cited as evidence of homelab ARM/CUDA demand | ❌ | It asks for *hosted* runners, not homelab self-hosting |
| Hetzner "three price rises in 2026" | ⚠️ | Two (April 1, June 15) are corroborated by secondary sources; a third is unconfirmed |

### 9.2 Method and coverage limits

1. **Blocked sources.** An egress proxy blocked most vendor and news domains: hetzner.com, docs.hetzner.com, api.hetzner.cloud, blacksmith.sh, ubicloud.com, depot.dev, warpbuild.com, namespace.so, buildjet.com, runs-on.com, docs.github.com, github.blog, techcrunch.com, HN, Reddit, dev.to, codeberg.org and forgejo.org. GitHub docs were read from the `github/docs` source repo instead. The shared WebSearch budget (200 calls) was exhausted. **No new web facts were fetched for this report.**
2. **Hetzner figures are secondary.** Every 2026 Hetzner price and availability figure comes from secondary sources: public repos recording `/v1/server_types` values, a price-tracker dataset and dated probes quoted in GitHub issues. They need confirming with an authenticated `GET /v1/pricing` and `/v1/server_types` before any public claim or default change.
3. **Vendor figures are secondary.** Vendor funding, customer counts, prices (except Depot, Ubicloud, RunsOn and Actuated, which were read from their GitHub-hosted sources) and shutdown dates (BuildJet) come from search snippets.
4. **User voice is github.com-only.**
   - Reddit sentiment and HN point counts are missing.
   - Some reaction counts would not render.
   - Commit-search counts are noisy: they include tutorials, bots and false positives.
   - The population estimate (10⁴–10⁵) is an inference from proxies, not a measurement.
   - **No user interviews were done.**
5. **RunnerKit gives no demand signal.** It was never launched, so its 0 stars and 35 downloads say nothing about demand either way.
6. **Security incidents are partly single-source.** Several 2026 supply-chain waves rely on a single community IOC tracker, not primary vendor reports. September 2026 was not checked.
7. **Several points are inferred, not tested:**
   - that Free/Pro accounts follow the 2026-09-25 enforcement date;
   - that RunnerKit's 2.334.0 pin still works through auto-update;
   - that scaleset works on repos owned by personal accounts;
   - that agent workloads route to RunnerKit runners.
8. **Arithmetic assumptions.** 1 EUR = 1.1622 USD, VAT excluded, a 10-minute average job for per-job VM costs, and no subscription, electricity or labor costs. Results are sensitive to all of these.
9. **Time sensitivity.** Ubicloud and Hetzner each repriced twice in 2026, and GitHub policy moves monthly. Treat everything here as a snapshot dated 2026-09-26.

---

## 10. Sources (deduplicated, grouped by topic)

Keys match the in-text citations. "(blocked)" marks sources that were cited by secondary summaries but could not be fetched, so claims resting only on them are ⚠️.

### GitHub pricing, policy and runner lifecycle
- [G1] https://raw.githubusercontent.com/github/docs/main/data/reusables/billing/actions-standard-runner-prices.md
- [G2] https://raw.githubusercontent.com/github/docs/main/content/billing/reference/actions-runner-pricing.md
- [G3] https://raw.githubusercontent.com/github/docs/main/data/reusables/billing/actions-included-quotas.md
- [G4] https://raw.githubusercontent.com/github/docs/main/content/billing/concepts/product-billing/github-actions.md
- [G5] https://github.com/resources/insights/2026-pricing-changes-for-github-actions
- [G6] https://github.com/github/roadmap/issues/1196
- [G7] https://github.com/orgs/community/discussions/182186
- [G8] https://github.com/orgs/community/discussions/182089
- [G9] https://raw.githubusercontent.com/github/docs/main/content/actions/reference/limits.md
- [G10] https://raw.githubusercontent.com/github/docs/main/content/actions/reference/runners/github-hosted-runners.md
- [G11] https://github.com/github/docs/blob/main/data/reusables/actions/supported-github-runners.md
- [G12] https://github.com/pricing
- [G13] https://github.com/orgs/community/discussions/194762
- [G14] https://raw.githubusercontent.com/github/docs/main/content/actions/reference/runners/self-hosted-runners.md
- [G15] https://raw.githubusercontent.com/github/docs/main/data/reusables/actions/self-hosted-runner-update-warning.md
- [G16] https://github.com/github/docs/blob/main/data/reusables/actions/self-hosted-runner-auto-removal.md
- [G17] https://github.com/actions/runner/releases ; https://github.com/actions/runner/releases.atom ; https://github.com/actions/runner/releases/tag/v2.337.0
- [G18] https://github.com/actions/runner/issues/4203 ; https://github.com/actions/runner/issues/4305 ; https://github.com/actions/runner/issues/4392 ; https://github.com/actions/runner/issues/4442 ; https://github.com/actions/runner/issues/4613 ; https://github.com/actions/runner/issues/4421 ; https://github.com/actions/runner/issues/3767
- [G19] https://github.com/orgs/community/discussions/179202
- [G20] https://raw.githubusercontent.com/github/docs/main/content/actions/concepts/runners/runner-scale-sets.md
- [G21] https://github.com/github/docs/blob/main/content/actions/reference/security/secure-use.md
- [G22] https://raw.githubusercontent.com/github/docs/main/content/actions/reference/security/securely-using-pull_request_target.md
- [G23] https://github.com/github/docs/blob/main/data/reusables/actions/self-hosted-runner-security.md
- [G24] (blocked) GitHub changelog entries:
  - https://github.blog/changelog/2025-12-16-coming-soon-simpler-pricing-and-a-better-experience-for-github-actions/
  - https://github.blog/changelog/2026-01-29-arm64-standard-runners-are-now-available-in-private-repositories/
  - https://github.blog/changelog/2026-03-26-custom-images-for-github-hosted-runners-are-now-generally-available/
  - https://github.blog/changelog/2026-06-18-actions-build-custom-images-from-custom-images/
  - https://github.blog/changelog/2026-04-27-github-copilot-code-review-will-start-consuming-github-actions-minutes-on-june-1-2026/
  - https://github.blog/changelog/2025-10-28-copilot-coding-agent-now-supports-self-hosted-runners/
  - https://github.blog/changelog/2026-06-12-github-actions-minimum-version-enforcement-timeline-for-self-hosted-runners/
  - https://github.blog/changelog/2026-03-31-eu-data-residency-region-expanding-to-include-efta-countries/
- [G25] https://github.com/orgs/community/discussions/186265

### Hetzner prices, availability and API
- [H1] https://raw.githubusercontent.com/robhunter/agentdeals/main/data/deal_changes.json
- [H2] https://raw.githubusercontent.com/robhunter/agentdeals/main/data/index.json
- [H3] https://raw.githubusercontent.com/jikig-ai/soleur/main/knowledge-base/operations/expenses.md ; https://github.com/jikig-ai/soleur/issues/6966 ; https://raw.githubusercontent.com/jikig-ai/soleur/main/apps/web-platform/infra/zot-registry.tf
- [H4] https://github.com/alethialabs-io/alethialabs/issues/2607 ; https://github.com/alethialabs-io/alethialabs/issues/5069
- [H5] https://github.com/kubermatic/dashboard/issues/8329
- [H6] https://raw.githubusercontent.com/hetznercloud/hcloud-go/main/CHANGELOG.md ; https://github.com/hetznercloud/hcloud-go/releases
- [H7] https://raw.githubusercontent.com/testflows/TestFlows-GitHub-Hetzner-Runners/main/README.rst
- [H8] (blocked) Hetzner's own pages:
  - https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/
  - https://www.hetzner.com/pressroom/statement-price-adjustment/
  - https://www.hetzner.com/pressroom/standardization-and-price-adjustment-of-our-server-products/
  - https://docs.hetzner.com/cloud/billing/faq/
- [H9] (blocked) secondary coverage:
  - https://byteiota.com/hetzner-june-2026-price-shock/
  - https://wz-it.com/en/blog/hetzner-price-increase-june-2026-cpx-ccx-alternatives/
  - https://www.vincentschmalbach.com/hetzner-cheap-cloud-unavailable-price-increases/
  - https://bex.co/blog/2026/09/24/hetzner-cheap-tier-unavailable-fleet-planning
  - https://bex.co/blog/2026/09/17/hetzner-server-auction-dedicated-vs-cloud-fleet-nodes
  - https://cloudtally.eu/blog/hetzner-cloud-billing-explained
  - https://www.tomshardware.com/tech-industry/hetzner-to-raise-prices-by-up-to-37-percent-from-april-1
- [H10] https://github.com/search?q=%22price+adjustment%22+CPX22&type=issues ; https://github.com/CanineHQ/canine/issues/652 ; https://github.com/monkeysees/small-cloud/issues/17

### Managed runner vendors
- [M1] https://github.com/depot/docs/blob/main/content/github-actions/runner-types.mdx ; https://github.com/depot/docs/blob/main/content/github-actions/overview.mdx ; https://github.com/depot/docs/blob/main/content/github-actions/quickstart.mdx
- [M2] https://raw.githubusercontent.com/ubicloud/ubicloud/main/config/billing_rates/github.yml
- [M3] https://raw.githubusercontent.com/ubicloud/documentation/main/about/pricing.mdx ; https://raw.githubusercontent.com/ubicloud/documentation/main/github-actions-integration/quickstart.mdx
- [M4] https://github.com/ubicloud/ubicloud
- [M5] https://github.com/runs-on/runs-on ; https://raw.githubusercontent.com/runs-on/runs-on/main/README.md ; https://github.com/runs-on/runs-on/blob/main/SPONSORSHIP.md
- [M6] https://raw.githubusercontent.com/self-actuated/actuated.com/master/components/PriceCalculator.jsx ; https://github.com/self-actuated/actuated.com/commits/master/components/PriceCalculator.jsx ; https://github.com/self-actuated
- [M7] (blocked or secondary) Blacksmith:
  - https://www.blacksmith.sh/pricing
  - https://docs.blacksmith.sh/introduction/quickstart
  - https://github.com/search?q=blacksmith+%223%2C000+free%22+%240.004&type=issues
  - https://github.com/useblacksmith
- [M8] (blocked) funding coverage:
  - https://techcrunch.com/2026/08/12/blacksmiths-valuation-jumps-10x-to-550m-as-ai-coding-fuels-software-validation/
  - https://www.prnewswire.com/news-releases/blacksmith-raises-10m-to-unblock-ai-development-with-fast-ci-for-github-actions-302559870.html
  - https://www.finsmes.com/2026/08/blacksmith-raises-45m-in-series-b-funding.html
  - https://namespace.so/blog/series-a
  - https://www.nea.com/blog/namespace-cicd-is-dead-agents-need-computers
  - https://depot.dev/blog/depot-raises-series-a
  - https://depot.dev/changelog/2026-03-24-depot-ci-now-available
- [M9] (blocked) BuildJet shutdown: https://buildjet.com/for-github-actions/blog/we-are-shutting-down ; https://www.warpbuild.com/blog/buildjet-shutting-down ; https://runs-on.com/blog/buildjet-shutting-down-runson-alternative/
- [M10] https://github.com/fish-shell/fish-shell/issues/12626 ; https://github.com/scipy/scipy/issues/24990 ; (blocked) https://cirrus-runners.app/pricing/ ; https://cirruslabs.org/
- [M11] https://github.com/ShipfoxHQ/shipfox
- [M12] https://github.com/TheMorpheus407/european-alternatives/issues/657 ; (blocked) https://craci.com/
- [M13] (blocked or secondary) other vendors:
  - https://www.warpbuild.com/pricing
  - https://namespace.so/pricing
  - https://cirun.io/ ; https://docs.cirun.io/ ; https://cirun.io/region/gcp-europe-west1/
  - https://www.tenki.cloud/pricing
  - https://latchkey.dev/
  - https://www.ycombinator.com/companies/starsling
  - https://machine.dev/
  - https://bitrise.io/platform/build-hub
  - https://buildkite.com/pricing/
  - https://sprinters.sh/ ; https://github.com/sprinters-sh/sprinters
  - https://dcxv.com/github-actions-runner
- [M14] (blocked) vendor marketing around the fee:
  - https://www.blacksmith.sh/blog/actions-pricing
  - https://www.warpbuild.com/blog/github-actions-price-change
  - https://runs-on.com/blog/github-self-hosted-runner-fee-2026/
  - https://cirrus-runners.app/blog/2025/12/16/new-pricing-of-self-hosted-github-actions-runners-explained/
  - https://northflank.com/blog/github-pricing-change-self-hosted-alternatives-github-actions
  - https://www.tenki.cloud/blog/github-actions-runner-pricing-2026
  - https://runnerbench.com/
- [M15] (blocked) https://runs-on.com/benchmarks/github-actions-cpu-performance/ ; https://runs-on.com/benchmarks/github-actions-cache-performance/

### Open-source and self-managed tools
- [O1] https://github.com/actions/scaleset ; https://github.com/actions/scaleset/tags ; https://github.com/actions/scaleset/commits/main ; https://github.com/actions/scaleset/issues
- [O2] https://github.com/actions/actions-runner-controller ; https://github.com/actions/actions-runner-controller/tags ; https://github.com/actions/actions-runner-controller/releases/tag/gha-runner-scale-set-0.14.0
- [O3] https://github.com/github-aws-runners/terraform-aws-github-runner
- [O4] https://github.com/cloudbase/garm ; https://github.com/cloudbase/garm/releases
- [O5] https://github.com/myoung34/docker-github-actions-runner ; https://hub.docker.com/v2/repositories/myoung34/github-runner/
- [O6] https://github.com/MonolithProjects/ansible-github_actions_runner ; https://github.com/macunha1/ansible-github-actions-runner
- [O7] https://github.com/Cyclenerd/hcloud-github-runner ; https://github.com/Cyclenerd/hcloud-github-runner/network/dependents ; https://github.com/marketplace/actions/self-hosted-github-actions-runner-on-hetzner-cloud
- [O8] https://github.com/testflows/TestFlows-GitHub-Hetzner-Runners ; https://github.com/testflows/testflows-github-hetzner-runners/commits/main
- [O9] https://github.com/eyupio/zoomies ; https://github.com/eyupio/zoomies/tags
- [O10] https://github.com/soulteary/runner-fleet
- [O11] https://github.com/solutionforest/ephemeral-action-runner
- [O12] https://github.com/machulav/ec2-github-runner
- [O13] https://github.com/hostinger/fireactions ; https://github.com/cisco-open/forge
- [O14] https://github.com/openai/tart ; https://github.com/traderepublic/Cilicon ; https://github.com/shapehq/tartelet ; https://github.com/mirego/ekiden
- [O15] https://github.com/Flinner/gh-runnerctl ; https://github.com/srz-zumix/gh-runner-kit ; https://github.com/madkoo/gh-runner ; https://pkg.go.dev/github.com/Dedac/gh-runner ; https://github.com/1XP-AI/gh-runnerd
- [O16] https://github.com/kube-hetzner/terraform-hcloud-kube-hetzner
- [O17] https://github.com/mikehardy/runner-fallback-action
- [O18] https://github.com/actions/runner
- [O19] https://github.com/topics/self-hosted-runner?o=desc&s=stars
- [O20] https://github.com/marketplace/actions/hetzner-cloud-self-hosted-runner-for-github-ci ; https://github.com/louisgundelwein/runner-autoscaler

### User voice and demand
- [U1] https://github.com/orgs/community/discussions/120813
- [U2] https://github.com/actions/runner/issues/3609 ; https://github.com/actions/runner/issues/3857
- [U3] https://github.com/actions/runner/issues/434 ; https://github.com/actions/runner/issues/2708 ; https://github.com/orgs/community/discussions/68915
- [U4] https://github.com/orgs/community/discussions/20019
- [U5] https://github.com/orgs/community/discussions/58146
- [U6] https://api.github.com/search/commits?q=%22self-hosted%20runner%22%20committer-date:2026-01-01..2026-09-25 (plus the variants with "offline", "minutes", "disk" and "back to GitHub-hosted")
- [U7] https://api.github.com/search/issues?q=repo:actions/runner+is:issue+created:2026-01-01..2026-09-25
- [U8] https://github.com/community/community/discussions/206900 ; https://github.com/community/community/discussions/206494
- [U9] https://github.com/orgs/community/discussions/204152 ; community discussions #208693, #208250 and #205214 via https://github.com/orgs/community/discussions?discussions_q=self-hosted+runner
- [U10] (blocked) https://www.techtimes.com/articles/324820/20260818/github-actions-hit-three-nines-failure-one-august-outage-consumed-years-downtime-budget.htm ; https://blog.incidenthub.cloud/github-reliability-outage-history-2025-2026
- [U11] (blocked) https://news.ycombinator.com/item?id=46294001 ; https://news.ycombinator.com/item?id=46291156 ; https://news.ycombinator.com/item?id=46301772 ; https://news.ycombinator.com/item?id=46310661
- [U12] https://github.com/Sparxx947/romseerr/commit/fd097155c9976931f00a21e21a9c3b154b55dc33 ; https://github.com/Sparxx947/romseerr/commit/82b7e1ed6c7e81d8c43d2e34973bd92885f3b7d1
- [U13] https://api.github.com/search/issues?q=repo:myoung34/docker-github-actions-runner+is:issue+created:2025-01-01..2026-09-25 ; https://api.github.com/search/issues?q=repo:Cyclenerd/hcloud-github-runner+is:issue
- [U14] https://github.com/search?type=discussions (repo:community/community, created ≥ 2026-01-01)
- [U15] https://github.com/actions/runner/issues/3724 ; https://github.com/actions/runner/issues/805 ; https://github.com/actions/runner/issues/801 ; https://github.com/actions/runner/issues/3792
- [U16] https://github.com/orgs/community/discussions/182419 ; https://github.com/orgs/community/discussions/178748

### Trends: AI agents, security, platforms
- [T1] https://github.com/aavetis/PRarena ; https://github.com/aavetis/PRarena/blob/main/data.csv
- [T2] https://raw.githubusercontent.com/github/docs/main/content/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/customize-the-agent-environment.md
- [T3] https://raw.githubusercontent.com/github/docs/main/content/copilot/how-tos/copilot-on-github/set-up-copilot/configure-runners.md
- [T4] https://raw.githubusercontent.com/github/docs/main/content/copilot/how-tos/administer-copilot/manage-for-organization/configure-runner-for-coding-agent.md
- [T5] https://raw.githubusercontent.com/github/docs/main/data/reusables/gated-features/copilot-cloud-agent.md ; https://raw.githubusercontent.com/github/docs/main/data/reusables/copilot/cloud-agent-required-hosts.md ; https://raw.githubusercontent.com/github/docs/main/content/copilot/concepts/agents/about-third-party-coding-agents.md
- [T6] https://github.com/github/gh-aw ; https://raw.githubusercontent.com/github/gh-aw/main/docs/src/content/docs/reference/self-hosted-runners.md ; https://github.com/github/docs/blob/main/content/copilot/concepts/agents/about-github-agentic-workflows.md
- [T7] https://github.com/anthropics/claude-code-action ; https://github.com/anthropics/claude-code-action/issues/1688 ; https://github.com/anthropics/claude-code-action/network/dependents
- [T8] https://github.com/openai/codex-action ; https://github.com/openai/codex-action/issues/160 ; https://github.com/openai/codex-action/issues/69
- [T9] https://raw.githubusercontent.com/github/docs/main/content/billing/concepts/product-billing/cloud-and-local-sandboxes.md ; https://github.com/github/docs/blob/main/content/copilot/concepts/security-governance-and-network-settings/about-cloud-and-local-sandboxes.md
- [T10] https://github.com/github/roadmap/issues/1192
- [T11] https://github.com/advisories/GHSA-mrrh-fwg8-r2c3
- [T12] https://github.com/TanStack/router/security/advisories/GHSA-g7cv-rxg3-hmpx ; https://github.com/nrwl/nx-console/security/advisories/GHSA-c9j4-9m59-847w
- [T13] https://github.com/Cobenian/shai-hulud-detect/blob/main/README.md ; https://github.com/Cobenian/shai-hulud-detect/blob/main/CHANGELOG.md ; https://github.com/DataDog/indicators-of-compromise/tree/main/shai-hulud-2.0
- [T14] https://github.com/step-security/harden-runner
- [T15] https://github.com/AdnaneKhan/gato-x
- [T16] https://github.com/ziglang/www.ziglang.org/blob/main/content/en-US/news/migrating-from-github-to-codeberg.smd ; https://github.com/ziglang/zig
- [T17] https://raw.githubusercontent.com/github/docs/main/data/reusables/copilot/code-review/code-review-actions-usage.md
- [T18] https://github.com/github/github-mcp-server ; https://raw.githubusercontent.com/github/docs/main/content/copilot/concepts/agents/about-agent-skills.md ; https://github.com/github/docs/blob/main/content/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/add-skills.md ; https://github.com/agentskills/agentskills
- [T19] https://github.com/cursor/terraform-provider-cursor ; https://github.com/e2b-dev/E2B ; https://github.com/daytonaio/daytona ; https://github.com/modal-labs/modal-client
- [T20] https://github.com/go-gitea/gitea ; https://gitea.com/gitea/runner ; (blocked) https://blog.gitea.com/release-of-runner-1.0.0/ ; https://forgejo.org/2026-08-runner-release-v13/ ; https://codeberg.org/pierreprinetti/forgejo-hetzner-runner ; https://en.wikipedia.org/wiki/Codeberg ; https://www.devclass.com/ci-cd/2025/11/27/zig-project-ditches-github-for-codeberg-but-move-could-be-costly/1727053
- [T21] (blocked) https://www.sysdig.com/blog/how-threat-actors-are-using-self-hosted-github-actions-runners-as-backdoors ; https://www.praetorian.com/blog/self-hosted-github-runners-are-backdoors/
- [T22] https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json (generate-jitconfig) ; https://raw.githubusercontent.com/github/docs/main/content/actions/how-tos/manage-runners/use-actions-runner-controller/authenticate-to-the-api.md

### Go-to-market, business models and channels
- [B1] https://github.com/coollabsio/coolify ; https://raw.githubusercontent.com/coollabsio/coolify/v4.x/README.md ; https://github.com/coollabsio/coolify-docs ; https://github.com/sponsors/andrasbacsai
- [B2] https://github.com/Dokploy/dokploy ; https://github.com/Dokploy/dokploy/blob/canary/LICENSE_PROPRIETARY.md ; https://raw.githubusercontent.com/Dokploy/website/main/apps/website/components/pricing.tsx ; https://github.com/Dokploy/mcp
- [B3] https://github.com/dokku/dokku ; https://raw.githubusercontent.com/dokku/dokku/master/docs/enterprise/pro.md
- [B4] https://github.com/caprover/caprover ; https://github.com/basecamp/kamal
- [B5] https://github.com/earthly/earthly
- [B6] https://github.com/sponsors/alexellis
- [B7] https://raw.githubusercontent.com/Homebrew/brew/main/docs/Package-Acceptance-Policy.md ; https://raw.githubusercontent.com/Homebrew/brew/main/docs/Acceptable-Casks.md ; https://raw.githubusercontent.com/Homebrew/brew/main/docs/Acceptable-Formulae.md
- [B8] https://github.com/jonico/awesome-runners ; https://raw.githubusercontent.com/jonico/awesome-runners/main/README.md
- [B9] https://github.com/awesome-selfhosted/awesome-selfhosted ; https://raw.githubusercontent.com/awesome-selfhosted/awesome-selfhosted-data/master/CONTRIBUTING.md
- [B10] https://raw.githubusercontent.com/hetzneronline/community-content/master/contributing.md ; https://github.com/hetzneronline/community-content
- [B11] https://github.com/hetznercloud/apps ; https://github.com/digitalocean/droplet-1-clicks
- [B12] https://raw.githubusercontent.com/github/docs/main/content/apps/github-marketplace/github-marketplace-overview/about-github-marketplace-for-apps.md ; https://raw.githubusercontent.com/github/docs/main/content/apps/github-marketplace/creating-apps-for-github-marketplace/requirements-for-listing-an-app.md
- [B13] https://github.com/open-source/github-secure-open-source-fund
- [B14] https://github.com/self-actuated/agent-skills ; https://github.com/self-actuated/actuated
- [B15] https://github.com/runnerkit ; https://registry.npmjs.org/-/v1/search?text=runnerkit&size=20 ; https://pypi.org/pypi/runnerkit/json ; https://crates.io/api/v1/crates/runnerkit ; https://rubygems.org/api/v1/search.json?query=runnerkit ; https://hub.docker.com/v2/search/repositories/?query=runnerkit (plus a local DNS lookup on 2026-09-25)

### RunnerKit itself
- [R1] https://github.com/accidentally-awesome-labs/runnerkit ; https://api.github.com/repos/accidentally-awesome-labs/runnerkit ; https://api.github.com/repos/accidentally-awesome-labs/runnerkit/releases
- [R2] Local code at HEAD 64c3003:
  - `/home/user/runnerkit/internal/provider/profile.go`
  - `/home/user/runnerkit/internal/bootstrap/package.go`
  - `/home/user/runnerkit/internal/bootstrap/image_setup.go`
  - `/home/user/runnerkit/internal/ops/doctor.go`
  - `/home/user/runnerkit/internal/preflight/checks.go`
  - `/home/user/runnerkit/internal/cli/wizard.go`
  - `/home/user/runnerkit/docs/safety.md`
  - `/home/user/runnerkit/.planning/smoke-discovery-2026-05-18/FINDINGS.md`
- [R3] https://github.com/accidentally-awesome-labs/dat0/pull/6
- [R4] Released-binary BYO end-to-end test (dossier `gaps.md`, "released-byo-e2e-job"). The test harness lived in ephemeral scratch space and was not committed.
