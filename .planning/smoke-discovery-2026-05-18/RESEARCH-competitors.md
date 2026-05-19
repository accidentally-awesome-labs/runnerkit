# Research: GitHub Actions self-hosted runner — direct competitor BYO patterns

## Methodology
- **Date:** 2026-05-18
- **Approach:** WebSearch + `curl raw.githubusercontent.com` for primary-source READMEs (WebFetch was denied)
- **Primary sources confirmed:** RunsOn README, Actuated README, ARC README, Fireactions README (all fetched raw). Other competitors verified via WebSearch snippets that quote official docs.

## Per-competitor analysis

### Actuated
- **What:** Firecracker microVM orchestrator for GHA on BYO bare-metal / nested-virt cloud VMs.
- **BYO install:** Run **on the host as root**: `curl -LSsf https://get.actuated.com | LICENSE="" TOKEN="" HOME="/root" sudo -E bash -`. Then `sudo -E agent install-service ...`. No maintainer-laptop SSH; user runs it locally. TTY not required (env-driven).
- **Lifecycle reach:** **Agent dial-out** over HTTPS to Actuated SaaS control plane. Behind NAT? Use `inlets` reverse tunnel, not inbound SSH.
- **Privilege model:** Full root daemon (or `--user` mode with sudo-NOPASSWD).
- **Relevance:** Dominant industry pattern — root agent dialing out. They don't try to scope sudo; they own the box.
- **Sources:** docs.actuated.com/install-agent, /provision-server, /expose-agent; github.com/self-actuated/actuated

### RunsOn
- **What:** CloudFormation-deployed ephemeral EC2 runners in user's own AWS account. AWS-only.
- **BYO install:** No BYO-host. Install = CloudFormation template + private GitHub App, deployed by AWS-account owner from the console. ~10 min. No SSH, no per-host sudo.
- **Lifecycle reach:** Control plane runs **inside** user's AWS as Lambda+SQS+S3. Each job = fresh EC2 from public AMI (or BYOI), cloud-init/user-data registers ephemeral runner via JIT token.
- **Privilege model:** AWS IAM, not Linux sudo. Per-job VMs are throwaway.
- **Relevance:** "No persistent host = no sudo problem." They sidestepped the whole class.
- **Sources:** github.com/runs-on/runs-on (raw README), runs-on.com/architecture, runs-on.com/guides/install

### WarpBuild
- **What:** Managed runner SaaS with optional BYOC (AWS/GCP/Azure VPC).
- **BYO install:** No BYO-host. BYOC = "1-click connect cloud," provisions a "WarpBuild Stack" (VPC, subnets, buckets) — exact mechanism (CloudFormation/Terraform) not publicly documented. Importing existing AWS resources is AWS-only.
- **Lifecycle reach:** SaaS control plane uses cloud IAM role assumption into customer account.
- **Privilege model:** Cloud IAM-scoped, not OS sudo.
- **Relevance:** Same sidestep as RunsOn/Cirun. Sells "data sovereignty" without OS-level pain.
- **Sources:** warpbuild.com/blog/launch-byoc, docs.warpbuild.com/ci/byoc, warpbuild.com/products/ci/byoc/aws

### Ubicloud
- **What:** Open-source AWS alternative (AGPL-3.0, Ruby/Roda). Has managed GHA runners + self-hostable "build your own cloud."
- **BYO install (self-hosted):** Ubicloud control plane via Docker Compose. Per bare-metal host: `docker exec -it ubicloud-app ./demo/cloudify_server`, prompts for hostname/IP. Prereq: user puts SSH pubkey into `/root/.ssh/authorized_keys` on target box.
- **Lifecycle reach:** Persistent **SSH-as-root** from control plane to data-plane hosts. Each VM = Cloud Hypervisor inside Linux namespaces.
- **Privilege model:** **Root SSH access required.** No sudo allowlist.
- **Relevance:** Direct architectural sibling to RunnerKit (control plane SSHes to BYO host). Key difference: Ubicloud externalizes "we need root" as a precondition, not a runtime negotiation.
- **Sources:** github.com/ubicloud/ubicloud README, ubicloud.com/docs/quick-start/build-your-own-cloud, GH discussion #1159

### Cirun.io
- **What:** Multi-cloud (AWS/GCP/Azure/DO/Oracle/OpenStack) ephemeral-VM-per-job orchestrator.
- **BYO install:** (1) add cloud creds in Cirun dashboard, (2) commit `.cirun.yml`. No per-host install. Webhook → cloud API → fresh VM.
- **Lifecycle reach:** SaaS holds cloud creds, calls cloud APIs. No persistent agent.
- **Privilege model:** Cloud-credential scoped. On-prem only via OpenStack — they refuse raw-SSH'd boxes.
- **Relevance:** "Keep hosts ephemeral, no privilege model to negotiate."
- **Sources:** docs.cirun.io, docs.cirun.io/quickstart, docs.cirun.io/reference/yaml

### Namespace (namespace.so)
- **What:** Fully managed CI SaaS, ephemeral Firecracker microVMs on Namespace hardware (AMD EPYC, AmpereOne, Apple M-series).
- **BYO install:** **None publicly documented.** GitHub App → dashboard → "New Profile" → swap `runs-on:` label.
- **Lifecycle/privilege:** N/A (managed).
- **Relevance:** Polar opposite of RunnerKit. Clarifies what RunnerKit is *not* trying to be.
- **Sources:** namespace.so/docs/architecture/github-runners, namespace.so/docs/reference/github-actions/runner-configuration

