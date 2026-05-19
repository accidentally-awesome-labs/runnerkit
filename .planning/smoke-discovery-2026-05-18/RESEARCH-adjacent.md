# Research: BYO host install patterns in adjacent runner/agent tooling

## Methodology

- **Date:** 2026-05-18
- **Scope:** How non-GitHub-Actions agent/runner tools install on BYO Linux hosts, with focus on (a) privilege model, (b) direction of long-lived communication, (c) enrollment/token TTL.
- **Sources:** Official vendor docs and install scripts (Buildkite, GitLab, Tailscale, Datadog, AWS SSM, k3s, CircleCI, Jenkins, Drone, Coder, HashiCorp Nomad/Boundary, Twingate, Sentry).

## Per-tool analysis

### Buildkite agent
- **What:** Go agent that polls Buildkite SaaS to run CI jobs on customer hosts.
- **Install (verbatim, Ubuntu):**
  ```
  curl -fsSL https://keys.openpgp.org/vks/v1/by-fingerprint/32A37959C2FA5C3C99EFBC32A79206696452D198 \
    | sudo gpg --dearmor -o /usr/share/keyrings/buildkite-agent-archive-keyring.gpg
  echo "deb [signed-by=/usr/share/keyrings/buildkite-agent-archive-keyring.gpg] https://apt.buildkite.com/buildkite-agent stable main" \
    | sudo tee /etc/apt/sources.list.d/buildkite-agent.list
  sudo apt-get update && sudo apt-get install -y buildkite-agent
  sudo sed -i "s/xxx/<token>/g" /etc/buildkite-agent/buildkite-agent.cfg
  sudo systemctl enable --now buildkite-agent
  ```
- **Privilege:** sudo for apt + systemd; no TTY. Trust anchored on GPG fingerprint — *not* `curl|bash`.
- **Lifecycle reach:** **Agent dials home.** Docs: "Self-hosted Buildkite agents only make outbound HTTPS connections... no need to open any inbound ports."
- **Enroll:** Long-lived **agent token** in cfg. UI tokens never expire; API tokens accept `expires_at` (min 10 min, no max) for rotation. Per-job session/job tokens scoped internally.
- **Borrow:** GPG-pinned apt repo; outbound-only ("no inbound ports"); API token rotation w/ `expires_at`.
- **Avoid:** Default of non-expiring tokens.
- **Sources:** https://buildkite.com/docs/agent/v3/ubuntu , https://buildkite.com/docs/agent/self-hosted/security/network-requirements , https://buildkite.com/docs/agent/v3/tokens

### GitLab Runner
- **What:** Go binary that registers against a GitLab instance and runs CI jobs.
- **Install (verbatim):**
  ```
  curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash
  sudo apt-get install -y gitlab-runner
  sudo gitlab-runner register   # or --non-interactive --url ... --registration-token ... --executor shell
  ```
- **Privilege:** `curl|sudo bash` for repo (PackageCloud script — installs apt key + source). No TTY needed with `--non-interactive`.
- **Lifecycle reach:** **Runner dials home** over long-poll HTTPS.
- **Enroll:** Two-token model — short-lived **registration token** (resettable from UI) traded at `register` for permanent **runner authentication token** in `config.toml`. Newer flow uses runner-auth-tokens directly.
- **Borrow:** Repo-bootstrap script via PackageCloud avoids `curl|bash` of binaries. Two-token model mirrors GitHub's 1h registration token.
- **Avoid:** Interactive prompt as the default.
- **Sources:** https://docs.gitlab.com/runner/install/linux-repository/ , https://packages.gitlab.com/runner/gitlab-runner/install

### Jenkins JNLP agent
- **What:** Java `agent.jar` running on worker, dialing controller over WebSocket/JNLP-4 TLS.
- **Install (verbatim):**
  ```
  curl -sO http://<controller>:8080/jnlpJars/agent.jar
  java -jar agent.jar -url http://<controller>:8080/ -secret @secret-file -name <agent> -workDir /home/jenkins/agent
  ```
