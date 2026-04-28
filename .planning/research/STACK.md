# Stack Research

**Domain:** Developer CLI for provisioning and managing GitHub Actions self-hosted runners
**Researched:** 2026-04-28
**Confidence:** MEDIUM-HIGH

## Recommended Stack

### Core Technologies

| Technology                                                        | Version                                                      | Purpose                                                                         | Why Recommended                                                                                                                                                                                        |
| ----------------------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Go                                                                | 1.25.x                                                       | Single-binary CLI, SSH/provisioning orchestration, GitHub/cloud API integration | Best fit for a cross-platform developer tool that must ship as one binary, run quickly, manage processes, use SSH, and distribute through package managers without Node/Python runtime dependencies.   |
| Cobra                                                             | v1.10.2                                                      | CLI command framework                                                           | Mature standard for Go CLIs; latest documented release is v1.10.2 (Dec 2025). Good for `runnerkit init`, `status`, `doctor`, `logs`, `destroy`, and provider subcommands.                              |
| google/go-github                                                  | v76.0.0                                                      | GitHub REST API client                                                          | Avoid hand-rolling API calls for runner listing/removal and repository/org metadata. Current release found: v76.0.0. Use direct REST calls only where the client lags newer runner endpoints.          |
| GitHub REST API                                                   | 2026-03-10 API version                                       | Self-hosted runner registration/removal tokens, runner list/delete, labels      | Official API supports creating short-lived registration/remove tokens and managing repository/organization runners. RunnerKit should request tokens just-in-time and never persist them.               |
| OpenSSH over `golang.org/x/crypto/ssh` plus system `ssh` fallback | Go module latest                                             | BYO machine install, remote command execution, file upload, service inspection  | BYO SSH is the lowest-friction path for solo developers who already have a VPS/homelab machine. A system `ssh` fallback preserves user agent/config behavior.                                          |
| systemd                                                           | Linux distro native                                          | Runner service management on Linux hosts                                        | GitHub runner service install flows are service-oriented; Linux/systemd-first keeps v1 reliable and understandable. macOS/Windows runners should be deferred unless explicitly scoped.                 |
| Hetzner Cloud                                                     | hcloud API/CLI ecosystem v1.62.x; Terraform provider v1.61.x | Default low-cost cloud provisioning candidate                                   | Strong cost story for solo developers, mature Go/CLI ecosystem, simple VPS model, and good fit for “cheap self-hosted runner” positioning. Validate region/support expectations during implementation. |

### Supporting Libraries

| Library                                     | Version              | Purpose                                    | When to Use                                                                                                                                       |
| ------------------------------------------- | -------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Viper or koanf                              | latest stable        | Config loading from env/files/flags        | Use for user config (`~/.config/runnerkit/config.yaml`) and optional project config. Prefer koanf if you want less global state than Viper.       |
| Bubble Tea                                  | v2.0.6               | Optional guided terminal UX                | Use only for an interactive setup wizard if plain prompts feel insufficient. Do not make TUI required for automation.                             |
| survey or huh                               | latest stable        | Simple prompts/confirmations               | Use for first-run CLI questions: repo, labels, provider, machine profile, safety warnings.                                                        |
| ByteNess/keyring or OS keychain integration | v1.9.1 candidate     | Store provider/GitHub credentials securely | Prefer using existing `gh` auth where possible; store provider tokens in OS credential store, not plaintext config.                               |
| yaml.v3 successor (`go.yaml.in/yaml/v3`)    | latest stable        | YAML config parsing                        | Cobra release notes indicate migration away from deprecated `gopkg.in/yaml.v3`; use current maintained module.                                    |
| mattn/go-sqlite3 or modernc.org/sqlite      | latest stable        | Local inventory/state database             | Use if JSON state becomes fragile. For earliest v1, a versioned JSON state file may be enough; SQLite becomes useful for multiple runners/events. |
| zerolog or slog                             | Go stdlib + optional | Structured logs and redaction              | Use standard `log/slog` first; add zerolog only if performance/format control matters. Redaction is more important than library choice.           |
| pkg/sftp                                    | latest stable        | Remote file upload over SSH                | Needed if not relying on shell heredocs/scp for bootstrap scripts.                                                                                |
| hcloud-go or hcloud CLI invocation          | current              | Hetzner API access                         | Prefer direct API client for reliable automation; invoking `hcloud` can reduce initial complexity but adds external dependency.                   |

