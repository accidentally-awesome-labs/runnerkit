# Pitfalls Research

**Domain:** CLI-first self-hosted GitHub Actions runner setup and management for solo developers
**Researched:** 2026-04-28
**Confidence:** MEDIUM

## Critical Pitfalls

### Pitfall 1: Letting public or untrusted workflows run on persistent self-hosted runners

**What goes wrong:**
A public repository, fork pull request, compromised dependency, or malicious workflow gets scheduled onto a persistent self-hosted runner. The job can inspect the machine, reuse previous workspace residue, consume compute, attack the local network, steal credentials left on disk, or tamper with future builds.

**Why it happens:**
Self-hosted runners feel like a cheaper drop-in replacement for GitHub-hosted runners, and the cheapest/simple default is a persistent VM. Solo developers may not realize GitHub's own guidance treats public-repo self-hosted runners as high risk because arbitrary workflow code can execute on infrastructure they own.

**How to avoid:**
- Make the default happy path explicitly for private/trusted repositories.
- Detect public repositories and fork-enabled workflows during setup; block or require an explicit `--allow-public-repo-risk` confirmation.
- Recommend GitHub-hosted runners, no self-hosted runner, or true ephemeral single-job isolation for public/untrusted jobs.
- Use a dedicated `runnerkit` OS user, dedicated VM, no ambient cloud credentials, and minimal network reachability.
- Generate labels that do not accidentally match unrelated workflows, and print safe `runs-on` guidance.

**Warning signs:**
- Repository visibility is public and RunnerKit is about to install a persistent runner.
- Workflows use `pull_request`, run fork contributions, or use broad `runs-on: self-hosted` labels.
- Runner service runs as root or has broad local/cloud credentials.
- The user asks for the cheapest persistent runner without discussing trust boundaries.

**Phase to address:**
Phase 1 / GitHub registration and mode-selection: safety gate before any runner is registered; Phase 4 / ephemeral mode for safer untrusted-job support.

---

### Pitfall 2: Mishandling GitHub registration and removal tokens

**What goes wrong:**
Short-lived runner registration/removal tokens expire mid-install, get pasted into logs, are stored in local state, or are reused incorrectly during retries. Setup becomes fragile and leaked tokens create a short but serious attack window.

**Why it happens:**
GitHub's manual setup flow includes copy-paste registration commands. Automation often wraps those commands without modeling token expiration, retry behavior, or redaction. Remote SSH bootstrap scripts make the problem worse because tokens must cross process and machine boundaries.

**How to avoid:**
- Request registration/removal tokens just-in-time and use them immediately.
- Treat registration/removal token expiration as normal: re-request on retry instead of replaying stale scripts.
- Never persist raw registration tokens in RunnerKit state, shell history, generated scripts, debug bundles, or cloud-init user data that remains readable.
- Redact tokens in command output and logs; pass via protected stdin/temp file with `0600` permissions when needed.
- Prefer a GitHub App/device flow or fine-grained token guidance with minimal repository administration scope for obtaining registration tokens.

**Warning signs:**
- State files contain `token`, `registration_token`, `remove_token`, or generated `config.sh --token ...` commands.
- Installs fail after a long prompt/provisioning delay with authentication or 401/404-style errors.
- Retry uses the same bootstrap script created more than a few minutes earlier.
- Debug mode prints full GitHub API responses or `config.sh` arguments.

**Phase to address:**
Phase 1 / GitHub auth and registration: token lifecycle abstraction, redaction, and retry semantics must be present before BYO/cloud setup uses them.

---

### Pitfall 3: Persistent runner contamination between jobs

**What goes wrong:**
A persistent runner accumulates workspace files, Docker containers/images, package caches, environment changes, credentials, and tool versions from previous jobs. Later jobs become flaky, non-reproducible, or exposed to data left by earlier jobs.

**Why it happens:**
GitHub-hosted runners are fresh for each job, but a self-hosted persistent runner is a long-lived machine. Developers often assume workflow cleanup is enough, while build tools, Docker, language package managers, and failed jobs leave state outside the repository checkout.

