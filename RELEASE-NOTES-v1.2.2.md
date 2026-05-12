# RunnerKit v1.2.2 Release Notes

Date: 2026-05-12

## Highlights

- **Hetzner cloud readiness:** `cloud.cloudinit.wait` no longer treats `cloud-init status --wait || test -f …/boot-finished` as success. When cloud-init ends in **`status: error`** (for example failed **`runcmd`** / **`visudo`** for `/etc/sudoers.d/runnerkit-installer`), **`boot-finished`** can still exist; the old shortcut exited **0**, SSH bootstrap then failed seconds later with **`sudo: a password is required`**. The wait script now requires **`status: done`** or **`disabled`** (and rejects **`error`**).
- **Cloud preflight:** Hetzner **`waitCloudTargetReady`** runs preflight with **`RequirePasswordlessSudo`**, emitting **`host.privilege.cloud_bootstrap`** as a **failure** when passwordless sudo is still missing — so **`runnerkit up --cloud`** fails at readiness with clear diagnostics instead of only at the apt bootstrap step (BYO already blocked via **`host_install_required`**).

## Docs

- [`docs/cloud-quickstart.md`](docs/cloud-quickstart.md), [`docs/troubleshooting/bootstrap.md`](docs/troubleshooting/bootstrap.md), [`CLAUDE.md`](CLAUDE.md) — aligned with strict cloud-init gating and **`host.privilege.cloud_bootstrap`**.

## Upgrade path

From [docs/upgrade.md](docs/upgrade.md): **`runnerkit upgrade`** for the CLI. Re-provision ephemeral or persistent cloud runners after upgrading so new VMs get the fixed readiness gate.
