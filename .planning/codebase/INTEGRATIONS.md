# External Integrations

**Analysis Date:** 2026-05-17

RunnerKit integrates with two external services at runtime (GitHub REST API + Hetzner Cloud API), drives remote hosts via SSH/systemd, and ships through GitHub Releases + a Homebrew tap signed by Sigstore cosign. There is no application database, no webhook receiver, and no inbound network surface.

## APIs & External Services

### GitHub REST API

- Base URL: `https://api.github.com` (overridable via `ClientOptions.BaseURL`; used by tests). Set in `internal/github/client.go` line 67.
- Headers: `Accept: application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28` (`internal/github/client.go` lines 21-22, 149-150).
- Authorization: `Bearer <token>` when a token is configured (`client.go` line 155).
- Client: hand-rolled over `net/http` — no third-party SDK. `Client.do` (lines 141-214) handles request building, structured slog logging, and `APIError` unwrapping for non-2xx responses (with secret-safe message redaction via `redact.Redactor`).

**Endpoints called:**
- `GET /repos/{owner}/{name}` — repo metadata (`Client.Repository`, `client.go` lines 87-115).
- `POST /repos/{owner}/{name}/actions/runners/registration-token` (`Client.CreateRegistrationToken` lines 117-119, dispatched through `createRunnerToken`).
- `POST /repos/{owner}/{name}/actions/runners/remove-token` (`Client.CreateRemovalToken`).
- `GET /repos/{owner}/{name}/actions/runners` (`Client.ListRunners`, `internal/github/runners.go` lines 25-51).
- `DELETE /repos/{owner}/{name}/actions/runners/{id}` (`Client.DeleteRunner`, `runners.go` lines 53-55).
- `GET /repos/accidentally-awesome-labs/runnerkit/releases/latest` — lazy update-notice check (`internal/update/check.go` line 26), with `If-None-Match` ETag conditional GET and 24h on-disk cache at `<state-dir>/update-check.json` (`check.go` lines 22-28, 80-126).

**Authentication discovery** (`internal/github/auth.go` `DiscoverAuth`, lines 38-60):
1. Prefer the `gh` CLI (`exec.LookPath("gh")` then `gh auth token`).
2. Fall back to `RUNNERKIT_GITHUB_TOKEN` env var.
3. Otherwise return `"GitHub authentication not found"`.

Tokens must be fine-grained PATs scoped to the target repo with `Administration: read/write` and `Metadata: read` (remediation string built by `FineGrainedTokenRemediation` in `auth.go` lines 62-71). Every credential value is registered with the `redact.Redactor` on discovery and on token creation (`client.go` line 82, `client.go` line 137) so it never surfaces in logs.

### Hetzner Cloud API

- SDK: `github.com/hetznercloud/hcloud-go` v1.59.2 — wrapped behind a minimal `Client` interface in `internal/provider/hetzner/client.go` (lines 11-42) so unit tests can substitute fakes.
- Construction: `hcloud.NewClient(hcloud.WithToken(token))` (`client.go` line 49).

**Auth resolution** (`internal/provider/hetzner/credentials.go` `ResolveToken`, lines 26-43):
1. `HCLOUD_TOKEN` env var (preferred).
2. `HETZNER_CLOUD_TOKEN` env var (alternate).
3. Otherwise return `MissingTokenError` with remediation pointing at the Hetzner Cloud Console.

Token must be project-scoped read+write (per `.env.example` comment: "project-scoped, NOT account-scoped").

**Resources created** (`internal/provider/hetzner/provision.go` `Provision`, lines 97-187):
- `Location` / `ServerType` / `Image` lookups (`client.GetLocation`, `GetServerType`, `GetImage`).
- `SSHKey` (`client.CreateSSHKey`) — uploaded per provision so cloud-init can authorize the operator.
- `Firewall` with one ingress rule on TCP/22 from `--ssh-allowed-cidr` (default in `provider.HetznerDefaultSSHAllowedCIDR`; rule built by `firewallRules` in `provision.go` lines 334-342).
- `Server` with `EnableIPv4: true`, `EnableIPv6: true`, auto-allocated primary IPs (Bug 26 / Plan 06-11 — IPs carry `AutoDelete: true` so destroy cascades cleanly; see comment block at `provision.go` lines 144-153).
- The full create call:
  ```go
  client.CreateServer(ctx, hcloud.ServerCreateOpts{
      Name: plan.ResourceNames["server"], ServerType: serverType, Image: image,
      SSHKeys: []*hcloud.SSHKey{sshKey}, Location: location,
      UserData: cloudInitUserData(profile.SSHUser, input.PublicKey, input.ExtraPackages),
      Firewalls: []*hcloud.ServerCreateFirewall{{Firewall: *firewall}},
      PublicNet: &hcloud.ServerCreatePublicNet{EnableIPv4: true, EnableIPv6: true},
      ...
  })
  ```
  (`provision.go` lines 154-167).