**How to avoid:**
- Be explicit that persistent mode is only for trusted workloads and trades isolation for cost/speed.
- Run jobs under a dedicated unprivileged user and dedicated `_work` directory.
- Add pre/post-job cleanup hooks or `runnerkit doctor` checks for old workspaces, Docker leftovers, disk pressure, and mutated permissions.
- Offer ephemeral mode for workflows that handle secrets, untrusted code, or high contamination risk.
- Document cache locations that are intentionally preserved versus directories RunnerKit may wipe.

**Warning signs:**
- Jobs pass on RunnerKit but fail on GitHub-hosted runners, or vice versa.
- Disk usage grows steadily; old repositories remain under `_work`.
- `docker ps -a`, `docker volume ls`, or package caches show old job artifacts.
- Builds rely on tools installed by a previous workflow instead of setup steps.

**Phase to address:**
Phase 2 / BYO service management: safe user/workdir layout and cleanup checks; Phase 4 / ephemeral mode for strong isolation.

---

### Pitfall 4: Running the runner with excessive host privileges

**What goes wrong:**
The runner service runs as root, has passwordless sudo, can access the Docker socket, or lives on a host with unrelated services and secrets. Any workflow code effectively gains control of the machine, and often root-equivalent access through Docker.

**Why it happens:**
Manual setup examples are often executed from whichever shell the developer has open. Package installs and service creation need elevated privileges, so it is easy to keep running the actual runner under the same privileged account.

**How to avoid:**
- Bootstrap with privilege only for installation, then run the runner service as a dedicated unprivileged `runnerkit` user.
- Create a dedicated VM for runners; do not install on a laptop, workstation, production server, or machine with personal secrets.
- Treat Docker access as privileged; warn if the runner user is added to the `docker` group and recommend dedicated/ephemeral VMs for Docker-heavy jobs.
- Lock down file permissions for RunnerKit state, runner directories, SSH keys, and provider credentials.
- Record privilege posture in `runnerkit status` and fail `doctor` on root-runner configurations unless explicitly overridden.

**Warning signs:**
- `runsvc.sh` or systemd service runs as `root`.
- Runner home directory is `/root` or shares a real user's home.
- The runner user has broad sudo or cloud provider credentials.
- Docker socket is mounted/accessible without an explicit risk warning.

**Phase to address:**
Phase 2 / BYO install and service management; Phase 3 / cloud image provisioning should produce the same privilege model by default.

---

### Pitfall 5: Leaking secrets through logs, state files, process lists, or support bundles

**What goes wrong:**
GitHub tokens, cloud API keys, SSH private keys, repository secrets, or workflow-derived credentials appear in CLI output, systemd logs, runner diagnostic logs, cloud-init logs, shell history, debug archives, or process command lines.

**Why it happens:**
Troubleshooting self-hosted runners often means printing commands, environment variables, and remote logs. GitHub masks known secrets in workflow logs, but RunnerKit must also protect its own setup secrets and cannot assume every derived value is masked.

**How to avoid:**
- Centralize secret handling and redaction for all CLI logs, remote command output, diagnostics, and error reports.
- Avoid passing secrets as command-line arguments where they can appear in process listings; prefer stdin or protected files removed immediately.
- Store long-lived configuration separately from secrets; prefer OS keychain/credential helpers or `0600` files with clear warnings.
- Make `runnerkit doctor --bundle` produce a redacted archive by default with a manifest of included files.
- Never include cloud provider credentials in remote cloud-init/user-data if a shorter-lived bootstrap token or local provisioning call will do.

**Warning signs:**
- `runnerkit --debug` prints full environment variables or API payloads.
- Support instructions ask users to paste raw `_diag`, cloud-init, or systemd logs without redaction.
- State/config files include provider tokens or GitHub auth material.
- Tokens are visible in `ps`, shell history, generated scripts, or CI logs.

**Phase to address:**
Phase 1 / CLI foundation and GitHub auth; keep redaction mandatory before adding cloud provisioning and diagnostics.

---

### Pitfall 6: Split-brain status and stale/offline runner records

**What goes wrong:**
GitHub shows a runner as offline while the local service is running, a destroyed VM remains registered in GitHub, duplicate runner names confuse job routing, or queued jobs never start because labels no longer match any online runner.

**Why it happens:**
Runner state exists in at least three places: RunnerKit local state, the target host/service, and GitHub's runner registry. Cloud providers add a fourth. Manual fixes or partial failures easily make these disagree.

