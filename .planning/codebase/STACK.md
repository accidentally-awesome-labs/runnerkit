# Technology Stack

**Analysis Date:** 2026-05-17

RunnerKit is a single-binary Go CLI that prepares and manages GitHub Actions self-hosted runners on BYO SSH hosts or freshly-provisioned Hetzner Cloud VMs. The entire toolchain is Go + POSIX shell — no JS/Python runtime, no database, no web server.

## Languages

**Primary:**
- Go 1.22 — entire CLI binary, all production logic under `internal/` and `cmd/`. Declared in `go.mod` line 3.

**Secondary:**
- Bash (POSIX, `set -euo pipefail`) — host install (`install.sh`), maintainer live-smoke scripts (`scripts/smoke/*.sh`), and remote bootstrap scripts rendered as Go string templates in `internal/bootstrap/script.go` and `internal/bootstrap/image_setup.go`.
- Python 3 — required only by the smoke harness for JSON-contract assertions (`scripts/smoke/assert-doctor-json-contract.sh`, `scripts/smoke/assert-list-json-contract.sh`). Not a runtime dependency of the shipped binary.
- YAML — GitHub Actions workflows (`.github/workflows/`), GoReleaser config (`.goreleaser.yaml`), and the cloud-init user-data document rendered by `cloudInitUserData` in `internal/provider/hetzner/provision.go`.

## Runtime

**Environment:**
- macOS (darwin) and Linux on amd64 and arm64 — release matrix declared in `.goreleaser.yaml` lines 13-14 (`goos: [darwin, linux]`, `goarch: [amd64, arm64]`).
- CGO disabled (`CGO_ENABLED=0`, `.goreleaser.yaml` line 12) — static binaries.
- ldflags strip symbol/debug info and inject the release tag: `-s -w -X main.version={{.Version}}` (`.goreleaser.yaml` line 16, consumed by `var version = "dev"` in `cmd/runnerkit/main.go` line 12).

**Package Manager:**
- Go modules (`go.mod`, `go.sum`).
- Lockfile: `go.sum` (5 KB, present and committed).

## Frameworks

**Core:**
- `github.com/spf13/cobra` v1.10.1 — CLI framework. Root command + 14 subcommands assembled in `internal/cli/root.go` (lines 115-179: `newVersionCommand`, `newInitCommand`, `newRegisterCommand`, `newUpCommand`, `newListCommand`, `newStatusCommand`, `newLogsCommand`, `newDoctorCommand`, `newRecoverCommand`, `newDownCommand`, `newDestroyCommand`, `newStateCommand`, `newUpgradeCommand`, `newUpgradeRunnerCommand`).

**Testing:**
- Standard library `testing` only. No third-party test framework.
- `-race` enabled in CI and local (`Makefile` line 13: `test-race`).
- Integration tests gated by `RUNNERKIT_INTEGRATION=1` with the `integration` build tag (`Makefile` line 16); they exercise real `sudo`/`useradd` on the host.

**Build/Dev:**
- `goreleaser` v2.15.4 — full release matrix, archive packaging, cosign signing, macOS notarization, and Homebrew cask publishing. Pinned in both workflow files (`.github/workflows/release.yml` line 23 and `pr-checks.yml` line 24).
- `make` — entrypoint for tests, smokes, snapshot builds (`Makefile`).
- `cosign` (via `sigstore/cosign-installer@v3`) — keyless OIDC signing of release checksums.

## Key Dependencies

**Critical (direct, declared in `go.mod` lines 5-10):**
- `github.com/spf13/cobra` v1.10.1 — CLI framework.
- `github.com/hetznercloud/hcloud-go` v1.59.2 — Hetzner Cloud SDK. Used in `internal/provider/hetzner/client.go` (wrapper interface + `APIClient` over `hcloud.NewClient(hcloud.WithToken(token))`).
- `github.com/hashicorp/go-version` v1.9.0 — SemVer comparison for the lazy CLI update notice (`internal/update/version.go`: `IsNewer(current, latest)` returns `lat.GreaterThan(cur)`).
- `golang.org/x/term` v0.10.0 — TTY-safe password prompts (`internal/ui/cli_prompter.go` line 12 — uses `term.ReadPassword` for the Path B sudo-password fallback).

**Indirect (transitive, `go.mod` lines 12-27):**
- Prometheus client libraries (`prometheus/client_golang` v1.16.0, `client_model`, `common`, `procfs`) — pulled in transitively by `hcloud-go` for metrics; RunnerKit does not expose `/metrics`.
- `golang/protobuf`, `google.golang.org/protobuf`, `golang.org/x/net`, `golang.org/x/sys`, `golang.org/x/text` — transitive HTTP/protobuf plumbing.
- `spf13/pflag`, `inconshreveable/mousetrap` — cobra deps.
- `cespare/xxhash/v2`, `beorn7/perks`, `matttproud/golang_protobuf_extensions` — prometheus deps.

**Infrastructure (standard library — no third-party for these):**
- `net/http` — GitHub API client (`internal/github/client.go`).
- `log/slog` — structured logging (`internal/rklog/rklog.go` configures JSON handler from `RUNNERKIT_LOG`/`RUNNERKIT_LOG_DEST`).
- `os/exec` — local `ssh`/`ssh-keyscan`/`gh` invocation (`internal/remote/system.go`, `internal/github/auth.go`).
- `encoding/json` — all state files, API payloads, and `--json` CLI output.

