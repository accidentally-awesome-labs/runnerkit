# Phase 4: Recommended Cloud Path and Billable Cleanup - Research

**Researched:** 2026-04-30
**Status:** Complete
**Phase:** 04-recommended-cloud-path-and-billable-cleanup

## RESEARCH COMPLETE

<objective>
What do we need to know to plan Phase 4 well?
</objective>

Phase 4 should add a single recommended cloud path by introducing a provider boundary, implementing one Hetzner Cloud adapter, reusing the existing BYO SSH/preflight/bootstrap path after a cloud VM is ready, then adding `runnerkit destroy` to remove GitHub registration plus all RunnerKit-created billable provider resources. The risky parts are not GitHub registration or runner install — those are already implemented for BYO — but cloud provider identity, partial provisioning cleanup, SSH/cloud-init readiness, primary-IP billing semantics, and making destroy verifiably non-billable.

---

## Recommendation Summary

### Recommended provider/profile

Choose **Hetzner Cloud** for the v1 recommended cloud path.

Default profile for planning:

| Field           | Recommended value                                                                  | Why                                                                                                                       |
| --------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Provider        | `hetzner` / `hcloud`                                                               | Strong cost story, official Go SDK, simple VPS primitives, labels/tags, SSH keys, firewalls, server actions.              |
| Region/location | `fsn1` by default, overridable with `--cloud-region`                               | Low-cost EU default with strong availability; avoids US/SIN traffic/pricing surprises unless user opts in.                |
| Server type     | `cpx22`                                                                            | Reliability-first low-cost default: enough RAM/disk for a persistent CI runner without jumping to dedicated vCPU pricing. |
| Image           | `ubuntu-24.04`                                                                     | Current LTS, supported by existing Linux/systemd/preflight assumptions.                                                   |
| Architecture    | x64                                                                                | Existing bootstrap package and labels support `linux/x64`; arm64 can follow after the first path is reliable.             |
| SSH user        | `runnerkit-admin` via cloud-init                                                   | Avoids routinely SSHing as root; separate from non-root `runnerkit-runner` service user created by bootstrap.             |
| Network         | public IPv4 + IPv6, no volumes/snapshots/backups/floating IPs                      | Minimal happy path; avoid optional billable add-ons in v1 cloud setup.                                                    |
| Firewall        | Hetzner Cloud Firewall allowing TCP/22 inbound; outbound all                       | Firewalls are provider-managed and free; show the rule in the plan and support `--ssh-allowed-cidr`.                      |
| Tags/labels     | `runnerkit=true`, repo slug, runner name, state ID, mode, managed=true, created-at | Destroy/status can identify ownership and avoid deleting unrelated resources.                                             |

Cost estimates should be shown as **approximate**. Use a small static pricing catalog for the default profile plus explicit caveats, and keep the code structured so pricing can be refreshed. Research evidence: Hetzner bills servers hourly with a monthly cap and continues billing stopped servers until deletion; Primary IPv4 resources are billed separately and must not be left orphaned. Display copy should say prices vary by region/currency/date and billing stops only after all RunnerKit-created billable resources are deleted or absent.

### Why Hetzner over DigitalOcean for v1

- Hetzner remains materially cheaper for always-on small VMs, which fits RunnerKit's solo-developer cost positioning.
- Hetzner provides official Go libraries and a mature REST API for servers, SSH keys, firewalls, labels, actions, and deletes.
- Hetzner now has EU, US, and Singapore locations, so it is less Europe-only than earlier research suggested.
- DigitalOcean has smoother global familiarity and docs, but its comparable 4GB profiles are significantly more expensive. It is a good second-provider candidate after the first path is proven, not the Phase 4 default.

Caveat: Hetzner account creation, project quotas, location/server-type availability, and 2026 pricing changes are the main friction points. The adapter must validate credentials, quota/capacity, location, image, and server type before mutation where possible, and give exact remediation if validation fails.

---

## Provider/API Implementation Findings

### Go SDK selection

Use `github.com/hetznercloud/hcloud-go` **v1.x** for this phase, not `github.com/hetznercloud/hcloud-go/v2`, unless the project upgrades from Go 1.22. Current package research showed:

- `github.com/hetznercloud/hcloud-go` v1.59.2 supports Go 1.21+ and works with this repo's current `go 1.22` module.
- `github.com/hetznercloud/hcloud-go/v2` currently requires Go 1.25, which would be a separate release/platform decision.

Isolate the SDK behind `internal/provider` interfaces and fake clients so a later SDK upgrade is local to the adapter.

### Required provider package shape

Add a provider boundary before implementing Hetzner-specific code:

```go
type Provider interface {
    Name() string
    Validate(ctx context.Context, input ProvisionInput) (ValidationResult, error)
    Plan(ctx context.Context, input ProvisionInput) (ProvisionPlan, error)
    Provision(ctx context.Context, input ProvisionInput) (Machine, error)
    WaitReady(ctx context.Context, machine Machine) (Machine, error)
    Describe(ctx context.Context, ref state.ProviderRef) (ProviderStatus, error)
    Destroy(ctx context.Context, ref state.ProviderRef) (DestroyResult, error)
    VerifyDestroyed(ctx context.Context, ref state.ProviderRef) (VerificationResult, error)
}
```

The provider must return a normalized `remote.Target`/machine reference and provider resource inventory. Core/CLI should remain responsible for GitHub auth, registration tokens, BYO bootstrap, labels, status composition, and state persistence.

### Credential discovery

For Hetzner, support these sources in order:

1. `HCLOUD_TOKEN` environment variable.
2. `HETZNER_CLOUD_TOKEN` environment variable as RunnerKit-friendly alias.
3. Provider CLI guidance only when env is missing: tell the user to create a Hetzner API token and export it; do not persist it in RunnerKit state.

If credentials are missing or invalid, fail before provider/GitHub/SSH/bootstrap mutation and print exact setup steps. Do not request a GitHub registration token until after the cloud VM is ready and BYO preflight passes.

### Resource inventory to persist

Extend state to capture enough provider identity for status/destroy/reconciliation:

- provider kind: `hetzner`
- server ID, name, status, location, datacenter when available
- server type/profile and image
- public IPv4 and IPv6, plus primary IP IDs if returned
- SSH key ID/name/fingerprint/public key reference
- firewall ID/name/rule shape
- tags/labels used for ownership
- cloud-init/user-data version/checkpoint
- cleanup resource IDs and operation checkpoints
- approximate cost profile used at provisioning time

This can be done by extending `ProviderRef`, `MachineRef`, `CleanupMetadata`, and/or adding nested structs to `state.RepositoryState`. Keep secrets out of state: token values, private key bytes, and registration/removal tokens must never be stored.

---

## Cloud Setup Flow to Plan

### `runnerkit up` UX

Existing `up` currently requires BYO `--host`. Phase 4 should preserve BYO flags and add explicit cloud intent:

Interactive:

- If no `--host` is provided and stdin is interactive, offer:
  1. BYO SSH host
  2. Recommended cloud runner (Hetzner, default profile)
- Show the cloud provisioning plan before mutation.
- Ask for explicit confirmation before creating billable resources.

Non-interactive:

- BYO remains `runnerkit up --repo owner/name --host user@host --yes`.
- Cloud must require explicit intent, e.g. `runnerkit up --repo owner/name --cloud hetzner --yes` or `--provider hetzner --yes`.
- A missing `--host` plus `--yes` must fail with `input_required`; it must not silently provision cloud infrastructure.

### Provisioning plan contents

Before mutation, human and JSON output should include:

- provider: `hetzner`
- location/region: default `fsn1` unless overridden
- server type: default `cpx22`
- image: `ubuntu-24.04`
- cost estimate: approximate hourly/monthly plus caveat
- resources to create: server, SSH key, cloud firewall, public IP(s), no backups/snapshots/volumes
- names/tags/labels: full values printed before create
- SSH key source/path and public fingerprint only
- firewall/network shape including SSH CIDR
- runner labels and exact `runs-on` snippet preview
- exact future cleanup command: `runnerkit destroy --repo owner/name`

### Mutation ordering

Use this order to avoid partial, billable, or split-brain states:

1. Resolve repo and GitHub permissions/safety, but do **not** request registration token.
2. Resolve cloud credentials and validate profile/region/image/server type.
3. Build and render provisioning plan.
4. Confirm billable cloud creation.
5. Create provider resources with deterministic names/tags.
6. Persist a pending operation checkpoint as soon as a billable resource exists.
7. Wait for provider status `running` and public IP assignment.
8. Wait for SSH host key and cloud-init readiness.
9. Run existing BYO preflight on the cloud target.
10. Only then request GitHub registration token and call the existing bootstrap/install path.
11. Verify GitHub runner online with labels and service active.
12. Save final state with provider inventory and cleanup IDs.

If provisioning fails after billable resources exist, keep state/checkpoints and print `runnerkit destroy --repo owner/name` as the next action. Do not hide the resource IDs.

---

## Readiness and Bootstrap Findings

Fresh cloud VMs are not ready when the create API returns. Readiness should be a state machine, not a fixed sleep:

1. Provider action complete / server status running.
2. Public address assigned.
3. `ssh-keyscan` obtains a host key and host-key fingerprint is recorded.
4. SSH command works with the chosen key/user.
5. Cloud-init is finished (`cloud-init status --wait` when present, or `/var/lib/cloud/instance/boot-finished`).
6. Existing `preflight.Run` passes with `AllowUnknownLinux=false` for Ubuntu 24.04.

Use cloud-init only for base access and readiness, not GitHub registration tokens. Suggested user-data responsibilities:

- create `runnerkit-admin` with the selected public SSH key
- grant passwordless sudo for bootstrap commands
- install/ensure `sudo` and basic packages if the image is minimal
- write a RunnerKit bootstrap marker under `/var/lib/runnerkit/cloud-init.json`

The existing bootstrap path should remain the only runner installer. The cloud path should feed it a `remote.Target` and the same `bootstrap.Options` used for BYO.

---

## Status, Doctor, Logs, and Destroy Implications

### Status/logs/doctor

`status` currently reconciles state, GitHub, SSH, service, and labels. Phase 4 should add a provider source to the observed model:

- provider found/missing/error
- instance status (`running`, `off`, `deleting`, missing)
- provider location/profile/image
- public host/IP
- billable resources summary
- drift/orphan warnings when state IDs are missing in the provider or provider-tagged resources exist without matching state

Human output should add a `Provider` source line. JSON should include `sources.provider` with stable fields for tests.

### `runnerkit destroy`

Keep `runnerkit down` for BYO cleanup. Add `runnerkit destroy` for cloud-managed resources.

Destroy plan should include, in order:

- GitHub runner record/removal
- remote runner unconfiguration and service/files cleanup when SSH is reachable
- Hetzner server deletion
- SSH key deletion if RunnerKit-created
- firewall deletion if RunnerKit-created
- Primary IPv4 deletion/unassignment if RunnerKit-created or still billable
- local state removal only after selected cleanup succeeds and provider resources verify absent/non-billable

`destroy --dry-run` should render the plan. `destroy --yes` should apply the safe default full cleanup. Interactive destroy should require explicit confirmation because billable resources are involved.

Do not declare success on the provider delete action alone. Verify by re-describing each saved resource ID and checking it is absent or explicitly non-billable. If any billable resource is still present or verification cannot be completed, keep state with pending checkpoints and return non-success JSON (`ok:false`, `partial_cleanup:true`, pending IDs listed).

---

## Documentation Findings

Add a concise cloud quickstart beside the BYO quickstart:

- `docs/cloud-quickstart.md`
- README section linking BYO and cloud paths

Required quickstart content:

1. Hetzner API token setup using `HCLOUD_TOKEN`.
2. Cost estimate caveat and why RunnerKit chooses one default profile.
3. `runnerkit up --repo owner/name --cloud hetzner` interactive flow.
4. Non-interactive example requiring `--cloud hetzner --yes`.
5. Exact `runs-on` labels/snippet guidance.
6. Lifecycle reuse: `status`, `logs`, `doctor`.
7. Cleanup: `runnerkit destroy --repo owner/name --dry-run`, then `runnerkit destroy --repo owner/name` or `--yes`.
8. Limitations: one recommended provider path, persistent trusted/private default, no workflow YAML edits, ephemeral mode deferred to Phase 5.

---

## Planning Slices

Use the roadmap's four slices; they map cleanly to implementation risk:

1. **04-01 Provider interface/profile/credential/plan**
   - Add provider interfaces, cost/profile config, Hetzner adapter skeleton/fakes, CLI flags/interactive branch, and provisioning plan rendering.
   - No live resources created yet unless behind fake tests.

2. **04-02 VM/SSH key/firewall/tags/readiness**
   - Implement Hetzner create/wait/describe with fake API tests.
   - Add SSH key handling, cloud-init, firewall rules, tags, pending checkpoints, and readiness before registration.

3. **04-03 Shared runner installation/status/logs/doctor**
   - Reuse BYO bootstrap after cloud readiness.
   - Save full provider inventory.
   - Extend status/doctor/logs JSON/human output with provider facts.

4. **04-04 Destroy/billing verification/docs**
   - Add `runnerkit destroy` with dry-run/yes/interactive flows.
   - Delete GitHub + remote + provider resources, verify absent/non-billable, keep pending state on partial failure.
   - Add cloud quickstart and README updates.

---

## Key Risks and Mitigations