**How to avoid:**
- Make `runnerkit status` compare local state, remote service state, GitHub API runner status, labels, and provider resource state.
- Use stable unique runner names and predictable labels; detect duplicates before registration.
- On setup, detect and offer to remove stale registrations for the same repo/name.
- Make cleanup idempotent: if the host is gone, remove GitHub registration through API; if GitHub registration is gone, clean local service/files anyway.
- Include label-match diagnostics so users know exactly why a workflow job is queued.

**Warning signs:**
- GitHub UI shows offline runners that RunnerKit no longer knows about.
- `systemctl status` is green but GitHub reports offline.
- Jobs remain queued with `runs-on` labels that do not exactly match an online runner.
- Multiple runners share similar names or only generic labels.

**Phase to address:**
Phase 1 / registration naming and labels; Phase 2 / status and service management; Phase 3 / cloud state reconciliation.

---

### Pitfall 7: Cloud resources that keep billing after the user thinks they are done

**What goes wrong:**
VMs, disks, snapshots, static IPs, load balancers/firewalls, or orphaned ephemeral machines remain after failed provisioning, failed cleanup, or forgotten experiments. RunnerKit solves CI cost only to create surprise cloud bills.

**Why it happens:**
Cloud provisioning has many partial-success paths, and developers often stop after jobs work. Destroy flows are less exciting than setup and are frequently bolted on later. Ephemeral runners intensify the risk because every job may create resources.

**How to avoid:**
- Tag/label every cloud resource with `runnerkit`, project/repo, creation time, mode, and state ID.
- Show estimated hourly/monthly cost before provisioning and after setup.
- Implement `runnerkit destroy` in v1, not later; make it idempotent and show a plan before destructive actions.
- Add TTL/idle shutdown safeguards for cloud runners, especially ephemeral mode.
- Provide `runnerkit doctor --orphans` to compare provider resources, local state, and GitHub registrations.

**Warning signs:**
- Provider resources lack RunnerKit tags or local state references.
- `destroy` cannot find resources if local state is missing.
- Ephemeral jobs create instances but no finalizer/cleanup path runs on failure.
- VM uptime grows while no CI jobs have run recently.

**Phase to address:**
Phase 3 / default cloud provisioning must include cost display, tagging, cleanup, and orphan detection from the first release.

---

### Pitfall 8: Fragile SSH, cloud-init, and remote bootstrap flows

**What goes wrong:**
Provisioning intermittently fails because SSH is not ready, package managers are locked, outbound GitHub connectivity is blocked, architecture/OS is unsupported, Docker permissions are wrong, or the install script is not resumable after a partial failure.

**Why it happens:**
The one-command experience depends on unreliable remote systems: fresh cloud VMs, different Linux images, network firewalls, DNS, package mirrors, and user-provided machines. Naive scripts assume a clean Ubuntu box and a stable SSH session.

**How to avoid:**
- Run local and remote preflight checks before registration: OS, architecture, systemd, disk, memory, outbound HTTPS to GitHub, tar/gzip/curl, Docker if requested, and required privileges.
- Treat cloud-init/SSH readiness as a state machine with retries and clear timeouts, not a single sleep.
- Make bootstrap idempotent and resumable: safe to rerun after package install, runner download, service creation, or registration fails.
- Capture remote logs (`cloud-init`, package manager, runner config, systemd) with redaction and actionable failure hints.
- Keep supported platform matrix narrow in v1 (for example Linux x64/arm64) and fail fast elsewhere.

**Warning signs:**
- Setup instructions say "wait a bit and try again" without detecting readiness.
- Re-running setup creates duplicate users, services, runner directories, or GitHub registrations.
- Failures occur at different steps on identical cloud instances.
- BYO install works only on the developer's personal distribution/image.

**Phase to address:**
Phase 2 / BYO bootstrap and Phase 3 / cloud provisioning; include preflight before promising 10-minute setup.

---

### Pitfall 9: Unsafe or confusing label design

**What goes wrong:**
Workflows accidentally route to the wrong runner, never route at all, or route every `self-hosted` job to a low-powered personal VM. Users cannot tell which labels to put in `runs-on`, and old labels remain after mode/provider changes.

