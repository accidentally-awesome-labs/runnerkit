# RunnerKit backlog: engineering and product

*Date: 2026-09-26. Implements `STRATEGY.md` as revised after the red-team review and editorial rulings R1–R10 (made during this analysis and open to maintainer override), and is re-baselined against it the same day. All dates come from STRATEGY §10.1, the only schedule; milestone IDs C1–C13 refer to that table. Defect IDs (P0-n, P1-n, P2-n, SEC-n) and their severities are CODEBASE-ASSESSMENT §5–6. KEYFACTS labels appear only where they differ. Defect detail comes from `evidence/code-condensed.md`, `evidence/gaps.md`, `evidence/KEYFACTS.md` and `CODEBASE-ASSESSMENT.md`. Repository: `accidentally-awesome-labs/runnerkit` at HEAD `64c3003`, which is also the `v1.3.3` tag.*

**Status 2026-09-27**, recorded before `.planning/` moves to a private repository (H-05). `main` is at `930724e`; v1.3.4 is not tagged.
- **Merged** (2026-09-26/27):
  - PR #1 (`e67a324`): the v1.3.4 harm reduction and core-path repair, A-01 to A-10 and A-14 to A-19 (the §0 cut 1–3 items shipped early, except C-02's CI guard, which is in PR #3), with LICENSE, CONTRIBUTING, SECURITY, `docs/security-posture.md` and the CHANGELOG.
  - PR #2 (`f652142`): `docs/testkit/` and `scripts/testkit/`: the A-20 runbook with `gate.sh`, and the V-1 to V-4 scripts.
  - PR #3 (`60e00df`): the C-02 guard in `go-test`; a DCO job for fork pull requests (H-02's code half); IPv4-only `--ssh-allowed-cidr`; the cloud Known issues (§A.6); the RELEASE-NOTES content in the CHANGELOG (the files are still tracked); the wording sweep (N-02); `TestRenderSudoersEntryGolden`.
  - PR #4 (`80e1837`): `register`'s `lifecycle_foundation_missing` points at `runnerkit up`.
  - PR #5 (`03c6e00`): the V-3 probe fix; `gate.sh` G9 checks A-16's acceptance; identical README and CHANGELOG Known issues (`TestChangelogKnownIssuesMatchReadme`).
  - PR #8 (`930724e`): `gate.sh` writes its CHANGELOG evidence line only on a clean PASS, with `install.sh`'s SHA-256.
- **Held:** draft PR #6 (A-21) and draft PR #7 (H-08's "Fixed in v1.3.4" wording and `TestBYOClaimNeedsRealJobEvidence`). Exactly one merges after the A-20 gate session.
- **Open:** A-20, A-21, H-04a, H-05, the V-1 to V-4 runs (only their scripts merged; V-1 starts by C3) and the §A.7 checklist. Boxes are ticked only where the item's acceptance is met on `main`: A-16 and A-18 are merged, but their acceptance runs in A-20, and A-17a's needs the govulncheck link for the tagged SHA. On 2026-09-27, 2.337.0 was still the latest actions/runner release (A-16).
- **Next:** G-0 and G-1 by C1 (2026-10-03), the A-20 gate session around 2026-10-05, and the v1.3.4 tag by C2 (2026-10-10).
- Where the text below puts A-01..A-03, C-02, A-06b or A-17b in v1.3.5 or in the post-launch ledger, read v1.3.4; §0 counts their hours as provisional Stage 0 spend. File:line references are to `64c3003`; PR #1 moved many of them.

---

## How to use this document

**Item IDs**

| Prefix | Meaning | Where |
|---|---|---|
| `G-` | Gates checked before any other work | §0 |
| `A-` | Release items: v1.3.4 harm reduction (STRATEGY 0.3), the v1.3.4 core-path repair (0.3b), and the items the §0 budget moves to v1.3.5 | §A.1–A.3 |
| `F-` | v1.3.5 fixes that `runnerkit checkup` (W2) needs; built only if W2 gets a go | §A.4 |
| `R-` | Stage 2: item 0, the privilege model (R-01, R-05), then the demand-gated BYO `up` rebuild (the rest) | §A.5 |
| `Z-` | Frozen: fix only if a Stage 3 trigger fires | §A.6 |
| `H-` | Repo, licensing and hygiene | §B |
| `C-` | Test and CI gates | §C |
| `V-` | Day-1 validations (Stage 0) | §D.0 |
| `E1.x` / `E2.x` / `E3.x` | Stage 1 / 2 / 3 epics | §D |
| `N-` | Explicitly not doing | §E |

**Evidence tags** (the same as CODEBASE-ASSESSMENT):
- **[R]** reproduced;
- **[C]** confirmed by reading the code and by the dossier verifier;
- **[I]** inferred, not live-tested;
- **[S]** secondary source.

**Effort sizes**

| Size | Duration |
|---|---|
| S | ≤1 day |
| M | 1–3 days |
| L | 1–2 weeks |
| XL | >2 weeks |

Stage 0 and Stage 1 items also carry an **hour estimate**: maintainer-hours with AI assistance, counted against the STRATEGY §10.1 caps. The §0 ledger sums them.

**Reference verification.** Every `file:line` cited in §A and §C was re-checked against the working tree on 2026-09-26: HEAD `64c3003`, `git status` clean. I also built HEAD in a scratch directory (outside the repo) and re-ran the silent-error cases:

| Invocation | Output | Exit |
|---|---|---|
| `runnerkit bogus` | nothing | 1 |
| `runnerkit status --bogus` | nothing | 2 |
| `runnerkit --version` | nothing | 2 |
| `runnerkit byo-prepare --host x` | nothing | 1 |

Only the `version` subcommand prints anything.

