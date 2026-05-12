# RunnerKit v1.2.1 Release Notes

Date: 2026-05-12

## Highlights

- **Documentation:** Clarify that **RunnerKit-provisioned Hetzner** VMs apply the same **scoped** `/etc/sudoers.d/runnerkit-installer` during **cloud-init** (user-data **`runnerkit-cloud-init-v2`**) so SSH bootstrap stays non-interactive. Cross-links in [`docs/cloud-quickstart.md`](docs/cloud-quickstart.md), [`docs/troubleshooting/bootstrap.md`](docs/troubleshooting/bootstrap.md#rkd-boot-015), and maintainer notes in [`CLAUDE.md`](CLAUDE.md).

## Upgrade path

Same CLI and behavior as **v1.2.0** for installs; this tag is primarily for **docs + maintainer memory** alignment with the cloud-init provisioning path already shipped in v1.2.0.