### Development Tools

| Tool                              | Purpose                                  | Notes                                                                                           |
| --------------------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| GoReleaser                        | v2.15/v2.16 stable line; avoid nightlies | Multi-platform binary releases, checksums, Homebrew tap packages                                | Use stable GoReleaser, not nightly releases. Configure macOS/Linux amd64/arm64 at minimum.                            |
| GitHub Actions                    | CI for RunnerKit itself                  | Use GitHub-hosted runners initially; self-hosted dogfooding can come after RunnerKit is stable. |
| golangci-lint                     | latest stable                            | Static analysis                                                                                 | Enable security/reliability-focused linters and formatting.                                                           |
| govulncheck                       | latest stable                            | Go vulnerability scanning                                                                       | Run in CI and release pipeline.                                                                                       |
| testcontainers-go or local Docker | latest stable                            | Integration tests for Linux/service scripts where possible                                      | Useful for bootstrap script testing, but real GitHub runner registration requires mocked API or controlled test repo. |
| httptest + golden fixtures        | Go stdlib                                | GitHub/provider API testing                                                                     | Mock GitHub and cloud APIs; do not hit live APIs in unit tests.                                                       |
| ShellCheck                        | latest stable                            | Bootstrap script correctness                                                                    | Any generated remote shell scripts must pass ShellCheck.                                                              |
| cosign/SLSA provenance            | latest stable                            | Release integrity                                                                               | Important for a CLI that asks for GitHub/provider access. Add once release pipeline exists.                           |

## Installation