### BuildJet
- **What:** Managed hosted runner SaaS. Swap `runs-on:` label only.
- **BYO install:** **No public BYO option found.** SaaS-only.
- **Relevance:** Confirms half the "competitor" landscape opted out of BYO entirely.
- **Sources:** buildjet.com, buildjet.com/for-github-actions/docs/getting-started/troubleshooting

### Blacksmith (blacksmith.sh)
- **What:** YC-backed managed SaaS, Firecracker microVMs on AWS bare-metal. Control plane on AWS, metadata in Supabase.
- **BYO install:** **None.** Install GitHub App, swap label.
- **Relevance:** Managed-only. Security narrative is "no access to your repo secrets," not "you own the box."
- **Sources:** docs.blacksmith.sh/introduction/quickstart, blacksmith.sh/security, docs.blacksmith.sh/github-actions-runners/config

### GitHub ARC (Actions Runner Controller)
- **What:** Official Kubernetes operator for autoscaling ephemeral runner pods.
- **BYO install:** Helm chart into K8s. All "BYO" concerns delegated to Kubernetes — no per-Linux-host install.
- **Lifecycle reach:** **Pure dial-out.** Listener pod opens HTTPS long-poll to GitHub Actions service. Job queued → ARC patches CR → ephemeral runner pod spawns → runner also dial-outs via HTTPS long-poll with JIT token. Zero inbound traffic.
- **Privilege model:** K8s RBAC + GitHub App. No host-level sudo because no hosts in the user's mental model.
- **Relevance:** **Cleanest answer to "how does the control plane reach the host?" — never reach it.** Cost: user must already operate K8s.
- **Sources:** github.com/actions/actions-runner-controller (raw README), docs.github.com/en/actions/concepts/runners/actions-runner-controller

### Fireactions (hostinger/fireactions) — bonus
- **What:** Open-source BYOM Firecracker orchestrator. Found while researching; architecturally adjacent to RunnerKit.
- **BYO install:** Single Go binary: `fireactions server` on control host, `fireactions agent` inside VM. YAML pools.
- **Lifecycle:** Server **co-located** with the metal — no maintainer-laptop-SSH layer. Pools pre-create JIT-tokened runners; GitHub `workflow_job` webhook scales pools.
- **Privilege:** Server owns its host (root, runs Firecracker).
- **Relevance:** Even open-source BYO-metal tools collapse the "laptop → SSH → host" architecture by co-locating the orchestrator with the metal.
- **Sources:** github.com/hostinger/fireactions (raw README)

## Patterns observed

1. **No one else uses RunnerKit's scoped-sudoers model.** Every competitor either (a) demands root outright (Actuated, Ubicloud, Fireactions), (b) sidesteps via cloud-API (RunsOn, WarpBuild, Cirun), or (c) delegates to K8s RBAC (ARC). Path-C appears to be a RunnerKit invention.
2. **Agent dial-out beats SSH-in.** Actuated + ARC use outbound HTTPS. Works behind NAT without reverse tunnels. Ubicloud and RunnerKit are the SSH-from-control-plane outliers.
3. **Ephemeral wins.** RunsOn, Cirun, WarpBuild, ARC, Namespace, Blacksmith, Fireactions all run one-job-per-VM/pod. Persistent BYO hosts (RunnerKit's model) are the rare niche — only Actuated and Ubicloud still do persistent.
4. **The install moment is "run one command as root, on the box."** Actuated, Fireactions, the bare GHA runner: all expect the user to be locally on the box (or have full root SSH). None negotiate scoped sudo over non-TTY.
5. **"BYO" has bifurcated:** "Bring Your Own Cloud-account" (RunsOn, WarpBuild, Cirun) is the dominant modern meaning. "Bring Your Own Linux box" (Actuated, Ubicloud, RunnerKit, Fireactions) is the niche where the privilege-model pain lives.

## What RunnerKit should consider adopting

- **Agent dial-out instead of SSH-in.** Borrow ARC/Actuated pattern: a `runnerkit-agent` binary on the host opens outbound HTTPS/WebSocket. Eliminates the entire "non-TTY sudo over SSH" failure class. Matches SEED-001's mental model.
- **Curl-pipe-sudo bootstrap as documented install path.** Actuated's `curl … | sudo -E bash -` is honest about needing root, runs on-box with a TTY, avoids the maintainer-laptop layer for bootstrap. `runnerkit byo-prepare` is already moving here; make it the only supported path.
- **Be honest: "RunnerKit needs root."** Ubicloud says "add our key to `/root/.ssh/authorized_keys`." Actuated says "run as root." This is normal in the space.
- **Ephemeral-per-job for v2.** Persistent systemd is fighting the industry tide. Firecracker-on-Hetzner (Fireactions style) is open-source and well-trodden. SEED-002+ direction.

## What RunnerKit should NOT adopt

- **Kubernetes operator model (ARC).** Solo-dev users don't run K8s.
- **AWS CloudFormation install (RunsOn).** AWS lock-in is antithetical to Hetzner-or-BYO positioning.
- **SaaS-only control plane (Blacksmith/Namespace/BuildJet).** Erases the "your hardware, your control" differentiator.
- **Scoped sudoers allowlists.** No competitor uses this pattern — strong signal. Either go full-root (Actuated/Ubicloud) or full-ephemeral (RunsOn/Cirun/ARC).
