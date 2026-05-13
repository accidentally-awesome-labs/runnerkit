package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractAptPackagesBasicLine(t *testing.T) {
	content := `    - run: sudo apt-get install -y libsecret-1-dev dbus-x11 gnome-keyring`
	got := extractAptPackages(content)
	want := []string{"libsecret-1-dev", "dbus-x11", "gnome-keyring"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("extractAptPackages = %v, want %v", got, want)
	}
}

func TestExtractAptPackagesMultipleFlags(t *testing.T) {
	content := `      sudo apt-get install -y --no-install-recommends curl wget git`
	got := extractAptPackages(content)
	want := []string{"curl", "wget", "git"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("extractAptPackages = %v, want %v", got, want)
	}
}

func TestExtractAptPackagesAptWithoutGet(t *testing.T) {
	content := `      sudo apt install -y build-essential`
	got := extractAptPackages(content)
	if len(got) != 1 || got[0] != "build-essential" {
		t.Fatalf("extractAptPackages = %v, want [build-essential]", got)
	}
}

func TestExtractAptPackagesNoSudo(t *testing.T) {
	content := `      apt-get install -y libssl-dev`
	got := extractAptPackages(content)
	if len(got) != 1 || got[0] != "libssl-dev" {
		t.Fatalf("extractAptPackages = %v, want [libssl-dev]", got)
	}
}

func TestExtractAptPackagesMultiLine(t *testing.T) {
	content := `    - run: |
        sudo apt-get update
        sudo apt-get install -y \
          libsecret-1-dev dbus-x11 \
          gnome-keyring libpango1.0-dev`
	got := extractAptPackages(content)
	want := []string{"libsecret-1-dev", "dbus-x11", "gnome-keyring", "libpango1.0-dev"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("extractAptPackages = %v, want %v", got, want)
	}
}

func TestExtractAptPackagesSkipsAptUpdate(t *testing.T) {
	content := `    - run: sudo apt-get update && sudo apt-get install -y curl`
	got := extractAptPackages(content)
	if len(got) != 1 || got[0] != "curl" {
		t.Fatalf("extractAptPackages = %v, want [curl]", got)
	}
}

func TestExtractAptPackagesStopsAtShellOperators(t *testing.T) {
	content := `    - run: sudo apt-get install -y curl && echo done`
	got := extractAptPackages(content)
	if len(got) != 1 || got[0] != "curl" {
		t.Fatalf("extractAptPackages = %v, want [curl]", got)
	}
}

func TestExtractAptPackagesNoMatch(t *testing.T) {
	content := `    - run: echo "hello world"
    - uses: actions/checkout@v4`
	got := extractAptPackages(content)
	if len(got) != 0 {
		t.Fatalf("extractAptPackages = %v, want empty", got)
	}
}

func TestExtractAptPackagesRealWorldWorkflow(t *testing.T) {
	content := `name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install system deps
        run: |
          sudo apt-get update
          sudo apt-get install -y \
            libsecret-1-dev dbus-x11 gnome-keyring \
            libpango1.0-dev libxkbcommon-dev libxkbcommon-x11-dev \
            libfontconfig1-dev libssl-dev
      - name: Build
        run: npm ci && npm run build`
	got := extractAptPackages(content)
	want := []string{
		"libsecret-1-dev", "dbus-x11", "gnome-keyring",
		"libpango1.0-dev", "libxkbcommon-dev", "libxkbcommon-x11-dev",
		"libfontconfig1-dev", "libssl-dev",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("extractAptPackages = %v, want %v", got, want)
	}
}

func TestScanWorkflowExtraPackagesFromDir(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatal(err)
	}
	ci := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: sudo apt-get install -y libsecret-1-dev dbus-x11
      - run: npm test
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(ci), 0644); err != nil {
		t.Fatal(err)
	}
	deploy := `name: Deploy
on: push
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - run: sudo apt-get install -y dbus-x11 gnome-keyring
`
	if err := os.WriteFile(filepath.Join(workflowDir, "deploy.yaml"), []byte(deploy), 0644); err != nil {
		t.Fatal(err)
	}

	got := scanWorkflowExtraPackages(root)
	want := []string{"libsecret-1-dev", "dbus-x11", "gnome-keyring"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("scanWorkflowExtraPackages = %v, want %v", got, want)
	}
}

func TestScanWorkflowExtraPackagesNoWorkflowDir(t *testing.T) {
	got := scanWorkflowExtraPackages(t.TempDir())
	if got != nil {
		t.Fatalf("expected nil for missing workflow dir, got %v", got)
	}
}

func TestScanWorkflowExtraPackagesNoAptLines(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("name: CI\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := scanWorkflowExtraPackages(root)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