**Destroy ordering** (`internal/provider/hetzner/destroy.go` — referenced from `client.go` comments lines 28-41): firewall must be detached from the server before deletion (`DetachFirewallFromServer`), and primary IPs auto-cascade with the server.

**Cloud-init contract** (`internal/provider/hetzner/provision.go` `cloudInitUserData`, lines 351-405):
- Writes a `#cloud-config` document with a `users:` block creating `runnerkit-admin` (default SSH user) with the operator's public key, an `apt:` block, the `BaselinePackages` set + any auto-detected workflow packages, and a `runcmd:` block that runs `visudo` on a staged sudoers file before atomically installing it to `/etc/sudoers.d/runnerkit-installer`.
- Marker written to `/var/lib/runnerkit/cloud-init.json` so RunnerKit can read it back during readiness.
- Version string: `runnerkit-cloud-init-v3` (`provision.go` line 23 — `CloudInitUserDataVersion`).
- Readiness waits on `cloud-init status --wait` and refuses on `status: error` (see `CLAUDE.md` "Hetzner cloud provisioning" section).

## Data Storage

**Databases:**
- None. RunnerKit has no database.

**File Storage (local):**
- State directory resolution order (`internal/state/store.go` `DefaultBaseDir`, lines 31-45):
  1. `RUNNERKIT_STATE_DIR`
  2. `$XDG_STATE_HOME/runnerkit`
  3. `$HOME/.local/state/runnerkit`
  4. `os.UserConfigDir()/runnerkit/state`
  5. `os.TempDir()/runnerkit`
- Files inside the state dir:
  - `state.json` — repository inventory, provider refs, runner metadata, cloud cost profile (loaded by `Store.Load`, `store.go` line 52).
  - `update-check.json` — cached GitHub Releases latest (`internal/update/check.go` line 24, atomic tmp+rename write at lines 162-169).
  - `sessions/` — BYO checklist progress (SEED-004).
- Remote per-host filesystem layout (driven by RunnerKit):
  - `/opt/actions-runner/runnerkit-<owner>-<repo>-local/` — per-repo runner install dir.
  - `/opt/actions-runner/runnerkit-shared-bin/<runner-version>/` — shared runner tarball cache (`internal/bootstrap/install.go` line 202 `SharedRunnerCacheRoot`).
  - `/var/lib/runnerkit/` — host state, finalizer dirs, ephemeral log archives.
  - `/etc/sudoers.d/runnerkit-installer` — scoped NOPASSWD entry (`internal/bootstrap/sudoers.go` line 15 `SudoersFilePath`).
  - `/etc/systemd/system/runnerkit-ephemeral.<runner>.{service,ttl.service,ttl.timer}` — ephemeral unit files.

**Caching:**
- 24h on-disk cache for the GitHub Releases update-notice (`internal/update/check.go` line 25 `cacheTTL`).
- Shared runner tarball cache on the remote host (one tarball per `RunnerVersion` reused across all per-repo installs on the same BYO host — SEED-002 / multi-repo).
- No in-process LRU or in-memory caches.

## Authentication & Identity

**GitHub:** See "GitHub REST API → Authentication discovery" above. RunnerKit never acquires tokens itself — it relies on `gh auth login` or operator-provided PATs.

**Hetzner:** See "Hetzner Cloud API → Auth resolution" above.

