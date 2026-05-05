package bootstrap

import (
	"fmt"
	"path/filepath"
	"strings"
)

const DefaultServiceUser = "runnerkit-runner"

func RenderDependencyFixScript(missing []string) string {
	if len(missing) == 0 {
		return "set -euo pipefail\necho 'RunnerKit dependencies already present'\n"
	}
	return "set -euo pipefail\n" +
		"if command -v apt-get >/dev/null 2>&1; then\n" +
		"  sudo apt-get update\n" +
		"  sudo apt-get install -y " + strings.Join(missing, " ") + "\n" +
		"elif command -v dnf >/dev/null 2>&1; then\n" +
		"  sudo dnf install -y " + strings.Join(missing, " ") + "\n" +
		"elif command -v yum >/dev/null 2>&1; then\n" +
		"  sudo yum install -y " + strings.Join(missing, " ") + "\n" +
		"else\n" +
		"  echo 'Install missing dependencies manually: " + strings.Join(missing, " ") + "' >&2\n" +
		"  exit 20\n" +
		"fi\n"
}

func RenderInstallScript(opts Options) string {
	serviceUser := defaultString(opts.ServiceUser, DefaultServiceUser)
	installPath := defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName))
	workDir := defaultString(opts.WorkDir, filepath.Join("/var/lib/runnerkit/work", opts.RunnerName))
	pkg := opts.Package
	labels := strings.Join(opts.Labels, ",")
	// register_runner: invoke config.sh via `su` from a root sudo context
	// so the host's sudoers needs only (root) NOPASSWD — no (ALL) runas
	// required. Closes Bug 3 from 06-GAP-byo-sudo-handling.md.
	// sudo -u <non-root> would match (ALL) runas, which neither the
	// byo-prepare scoped template nor a typical (root) NOPASSWD: ALL host
	// sudoers covers. See gap doc lines 122-199 for the full rationale.
	return fmt.Sprintf(`set -euo pipefail
id -u %[1]s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %[1]s
sudo install -d -o %[1]s -g %[1]s %[2]s
sudo install -d -o %[1]s -g %[1]s /var/lib/runnerkit
sudo install -d -o %[1]s -g %[1]s %[3]s
cd %[2]s
if [ ! -f %[4]s ]; then
  sudo curl -fL --retry 3 --connect-timeout 10 -o %[4]s %[5]s
fi
printf '%%s  %%s\n' '%[6]s' '%[4]s' | sudo sha256sum -c -
sudo tar xzf %[4]s --skip-old-files
sudo chown -R %[1]s:%[1]s %[2]s %[3]s
sudo su -s /bin/bash - %[1]s -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url %[7]s --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name %[8]s --labels %[9]s --work %[3]s --replace"
`, serviceUser, installPath, workDir, pkg.Filename, pkg.URL, pkg.SHA256, opts.RepoURL, opts.RunnerName, labels)
}

func RenderServiceScript(opts Options) string {
	return "set -euo pipefail\n" +
		"cd " + defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName)) + "\n" +
		"sudo ./svc.sh install runnerkit-runner\n" +
		"sudo ./svc.sh start\n" +
		"sudo ./svc.sh status\n"
}

// RenderEphemeralInstallScript renders the configure step for an
// ephemeral runner: same dependency/user/download preparation as the
// persistent install path, but invokes ./config.sh with --ephemeral
// and the ephemeral runner name/labels/work_dir. The registration
// token is read only from the RUNNERKIT_REGISTRATION_TOKEN env var
// so the script never interpolates a token value.
func RenderEphemeralInstallScript(opts Options) string {
	serviceUser := defaultString(opts.ServiceUser, DefaultServiceUser)
	installPath := defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName))
	workDir := defaultString(opts.WorkDir, filepath.Join("/var/lib/runnerkit/work", opts.RunnerName))
	pkg := opts.Package
	labels := strings.Join(opts.Labels, ",")
	// register_runner: invoke config.sh via `su` from a root sudo context
	// so the host's sudoers needs only (root) NOPASSWD — no (ALL) runas
	// required. Closes Bug 3 from 06-GAP-byo-sudo-handling.md.
	// sudo -u <non-root> would match (ALL) runas, which neither the
	// byo-prepare scoped template nor a typical (root) NOPASSWD: ALL host
	// sudoers covers. See gap doc lines 122-199 for the full rationale.
	return fmt.Sprintf(`set -euo pipefail
id -u %[1]s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %[1]s
sudo install -d -o %[1]s -g %[1]s %[2]s
sudo install -d -o %[1]s -g %[1]s /var/lib/runnerkit
sudo install -d -o %[1]s -g %[1]s %[3]s
cd %[2]s
if [ ! -f %[4]s ]; then
  sudo curl -fL --retry 3 --connect-timeout 10 -o %[4]s %[5]s
fi
printf '%%s  %%s\n' '%[6]s' '%[4]s' | sudo sha256sum -c -
sudo tar xzf %[4]s --skip-old-files
sudo chown -R %[1]s:%[1]s %[2]s %[3]s
sudo su -s /bin/bash - %[1]s -c "RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url %[7]s --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name %[8]s --labels %[9]s --work %[3]s --replace --ephemeral"
`, serviceUser, installPath, workDir, pkg.Filename, pkg.URL, pkg.SHA256, opts.RepoURL, opts.RunnerName, labels)
}