**Definition of done, for every item:**
1. A test that fails on HEAD and passes after the fix.
2. A CHANGELOG entry.
3. Docs updated where the behaviour is user-visible.
4. `RenderSudoersEntry` gains no entries, and `install.sh` grants nothing beyond that list. A-14 makes the two identical; that adds lines to `install.sh` but no capability, because the list was already root-equivalent (STRATEGY §4 stop #2, SEC-1).
5. No release note says BYO works unless, for that tag, a real job ran on a fresh password-sudo host prepared only by `install.sh`. For v1.3.4 that is A-20 (one container); from Stage 2 it is C-09 (the full matrix).

---

## 0. Gates and budget

- [ ] **G-0 Capacity check (by C1, 2026-10-03).** Capacity means availability, not spend (STRATEGY §5, ruling R7): **at least 2 work sessions a week, and issue responses within 72 hours**, through C10 (2026-12-21). If the maintainer cannot commit to that, do only STRATEGY 0.1–0.3, then run the archive playbook (H-17) with an honest notice:
  - H-01..H-05, H-07..H-10 and H-13 (license, hygiene, honesty, including the security page and its revocation section);
  - the §A.1 harm-reduction items;
  - option A (A-21, which refuses BYO `up`/`register`) instead of the 0.3b repair. 0.3b sits outside 0.1–0.3, and an archived repo must not claim BYO works **[backlog reading of STRATEGY §5]**.

  Reason: a public artifact that grants root and makes false safety claims is itself harmful.
- [ ] **G-1 Hour log and weekly table**, started by C1: a public table in a pinned Discussion (H-16). The §10.1 hour caps are the only spend limits:
  - Stage 0: ≤25 h (C4, 2026-10-17).
  - Cumulative: ≤55 h at launch (C8, latest 2026-11-16).
  - W1–W3 build: ≤30 h (checked at C9, 2026-11-23). This is the one exemption from the two-request rule.
  - Cumulative: ≤85 h at the decision memo (C10, 2026-12-21).

  Kill criterion 1: no logged progress for 2 consecutive weeks, or any C-milestone slipping by more than 7 days. Finish the harm patch if it is unfinished, then archive. Kill criterion 2: launch not done by 2026-11-16, or more than 55 h spent before launch, means archive.

**Budget ledger.** Re-baseline from the G-1 log at C2, C4, C7 and C8. Every figure is the sum of the per-item hours stated in §A–§D. "Full scope" is everything STRATEGY lists for the stage before any cut. "Committed" is what the plan fits under the cap.

*Stage 0 (2026-09-26 → C4 2026-10-17; cap ≤25 h)*

| Block (STRATEGY) | Committed items (hours) | Full scope | Committed |
|---|---|---:|---:|
| 0.1 Licensing and hygiene | H-01 0.25, H-02 0.5, H-03 0.5, H-04a 0.25, H-05 0.5, H-07 0.25, H-13 0.25, G-1/H-16 for 3 weeks 0.75 | 5.0 (adds H-06 0.5, H-11 1, H-04b 0.25) | 3.25 |
| 0.2 Honesty pass | H-08 1.5, H-09 1.5, H-10 0.25 | 3.25 | 3.25 |
| 0.3 v1.3.4 harm reduction | A-04 1, A-05 1, A-06a 1, A-07 2, A-08 1, A-09 0.5, A-10 0.5, A-15 0.5, A-16 0.5, A-17a 0.5, release run (§A.7) 0.5 | 17.5 (adds A-01 1.5, A-02 1.5, A-03 1, C-02 0.5, A-06b 0.5, A-17b 3.5) | 9.0 |
| 0.3b Core-path repair (hard box) | A-14 2, A-18 1, A-19 1, A-20 1.5, A-21 reserve 0.5 | 6.0 | 6.0 |
| 0.4 Day-1 validations | V-1 0.5, V-2 0.5, V-3 1, V-4 1, publishing 0.5 (reusing A-20's container and throwaway repo) | 3.5 | 3.5 |
| **Stage 0 total** | | **35.25** | **25.0** |

**Cuts from full scope to committed.** These are decided now, not at the deadline:
1. **P1-1, P1-2 and P1-3** (A-01, A-02, A-03, plus C-02, which fails on HEAD until A-03 lands): −4.5 h, moved to v1.3.5. This applies STRATEGY §5's own rule ("If Stage 0 projects over 25h, P1-1, P1-2 and P1-3 … move to v1.3.5"). The banner lists the workarounds. **Overtaken 2026-09-27:** all four shipped early, on `main` for v1.3.4 (PR #1; C-02's CI guard in PR #3).
2. **Toolchain bump** (A-17b: supported Go, `x/*` refresh, CI pins): −3.5 h, moved to v1.3.5. **This defers an item STRATEGY R2 put in v1.3.4; STRATEGY §5 records the deferral as needing the maintainer's sign-off.** govulncheck (A-17a) stays in v1.3.4. If it reports a vulnerability reachable from shipped code, A-17b returns to v1.3.4 and cut 6 pays for it. **Overtaken 2026-09-27:** A-17b shipped early (Go 1.26, `toolchain go1.26.8`; PR #1), so nothing was deferred.
3. **`recover` refusals** (A-06b, P1-10; not in STRATEGY's 0.3 table): −0.5 h, moved to v1.3.5. The banner lists both actions as known broken. **Overtaken 2026-09-27:** A-06b shipped early (PR #1).
4. **Name check (H-11) and git-history check (H-06):** −1.5 h, moved to Stage 1. H-06 must still finish before the first outreach, the C5 thread replies, because STRATEGY §4 requires it "before announcing". H-11 must finish before C8.
5. **CHANGELOG reconstruction of v1.3.0–v1.3.2** (H-04b): −0.25 h, moved to Stage 1.
6. **Contingency, if the G-1 log shows an overrun at C2 or A-17b returns:** first move A-05 (P0-5) to v1.3.5 with a banner line (−1 h; cloud is already behind `--experimental`, A-15). Then drop A-16's pin bump (−0.25 h), keeping its `--disableupdate` invariant test; auto-update keeps 2.334.0 routable [I], which V-3 tests. **Overtaken 2026-09-27:** A-05 and A-16's pin bump both shipped (PR #1), so neither is left to cut.

Stage 0 has zero slack. The likeliest overrun is 0.3b, and its fallback (A-21) is paid for inside the box.

**Provisional Stage 0 spend (2026-09-27).** Cuts 1–3 were overtaken: A-01, A-02, A-03, C-02, A-06b and A-17b shipped early (§A.0 rows 7–9, 11 and 44). Their 8.5 h of backlog estimates (1.5 + 1.5 + 1 + 0.5 + 0.5 + 3.5) therefore count as Stage 0 spend, not as capacity freed after launch: on estimates, Stage 0 is 33.5 h against the ≤25 h C4 cap, and the pre-launch cumulative figures below rise by the same 8.5 h (to 63.0 h against the ≤55 h cap). The post-launch table already includes them. These are estimates, not logged hours; the G-1 log replaces them at the C2 re-baseline.

*Stage 1 before launch (C4 → C8; 30 h available under the ≤55 h cap)*

| Block | Items (hours) | Hours | Cumulative |
|---|---|---:|---:|
| Stage 0 committed | | | 25.0 |
| Carried from Stage 0 | H-06 0.5 (before C5), H-11 1 (before C8), H-04b 0.25 | 1.75 | 26.75 |
| Repo and CI hygiene | H-12 0.5, H-14 0.5, C-01b 0.25, C-12 0.25 | 1.5 | 28.25 |
| W3 recipes and 4 thread replies (E1.3), by C5 | recipes 4.5, replies 1.5 | 6.0 | 34.25 |
| W1 `runnerkit-watch` (E1.1), by C7 | | 12.0 | 46.25 |
| Launch (E1.4) | prep 2, posting 2 | 4.0 | 50.25 |
| Discovery interviews (E1.5), first half | | 2.5 | 52.75 |
| Metric definitions (E1.6, part 1) | | 0.5 | 53.25 |
| Weekly table (H-16), 5 weeks | | 1.25 | **54.5** (cap 55) |

**W1–W3 build cap (≤30 h):** W3 6 + W1 12 + the W2 minimum 12 = 30.0, with zero slack. If W1 logs more than 12 h, W2 is not built this stage; the PASS gate's Watch branch still applies.

**W2 does not fit before launch** (54.5 + 12 = 66.5 h > 55 h). If W2 gets a go at C7 or C9, build it **after** launch and ship it by 2026-12-07, as STRATEGY C7 now says. (An earlier STRATEGY draft said "if go, W2 by 2026-11-16"; with these estimates that date and the ≤55 h cap cannot both hold, and the cap is a kill criterion.) Build W2 before launch only if the C7 re-baseline shows the logged hours plus W2 still fit under 55 h.

*Stage 1 after launch (C8 → C10; 30.5 h available under the ≤85 h cap, 22.0 h once the provisional Stage 0 spend in §0 is counted)*

| Block | W2 no-go | W2 go |
|---|---:|---:|
| W2 checkup minimum (E1.2): core 4, F-01 1.5, F-02 1, F-04 2, F-05 1.5, fixture 1.5, C-10 0.5 | — | 12.0 |
| H-15 `byo-prepare` sweep 0.5, H-17 archive playbook 0.5 | 1.0 | 1.0 |
| Discovery interviews (E1.5), second half | 2.5 | 2.5 |
| Weekly table (H-16), 6 weeks | 1.5 | 1.5 |
| Decision memo (E1.6, part 2) | 2.5 | 2.5 |
| Support, fixes and thread follow-ups (72 h responses) | 14.5 | 8.0 |
| **Block total / cumulative at C10** | 22.0 / **85.0** | 27.5 / **90.5** |

A-01, A-02, A-03, C-02, A-06b and A-17b left this table on 2026-09-27 (shipped early). The 8.5 h this frees in the no-go column (3.0 h in the go column) is **not** free capacity: the cumulative figures count it in Stage 0 (provisional, see above). The go column had A-02, A-06b and A-17b after the memo, so on estimates it now ends 5.5 h over the ≤85 h cap.

**Cut from Stage 1 entirely** (after the memo, or never): C-04 lint (2 h), C-06 shellcheck (1.5 h), C-11 docs contract (1 h), the multi-fixture C-08 minimum (sized M), F-03 (3 h), F-06..F-08 (5 h), A-11 (1 h), A-12 (1.5 h), A-13 (2 h), and, inside W2, container discovery and the anonymized summary block (STRATEGY W2's own first cuts). If W2 is a no-go, F-01, F-02, F-04 and F-05 are not built either.

## Overview: now, next, later

| Window | Dates (STRATEGY §10.1) | Items | Exit condition |
|---|---|---|---|
| Stage 0: stabilize | 2026-09-26 → C4 2026-10-17 | G-0 (C1), H-01..H-05, H-07..H-10, H-13, H-16; §A.1 (A-04..A-10, A-15, A-16, A-17a); §A.2 (A-14, A-18..A-20, or A-21); V-1..V-4 | **C2 2026-10-10:** LICENSE, banner, corrected notes and the security page with its revocation section. v1.3.4 tagged (0.3b repair or option-A refusal), or the cask deprecated (H-13). **C3 2026-10-12:** 15-day offline test started (V-1). **C4:** V-2..V-4 and the V-1 docs grep published; ≤25 h |
| Stage 1: pre-gate | C4 → C5 2026-10-24 | H-06 (before the replies), E1.3 (W3) recipes and 4 thread replies | Posted by C5. V-1 offline readout by C6 (2026-10-27) |
| Stage 1: build | C5 → C7 2026-11-02 | E1.1 (W1), H-11, H-04b, H-12, H-14, C-01b, C-12 | W1 released at C7; W2 go/no-go #1 |
| Stage 1: launch | week of 2026-11-09 (C8, latest 2026-11-16) | E1.4, E1.5 (first half), E1.6 definitions | 4+ channels plus the awesome-runners PR; cumulative ≤55 h |
| Stage 1: measure | C8 → C10 2026-12-21 | W2 go/no-go #2 at C9 (2026-11-23). If go at either date: E1.2 with F-01, F-02, F-04, F-05, A-01, A-03 and C-02, as v1.3.5 by 2026-12-07. If not: the v1.3.5 CLI fixes alone. E1.5, E1.6 memo | W1–W3 build ≤30 h (C9). Memo on 2026-12-21, ≤85 h: PASS, KILL or grey zone (extends to C11, 2027-02-01, with no new features) |
| Stage 2: extend (only on PASS) | 2027-01-04, or 2027-02-02 after a grey-zone pass → 2027-03-26 (C12) | E2.0 item 0 (R-01, R-05, C-07b, C-08 full) → E2.1 heartbeat (its no-root variant may come before E2.0) → E2.2 hygiene → E2.3 adopt → E2.4 `up` rebuild (only with ≥5 setup requests) → E2.5 deletion; C-04, C-05 (blocking), C-06, C-09, C-11 | Stage 2 gate memo on 2027-03-26 (E2.6) |
| Stage 3: expand (trigger-gated) | 2027-04 → 2027-09-26 (C13) | E3.1..E3.7, Z-* | Each bet's own trigger |

---

## Scope decision for v1.3.4 (read before §A)

After ruling R1, STRATEGY §5 defines v1.3.4 as a **harm-reduction release plus a time-boxed minimal core-path repair**, tagged by C2 (2026-10-10):
- **0.3 harm reduction (§A.1).** No command may destroy a registration, orphan a billing VM, show an invented price, or run cloud or ephemeral BYO without `--experimental`.
- **0.3b repair (§A.2), ≤6 maintainer-hours:**
  - `install.sh` carries the sudoers list that `RenderSudoersEntry` already defines (P0-1);
  - the runner user is created before image setup (P0-2);
  - failures name their step (P1-15);
  - one real job runs on a containerized password-sudo host prepared only by `install.sh` (A-20).
- **Fallback, option A (A-21).** If the box is exceeded, BYO `up`/`register` refuse to run without `--accept-known-issues`, and the banner says BYO is not supported in this release.

**What v1.3.4 does not do** is narrow the sudoers. `install.sh` gains lines but no capability, because the `RenderSudoersEntry` list was already root-equivalent (SEC-1), and `RenderSudoersEntry` gains no entries. The privilege model is fixed only in Stage 2 item 0 (E2.0). The wider BYO rebuild (the R-items outside item 0) stays gated on ≥5 concrete setup requests.

"The core path works" in v1.3.4 therefore means one of two things, and the banner (H-08) must say which one shipped:
- **0.3b passed.** On a fresh password-sudo Ubuntu 24.04 x86_64 host prepared only by the v1.3.4 `install.sh`, `up` brings a runner online, and it runs a job with gcc and, as the runner user, `docker run hello-world`. Hosts prepared by an older `install.sh` must re-run the new one. Until they do, `up` fails at a named step (A-19), not at "(unknown)".
- **Option A.** BYO `up`/`register` refuse immediately and point to the manual `config.sh` alternative. Nothing is claimed to work.

Every other shipped command either does what it says or refuses clearly, with the manual workaround. The stated exception is P1-1..P1-3 (silent CLI errors, the missing `Input` prompter, writes into the CWD). They move to v1.3.5 under STRATEGY §5's over-budget rule (§0 cut 1), and the banner lists them.

Every P0 and P1 defect still appears in §A.0 with its target release.

---

## A. Release v1.3.4, and where every other P0/P1 lands

### A.0 Traceability: every P0/P1 and SEC defect → backlog item

Severity is CODEBASE-ASSESSMENT's: §5 for P-IDs, and the §6.6 "Priority" column for SEC-IDs. Where KEYFACTS gives a different label, it is noted. *Earlier drafts of this table followed KEYFACTS for P1-1..P1-3, which KEYFACTS labels P0 (B.6–B.8). That mis-sorted them against the assessment, whose P1 is what STRATEGY's over-budget rule relies on.* Rows without a P-ID are engineering-quality findings from assessment §8 that carry no severity ID.

| # | ID | Defect | Sev | Disposition | Items | Release |
|---|---|---|---|---|---|---|
| 1 | P0-1 | BYO password-sudo path broken: install.sh lacks 16 sudoers paths; `byo-prepare` deleted but still in the notes [R] | P0 | Sync install.sh to `RenderSudoersEntry` with a full-body test (0.3b), or refuse BYO (option A); erratum; helper in Stage 2 item 0 | A-14, A-20/A-21, H-08, H-10 → A-13 (probe for hosts prepared by an older install.sh), R-01, C-07b | v1.3.4 / v1.3.5 (probe, should) / Stage 2 item 0 |
| 2 | P0-2 | `setup_runner_image` runs before `create_runner_user`, so no docker group and no Rust [R] | P0 | Reorder and bump `ImageSetupVersion` (0.3b); Docker as a disclosed opt-in in the rebuild | A-18, H-08 → R-02 | v1.3.4 (unless option A) / v1.4.0 |
| 3 | P0-3 | `upgrade-runner` / `doctor --fix` wipe credentials and run `config.sh --token ""`; `--skip-old-files` [R] | P0 | Disable now; real fix later | A-06a → R-03 | v1.3.4 / v1.4.0 |
| 4 | P0-4 | `down`/`unregister` has no cloud guard, so VMs are orphaned [R] | P0 | Refuse | A-04 | v1.3.4 |
| 5 | P0-5 | `up --cloud --replace` overwrites cloud state [C] | P0 | Refuse | A-05 | v1.3.4 (first contingency cut, §0 cut 6) |
| 6 | P0-6 | Hard-coded "approx €4.90/month" [R]; current prices [S] | P0 | Live price from the Hetzner API, labelled as such; v2 migration and availability frozen | A-07 → Z-04 | v1.3.4 |
| 7 | P1-1 | Cobra errors swallowed; `--version` silent [R] | P1 (KEYFACTS B.6: P0) | Fix; shipped early (§0 cut 1 overtaken) | A-01, C-03 | v1.3.4 |
| 8 | P1-2 | Prompter has no `Input` [R] | P1 (KEYFACTS B.7: P0) | Fix; shipped early (§0 cut 1 overtaken) | A-02 | v1.3.4 |
| 9 | P1-3 | `StateBaseDir` unset: `config.json` and `sessions/` land in the CWD [R] | P1 (KEYFACTS B.8: P0) | Fix; shipped early (§0 cut 1 overtaken) | A-03, C-02 | v1.3.4 |
| 10 | P1-12 | Ephemeral cloud VM never destroyed; TTL ignored; finalizer unprivileged [R/C] | P1 | Block ephemeral cloud; ephemeral BYO behind `--experimental`; frozen | A-08, A-15 → Z-03 | v1.3.4 |
| 11 | P1-10 | `recover` reinstall/reregister miswired [C/I] | P1 | Refuse both actions (shipped early; §0 cut 3 overtaken); fix with the units | A-06b → R-05 | v1.3.4 / Stage 2 item 0 |
| 12 | P1-5 | Auto-detection on by default turns comment words into packages [R] | P1 | Off by default; parser frozen | A-11 → Z-07 | after the memo (cut) |
| 13 | P1-6 | arm64 advertised but broken [C] | P1 | Withdraw the docs claim now; preflight refusal later. v1.3.4 progress: docs claim withdrawn; preflight warns on arm64 but does not refuse (PR #1) | H-09, A-12 → Z-05 | v1.3.4 (docs) / after the memo |
| 14 | P1-7 | Non-Ubuntu passes preflight, then fails [C] | P1 | Banner says Ubuntu x86_64 only; preflight refusal later | H-08, A-12 → Z-06 | v1.3.4 (docs) / after the memo |
| 15 | P1-9 | `logs` and OOM hints query the wrong unit [R] | P1 | Fix if W2 gets a go; checkup reuses it | F-01 | v1.3.5 |
| 16 | P1-19 | Health commands exit 0 on ERROR; `ok:true` hard-coded [R] | P1 | `--fail-on` for checkup if W2 gets a go; doctor and status later; no JSON unification (N-13) | F-02 | v1.3.5 |
| 17 | P1-20 | Doctor false positives when facts are missing; blind to sudoers drift [R] | P1 | Tri-state facts (cut from Stage 1); content check later. v1.3.4 progress: with GitHub or SSH facts missing, doctor and recover report `github_unavailable` or the SSH problem instead of teardown advice (PR #1); no tri-state stage or drift check yet | F-03 → R-17 | after the memo / v1.4.0 |
| 18 | SEC-4 | Host-key pin not bound to the session; non-standard fingerprint [R] | P1 | Strict `known_hosts` in checkup (W2); every command later | F-05 → R-06 | v1.3.5 / v1.4.0 |
| 19 | P1-25 | No state lock; no signal handling [R] | P1 | Stretch (cut from Stage 1) | F-06 | after the memo |
| 20 | P1-13 | Cleanup cannot converge on a dead host; partial cleanup exits 0 [R] | P1 | `state forget` plus an exit code (cut); cloud part frozen | F-07 → Z-09 | after the memo |
| 21 | P1-23 | Mutating day-2 commands ignore `busy` [C] | P1 | Busy gate (cut); upgrade later | F-08 → R-03 | after the memo / v1.4.0 |
| 22 | SEC-1 | "Scoped" sudoers is root-equivalent [C] | P1 | Honest comment and security page now; helper in item 0. v1.3.4 progress: both merged (PR #1); `TestRenderSudoersEntryGolden` pins the list (PR #3) | A-09, H-09 → R-01 | v1.3.4 / Stage 2 item 0 |
| 23 | SEC-2, SEC-3 | Runner-writable `svc.sh` run by root; `/var/lib/runnerkit` runner-owned [C] | P1 | Disclose, with revocation steps, now; root-owned units in item 0 | H-09 → R-05 | v1.3.4 (disclose) / Stage 2 item 0 |
| 24 | SEC-5 | Docker group is root-equivalent, silently [C] | P1 (decide with P0-2) | A-18 makes the group work, so disclosure in the same release is mandatory; opt-in later | H-08, H-09 → R-02 | v1.3.4 / v1.4.0 |
| 25 | SEC-6 | Registration tokens in argv and sudo logs [C] | P2 (medium/low; verifier byo-bootstrap-15 confirmed/low) | Disclose; fix with the helper | H-09 → R-07 | Stage 2 |
| 26 | SEC-7 | All repos on a host share one Unix user [C] | P2 (P1 with mixed trust tiers) | Disclose; per-repo users | H-09 → R-08 | Stage 2 |
| 27 | SEC-8, SEC-9, SEC-10 | Trust gate not rechecked; auth precedence inverted and docs push the `workflow` scope; split redaction and 0644 logs [C] | P2 | Remove the `workflow`-scope advice now (H-09); the rest in the rebuild. v1.3.4 progress: advice removed; token remediation says a logged-in `gh` wins over `RUNNERKIT_GITHUB_TOKEN` (SEC-9), precedence unchanged (PR #1) | H-09 → R-18 | v1.3.4 (docs) / Stage 2 |
| 28 | SEC-11 | Cloud SSH open to 0.0.0.0/0; cloud admin `NOPASSWD:ALL` (intentional, assessment §6.1) [C] | P2 (SEC-11 low) | Disclose; cloud frozen; fixed only if cloud is revived. v1.3.4 progress: disclosed in Known issues; `--ssh-allowed-cidr` accepts only an IPv4 CIDR (a bare address opened 0.0.0.0/0; PR #3); the default is unchanged | H-09 → Z-10 | Stage 3 trigger |
| 29 | SEC-12 | Opt-in CI apt sudo is root-equivalent, undocumented [C] | P2 | Disclose in banner and security page; `--ci-sudo` profiles later | H-08, H-09 → R-09 | v1.3.4 / v1.4.0 |
| 30 | SEC-13 | Release workflow on mutable action tags, no repo guard, no test gate [C] | P2 | Test gate and repo guard in v1.3.4; SHA pinning in Stage 1 (STRATEGY 0.2) | A-10 (C-01), C-01b | v1.3.4 / Stage 1 |
| 31 | P1-11 | Job user has no sudo (cloud) or only opt-in apt (BYO) [R] | P1 | `--ci-sudo` profiles; cloud frozen | R-09, Z-08 | v1.4.0 |
| 32 | P1-14 | BYO plan not disclosed; heavy image forced; 2 GiB disk check against a 4.5–5 GB need [R] | P1 | Disclose in the README now; minimal profile later. v1.3.4 progress: README Known issues disclose the footprint; the BYO plan lists `setup_runner_image`, its apt sources and the docker grant (PR #1); the image and the 2 GiB check are unchanged | H-08 → R-10 | v1.4.0 |
| 33 | P1-15 | Failures lose the failing step ("(unknown)") [R] | P1 | Fix (0.3b) | A-19 | v1.3.4 |
| 34 | P1-16 | Image marker freezes partial failures; fetch failures abort [R] | P1 | Fix. v1.3.4 progress: a failed Geckodriver lookup no longer aborts the step (PR #1); the Go lookup and the single marker remain | R-12 | v1.4.0 |
| 35 | P1-17 | Tarball cache verified only on first download; duplicate 225 MB download [R] | P1 | Fix | R-13 | v1.4.0 |
| 36 | P1-18 | `register` dead end after install.sh [C] | P1 | Fix. v1.3.4 progress: `lifecycle_foundation_missing` names the `runnerkit up` command to run first, not install.sh (PR #4); install.sh still does not create the runner user | R-14 | v1.4.0 |
| 37 | P1-21 | BYO rerun mutates the host before the state-replace gate [C] | P1 | **Closed.** Fixed early: the replace check runs before host-key, preflight and bootstrap (PR #1; `TestUpExistingStateRefusesBeforeBootstrapWithoutReplace`) | R-15 | v1.3.4 |
| 38 | P1-24 | apt: no lock timeout; `DEBIAN_FRONTEND` dropped [C] | P1 | Fix | R-16 | v1.4.0 |
| 39 | P1-4 | cloud-init `set -e` makes the tolerance dead code; 15-min retry-all [R] | P1 | Frozen | Z-01 | Stage 3 trigger |
| 40 | P1-8 | Hetzner SSH key re-uploaded on every provision [I] | P1 | Frozen. STRATEGY §5 allows it in v1.3.4 "only if hours remain"; §0 shows none | Z-02 | Stage 3 trigger |
| 41 | P1-22 | One runner per repo, so matrix legs serialize [C] | P1 | Stage 3 bet | E3.1 | Stage 3 trigger |
| 42 | P2-9 (with P2-8) | Runner version model not grounded in the host; pin 2.334.0; version-stale finding never armed | P2 | Pin bump and `--disableupdate` test now; host-read version in checkup; resolve-latest in the rebuild | A-16, F-04, R-04 | v1.3.4 / v1.3.5 / v1.4.0 |
| 43 | — | hcloud-go v1.59.2; the API dropped `server.datacenter` (reads are nil-guarded at `provision.go:224,519-522`; no code calls `/v1/datacenters`) [C] | No P-ID; assessment §8.3 rates the impact low (KEYFACTS: P1; verifier cloud-ephemeral-14: partially true/low) | Frozen | Z-04 | Stage 3 trigger |
| 44 | — | Go 1.22 out of support; x/net from 2023; no lint, vuln or shellcheck gates [C] | No P-ID (assessment §8.2–8.3) | govulncheck and the toolchain bump (Go 1.26; §0 cut 2 overtaken) in v1.3.4; lint and shellcheck after the memo | A-17a, A-17b, C-04, C-05, C-06 | v1.3.4 (A-17a, A-17b) → Stage 2 (C-04..C-06) |
| 45 | — | Tests never execute the rendered shell; no smoke ever runs a job [C] | No P-ID (assessment §8.1) | One real job before v1.3.4; fixtures and a CI canary later | A-20, C-08, C-09 | v1.3.4 → Stage 2 |

The remaining P2 items are N-10 (P2-1, P2-2, P2-7), Z-08 (P2-13) and N-22 (the rest).

### A.1 v1.3.4 must-ship: harm reduction (STRATEGY 0.3). 9.0 h committed, including the release run

Order of work: A-10 first (so every later commit is test-gated), then A-04, A-05, A-06a, A-08 and A-15 (the billing and credential guards), then A-07, A-16, A-17a and A-09. A-01..A-03 were in this section before the re-baseline; they now sit in §A.3 (§0 cut 1).

- [x] **A-04 `down`/`unregister` refuses cloud state** (P0-4; KEYFACTS B.4) [R]
  - **Files:**
    - `internal/cli/down.go:61-69`: guard goes after `GetRepository` and before `BuildCleanupPlan`. A grep of `down.go` for `provider|cloud|hetzner` finds nothing.
    - Mirror of `internal/cli/destroy.go:77-81`.
    - `internal/ops/doctor.go:142`: `cleanup_pending` always suggests `down --dry-run`.
    - `docs/troubleshooting/cleanup.md`.
  - **Fix:**
    - If `isCloudProvider(repoState.Provider)` (`status.go:221`), call `renderer.Error("wrong_cleanup_command", "This runner is a RunnerKit-managed Hetzner server; down would delete the only record of it while it keeps billing.", ["runnerkit destroy --repo X (verifies deletion with Hetzner)"])` and return exit 2. This also applies to `--dry-run`.
    - In `ops/doctor.go:142`, recommend `destroy --dry-run` when the state is cloud.
  - **Acceptance:**
    - Exit 2 on cloud state.
    - `state.json` is byte-identical afterwards.
    - Zero provider, SSH or GitHub calls.
    - Doctor on pending cloud state points at `destroy`.
  - **Test that would have caught it:**
    - `TestDown_RefusesCloudState` and `TestUnregisterAlias_RefusesCloudState`, using recording fakes and a byte comparison of `state.json`.
    - `TestDoctor_CleanupPendingCloudPointsToDestroy`.

    The dossier's scratch repro becomes the fixture: `down --yes --json` on seeded Hetzner state returned `ok:true, state_removed:true` with no provider call.
  - **Effort:** S (1 h).

- [x] **A-05 `up --replace` refuses live cloud state** (P0-5; KEYFACTS B.4) [C]
  - **Files:**
    - `internal/cli/up.go:1199-1210`: `confirmCloudStateReplaceBeforeProvision` returns `true` on `--replace` with no provider check.
    - `up.go:721-746`: provisioning and save.
    - `up.go:159`: where the BYO path loads `existing`.
    - `up.go:1707-1717` and `:2177`: `confirmStateReplace`.
  - **Fix:**
    - Add a helper `refuseIfLiveCloudState(existing)`. When the saved state is cloud and records any server, firewall or SSH-key ID, fail with `cloud_state_exists`. The message lists those IDs and says "run runnerkit destroy --repo X first".
    - Call it in `runUp` right after `:159`, before host-key, preflight or any mutation, and in `runCloudUp` before `:722`.
    - The interactive "replace owner/name" phrase gets the same refusal.
  - **Acceptance:**
    - Cloud→cloud and cloud→BYO replace both exit 2 before any provider or SSH call.
    - BYO→BYO `--replace` is unchanged.
  - **Test that would have caught it:** `TestUpCloudReplace_RefusesLiveCloudState` asserts `FakeProvider.Provision` calls == 0 and `state.json` unchanged. `TestUpBYOReplace_RefusesLiveCloudState` asserts recording-executor calls == 0.
  - **Effort:** S (1 h).

- [x] **A-06 Disable the broken lifecycle mutators.** A-06a (v1.3.4) covers `upgrade-runner` and `doctor --fix` (P0-3). A-06b (v1.3.5, §0 cut 3) covers `recover --reinstall-service` and `recover --reregister` (P1-10). [R/C/I]
  - **Files and why:**

    | Command | Files | Problem | Part |
    |---|---|---|---|
    | `upgrade-runner` | `internal/cli/upgrade_runner.go:109-146` | `bootstrap.Options` has no `RunnerToken`. `configure_runner` then runs `sudo rm -f .runner .credentials .credentials_rsaparams` and `config.sh --token ""` (`internal/bootstrap/script.go:55-56`) | A-06a |
    | same | `install.go:241`, `script.go:52` | `tar … --skip-old-files` never replaces binaries | A-06a |
    | same | `:150` | Records the bundled pin even so | A-06a |
    | `doctor --fix` | `internal/cli/doctor_fix.go:36-61`, `doctor.go:125-130` | Calls `upgrade-runner` with `yes: true` | A-06a |
    | `recover --reinstall-service` | `internal/cli/recover.go:170` | Runs `svc.sh install` without uninstalling first | A-06b |
    | `recover --reregister` | `recover.go:201-204` then `:233` | Uninstalls the service, then only runs `svc.sh start` | A-06b |

    Docs: `docs/upgrade.md:47-48` calls upgrade-runner "idempotent — safe to re-run".
  - **Fix:**
    - A-06a, `upgrade-runner`: refuse with `command_disabled` (exit 2). Explain the v1.3.3 defect, and that GitHub runners update themselves because RunnerKit never passes `--disableupdate` [I, medium; V-3 tests this]. Print the manual re-register steps.
    - A-06a, `doctor --fix`: refuse, and drop the `upgrade-runner` remediation from `runner_version_stale` (`ops/doctor.go:149-153`).
    - A-06a: fix `docs/upgrade.md`.
    - A-06b, `recover`: keep `restart_service`. Refuse the `reinstall_service` and `reregister_runner` actions with manual steps, and have the recovery planner stop recommending them. Until A-06b ships, the banner (H-08) lists both actions as known broken.
  - **Acceptance:** after A-06a and A-06b, no command other than `up`/`register` ever sends a script that configures a runner (`config.sh --unattended`), deletes `.runner`/`.credentials`, or runs `svc.sh install`. `down`/`unregister` and `destroy` still legitimately send `config.sh remove` (`RenderRemoveConfigScript`, `script.go:265-276`; callers `down.go:291`, `destroy.go:274`) and `svc.sh uninstall` (`down.go:445-447`). The invariant must allow those.
  - **Test that would have caught it:** an invariant test in `internal/cli/lifecycle_invariants_test.go`. It runs every subcommand except `up`/`register` against seeded BYO state with a recording executor, and fails if any issued script matches `config\.sh --unattended|rm -f [^\n]*\.credentials|svc\.sh install`. A bare `config\.sh` pattern would wrongly fail `down`/`destroy`. On HEAD it fails for `upgrade-runner` and `recover`. In v1.3.4 the test carries an explicit, commented exemption for `recover`, which A-06b removes. Also add `TestUpgradeRunner_Disabled_NoRemoteCalls` and `TestDoctorFix_Disabled`.
  - **Effort:** S. A-06a 1 h; A-06b 0.5 h. The real fixes are R-03 and R-05.

- [x] **A-07 Show the live Hetzner API price instead of the hard-coded cost** (P0-6) [R]; current prices [S]
  - **Files:**
    - `internal/provider/profile.go:27-28` and `:113-118`: "approx €0.0081/hour" and "approx €4.90/month", applied to every type and region.
    - `internal/provider/hetzner/provision.go:300-301,496`.
    - `internal/provider/hetzner/client.go:53-65`: `ServerType.GetByName` already returns the type with its per-location `Pricings`, which are discarded (assessment P0-6). The `Client` interface (`:11-42`) gains a pricing method.
    - Plan rendering in `internal/cli/up.go` (`renderCloudProvisionPlan`, `:1093`; JSON key `estimated_monthly_cost`, `:342`).
    - hcloud-go v1.59.2 exposes both inputs, as I checked in the module source: `ServerType.Pricings` (`ServerTypeLocationPricing`, `hcloud/pricing.go:103`) and `PricingClient.Get` → `Pricing.PrimaryIPs` (`pricing.go:14,68,137`). No v2 migration is needed for this.
  - **Fix:**
    - Delete both cost constants. No fallback constant exists, and the dossier's secondary-source prices must never be printed (STRATEGY §9).
    - Monthly cost = the server type's monthly price for the chosen location, plus the primary IPv4 monthly price for that location. Hourly cost is computed the same way. Show net and gross exactly as the API returns them, labelled "reported by the Hetzner API at <fetch time>; excludes traffic overage".
    - If the chosen type has no price entry for the chosen location, refuse with `cloud_location_unpriced` (exit 2): "Hetzner reports no price for <type> in <location>; choose another --cloud-region".
    - JSON: `estimated_monthly_cost: {amount, currency, source: "hetzner_api", fetched_at}`.
  - **Acceptance:** the cloud plan contains no amount that does not come from the API response. A location with no price entry refuses before any create call.
  - **Test that would have caught it:**
    - `TestCloudPlan_PriceFromAPI`: a fake client returns different pricings for `(cpx22, nbg1)` and `(ccx63, sin)`, and each plan shows its own figure. On HEAD both print €4.90, which matches the dossier's scratch repro.
    - `TestCloudPlan_UnpricedLocationRefuses`: zero create calls.
    - `TestNoHardcodedCost`: a grep of non-test Go for `4\.90|0\.0081|approx €`.
  - **Effort:** S (2 h). Availability (`Locations[].Available`), which needs hcloud-go v2, stays in Z-04.

- [x] **A-08 Block ephemeral cloud, and stop recommending it for untrusted code** (P1-12) [R/C]
  - **Files:**
    - `internal/cli/up.go:115` (`resolveModeDecision`); `runmode.ProfileEphemeralCloud` branches at `:479,:1139`.
    - Copy at `internal/github/safety.go:22`, `internal/cli/status.go:102`, `up.go:2019` and `up.go:2130`.
    - `docs/safety.md:9-14,84-88`.
    - `docs/cloud-quickstart.md:70`.
  - **Fix:**
    - When the safety profile is ephemeral-cloud, fail with `ephemeral_cloud_disabled` (exit 2) **before** `Provider.Validate`: "the VM is never destroyed after its job and keeps billing (known issue); for untrusted or public code use GitHub-hosted runners (free and unlimited for public repos)".
    - Rewrite all copy that sends public or untrusted repos to `--mode ephemeral --cloud hetzner` so it says GitHub-hosted instead.
    - Label BYO ephemeral "experimental; not isolation".
  - **Acceptance:** `up --mode ephemeral --cloud hetzner --dry-run` exits 2 with zero provider calls, and no user-facing string recommends ephemeral cloud.
  - **Test that would have caught it:** `TestUp_EphemeralCloudDisabled` asserts `FakeProvider.Validate` calls == 0. `TestNoCopyRecommendsEphemeralCloud` greps Go string literals and `docs/**/*.md` for `--mode ephemeral --cloud`.
  - **Effort:** S (1 h).

- [x] **A-15 `--cloud` and ephemeral BYO behind `--experimental`; explicit `--cloud-region`** (STRATEGY 0.3 and §4 de-emphasis; P0-6 defaults; P1-12) [C]
  - **Files:**
    - `internal/cli/up.go:87-88`: `--cloud` help ("recommended cloud provider; only hetzner is supported in Phase 4"), and `--cloud-region`, which defaults to `provider.HetznerDefaultRegion`.
    - `internal/provider/profile.go:10-11`: `fsn1` and `cpx22`. fsn1 reportedly had no orderable types on 2026-08-25 [S].
    - `up.go:115` (`resolveModeDecision`), and the `register` command's mode handling.
    - `internal/cli/wizard.go:67`.
    - `docs/cloud-quickstart.md`.
  - **Fix:**
    - Add an `--experimental` flag to `up` and `register`. Without it, any `--cloud` value, and `--mode ephemeral` on BYO, fail with `experimental_required` (exit 2). The message says why: "billed by Hetzner; unsupported" or "not isolation; broken on install.sh hosts".
    - Run the check in `PreRunE`, before the first GitHub call (`up.go:109`, assessment P2-20), so the refusal is offline.
    - `--cloud-region` has no default. With `--cloud` set and no region, fail with `cloud_region_required` and tell the user to check where their server type is offered. Do not hard-code a location list. `cpx22` stays the default type, because A-07 now shows its live price.
    - The wizard's cloud option (relabelled by A-09) prints its command with `--experimental --cloud-region <choose>`.
    - A-08 still blocks ephemeral cloud, even with `--experimental`.
  - **Acceptance:**
    - `up --repo o/r --cloud hetzner --dry-run` exits 2 with zero provider and GitHub calls.
    - Adding `--experimental` without `--cloud-region` still exits 2.
    - `up --repo o/r --host x --mode ephemeral` without the flag exits 2 with zero SSH calls.
    - The BYO persistent path is unchanged.
  - **Test:** `TestUp_CloudRequiresExperimental`, `TestUp_CloudRequiresRegion` and `TestUp_EphemeralBYORequiresExperimental`, with recording fakes that assert zero calls.
  - **Effort:** S (0.5 h, sharing A-08's gate code).

- [x] **A-09 Correct misleading strings (STRATEGY §4 stop-list)** [C]
  - **Files:**
    - `internal/cli/wizard.go:67`: "Hetzner cloud (recommended default)" becomes "Hetzner cloud (experimental; billed by Hetzner)".
    - The other "recommended cloud" strings, found in the repo during fact-check: `internal/cli/up.go:87` (the `--cloud` flag help, "recommended cloud provider"), `:644` ("Provision recommended cloud runner (Hetzner)") and `:655` ("…to provision the recommended cloud runner."). Drop "recommended" from each.
    - `internal/bootstrap/sudoers.go:65-66`: "NOT a blanket NOPASSWD ALL" becomes "Root-equivalent: su, tee, cp, apt-get, systemctl etc. with any arguments allow a root shell; see docs/security-posture.md". This is a comment-only change and does not touch the allowlist. The text is at `:65-66` at HEAD (STRATEGY §4 now cites the same lines). Also fix the preceding comment's promise that "v1.4.0 (install.sh pivot) will tighten these" (`:62-63`), since item 0 (R-01) replaces the list instead.
    - `docs/troubleshooting/bootstrap.md` "scoped to RunnerKit bootstrap commands only".
    - `.goreleaser.yaml` `homebrew_casks.description` ("Reliable GitHub Actions self-hosted runners for solo developers") becomes "CLI to set up and check GitHub Actions self-hosted runners (experimental)".
  - **Test:** in v1.3.4, a small Go test over non-test string literals and `.goreleaser.yaml` fails on `recommended default`, `recommended cloud` and `NOT a blanket`. The full docs and copy contract (C-11) comes after the memo.
  - **Effort:** S (0.5 h).

- [x] **A-10 The release workflow runs tests before GoReleaser, with a repository guard** (SEC-13, test-gate part). This is C-01's v1.3.4 half. SHA-pinning the actions is C-01b in Stage 1, as STRATEGY 0.2 schedules it. **Effort:** S (0.5 h).

- [ ] **A-16 Bump the runner pin to 2.337.0, and test that `--disableupdate` is never passed** (P2-9; gaps.md runner-version-enforcement) [C]
  - **Files:**
    - `internal/bootstrap/package.go:5`: `RunnerVersion = "2.334.0"`.
    - `package.go:23-34`: the x64 and arm64 URLs and SHA-256s.
    - A new test file in `internal/bootstrap`.
  - **Fix:** set 2.337.0, with the SHA-256s from the release-body markers, per gaps.md. Re-verify both against the release page when implementing:
    - x64: `70920811a4f8ad4328818682bca5c6469c1c942fab52448868071d0063816613`;
    - arm64: `9b1dc70626422526e3c94767cf024896beb15da5342a3f4819bf2feac13e0393`.
  - **Tests:**
    - `TestRenderedScriptsNeverDisableUpdate` renders every install, ephemeral-install and reconfigure script (persistent and ephemeral, with and without extra packages) and fails on `--disableupdate`. This is the written invariant from H-14 and N-15.
    - `TestRunnerPinAtOrAboveRegistrationFloor` asserts the pin is ≥2.329.0.
  - **Acceptance:** in A-20, the runner reports 2.337.0 and takes its first job without a self-update.
  - **Effort:** S (0.5 h). Resolve-latest and the scheduled pin-bump PR stay in R-04. The pin bump is the second contingency cut (§0 cut 6); the test is never cut.

- [ ] **A-17a govulncheck before tagging, plus a scheduled report-only job** (assessment §8.3; the v1.3.4 half of C-05)
  - **Files:** `.github/workflows/pr-checks.yml`, which gains a `govulncheck` job with a weekly `schedule:` trigger.
  - **Fix:** run `govulncheck ./...` on a GitHub-hosted runner. vuln.go.dev was blocked from the research sandbox, so exposure is **unmeasured**, not known to be bad. The job is report-only (`continue-on-error: true`) and posts its findings to the job summary.
  - **Rule:** before tagging, read the result for the release-candidate SHA. A finding **reachable** from shipped code paths (`os/exec` ssh, the `net/http` client, `crypto/tls`) pulls A-17b back into v1.3.4, paid for by §0 cut 6.
  - **Acceptance:** the CHANGELOG links a govulncheck run for the tagged SHA.
  - **Effort:** S (0.5 h). It becomes blocking in Stage 2 (C-05).

### A.2 v1.3.4 must-ship: minimal core-path repair (STRATEGY 0.3b). Hard box of 6 maintainer-hours, including the fallback

**Why this is in v1.3.4 (ruling R1).** The launch sends strangers to this repo, and a core command documented as working but broken destroys trust. The defects are reproduced and small (gaps.md released-byo-e2e). The judges objected to an L-sized correctness release, not to a 6 h fix.

**Box rule.** Log the hours for A-14, A-18, A-19 and A-20 as they are spent. If A-20 is not green by **5.5 logged hours**, stop and ship A-21 (option A) from the remaining 0.5 h. Ship whichever of A-14, A-18 and A-19 passed their own tests: without the BYO claim they are harmless, and the banner discloses what they change (SEC-5 for A-18). Neither path narrows the root-equivalent sudoers.

- [x] **A-14 Generate `install.sh`'s sudoers block from `RenderSudoersEntry`, with a full-body equality test** (P0-1; this is C-07(a), pulled forward from Stage 2) [R]
  - **Files:**
    - `install.sh:30-46`: `render_sudoers()`, a heredoc with 11 command groups. It is validated with `visudo -cf` at `:55`.
    - `internal/bootstrap/sudoers.go:70-91`: `RenderSudoersEntry`, which additionally grants tee, gpg, mkdir, unzip, usermod, dpkg, add-apt-repository, chmod, cp, cat and ln (16 paths).
    - `install_sh_test.go:24-38`: compares only the header line.
    - `docs/byo-quickstart.md:35`: already claims the fuller list (assessment §8.5); after A-14 the claim is true.
  - **Fix:**
    - A `go generate` step (e.g. `internal/bootstrap/gen_installsh.go`) rewrites the heredoc body between `# BEGIN runnerkit-sudoers` and `# END runnerkit-sudoers` markers from `RenderSudoersEntry("${u}")`.
    - `install.sh` keeps its `visudo -cf` validation.
    - `RenderSudoersEntry` itself is **not edited**: no new entries (STRATEGY §4 stop #2). Its misleading comment is fixed by A-09.
  - **Acceptance:**
    - For user `alice`, install.sh's rendered fragment is byte-identical to `RenderSudoersEntry("alice")`.
    - `visudo -cf` accepts it on Ubuntu 24.04.
    - In CI, `go generate ./... && git diff --exit-code` is clean.
  - **Test that would have caught it:** `TestInstallShSudoersMatchesTemplate` replaces the header-only check. It runs `render_sudoers alice` from install.sh in bash and compares the whole body. On HEAD it fails, listing the 16 missing paths.
  - **Honesty note:** this narrows nothing. The list stays root-equivalent (SEC-1), as the banner and security page say.
  - **Effort:** S (2 h).

- [ ] **A-18 Create the runner user before image setup, and bump `ImageSetupVersion`** (P0-2) [R]
  - **Files:**
    - `internal/bootstrap/install.go:117-123` (`Apply`) and `:168-174` (`ApplyEphemeral`): both run `fix_dependencies` → `setup_runner_image` → `create_runner_user`.
    - `image_setup.go:8`: `ImageSetupVersion = "1"`. The marker gate at `:30-35` compares versions, so a bump makes hosts re-run the script.
    - `image_setup.go:101`: `usermod -aG docker … || true`. It sits outside the Docker install guard, so it re-runs.
    - `image_setup.go:69-72`: rustup as the service user.
  - **Fix:**
    - Move `create_runner_user` before `setup_runner_image` in both lists.
    - Set `ImageSetupVersion = "2"`. On existing hosts, the next `up` re-runs the script, which now finds the user. `install_service` (`script.go:68-74`: stop, uninstall, install, start) restarts the service, so it picks up the new group.
  - **Out of scope (stays in R-02):** Docker as an opt-in, and `verify_service` checks as the runner user.
  - **Disclosure (mandatory in the same release):** once this works, every job is root-equivalent through the docker group (SEC-5). The banner and the security page say so.
  - **Test that would have caught it:** an order test **with `OSReleaseID: "ubuntu"`** asserting that `create_runner_user` precedes `setup_runner_image` in both `Apply` and `ApplyEphemeral`. The HEAD order tests never set it (assessment §8.1 item 4).
  - **Acceptance:** in A-20, `id -nG runnerkit-runner` includes `docker`, and `docker run hello-world` succeeds as the runner user.
  - **Effort:** S (1 h).

- [x] **A-19 Failures name the failing step** (P1-15; formerly R-11, pulled forward) [R]
  - **Files:**
    - `internal/bootstrap/install.go:133-141` and `:186-194`: when `exec.Run` returns an error, the raw `*exec.ExitError` is returned.
    - `internal/cli/up.go:2314-2327`: prints "(unknown)".
    - `internal/remote/system.go:206-225`: `sshArgs`.
  - **Fix:**
    - Always wrap the error as `remote.RemoteError{CommandID, ExitCode}`.
    - Rendered bootstrap scripts start with `trap 'echo "RKFAIL:${BASH_COMMAND}" >&2' ERR`.
    - Pass `-o LogLevel=ERROR` to strip the "Permanently added" noise.
    - Include the tail of stdout, which is where apt reports its errors.
  - **Test:**
    - A fake executor returns an `*exec.ExitError` for `setup_runner_image`, and the rendered error names `setup_runner_image` and the `RKFAIL` command.
    - A unit test asserts that every rendered bootstrap script starts with the trap.
  - **Effort:** S (1 h).

- [ ] **A-20 One real job on a containerized password-sudo host prepared only by `install.sh`** (STRATEGY 0.3b(d); the v1.3.4 form of C-09)
  - **Setup:**
    - An Ubuntu 24.04 container with systemd as PID 1, openssh-server, and a user `alice` in `sudo` **with a password**, using key auth. The gaps.md `released-byo-e2e-job` prototype (`jobsim.sh`, `rendered/`) is the starting point.
    - Docker inside the container with `--storage-driver=vfs`, which worked in the prototype.
    - A throwaway private repo, and a fine-grained PAT with Administration: read/write on that repo only.
  - **Run:**
    1. Prepare the host with the release-candidate `install.sh` only.
    2. `runnerkit up --host alice@<container> --repo <throwaway>`.
    3. Dispatch a workflow on the runner's label with `gcc hello.c` and `docker run hello-world`.
  - **Pass:**
    - the runner is online;
    - the job is green;
    - `id -nG runnerkit-runner` contains `docker`.

    Record the run URL in the CHANGELOG. This is a manual run. The CI canary is C-09, in Stage 2.
  - **Reuse:** V-1, V-3 and V-4 use the same container image and throwaway repo (§0 ledger).
  - **Effort:** S (1.5 h).

- [ ] **A-21 Option A fallback: BYO `up`/`register` refuse without `--accept-known-issues`** (only if the box is exceeded)
  - **Files:** `internal/cli/up.go`, in the BYO branch of `runUp` after mode resolution and before any SSH call; the `register` command; the README banner (H-08).
  - **Fix:** fail with `byo_unsupported_release` (exit 2): "BYO is not supported in v1.3.4 (known issues: <link>). Use GitHub's config.sh instructions or myoung34/docker-github-actions-runner. --accept-known-issues proceeds anyway."
  - **Test:** `TestUpBYO_RefusesWithoutAcceptKnownIssues` and the same for `register`, each asserting zero recording-executor calls.
  - **Effort:** S (0.5 h, reserved inside the box).

### A.3 v1.3.5: items moved out of v1.3.4 by the §0 budget, and the old should-ship items

**When v1.3.5 ships:** after launch. If W2 gets a go, it ships bundled with `checkup` by 2026-12-07. If not, it ships once the post-launch ledger allows, before C10. Scheduling depends on the W2 decision:
- **W2 go:** A-01, A-03 and C-02 are W2 dependencies, because checkup reuses the CLI and must not fail silently. A-02, A-06b and A-17b slip to after the memo, unless govulncheck finds a reachable issue (A-17a).
- **W2 no-go:** all of A-01..A-03, C-02, A-06b and A-17b ship in v1.3.5.

A-11..A-13 were "should-ship" and are outside the Stage 1 ledger (§0 "cut from Stage 1 entirely"). Build them after the memo, or earlier only from the support hours.

- [x] **A-01 Print CLI errors, wire `--version`, and make `byo-prepare` a clear tombstone** (P1-1; KEYFACTS B.6 labels it P0) [R]. **v1.3.5 (§0 cut 1).**
  - **Files:**
    - `cmd/runnerkit/main.go:41-45`: only `os.Exit(cli.ExitCode(err))`.
    - `internal/cli/root.go:126-127`: `SilenceUsage`/`SilenceErrors`.
    - `root.go:131-133`: `SetFlagErrorFunc` wraps into an `ExitError` without printing.
    - `root.go:123`: `root.Version` is never set.
    - New file `internal/cli/removed.go`.
  - **Fix:**
    1. Add a process-wide "an error was rendered" flag in `internal/ui` that `Renderer.Error` and JSON error payloads set. In `main`, when `Execute` returns an error and nothing was rendered, print `runnerkit: <err>` to stderr, or in `--json` mode an `{"ok":false,"error":{"code":"cli_usage",...}}` envelope. Cobra's "Did you mean …" text is part of its error string, so it survives.
    2. In `SetFlagErrorFunc`, render `ERROR invalid_flag: <err>` plus `Run 'runnerkit <cmd> --help'` before returning.
    3. Set `root.Version = deps.Version` and `SetVersionTemplate("runnerkit {{.Version}}\n")`.
    4. Add a hidden `byo-prepare` command (`DisableFlagParsing`). It prints: "`byo-prepare` was removed in v1.0.8. The v1.3.3 release notes that mention it are wrong. See the README 'Known issues' section." Exit 2.
  - **Acceptance:** every non-zero exit writes at least one line to stderr naming the cause and a next step. `--version` prints the version and exits 0. Errors that are already rendered are not printed twice.
  - **Test that would have caught it:** C-03 `cmd/runnerkit/cli_contract_test.go`. It builds the binary once with `-X main.version=9.9.9-test`, then table-tests these invocations:
    - `bogus`
    - `status --bogus`
    - `status --json=maybe`
    - `register --extra-packages x`
    - `byo-prepare --host x`
    - `--version`

    It asserts `stderr != ""` whenever the exit code is non-zero, and that `--version` prints exactly `runnerkit 9.9.9-test`. Every row fails on HEAD (re-verified 2026-09-26).
  - **Effort:** S (1.5 h).

- [x] **A-02 Implement `Input` on the production prompter** (P1-2; KEYFACTS B.7 labels it P0) [R]. **v1.3.5 (§0 cut 1); after the memo if W2 is a go.**
  - **Files:**
    - `internal/ui/cli_prompter.go:35-124`: implements Confirm (:44), Select (:70) and Password (:106), but not Input.
    - `internal/ui/prompt.go:17,30`: interface declarations.
    - Anonymous `interface{ Input(...) }` assertions at `internal/cli/up.go:1447,2088,2112,2188` and `internal/cli/destroy.go:112-124`.
    - `internal/ui/cli_prompter_test.go:18`: a nil comparison that can never fail (staticcheck SA4023).
  - **Fix:**
    - Declare `type InputPrompter interface{ Input(context.Context, Prompt) (string, error) }` in `prompt.go`.
    - Implement `(*CLIPrompter).Input`: `readLine`, trim `\r\n`, return the text.
    - Replace the anonymous assertions with `ui.InputPrompter`.
    - Add compile-time guards in non-test code: `var _ InputPrompter = (*CLIPrompter)(nil)`, and likewise for `PasswordPrompter`.
    - Delete the never-failing test.
  - **Acceptance:**
    - In a TTY, `destroy` accepts the typed phrase.
    - A wrong phrase exits non-zero with no provider calls.
    - `up --repo X` with BYO chosen interactively accepts `user@host`.
  - **Test that would have caught it:** `internal/cli/prompts_production_test.go`. It wires `Prompts: ui.NewCLIPrompter(strings.NewReader("destroy owner/repo\n"), &out)`, `TTY.StdinTTY=true`, seeded Hetzner state and `provider.FakeProvider`, then asserts that Destroy is called. A second case with the wrong phrase asserts zero provider calls.

    Why the old tests missed it: they always injected a fake prompter that happened to implement `Input`.
  - **Effort:** S (1.5 h).

- [x] **A-03 Default `StateBaseDir`** (P1-3; KEYFACTS B.8 labels it P0) [R]. **v1.3.5 (§0 cut 1).**
  - **Files:**
    - `internal/cli/root.go:52-113` (`normalizeDependencies` never sets it);
    - `cmd/runnerkit/main.go:23-38`;
    - consumers `internal/cli/userconfig.go:16-40` and `internal/ux/checkliststore/store.go:31-36`, which use `""` verbatim;
    - `internal/cli/update_notice.go:17-20` (a local fallback that becomes redundant).
  - **Fix:**
    - In `normalizeDependencies`: `if strings.TrimSpace(deps.StateBaseDir) == "" { deps.StateBaseDir = rkstate.DefaultBaseDir() }`.
    - CHANGELOG note: users who ran v1.3.x `up --host` inside a repo may have a `sessions/` directory there that contains `user@host`. Tell them to delete it.
  - **Acceptance:**
    - `doctor --ignore <id>` persists to `$RUNNERKIT_STATE_DIR/config.json`, or `~/.local/state/runnerkit/config.json`.
    - No command writes into the CWD.
  - **Tests that would have caught it:**
    1. `TestNormalizeDependencies_DefaultsStateBaseDir`, using `t.Setenv("RUNNERKIT_STATE_DIR", dir)`.
    2. C-02: `git status --porcelain --ignored` is empty after `go test ./...`. This catches the known stray file `internal/cli/sessions/byo-owner_repo__alice_example_com.json`.
    3. A C-03 binary row: CWD is `t.TempDir()`, run `doctor --repo o/r --ignore host_mem_low` against seeded state, then assert the CWD is still empty.
  - **Effort:** S (1 h).

- [x] **A-06b Refuse `recover --reinstall-service` and `recover --reregister`** (P1-10). The specification is in A-06 (§A.1). It moved here by §0 cut 3. **Effort:** S (0.5 h).

- [x] **A-17b Move to a supported Go toolchain** (assessment §8.3; formerly C-13). §0 cut 2 moved it from v1.3.4.
  - **Files:**
    - `go.mod:3` (`go 1.22`) and its `x/*` requirements (x/net v0.12.0, x/sys v0.10.0, x/term v0.10.0, x/text v0.11.0);
    - `go-version: '1.22'` at `.github/workflows/pr-checks.yml:20,46` and `release.yml:19`.
  - **Fix:**
    - Move to the newest Go release that the maintainer confirms is supported at implementation time. 1.26.x is the one tested in gaps.md; "1.27.1 current" is reported but not verified. Add a `toolchain` directive.
    - Refresh `x/*`, then run `go mod tidy`.
    - Run the existing GoReleaser snapshot build in `pr-checks.yml` (assessment §8.2) on the new toolchain as the `--snapshot` dry run.
  - **Evidence:** a scratch bump to 1.26.3 built in 19.5 s, `go vet` was clean and every test passed. The remaining work is the CI pins, "roughly half a day" (gaps.md).
  - **Acceptance:** CI is green on the new toolchain, the snapshot archives pass their assertions, and govulncheck (A-17a) is re-run.
  - **Effort:** S (3.5 h).

- [ ] **A-11 Workflow apt auto-detection off by default** (P1-5) [R]
  - **Files:**
    - `internal/cli/up.go:188` (BYO) and `:681` (cloud) call `autoDetectExtraPackages`.
    - `up.go:2291-2297`: scans `os.Getwd()`; the notice is hidden in JSON mode.
    - `internal/cli/workflow_packages.go:14-17`: an unanchored regex; `-\S+\s+` eats `-t` but not its argument.
  - **Fix:**
    - New `--detect-packages` flag (default `false`) on `up` and `register`.
    - When it is set: print the detected list in every mode (JSON `detected_packages`) and require confirmation or `--yes`.
    - `--extra-packages` is unchanged.
  - **Acceptance:** a workflow with `# needed for the build step` and `apt-get install -t bookworm-backports foo` adds nothing unless the flag is passed.
  - **Test that would have caught it:** `TestUp_AutoDetectOffByDefault` (bootstrap `ExtraPackages` is empty).
  - **Effort:** S (1 h). The parser rewrite is Z-07.

- [ ] **A-12 Preflight states the real support matrix: Ubuntu x86_64 only** (P1-6, P1-7) [C]
  - **Files:**
    - `internal/preflight/checks.go:315-322` (`isRecognizedLinux` accepts 11 distros) and the arch check.
    - `internal/bootstrap/install.go:333-339` (`isUbuntuLike` includes debian and linuxmint).
    - `image_setup.go:60,97,107,119,141,153` (amd64 hard-coded).
    - `docs/runner-platforms.md:11-12` (arm64 on both paths; macOS "Supported").
  - **Fix:**
    - Preflight fails with `host.os.unsupported` for any case other than `ID=ubuntu` on x86_64: "RunnerKit bootstrap is only tested on Ubuntu x86_64; <id>/<arch> fails during image setup (known issue). Use GitHub's config.sh instructions or myoung34/docker-github-actions-runner."
    - `--allow-unknown-linux` stays, relabelled "unsupported; expect failures".
    - Fix the docs rows.
  - **Acceptance and test:** a table test in `internal/preflight/checks_test.go` over (os-release ID, arch) pairs. Only `(ubuntu, x86_64)` passes. On HEAD, 11 IDs and arm64 pass even though bootstrap cannot complete for them.
  - **Effort:** S (1.5 h).

- [ ] **A-13 Preflight privilege-coverage probe: fail fast on hosts prepared by an older install.sh** (detects P0-1 on hosts prepared before v1.3.4) [R]
  - **Files:**
    - `internal/preflight/checks.go:180-218`: only `sudo -n install --version`, which passes on install.sh hosts and gives a false green.
    - `internal/bootstrap/sudoers.go:70-91`: extract the command paths into an exported `RequiredSudoCommands` slice, so the template, the A-14 generator and the probe share one list. The template also feeds cloud-init (`provision.go:360`), so the refactor must leave `RenderSudoersEntry` output byte-identical: add a golden test first. If that cannot be guaranteed, hard-code the list in the probe instead.
    - `internal/cli/installhint.go`.
  - **Fix:**
    - One batched SSH script runs `sudo -n <path> --version` (or `--help` for `usermod` and `add-apt-repository`) for each required command, and prints the ones that are refused.
    - If any is refused, fail with a new `host.privilege.incomplete` code (RKD-BOOT-0xx) that lists them.
    - **Do not** use `sudo -n -l`: gaps.md found it returns 0 for every path because of the password-protected `%sudo` rule.
  - **Acceptance:** on a host prepared only by the v1.3.3 install.sh, `up` fails in preflight in under 15 s and names the refused commands (gaps.md measured 9 refused Apply commands plus cat and cp, 16 paths). A host prepared by the v1.3.4 install.sh (A-14) passes, and so does a NOPASSWD host.
  - **Test that would have caught it:** a fake-executor unit test. v1.3.3 instead fails 39.7 s in, at `setup_runner_image`, with "(unknown)".
  - **Effort:** S (2 h).
  - **Remediation text.** "Re-run the install.sh from RunnerKit v1.3.4 or later. Its sudoers fragment is root-equivalent (`sudo su -`), as it has been since v1.0.8; see docs/security-posture.md." If v1.3.4 shipped option A (A-21) instead, the text says BYO is unsupported.

### A.4 v1.3.5: fixes that `runnerkit checkup` (W2) needs. Built only if W2 gets a go at C7 or C9

These are the fixes STRATEGY §6 W2 lists as "required", in the minimum form the §0 ledger funds: F-01 1.5 h, F-02 1 h, F-04 2 h and F-05 1.5 h, inside W2's 12 h. F-03 and F-06..F-08 are cut from Stage 1 (after the memo). If W2 is a no-go, none of these is built.

- [ ] **F-01 `logs` and OOM hints use the real systemd unit** (P1-9) [R]
  - **Files:**
    - `internal/ops/logs.go:55` and `:152` pass the unresolved `repoState.Machine.ServiceName`. That value comes from `runnerServiceName()` at `internal/cli/up.go:2303-2304` and is never the real `actions.runner.<owner-repo>.<runner>.service`.
    - Resolver: `internal/ops/probes.go:116-150` (`ResolveActionsRunnerSystemdUnit`).
    - `internal/cli/doctor.go:160` discards collection warnings.
  - **Fix:**
    - Resolve the unit before every `journalctl -u`.
    - Kernel evidence: read `-b 0` and `-b -1`.
    - Surface warnings as a `logs_partial` finding. When the SSH user lacks journal access, say "add <user> to systemd-journal or adm" instead of returning an empty section.
  - **Acceptance:** on a host whose real unit differs from the saved name, `logs` shows journal lines, and a synthetic `oom-kill` kernel line yields a `host_incident_hints` entry.
  - **Test that would have caught it:** a fake executor that returns entries **only** for the exact real unit name and `-- No entries --` otherwise. The HEAD fakes answer any unit name.
  - **Effort:** S (1.5 h).

- [ ] **F-02 Health is reflected in exit codes; `--fail-on`** (P1-19) [R]
  - **Files:** `internal/cli/doctor.go:145` (`"ok": true` hard-coded), `status.go`, `list.go`, `internal/cli/exit.go:10-18` (add codes), the docs exit-code table.
  - **Fix:**
    - Add `ExitDegraded=8` (worst finding WARN), `ExitBroken=9` (ERROR or unknown) and `ExitPartial=10` (partial cleanup, used by F-07).
    - Add `--fail-on none|warn|error`, defaulting to `error`. `--exit-zero` is an alias for `none`.
    - JSON `ok` = no finding at or above the threshold.
    - **Stage 1 minimum (1 h):** the codes and `--fail-on` on `checkup` only. Adding them to `doctor` (default `error`) and `status` (default `none`) comes after the memo, because it changes existing exit behaviour.
  - **Acceptance:** `checkup` with an ERROR finding exits 9 with `ok:false`. With only WARN findings it exits 0, or 8 under `--fail-on warn`.
  - **Test that would have caught it:** a table test over synthetic reports, plus binary rows (C-03).
  - **Effort:** S (1 h for the minimum; 1.5 h including doctor and status).

- [ ] **F-03 "Unknown" is not "uninstalled": explicit collection findings** (P1-20) [R]. **Cut from Stage 1 (§0); after the memo.** checkup reuses fewer doctor paths than doctor itself, so the Stage 1 false-positive risk is covered by the W2 fixture acceptance ("a clean fixture reports none").
  - **Files:** `internal/ops/doctor.go:71-75,90-92,105-112,175`; `internal/ux/stage/stage.go:46-52` (returns `Uninstalled` whenever the install probe fails for any reason); `internal/cli/doctor.go:168-190`.
  - **Fix:**
    - Probe results become tri-state (ok / failed / unknown).
    - When SSH is unreachable (exit 255), the local `ssh` binary is missing, or GitHub auth failed, emit a single `facts_unavailable` finding with the cause and suppress the dependent checks.
    - Never recommend re-registering when facts are unknown.
    - Stage becomes `unreachable`, not `uninstalled`.
  - **Acceptance:** doctor with no GitHub credential gives one finding and no "offline" or re-register advice. With SSH down, the stage is `unreachable`.
  - **Test that would have caught it:** doctor tests with failing fakes (auth error, executor exit 255, `LookPath("ssh")` failure).
  - **Effort:** S–M (3 h).

- [ ] **F-04 Runner-version truth on the host, mirroring W1's rules** (P2-9, P2-8; promoted because W2 needs it; gaps.md runner-version-enforcement)
  - **Files:**
    - `internal/github/runners.go:9-16`: `Runner` drops `version`. The REST `runner` schema has an optional `version` property, per GitHub's OpenAPI description (github/rest-api-description); V-2 tests whether it is actually populated.
    - `internal/ops/doctor.go:144-153`: compares the saved pin with the bundled pin using `!=` and "older than" wording.
    - `internal/cli/upgrade_runner.go:150`.
    - A new package, `internal/runnerversion`.
  - **Fix:**
    1. Add `Version string` with a json tag to `Runner`.
    2. Read the installed version **without executing runner-owned files**:
       - first `readlink <install>/bin`, which points to `bin.<ver>` after a self-update;
       - otherwise the newest `_diag/Runner_*.log` "version" line.
    3. Get the release list from `GET /repos/actions/runner/releases`, cached for 24 h like the update notifier.
    4. Findings, with the thresholds **defined in W1's spec** (`docs/spec/runner-version-rules.md` in the Watch repo, E1.1) and mirrored here:
       - `runner_update_window`: WARN when installed < latest **and** the release immediately after the installed one is more than 10 days old (configurable, 10–14); ERROR beyond 25 days.
       - `runner_below_registration_floor`: ERROR below 2.329.0. The floor is config, not code.
       - `runner_deprecated_messages`: a journal grep for "is deprecated and cannot receive messages".
       - `runner_runsvc_empty`: a 0-byte `bin.*/runsvc.sh` (actions/runner #4421).
  - **Acceptance:** with a fixed clock (2026-09-26), a fixture with `bin → bin.2.334.0`, where 2.335.0 was published on 2026-06-08 and latest is 2.337.0, produces ERROR. The same fixture pointing at `bin.2.337.0` produces no finding.
  - **Test:** pure-function window tests (fixed clock), and parser tests for the readlink and `_diag` formats.
  - **Effort:** S (2 h; the spec document moved to W1).

- [ ] **F-05 Strict host keys for `checkup`** (SEC-4) [R]
  - **Files:**
    - `internal/remote/system.go:206-225`: every session uses `StrictHostKeyChecking=no` and `UserKnownHostsFile=/dev/null`, and always passes `-p`.
    - `system.go:114`: the fingerprint is `SHA256` of the whole keyscan line.
  - **Fix (Stage 1 minimum, 1.5 h):**
    - Add a new `Target.HostKeyPolicy`.
    - For `checkup`, run the system `ssh` with the user's own `~/.ssh/config` and `known_hosts`, and `StrictHostKeyChecking=yes`.
    - No keyscan and no TOFU. Pass `-p` only when the flag is given, and add `IdentitiesOnly=yes` when `-i` is given.
    - OpenSSH then does the verification, so checkup never displays a fingerprint of its own.
  - **Later (R-06):**
    - the same policy for `status`, `doctor`, `logs` and every mutating command;
    - `ssh -G` alias resolution for RunnerKit's own pin;
    - `x/crypto/ssh.FingerprintSHA256` over the key blob, so fingerprints match `ssh-keygen -lf`.
  - **Acceptance:** an unknown host key fails with "connect once with ssh to verify, or ssh-keygen -F". A changed key fails closed. Host aliases and ProxyJump from `~/.ssh/config` work, because OpenSSH resolves them.
  - **Test:** `sshArgs` unit tests; the W2 fixture container with known_hosts cases (match, mismatch, missing).
  - **Effort:** S (1.5 h for the minimum; 3 h including the fingerprint work).

- [ ] **F-06 (cut from Stage 1; after the memo) State file lock and signal handling** (P1-25) [R]
  - **Files:** `internal/state/store.go:114-172` (Save / SaveRepository / UpdateRepository with no lock); `cmd/runnerkit/main.go`; about 11 `context.Background()` calls in `internal/cli`.
  - **Fix:**
    - Hold a `flock` on `state.json.lock` across load–modify–save.
    - In `main`, `signal.NotifyContext(SIGINT, SIGTERM)` and pass the context down. On cancel, print "interrupted during <step>; run runnerkit status / destroy --dry-run".
  - **Acceptance and test:** 20 goroutines running `SaveRepository` on distinct repos persist 20 entries. The dossier measured 1 surviving.
  - **Effort:** S (2 h).

- [ ] **F-07 (cut from Stage 1; after the memo) `state forget` and a partial-cleanup exit code** (P1-13) [R]
  - **Files:**
    - `internal/cli/state.go` (only `show`);
    - `internal/cli/destroy.go:92-99,199-205,253-267`;
    - `internal/cli/down.go:343-352,376-382,441-447`;
    - `docs/troubleshooting/cleanup.md:111-121`, which recommends `rm -rf /opt/actions-runner/runnerkit-* /var/lib/runnerkit`. On a multi-repo host that wipes every repo.
  - **Fix:**
    - `runnerkit state forget --repo X`. It prints what it drops. For cloud entries it lists the resource IDs and the Hetzner label filter `runnerkit=true`. It requires a typed confirmation or `--yes`.
    - Partial `down`/`destroy` returns `ExitPartial` (10).
    - Change the doc to per-runner paths.
  - **Test:** `forget` removes only the target entry; a partial destroy returns 10.
  - **Effort:** S (2 h).

- [ ] **F-08 (cut from Stage 1; after the memo) Busy gate for `recover` restart** (P1-23) [C]
  - **Files:** `internal/cli/recover.go:151-160`.
  - **Fix:** query GitHub `busy`; refuse unless `--force`.
  - **Test:** a fake GitHub with `busy=true` produces a refusal and zero executor calls.
  - **Effort:** S (1 h).

### A.5 Stage 2 (only if Stage 1 PASSES): item 0, the privilege model (E2.0), then the demand-gated BYO `up` rebuild (E2.4, v1.4.0)

STRATEGY §7 orders this work:
- **Item 0 (R-01, R-05, with C-07(b) and the full C-08)** is required before any root-installed component: the root heartbeat variant, hygiene and `adopt`. It does not depend on setup demand. It closes SEC-1 to SEC-3 and retires the sudoers list that v1.3.4 only disclosed.
- **Item 4 (every other R-item)** is built only with **≥5 concrete setup requests** (§10 metric).

v1.3.4 already made four smaller changes: the install.sh sync (A-14), the step reorder (A-18), the error wrapping (A-19) and the pin bump (A-16). The items below say what remains.

Every item is gated by C-07(b) and C-08 on PRs, and by C-09 (a real job) before any tag that says BYO works. All items share the item-0 privilege model:
- a one-time root `install.sh`, plus one root-owned helper with fixed verbs;
- root-owned systemd units rendered by RunnerKit, never `sudo ./svc.sh`;
- **root never executes files the runner user can write**;
- the install dir stays runner-writable, so the runner's self-update keeps satisfying the 30-day rule. Per gaps.md the self-update writes into the install dir, so a root-owned one would break it (inferred from the update mechanics, not live-tested; N-14).

- [ ] **R-01 (item 0) Replace the allowlist with a root-owned fixed-verb helper** (SEC-1; retires the list that A-14 copied into install.sh for P0-1)
  - **Files:**
    - `install.sh:30-46`: after A-14, this block is generated from `RenderSudoersEntry` (`internal/bootstrap/sudoers.go:70-91`).
    - `install_sh_test.go`: after A-14, a full-body equality test.
    - The other production caller of `RenderSudoersEntry` is cloud-init (`internal/provider/hetzner/provision.go:360`).
    - All rendered scripts in `internal/bootstrap/*.go`.
  - **Fix:**
    - `install.sh`, run once as root, installs `/usr/local/libexec/runnerkit-hostd`. It is root-owned, 0755 and versioned, and exposes these verbs:
      - `prepare`
      - `install-deps <profile>`
      - `runner create|configure|unit|remove <slug>`
      - `hygiene …` (E2.2)
    - It also writes a single sudoers line for that one path. The helper validates every argument itself, resolves paths from its own root-owned config, and never takes a path argument.
    - Bootstrap scripts call only `sudo -n runnerkit-hostd <verb>`.
    - The old fragment, and the CI fragment, are removed on upgrade. The v1.3.4 revocation guide (H-09) covers hosts that never upgrade.
  - **Acceptance:** on a C-08 Ubuntu 22.04 or 24.04 password-sudo container prepared **only** by the candidate `install.sh`:
    - `sudo -n -l` for the SSH user shows exactly one NOPASSWD path;
    - `up` completes;
    - C-07(b) passes;
    - C-09 is green.
  - **Test that would have caught P0-1:** C-07 (every `sudo` in the rendered scripts ⊆ the granted paths) and C-08 (a password-sudo host).
  - **Effort:** L.

- [ ] **R-02 Docker only as a disclosed opt-in; verify as the runner user** (SEC-5; P0-2's order fix shipped in A-18)
  - **Files:**
    - `image_setup.go:101`: `usermod -aG docker … || true`;
    - `:69-72`: rustup as the service user;
    - `:30-35`/`:166-167`: the marker;
    - `image_setup.go:8`: `ImageSetupVersion`, which A-18 set to `"2"`.
  - **Fix:**
    - `--docker` (default off on BYO) adds the group, with plan text saying "docker group membership is root-equivalent", then restarts the service.
    - `verify_service` checks, as the runner user: `id -nG` contains docker when requested, `docker info` succeeds, and `cargo -V` succeeds when Rust is in the profile.
    - Bump `ImageSetupVersion` to `"3"` if the script changes.
    - Hosts that got the group silently from v1.3.4 keep it until the user runs a reconcile without `--docker`, which removes it after a confirmation.
  - **Acceptance:** the C-09 canary job (gcc, docker build, `services: postgres`, `container:`) passes with `--docker`. Without `--docker`, docker jobs fail with a RunnerKit-authored hint.
  - **Test:** C-08 asserts `id -nG runnerkit-runner` both with and without `--docker`.
  - **Effort:** M.

- [ ] **R-03 A real `upgrade-runner`, as "reconcile", with a busy gate** (P0-3, P1-23, WC-M1)
  - **Files:** `internal/cli/upgrade_runner.go:61-155`; `script.go:52-56`; `install.go:241`.
  - **Fix:** default behaviour is `reconcile`: re-apply the host template (packages, helper version, units) **without** re-running `config.sh`. The runner's own self-update handles binaries (R-04).

    An explicit `--replace-binaries` does the following:
    1. refuse while busy unless `--force`;
    2. stop the service;
    3. extract into `bin.<ver>`/`externals.<ver>` and swap the symlinks, as upstream `update.sh` does;
    4. keep `.runner`, `.credentials` and `_work`;
    5. restart;
    6. read the version back (F-04) and record the *observed* version.
  - **Acceptance** (C-08 with the stub runner, v1→v2):
    - the stub `Runner.Listener --version` prints v2;
    - `.runner` and `.credentials` are byte-identical;
    - the stub `config.sh` was never invoked;
    - the fake GitHub saw zero registration-token requests;
    - a busy runner exits 4.
  - **Test that would have caught P0-3:** the C-08 stub `config.sh` rejects an empty `--token`.
  - **Effort:** M.

- [ ] **R-04 Resolve-latest runner, with an auto-bumped fallback pin** (gaps.md runner-version-enforcement; the 2.337.0 bump and the `--disableupdate` test shipped in A-16)
  - **Files:** `internal/bootstrap/package.go:5,23-34`.
  - **Fix:**
    - At `up` time, fetch the latest actions/runner release and verify the SHA-256 between the `<!-- BEGIN SHA linux-x64 -->` markers in the release body.
    - Keep the bundled pin as an offline fallback, which must be at least the 2.329.0 floor.
    - Add a scheduled workflow that opens a pin-bump PR.
  - **Test:** release-body marker parser tests; a fake releases API.
  - **Effort:** S–M.

- [ ] **R-05 (item 0) RunnerKit-rendered root-owned units; never `sudo ./svc.sh`; root-owned `/var/lib/runnerkit`; fix recover** (SEC-2, SEC-3, P1-10)
  - **Files:**
    - `internal/bootstrap/script.go:44-45,53` (install dir and `/var/lib/runnerkit` chowned to the runner user);
    - `script.go:68-74` (`sudo ./svc.sh stop/uninstall/install/start/status`);
    - `recover.go:170,201-233`;
    - `down.go:442-446`;
    - the `sudoers.go:90` svc.sh glob.
  - **Fix:**
    - The helper writes `/etc/systemd/system/actions.runner.<slug>.service` (root-owned, 0644) with `User=<runner>` and `ExecStart=<install>/runsvc.sh`, which runs as the runner and never as root.
    - `/var/lib/runnerkit` is root-owned 0755; the runner owns only its `work/<slug>`.
    - `recover --reinstall-service`/`--reregister` reuse the same unit verbs (stop, uninstall, install, start) and re-resolve the unit afterwards.
  - **Acceptance:** C-08 asserts:
    - `find /etc/systemd/system/actions.runner.* /usr/local/libexec/runnerkit* /var/lib/runnerkit -maxdepth 0 ! -user root` is empty;
    - acting as the runner user, writing to every `Exec*=` target that runs as root fails with EACCES;
    - `recover --reregister` ends with the unit active.
  - **Effort:** M.

- [ ] **R-06 Enforced host-key pin for every command** (SEC-4)
  - **Fix:**
    - Extend F-05's policy to `status`, `doctor`, `logs`, `up`, `down`, `destroy`, `recover` and `upgrade-runner`. `upgrade-runner` and `logs` skip the check today.
    - Resolve aliases through `ssh -G`. Compute fingerprints with `x/crypto/ssh.FingerprintSHA256` over the key blob (`system.go:114` hashes the whole keyscan line), so they match `ssh-keygen -lf`.
    - Use a RunnerKit-managed `known_hosts` with `HostKeyAlias` and `StrictHostKeyChecking=yes`.
    - Add a `--host-key-fingerprint SHA256:…` flag.
    - Separate `--accept-host-key` from `--yes`, which today auto-accepts new keys (`up.go:1482-1485`).
  - **Test:** a C-08 key-rotation scenario fails closed on every command.
  - **Effort:** S–M.

- [ ] **R-07 Registration and removal tokens out of argv and sudo logs** (SEC-6; KEYFACTS security)
  - **Files:** `script.go:56,109` (install scripts), the removal script at `:274` and the reconfigure script at `:287`. In each, the token is expanded into `sudo su … -c "… --token $TOKEN"`.
  - **Fix:**
    - The laptop sends the token over SSH stdin to `runnerkit-hostd runner configure`.
    - The helper passes it to `config.sh` through the environment. Upstream reads `ACTIONS_RUNNER_INPUT_TOKEN` **[inference: verify in actions/runner `src/Runner.Listener/CommandSettings.cs` before building]**. Otherwise use a 0600 root-owned file that is deleted after use.
  - **Test:** C-08 greps `journalctl _COMM=sudo` and `/var/log/auth.log` for the fake token value and must find nothing.
  - **Effort:** S.

- [ ] **R-08 Per-repo Unix users** (SEC-7)
  - **Files:** `bootstrap.DefaultServiceUser` usage at `up.go:1608`, `recover.go:212,228`, `upgrade_runner.go:116`, `down.go:291` and `destroy.go:274`.
  - **Fix:** create a user `rk-<hash8(repo)>` per repo, with 0700 install and work dirs. Migrate existing hosts on reconcile.
  - **Test:** C-08: repo A's user cannot read repo B's `.credentials_rsaparams`.
  - **Effort:** M.

- [ ] **R-09 `--ci-sudo none|packages|full`** (P1-11)
  - **Files:** `install.sh:65-93` (opt-in apt sudo only); `internal/bootstrap/ci_sudoers.go:24-35`.
  - **Fix:**
    - The profile is explicit in the plan.
    - `full` is documented as root-equivalent and is the only way to get hosted-runner parity for `playwright --with-deps`.
    - The default is `none` on BYO.
  - **Test:** a C-09 job step `sudo -n apt-get install -y jq` passes only with `packages` or `full`.
  - **Effort:** S–M.

- [ ] **R-10 Minimal image profile by default; the plan shows what will be installed; disk check per profile** (P1-14)
  - **Files:**
    - `install.go:100` (`Plan` ignores its options);
    - `internal/workflow/plan.go:79-92` (a static 6-step plan that omits `setup_runner_image`);
    - `up.go:1617-1636` (`confirmBootstrapPlan` shows nothing);
    - `internal/preflight/checks.go:48,219-222` (2 GiB, against 3.16 GiB measured and 4.5–5 GB projected per gaps.md).
  - **Fix:**
    - `--image none|minimal|full`. `minimal` is baseline apt plus git/jq/zstd. `full` is today's parity script, pinned: Node LTS, not EOL Node 20.
    - Derive the plan from the executed step list, including packages and third-party apt sources.
    - Disk check: minimal ≥4 GiB, full ≥8 GiB.
  - **Test:** a plan-equals-executed-steps property test.
  - **Effort:** M.

- [x] **R-11 Failures name the failing step** (P1-15): moved to v1.3.4 as A-19. It is listed here only so old references resolve.

- [ ] **R-12 Per-section image markers; network lookups are non-fatal per section** (P1-16) [R]
  - **Files:** `image_setup.go:30-35` (a single marker gate) and `:166-167` (written unconditionally); `:44,96,106` (`gpg … || true` while `.list` files are still written); `:59,139` (the Go and geckodriver lookups abort under `set -e`, rc=1 reproduced).
  - **Fix:**
    - Each tool section records its own marker only after `command -v` verifies it.
    - When a key fetch fails, remove that section's `.list` and continue.
    - Doctor reports per-tool status.
  - **Test:** a C-08 run with blocked egress to go.dev leaves the other sections installed and a re-run heals.
  - **Effort:** M.

- [ ] **R-13 Tarball cache verified before use; no duplicate download** (P1-17) [R]
  - **Files:** `install.go:230-249` (verified only inside `if [ ! -f ]`; curl writes to the final path); `script.go:48-52` (a second 225 MB download into each install dir).
  - **Fix:** download to a temp file, verify, then `mv`; verify before every extract; delete the second download.
  - **Test:** C-08 with a truncated cache file: the next run re-downloads and succeeds.
  - **Effort:** S.

- [ ] **R-14 `register` works after install.sh** (P1-18) [C]
  - **Files:** `up.go:1586-1597` (requires the user); `internal/cli/installhint.go:60-83` (tells the user to re-run install.sh, which loops).
  - **Fix:** `install.sh` (the helper's `prepare` verb) creates the runner user(s) and the `/var/lib/runnerkit` layout.
  - **Test:** in C-08, `register` on a fresh install.sh host succeeds.
  - **Effort:** S.

- [x] **R-15 The state-replace gate comes before any remote mutation** (P1-21) [C]
  - **Files:** `up.go:235-296` (token, Apply and online wait) happen before `saveRepositoryState`/`confirmStateReplace` (`:1707-1717`).
  - **Fix:** check at `up.go:159`. A same-host rerun is an idempotent update.
  - **Test:** a recording executor sees zero mutations before the gate.
  - **Effort:** S.

- [ ] **R-16 apt robustness** (P1-24) [C]
  - **Files:** `script.go:15-19`; `image_setup.go:38` (the export is lost to sudo's `env_reset`); `install.go:113-127` (no `Timeout`); `internal/remote/system.go` (buffered output).
  - **Fix:**
    - `-o DPkg::Lock::Timeout=600`.
    - Set `DEBIAN_FRONTEND=noninteractive` inside the helper.
    - Scripts' stdin from `/dev/null` for apt.
    - Per-step timeouts.
    - Stream the step log.
  - **Test:** a C-08 run with `unattended-upgrades` holding the lock waits and succeeds.
  - **Effort:** S.

- [ ] **R-17 Doctor checks privilege coverage by content, not existence** (P1-20)
  - **Files:** `internal/cli/doctor.go:181` (`test -f`); `SudoersIsPrepared` at `internal/bootstrap/sudoers.go:157-168` has no non-test caller.
  - **Fix:** doctor reuses A-13's probe, or the helper's `version` verb after R-01.
  - **Test:** a stale fragment yields a WARN.
  - **Effort:** S.

- [ ] **R-18 Trust-gate recheck, auth precedence and one redactor** (SEC-8, SEC-9, SEC-10; STRATEGY 0.2 assigns them to Stage 2)
  - **Files:**
    - SEC-8: `internal/cli/status.go` and `doctor.go` (visibility is not rechecked); `recover.go` and `upgrade_runner.go` (re-registration without the gate).
    - SEC-9: `internal/github/auth.go:42-58` (an ambient `gh auth token` wins over `RUNNERKIT_GITHUB_TOKEN`). The `workflow`-scope advice in `docs/troubleshooting/auth.md:148` and `github.md:137` is removed earlier, in v1.3.4 (H-09).
    - SEC-10: at least 5 `redact.New()` instances; 0644 debug logs; a runner-token regex that matches only the fixture shape.
  - **Fix:**
    - Recheck repo visibility in `status` and `doctor`, and gate re-registration on it.
    - Env first, then `gh auth token --hostname github.com`.
    - One process-wide redactor, 0600 logs, and patterns for classic PATs, `Bearer` headers and URL credentials.
  - **Test:**
    - A public-repo fixture flips `status` to a WARN.
    - An auth-precedence table test.
    - Redaction golden tests.
  - **Effort:** S each (assessment §6.6).

### A.6 Frozen: fix only if a Stage 3 trigger fires (E3.4 cloud revival, or ≥5 explicit requests)

Until then each item stays documented as a known issue in the README (H-08). The fix is specified here so it can be picked up cold.

- [ ] **Z-01 cloud-init readiness fails fast** (P1-4) [R]
  - **Files:** `internal/cli/up.go:994-1011` (`cloud-init status --wait; rc=$?` under `set -euo pipefail`); `:1034-1049` (retries any error); `:1056` (15 min).
  - **Fix:** `rc=0; cloud-init status --wait || rc=$?`. Retry only on ssh transport exit 255. Fail fast on 1/2 with `status --long`. Add `--destroy-on-failure`.
  - **Test:** execute the script in bash with a stub `cloud-init` that exits 2.
  - **Effort:** S.
- [ ] **Z-02 Hetzner SSH key reuse** (P1-8) [I]
  - **Files:** `internal/provider/hetzner/provision.go:126` (always `CreateSSHKey`).
  - **Fix:** `SSHKey.GetByFingerprint` and reuse it, marked not-owned; or generate a key per runner.
  - **Test:** a fake client that returns `uniqueness_error`.
  - **Effort:** S–M.
- [ ] **Z-03 Honest ephemeral mode** (P1-12) [R/C]
  - **Files:** `script.go:227` (`OnActiveSec=24h`); `up.go:1913` (`TTL: "24h"`); `script.go:182-187` (`User=` with `ExecStopPost` and no `+`); the finalizer writes into root-owned 0755/0750 dirs (`:127-129`, `:147`).
  - **Fix:** render the TTL; use `ExecStopPost=+…`; disarm the timer on completion; pull the log archive before destroy; auto-destroy the cloud VM from the laptop, or refuse. The longer-term option is JIT (E3.3).
  - **Test:** execute the rendered unit in a C-08 container.
  - **Effort:** M.
- [ ] **Z-04 hcloud-go v2, availability and defaults** (the rest of P0-6 after A-07; §A.0 row 43)
  - **Files:** `go.mod` (hcloud-go v1.59.2); `internal/provider/hetzner/client.go:53-65` (deprecated `Image.GetByName`, `:63`); `provision.go:224,466,519-522` (Datacenter reads, nil-guarded).
  - **Fix:** migrate to v2; check `Locations[].Available` before offering a location; use a stock-out fallback order; typed `hcloud.IsError` mapping. The live price shipped in v1.3.4 (A-07, on v1). Revisit the default type only with live data.
  - **Test:** recorded API fixtures.
  - **Effort:** M.
- [ ] **Z-05 arm64** (P1-6) [C]
  - **Files:** `image_setup.go:60,97,107,119,141,153`.
  - **Fix:** a `dpkg --print-architecture`-driven script; `GetByNameAndArchitecture`; an arm64 CI leg on `ubuntu-24.04-arm`.
  - **Test:** C-08 on arm64.
  - **Effort:** M.
- [ ] **Z-06 Non-Ubuntu distros** (P1-7) [C]
  - **Files:** `install.go:313-329` (the Ubuntu baseline is always appended); `script.go:20-26`; `ID_LIKE` ignored.
  - **Fix:** per-family package maps, or keep the A-12 refusal.
  - **Test:** a C-08 debian:12 and fedora leg.
  - **Effort:** M.
- [ ] **Z-07 A real workflow package parser** (P1-5)
  - **Files:** `internal/cli/workflow_packages.go:14-17,75-86`.
  - **Fix:** YAML-aware parsing of `run:` blocks in jobs that target RunnerKit labels; a shell-aware tokenizer; per-package non-fatal installs.
  - **Test:** dossier fixtures (comment words, `-t bookworm-backports`, `apt-get -y install x`).
  - **Effort:** M.
- [ ] **Z-08 Cloud job-user sudo, swap and presets** (P1-11 cloud part; P2-13)
  - **Files:** `provision.go:379-404`; `profile.go:11`.
  - **Fix:** `--ci-sudo full` default on single-tenant VMs; a swapfile; size presets.
  - **Test:** a C-09 cloud leg.
  - **Effort:** S–M.
- [ ] **Z-09 `destroy` converges once the provider confirms deletion** (P1-13 cloud part)
  - **Files:** `destroy.go:199-205,253-267`.
  - **Fix:** provider-verified deletion satisfies remote cleanup.
  - **Test:** a fake with SSH unreachable and provider gone removes the state.
  - **Effort:** S.
- [ ] **Z-10 Cloud network defaults**
  - **Files:** `internal/provider/profile.go:14` (`0.0.0.0/0`); `provision.go:158` (owner key on root); `:385` (admin `NOPASSWD:ALL`).
  - **Fix:** default to the caller's IP/32; disable root login; inject host keys through cloud-init (no TOFU).
  - **Test:** plan golden files.
  - **Effort:** S.

### A.7 v1.3.4 release checklist (pre-tag; tag by C2, 2026-10-10)

- [ ] H-07 done: the tap token was rotated, and the fine-grained PAT is verified before this release runs. Check OIDC/cosign keyless signing too, which has been idle for 130 days, per the CLAUDE.md sequence (STRATEGY §5).
- [ ] CI green on `main`, including A-10 (C-01's test gate) and every §A.1/§A.2 test.
- [ ] govulncheck result for the candidate SHA read and linked (A-17a). If a finding is reachable, A-17b is in this release.
- [ ] Manual check with a scratch `RUNNERKIT_STATE_DIR` and seeded state, run from a throwaway directory (no Hetzner spend):
  - [ ] A-04: `down` on cloud state is refused;
  - [ ] A-05: `up --replace` on cloud state is refused;
  - [ ] A-06a: `upgrade-runner` and `doctor --fix` are refused;
  - [ ] A-08: `--mode ephemeral --cloud` is refused, even with `--experimental`;
  - [ ] A-15: `--cloud` without `--experimental`, and `--cloud` without `--cloud-region`, are refused offline;
  - [ ] A-07: a `--dry-run` plan with a real read-only Hetzner token shows the API price and its label. This costs nothing, because no server is created.
- [ ] 0.3b outcome recorded: either A-20 is green (run URL in the CHANGELOG), or A-21 ships and the banner says BYO is not supported in this release.
- [ ] The CHANGELOG `v1.3.4` section:
  - lists every shipped A-item;
  - includes the "Known issues" block that mirrors the README banner (H-08), including the v1.3.5 items (P1-1..P1-3, A-06b) (overtaken 2026-09-27: P1-1..P1-3 and A-06b are fixed in v1.3.4 and are not known issues);
  - includes the erratum for the v1.3.3 notes (H-10);
  - says BYO works on password-sudo hosts **only** if A-20 is green.
- [ ] README banner (H-08), security page with its revocation section (H-09), and LICENSE (H-01) merged before the tag.
- [ ] Tag `v1.3.4` (annotated) from upstream. Confirm that the release workflow ran tests (A-10), the assets are there, cosign signed the checksums, and the cask bump landed (per CLAUDE.md).
- [ ] If the tag has not shipped by **2026-10-10 (C2)**, do H-13 (deprecate the cask) instead.

---

## B. Repo, licensing and hygiene

| ID | Task | Why (evidence) | Done when | Due | Est. |
|---|---|---|---|---|---|
| H-01 | Add `LICENSE` (Apache-2.0) | No LICENSE, so the project is legally not open source. homebrew-core requires one. Apache-2.0 gives a patent grant and contrasts with Zoomies' AGPL (KEYFACTS C, D) | `LICENSE` at the root; GitHub detects "Apache-2.0"; the README badge links it | C2 | 0.25 h |
| H-02 | `CONTRIBUTING.md` with a DCO (no CLA) and a PR check for `Signed-off-by` | STRATEGY 0.1 | Includes the scope rule "no feature without 2 distinct external requests" and its one exemption (W1–W3, capped at 30 h), and links §E. The DCO check becomes required on `main` with H-12 | C2 | 0.5 h |
| H-03 | `SECURITY.md`, and turn on private vulnerability reporting | STRATEGY 0.1; kill criterion 9 | 14-day response target. Supported versions = the latest minor. It links the security-posture page and says that issues listed there, each with its fix stage, are **known and disclosed**. Kill criterion 9 applies only to newly reported issues that cannot be fixed within 14 days | C2 | 0.5 h |
| H-04 | `CHANGELOG.md` (Keep a Changelog). **H-04a:** fold the 8 `RELEASE-NOTES-*.md` files into it and delete them. **H-04b** (Stage 1, §0 cut 5): reconstruct v1.3.0–v1.3.2 from the tags | 8 scattered notes; v1.3.0–v1.3.2 missing (assessment §8.5) | One file; the root no longer has `RELEASE-NOTES-*` | H-04a C2; H-04b C8 | 0.25 h + 0.25 h |
| H-05 | Remove `.planning/` (147 files, 28 naming the maintainer's hosts), `smoke-output.log` and `GEMINI.md` from the tree. Archive `.planning` to a **private** repo | STRATEGY §4 "Delete now" | `git ls-files .planning smoke-output.log GEMINI.md` is empty, and `.gitignore` covers them | C2 | 0.5 h |
| H-06 | History check before any outreach. Run `gitleaks detect --log-opts=--all`, and grep all history for the maintainer's host strings, IPs and `ghp_` / `github_pat_` / `hcloud` tokens | STRATEGY §4 "Git history, before announcing" | A written decision. **Default: do not rewrite history** (18 tags and the cosign-signed release checksums reference the SHAs). If hostnames or IPs appear, rotate keys or decommission those hosts. Confirm which hosts still exist: the dossier says the dat0 runner was destroyed in the 2026-05-18 smoke test, but a RunnerKit host running dat0 CI hit df=0 on 2026-05-26. Rewrite only if actual secrets appear; then publish a notice and re-tag | **Before C5** (the first thread replies); §0 cut 4 | 0.5 h |
| H-07 | Rotate `HOMEBREW_TAP_GITHUB_TOKEN` and replace it with a fine-grained PAT (contents:write on `homebrew-tap` only, with an expiry); verify the tap push and OIDC signing | CONCERNS.md records that it was pasted into chat; rotation status is unknown (assessment §6.4). Release machinery idle for 130 days (STRATEGY §5) | New secret in place **before** the v1.3.4 tag | C2 | 0.25 h |
| H-08 | README honesty rewrite. Details below the table | STRATEGY 0.2; P0-1, P0-2 | Reviewed by reading it as a stranger would: no claim in it is contradicted by §A.0, and the BYO line matches what §A.2 actually shipped | C2 | 1.5 h |
| H-09 | `docs/security-posture.md` with every SEC item, its fix stage and a revocation section; retract `docs/safety.md:9-14`; fix the drifted docs. Details below the table | STRATEGY 0.2 (ruling R3); assessment §6, §8.5 | Every SEC-1..SEC-13 item appears with a stage; the revocation steps have been tried on the A-20 container; each drift row is resolved | C2 | 1.5 h |
| H-10 | v1.3.3 erratum: edit the GitHub Release body for v1.3.3 to put a correction at the top; if the body cannot be edited, put the erratum in the v1.3.4 notes and the README | `RELEASE-NOTES-v1.3.3.md:7,12,26` tell users to run `byo-prepare`, which was deleted in v1.0.8 | Erratum visible on the v1.3.3 release page | C2 | 0.25 h |
| H-11 | Name check (≤1 h): srz-zumix/gh-runner-kit (a gh extension for self-hosted runner management with a similar name; 0★, active Sep 2026 per MARKET-RESEARCH §7.2), EUIPO/USPTO. **Buy no domain** | STRATEGY 0.1; KEYFACTS D (.com/.dev/.app taken) | Decision recorded in the CHANGELOG or a Discussion | Before C8 (§0 cut 4) | 1 h |
| H-12 | Repo settings. Details below the table | Needed for W2 report collection and E1.6 metrics | Settings applied; templates merged | C7 | 0.5 h |
| H-13 | Homebrew cask. In `homebrew-tap/Casks/runnerkit.rb`, update `desc` (A-09). **Fallback:** if v1.3.4 misses C2, add `deprecate! date: "2026-10-10", because: "has known defects; see https://github.com/accidentally-awesome-labs/runnerkit#known-issues"` | STRATEGY 0.3 | Cask updated, or deprecated by C2 | C2 | 0.25 h |
| H-14 | Update the stale claims in `CLAUDE.md` and add the strategy rules. Details below the table | Agent notes drive AI-assisted work, and stale notes sent the v1.3.3 fix to the wrong allowlist (assessment §8.5) | Every claim in CLAUDE.md matches the code | C7 | 0.5 h |
| H-15 | Remove the user-facing `byo-prepare` references. 48 files mention it, most in `.planning`. Outside it (15 files): `CLAUDE.md`, the v1.3.3 notes, `scripts/smoke/byo-permission.sh`, `smoke-output.log` (removed by H-05), and comments, copy or tests in `internal/{bootstrap,preflight,cli,ops,provider}` | engineering-quality-14 | `grep -rn byo-prepare --exclude-dir=.git` finds only CHANGELOG history and the A-01 tombstone | With v1.3.5 | 0.5 h |
| H-16 | Weekly public table in a pinned Discussion (the G-1 log). Columns: date; hours (cumulative against the C4/C8/C9/C10 caps); opt-in Watch users (Discussion +1s and `report_usage` issues, reported as **lower bounds**); public-repo dependents (a separate column); distinct external authors; checkup reports; real problems caught (Watch catches plus user-confirmed actionable checkup findings); Watch billed minutes | STRATEGY §6 launch and exit gate; §10 | First row posted by C1; updated every Monday | C1, weekly | 0.25 h/wk |
| H-17 | **Archive playbook**, written in advance and used only on a kill or capacity-fail decision. Details below the table | STRATEGY §5 capacity check; §10 kill criteria 1, 2 and 4 | The playbook exists as a checklist in `docs/maintainers/archive.md` | Before C10 (at C1 if G-0 fails) | 0.5 h |

**H-08 README content.** The Known-issues banner must say which 0.3b outcome shipped (§A.2):
1. **BYO.**
   - If A-20 passed: "Fixed in v1.3.4 for fresh Ubuntu 24.04 x86_64 hosts prepared by the v1.3.4 install.sh. Hosts prepared by an older install.sh must re-run it."
   - If A-21 shipped: "BYO is not supported in this release."
   - Either way: "The install.sh sudoers fragment is root-equivalent; see the security page."
2. **Docker.**
   - If A-18 shipped: the runner user is in the docker group, **which is root-equivalent** (SEC-5). Existing hosts get it on their next `up`.
   - If A-18 did not ship: `services:`, `container:` and Docker jobs fail on fresh hosts. The manual fix is `sudo usermod -aG docker runnerkit-runner && sudo systemctl restart 'actions.runner.*'`, with the same root-equivalence warning.
3. **Job sudo.** Job `sudo apt-get` needs `RUNNERKIT_GRANT_CI_SUDO=1` at install time (P1-11). That grant is root-equivalent (SEC-12).
4. **Cloud and ephemeral.**
   - Cloud requires `--experimental` and an explicit `--cloud-region`, and shows the price the Hetzner API reports.
   - Ephemeral cloud is disabled.
   - Ephemeral BYO requires `--experimental`, is broken on install.sh hosts (P0-1, P1-12), and is not isolation.
5. **Disabled commands.** `upgrade-runner` and `doctor --fix` are disabled. `recover --reinstall-service` and `recover --reregister` are known broken until v1.3.5 (A-06b).
6. **CLI defects until v1.3.5 (§0 cut 1):**
   - unknown commands, bad flags and `--version` print nothing (P1-1);
   - typed confirmations fail in a terminal (P1-2). `--yes` works around it, but it also auto-accepts a new host key;
   - `doctor --ignore` and `up --host` write `config.json` or `sessions/` into the current directory (P1-3).
7. **Platforms.** Only Ubuntu x86_64 is supported (P1-6, P1-7).

The rest of the README:
- Remove the "10 minutes" and "cheaper than GitHub" claims, the `D-01` jargon (`README.md:7`) and the `TAG=v1.0.0` snippets (`:42`, `:59`, where v1.0.0 has no Release). Move the maintainer section (`:83-85`) to `docs/maintainers.md`.
- Disclose the BYO install footprint: Docker CE, Chrome, Firefox, JDK, .NET, Node 20 (EOL), Go and 5+ third-party apt sources; about 4.5–5 GB of disk and about 1.4 GB of downloads (gaps.md, projected).
- Add a **"When NOT to use RunnerKit"** section:
  - public repos (hosted runners are free, and GitHub says self-hosted runners should "almost never" run public code);
  - untrusted code;
  - saving money at low volume (an inferred 97% or more of individuals fit in the free minutes);
  - ephemeral or agent pools (ARC, actions/scaleset, Zoomies);
  - macOS or Windows;
  - teams wanting managed runners.
- Status line: "Experimental. Maintained on a capped-hours basis until a published go/kill decision on 2026-12-21."

**H-09 security posture page** (STRATEGY 0.2, ruling R3). Every known issue is listed with its fix stage:

| Issue | Plain statement | Fix stage |
|---|---|---|
| SEC-1 | The SSH-user sudoers fragment is root-equivalent (`sudo su -`, `tee`, `cp`, `apt-get` hooks), and has been since v1.0.8. v1.3.4 makes install.sh carry the full list (A-14) without adding capability | Stage 2 item 0 (R-01) |
| SEC-2 | The runner user can rewrite `svc.sh`, which RunnerKit later runs as root through `sudo ./svc.sh` (`script.go:53`, `:68-74`) | Stage 2 item 0 (R-05) |
| SEC-3 | `/var/lib/runnerkit` is runner-owned while root writes into it | Stage 2 item 0 (R-05) |
| SEC-4 | The host-key pin is advisory and its fingerprint is non-standard | Strict `known_hosts` in `checkup` (Stage 1, if W2 is built; F-05); the rest of the CLI in Stage 2 (R-06) |
| SEC-5 | Docker group membership is root-equivalent | Disclosed now; opt-in in Stage 2 (R-02) |
| SEC-6 | Registration tokens appear in argv and sudo logs | Stage 2 (R-07) |
| SEC-7 | All repos on a host share one Unix user | Stage 2 (R-08) |
| SEC-8..SEC-10 | Trust gate not rechecked; auth precedence inverted; split redaction | Stage 2 (R-18). The `workflow`-scope advice is removed now |
| SEC-11, plus cloud-init `NOPASSWD:ALL` (`provision.go:385`) | Cloud SSH is open to 0.0.0.0/0, and the cloud admin has full sudo | Cloud frozen behind `--experimental`; fixed only if cloud is revived (Stage 3, Z-10) |
| SEC-12 | The opt-in CI apt sudo is root-equivalent (`apt-get -o APT::Update::Pre-Invoke`) | Disclosed now; opt-in profiles in Stage 2 (R-09) |
| SEC-13 | The release workflow had no test gate or repo guard, and uses mutable action tags | Test gate and guard in v1.3.4 (A-10); SHA-pinning in Stage 1 (C-01b) |

The page also says plainly:
- ephemeral BYO is not isolation;
- public or untrusted code belongs on GitHub-hosted runners.

**"If you already installed RunnerKit"** goes on the same page, verbatim from STRATEGY 0.2. Test each step on the A-20 container before publishing:
1. `sudo rm /etc/sudoers.d/runnerkit-installer /etc/sudoers.d/runnerkit-runner-ci`, then `sudo visudo -c`.
2. Make `svc.sh` and `bin/` under `/opt/actions-runner/runnerkit-*/` root-owned, or reinstall the service from a root-owned unit.
3. Review docker-group membership (`id -nG runnerkit-runner`; `sudo gpasswd -d runnerkit-runner docker` if unwanted).
4. Delete RunnerKit-created Hetzner resources with `runnerkit destroy` or in the Hetzner console (label `runnerkit=true`).

Docs to fix:

| Doc | Change |
|---|---|
| `docs/safety.md:9-14,84-88` | Retract "stronger isolation per job", and stop routing untrusted work to ephemeral cloud VMs (P1-12, SEC-11) |
| `docs/upgrade.md:47-48` | "Idempotent" → the A-06 text |
| `docs/troubleshooting/cleanup.md:111-121` | Destructive `rm -rf` → per-runner paths |
| `docs/runner-platforms.md:11-12` | Remove the arm64 and macOS claims (P1-6) |
| `docs/cloud-quickstart.md:23-32,70` | Remove Ruby, "added to docker group" and "cleaned up automatically"; add `--experimental` and `--cloud-region` |
| `docs/troubleshooting/auth.md:148`, `github.md:137` | Remove the `workflow` scope advice (SEC-9) |
| `docs/troubleshooting/bootstrap.md` | cloud-init v2 → v3; "scoped only" → the honest wording |

**H-12 repo settings:**
- description and topics (`github-actions`, `self-hosted-runner`, `homelab`, `devops`);
- enable Discussions, with these categories:
  - "Checkup reports" (announcement-style, with a pinned post and a template);
  - a pinned **"I'm running Watch/checkup"** post, where +1s are the opt-in adoption count (STRATEGY §6 exit gate);
  - "Q&A";
  - "Requests (needs 2)";
- issue templates: bug, checkup report, and a feature request with a "who else needs this?" field;
- branch protection on `main` requiring the existing `pr-checks.yml` test job, A-10's release gate, and DCO. Add C-03 when A-01 lands;
- disable the wiki.

**H-14 CLAUDE.md corrections:**
- `sessions/` lives in the state dir only after A-03;
- readiness is not fail-fast (Z-01);
- `ImageSetupVersion` is never persisted in state;
- there are 70 baseline packages, not about 75;
- the sudoers fragment is root-equivalent.

Add the rules:
- no feature without 2 external requests (W1–W3 exempt, capped at 30 h);
- `RenderSudoersEntry` gains no entries, and install.sh is generated from it (A-14);
- never tag a release that claims BYO works without a real job on a fresh password-sudo host (A-20 or C-09);
- never pass `--disableupdate`.

**H-17 archive playbook steps:**
- README notice with the reason and the date;
- publish the E1.3 recipes as gists or a small composite action;
- `disable!` the cask;
- hand Watch to a volunteer or archive it;
- archive the repo;
- a final Discussion post with the data.

Capacity-fail path (G-0): ship the §A.1 items and A-21, publish the banner and the security page, then archive.

---

## C. Test and CI gates

**Why the green suite missed every P0.** Tests compare rendered bash strings against fake executors, and none executes the generated shell. The real adapters (`remote/system.go`, `hetzner/client.go`) have 0% coverage. Test wiring differs from production wiring. No smoke ever dispatches a job. The release workflow runs no tests (assessment §8.1–8.2).

| ID | Gate | Catches (defects) | When (after the re-baseline) | Blocking? | Effort |
|---|---|---|---|---|---|
| C-01 | Release gated on tests plus a repo guard (**= A-10**); **C-01b**: SHA-pinned actions | SEC-13. Releases that failing tests do not block. Tests do run in parallel through `pr-checks.yml` on pushes to `main` (verifier caveat), and the v1.3.1–v1.3.3 "5 s apart" timing is unverified. The suite was green anyway, so C-01 alone would not have caught any P0 | C-01: v1.3.4. C-01b: Stage 1, before C8 (STRATEGY 0.2 puts SHA-pinning in Stage 1) | Yes | S (0.5 h + 0.25 h) |
| C-02 | Test hermeticity guard | Tests writing into the checkout, live network, the real state dir (P1-3 masked) | v1.3.5, with A-03 (§0 cut 1) | Yes | S (0.5 h) |
| C-03 | Binary CLI contract tests | P1-1 silent errors, `--version`, the exit-code contract (F-02), CWD writes (P1-3) | v1.3.5, with A-01 | Yes | S (inside A-01's 1.5 h) |
| C-04 | gofmt + golangci-lint (baseline mode) | 9 unformatted files; 46 staticcheck findings, including the SA4023 never-failing test | After the memo (cut from Stage 1) → Stage 2 CI gate | Yes (new issues) | S (2 h) |
| C-05 | govulncheck | Unmeasured CVE exposure (Go 1.22 out of support, x/net v0.12.0) | v1.3.4 report-only and scheduled (**= A-17a**) → blocking in Stage 2 | Stage 2 | S (0.5 h) |
| C-06 | shellcheck of **rendered** scripts, install.sh, the smoke scripts and Watch | Shell-level bugs in generated bash | Watch's own shellcheck is inside E1.1. The CLI corpus comes after the memo → Stage 2 CI gate (STRATEGY §7 item 4) | Yes | S (1.5 h) |
| C-07 | (a) install.sh/Go template full-body equality (**= A-14**); (b) sudo-allowlist lint over the rendered scripts | P0-1 (16 missing paths) and any future drift | (a) v1.3.4. (b) Stage 2 item 0, before R-01 | Yes | (a) inside A-14; (b) S (1.5 h) |
| C-08 | **Host-e2e tier 1**: disposable sshd+systemd containers, fake GitHub API, stub runner, real `SystemExecutor` | P0-1, P0-2, P0-3, P1-10, P1-15/16/17, SEC-2/3/6, checkup findings | W2 (if go): one fixture container only (1.5 h inside W2). Full: Stage 2 item 0 | Yes (PR) | S (W2 fixture) / L (full) |
| C-09 | **Real-job canary tier 2**: the same container host, real GitHub, **dispatches a real workflow job** | Every workload failure (docker group, Rust, sudo, `services:`, `container:`) and self-update | v1.3.4: one manual run (**= A-20**). Stage 2: CI, 22.04 and 24.04 | Yes, for tags that claim BYO works | S (A-20) / M (CI) |
| C-10 | macOS `go test` job | checkup and Watch-adjacent code runs from laptops (darwin binaries ship, untested) | Inside W2 (if go) | Yes | S (0.5 h) |
| C-11 | Docs and copy contract test | `byo-prepare` in release notes; banned claims; dead commands in docs | After the memo (cut from Stage 1) | Yes | S (1 h) |
| C-12 | Dependabot (gomod + github-actions, weekly) | Stale deps; mutable action tags (keeps C-01b's pins fresh) | Stage 1, before C8 | n/a | S (0.25 h) |
| C-13 | Go toolchain bump (plus `x/*` refresh) | Out-of-support toolchain; unblocks the scaleset spike (E3.3) | **= A-17b**: v1.3.5 (§0 cut 2), or v1.3.4 if A-17a finds a reachable vulnerability | via C-05 | S (3.5 h; "roughly half a day" per gaps.md) |

- [ ] **C-01 Release gated on tests (= A-10), and C-01b SHA-pinned actions**
  - **Files:** `.github/workflows/release.yml`: a single `goreleaser` job, `go-version: '1.22'`, no test step, no `needs:`, no repository guard, and actions on mutable major tags.
  - **Change in v1.3.4 (C-01):**
    1. Add a `test` job: `go vet ./... && go test ./... -count=1 -race`. The C-03 binary tests join it when A-01 lands.
    2. `goreleaser` gets `needs: [test]` and `if: github.repository == 'accidentally-awesome-labs/runnerkit'`.
    3. Scope `permissions` per job: `test` gets `contents: read`.
  - **Change in Stage 1 (C-01b):** pin `actions/checkout`, `actions/setup-go`, `sigstore/cosign-installer` and `goreleaser/goreleaser-action` by full commit SHA, with a version comment. C-12 keeps the pins fresh.
  - **Acceptance:** pushing a tag on a commit with a failing test produces no release. Show this with a dry run on a fork, with the guard temporarily pointed at the fork.
  - **Test:** the workflow itself; `actionlint` once C-06 exists.

- [x] **C-02 Test hermeticity guard**
  - **Files:** `.github/workflows/pr-checks.yml` (the `go-test` job).
  - **Change:**
    - After `go test`, run `git status --porcelain --ignored -- . ':!dist'` and fail if the output is non-empty.
    - Set `RUNNERKIT_STATE_DIR=$RUNNER_TEMP/rkstate`, `HOME=$RUNNER_TEMP/home`, `RUNNERKIT_NO_UPDATE_NOTIFIER=1` and `GH_TOKEN=` for the test step.
    - Tests that hit the live GitHub releases API take an injected client.
  - **Acceptance:** on HEAD the guard fails, because of `internal/cli/sessions/byo-owner_repo__alice_example_com.json`. After A-03 it passes.

- [x] **C-03 Binary CLI contract tests** (`cmd/runnerkit/cli_contract_test.go`)
  - A `TestMain` builds the binary once with `-ldflags "-X main.version=9.9.9-test"`.
  - Table rows as in A-01, plus:
    - **Invariant:** for every row with a non-zero exit, `stderr` is non-empty.
    - **Exit codes:** usage errors exit 2; F-02 codes are asserted once F-02 lands.
    - **No CWD writes:** CWD = `t.TempDir()`, and it is still empty after each row.
  - **Acceptance:** fails on HEAD; passes after A-01 and A-03.

- [ ] **C-04 gofmt + golangci-lint**
  - Add a `lint` job running `gofmt -l .` (empty) and `golangci-lint run` with `govet, staticcheck, errcheck, ineffassign, unused, gosec (medium+)`.
  - Use `--new-from-rev=<the v1.3.4 tag>` so the 46 existing staticcheck findings do not block. Burn them down opportunistically.
  - Replace the phony `make lint` target (declared in the `Makefile` `.PHONY` list, no recipe) with a real one.
  - **Acceptance:** a PR that adds an unused function fails.

- [ ] **C-05 govulncheck**
  - v1.3.4: A-17a adds a scheduled weekly job and a PR job with `continue-on-error: true`, and posts findings to the job summary.
  - **Rule:** a finding that is **reachable** from shipped code paths (`os/exec` ssh, the `net/http` client, `crypto/tls`) pulls A-17b (C-13) into the next release. Before v1.3.4 is tagged, that means into v1.3.4.
  - Stage 2: blocking (STRATEGY §7 item 4).
  - Note: `vuln.go.dev` was unreachable from the research sandbox, so exposure is currently **unmeasured**, not known to be bad.

- [ ] **C-06 shellcheck of rendered scripts**
  - Add a test helper `internal/bootstrap/render_all_test.go` (build tag `shellcheck`). It renders every script with representative `Options`:
    - persistent and ephemeral;
    - with and without extra packages;
    - image setup;
    - finalizer, TTL, removal;
    - the cloud-init wait script from `up.go`;
    - each checkup probe (E1.2).
  - The helper writes them to `$TMP/rendered/*.sh`. CI then runs `shellcheck -S warning` on them, on `install.sh` and on `scripts/smoke/*.sh`, and runs `actionlint` on `.github/workflows/*`.
  - **Acceptance:** CI fails when a new SC2086 or similar is introduced. The research session prototyped a `zz_render_scripts_test.go` that renders and shellchecks every script; it was not committed, so rebuild it from this description.

- [ ] **C-07 sudo-allowlist checks**
  - (a) **v1.3.4, = A-14.** Full-body equality between `install.sh`'s rendered fragment and `RenderSudoersEntry("alice")`, replacing the header-only check at `install_sh_test.go:24-38`.
  - (b) **Stage 2 item 0, before R-01.** Parse every rendered script (the C-06 corpus) for `sudo [-n] <cmd>`, resolve `<cmd>` to an absolute path the way `secure_path` would, and assert that it is in the granted set. After R-01, the lint becomes: no `sudo` token except `sudo -n /usr/local/libexec/runnerkit-hostd`.
  - **Acceptance:** on HEAD, (a) fails with 16 missing paths and (b) lists the 11 commands.

- [ ] **C-08 Host-e2e tier 1 (hermetic host; runs on every PR; no secrets, so it works on forks).** Full harness: Stage 2 item 0 (E2.0). A-20 reuses its container recipe manually in v1.3.4.
  - **Hosts:**
    - `test/e2e/host/Dockerfile.ubuntu2404` and `Dockerfile.ubuntu2204`: systemd as PID 1, openssh-server, a user `alice` in `sudo` **with a password**, key auth.
    - Run with `--privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw`.
    - Docker-in-host for R-02 needs dockerd inside with `--storage-driver=vfs` (this worked in the gaps.md prototype). Alternative for full fidelity: an Ubuntu cloud-image VM under QEMU/KVM on the GitHub-hosted runner.
  - **GitHub side:** inject `deps.GitHub` with the existing fakes from `internal/testsupport`, run in-process through `cli.NewRootCommand`. The test records every call, including registration-token requests.
  - **Runner side:** add a DI seam `Dependencies.RunnerPackageFor` (default `bootstrap.PackageFor`) so the test serves a **stub runner tarball** from `httptest` with a matching SHA. The stub contains:
    - `config.sh`: **rejects an empty `--token`**, validates `--url/--name/--labels`, writes `.runner`/`.credentials`, and logs every invocation;
    - `run.sh`/`runsvc.sh`;
    - `svc.sh`: mimics upstream, including `Failed: error: exists` on a double install;
    - `bin/Runner.Listener --version`.
  - **Stage 1 (only if W2 gets a go):** the §0 ledger funds **one** fixture container (1.5 h inside W2): (i) plus (iv), plus a leftover `runnerkit-installer` sudoers file, plus the known_hosts cases. (ii) and (iii) wait for the full harness. The full list, for reference:
    - (i) a hand-installed `config.sh`/`svc.sh` layout;
    - (ii) the RunnerKit v1.3.3 layout;
    - (iii) a myoung34-style container runner;
    - (iv) planted problems: a 0-byte `runsvc.sh`, leftover `bin.2.328.0`, root-owned files in `_work`, a small tmpfs to simulate a full disk, a synthetic `oom-kill` kernel line, a synthetic "is deprecated and cannot receive messages" journal line;
    - known_hosts match, mismatch and missing (F-05).
  - **Full for Stage 2 (bootstrap):** after each step, assert:
    - `sudo -n` coverage;
    - `id -nG runnerkit-runner`;
    - `docker info` as the runner (with `--docker`);
    - unit active and root-owned;
    - root-owned `/var/lib/runnerkit`;
    - `install.sh`-only preparation;
    - the token absent from `journalctl _COMM=sudo`;
    - the CWD untouched;
    - stub `config.sh` invocation counts (upgrade and recover paths).
  - **Wiring:** `make e2e`, build tag `e2e`, workflow `.github/workflows/e2e.yml`.
  - **Acceptance:**
    - Stage 1 (W2 fixture): every checkup finding is detected on (iv), and zero files are created on the host outside `/tmp/ssh-*` (check with `find / -xdev -newer <stamp>` excluding `/proc`, `/run`, `/var/log/journal`).
    - Stage 2: running the v1.3.3 code through the harness reproduces the gaps.md results (a `setup_runner_image` failure and no docker group). After R-01..R-05, everything is green.

- [ ] **C-09 Real-job canary tier 2 (a hermetic disposable host that runs a real workflow job; gates any tag that says BYO works).** v1.3.4 uses the manual single-host form (A-20). This CI form, on 22.04 and 24.04, is Stage 2 (STRATEGY §7 item 4).
  - **Setup:**
    - A throwaway private canary repo, e.g. `accidentally-awesome-labs/runnerkit-canary`.
    - A fine-grained PAT with **Administration: read/write and Actions: read/write on that repo only**, stored in a GitHub **environment** `canary` with required reviewers, and available only to `workflow_dispatch` and tags.
    - **No Hetzner token in CI.**
  - **Flow:** on a GitHub-hosted runner:
    1. start the C-08 password-sudo container host;
    2. `runnerkit up --host alice@container --repo <canary> --labels canary-<run_id>`;
    3. `gh workflow run canary.yml -f label=canary-<run_id>`;
    4. poll the run to completion;
    5. `runnerkit down`;
    6. assert the runner is gone from the API.
  - **Canary job steps:**
    - `gcc hello.c`;
    - `docker build .` (with `--docker`);
    - `services: postgres` plus a `psql` select;
    - a `container: ubuntu:24.04` step;
    - `sudo -n apt-get install -y jq` (with `--ci-sudo packages`);
    - `actions/cache` save and restore;
    - checkout twice in a row after a container step wrote root-owned files (E2.2 hygiene).
  - **Self-update check:** a second leg installs the fallback pin one release behind and asserts the runner self-updates, then runs the job. This live-tests the auto-update premise every time (gaps.md).
  - **Policy:** this knowingly supersedes the Makefile's D-11 rule ("live smokes must NOT be invoked from CI") in a narrow way: a least-privilege PAT on a throwaway repo, and no cloud spend. Record the change in `docs/release-process.md`.
  - **Acceptance:** `release.yml` makes a tag that claims BYO support depend on a green C-09 for the same SHA.

- [ ] **C-10 macOS test job:** add `macos-latest` to the `go-test` matrix, unit tests only.
- [ ] **C-11 Docs and copy contract test:** `internal/docs_test.go` scans `*.md`, `CHANGELOG.md` and Go string literals.
  - Every `runnerkit <sub> [--flag]` mention must resolve in the Cobra tree. This would have caught `byo-prepare` in the v1.3.3 notes.
  - Banned phrases fail the test: `recommended default`, `recommended cloud`, `NOT a blanket`, `10 minutes`, `10-minute`, `cheaper than`, `cleaned up automatically`, `--mode ephemeral --cloud` (as a recommendation), `approx €`.
- [ ] **C-12 Dependabot:** `.github/dependabot.yml` for `gomod` and `github-actions`, weekly. Group minor and patch updates.
- [x] **C-13 Go toolchain bump:** now **A-17b** (§A.3), which has the files, fix and acceptance. gaps.md: a scratch bump to 1.26.3 built and passed every test.

---

## D. Epics by stage

### D.0 Stage 0 day-1 validations (3.5 h of work; they decide whether the probe is worth building)

These reuse A-20's container image and throwaway repo.

| ID | Question | Method | Pass / fail criterion | Decision it drives | Due |
|---|---|---|---|---|---|
| V-1 | Does GitHub already notify owners about runners that are offline, auto-removed or deprecated? | (a) Grep raw `github/docs` (`content/actions/**`, `data/reusables/**`) for runner notification text. (b) Stop the V-3 throwaway runner **by C3 (2026-10-12) for 15 days**, past the 14-day auto-removal, and record any email, notification or UI notice | GitHub notifies about a condition → drop that Watch rule | Kill criterion 3: which W1 rules survive | (a) C4 2026-10-17; (b) readout C6 2026-10-27 |
| V-2 | What is the minimal token for listing a personal repo's runners, and are `version`, `status` and `busy` populated? | Try `GITHUB_TOKEN` inside a workflow (expected to fail: the endpoint needs admin access per the REST description), then a fine-grained PAT with Administration: read. Inspect the JSON [new per the skeptic; schema-verified only] | Record the minimal permission set, and whether `version` is present (the schema marks it optional) | W1 permissions table; the F-04 source of truth | C4 |
| V-3 | Does a 2.334.0 runner installed **by hand** self-update and run a job? | In the disposable container, install 2.334.0 manually with `config.sh`/`svc.sh` (**not** RunnerKit `up`). Trigger one job; read `_diag` for the refresh message and exit code 3; check for a 0-byte `runsvc.sh` (#4421) | It updates to ≥2.337.0 and the job succeeds | Confirms the "never `--disableupdate`" invariant (A-16, N-15) and W1's version-rule semantics; V-1's throwaway runner | C4 |
| V-4 | Does fallback via variables work, and what happens to queued jobs? | In the throwaway repo: `runs-on: ${{ fromJSON(vars.RUNS_ON) }}`. Flip `RUNS_ON` between `["self-hosted","x"]` and `"ubuntu-latest"` with `gh variable set`, using a Variables: write PAT (docs-verified, never tested live). Queue a job while the runner is stopped, flip, and observe | New jobs route correctly both ways. An already-queued job keeps its target, and the recipe says to cancel and re-run it | W1 fallback; W3 recipe 3 | C4 |

- [ ] Publish V-2..V-4 and V-1(a) in one Discussion post by C4. Publish V-1(b) by C6.

### D.1 Stage 1: validate (pre-gate from C4 → launch at C8 → decision at C10)

**Order (STRATEGY §6, ruling R4):** W3 recipes and thread replies (the zero-code pre-gate) → W1 Watch → W2 checkup, only if ≥2 external requests arrive. The epic IDs keep their old numbers (E1.1 = W1, E1.2 = W2, E1.3 = W3) but are listed in build order. **Caps:** W1–W3 build ≤30 h, as the explicit exemption from the two-request rule; cumulative ≤55 h at launch and ≤85 h at the memo (§0 ledger).

#### E1.3 (W3, first) Recipes plus thread replies: the zero-code pre-gate (≈6 h, by C5 2026-10-24)

- **Problem.** The biggest pains have scattered, unanswered threads: #120813 (online but not picking up jobs; unanswered), #4442 (version deprecation), #434 (open since 2020) and #20019 (107 upvotes; GitHub has "no plans").
- **Scope:** `docs/recipes/`, human-written and useful without RunnerKit. They are also the fallback deliverable if the probe is killed.
  1. **"Runner online but not picking up jobs":** a label-subset check, runner group and repo access, the `busy` field, listener restart, stale sessions and time skew, and re-register as the last resort.
  2. **"Disk full and root-owned workspace files":**
     - an `ACTIONS_RUNNER_HOOK_JOB_COMPLETED` cleanup script;
     - a systemd prune timer with a build-cache cap (`docker builder prune --keep-storage`);
     - a `JOB_STARTED` per-job `HOME` reset for claude-code-action #1688;
     - an honest note that chowning root-owned files needs root, so the recipe shows a root-owned timer.
  3. **"Fall back to GitHub-hosted when your runner is down", for build and test jobs only:**
     - `runs-on: ${{ fromJSON(vars.RUNS_ON) }}` goes only in build/test jobs;
     - **deploy jobs keep their literal self-hosted labels**, or fall back to an SSH-based deploy step;
     - flipping the variable does not rescue already-queued jobs; cancel and re-run them (V-4);
     - flip manually with `gh variable set`, or later with Watch.
- **Thread replies:** one helpful reply each on #120813, #4442, #20019 and #434. Recipe first, authorship disclosed, and a mention that a read-only host checkup is being considered. Replies asking for host-side tooling count toward W2's go/no-go.
- **Prerequisite:** H-06 (history check) done before the first reply.
- **Non-goals:** bundling into the CLI (that is E2.2); covering ARC or Kubernetes.
- **Acceptance criteria:**
  - [ ] Every command was executed on a real runner (the V-3 container, or the maintainer's own runner) and the output was captured.
  - [ ] Each recipe stands alone (no RunnerKit install needed), fits in one screen of core steps, and is also published as a gist.
  - [ ] Written by the maintainer. AI help is allowed for checking, not for prose, because awesome-selfhosted-style venues reject AI-written submissions.
  - [ ] All four replies posted by C5.
- **Dependencies:** V-3, V-4, H-06.
- **Effort:** S (≈6 h: recipes 4.5 h, replies 1.5 h; STRATEGY §6 "about 6h").
- **Validation signal served:** external authors (≥5 distinct is a PASS condition); requests for host-side tooling (the W2 gate); the archive deliverable.

#### E1.1 (W1, second) `runnerkit-watch`: a scheduled GitHub Action in its own repo (by C7 2026-11-02)

- **Problem.** Runners silently go offline, stay online but take no jobs (#120813), get auto-removed after 14 days offline, or fall outside the 30-day update window (fully enforced for GHEC from 2026-09-25; the same date is presumed for personal accounts). Owners find out when CI or deploys stop.
- **Scope:**
  - A new repo `accidentally-awesome-labs/runnerkit-watch` with a composite action (`action.yml`, bash, `gh api`, `jq`), **≤800 LOC excluding tests**, runnable on `ubuntu-latest`. Marketplace-listable.
  - **Inputs:**
    - `repos` (default: this repo);
    - `runner-token` (PAT secret);
    - thresholds: `queued-after` (15m), `version-warn-days` (10), `version-escalate-days` (25), `registration-floor` (2.329.0), `offline-warn-days` (10), `token-expiry-warn-days` (14);
    - `known-runners` (optional allowlist);
    - `notify` (`issue` by default, plus optional `ntfy-url` and `webhook-url`);
    - `fallback` (off), with `fallback-variable` (`RUNS_ON`), `self-hosted-labels`, `hosted-label` (`ubuntu-latest`) and `variables-token`;
    - `report_usage` (`false`).
  - **Rules spec.** This is `docs/spec/runner-version-rules.md` plus the README rules table, and it is the single source that F-04 mirrors. The rules are config-driven so thresholds can change without a code change (STRATEGY §11):
    1. **Offline and disappearance.** The REST runner object has no last-seen field. Watch therefore records the **first-seen-offline time in the runner's issue body**, which survives cache eviction. Granularity equals the cadence (6 h by default), and the docs say so. Watch opens the issue on the first offline observation, and escalates at `offline-warn-days` (10), before the 14-day auto-removal. A runner that was in the last snapshot and is now missing gets "removed (auto-removal or manual)".
    2. **Idle-while-queued.** A job queued longer than `queued-after` whose `labels` are a subset of the labels of a runner that is **online and not busy** (#120813). It uses `GET /repos/{o}/{r}/actions/runs?status=queued` and `/runs/{id}/jobs`; the job schema has `labels` and `created_at`.
    3. **Version lag.** Let *next* be the oldest actions/runner release newer than the installed version, so its age is the 30-day clock:
       - **WARN** when installed < latest **and** *next* is more than `version-warn-days` old. The default of 10 (allowed range 10–14) sits above GitHub's documented "within a week" update for idle auto-updating runners, so healthy idle runners are not flagged.
       - **ESCALATE** when *next* is more than 25 days old: 5 days before the 30-day rule.
       - **ERROR** below the registration floor.
       - The API cannot tell pinned runners from auto-updating ones, and healthy runners lag until a job arrives, so a lagging runner is labelled **"suspected pinned or not updating"**, never "pinned".
       - #4613 (2.336.0 refused while it was the latest release) is documented as an unexplained case that Watch cannot predict.
    4. **Unknown registrations.** A diff against the last snapshot, kept in `actions/cache`. The first run, and any run after cache eviction, re-baselines and says so in its summary.
    5. **Token expiry.** Warn `token-expiry-warn-days` before a fine-grained PAT expires **[new; confirm how the API exposes expiry during the build. If it does not, drop the rule and document that]**.
    6. **Optional fallback, for build/test jobs only.** Flip `RUNS_ON` between the self-hosted labels and `ubuntu-latest` when every matching runner has been offline for at least one cycle, and flip it back on recovery.
       - Only jobs that reference `fromJSON(vars.RUNS_ON)` move. Deploy jobs keep literal labels (recipe 3).
       - Already-queued jobs keep their target.
       - At the 6 h cadence the flip is hours late.
       - Watch already holds a token on hosted infrastructure, so no third-party monitor sends an (unverified) authenticated PATCH (N-18).
  - **Opt-in, observable adoption metrics** (no default-on telemetry, N-20):
    - Watch's first issue links the pinned "I'm running Watch/checkup" Discussion (H-12) and asks for a +1.
    - `report_usage: true` (off by default) opens **one** issue on the Watch repo on the first run, containing only the Watch version and the date.
    - Public-repo dependents are counted from the dependents graph and reported **separately**. Private repos are invisible.
    - All counts are **lower bounds**.
  - **Alerts:** one GitHub issue per runner, labelled `runnerkit-watch`, updated in place and closed on recovery, written with `GITHUB_TOKEN` (`issues: write`).
  - **Cadence:** the default schedule is `17 */6 * * *`. That is about 120 billed minutes a month on a private repo, because each job rounds up to a minute (hourly ≈ 720) [STRATEGY §6, per skeptic].
- **Documented limits:**
  - Watch is a slow detector (≤6 h). The fast detector is E2.1's heartbeat.
  - Public-repo schedules are disabled after 60 days of repo inactivity.
  - Watch is blind during GitHub Actions outages.
  - It must run on GitHub-hosted runners, never on the watched runner.
- **Non-goals:** a hosted service, a dashboard, or storing tokens anywhere but repo secrets; third-party monitors PATCHing variables; org-wide scanning in v1.
- **Acceptance criteria:**
  - [ ] bats fixture tests from recorded API JSON for every rule, covering open, update and close, with no duplicate issues across runs.
  - [ ] A version-rule table test with a fixed clock (2026-09-26): 2.334.0 installed (next release 2.335.0, published 2026-06-08) gives ESCALATE; 2.337.0 gives nothing; 2.328.0 gives ERROR (below the floor).
  - [ ] Live on the V-3 container runner at a 15-minute test cadence: stopping the runner opens an issue within one cycle, with its first-seen-offline time recorded; starting it closes the issue.
  - [ ] A fallback flip tested live in the throwaway repo (per V-4), in both directions. A deploy job with literal labels does not move.
  - [ ] The README permissions table is based on V-2:
    - PAT: Administration: read and Actions: read on the watched repos;
    - Variables: write only with fallback;
    - `GITHUB_TOKEN`: `issues: write`.
  - [ ] Billed minutes measured over 7 days on a private repo, extrapolating to **≤150 min/month** at the default cadence.
  - [ ] shellcheck and actionlint clean; ≤800 LOC; a v1 tag as an immutable release; the example pinned by SHA; the Marketplace listing live.
- **Dependencies:** V-1 (drop the rules GitHub already covers), V-2, V-3, V-4; H-01 (license).
- **Effort:** M (≈12 h). It was 10 h before the token-expiry rule, `report_usage`, the release-age version rule and the spec document (moved from F-04) were added.
- **Validation signal served:** ≥10 external **opt-in** Watch users (Discussion +1s plus `report_usage` issues; lower bounds; public dependents reported separately); Watch catches counted in "real problems caught"; billed minutes ≤150/month.

#### E1.2 (W2, gated) `runnerkit checkup user@host`: a read-only host report for any runner, however it was installed

- **Gate (STRATEGY §6).** Build only if the pre-gate (W3 replies) or W1 yields **at least 2 external responses asking for host-side tooling**. Check at C7 (2026-11-02) and C9 (2026-11-23). If it is a go, build after launch unless the C7 re-baseline shows it fits under the 55 h cap, and ship it as v1.3.5 by **2026-12-07** (§0).
- **Problem.** Watch cannot see the host: disk, OOM, stale listeners, root-owned `_work`, broken self-updates. The prevalence of these problems is exactly what no dossier source could measure.
- **Scope (the minimum the ledger funds, 12 h):**
  - A new subcommand in the existing CLI that reuses `ops.Classify`, preflight's probe, `hostkillhint` and `remote.Executor`, with F-01, F-02, F-04 and F-05 as prerequisites and A-01/A-03 shipped alongside.
  - **Discovery:**
    - `actions.runner.*` systemd units, whether hand-installed or installed by RunnerKit, via `systemctl list-units` and `systemctl show`;
    - install dirs from the unit's `ExecStart`/`WorkingDirectory`;
    - `.runner` JSON, for the repo and runner name; **never read `.credentials*`**.
    - myoung34 container discovery is **cut** (STRATEGY W2's first cut).
  - **Checks, all read-only:**
    - service state and `NRestarts`;
    - listener staleness (the newest `_diag/Runner_*.log` timestamp);
    - installed version against latest and the window (F-04, same thresholds as W1);
    - "deprecated and cannot receive messages" journal lines;
    - a 0-byte `runsvc.sh` (#4421);
    - leftover `bin.<ver>`/`externals.<ver>` directories beyond the current and previous ones;
    - `df -P` and `df -Pi` for the install and work dirs;
    - `docker system df`;
    - root-owned files under `_work` (#434);
    - OOM kills (`journalctl -k -b 0` and `-b -1`);
    - time sync;
    - memory and swap;
    - **leftover RunnerKit sudoers files** (`/etc/sudoers.d/runnerkit-installer`, `runnerkit-runner-ci`), and a **root-run, runner-owned `svc.sh`**. Both link to the revocation steps (H-09).
  - **Security invariants:**
    - **never execute files under a runner install dir** (a compromised job could plant them);
    - no `sudo`, except an optional `sudo -n` for journal reads, which degrades gracefully with "add <user> to adm/systemd-journal";
    - strict `known_hosts` (F-05).
  - **Output:** human and JSON, with exit codes per F-02.
    - The anonymized `--summary` block is **cut** (STRATEGY W2's first cut). It is the first thing added back if W2 logs under 12 h.
    - Until then, a report is a Discussion post using the checkup-report template (H-12), with the finding IDs and severities pasted from `--json`, redacted by the user.
  - **Docs:** "What checkup reads" lists every command it runs, for trust.
- **Non-goals:** any mutation or auto-fix; installing anything; telemetry or sending data anywhere (the user posts results themselves); Windows or macOS hosts; org-level API queries.
- **Acceptance criteria:**
  - [ ] Against the W2 fixture container, which has a hand-installed layout and planted problems (a 0-byte `runsvc.sh`, leftover `bin.2.328.0`, root-owned `_work` files, a small tmpfs for a full disk, synthetic `oom-kill` and "deprecated" journal lines, a leftover `runnerkit-installer` sudoers file), every planted problem is reported with the right severity, and a clean fixture reports none.
  - [ ] Zero host writes, verified with a `find -newer` stamp.
  - [ ] Unknown, changed and missing host keys fail closed.
  - [ ] It works from macOS and Linux laptops (C-10).
- **Dependencies:** the gate; A-01 and A-03 (checkup reuses the CLI, so the CLI must not look broken); F-01, F-02, F-04, F-05; H-12 (Discussions).
- **Effort:** M (12 h: core 4, F-01 1.5, F-02 1, F-04 2, F-05 1.5, fixture 1.5, C-10 0.5), plus A-01/A-03/C-02 (3 h), which are counted in the post-launch ledger.
- **Validation signal served:** ≥15 external checkup reports (distinct non-maintainer accounts). **≥30% of reports show an actionable day-2 problem once there are ≥8 reports**, with "actionable" defined in E1.6 **before** launch. Under 20% with ≥8 reports means the hypothesis is false (kill criterion 5).

#### E1.4 Launch and distribution (one deliberate launch, week of 2026-11-09 (C8), no later than 2026-11-16)

- **Scope:**
  - Hook: "Is GitHub about to stop sending jobs to your self-hosted runner?", anchored on the 2026-09-25 enforcement and the 14-day auto-removal, **not on cost**.
  - Same week, all human-written (awesome-selfhosted forbids AI-written submissions): Show HN; r/selfhosted, r/homelab, r/github; a PR adding a BYO/day-2 row to `jonico/awesome-runners` (895★); the Marketplace listing (from E1.1).
  - Follow up in the pre-gate threads (W3) only where useful. The four replies themselves were posted at C5.
  - A dogfood post: the dat0 df=0 story and the recipe-2 fix.
- **Non-goals:** paid ads; buying a domain; a Hetzner community tutorial; launching on more than one date.
- **Acceptance criteria:**
  - [ ] All channels posted by 2026-11-16 (otherwise kill criterion 2).
  - [ ] Pre-launch checklist:
    - H-01, H-03, H-06, H-08, H-09 and H-11 done;
    - v1.3.4 tagged, or the cask deprecated;
    - W3 and W1 done;
    - the E1.6 definitions published;
    - cumulative hours ≤55 in the G-1 log.
- **Dependencies:** everything above, except W2.
- **Effort:** S (≈4 h: 2 h prep, 2 h posting).
- **Validation signal served:** all Stage 1 gate metrics start counting here.

#### E1.5 No-code discovery (≤5 h, in parallel: 2.5 h before launch, 2.5 h after)

- **Scope:** 10 short interviews with people from #120813, #4442, #20019 and #434, r/selfhosted and myoung34 issues. Ask what broke, how they found out, and whether they would run a read-only checkup, a root heartbeat agent, or neither. **No price questions** unless org adopters appear.
- **Acceptance:** anonymized notes and a tally table in a Discussion; at least 6 completed.
- **Validation signal served:** the "concrete setup requests ≥5" metric that gates the E2.4 rebuild; the heartbeat-vs-none question for E2.1.

#### E1.6 Metrics, definitions and the 2026-12-21 decision memo

- **Scope:** publish these definitions **before launch**, so the goalposts cannot move. Without default-on telemetry, adoption counts are opt-in and are reported as **lower bounds** (STRATEGY §6, ruling R5).
  - An *external* user or author: neither the maintainer nor the org.
  - An *opt-in Watch user*: a distinct external account that +1'd the pinned "I'm running Watch/checkup" Discussion, or opened a `report_usage` issue. Public-repo dependents are reported in a separate column and never merged into this count.
  - An *external checkup report*: a Discussion post from a distinct external account containing checkup output (the summary block, if it ships, or the template with finding IDs).
  - An *actionable problem*: at least one of:
    - offline or a stale listener;
    - outside the update window or below the floor;
    - disk ≥85% or inodes ≥85%;
    - root-owned `_work`;
    - an OOM kill in the last 2 boots;
    - a 0-byte `runsvc.sh`;
    - more than 2 leftover `bin.*` directories;
    - a leftover RunnerKit sudoers file or a root-run, runner-owned `svc.sh`.
  - A *real problem caught*: a Watch catch or a user-confirmed actionable checkup finding, recorded in an issue or Discussion labelled `caught-it` by the reporter or the maintainer, and quoted.

  Keep the weekly table (H-16). Publish the memo on **2026-12-21 (C10)** with the pre-committed thresholds:
  - **PASS**, all of:
    - ≥10 external opt-in Watch users **or** ≥15 external checkup reports;
    - ≥5 distinct external issue or discussion authors;
    - ≥3 real problems caught;
    - with ≥8 checkup reports, ≥30% of them showing an actionable problem.
  - **KILL**, any of:
    - <5 opt-in Watch users **and** <8 checkup reports **and** <3 external authors;
    - with ≥8 reports, actionable problems on <20% of hosts (fewer reports means grey zone);
    - more than 85 h spent.
  - **Grey zone:** anything else. One 6-week extension to **2027-02-01 (C11)** with no new features, then a binary decision that defaults to archive.

  Stars are reported but never decide the outcome.
- **Acceptance:** the memo is published on the date, whatever the result. On KILL, run H-17 within 2 weeks: publish the recipes as gists or a small composite action, archive the CLI, and hand off or archive Watch.
- **Effort:** S (≈3 h: 0.5 h for the definitions before launch, 2.5 h for the memo).

### D.2 Stage 2: extend (2027-01-04, or 2027-02-02 after a grey-zone pass → 2027-03-26 (C12); **only if Stage 1 PASSED**)

Build in the STRATEGY §7 order, and move to the next item only when the earlier ones are being used. STRATEGY sets no Stage 2 hour cap, so keep the G-1 log. **Pivot rule:** if the BYO rebuild (E2.4) exceeds 3 weeks or 15 live-smoke attempts, freeze `up` and tell users to install with `config.sh` or myoung34, then run `adopt`.

#### E2.0 (item 0) Honest privilege model: required before any root-installed component, independent of setup demand

- **Problem.** The v1.3.4 sudoers list is root-equivalent and only disclosed (SEC-1). Root runs a runner-writable `svc.sh` (SEC-2), and root writes into a runner-owned `/var/lib/runnerkit` (SEC-3). E2.1's root variant, E2.2 and E2.3 would all install root components on top of this.
- **Scope:**
  - C-07(b), then the full C-08 harness, so every later change is tested;
  - R-01, the root-owned fixed-verb helper installed once by `install.sh`;
  - R-05, root-owned systemd units rendered by RunnerKit (never `sudo ./svc.sh`) and a root-owned `/var/lib/runnerkit`.
- **Invariants:**
  - **root never executes runner-writable files**;
  - the install dir stays runner-writable, so self-update keeps satisfying the 30-day rule (N-14).
- **Non-goals:** any of the demand-gated rebuild (E2.4); a root-owned install dir.
- **Acceptance criteria:**
  - [ ] `sudo -n -l` for the SSH user shows exactly one NOPASSWD path.
  - [ ] C-08 asserts that no root-executed path is writable by the runner user.
  - [ ] A self-update from the fallback pin to latest still succeeds (C-09 self-update leg).
  - [ ] `docs/security-posture.md` moves SEC-1 to SEC-3 to "fixed", and keeps the revocation guide for hosts that never upgrade.
- **Dependencies:** Stage 1 PASS.
- **Effort:** L (R-01 L, R-05 M, C-07(b) S, C-08 full L; overlapping work).
- **Validation signal served:** a prerequisite for E2.1's root variant, E2.2 and E2.3; zero outages or data loss caused by RunnerKit.

#### E2.1 Heartbeat / dead-man's switch (Tier 0: no GitHub token on the host)

- **Problem.** Watch detects a dead host only after hours. Deploy-agent VPS owners (segment 1) need minutes.
- **Scope, in two variants (STRATEGY §7 item 1, ruling R6):**
  - **Before E2.0, the no-root variant only:** `runnerkit heartbeat install user@host --url <healthchecks.io | Uptime Kuma | ntfy URL>` installs a script under the SSH user's home and schedules it with a **user crontab**, or with a `systemd --user` timer where lingering is already enabled. It needs no sudo and no root-owned file. **[inference: enabling lingering can itself need privileges; check before offering the systemd variant]**
  - **After E2.0, the root variant:** a **root-owned** script `/usr/local/lib/runnerkit/heartbeat` (0755) and a systemd timer (every 2–5 min), installed through R-01's helper.
  - Both variants ping `<url>` when every discovered `actions.runner.*` unit is active and the disk is below its threshold. Otherwise they send each monitor's own failure signal with a reason: healthchecks.io `<url>/fail`, the Uptime Kuma push `status=down` parameter, or an ntfy priority message **[inference: check each service's docs before building]**. `heartbeat remove` undoes it.
- **Non-goals:** any GitHub credential on the host; a hosted receiver; metrics export.
- **Acceptance criteria:**
  - [ ] Stopping the runner unit gives a failure signal within one interval. Killing the container host gives the monitor's missed-ping alert within the interval plus grace (C-08).
  - [ ] The no-root variant makes zero `sudo` calls (recording executor). The root variant's files are root-owned and not writable by the runner user (C-08 assertion).
  - [ ] Tested with healthchecks.io, ntfy and Uptime Kuma push URLs.
- **Dependencies:** Stage 1 PASS; E1.5 shows that a heartbeat agent is acceptable to users. The root variant also needs E2.0.
- **Effort:** M.
- **Validation signal served:** ≥10 external hosts running heartbeat or hygiene for 30+ days, with 30-day retention ≥60%.

#### E2.2 Hygiene pack (the E1.3 recipes as managed hooks and timers)

- **Problem.** Disk and Docker rot, and root-owned workspace files (#434); the maintainer's own df=0.
- **Scope:**
  - `ACTIONS_RUNNER_HOOK_JOB_STARTED/COMPLETED` set in the runner `.env`, pointing to **root-owned** scripts that run as the runner user;
  - a root-owned timer and fixed verb `runnerkit-hostd hygiene chown-workspace <slug>` (the path is resolved by the helper and never passed in) to fix root-owned `_work` files;
  - a stray-container kill;
  - a tiered `docker system prune` with a build-cache cap;
  - cleanup of old `bin.<ver>`/`externals.<ver>` (keep current plus previous);
  - a per-job `HOME` for claude-code-action #1688;
  - journald and docker log rotation.
- **Non-goals:** workspace wipe by default (opt-in); container or microVM isolation.
- **Acceptance criteria:**
  - [ ] C-09: after a `container:` step writes root-owned files, the next job's checkout succeeds.
  - [ ] After N synthetic builds, docker disk stays under the cap.
  - [ ] Two consecutive jobs do not share `~/.claude`.
  - [ ] Hooks cannot be modified by the runner user.
- **Dependencies:** E2.0 (R-01's helper and R-05's units; STRATEGY §7: hygiene comes after item 0); E2.1.
- **Effort:** M.
- **Validation signal served:** ≥1 user-confirmed prevented incident per 3 active hosts per month; retention.

#### E2.3 `runnerkit adopt user@host`

- **Problem.** Most self-hosters already have a runner from `config.sh`. Re-registering is friction and a risk.
- **Scope:**
  - Discover the units and `.runner`, and build the state **without re-registration**.
  - Host-resident state `/var/lib/runnerkit/hosts.json` (root-owned), so the laptop state is only a cache.
  - Record provenance as `adopt` or `up` on every beta host.
  - Pair with F-07's `state forget`.
- **Non-goals:** changing the runner's registration, labels or user; moving the install dir.
- **Acceptance criteria:**
  - [ ] Adopting the C-08 hand-installed fixture makes zero registration-token requests (fake GitHub) and zero `config.sh` invocations.
  - [ ] `status`, `doctor`, `checkup` and `heartbeat` all work afterwards.
  - [ ] It is idempotent.
- **Dependencies:** E2.0 (STRATEGY §7: `adopt` comes after item 0); F-04, F-05.
- **Effort:** M.
- **Validation signal served:** the adopt-only pivot rule (≥50% of beta hosts arriving through `adopt` → freeze `up`).

#### E2.4 Rebuild BYO `up` beyond the v1.3.4 repair (v1.4.0): the §A.5 R-items outside item 0, with C-09 and the Stage 2 CI gates

- **Problem.** The v1.3.4 repair (A-14, A-18, A-19) made the documented path work on one tested host. It did not fix the heavy forced image, the opt-in-less Docker group, argv tokens, the shared Unix user, the missing upgrade path, or the robustness defects (§A.0 rows 2, 3, 24–26, 31–38).
- **Gate:** Stage 1 PASS, E2.0 done, **and ≥5 concrete setup requests** (§10 metric). Without them, `up` stays as v1.3.4 left it.
- **Scope, in this order:**
  1. R-02 (Docker opt-in);
  2. R-13, R-12, R-16 (bootstrap robustness);
  3. R-09, R-10 (profiles);
  4. R-03, R-04 (upgrade and version);
  5. R-06, R-07, R-08, R-18 (security);
  6. R-14, R-15, R-17.

  Add C-04, C-05 (blocking), C-06 and C-11 as CI gates (STRATEGY §7 item 4: govulncheck and shellcheck). C-09 must be green before the tag.
- **Non-goals:** a root-owned install dir (it breaks self-update); widening `RenderSudoersEntry`; arm64 or non-Ubuntu (Z-05, Z-06); cloud.
- **Acceptance criteria:**
  - [ ] C-09 green on Ubuntu 22.04 and 24.04 password-sudo hosts prepared **only** by `install.sh`, for gcc, docker build, `services:`, `container:`, `sudo apt-get` (ci-sudo) and self-update.
  - [ ] Tokens are absent from the sudo logs.
  - [ ] `docs/security-posture.md` is updated to the new model.
- **Dependencies:** E2.0; E2.3 first, because adopt shares the host-state layout.
- **Effort:** L–XL (3–5 weeks; the pivot rule applies).
- **Validation signal served:** "real-job canary green on every tag that claims BYO works: 100%"; zero outages or data loss caused by RunnerKit.

#### E2.5 Scope deletion

- **Problem.** About 1.7k production LOC of Hetzner code, the ephemeral paths, the wizard and apt auto-detection are frozen, unvalidated and a source of defects.
- **Scope:** for each feature (cloud, ephemeral, the wizard, apt auto-detection) with fewer than 5 requests by the 2027-03-26 gate, one release **warns** (and keeps `destroy` working for existing cloud state), and the next release removes it. Existing cloud state gets a clear message: the last version that can `destroy`, and the Hetzner console filter `runnerkit=true`.
- **Non-goals:** removing `destroy` before the warning release.
- **Acceptance criteria:**
  - [ ] The binary no longer registers the removed commands and flags.
  - [ ] Loading a state file with cloud entries prints the migration message.
  - [ ] LOC and test counts are reported in the CHANGELOG.
- **Dependencies:** a request tally from Discussions and issues.
- **Effort:** M.
- **Validation signal served:** maintainer hours (a smaller surface).

#### E2.6 Stage 2 gate memo (2027-03-26)

- [ ] Publish the memo against these thresholds, all required:
  - ≥25 weekly-active watched repos or hosts (opt-in counts, reported as lower bounds, as in E1.6);
  - ≥10 external hosts running heartbeat or hygiene for ≥30 days, with ≥60% 30-day retention;
  - ≥1 prevented incident per 3 active hosts per month;
  - zero outages or data loss caused by RunnerKit.

  On a fail, move to maintenance mode (Watch and checkup kept current) and stop. Retention under 40%, or fewer than 10 retained hosts, also means maintenance mode.

### D.3 Stage 3: expand (2027-04 → 2027-09-26). Optional bets, each with its own trigger

Do not start any of these without its trigger. Each bet is written up only to the depth needed to decide.

| ID | Bet | Problem | Scope (if triggered) | Non-goals | Acceptance | Dependencies | Effort | Trigger / signal |
|---|---|---|---|---|---|---|---|---|
| E3.1 | N runners per host + org-level runners and groups | One runner per repo serializes matrix legs (P1-22; `labels.go:73-77` fixed name, `--replace` at `script.go:56`) | `--runners N`, host discriminator in names, `--labels` (wire the unused `ExtraLabels`), org runners limited to selected private repos | GitHub App auth; autoscaling | A 6-leg matrix runs 3 legs concurrently with `--runners 3` (C-09) | E2.4 (per-repo users, units) | L | ≥5 inbound team/org requests, or a top-3 request. Prerequisite for any team offer |
| E3.2 | Security `audit` / `doctor --security` | Rogue-runner and supply-chain waves target CI | Unknown-runner diff (from W1), visibility re-check, host posture summary. **Workflow linting delegated to zizmor** | Rebuilding a workflow linter | Detects a planted unknown runner in the fixture; wraps zizmor output | W1 rule data | M | Watch's unknown-runner rule records ≥1 true positive, or a new rogue-runner wave creates pull |
| E3.3 | Agent pools (JIT or scale set, sandbox per job, egress allowlist) | GitHub recommends single-use runners for agents | 2-day spike on a **personal** repo: a repo-level scale set with a fine-grained PAT, 3 parallel jobs, one Copilot cloud agent session. Then a pool behind an internal interface (actions/scaleset is in preview; breaking change 2026-09-15) or a REST `generate-jitconfig` loop | Claiming Copilot code review (ARC-only); breaking the deploy-agent use case for existing users | Spike report; pool passes C-09 with per-job fresh sandbox | A-17b (the Go bump the scaleset module needs), E2.0, E2.4 | XL | ≥5 requests for agent jobs on owned hardware **and** the spike works on a personal repo **and** ≥3 users commit. Until then ship only the #1688 HOME hook and a compatibility matrix |
| E3.4 | Second provider / revive cloud | Cloud frozen with Z-01..Z-10 open | Z-01..Z-10, then optionally a second provider | Reselling compute | Live authenticated `/v1/pricing` + `/v1/server_types` check in CI (spend-capped); C-09 cloud leg | Z-04 first | L | ≥5 explicit requests **and** a live pricing check. Otherwise the advice is "provision any box, then `adopt`" |
| E3.5 | macOS | Not supported; docs had claimed it | Design only after the trigger; Tart is now OpenAI-owned and FSL-licensed | — | — | — | XL | ≥10 requests |
| E3.6 | Forgejo runner adapter | The same pet-runner problem exists on Codeberg/Forgejo, with no hosted fallback | `checkup` and heartbeat for `forgejo-runner` units | Forgejo API parity | Checkup detects Forgejo runner units in a fixture | E1.2, E2.1 | M | ≥10 requests or a partner (Codeberg data was unverifiable) |
| E3.7 | Commercial layer | Payment is unproven | A RunsOn-style offline signed licence file for org features (multi-host, org runners, fleet reports, audit export), offline and with no phone-home, in a separate source-available module. Single-host stays free forever. Verified comparables: RunsOn €300/yr (README) and Actuated $150/month for the first server plus $125 for each additional one (MARKET-RESEARCH §7.1). The commercial lens's €29/org/month (€290/yr) is a **lens estimate, not dossier-sourced** | A hosted plane, phone-home, paywalled security | 10+ interviews; a priced waitlist with ≥10 org sign-ups before any code | E3.1 | L | ≥100 active hosts **and** ≥5 inbound team requests. STRATEGY §9: the lens's €2–9k ARR base case is an estimate, not dossier-sourced. A buyer must be at roughly 30k+ min/month or own hardware, and buy on capability, not price |

---

## E. Explicitly NOT doing

| ID | Not doing | Why (evidence) | Revisit only if |
|---|---|---|---|
| N-01 | "10-minute setup" or "cheaper than GitHub" messaging | An inferred 97% or more of individuals fit in the free minutes. Public repos are free. cpx22 break-even is about 5,900–6,900 min/month [S]. The *projected* BYO first run is 8–12 min on small hosts or slow links (gaps.md projection, not measured end to end) [I] | Never, as a headline |
| N-02 | Widening the sudoers allowlist (adding entries to `RenderSudoersEntry`), or calling it "scoped" | Root-equivalent since the first version (SEC-1). 62% of fix commits touched it. 21 live-smoke attempts in Phase 6. A-14 copies the existing list into install.sh: more lines there, no new capability, disclosed (STRATEGY §4 stop #2) | Never. It is replaced by R-01 (Stage 2 item 0) |
| N-03 | Recommending Hetzner cloud; publishing any price not fetched from the Hetzner API at plan time | €4.90 was about 4× wrong for new orders [S]; fsn1 stock-outs [S]. Cloud is behind `--experimental` (A-15), with live API prices only (A-07) | E3.4 trigger with a live pricing check |
| N-04 | Feature work driven by AI planning; a public `.planning` corpus | 265 commits in 21 days, 15.4k LOC, 0 external requests; the planning drift caused the misdirected v1.3.3 fix | Never. Rule: **2 distinct external requests per feature** |
| N-05 | Tagging a release that claims BYO works without a real job on a fresh password-sudo host | v1.3.1–v1.3.3 shipped broken BYO | Never (A-20 for v1.3.4; C-09 from Stage 2) |
| N-06 | Agent-pool, scale-set and autoscaling roadmaps | No solo demand evidence. scaleset is in preview and churning. It puts an admin credential next to prompt-injectable agents | E3.3 trigger |
| N-07 | arm64, non-Ubuntu, macOS or Windows support work | Each is broken or absent; support cost exceeds the evidence of demand | Z-05/Z-06 (≥5 requests), E3.5 (≥10) |
| N-08 | Multi-provider work; hcloud-go v2 migration in month 1 | Cloud is frozen. The `datacenter` reads are nil-guarded (low impact), and v1 already exposes the pricing A-07 needs | E3.4 trigger |
| N-09 | MCP / SEED-003 agent plugin; a "set up a runner from chat" channel | Onboarding is broken, there are no users, and the JSON contract is heterogeneous | Stage 2 PASS **and** ≥5 requests |
| N-10 | UX polish roadmap: wizard rework, `--explain`/`--unicode`/`--no-color`, checklist resume, verb consolidation (P2-1, P2-2, P2-7) | Cosmetic; no user asked; wizard scheduled for deletion (E2.5) | ≥5 requests |
| N-11 | Ephemeral-mode fixes, beyond A-08's block and A-15's `--experimental` gate | Never live-smoked; one job per `up`; a JIT pool would be a rewrite | Z-03 / E3.3 triggers |
| N-12 | Proper apt auto-detection parser | Off by default after A-11 | Z-07 (≥5 requests) |
| N-13 | A unified JSON envelope, a published JSON Schema, and SEED-003 contract work | Only F-02's exit codes are needed for Watch and checkup automation | N-09's trigger |
| N-14 | A root-owned install dir | Breaks the runner self-update that keeps the 30-day rule satisfied (gaps.md) | Never. Use "root never executes runner-writable files" (E2.0: R-01, R-05) |
| N-15 | Passing `--disableupdate`, anywhere | gaps.md reads every 2026 "deprecated and cannot receive messages" report as a non-updating runner; that is explicit only in #4203 and #4392, and #4613 is unexplained (MARKET-RESEARCH §2.5). GitHub's docs say auto-updating runners meet the 30-day rule | Never. A-16's test enforces it from v1.3.4 |
| N-16 | Rebuilding workflow linting | zizmor exists | Never. Delegate (E3.2) |
| N-17 | Claiming Copilot code review support, or "outbound-only" as a differentiator | Code review is ARC-only by policy; Zoomies already has outbound-only agents | Never |
| N-18 | A hosted Watch tier; third-party monitors PATCHing variables; any hosted service that holds users' GitHub admin tokens or can reach their hosts | Trust and liability; STRATEGY §9 "never" | Never |
| N-19 | Reselling per-minute compute; raising money; operating root CI for customers ("concierge"); a Coolify-style $3–5/host hosted plane | BuildJet shut down; $5k MRR would need about 1,250 paying hosts, i.e. 25–42k active hosts, close to the whole estimated population (STRATEGY §9) | Never |
| N-20 | Default-on telemetry; paywalling security defaults | Trust; STRATEGY §9 | Never. Adoption is counted only through opt-in signals: Discussion +1s, Watch's `report_usage` (off by default), and user-posted checkup reports |
| N-21 | Using stars as a gate signal; paid ads; buying a domain; a Hetzner community tutorial before traction | STRATEGY §6 | Never for stars. The others after a Stage 2 PASS |
| N-24 | Building W2 `checkup` without ≥2 external requests for host-side tooling; any feature outside W1–W3 without 2 distinct external requests | STRATEGY §4 stop #4 and §6 (ruling R4); W1–W3 are the only exemption, capped at 30 h | The request counts, checked at C7 and C9 |
| N-22 | The remaining P2 polish: P2-3..P2-6, P2-10..P2-12, P2-14..P2-20 from CODEBASE-ASSESSMENT §5 (jargon sweep, `init` output, list I/O, preflight NTP/curl nits, SSH multiplexing, slug collisions, pagination, errcode registry drift, cloud-init YAML via Sprintf, output polish, dead code) | Low severity; consumes the hour cap | Only when touched by an in-scope item, or in maintenance mode |
| N-23 | Rewriting git history by default | Breaks 18 tags and release provenance | H-06 finds credentials or private IPs |

---

## Appendix: source map

| Topic | Source |
|---|---|
| Strategy, stages, gates, hour caps, kill criteria | `STRATEGY.md` §5–§11; the calendar (C1–C13) is §10.1; rulings R1–R10 are its revision log |
| P0/P1 defect evidence and severities | `CODEBASE-ASSESSMENT.md` §5–§6 (IDs P0-n, P1-n, P2-n, SEC-n; authoritative for severity); `evidence/code-condensed.md` (stream IDs and verifier verdicts); KEYFACTS §B labels are noted only where they differ |
| hcloud-go v1.59.2 pricing API (A-07) | Module source `hcloud/pricing.go:14,68,103,137` (`Pricing.PrimaryIPs`, `ServerTypeLocationPricing`, `PricingClient.Get`), read 2026-09-26 |
| Released-binary BYO failure, disk and time footprint, job simulation | `evidence/gaps.md` "released-byo-e2e-job". The research session's prototype harness (`jobsim.sh`, a fake GitHub API, rendered scripts) lived in ephemeral scratch space and was not committed; `evidence/gaps.md` describes its method |
| Runner version floor, 30-day rule, self-update mechanics | `evidence/gaps.md` "runner-version-enforcement" |
| REST `runner` schema (`version` optional), runner-listing needs admin access, job `labels` field | GitHub's OpenAPI description (github/rest-api-description), read 2026-09-26 **[new in this document; schema-level only, not live-tested → V-2]** |
| Market, segments, launch channels | `evidence/KEYFACTS.md` §D; `evidence/market-brief.md`; `evidence/players.md` |
| Reference re-verification and silent-error reproduction | This document, 2026-09-26, HEAD `64c3003` (scratch build outside the repo; the repo was not modified) |

