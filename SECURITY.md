# Security policy

## Reporting a vulnerability

Please report vulnerabilities **privately** through GitHub's private
vulnerability reporting:
<https://github.com/accidentally-awesome-labs/runnerkit/security/advisories/new>
(repository **Security** tab, **Report a vulnerability**).

Do not open a public issue, Discussion or pull request for an undisclosed
vulnerability.

Include the RunnerKit version (`runnerkit --version`), the command you ran,
what you expected, what happened, and whether the issue needs access to the
workstation, the runner host, or a workflow in the repository.

**Response target:** an acknowledgement and first assessment within
**14 days** of the report. The project is maintained on a capped-hours
basis, so fixes may take longer; you will be told the plan and the expected
fix release.

## Supported versions

Only the **latest minor release** receives security fixes. Older minors are
not patched; upgrade to the latest release.

| Version | Supported |
| --- | --- |
| Latest minor (currently 1.3.x) | Yes |
| Anything older | No |

## Known issues

RunnerKit has known security weaknesses that are already public. They are
listed, each with the stage in which it is planned to be fixed, on
[docs/security-posture.md](docs/security-posture.md). That page also has the
steps to revoke what RunnerKit installed on a host.

Issues listed there are **known and disclosed**; reporting them again is not
necessary, but new information about their impact is welcome. The most
important ones:

- the installer sudoers fragment RunnerKit relies on is root-equivalent;
- the runner service user is in the `docker` group, which is root-equivalent
  for every job;
- ephemeral BYO mode is not isolation, and public or untrusted code belongs
  on GitHub-hosted runners.