| Risk                                                     | Impact                 | Mitigation                                                                                                              |
| -------------------------------------------------------- | ---------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Hetzner profile/price changes                            | Plan/docs become stale | Store approximate catalog with caveats; validate server type/location via API before mutation; docs say pricing varies. |
| Primary IP left billing                                  | Surprise bill          | Persist primary IP IDs and verify deletion/absence during destroy.                                                      |
| Partial provision creates billable VM then install fails | User pays for orphan   | Write pending state as soon as first billable resource exists; show destroy command; tag all resources.                 |
| SSH/cloud-init race                                      | Flaky first-run        | Wait for provider running, host key, SSH, cloud-init finished, then BYO preflight.                                      |
| Provider token leakage                                   | Trust loss             | Env-only token handling, redaction, no state persistence, no tokens in cloud-init.                                      |
| Broad provider abstraction delays work                   | Scope creep            | Define a thin interface but implement only Hetzner and fake provider in Phase 4.                                        |
| Non-interactive accidental bill                          | Bad UX/safety          | Require explicit `--cloud hetzner` plus `--yes`; missing host + yes fails.                                              |

---

## Validation Architecture

### Test infrastructure

- Framework: Go `testing` with existing fake GitHub/remote/state helpers.
- Quick command: `go test ./internal/provider/... ./internal/cli/... ./internal/ops/... ./internal/state/...`
- Full command: `go test ./...`
- No live Hetzner or GitHub calls in automated tests. Use fake provider clients and/or `httptest`.

### Required automated coverage by plan

| Plan  | Validation focus                                                                                                               | Required commands/checks                                                               |
| ----- | ------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| 04-01 | Provider interfaces, Hetzner profile constants, credential errors, plan rendering, non-interactive cloud intent safety         | `go test ./internal/provider/... ./internal/cli/...`                                   |
| 04-02 | Resource names/tags, SSH key/firewall/server create ordering, pending checkpoints, readiness wait, failure before GitHub token | `go test ./internal/provider/... ./internal/cli/... ./internal/state/...`              |
| 04-03 | Cloud path reuses BYO bootstrap, state inventory saved, provider source in status/doctor/logs JSON/human output                | `go test ./internal/cli/... ./internal/ops/... ./internal/state/...`                   |
| 04-04 | `destroy` plan/apply, partial failure checkpoints, verification absent/non-billable, docs contain required quickstart commands | `go test ./internal/cli/... ./internal/ops/... ./internal/provider/...` plus grep docs |

### Nyquist sampling guidance

- After each task: run the focused package tests touched by that task.
- After each plan: run `go test ./...`.
- For docs tasks: grep exact command strings and limitation/cost caveat text.
- Sampling continuity: no more than two implementation tasks should occur without an automated provider/CLI/status/destroy test.

### Manual-only checks

Live cloud provisioning is manual/smoke-only before public release because it creates billable resources and requires a real Hetzner project. The plans should include an optional manual smoke in docs/verification, but unit tests must not require credentials.

Suggested smoke after Phase 4 implementation:

```bash
export HCLOUD_TOKEN=...
runnerkit up --repo owner/private-repo --cloud hetzner --dry-run
runnerkit up --repo owner/private-repo --cloud hetzner
runnerkit status --repo owner/private-repo
runnerkit logs --repo owner/private-repo --since 10m
runnerkit doctor --repo owner/private-repo
runnerkit destroy --repo owner/private-repo --dry-run
runnerkit destroy --repo owner/private-repo
runnerkit status --repo owner/private-repo --json
```

Pass criteria: runner reaches online, status includes provider facts, destroy verifies GitHub runner absent and saved provider resources absent/non-billable, and no RunnerKit-created server/firewall/SSH key/primary IP remains.

---

## Sources Consulted

- `.planning/phases/04-recommended-cloud-path-and-billable-cleanup/04-CONTEXT.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md`
- `.planning/ROADMAP.md`
- `.planning/research/STACK.md`
- `.planning/research/ARCHITECTURE.md`
- `.planning/research/FEATURES.md`
- `.planning/research/PITFALLS.md`
- `internal/cli/up.go`, `internal/cli/status.go`, `internal/cli/down.go`
- `internal/ops/status.go`, `internal/ops/cleanup.go`
- `internal/state/schema.go`
- `internal/preflight/checks.go`
- Hetzner Cloud pricing/product pages: https://www.hetzner.com/cloud/regular-performance and https://www.hetzner.com/cloud/cost-optimized
- Hetzner billing FAQ: https://docs.hetzner.com/cloud/billing/faq
- Hetzner Cloud API reference: https://docs.hetzner.cloud/reference/cloud
- Hetzner hcloud-go package research: https://pkg.go.dev/github.com/hetznercloud/hcloud-go and https://pkg.go.dev/github.com/hetznercloud/hcloud-go/v2
- DigitalOcean Droplets/API/firewall/SSH key documentation for comparison: https://docs.digitalocean.com/reference/api/reference/droplets/