**Remote SSH:**
- Connection: forks `ssh` from `os/exec` (`internal/remote/system.go` `sshArgs`, lines 206-226). Operator's local SSH config and `ssh-agent` are honored.
- Flags pinned: `BatchMode=yes`, `ConnectTimeout=10`, `StrictHostKeyChecking=no`, `UserKnownHostsFile=/dev/null` (Plan 06-16 Bug 34 — RunnerKit persists its own host-key fingerprint in `state.json` and rejects mismatches, so ambient `known_hosts` would only cause false negatives on IP-recycle).
- Optional explicit key via `target.KeyPath` → `-i <path>`.
- Host-key discovery: `ssh-keyscan -p <port> -T 5 <host>` via `scanHostKey` (`system.go` lines 94-115). The selected line is deterministic (ed25519 → ecdsa → rsa precedence in `selectHostKeyLine`, lines 137-174) so two scans of the same host produce a byte-stable SHA256 fingerprint.

**Sudo:** Two paths, both rooted in `/etc/sudoers.d/runnerkit-installer`:
- **Path C** (recommended): `install.sh` or `runnerkit byo-prepare` writes a scoped NOPASSWD entry granting the SSH user only the commands the bootstrap actually invokes (`internal/bootstrap/sudoers.go` `RenderSudoersEntry`, lines 59-77). The entry is validated with `visudo -cf` on a `mktemp`-created staging file before being atomically moved into place (`RemoteVisudoCheckScript`, lines 97-111). Hetzner cloud-init applies the identical template.
- **Path B** (fallback): operator-provided sudo password threaded through `RUNNERKIT_SUDO_PASSWORD`. The literal value is registered with `redact.SudoPassword` so it never lands in logs or JSON output.

## Monitoring & Observability

**Error tracking:**
- None (no Sentry/Datadog/Rollbar integration).

**Logs:**
- Local: structured `log/slog` JSON to stderr (or `RUNNERKIT_LOG_DEST` — `stdout` or `file:/path`). Configured in `internal/rklog/rklog.go` `NewFromEnv` (lines 25-41). Default level when `RUNNERKIT_LOG` is unset: discarded.
- Sample event from `internal/github/client.go` lines 193-202:
  ```go
  c.log.InfoContext(ctx, "github.api",
      slog.String("method", method), slog.String("path", apiPath),
      slog.Int("status", resp.StatusCode), slog.Duration("duration", time.Since(start)),
      slog.Bool("ok", ok), slog.Int("response_bytes", responseLen))
  ```
- All log writers pipe through `redact.Redactor` (`internal/redact/redact.go`) which masks both registered values and pattern-matched secrets (GitHub PATs `gh[pousr]_…` / `github_pat_…`, runner registration/removal tokens, PEM private keys, `HCLOUD_TOKEN=…` env fragments — patterns at `redact.go` lines 48-55).

**Remote diagnostics:**
- `journalctl -u <service>` collection bounded by line count, captured by `internal/ops/logs.go` (`CollectLogs`, `CollectBoundedJournalsForHints` — referenced from `CLAUDE.md` Phase 7).
- Heuristic OOM/SIGKILL hint detection at `internal/ops/hostkillhint.go`, surfaced as `host_incident_hints` in `doctor --json` output (`internal/cli/doctor.go`).

## CI/CD & Deployment

**Hosting:**
- Source: GitHub `accidentally-awesome-labs/runnerkit`.
- Binaries: GitHub Releases on the same repo.
- Homebrew tap: GitHub `accidentally-awesome-labs/homebrew-tap` (`.goreleaser.yaml` lines 54-67), updated by GoReleaser on every release tag.

**CI Pipeline:**
- `.github/workflows/pr-checks.yml` (PR + push to main): checkout, `setup-go@v5` with Go 1.22, `cosign-installer@v3`, `goreleaser-action@v7` install-only at v2.15.4 → `goreleaser check`, `goreleaser release --snapshot --skip=publish --clean --skip=sign`, asserts the dist matrix (`runnerkit_<ver>_<os>_<arch>.tar.gz` for darwin/linux × amd64/arm64 plus a checksums file). Separate `go-test` job runs `go test ./... -count=1 -race`.
- `.github/workflows/release.yml` (push of tag `v*`): same setup → `goreleaser release --clean` with secrets exported (`GITHUB_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`, `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`). `permissions: contents: write, id-token: write` (the `id-token` permission is what enables keyless cosign OIDC signing).

**Release signing (Sigstore cosign):**
- `.goreleaser.yaml` lines 29-37:
  ```yaml
  signs:
    - cmd: cosign
      signature: '${artifact}.sigstore.json'
      args: [sign-blob, '--bundle=${signature}', '${artifact}', '--yes']
      artifacts: checksum
  ```
