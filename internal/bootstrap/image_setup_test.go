package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderImageSetupScriptContainsExpectedSections(t *testing.T) {
	script := RenderImageSetupScript("runnerkit-runner", "")
	for _, section := range []string{
		"Node.js 20.x",
		"nodesource",
		"Python",
		"python3-pip",
		"Go",
		"go.dev",
		"Rust",
		"rustup",
		"Java 17",
		"openjdk-17",
		"Docker CE",
		"docker-ce",
		"usermod -aG docker",
		"Google Chrome",
		"google-chrome",
		"ChromeDriver",
		"Firefox",
		"Geckodriver",
		"GitHub CLI",
		"cmake",
		"ninja-build",
		"zstd",
		"image-setup.json",
	} {
		if !strings.Contains(script, section) {
			t.Errorf("script missing section marker %q", section)
		}
	}
}

func TestRenderImageSetupScriptIdempotencyMarker(t *testing.T) {
	script := RenderImageSetupScript("runnerkit-runner", "")
	if !strings.Contains(script, `WANT_VERSION=`+ImageSetupVersion) {
		t.Fatal("script does not check WANT_VERSION against ImageSetupVersion constant")
	}
	if !strings.Contains(script, "already complete, skipping") {
		t.Fatal("script missing early-exit message for matching version")
	}
}

func TestRenderImageSetupScriptSkipsWhenVersionMatches(t *testing.T) {
	script := RenderImageSetupScript("runnerkit-runner", ImageSetupVersion)
	if !strings.Contains(script, `WANT_VERSION=`+ImageSetupVersion) {
		t.Fatal("script should still contain the version check even when currentVersion matches")
	}
}

func TestRenderImageSetupScriptUsesServiceUser(t *testing.T) {
	script := RenderImageSetupScript("custom-user", "")
	if !strings.Contains(script, "custom-user") {
		t.Fatal("script does not reference the provided service user")
	}
	if strings.Contains(script, DefaultServiceUser) {
		t.Fatal("script should use custom-user, not default service user")
	}
}

func TestRenderImageSetupScriptDefaultServiceUser(t *testing.T) {
	script := RenderImageSetupScript("", "")
	if !strings.Contains(script, DefaultServiceUser) {
		t.Fatal("empty serviceUser should fall back to DefaultServiceUser")
	}
}

func TestBaselinePackagesScaleNoDuplicates(t *testing.T) {
	if len(BaselinePackages) < 70 {
		t.Fatalf("BaselinePackages has %d entries, expected ~75 for GitHub runner parity", len(BaselinePackages))
	}
	seen := map[string]bool{}
	for _, pkg := range BaselinePackages {
		if seen[pkg] {
			t.Fatalf("duplicate baseline package: %s", pkg)
		}
		seen[pkg] = true
	}
}

func TestMergePackagesCloudProvisionedSkipsBaseline(t *testing.T) {
	merged := mergePackages([]string{"curl"}, []string{"extra-pkg"}, true)
	for _, bp := range BaselinePackages {
		for _, m := range merged {
			if m == bp && m != "curl" {
				t.Fatalf("cloud-provisioned mergePackages should skip baseline %q but found it", bp)
			}
		}
	}
	found := false
	for _, m := range merged {
		if m == "extra-pkg" {
			found = true
		}
	}
	if !found {
		t.Fatal("extra-pkg missing from cloud-provisioned merge")
	}
}

func TestIsUbuntuLike(t *testing.T) {
	for _, id := range []string{"ubuntu", "debian", "linuxmint"} {
		if !isUbuntuLike(id) {
			t.Errorf("isUbuntuLike(%q) = false, want true", id)
		}
	}
	for _, id := range []string{"fedora", "centos", "arch", ""} {
		if isUbuntuLike(id) {
			t.Errorf("isUbuntuLike(%q) = true, want false", id)
		}
	}
}

// TestImageSetupGeckodriverLookupFailureIsNotFatal runs the rendered
// Geckodriver section in real bash with a curl that fails like a
// rate-limited or blocked api.github.com (exit 22). The section already
// skips when GD_VER is empty, but under set -euo pipefail the failed
// $(curl | grep | head) assignment aborted the whole setup_runner_image
// step first (seen in the v1.3.4 local BYO e2e run: RKFAIL on GD_VER=).
func TestImageSetupGeckodriverLookupFailureIsNotFatal(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	script := RenderImageSetupScript("runnerkit-runner", "")
	start := strings.Index(script, "# ── Geckodriver ──")
	end := strings.Index(script, "# ── GitHub CLI (gh) ──")
	if start < 0 || end < start {
		t.Fatalf("Geckodriver section not found in rendered script")
	}
	section := script[start:end]

	bin := t.TempDir()
	for name, body := range map[string]string{
		"curl": "#!/bin/sh\necho 'curl: (22) The requested URL returned error: 403' >&2\nexit 22\n",
		"sudo": "#!/bin/sh\necho unexpected sudo \"$@\" >&2\nexit 99\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// bash, grep and head from the system; no geckodriver on PATH.
	for _, tool := range []string{"grep", "head"} {
		p, err := exec.LookPath(tool)
		if err != nil {
			t.Skipf("%s not available", tool)
		}
		if err := os.Symlink(p, filepath.Join(bin, tool)); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("bash", "-s")
	cmd.Env = []string{"PATH=" + bin}
	cmd.Stdin = strings.NewReader(withFailTrap("set -euo pipefail\n" + section + "echo section-done\n"))
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Geckodriver section aborted on a failed version lookup: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "section-done") || strings.Contains(stderr.String(), FailTrapMarker) {
		t.Fatalf("section did not complete cleanly: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