**Why it happens:**
GitHub runner matching depends on labels, but the native defaults (`self-hosted`, OS, arch) are too generic for product-managed fleets. Solo developers may copy `runs-on: self-hosted` and unintentionally bind unrelated workflows to RunnerKit.

**How to avoid:**
- Always add stable RunnerKit-specific labels such as `runnerkit`, `runnerkit-<repo-slug>`, `persistent`/`ephemeral`, OS, architecture, and capability labels.
- Print the exact recommended `runs-on` array after setup and in `runnerkit status`.
- Avoid per-run random labels unless the workflow snippet is updated or intentionally generated for that runner.
- Add read-only workflow/label validation later: show which jobs would match which runners without editing YAML.
- Warn when workflows use only `self-hosted` and multiple self-hosted runners exist.

**Warning signs:**
- Jobs queued with "waiting for a runner to pick up this job" even though a runner is online.
- Workflows use `runs-on: self-hosted` without RunnerKit-specific labels.
- Cloud runner is x64/Linux but workflow expects ARM, GPU, Docker, or larger disk.
- Label list changes after every reinstall.

**Phase to address:**
Phase 1 / registration and workflow snippet; Phase 5 / read-only workflow readiness checks.

---

### Pitfall 10: Update and upgrade drift between RunnerKit, runner binaries, services, and state

**What goes wrong:**
GitHub runner binaries become too old, auto-update behavior surprises users, service definitions drift from the CLI's expectations, or a RunnerKit update cannot read old local state. Existing runners break after an otherwise routine upgrade.

**Why it happens:**
Self-hosted runners are long-lived agents controlled partly by GitHub, partly by local service files, and partly by RunnerKit. Product teams often build setup before building lifecycle management.

**How to avoid:**
- Track RunnerKit CLI version, runner binary version, service template version, state schema version, and provider plugin version.
- Let GitHub runner auto-update unless RunnerKit has a deliberate update-management story; if disabling auto-update, enforce an update SLA and surface required upgrades clearly.
- Make state migrations explicit, reversible where practical, and tested from old versions.
- Provide `runnerkit upgrade --plan` and `runnerkit upgrade --rollback` or at minimum service backup/restore instructions.
- Include a smoke test after upgrade: service starts, GitHub sees online, labels unchanged, sample job can be accepted.

**Warning signs:**
- `runnerkit status` does not show runner binary version or service template version.
- Upgrades require users to manually rerun GitHub's setup commands.
- State file has no schema version.
- Users report runners going offline after GitHub or RunnerKit updates.

**Phase to address:**
Phase 5 / lifecycle management and upgrades, with version fields added in Phase 1 state design.

---

### Pitfall 11: Poor diagnostics that turn every failure into SSH spelunking

**What goes wrong:**
Users see "runner offline" or "job queued" but must inspect GitHub UI, SSH into the host, read systemd logs, search `_diag`, inspect cloud console resources, and guess what went wrong. The product feels as fragile as manual setup.

**Why it happens:**
Status, logs, and repair are often treated as polish after setup. In this domain they are core product value because the user's pain is fragility and opacity.

**How to avoid:**
- Ship `runnerkit status`, `runnerkit logs`, and `runnerkit doctor` early.
- Diagnose by category: GitHub auth/registration, GitHub runner online/offline, label match, local service, runner process logs, disk/memory, Docker, network, cloud resource state, and cost/orphan checks.
- Make each finding actionable: "run this command", "RunnerKit can fix this", or "manual provider action required".
- Redact secrets automatically in all diagnostic output.
- Include a support bundle command with manifest, timestamps, versions, and redacted logs.

**Warning signs:**
- Errors only say "failed" without the command, host, phase, or next step.
- The CLI cannot explain queued jobs or offline status.
- Troubleshooting docs start with "SSH into the box" for common cases.
- Logs are inaccessible after ephemeral machines are destroyed.

**Phase to address:**
Phase 2 / status, logs, and doctor should be part of MVP, not deferred polish; Phase 4 must add external/preserved logs for ephemeral runners.

---

### Pitfall 12: Naive ephemeral runners that are not actually safe or cheap

**What goes wrong:**
Ephemeral mode runs only one job but leaves the VM alive, loses logs when the machine is destroyed, reuses a dirty image, registers too late to catch jobs, or fails to deregister after cancellation. Users get higher cost and operational complexity without reliable isolation.