- **Privilege:** No sudo required (runs as any user with Java 11+). Bring-your-own JRE. No installer script.
- **Lifecycle reach:** **Agent dials home** (Jenkins confusingly calls this "inbound agent" from controller's perspective).
- **Enroll:** Per-agent **secret** generated at Node creation; embedded in JNLP URL or via `-secret @file`. No TTL. Compromise => never reuse agent name.
- **Borrow:** `-secret @file` indirection (no secret on argv); systemd-wrapped unprivileged user pattern.
- **Avoid:** Manual JAR curl + BYO JRE; forever-secrets.
- **Sources:** https://github.com/jenkinsci/remoting/blob/master/docs/inbound-agent.md , https://www.jenkins.io/blog/2022/12/27/run-jenkins-agent-as-a-service/

### Drone CI runner
- **What:** Container-packaged runner connecting to a Drone server.
- **Install (verbatim):**
  ```
  docker run --detach \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    --env=DRONE_RPC_PROTO=https --env=DRONE_RPC_HOST=drone.company.com \
    --env=DRONE_RPC_SECRET=super-duper-secret \
    --env=DRONE_RUNNER_CAPACITY=2 --env=DRONE_RUNNER_NAME=my-first-runner \
    --publish=3000:3000 --restart=always --name=runner drone/drone-runner-docker:1
  ```
- **Privilege:** sudo only to install Docker; runner is a container.
- **Lifecycle reach:** **Runner dials home** via gRPC/HTTPS.
- **Enroll:** **Pre-shared `DRONE_RPC_SECRET`** (`openssl rand -hex 16`) — same value server + every runner. No rotation story.
- **Borrow:** Env-vars-to-container is low friction; `runnerkit --docker` mode worth considering.
- **Avoid:** Single shared secret with no rotation.
- **Sources:** https://docs.drone.io/runner/docker/installation/linux/

### CircleCI self-hosted runner
- **What:** Resource-class-scoped runner.
- **Install (sketch):**
  ```
  circleci runner resource-class create my-org/linux-x86 "self-hosted linux"
  # write /etc/opt/circleci/launch-agent-config.yaml (owner circleci, mode 0600) with AUTH_TOKEN
  sudo systemctl enable --now circleci.service
  ```
- **Privilege:** sudo for package + systemd. Config 0600 owned by `circleci`.
- **Lifecycle reach:** **Runner dials home** (claims tasks).
- **Enroll:** **Resource-class token** — only allowed to claim tasks for `<namespace>/<resource-class>`. Pipeline opts in via `resource_class:` field.
- **Borrow:** Scope-limited tokens; config-file-owned-by-system-user-mode-0600.
- **Avoid:** Two-step "mint token, paste into config" handoff.
- **Sources:** https://circleci.com/docs/guides/execution-runner/runner-installation-linux/

### Tailscale (gold-standard `curl|sh`)
- **What:** Mesh VPN; `tailscaled` joins a tailnet.
- **Install (verbatim):**
  ```
  curl -fsSL https://tailscale.com/install.sh | sh
  sudo tailscale up                            # interactive (browser)
  sudo tailscale up --auth-key=tskey-auth-...  # unattended
  ```
- **Privilege:** `install.sh` detects distro via `/etc/os-release`+`uname` and dispatches to apt/dnf/yum/zypper/pacman — adds a **signed repo**, doesn't drop binaries. Unattended needs no TTY (auth-key); interactive default needs a browser (device-code style).
- **Lifecycle reach:** **Agent dials home** (controlplane.tailscale.com).
- **Enroll:** Pre-auth keys with **configurable TTL** (default 90d, max 1y), one-time or reusable, ephemeral, tagged. Or interactive device-code login.
- **Borrow:** **This is the model.** Distro-detecting installer that wraps the system package manager; device-code login for humans + auth-key for automation; TTL + one-time-use + tags on keys.
- **Avoid:** Tailscale runs as root with broad netlink — RunnerKit can stay narrower.
- **Sources:** https://tailscale.com/docs/install/linux , https://github.com/tailscale/tailscale/blob/main/scripts/installer.sh , https://tailscale.com/install.sh

### Coder workspace agents
- **What:** Per-workspace agent connecting a remote dev env to Coder control plane.
- **Install:** Embedded in workspace Terraform template; `CODER_AGENT_TOKEN` env var; `coder_agent.main.init_script` provided by API.
- **Privilege:** Runs as workspace user.
- **Lifecycle reach:** **Agent dials home.**
- **Enroll:** Per-workspace token, scoped (`all` / `no_user_data`); rotated when workspace is rebuilt.
- **Borrow:** Per-resource scoped tokens (per repo+host); init-script-returned-by-API pattern.
- **Avoid:** Plain env-var token for long-lived BYO.
- **Sources:** https://coder.com/docs/reference/api/agents , https://registry.terraform.io/providers/coder/coder/latest/docs/resources/agent

### Datadog Agent
- **What:** Host/container monitoring agent.
- **Install (verbatim):**
  ```
  DD_API_KEY=<key> DD_SITE="datadoghq.com" bash -c "$(curl -L https://install.datadoghq.com/scripts/install_script_agent7.sh)"
  ```
- **Privilege:** Script sudos internally; no TTY. Adds signed apt/yum repo, installs `datadog-agent`, writes `/etc/datadog-agent/datadog.yaml`, enables systemd. **Inputs honored only on first install.**
- **Lifecycle reach:** **Agent dials home** (outbound to `*.datadoghq.com`).
- **Enroll:** Long-lived API key in cfg. No installer rotation.
- **Borrow:** Env-var-prefix UX (creds never in URL or script body); "first-install-only" config protection.
- **Avoid:** Long-lived API key without rotation tooling.
- **Sources:** https://github.com/DataDog/agent-linux-install-script , https://install.datadoghq.com/scripts/install_script_agent7.sh

### Sentry Relay
- **What:** On-prem buffer/proxy in front of Sentry SaaS.
- **Install (verbatim):**
  ```
  docker run --rm -it -v "$(pwd)"/config/:/work/.relay/:z ghcr.io/getsentry/relay config init
  docker run --rm -it -v "$(pwd)"/config/:/etc/relay/ -p 3000:3000 ghcr.io/getsentry/relay run --config /etc/relay/
  ```
- **Privilege:** Docker only.
- **Lifecycle reach:** **Relay dials home** to Sentry; app SDKs talk to relay locally.
- **Enroll:** `config init` generates **public/private keypair** + ID in `credentials.json`; public key registered with Sentry on first connect (UI approval).
- **Borrow:** Locally-generated asymmetric keypair + public-key registration — secret never on the wire (strong model for SEED-001).
- **Avoid:** Manual UI approval friction.
- **Sources:** https://docs.sentry.io/product/relay/getting-started/

### AWS SSM Agent (hybrid activation — closest analogue)
- **What:** AWS agent for remote command exec, patching, Session Manager.
- **Install (verbatim, Ubuntu hybrid):**
  ```
  sudo snap install amazon-ssm-agent --classic
  sudo snap stop amazon-ssm-agent
  sudo /snap/amazon-ssm-agent/current/amazon-ssm-agent \
    -register -code "<activation-code>" -id "<activation-id>" -region "<region>"
  sudo snap start amazon-ssm-agent
  ```
- **Privilege:** sudo for snap + register. No TTY.
- **Lifecycle reach:** **Agent dials home** — outbound HTTPS to `ssm.<region>`, `ec2messages.*`, `ssmmessages.*`. **Bidirectional command channel multiplexed over single outbound long-poll.**
- **Enroll:** EC2 path uses IAM instance profile (no token). **Hybrid activation:** code+id pair with **TTL (default 24h)** and instance-count limit, traded at `-register` for a long-lived managed-instance ID + cert.
- **Borrow:** **Closest analogue to RunnerKit.** Short-lived activation token traded once for long-lived per-host identity; bidirectional command channel over outbound long-poll (deletes need for inbound SSH); TTL+use-count on activation.
- **Avoid:** Cloud-IAM dependency — but the *shape* of the activation flow is portable.
- **Sources:** https://docs.aws.amazon.com/systems-manager/latest/userguide/hybrid-multicloud-ssm-agent-install-linux.html , https://docs.aws.amazon.com/systems-manager/latest/userguide/agent-install-ubuntu-64-snap.html

### Twingate Connector
- **What:** Zero-trust connector brokering app access.
- **Install (sketch):**
  ```
  curl -fsSL "https://binaries.twingate.com/connector/setup.sh" \
    | sudo TWINGATE_ACCESS_TOKEN=... TWINGATE_REFRESH_TOKEN=... TWINGATE_URL=https://acme.twingate.com bash
  ```
- **Privilege:** sudo; no TTY; tokens via env.
- **Lifecycle reach:** **Connector dials home.**
- **Enroll:** **Access + refresh token pair**, connector-scoped, OAuth-style rotation. Not shareable across connectors.
- **Borrow:** Refresh-token rotation for long-lived agent credentials.
- **Avoid:** Nothing notable.
- **Sources:** https://www.twingate.com/docs/connectors-on-linux , https://www.twingate.com/docs/deployment-semi-automation

### HashiCorp Nomad / Consul
- **What:** Cluster orchestrators with gossip + Raft.
- **Install:** apt repo `nomad`/`consul` + HCL config (`bootstrap_expect`, `encrypt`, `acl {}`).
- **Privilege:** sudo + systemd; mature cloud-init flows.
- **Lifecycle reach:** Peer **gossip mesh** within cluster boundary.
- **Enroll:** Symmetric gossip key (`operator gossip keyring generate`, rotatable via `keyring use`). **ACL bootstrap token is one-shot** ("cannot redo cleanly").
- **Borrow:** "Bootstrap is a one-time event"; `operator credentials rotate/list` CLI surface.
- **Avoid:** Gossip mesh — overkill for isolated BYO hosts.
- **Sources:** https://developer.hashicorp.com/nomad/docs/configuration/server , https://developer.hashicorp.com/nomad/docs/secure/acl/bootstrap

### HashiCorp Boundary worker (second-closest analogue)
- **What:** Identity-aware proxy; workers broker user-to-target traffic.
- **Install:** HashiCorp apt repo install of `boundary-worker`; HCL config; `auth_storage_path=/var/lib/boundary`.
- **Privilege:** sudo + systemd.
- **Lifecycle reach:** **Worker dials home** to `initial_upstreams` controllers over mTLS.
- **Enroll:** Three explicit flows — **controller-led** (operator runs `boundary workers create controller-led`, gets activation token, pastes into `controller_generated_activation_token`), **worker-led** (worker generates request token to stdout+file, operator pastes back), or **KMS-encrypted**. Certs/keys **auto-rotate every 2 weeks**.
- **Borrow:** **Worker-led** flow for hosts without inbound reach; auto credential rotation on a schedule; explicit choice between 3 enrollment shapes.
- **Avoid:** Cleartext activation token on disk — use OS keyring/KMS.
- **Sources:** https://developer.hashicorp.com/boundary/docs/configuration/worker/worker-configuration , https://developer.hashicorp.com/boundary/docs/install-boundary/configure-workers

### k3s
- **What:** Lightweight Kubernetes.
- **Install (verbatim):**
  ```
  curl -sfL https://get.k3s.io | sh -                                                    # server
  curl -sfL https://get.k3s.io | K3S_URL=https://myserver:6443 K3S_TOKEN=mynodetoken sh - # agent
  ```
- **Privilege:** Script sudos internally; detects systemd vs openrc.
- **Lifecycle reach:** **Agent dials server** (long-lived TLS).
- **Enroll:** Static node token at `/var/lib/rancher/k3s/server/node-token`; rotation requires server restart.
- **Borrow:** Same install script, role switched by env var (`RUNNERKIT_REPO=... curl ... | sh`); init-system auto-detection.
- **Avoid:** Static non-rotatable shared token.
- **Sources:** https://docs.k3s.io/quick-start

### GitHub Actions runner (baseline)
- **What:** RunnerKit's target.
- **Install:** Tarball + `./config.sh --url --token`, `sudo ./svc.sh install`, `sudo ./svc.sh start`.
- **Privilege:** sudo for systemd; no TTY needed for non-interactive `config.sh`.
- **Lifecycle reach:** **Runner dials home** outbound HTTPS only.
- **Enroll:** **Registration token TTL = 1 hour** via `POST /repos/.../actions/runners/registration-token`; exchanged at `config.sh` time for long-lived `.credentials`. Long-lived creds invalidated only via UI/API removal.
- **Sources:** https://docs.github.com/en/rest/actions/self-hosted-runners , https://docs.github.com/en/actions/reference/runners/self-hosted-runners

## Cross-tool patterns

1. **The industry converged on "agent dials home, outbound HTTPS/443".** Every tool surveyed — Buildkite, GitLab, CircleCI, Drone, Datadog, AWS SSM, Tailscale, Twingate, k3s, Boundary, GitHub Actions itself — works this way. RunnerKit's "control plane SSHs in" model is the outlier; it's the root cause of the sudoers-allowlist and TTY-brittleness pain.
2. **Universally two-stage tokens:** short-lived bootstrap credential (registration token / activation code / pre-auth key / hybrid-activation pair, TTL minutes-to-hours) traded once for a long-lived per-host credential (runner credentials / managed-instance cert / Tailscale node key / Twingate access+refresh). Long-lived credentials are either non-expiring (Buildkite, Datadog, Jenkins — bad) or auto-rotated (Boundary 14d, Twingate refresh, Tailscale).
3. **The best `curl|sh` installers wrap the system package manager.** Tailscale, Datadog, k3s, GitLab — they detect `/etc/os-release`, add a **GPG-signed** apt/dnf/yum/zypper/pacman repo, and let the OS handle binary install + future upgrades. None of them just drop binaries.
4. **No tool requires interactive TTY for unattended install.** Even Tailscale supports `--auth-key`. The sudoers/NOPASSWD problem is universally solved by the operator running the installer under `sudo` *once*, not by surfacing narrow allowlist entries to an outside party.
5. **Container-first variants exist for every tool.** Drone is container-first; Buildkite, GitLab, Coder, Sentry ship official images. Lets sophisticated users skip the package layer entirely.

## Concrete ideas for RunnerKit

1. **Pivot the long-lived communication direction** (from AWS SSM Hybrid + Tailscale + Boundary). Drop "laptop SSHs in" for "agent dials home". A small RunnerKit control daemon on the host long-polls a relay (or, less ambitiously, just `runnerkit-agent` long-polling local on-disk state + the GitHub API) and accepts commands. **Deletes the sudoers allowlist entirely** — the #1 SEED-001 motivation.
2. **Tailscale-style installer.sh** — `curl -fsSL https://install.runnerkit.dev | sh` that detects distro, adds a GPG-signed apt/dnf repo for `runnerkit`, installs the package, prints next steps. Preserves apt-driven upgrades and signature verification; gives operators the audit trail they expect; trust anchored on a published GPG fingerprint, not on whatever the URL serves today.
3. **Env-prefixed one-liner for non-interactive use** (from Datadog) — `RUNNERKIT_REPO=owner/repo RUNNERKIT_TOKEN=ghs_... bash -c "$(curl -L https://install.runnerkit.dev)"`. Token never appears in the URL, in the script body, or in shell history's URL portion.
4. **Two-stage tokens with explicit TTL and rotation** (from Boundary + Twingate). Keep using GitHub's 1h registration token as bootstrap; store an opaque per-host RunnerKit credential after that; **rotate on a 14-day schedule** matching Boundary. `runnerkit doctor` surfaces credential age and next rotation.
5. **Worker-led enrollment path as an option** (from Boundary). For paranoid / air-gapped operators: `runnerkit register --print-request` on the host produces an enrollment blob; operator pastes into `runnerkit approve <blob>` on their laptop. No inbound reach needed, no third-party control plane needed.
6. **Resource-scoped tokens** (from CircleCI + Coder). RunnerKit's long-lived host credential should only be authorized for "register and run jobs for (org/repo, host)" — never a broader scope. Limits blast radius of `/etc/runnerkit/credentials` leak.
7. **First-class credential CLI** (from Nomad + Tailscale ACL keys) — `runnerkit operator credentials list / rotate / revoke`. Treat credentials as objects with a CLI surface; don't make operators hunt state directories.
