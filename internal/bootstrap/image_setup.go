package bootstrap

import "fmt"

// ImageSetupVersion is the marker version written to
// /var/lib/runnerkit/image-setup.json after a successful run. Bump
// this when the script changes materially so upgrade-runner re-runs.
const ImageSetupVersion = "1"

// RenderImageSetupScript returns a shell script that installs
// language runtimes, Docker, browsers, and CLI tools to match the
// GitHub-hosted Ubuntu 24.04 runner image. The script is idempotent:
// each section checks whether the tool is already present and skips
// if so. A marker file gates the entire script when the version
// matches.
//
// serviceUser is the runner service user (e.g. "runnerkit-runner")
// added to the docker group. currentVersion is the ImageSetupVersion
// already on the host (empty if none); when it matches
// ImageSetupVersion the script exits early.
func RenderImageSetupScript(serviceUser, currentVersion string) string {
	if serviceUser == "" {
		serviceUser = DefaultServiceUser
	}
	return fmt.Sprintf(`set -euo pipefail
MARKER=/var/lib/runnerkit/image-setup.json
WANT_VERSION=%[1]s

# Skip if marker shows current version already installed.
if [ -f "$MARKER" ]; then
  HAVE=$(cat "$MARKER" 2>/dev/null | grep -o '"version":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [ "$HAVE" = "$WANT_VERSION" ]; then
    echo "Runner image setup v${WANT_VERSION} already complete, skipping."
    exit 0
  fi
fi

export DEBIAN_FRONTEND=noninteractive

# ── Node.js 20.x LTS via NodeSource ──
if ! command -v node >/dev/null 2>&1; then
  echo "Installing Node.js 20.x..."
  sudo mkdir -p /etc/apt/keyrings
  curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg 2>/dev/null || true
  echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" | sudo tee /etc/apt/sources.list.d/nodesource.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y nodejs
fi

# ── Python extras (pip, venv) ──
if ! command -v pip3 >/dev/null 2>&1; then
  echo "Installing python3-pip and python3-venv..."
  sudo apt-get install -y python3-pip python3-venv
fi

# ── Go (latest stable) ──
if ! command -v go >/dev/null 2>&1; then
  echo "Installing Go..."
  GO_VERSION=$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -1)
  curl -fsSL "https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz
  sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
  sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
fi

# ── Rust (latest stable via rustup, as runner service user) ──
if ! sudo su -s /bin/bash - %[2]s -c "command -v rustc" >/dev/null 2>&1; then
  echo "Installing Rust for %[2]s..."
  sudo su -s /bin/bash - %[2]s -c "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable --profile minimal" || true
fi

# ── Java 17 (default) ──
if ! command -v java >/dev/null 2>&1; then
  echo "Installing OpenJDK 17..."
  sudo apt-get install -y openjdk-17-jdk-headless
fi

# ── .NET SDK ──
if ! command -v dotnet >/dev/null 2>&1; then
  echo "Installing .NET SDK..."
  sudo apt-get install -y dotnet-sdk-8.0 2>/dev/null || {
    curl -fsSL https://packages.microsoft.com/config/ubuntu/24.04/packages-microsoft-prod.deb -o /tmp/ms-prod.deb
    sudo dpkg -i /tmp/ms-prod.deb
    rm -f /tmp/ms-prod.deb
    sudo apt-get update -qq
    sudo apt-get install -y dotnet-sdk-8.0
  }
fi

# ── Docker CE ──
if ! command -v docker >/dev/null 2>&1; then
  echo "Installing Docker CE..."
  sudo mkdir -p /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null || true
  echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi
sudo usermod -aG docker %[2]s 2>/dev/null || true

# ── Google Chrome ──
if ! command -v google-chrome >/dev/null 2>&1; then
  echo "Installing Google Chrome..."
  curl -fsSL https://dl.google.com/linux/linux_signing_key.pub | sudo gpg --dearmor -o /etc/apt/keyrings/google-chrome.gpg 2>/dev/null || true
  echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/google-chrome.gpg] http://dl.google.com/linux/chrome/deb/ stable main" | sudo tee /etc/apt/sources.list.d/google-chrome.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y google-chrome-stable
fi

# ── ChromeDriver (matching Chrome major version) ──
if ! command -v chromedriver >/dev/null 2>&1 && command -v google-chrome >/dev/null 2>&1; then
  echo "Installing ChromeDriver..."
  CHROME_VER=$(google-chrome --version | grep -oP '\d+\.\d+\.\d+\.\d+' | head -1)
  CHROME_MAJOR=$(echo "$CHROME_VER" | cut -d. -f1)
  CD_URL=$(curl -fsSL "https://googlechromelabs.github.io/chrome-for-testing/LATEST_RELEASE_${CHROME_MAJOR}" 2>/dev/null || echo "")
  if [ -n "$CD_URL" ]; then
    curl -fsSL "https://storage.googleapis.com/chrome-for-testing-public/${CD_URL}/linux64/chromedriver-linux64.zip" -o /tmp/chromedriver.zip
    sudo unzip -o /tmp/chromedriver.zip -d /usr/local/share/chromedriver-linux64 >/dev/null
    sudo ln -sf /usr/local/share/chromedriver-linux64/chromedriver-linux64/chromedriver /usr/local/bin/chromedriver
    rm -f /tmp/chromedriver.zip
  fi
fi

# ── Firefox ──
if ! command -v firefox >/dev/null 2>&1; then
  echo "Installing Firefox..."
  sudo apt-get install -y firefox 2>/dev/null || {
    sudo add-apt-repository -y ppa:mozillateam/ppa 2>/dev/null || true
    sudo apt-get update -qq
    sudo apt-get install -y firefox
  }
fi

# ── Geckodriver ──
if ! command -v geckodriver >/dev/null 2>&1; then
  echo "Installing Geckodriver..."
  GD_VER=$(curl -fsSL https://api.github.com/repos/mozilla/geckodriver/releases/latest 2>/dev/null | grep -oP '"tag_name":\s*"v\K[^"]+' | head -1)
  if [ -n "$GD_VER" ]; then
    curl -fsSL "https://github.com/mozilla/geckodriver/releases/download/v${GD_VER}/geckodriver-v${GD_VER}-linux64.tar.gz" -o /tmp/geckodriver.tar.gz
    sudo tar -xzf /tmp/geckodriver.tar.gz -C /usr/local/bin geckodriver
    sudo chmod +x /usr/local/bin/geckodriver
    rm -f /tmp/geckodriver.tar.gz
  fi
fi

# ── GitHub CLI (gh) ──
if ! command -v gh >/dev/null 2>&1; then
  echo "Installing GitHub CLI..."
  sudo mkdir -p /etc/apt/keyrings
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg >/dev/null
  echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y gh
fi

# ── CMake, Ninja, zstd (apt) ──
for pkg in cmake ninja-build zstd; do
  if ! dpkg -s "$pkg" >/dev/null 2>&1; then
    sudo apt-get install -y "$pkg"
  fi
done

# ── Write marker ──
sudo mkdir -p /var/lib/runnerkit
printf '{"version":"%[1]s","timestamp":"%%s"}\n' "$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)" | sudo tee "$MARKER" >/dev/null
echo "Runner image setup v${WANT_VERSION} complete."
`, ImageSetupVersion, serviceUser)
}
