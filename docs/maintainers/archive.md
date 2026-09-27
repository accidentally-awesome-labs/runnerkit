# Archive playbook

This is the checklist for shutting RunnerKit down in an orderly way. It was
written in advance and is used **only** after one of the decisions in
[When to use it](#when-to-use-it): a kill decision or a failed capacity
check. Until one of those decisions is made, it is not a plan and nothing
in it applies.

An archived repository is read-only: its code, releases and discussions
stay readable, but nothing new can be pushed, released or posted. So
everything users need on the way out (the notice, the recipes, the
Homebrew change and the final numbers) goes out **before** the repository
is archived, which is the last step.

## When to use it

Use this playbook after one of these decisions, and name the decision in
the README notice and in the final Discussion post:

- **The capacity check fails (due 2026-10-03).** Capacity means
  availability, not spend: at least 2 work sessions a week, and issue
  responses within 72 hours, through 2026-12-21. If the maintainer cannot
  commit to that, follow
  [If the capacity check fails](#if-the-capacity-check-fails) first, then
  the [checklist](#checklist).
- **No progress.** No logged progress for 2 consecutive weeks, or any
  milestone in the table below slipping by more than 7 days. Finish the
  harm-reduction release first if it is unfinished (the release steps in
  [If the capacity check fails](#if-the-capacity-check-fails)), then
  archive with a notice.
- **Late or over-budget launch.** Launch not done by 2026-11-16, or more
  than 55 hours spent before launch.
- **KILL on 2026-12-21.** The decision memo meets a KILL threshold in
  [validation-metrics.md](../validation-metrics.md). Run the checklist
  within 2 weeks of the memo.
- **The grey zone ends in archive.** A grey-zone result gets one 6-week
  extension to 2027-02-01 with no new features, then a binary decision
  that defaults to archive.

Progress and hours are the ones the maintainer logs in the weekly table in
the pinned GitHub Discussion. W1 to W3 are the validation work described
in [CONTRIBUTING.md](../../CONTRIBUTING.md#scope-rule); V-1 to V-4 are in
[testkit/validations.md](../testkit/validations.md).

| Date | Milestone |
| --- | --- |
| 2026-10-03 | Capacity check; hour log and weekly table started |
| 2026-10-10 | LICENSE, README banner, corrected release notes and the security page with its revocation section published; v1.3.4 tagged (with the BYO repair or the BYO refusal), or the Homebrew cask deprecated |
| 2026-10-12 | Latest start of the 15-day offline test (V-1) |
| 2026-10-17 | End of the stabilization stage (Stage 0): V-2 to V-4 and the V-1 documentation check published; at most 25 hours logged |
| 2026-10-24 | W3 recipes posted, with replies in four existing GitHub threads about runner problems |
| 2026-10-27 | V-1 offline-test readout published |
| 2026-11-02 | W1 (`runnerkit-watch`) released; first go/no-go for W2 (`runnerkit checkup`) |
| Week of 2026-11-09 (latest 2026-11-16) | Launch; at most 55 hours logged |
| 2026-11-23 | Second go/no-go for W2 (if go, W2 ships by 2026-12-07); at most 30 hours logged on building W1 to W3 |
| 2026-12-21 | Decision memo; at most 85 hours logged |
| 2027-02-01 | End of the grey-zone extension, if there is one |

## If the capacity check fails

Do not archive the repository as it stands. A public tool that grants root
and makes false safety claims is itself harmful, so the archive must leave
behind a release that refuses its known-destructive commands and docs that
say what it really does. Do only this work, then run the
[checklist](#checklist) with an honest notice.

- [ ] **License, hygiene and honesty work is on `main`.** Some of these were
  done before this playbook was written; check each one:
  - `LICENSE` (Apache-2.0), `CONTRIBUTING.md`, `SECURITY.md` with private
    vulnerability reporting turned on in the repository settings, and
    `CHANGELOG.md` with the old `RELEASE-NOTES-*.md` files folded into it
    and the files removed from the tree;
  - the internal planning files (`.planning/`), `smoke-output.log` and
    `GEMINI.md` removed from the tree, with `.planning/` kept in a private
    repository;
  - the Homebrew tap token (`HOMEBREW_TAP_GITHUB_TOKEN`) rotated, as
    [release-process.md](../release-process.md#2-create-the-homebrew_tap_github_token-repo-secret)
    describes;
  - the README "Known issues" banner, and
    [security-posture.md](../security-posture.md) with its
    [If you already installed RunnerKit](../security-posture.md#if-you-already-installed-runnerkit)
    section;
  - a correction at the top of the v1.3.3 GitHub Release page: its notes
    tell users to run `byo-prepare`, which was removed in v1.0.8.
- [ ] **The v1.3.4 harm-reduction changes are on `main`.** They are listed
  in the v1.3.4 section of [CHANGELOG.md](../../CHANGELOG.md): the
  commands that could unregister a working runner are disabled, cleanup
  commands refuse Hetzner state instead of orphaning a server that keeps
  billing, the cloud path and BYO ephemeral mode need `--experimental`,
  ephemeral cloud runners are disabled, and no price is shown unless the
  Hetzner API reported it.
- [ ] **Merge the BYO refusal (A-21 in the CHANGELOG).** It is
  [draft pull request #6](https://github.com/accidentally-awesome-labs/runnerkit/pull/6).
  With it, BYO `up` and `register` refuse with `byo_unsupported_release`
  (exit 2) unless `--accept-known-issues` is passed, and the README and
  CHANGELOG say "BYO is not supported in this release." A release may say
  BYO works only after a real GitHub job has passed on a fresh
  password-sudo host prepared only by that release's `install.sh`; that
  check is not part of this minimal path, and an archived repository must
  not claim BYO works. The banner keeps its line that the `install.sh`
  sudoers fragment is root-equivalent, with the link to the security page.
- [ ] **Tag v1.3.4.** Follow the
  [pre-tag checklist](../release-process.md#pre-tag-checklist), taking its
  "BYO is not supported" branch at the real-job step, so that Homebrew and
  GitHub Releases users get the refusals. If the tag cannot ship, deprecate
  the Homebrew cask as that page says; step 3 of the checklist below then
  replaces the deprecation with `disable!`.

The reason in the README notice is then that the maintainer could not
commit the time the project needs.

## Checklist

Copy this list into a tracking issue and tick it there. Keep the order:
step 6 makes the repository read-only.

- [ ] **1. Put a notice at the top of the README.** Directly under the
  title, say that RunnerKit is archived and no longer maintained, the
  reason (which decision above, in a sentence or two) and the date of the
  decision. Link
  [If you already installed RunnerKit](../security-posture.md#if-you-already-installed-runnerkit),
  which tells users how to remove the root-equivalent sudo rules and the
  other access RunnerKit left on their hosts. Replace the **Status:** line
  with the same message; `TestReadmeHonestyBanner` in
  `internal/cli/docs_test.go` checks that line word for word, so change the
  test in the same pull request. Steps 2, 4 and 5 add their links to this
  notice.
- [ ] **2. Publish the recipes, if they were written.** The runner hygiene
  recipes (W3, in `docs/recipes/`) work without RunnerKit, so they should
  outlive it. Publish each one as a GitHub gist, or all of them as a small
  composite GitHub Action in its own repository, and link them from the
  README notice. If no recipes exist, skip this step.
- [ ] **3. Disable the Homebrew cask.** In
  [`accidentally-awesome-labs/homebrew-tap`](https://github.com/accidentally-awesome-labs/homebrew-tap),
  edit `Casks/runnerkit.rb` and add a `disable!` stanza after the
  `homepage` and `livecheck` stanzas, dated the day of the decision:

  ```ruby
  cask "runnerkit" do
    # version, sha256, url, name, desc, homepage and livecheck stay as they are
    disable! date: "YYYY-MM-DD", because: "is no longer maintained; see https://github.com/accidentally-awesome-labs/runnerkit"
    # binary and the remaining stanzas stay as they are
  end
  ```

  Homebrew prints the reason after "because it", so it starts with a
  verb. If the cask already has a `deprecate!` line (added when a release
  missed its date), replace it with the `disable!` line. Homebrew then
  refuses to install the cask and prints the reason. Push no further `v*`
  tag after this: the release workflow regenerates the cask from
  `.goreleaser.yaml` and would drop the stanza.
- [ ] **4. Hand `runnerkit-watch` to a volunteer, or archive it.** Watch
  (W1) is a scheduled GitHub Action in its own repository,
  `accidentally-awesome-labs/runnerkit-watch`, and other people's
  workflows may run it. If someone outside the project offers to maintain
  it, transfer the repository to them (Settings, General, then "Transfer
  ownership" under "Danger Zone") and link its new home from the README
  notice. Otherwise put the same kind of notice in its README and archive
  it as in step 6. If Watch was never released, skip this step.
- [ ] **5. Post a final Discussion with the data.** Post it in this
  repository's Discussions before step 6, pin it, and link it from the
  README notice. Include:
  - the decision, its reason and its date;
  - the last row of the weekly table, with the metrics defined in
    [validation-metrics.md](../validation-metrics.md): hours logged,
    opt-in Watch users, public-repo dependents (as a separate number),
    distinct external authors, checkup reports, real problems caught and
    Watch billed minutes;
  - a link to the decision memo, if one was published;
  - where the recipes and Watch went.
- [ ] **6. Archive the repository.** In
  `accidentally-awesome-labs/runnerkit`, open Settings, General, and under
  "Danger Zone" choose "Archive this repository". Do this last: from here
  on the repository is read-only.