- Only the checksums file is signed (keyless OIDC via GitHub Actions `id-token`); individual archives are verified transitively via the checksum.

**macOS notarization (optional, gated):**
- `.goreleaser.yaml` lines 39-52 — only runs when all of `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID` are set in the env (boolean `and`/`isEnvSet` template guards).

**Live smokes (maintainer-only, NOT in CI):**
- `Makefile` lines 32-58 — `smoke-live-byo`, `smoke-live-cloud`, `smoke-stopwatch`, composed into `smoke-live`.
- Drivers: `scripts/smoke/byo-permission.sh` (registers a real runner on a real BYO host then tears it down), `scripts/smoke/cloud-end-to-end.sh` (creates billable Hetzner resources behind a trap that calls `runnerkit destroy --yes` on Ctrl-C), `scripts/smoke/hetzner-empty-precheck.sh` + `hetzner-destroy-verify.sh`, `scripts/smoke/assert-doctor-json-contract.sh` + `assert-list-json-contract.sh` + `assert-list-host-repo-count.sh`.
- Secrets sourced from local `.env` (gitignored): `HCLOUD_TOKEN`, `RUNNERKIT_SMOKE_REPO`, `RUNNERKIT_SMOKE_BYO_HOST`, optional `RUNNERKIT_SMOKE_REPO2`, `RUNNERKIT_SMOKE_MULTI_REPO`.

## Environment Configuration

**Required for end users (depending on path):**
- BYO: a `gh auth login`-authenticated workstation OR `RUNNERKIT_GITHUB_TOKEN`. SSH access to the target host.
- Cloud: `HCLOUD_TOKEN` (or `HETZNER_CLOUD_TOKEN`) plus the GitHub credential above.

**Required for maintainer releases:**
- GitHub repo secrets: `GITHUB_TOKEN` (built-in), `HOMEBREW_TAP_GITHUB_TOKEN` (fine-grained PAT scoped to `accidentally-awesome-labs/homebrew-tap` with `Contents: read+write` — see `docs/release-process.md` lines 38-58).
- Optional: the `MACOS_*` notarization secrets.

**Secrets location:**
- Local `.env` (gitignored, `.gitignore` lines 15-17).
- GitHub Actions repository secrets (referenced in `.github/workflows/release.yml` lines 26-32).
- Hetzner Cloud Console (project API tokens) — RunnerKit reads them from env only.
- GitHub fine-grained PATs — RunnerKit reads them from `gh auth token` or env only.

## Webhooks & Callbacks

**Incoming:**
- None. RunnerKit is a CLI; it opens no listening sockets.

**Outgoing:**
- None. The only callback-like behavior is `client.WaitForAction` in the Hetzner SDK (`internal/provider/hetzner/client.go` lines 82-87), which polls the action endpoint internally; RunnerKit does not expose a callback URL.

## Systemd Integration

RunnerKit drives systemd entirely through `sudo` + bash on the remote host (no D-Bus, no `dbus-go` dependency):

- Persistent runner units installed via the upstream `./svc.sh` helper that ships with `actions/runner` (`internal/bootstrap/script.go` `RenderServiceScript`, lines 60-75 — `svc.sh stop`, `uninstall`, `install runnerkit-runner`, `start`, `status`).
- Ephemeral runner units written directly to `/etc/systemd/system/runnerkit-ephemeral.<runner>.service` with `Type=simple`, `ExecStart=<install>/run.sh`, `ExecStopPost=<finalizer> completed`, `Restart=no`, `KillMode=process` (`internal/bootstrap/script.go` `RenderEphemeralServiceScript`, lines 166-197).
- TTL safeguard timer at `/etc/systemd/system/runnerkit-ephemeral.<runner>.ttl.timer` with `OnActiveSec=24h` and a paired oneshot unit that stops the ephemeral runner + runs the finalizer (`RenderEphemeralTTLTimerScript`, lines 203-238).
- Service health probed via `systemctl is-active`, `systemctl status --no-pager`, and `test -d /run/systemd/system && command -v systemctl` (`internal/remote/system.go` line 32).
- Journal collection: `journalctl -u <service> -n 500 --no-pager` from `RenderEphemeralFinalizerScript` (`script.go` line 145) and `CollectLogs` (`internal/ops/logs.go`).

---

*Integration audit: 2026-05-17*
