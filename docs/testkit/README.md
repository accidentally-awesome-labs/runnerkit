# Release gate and validation kit (maintainer-only)

This kit runs the v1.3.4 real-job release gate and the four Stage 0
validations against real GitHub. End users never need it.

| Part | Question it answers | Runbook | Script | Time |
| --- | --- | --- | --- | --- |
| Real-job gate (A-20) | Does a real GitHub job run on a RunnerKit runner, on a fresh password-sudo host prepared only by the candidate `install.sh`? No release may say BYO works without it. | [release-gate.md](release-gate.md) | `gate.sh` | about 1 hour |
| Revocation drill | Do the revocation steps in [security-posture.md](../security-posture.md#if-you-already-installed-runnerkit) work as written? | [release-gate.md](release-gate.md#part-3-revocation-drill) | `gate.sh --after-revocation` | 15 minutes |
| V-1 offline runner | Does GitHub warn you before, or when, it removes a runner that stayed offline? | [validations.md](validations.md#v-1-offline-runner) | `offline-watch.sh` | 5 minutes, then 15 minutes on day 15 |
| V-2 smallest token | Which token can read runner status, busy and version for a personal repository? | [validations.md](validations.md#v-2-smallest-token) | `token-probe.sh` | 15 minutes |
| V-3 runner auto-update | Does a runner installed at 2.334.0 (RunnerKit v1.3.3's pin) update itself? | [validations.md](validations.md#v-3-runner-auto-update) | `autoupdate-probe.sh` | 30 minutes |
| V-4 hosted fallback | Can a repository variable in `runs-on` move jobs to GitHub-hosted runners? | [validations.md](validations.md#v-4-hosted-fallback) | `fallback-probe.sh` | 20 minutes |

The scripts are in [`scripts/testkit/`](../../scripts/testkit/) and the
workflows they dispatch are in
[`scripts/testkit/workflows/`](../../scripts/testkit/workflows/).

## What you need

- A workstation (macOS or Linux) with a RunnerKit checkout, Go 1.26, `bash`,
  `curl`, `jq`, `ssh` and `git`.
- A fresh Ubuntu 24.04 x86_64 host that nothing else uses: a small cloud VM
  (2 vCPU, 4 GB RAM or more, 20 GB disk), or on a Linux x86_64 workstation
  with Docker, a local container from `scripts/testkit/host-container.sh`.
  Details are in [release-gate.md](release-gate.md#part-1-prepare).
- A throwaway **private** repository under your personal account, for
  example `you/rk-gate`, holding the kit's workflows.
- A fine-grained personal access token for that repository only, with
  Administration, Actions and Variables set to Read and write, and an
  expiry of 30 days (it has to last through V-1). V-2 asks for three more
  narrow tokens.

## Order

One session of about 2.5 hours, then 15 minutes on day 15.

1. Prepare the candidate build, the repository, the token and the host
   ([release-gate.md, part 1](release-gate.md#part-1-prepare)).
2. Run the gate ([part 2](release-gate.md#part-2-run-the-gate)).
3. While the runner is online and `install.sh`'s sudo is still in place:
   V-2, V-4 and V-3 ([validations.md](validations.md)).
4. Revocation drill ([part 3](release-gate.md#part-3-revocation-drill)).
5. Start V-1 and stop the runner. Stop by 2026-10-12, so the day-15 check
   lands by 2026-10-27.
6. On day 15, check V-1 and clean up
   ([validations.md](validations.md#clean-up)).

## Safety rules

- Use a private repository only. The scripts refuse a public one.
- The workflows run only when dispatched (one is also scheduled on a
  GitHub-hosted runner), never on push or pull requests.
- Nothing in the kit passes `--disableupdate`: runners must keep updating
  themselves.
- Tokens never go on a command line. The scripts read them from `GH_TOKEN`,
  from `RK_TOKEN_<label>` or from a hidden prompt, and pass them to `curl`
  through a temporary file only you can read. Delete the tokens when you
  are done.
- Any workflow in the test repository can get root on the host (see
  [security posture](../security-posture.md)). Keep secrets off the host
  and delete it afterwards.

## Evidence

Every script writes its results to
`~/runnerkit-testkit/<owner>-<name>/` (set `RK_TESTKIT_OUT` to change the
base directory). The files contain no tokens. Each script prints a Markdown
summary with a PASS or FAIL line per check.

- Gate: `EVIDENCE.md` ends with the line to paste into the v1.3.4 section of
  [CHANGELOG.md](../../CHANGELOG.md), as the
  [pre-tag checklist](../release-process.md#pre-tag-checklist) asks.
- Validations: record the outcome in the results table at the end of
  [validations.md](validations.md#results) and commit it.

## Testing the scripts

The scripts talk to `GITHUB_API_URL` (default `https://api.github.com`),
so you can point them at a fake API. `go test ./...` checks that they
parse, that the workflows only run when dispatched (or on a schedule, on a
GitHub-hosted runner), and that nothing passes `--disableupdate`.
