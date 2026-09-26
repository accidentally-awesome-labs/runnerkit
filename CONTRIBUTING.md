# Contributing to RunnerKit

RunnerKit is experimental and maintained on a capped-hours basis until a
published go/kill decision on 2026-12-21. Small, focused changes are the
easiest to review. Bug reports with exact commands, output and versions
(`runnerkit --version`) are the most useful contribution.

## Scope rule

**No new feature is built without 2 distinct external requests** (two
different people, not the maintainer, asking for it in an issue or
Discussion). Bug fixes, security fixes, documentation corrections and test
coverage for existing behavior do not need requests.

The one exemption is the Stage 1 validation work, capped at 30 build hours
in total:

- **W1** `runnerkit-watch`, a scheduled GitHub Action that watches runners;
- **W2** `runnerkit checkup`, a read-only host report (built only if
  requested);
- **W3** runner hygiene recipes.

The project is also explicitly **not** doing, unless the evidence changes:

- headline claims about setup time or about saving money compared with
  GitHub-hosted runners;
- adding entries to the installer sudoers allowlist (`RenderSudoersEntry`);
- recommending the Hetzner cloud path, or printing any price that was not
  fetched from the Hetzner API at plan time;
- passing `--disableupdate` to the GitHub runner, anywhere;
- arm64, non-Ubuntu, macOS or Windows host support;
- autoscaling, agent pools or scale sets;
- default-on telemetry.

If you want to work on something larger, open an issue first so we can
check it against these rules before you spend time on it.

## Developer Certificate of Origin (DCO)

RunnerKit uses the [Developer Certificate of Origin](https://developercertificate.org/)
instead of a CLA. Every commit must carry a `Signed-off-by` line with your
real name and an email you can be reached at:

```bash
git commit -s -m "fix(cli): describe the change"
```

This adds:

```
Signed-off-by: Your Name <you@example.com>
```

By signing off you certify that you wrote the change or otherwise have the
right to submit it under the project's license
([Apache-2.0](LICENSE)). To sign off commits you already made:

```bash
git rebase --signoff main
```

The `dco` check in `pr-checks` fails a pull request that has a commit
without the line. Pull requests opened by the repository's owner, members
or collaborators are not checked.

## Building and testing

RunnerKit targets **Go 1.26** (see `go.mod`); CI and releases use the latest
Go 1.26.x patch release. `go.mod` also has `toolchain go1.26.8`, so if your
installed Go is older than that, the `go` command (`GOTOOLCHAIN=auto`, the
default) downloads at least go1.26.8 rather than an unpatched go1.26.0. To
build with the exact patch CI used, set it explicitly (replace `go1.26.8`
with the current 1.26.x patch):

```bash
GOTOOLCHAIN=go1.26.8 go build ./...
GOTOOLCHAIN=go1.26.8 go vet ./...
GOTOOLCHAIN=go1.26.8 go test ./... -count=1
```

Or use the Makefile:

```bash
make test            # go test ./...
make test-race       # go test -race ./...
make vet
make generate        # go generate ./...
make generate-check  # same check as CI; run on a clean tree (any diff or untracked file fails)
make vulncheck       # govulncheck (report-only in CI)
```

`install.sh`'s sudoers block is generated from `bootstrap.RenderSudoersEntry`.
After changing `internal/bootstrap/sudoers.go`, run `go generate ./...` and
commit the updated `install.sh`; `make generate-check` and CI fail otherwise.
Do not edit the block between `# BEGIN runnerkit-sudoers` and
`# END runnerkit-sudoers` by hand.

Tests must not call real GitHub or Hetzner APIs and must not write into the
current working directory; use the fakes in `internal/testsupport` and an
isolated `RUNNERKIT_STATE_DIR`. Live smokes (`make smoke-live*`) create real
runners and billable resources and are maintainer-only.

## Pull requests

- One logical change per pull request, with a test that fails without it.
- Add a line to the `Unreleased` section of [CHANGELOG.md](CHANGELOG.md) for
  any user-visible change, and update the docs it affects.
- Do not claim in docs or release notes that the BYO path works unless a
  real GitHub job ran on a fresh password-sudo host prepared only by
  `install.sh`.
- Security issues go through [SECURITY.md](SECURITY.md), not public issues.
