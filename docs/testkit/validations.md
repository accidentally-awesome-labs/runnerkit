# Stage 0 validations (V-1 to V-4)

Four questions decide whether the planned `runnerkit-watch` Action and the
"fall back to GitHub-hosted" recipe are worth building, and what they must
do. Each has a script in [`scripts/testkit/`](../../scripts/testkit/). They
reuse the gate's host, test repository and token
([release-gate.md, part 1](release-gate.md#part-1-prepare)), so run them in
the same session: V-2, V-4 and V-3 after the gate and before the revocation
drill, then V-1 last.

Record each outcome in the [results](#results) table and commit it.

## V-1 offline runner

**Question.** Does GitHub warn the owner when a self-hosted runner goes
offline, is removed after 14 days offline, or falls too far behind to get
jobs? If it does, `runnerkit-watch` drops those rules.

### What GitHub's documentation says (checked 2026-09-26)

Checked against `github/docs` at commit `18945a3` (2026-09-25): the
Actions runner pages, `data/reusables/actions`, and the notification docs
in `content/subscriptions-and-notifications`.

- "A self-hosted runner is automatically removed from GitHub if it has not
  connected to GitHub Actions for more than 14 days" (1 day for an
  ephemeral runner). (`data/reusables/actions/self-hosted-runner-auto-removal.md`)
- "If the job remains queued for more than 24 hours, the job will fail."
  (`content/actions/reference/runners/self-hosted-runners.md`)
- "If you do not perform a software update within 30 days, the GitHub
  Actions service will not queue jobs to your runner."
  (`data/reusables/actions/self-hosted-runner-update-warning.md`) A runner
  that updates itself "updates itself when a job is assigned to the runner,
  or within a week of release if the runner hasn't been assigned any jobs."
- The only advice on staying informed is to subscribe to releases of the
  `actions/runner` repository.
- The Actions notification settings cover **workflow runs** only ("Only
  notify for failed workflows"); the email reason `ci_activity` is "A
  GitHub Actions workflow run that you triggered was completed." No setting
  or notification mentions a runner going offline, being removed or being
  out of date.

So GitHub documents no warning to the owner. The one indirect signal is the
failure notice for a job that stayed queued for 24 hours, and only when
someone triggered such a job. The live test checks whether anything
undocumented arrives.

### Live test (start by 2026-10-12)

Start it at the end of the gate session, after the
[revocation drill](release-gate.md#part-3-revocation-drill):

```bash
ssh -t rkadmin@<ip> "sudo systemctl stop 'actions.runner.*'"
scripts/testkit/offline-watch.sh --repo you/rk-gate --runner runnerkit-you-rk-gate-local --queue-job start
```

Use the runner name from the gate's `EVIDENCE.md`. `start` refuses while
GitHub still shows the runner online; wait a minute and retry.
`--queue-job` dispatches `rk-probe.yml` at the runner's labels, so a job
waits for it and fails after 24 hours.

Then delete the host (or run `scripts/testkit/host-container.sh down`), so
the runner cannot reconnect, for example after a reboot. The GitHub
registration stays until GitHub removes it.

Optional, for exact removal timing: copy
`scripts/testkit/workflows/rk-offline-log.yml` into the test repository
and add a repository secret `RK_RUNNERS_READ_TOKEN`, a fine-grained token
for the repository with Administration: Read only. Every 6 hours it logs
each runner's status in its run summary.

Then run `check` on days 1, 13, 14, 15 and 16:

```bash
scripts/testkit/offline-watch.sh --repo you/rk-gate check
```

Each time, look in your inbox (search for "runner", "self-hosted",
"offline", "removed"), at <https://github.com/notifications> and at the
repository's **Settings > Actions > Runners** page, and note every warning:
date, channel and wording. On day 1, note whether the failed queued job
notified you and how.

**Result to record:** when GitHub removed the runner (between which two
checks), and every warning seen, or "none".

## V-2 smallest token

**Question.** What is the smallest token that can list a personal
repository's runners with `status`, `busy` and `version` filled in? This is
the token `runnerkit-watch` would ask users for.

What the REST schema says: the runner object has `version`, "only set if
the runner has connected to the service at least once"; listing runners
needs the fine-grained permission Administration: Read; the newer
`GET /repos/{owner}/{repo}/actions/runners/deprecations/{version}` (also
Administration: Read) returns the dates after which GitHub stops
registering, and stops sending jobs to, a runner version. The workflow
`GITHUB_TOKEN` has no Administration permission, so it is expected to get
403.

Create three more fine-grained tokens for the test repository only:
`admin-read` (Administration: Read), `actions-read` (Actions: Read) and
`metadata-only` (no permissions added). Optionally add the `admin-read`
token as the repository secret `RK_PROBE_PAT`, to see a PAT work inside
Actions. Then, with the runner online:

```bash
scripts/testkit/token-probe.sh --repo you/rk-gate --in-workflow env admin-read actions-read metadata-only
```

It asks for each token (hidden input; or set `RK_TOKEN_admin_read` and so
on) and prints a table: token, endpoint, HTTP status, the
`X-Accepted-GitHub-Permissions` header GitHub returns, and the runner
fields. `--in-workflow` dispatches `rk-token-probe.yml` on a GitHub-hosted
runner to test `GITHUB_TOKEN` (and `RK_PROBE_PAT`).

**Result to record:** the smallest token that returns all three fields,
whether `GITHUB_TOKEN` can (expected: no), and whether the deprecations
endpoint returns dates for 2.334.0.

## V-3 runner auto-update

**Question.** Does a runner that RunnerKit v1.3.3 installed at 2.334.0
update itself, given that RunnerKit never passes `--disableupdate`? If not,
every existing v1.3.3 runner falls out of support, and v1.3.4 needs an
advisory.

`autoupdate-probe.sh` registers a second runner, `runnerkit-autoupdate-probe`,
at 2.334.0 on the gate host, laid out exactly as RunnerKit lays out its
runners (owned by `runnerkit-runner`, configured through `su`, run by
`svc.sh`). It dispatches one job at it and checks what GitHub and the disk
show afterwards. Run it before the revocation drill: it needs `install.sh`'s
sudo.

```bash
scripts/testkit/autoupdate-probe.sh --repo you/rk-gate --host rkadmin@<ip>
```

| # | Check | What a FAIL means |
| --- | --- | --- |
| U1 | The probe registered at 2.334.0 | GitHub refused 2.334.0 or updated it on connect; record which |
| U2 | The probe job succeeded | Look at the job log |
| U3 | The runner now reports RunnerKit's pinned version (`RunnerVersion` in `internal/bootstrap/package.go`) or newer | v1.3.3 runners do not update themselves, or not far enough: advisory needed |
| U4 | `bin` and `externals` point at the new version, and the self-update log ends in `.succeed` | The update failed on disk; read the `_diag/SelfUpdate-*` log |
| U5 | `runsvc.sh` is not empty and matches `bin/runsvc.sh` | [actions/runner#4421](https://github.com/actions/runner/issues/4421) hits RunnerKit's layout |
| U6 | After `systemctl restart` the runner is online and its unit active | The updated runner does not survive a restart or reboot |
| U7 | The listener's `_diag/Runner_*.log` shows the update and `return code 3` (the runner's "exit for update" code) | The runner did not go through its own update path; read `diag-update.txt` |

The script removes the probe runner afterwards, also when a check fails.
To keep it for a closer look, run `install` and `run` separately and
`remove` later.

**Result to record:** the versions before and after, how long the update
took (U7's detail gives the download time from the listener's log and the
update script's time from `_diag/SelfUpdate-*.log`), and U1 to U7.

## V-4 hosted fallback

**Question.** Can build and test jobs move to GitHub-hosted runners by
changing one repository variable, and what happens to jobs already queued?
This is the mechanism behind the planned "fall back to GitHub-hosted when
your runner is down" recipe.

`rk-fallback.yml` runs on
`${{ fromJSON(vars.RUNS_ON || '["ubuntu-latest"]') }}`. With the runner
online:

```bash
scripts/testkit/fallback-probe.sh --repo you/rk-gate --host rkadmin@<ip> --flip-token
```

`--flip-token` asks for a fourth fine-grained token with only Variables:
Read and write, and uses it for the flip in F2, to confirm that is all a
"flip to hosted" automation needs.

| # | Check |
| --- | --- |
| F1 | `RUNS_ON` set to the RunnerKit labels sends the job to the RunnerKit runner |
| F2 | `RUNS_ON` set to `["ubuntu-latest"]` sends it to a GitHub-hosted runner |
| F3 | With no `RUNS_ON` the job runs on GitHub-hosted (the workflow's default) |
| F4 | With the runner service stopped, a job queued for the RunnerKit labels stays queued after the flip |
| F5 | Cancelling that run and re-running it sends the new attempt to GitHub-hosted |

F4 and F5 stop the runner service for a few minutes (`--skip-queued`
leaves them out). The script restores `RUNS_ON` and restarts the service
when it exits.

**Result to record:** F1 to F5, and the HTTP status of the flip with the
Variables-only token.

## Clean up

After the last V-1 check:

1. Delete the fine-grained tokens (Settings > Developer settings).
2. Delete the host, or `scripts/testkit/host-container.sh down`.
3. Delete the test repository, or remove its workflows and runners if you
   keep it for a later CI canary.
4. `rm -rf ~/runnerkit-testkit/state` (the scratch RunnerKit state). Keep
   the evidence directories until the results below are committed.

## Results

| Validation | Date | Result | Evidence | Decision it feeds |
| --- | --- | --- | --- | --- |
| V-1 docs | 2026-09-26 | GitHub documents no owner warning for offline, removed or outdated runners; only a failed-run notice for a job queued 24 hours | This page | Keep the Watch offline, removal and version rules unless the live test finds a warning |
| V-1 live | | | | Same |
| V-2 | | | | The token `runnerkit-watch` asks for |
| V-3 | | | | Advisory for v1.3.3 users; the Watch version rule |
| V-4 | | | | Recipe 3, fall back to GitHub-hosted |
