# Maintainer notes

## Releases

Installable binaries and the Homebrew cask update **only** when an annotated
`v*` tag is pushed on the upstream repository
(`accidentally-awesome-labs/runnerkit`). Merging to `main` does not release
anything. The tag triggers `.github/workflows/release.yml`, which runs
`go vet`, a `go generate` drift check and `go test -race`, and only then
GoReleaser (upstream only).

- Full pipeline, secrets, pre-tag checklist and verification:
  [release-process.md](release-process.md).
- Short checklist for agents: [CLAUDE.md](../CLAUDE.md).
- Record every release in [CHANGELOG.md](../CHANGELOG.md), including a
  Known-issues block that matches the README.

## Rules that gate a release

- Never tag a release whose notes say the BYO path works unless a real GitHub
  job ran on a fresh password-sudo host prepared only by that release's
  `install.sh`.
- `bootstrap.RenderSudoersEntry` gains no entries; `install.sh`'s sudoers
  block is generated from it (`go generate ./...`).
- Never pass `--disableupdate` to the runner's `config.sh`.
- No feature without 2 distinct external requests (see
  [CONTRIBUTING.md](../CONTRIBUTING.md)).

## Archiving

The [archive playbook](maintainers/archive.md) is the checklist for
shutting RunnerKit down. It is used only on a kill decision or a failed
capacity check; the triggers are listed in the playbook, and the
2026-12-21 thresholds are in [validation-metrics.md](validation-metrics.md).