// RenderEphemeralFinalizerScript renders the host-side finalize.sh
// helper that the ephemeral systemd unit's ExecStopPost calls and the
// TTL timer triggers. The script preserves _diag runner/worker logs
// plus a bounded systemd journal excerpt, writes a sentinel state
// file, and best-effort removes local runner credentials. It never
// interpolates registration or removal token values.
func RenderEphemeralFinalizerScript(opts Options) string {
	installPath := defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName))
	logArchive := defaultString(opts.LogArchivePath, "/var/lib/runnerkit/ephemeral/"+opts.RunnerName+"/logs")
	stateFile := strings.TrimSuffix(logArchive, "/logs") + "/state.json"
	finalizerPath := defaultString(opts.FinalizerPath, "/usr/local/lib/runnerkit/ephemeral/"+opts.RunnerName+"/finalize.sh")
	serviceName := defaultString(opts.EphemeralServiceName, "runnerkit-ephemeral."+opts.RunnerName+".service")
	finalizerDir := strings.TrimSuffix(finalizerPath, "/finalize.sh")
	return fmt.Sprintf(`set -euo pipefail
sudo install -d -m 0755 %[1]s
sudo install -d -m 0750 %[2]s
sudo install -d -m 0755 %[3]s
sudo tee %[4]s >/dev/null <<'EOSCRIPT'
#!/bin/bash
set -euo pipefail
status="${1:-completed}"
install_path=%[5]s
log_archive=%[2]s
state_file=%[6]s
service_name=%[7]s
mkdir -p "$log_archive"
# Preserve runner _diag Runner_*.log and Worker_*.log files best-effort.
if [ -d "$install_path/_diag" ]; then
  cp -f "$install_path"/_diag/Runner_*.log "$log_archive"/ 2>/dev/null || true
  cp -f "$install_path"/_diag/Worker_*.log "$log_archive"/ 2>/dev/null || true
fi
# Bounded systemd journal excerpt.
journalctl -u "$service_name" -n 500 --no-pager > "$log_archive"/systemd-journal.log 2>/dev/null || true
# Sentinel state.json with finalizer_status.
cat > "$state_file" <<EOSTATE
{"finalizer_status":"$status","service_name":"$service_name","log_archive":"$log_archive","timestamp":"$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)"}
EOSTATE
# Best-effort removal of local runner credentials. RunnerKit never
# interpolates registration or removal token values into this script.
if [ -d "$install_path" ]; then
  cd "$install_path"
  rm -f .runner .credentials .credentials_rsaparams || true
fi
EOSCRIPT
sudo chmod 0755 %[4]s
`, finalizerDir, logArchive, strings.TrimSuffix(strings.TrimSuffix(stateFile, "/state.json"), "/logs"), finalizerPath, installPath, stateFile, serviceName)
}

// RenderEphemeralServiceScript renders the systemd unit installer for
// the ephemeral one-shot runner. It writes
// /etc/systemd/system/<service> with Restart=no, ExecStart=<run.sh>,
// and ExecStopPost=<finalizer> completed. It must NOT use svc.sh
// install/start.
func RenderEphemeralServiceScript(opts Options) string {
	installPath := defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName))
	serviceUser := defaultString(opts.ServiceUser, DefaultServiceUser)
	serviceName := defaultString(opts.EphemeralServiceName, "runnerkit-ephemeral."+opts.RunnerName+".service")
	finalizerPath := defaultString(opts.FinalizerPath, "/usr/local/lib/runnerkit/ephemeral/"+opts.RunnerName+"/finalize.sh")
	workDir := defaultString(opts.WorkDir, filepath.Join("/var/lib/runnerkit/work", opts.RunnerName))
	unitPath := "/etc/systemd/system/" + serviceName
	return fmt.Sprintf(`set -euo pipefail
sudo tee %[1]s >/dev/null <<'EOUNIT'
[Unit]
Description=RunnerKit ephemeral GitHub Actions runner (%[2]s)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%[3]s
Group=%[3]s
WorkingDirectory=%[4]s
ExecStart=%[4]s/run.sh
ExecStopPost=%[5]s completed
Restart=no
KillMode=process
TimeoutStopSec=120
Environment=RUNNER_ALLOW_RUNASROOT=0
ReadWritePaths=%[4]s %[6]s
EOUNIT
sudo chmod 0644 %[1]s
sudo systemctl daemon-reload
sudo systemctl start %[2]s
`, unitPath, serviceName, serviceUser, installPath, finalizerPath, workDir)
}