**Why it happens:**
GitHub supports ephemeral self-hosted runners, but production-grade ephemeral behavior requires orchestration around registration, job assignment, logs, timeouts, finalizers, and cloud cleanup. It is easy to implement `--ephemeral` and miss everything around it.

**How to avoid:**
- Define v1 ephemeral scope narrowly: single-job runner lifecycle with clear limitations, not a full autoscaling platform.
- Always pair ephemeral registration with guaranteed cleanup/finalizer logic for GitHub deregistration and provider destroy.
- Forward or collect runner/application logs before destroying the host.
- Add max-runtime and no-job timeouts so unassigned instances do not linger.
- Use fresh images or explicit bootstrap each time; do not call a persistent runner "ephemeral" just because workspaces are cleaned.

**Warning signs:**
- Ephemeral mode has no TTL, no cleanup retry, or no cost estimate.
- Cloud instances remain after canceled workflows.
- Users cannot debug failed ephemeral jobs because the host and logs disappeared.
- Same VM handles multiple jobs despite being marketed as ephemeral.

**Phase to address:**
Phase 4 / ephemeral mode: only ship after cloud cleanup, redacted diagnostics, and status reconciliation are already reliable.

---

### Pitfall 13: Over-broad GitHub authentication and repository access

**What goes wrong:**
RunnerKit asks users for a classic PAT with broad `repo`/admin permissions, stores it long-term, or uses one token across unrelated repositories. A compromise of the CLI machine or runner state becomes a repository-wide compromise.

**Why it happens:**
Broad tokens make the API easy during prototyping, especially for listing repos and creating registration tokens. Fine-grained auth and GitHub App flows require more implementation work and clearer UX.

**How to avoid:**
- Prefer GitHub's least-privilege path available for the target flow: GitHub App, device flow, or fine-grained PAT guidance limited to the selected repository.
- Separate GitHub control-plane auth on the developer's machine from runner registration tokens on the runner host.
- Store auth tokens only when required for management commands; document where and how, and provide `runnerkit logout`.
- Never copy broad GitHub auth tokens to remote runner hosts.
- Scope org-level support later; repo-level v1 reduces permission surface.

**Warning signs:**
- Quickstart begins with "create a classic PAT with full repo/admin:org".
- Remote machine receives the user's durable GitHub token.
- RunnerKit cannot explain which GitHub permissions it needs and why.
- One local token manages every repository by default.

**Phase to address:**
Phase 1 / auth design before API integration hardens around broad PAT assumptions.

---

### Pitfall 14: Cleanup flows that fail after partial destruction

**What goes wrong:**
A user destroys the cloud VM first and can no longer run `config.sh remove`, or RunnerKit deregisters the GitHub runner but fails to delete cloud resources, or BYO cleanup deletes files but leaves services and users behind.

**Why it happens:**
Manual runner removal assumes the host is still accessible and tokens are fresh. Real users interrupt commands, lose SSH access, delete resources from the cloud console, or rerun setup with different state.

**How to avoid:**
- Treat cleanup as a reconciliation problem across GitHub, local/remote host, and provider resources.
- Use fresh removal tokens when local runner removal is possible; fall back to GitHub API deletion when the host is gone.
- Show a cleanup plan with each target and current state, then execute idempotently.
- Make every step safe to retry and record partial-completion markers.
- Include `runnerkit forget` only as a last resort, distinct from actual cleanup/destroy.

**Warning signs:**
- `destroy` assumes SSH is reachable.
- Failed cleanup leaves no record of which resources remain.
- Removing local state is treated as equivalent to deleting cloud/GitHub resources.
- Users can reinstall and create duplicate registrations after a failed cleanup.

