# Validation metrics and go/kill thresholds

RunnerKit is experimental and maintained on a capped-hours basis until a
published go/kill decision on **2026-12-21**. The definitions and
thresholds on this page are the criteria for that decision. They were
fixed and published before launch so that the goalposts cannot move, and
the decision memo is published on that date whatever the result.

Without default-on telemetry, adoption counts are opt-in and are reported
as **lower bounds**. The counts, and the maintainer's logged hours, are
kept in a weekly table in a pinned GitHub Discussion.

Terms used below:

- *Watch* is `runnerkit-watch`, a scheduled GitHub Action that watches
  runners, and *checkup* is `runnerkit checkup`, a read-only host report
  (W1 and W2 in [CONTRIBUTING.md](../CONTRIBUTING.md#scope-rule)). Watch's
  `report_usage` input is off by default; when it is set to `true`, Watch
  opens one issue on its own repository on the first run, containing only
  the Watch version and the date. That is a `report_usage` issue.
- *The org* is the `accidentally-awesome-labs` GitHub organization.
- The *update window* is GitHub's rule that a self-hosted runner must
  install each new runner release within 30 days, or GitHub stops queuing
  jobs to it. The *floor* is the oldest runner version GitHub accepts for
  registration (2.329.0 when these definitions were fixed).
- C10 is the decision date, 2026-12-21. C11 is 2027-02-01, the end of the
  only extension the thresholds allow.

## Definitions

- An *external* user or author: neither the maintainer nor the org.
- An *opt-in Watch user*: a distinct external account that +1'd the pinned
  "I'm running Watch/checkup" Discussion, or opened a `report_usage` issue.
  Public-repo dependents are reported in a separate column and never merged
  into this count.
- An *external checkup report*: a Discussion post from a distinct external
  account containing checkup output (the summary block, if it ships, or the
  template with finding IDs).
- An *actionable problem*: at least one of:
  - offline or a stale listener;
  - outside the update window or below the floor;
  - disk ≥85% or inodes ≥85%;
  - root-owned `_work`;
  - an OOM kill in the last 2 boots;
  - a 0-byte `runsvc.sh`;
  - more than 2 leftover `bin.*` directories;
  - a leftover RunnerKit sudoers file or a root-run, runner-owned `svc.sh`.
- A *real problem caught*: a Watch catch or a user-confirmed actionable
  checkup finding, recorded in an issue or Discussion labelled `caught-it`
  by the reporter or the maintainer, and quoted.

## Thresholds for the decision on 2026-12-21 (C10)

The decision memo applies these pre-committed thresholds:

- **PASS**, all of:
  - ≥10 external opt-in Watch users **or** ≥15 external checkup reports;
  - ≥5 distinct external issue or discussion authors;
  - ≥3 real problems caught;
  - with ≥8 checkup reports, ≥30% of them showing an actionable problem.
- **KILL**, any of:
  - <5 opt-in Watch users **and** <8 checkup reports **and** <3 external
    authors;
  - with ≥8 reports, actionable problems on <20% of hosts (fewer reports
    means grey zone);
  - more than 85 h spent.
- **Grey zone:** anything else. One 6-week extension to
  **2027-02-01 (C11)** with no new features, then a binary decision that
  defaults to archive.

Stars are reported but never decide the outcome.

On KILL, the project is archived within 2 weeks, following the
[archive playbook](maintainers/archive.md).
