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
	return fmt.Sprintf(`set -euo pipefail
id -u %[1]s >/dev/null 2>&1 || sudo useradd --system --create-home --shell /usr/sbin/nologin %[1]s
sudo install -d -o %[1]s -g %[1]s %[2]s
sudo install -d -o %[1]s -g %[1]s /var/lib/runnerkit
sudo install -d -o %[1]s -g %[1]s %[3]s
cd %[2]s
if [ ! -f %[4]s ]; then
  curl -fL --retry 3 --connect-timeout 10 -o %[4]s %[5]s
fi
printf '%%s  %%s\n' '%[6]s' '%[4]s' | sha256sum -c -
tar xzf %[4]s --skip-old-files
sudo chown -R %[1]s:%[1]s %[2]s %[3]s
sudo -u %[1]s RUNNERKIT_REGISTRATION_TOKEN="$RUNNERKIT_REGISTRATION_TOKEN" ./config.sh --unattended --url %[7]s --token "$RUNNERKIT_REGISTRATION_TOKEN" --name %[8]s --labels %[9]s --work %[3]s --replace
`, serviceUser, installPath, workDir, pkg.Filename, pkg.URL, pkg.SHA256, opts.RepoURL, opts.RunnerName, labels)
}

func RenderServiceScript(opts Options) string {
	return "set -euo pipefail\n" +
		"cd " + defaultString(opts.InstallPath, filepath.Join("/opt/actions-runner", opts.RunnerName)) + "\n" +
		"sudo ./svc.sh install runnerkit-runner\n" +
		"sudo ./svc.sh start\n" +
		"sudo ./svc.sh status\n"
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