**Phase to address:**
Phase 3 / cloud cleanup and Phase 2 / BYO cleanup; the state model in Phase 1 must support partial cleanup.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
| -------- | ----------------- | -------------- | --------------- |
| Prototype with broad classic PATs | Fast GitHub API integration | Hard-to-remove security smell; users distrust auth flow | Local spike only; never in public quickstart |
| Store generated `config.sh --token ...` scripts | Easy retry/debug | Token leakage and expired-token failures | Never for raw tokens |
| Use only `self-hosted` labels | Simple docs | Accidental routing, queued jobs, wrong machines | Never as the recommended snippet |
| Run runner as root | Avoids permissions errors | Workflow code owns the host | Never for v1 defaults |
| Cloud provisioning without tags | Faster provider code | Orphan/cost detection impossible | Never for managed cloud path |
| Setup first, cleanup later | Faster demo | Surprise bills and stale GitHub runners | Never; cleanup is table stakes |
| One bash script for all Linux | Quick bootstrap | Fragile distro/image behavior; poor diagnostics | MVP only if guarded by strict platform checks |
| No state schema version | Simpler config file | Upgrade/migration breakage | Never after first public release |
| Ephemeral mode as `--ephemeral` only | Marketing checkbox | Machines/logs/costs leak around the runner | Never; needs lifecycle orchestration |
| SSH-only troubleshooting | Avoids building diagnostics | Product still feels manual and fragile | Acceptable only for rare advanced failures |

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
| ----------- | -------------- | ---------------- |
| GitHub Actions runner registration | Reusing/storing short-lived registration tokens | Request just-in-time, redact, retry by re-requesting |
| GitHub runner removal | Assuming the runner host still exists | Use local remove when possible; fall back to GitHub API deletion |
| GitHub labels | Registering generic labels only | Add stable RunnerKit labels and print exact `runs-on` snippet |
| GitHub public repositories | Treating self-hosted as a cheap drop-in for public PRs | Block/warn strongly; require ephemeral or GitHub-hosted for untrusted code |
| GitHub auth | Asking for broad PATs and copying them to hosts | Least-privilege auth on developer machine; only short-lived registration token to host |
| Systemd | Installing service as root or losing environment | Dedicated user, explicit service file, status/log commands, restart policy |
| SSH | Assuming immediate connectivity after VM create | Wait for cloud-init/SSH readiness with retries and timeouts |
| Cloud provider API | Creating resources without tags/state | Tag every resource and reconcile provider inventory during status/destroy |
| Docker | Treating Docker access as harmless | Warn that Docker group is root-equivalent; isolate or use ephemeral VMs |
| Logging/support bundles | Bundling raw `_diag`, cloud-init, and env output | Redact by default and include a manifest of collected files |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
| ---- | -------- | ---------- | -------------- |
| One persistent low-spec VM for every workflow | Long queues, slow builds, disk pressure | Show capacity expectations; recommend labels/profiles by workload | More than one concurrent job or Docker-heavy builds |
| No queue/label readiness check | Jobs wait even though setup said success | Validate online runner and label match before declaring success | First real workflow run |
| Unbounded ephemeral provisioning | Many VMs start, costs spike, API limits hit | Max concurrency, TTL, cost estimate, cleanup finalizers | Parallel pushes/PRs or retry storms |
| Keeping every cache forever | Disk fills; builds become flaky | Cleanup policy and disk alerts in `doctor` | Weeks of builds or large Docker images |
| Polling GitHub/provider APIs aggressively | Rate limits and slow CLI | Backoff, cache non-sensitive metadata, clear progress | Multi-runner/pool management |
| Installing toolchains during every job on tiny VMs | Slow CI and high cloud time | Document cache strategy or larger instance profiles | Language builds with large dependency graphs |
| Ignoring architecture/capabilities in labels | Jobs scheduled to wrong CPU/OS | Include OS/arch/capability labels and validation | Mixed arm64/x64 or Docker/GPU workflows |

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
| ------- | ---- | ---------- |
| Persistent self-hosted runner for public/fork PRs | Malicious code can compromise runner and future jobs | Block/warn; use GitHub-hosted or true ephemeral isolation |
| Runner service runs as root | Workflow code gets host-level control | Dedicated unprivileged user and dedicated VM |
| Docker socket exposed silently | Docker access is effectively root on host | Explicit warning; isolated/ephemeral VM for Docker jobs |
| Durable GitHub PAT copied to runner host | Host compromise becomes GitHub repo compromise | Keep durable auth local; send only short-lived registration token |
| Registration tokens in logs/state | Runner hijack during token lifetime; trust loss | Just-in-time tokens and mandatory redaction |
| Cloud provider credentials on runner | Workflow can create/destroy infrastructure | Provision from local CLI; avoid provider creds on runner |
| Secrets in diagnostic bundles | Support/debug leaks credentials | Redacted bundles, denylist/allowlist, secret scanning before output |
| Shared machine with personal/prod services | CI job can read or attack unrelated workloads | Dedicated runner host only |
| Broad `runs-on: self-hosted` | Sensitive workflows route to unintended runner | Stable RunnerKit labels and workflow readiness checks |
| Persistent caches/workspaces | Cross-job data leakage | Cleanup policies or ephemeral runners |

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
| ------- | ----------- | --------------- |
| Asking users to understand runner architecture up front | Abandonment before first success | Opinionated profiles: cheap persistent, safer ephemeral, BYO |
| Hiding security tradeoffs until docs | Users make unsafe defaults | Inline warnings triggered by repo visibility/mode |
| Declaring success before a runner is online | First workflow still fails/queues | Verify GitHub online status and labels before success |
| Not printing workflow snippet | Users guess wrong `runs-on` labels | Always print copy-paste YAML and `runnerkit status` labels |
| Generic errors | Users return to manual SSH troubleshooting | Diagnose exact phase and next action |
| Cleanup buried in docs | Surprise bills and stale registrations | Print cleanup command at setup end and in status |
| Too many cloud/provider choices | Decision fatigue | One recommended low-cost path plus BYO |
| Ephemeral/persistent jargon | Users choose wrong mode | Explain as "cheapest for trusted private repos" vs "safer clean machine per job" |
| Debug commands leak secrets | Users cannot safely ask for help | Redacted support bundle by default |
| Reinstall creates duplicates | Confusing UI and queued jobs | Detect existing RunnerKit runners and offer adopt/repair/replace |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Registration:** Runner appears in GitHub, but setup did not verify it is online and label-matchable by a workflow.
- [ ] **Auth:** CLI can create a registration token, but stores durable GitHub credentials or sends them to the runner host.
- [ ] **BYO install:** Service starts once, but does not survive reboot or run as a dedicated unprivileged user.
- [ ] **Cloud provisioning:** VM starts, but resources are not tagged and `destroy` cannot clean up if local state is missing.
- [ ] **Persistent mode:** Jobs run, but no workspace/Docker/disk contamination checks exist.
- [ ] **Ephemeral mode:** Runner uses `--ephemeral`, but VM cleanup, log preservation, and no-job/cancel timeouts are missing.
- [ ] **Status:** CLI checks local systemd only, but not GitHub online/offline state, labels, or provider resource state.
- [ ] **Doctor:** Detects failures, but cannot explain exact remediation or redact logs.
- [ ] **Cleanup:** Removes local files, but leaves GitHub runner records or cloud disks/IPs behind.
- [ ] **Upgrade:** Updates the CLI, but not runner binaries/service templates/state schema safely.
- [ ] **Security:** Warns in docs, but setup does not gate public repos, root services, Docker socket, or broad PATs.
- [ ] **Cost:** Claims cheaper CI, but does not show estimated cost, idle duration, or cleanup command.

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
| ------- | ------------- | -------------- |
| Public/untrusted job ran on persistent runner | HIGH | Assume host compromise; stop runner, revoke GitHub/cloud/SSH credentials reachable from host, destroy/rebuild VM, rotate repo secrets, inspect workflow/audit history |
| Registration token expired mid-install | LOW | Request fresh registration token, rerun idempotent bootstrap from registration step, confirm old token was not logged |
| Token leaked in logs/state | MEDIUM | Revoke durable token if applicable; wait/expire short-lived token; delete/redact logs; fix redaction tests before retry |
| Persistent runner contamination | MEDIUM | Stop service, archive/redact diagnostics if needed, wipe `_work` and transient caches/containers, rerun setup validation, consider ephemeral mode |
| Runner service runs as root | MEDIUM | Stop service, create dedicated user, fix ownership/permissions, reinstall service under user, rotate secrets exposed to root-run jobs if untrusted |
| Stale/offline GitHub runner | LOW | Compare local/GitHub state; restart service if host exists; otherwise delete stale registration via GitHub API; recreate with stable labels |
| Cloud resource orphan | MEDIUM | Use provider tags/state to list resources; destroy VM/disks/IPs; delete GitHub registration; add or repair state mapping |
| Failed SSH/bootstrap partial install | LOW/MEDIUM | Run preflight, collect remote logs, rerun idempotent bootstrap; if image unsupported, destroy and recreate with supported image |
| Update broke runner | MEDIUM | Roll back service/template if possible, reinstall supported runner version, restore state backup, verify online/labels with smoke job |
| Ephemeral VM did not clean up | MEDIUM/HIGH | Destroy provider resource, remove GitHub runner, collect remaining logs if possible, add TTL/finalizer before re-enabling ephemeral mode |
| Secret found in support bundle | HIGH | Delete bundle, rotate secret, improve redaction denylist/tests, regenerate sanitized bundle |
| Label mismatch queued jobs | LOW | Show actual runner labels, update workflow `runs-on` snippet or re-register labels, add readiness validation |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
| ------- | ---------------- | ------------ |
| Public/untrusted workflows on persistent runners | Phase 1: GitHub registration + mode selection | Public repo setup is blocked or requires explicit risk override; persistent mode copy explains trusted-only usage |
| Registration/removal token lifecycle | Phase 1: GitHub auth + registration | Tokens are never persisted; retries re-request; logs/state redaction tests pass |
| Over-broad GitHub auth | Phase 1: GitHub auth + registration | Auth flow documents minimum scopes; durable token never reaches runner host |
| Unsafe/generic labels | Phase 1: Registration + workflow snippet | `runnerkit status` prints stable labels and a workflow snippet; queued-job label test passes |
| Secret leakage in CLI/state/logs | Phase 1: CLI foundation; Phase 2 diagnostics | Secret redaction tests cover setup, debug logs, doctor bundle, remote logs |
| Root/excessive host privileges | Phase 2: BYO bootstrap + service management | Service runs as dedicated user; `doctor` fails root runner by default |
| Persistent contamination | Phase 2: BYO service management; Phase 4 ephemeral | `doctor` reports workspace/Docker/disk residue; cleanup or ephemeral path available |
| Split-brain stale/offline runners | Phase 2: status/doctor; Phase 3 cloud reconciliation | Status compares local service, GitHub API, labels, and provider resources |
| SSH/provisioning fragility | Phase 2 BYO; Phase 3 cloud | Preflight covers OS/arch/network/systemd/disk; rerun after partial failure is safe |
| Cloud cost leaks/orphans | Phase 3: cloud provisioning + destroy | All resources tagged; cost shown; destroy works with missing/partial local state |
| Cleanup partial failures | Phase 2 BYO cleanup; Phase 3 cloud destroy | Cleanup plan can be retried and removes GitHub + host + provider resources |
| Naive ephemeral mode | Phase 4: ephemeral lifecycle | One job per runner; logs preserved; max TTL and cleanup finalizer verified under cancellation |
| Update/upgrade drift | Phase 5: lifecycle/upgrades | Version inventory shown; state migration tests; post-upgrade online/label smoke check |
| Poor diagnostics | Phase 2: status/logs/doctor, extended each phase | Doctor gives actionable category findings and redacted support bundle |