```bash
# Core Go module shape
go mod init github.com/<owner>/runnerkit
go get github.com/spf13/cobra@v1.10.2
go get github.com/google/go-github/v76/github

# Optional/supporting packages, exact choices to finalize during implementation
go get golang.org/x/crypto/ssh@latest
go get go.yaml.in/yaml/v3@latest
# choose one prompt/config/keyring implementation during Phase 1

# Dev tooling
go install github.com/goreleaser/goreleaser/v2@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Alternatives Considered

| Recommended                  | Alternative                                        | When to Use Alternative                                                                                                                                                          |
| ---------------------------- | -------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go CLI                       | Rust CLI                                           | Use Rust if the team strongly prioritizes memory-safety and is comfortable with slower iteration and a smaller GitHub/cloud SDK ecosystem.                                       |
| Go CLI                       | Node/TypeScript CLI                                | Use TypeScript if rapid GitHub API/product iteration matters more than single-binary distribution and remote ops reliability. Node increases runtime/package-manager complexity. |
| Cobra                        | urfave/cli                                         | Use urfave/cli for a smaller command tree. Cobra is better for a growing command hierarchy and docs generation.                                                                  |
| Direct provider API          | Terraform/OpenTofu as embedded provisioning engine | Use Terraform/OpenTofu if multi-provider declarative infrastructure becomes the product. For v1, embedding Terraform adds install/state complexity.                              |
| Hetzner default cloud path   | DigitalOcean                                       | Use DigitalOcean if developer familiarity/support in target market outweighs Hetzner's cost advantage. DO may be smoother globally but usually less cost-aggressive.             |
| BYO SSH + one cloud provider | Kubernetes/ARC                                     | Use Kubernetes/ARC only for organizations already operating clusters. It is too heavy for solo-dev v1.                                                                           |
| Local-first CLI state        | Hosted control plane                               | Use hosted control plane only if RunnerKit becomes a SaaS. It adds trust, ops, billing, and data handling scope.                                                                 |

## What NOT to Use

| Avoid                                                 | Why                                                                                                                                             | Use Instead                                                                                 |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Kubernetes/Actions Runner Controller as v1 foundation | Powerful but far too heavy for solo developers who want cheap, quick setup; requires cluster knowledge and ongoing ops.                         | Linux VPS + systemd runner management first.                                                |
| Persistent runners for public PR workloads by default | GitHub warns self-hosted runners on public repos can execute dangerous code from forks; persistent machines can be compromised or contaminated. | Private-repo-only persistent default, explicit safety warnings, optional ephemeral profile. |
| Long-lived GitHub registration tokens in config/state | Runner registration/removal tokens are short-lived and sensitive; persisting them creates avoidable risk.                                       | Request just-in-time, pass once, redact logs, discard immediately.                          |
| Broad provider matrix in v1                           | More providers means more auth, quota, networking, image, SSH, and cleanup failures before core value is proven.                                | One excellent default provider plus clean provider interface.                               |
| Auto-editing workflow YAML in v1                      | Can create surprising diffs and break project CI. User chose registration-only.                                                                 | Print labels and example snippets; developer edits workflows manually.                      |
| Root-running runner service                           | Increases blast radius for malicious workflows.                                                                                                 | Dedicated non-root `runnerkit`/`actions-runner` user with least privileges.                 |
| Opaque generated shell scripts                        | Hard to debug and dangerous when installing agents on servers.                                                                                  | Idempotent, readable bootstrap scripts with dry-run/logging and ShellCheck.                 |
| Plaintext provider/GitHub tokens                      | CLI security footgun.                                                                                                                           | Existing `gh` auth, OS keychain, or env references with clear warnings.                     |

## Stack Patterns by Variant

**If BYO machine setup:**

- Use local Go CLI → SSH executor → remote bootstrap script → systemd service.
- Because it minimizes external dependencies and gives solo developers immediate value on machines they already own.

**If default cloud provisioning:**

- Use local Go CLI → provider adapter (Hetzner candidate) → cloud-init/SSH bootstrap → GitHub runner registration.
- Because it preserves the same runner installation path while adding machine creation and cost tagging.

**If persistent runner mode:**

- Use repository-scoped runner labels, systemd service, health checks, and `runnerkit doctor` repair.
- Because persistent mode is simplest and fastest for trusted private solo repos.

**If ephemeral mode:**

- Treat it as a lifecycle-managed profile, not just `config.sh --ephemeral`.
- Because true safe ephemeral use requires one-job lifecycle, log forwarding/preservation, cleanup, and a controller or explicit user-triggered lifecycle.

**If future provider expansion:**

- Define a provider interface around `CreateMachine`, `InspectMachine`, `DestroyMachine`, `EnsureSSHReady`, `EstimateCost`, and `Tags`.
- Because cloud-specific details should not leak into GitHub/runner lifecycle code.

## Version Compatibility

| Package A                                    | Compatible With                                            | Notes                                                                                                                                             |
| -------------------------------------------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go 1.25.x                                    | Cobra v1.10.2                                              | Cobra release notes show modern dependencies and pflag updates; pin and test CLI behavior after updates.                                          |
| Go 1.25.x                                    | google/go-github v76.0.0                                   | go-github major versions may include breaking API changes; pin exact major and wrap it behind RunnerKit's own GitHub adapter.                     |
| GitHub REST API 2026-03-10                   | `Accept: application/vnd.github+json` + API version header | Use explicit API version header and test token scopes/permissions against GitHub docs.                                                            |
| GitHub runner binary                         | Linux x64/arm64 hosts                                      | Bootstrap must detect architecture and download matching runner package from GitHub's runner releases/docs.                                       |
| Hetzner Cloud provider v1.61.x / CLI v1.62.x | Hetzner Cloud API                                          | Provider/CLI versions are current ecosystem signals; RunnerKit should use API/client abstractions and not depend on Terraform provider internals. |
| Bubble Tea v2.0.6                            | Optional TUI                                               | Major v2 migration means avoid adding Bubble Tea until/if guided setup needs richer UX.                                                           |

## Sources

- GitHub Docs - REST API endpoints for self-hosted runners: registration/removal tokens, runner list/delete, labels, API version headers.
- GitHub Docs - Self-hosted runner reference/autoscaling: ephemeral runners are recommended for autoscaling; persistent autoscaling has assignment caveats.
- GitHub Docs - Secure use/hardening and runner groups: self-hosted runners should be used carefully with public repositories/untrusted forks.
- spf13/cobra GitHub releases - verified Cobra latest release v1.10.2.
- google/go-github releases - verified current major v76.0.0.
- charmbracelet/bubbletea releases - verified Bubble Tea v2.0.6.
- Hetzner hcloud CLI/pkg.go.dev - hcloud CLI v1.62.2 ecosystem signal.
- Hetzner Terraform provider/pkg.go.dev - provider v1.61.0 ecosystem signal.
- GoReleaser releases - v2 stable line with active nightlies; use stable release, not nightly.

---

_Stack research for: RunnerKit_
_Researched: 2026-04-28_
