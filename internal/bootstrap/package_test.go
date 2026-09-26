package bootstrap

import (
	"strings"
	"testing"
)

func TestPackageForPinnedLinuxPackages(t *testing.T) {
	x64, err := PackageFor("linux", "x64")
	if err != nil {
		t.Fatalf("PackageFor linux/x64: %v", err)
	}
	if x64.Filename != "actions-runner-linux-x64-2.337.0.tar.gz" || x64.SHA256 != "70920811a4f8ad4328818682bca5c6469c1c942fab52448868071d0063816613" {
		t.Fatalf("unexpected x64 package: %#v", x64)
	}
	arm64, err := PackageFor("linux", "arm64")
	if err != nil {
		t.Fatalf("PackageFor linux/arm64: %v", err)
	}
	if arm64.Filename != "actions-runner-linux-arm64-2.337.0.tar.gz" || arm64.SHA256 != "9b1dc70626422526e3c94767cf024896beb15da5342a3f4819bf2feac13e0393" {
		t.Fatalf("unexpected arm64 package: %#v", arm64)
	}
}

func TestPackageForUnsupportedNamesSupportedPairs(t *testing.T) {
	_, err := PackageFor("linux", "arm")
	if err == nil || !strings.Contains(err.Error(), "supported packages are linux/x64 and linux/arm64") {
		t.Fatalf("unsupported error = %v", err)
	}
}
