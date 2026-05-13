package bootstrap

import (
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