## Sources

- `.planning/PROJECT.md` project context: RunnerKit targets solo developers, GitHub Actions, CLI-only setup, BYO and one low-cost cloud path, persistent/ephemeral choice, status/recovery/cleanup, and 10-minute first success.
- `.planning/research/FEATURES.md` feature research: table stakes include GitHub registration/deregistration, BYO/cloud bootstrap, status/logs/doctor, cleanup/destroy, token safety, persistent/ephemeral profiles, and update/upgrade.
- GitHub Docs, self-hosted runner security guidance: self-hosted runners are recommended for private repositories; public/forked workflows can run dangerous code on the runner machine.
- GitHub Docs, REST API for self-hosted runners: registration and removal token endpoints return short-lived tokens with expiration metadata.
- GitHub Docs, autoscaling/ephemeral self-hosted runners: ephemeral runners are intended for one job and require external lifecycle/log handling around runner shutdown.
- GitHub Docs, self-hosted runner application behavior: runner services, labels, runner status, and runner software updates are part of the operational lifecycle.
- Common operational failure modes from self-hosted CI systems: stale registrations, cloud orphan resources, SSH/cloud-init races, root/Docker privilege escalation, and diagnostic gaps.

---
*Pitfalls research for: CLI-first self-hosted GitHub Actions runner setup and management for solo developers*
*Researched: 2026-04-28*