## Configuration

**Environment variables consumed by the binary** (grep `RUNNERKIT_` across `internal/`, plus Hetzner/CI conventions):
- `RUNNERKIT_GITHUB_TOKEN` — fallback GitHub PAT when `gh auth token` is unavailable (`internal/github/auth.go` line 55).
- `RUNNERKIT_LOG` — slog level: `debug|info|warn|error|off` (`internal/rklog/rklog.go` line 17).
- `RUNNERKIT_LOG_DEST` — `stderr|stdout|file:/path` (`internal/rklog/rklog.go` line 18).
- `RUNNERKIT_STATE_DIR` — overrides XDG/`~/.local/state/runnerkit` (`internal/state/store.go` line 32).
- `RUNNERKIT_NO_UPDATE_NOTIFIER` — silences the update check (`internal/update/check.go` line 56).
- `RUNNERKIT_PREFLIGHT_MEM_WARN_BYTES` — preflight `MemAvailable` threshold override (see `CLAUDE.md` Phase 7 section).
- `RUNNERKIT_CLOUD_INIT_TIMEOUT`, `RUNNERKIT_DESTROY_PRIMARY_IP_TIMEOUT` — operational timeouts.
- `RUNNERKIT_REGISTRATION_TOKEN`, `RUNNERKIT_REMOVAL_TOKEN`, `RUNNERKIT_SUDO_PASSWORD`, `RUNNERKIT_SUDOERS_CONTENT`, `RUNNERKIT_CI_SUDOERS_CONTENT` — internally set by the CLI when invoking remote bash scripts; not for end-user export.
- `RUNNERKIT_INTEGRATION` — gates the `integration` build-tag tests (`Makefile` line 16).
- `RUNNERKIT_DOCS_BASE` — docs URL base override.
- `HCLOUD_TOKEN` or `HETZNER_CLOUD_TOKEN` — Hetzner project-scoped API token (`internal/provider/hetzner/credentials.go` lines 9-10).
- `CI`, `NO_COLOR`, `CLICOLOR`, `TERM`, `XDG_STATE_HOME`, `SUDO_USER`, `TMPDIR` — standard conventions honored by the CLI.

**Build:**
- `go.mod` / `go.sum` — module + checksum lockfile.
- `.goreleaser.yaml` — release matrix, signing, notarization, Homebrew cask, archive naming (`{{.ProjectName}}_{{.Version}}_{{.Os}}_{{.Arch}}.tar.gz`).
- `Makefile` — `test`, `test-race`, `test-integration`, `vet`, `release-snapshot`, and the maintainer-only live smokes.
- `.github/workflows/pr-checks.yml` — PR/main CI: `goreleaser check`, snapshot build with dist assertion, `go test ./... -count=1 -race`.
- `.github/workflows/release.yml` — tag-triggered release pipeline.

**Local-only files (gitignored):**
- `.env` / `.env.local` — maintainer smoke secrets (`HCLOUD_TOKEN`, `RUNNERKIT_SMOKE_REPO`, `RUNNERKIT_SMOKE_BYO_HOST`). `.env.example` is the public template.
- `dist/`, `bin/`, `.smoke-state/`, `*.smoke.log`, `sessions/` — build/runtime artifacts.

**Notably absent:**
- No `.golangci.yml`, no `.prettierrc`, no `.editorconfig`. Linting relies on `go vet` (`Makefile` line 19) and `goreleaser check` in CI. No third-party linter configured.
- No `.nvmrc`, `requirements.txt`, `Cargo.toml`, `pyproject.toml`, `Dockerfile`, or `docker-compose.yml`.

## Platform Requirements

**Development (maintainer workstation):**
- Go 1.22+.
- `goreleaser` v2.15.4 (for `make release-snapshot`).
- `gh` CLI (`Makefile` line 41 — required for live smokes).
- `python3` (`Makefile` line 43 — required for JSON-contract assertion scripts).
- `cosign` (for tag releases, installed in CI via the GitHub Action).
- `ssh`, `ssh-keyscan` from OpenSSH client (used at runtime by `internal/remote/system.go`).

**Production (target hosts that RunnerKit drives):**
- BYO host: Linux with `sudo`, `bash`, `apt-get` / `dnf` / `yum`, `systemd`, `curl`, `tar`, `gzip`, `sha256sum`, `id`, `useradd`, `install`, `timedatectl` (probed by `SystemExecutor.Probe` in `internal/remote/system.go` lines 35-39).
- Cloud host: Ubuntu 24.04 LTS (Hetzner image `ubuntu-24.04`, default in `internal/provider/hetzner/provision.go`). cloud-init bootstraps from the user-data document built by `cloudInitUserData`.
- Runner package: Linux x64 or arm64, GitHub `actions/runner` v2.334.0 (`internal/bootstrap/package.go` lines 5, 19-35) — downloaded with pinned SHA256 from `https://github.com/actions/runner/releases/download/v2.334.0/...`.

**Distribution channels:**
- GitHub Releases (tarballs `runnerkit_<ver>_<os>_<arch>.tar.gz` + `runnerkit_<ver>_checksums.txt` + `install.sh`).
- Homebrew cask `accidentally-awesome-labs/homebrew-tap` → `brew install --cask runnerkit` (auto-updated by GoReleaser on each tag).

---

*Stack analysis: 2026-05-17*