// RenderEphemeralTTLTimerScript installs a TTL safeguard timer that
// finalizes the ephemeral runner after 24h if no job has completed.
// It writes the TTL service+timer units, runs daemon-reload, and
// enables/starts the timer.
func RenderEphemeralTTLTimerScript(opts Options) string {
	serviceName := defaultString(opts.EphemeralServiceName, "runnerkit-ephemeral."+opts.RunnerName+".service")
	ttlServiceName := defaultString(opts.EphemeralTTLServiceName, "runnerkit-ephemeral."+opts.RunnerName+".ttl.service")
	ttlTimerName := defaultString(opts.EphemeralTTLTimerName, "runnerkit-ephemeral."+opts.RunnerName+".ttl.timer")
	finalizerPath := defaultString(opts.FinalizerPath, "/usr/local/lib/runnerkit/ephemeral/"+opts.RunnerName+"/finalize.sh")
	ttlServicePath := "/etc/systemd/system/" + ttlServiceName
	ttlTimerPath := "/etc/systemd/system/" + ttlTimerName
	return fmt.Sprintf(`set -euo pipefail
# TTL safeguard: stop the ephemeral runner service and run the finalizer
# with ttl_expired status when no job completes within OnActiveSec.
sudo tee %[1]s >/dev/null <<'EOSVC'
[Unit]
Description=RunnerKit ephemeral runner TTL safeguard (%[2]s)

[Service]
Type=oneshot
ExecStart=/bin/bash -lc 'systemctl stop %[2]s || true; %[3]s ttl_expired'
EOSVC
sudo chmod 0644 %[1]s
sudo tee %[4]s >/dev/null <<'EOTIMER'
[Unit]
Description=RunnerKit ephemeral runner TTL timer (%[2]s)

[Timer]
OnActiveSec=24h
AccuracySec=1m
Unit=%[5]s

[Install]
WantedBy=timers.target
EOTIMER
sudo chmod 0644 %[4]s
sudo systemctl daemon-reload
sudo systemctl enable --now %[6]s
`, ttlServicePath, serviceName, finalizerPath, ttlTimerPath, ttlServiceName, ttlTimerName)
}

// RenderEphemeralLogPreservationScript renders the cleanup-time log
// preservation script used by `runnerkit down` and `runnerkit destroy`
// before they delete remote runner files or cloud resources. It copies
// _diag Runner_/Worker_ logs plus a bounded systemd journal excerpt
// into the ephemeral log archive directory.
func RenderEphemeralLogPreservationScript(installPath string, logArchivePath string, serviceName string) string {
	if installPath == "" {
		installPath = "/opt/actions-runner"
	}
	if logArchivePath == "" {
		logArchivePath = "/var/lib/runnerkit/ephemeral/logs"
	}
	if serviceName == "" {
		serviceName = "runnerkit-ephemeral.service"
	}
	return fmt.Sprintf(`set -euo pipefail
sudo install -d -m 0750 %[1]s
if [ -d %[2]s/_diag ]; then
  sudo cp -f %[2]s/_diag/Runner_*.log %[1]s/ 2>/dev/null || true
  sudo cp -f %[2]s/_diag/Worker_*.log %[1]s/ 2>/dev/null || true
fi
journalctl -u %[3]s -n 500 --no-pager | sudo tee %[1]s/systemd-journal.log >/dev/null 2>&1 || true
`, logArchivePath, installPath, serviceName)
}

func RenderRemoveConfigScript(installPath string, serviceUser string) string {
	serviceUser = defaultString(serviceUser, DefaultServiceUser)
	return "set -euo pipefail\n" +
		"cd " + installPath + "\n" +
		"sudo -u " + serviceUser + " RUNNERKIT_REMOVAL_TOKEN=\"$RUNNERKIT_REMOVAL_TOKEN\" ./config.sh remove --token \"$RUNNERKIT_REMOVAL_TOKEN\"\n"
}

func RenderReconfigureScript(opts Options) string {
	serviceUser := defaultString(opts.ServiceUser, DefaultServiceUser)
	installPath := defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName))
	workDir := defaultString(opts.WorkDir, filepath.Join("/var/lib/runnerkit/work", opts.RunnerName))
	labels := strings.Join(opts.Labels, ",")
	return "set -euo pipefail\n" +
		"cd " + installPath + "\n" +
		"sudo -u " + serviceUser + " RUNNERKIT_REGISTRATION_TOKEN=\"$RUNNERKIT_REGISTRATION_TOKEN\" ./config.sh --unattended --url " + opts.RepoURL + " --token \"$RUNNERKIT_REGISTRATION_TOKEN\" --name " + opts.RunnerName + " --labels " + labels + " --work " + workDir + " --replace\n"
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
