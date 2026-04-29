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
	if x64.Filename != "actions-runner-linux-x64-2.334.0.tar.gz" || x64.SHA256 != "048024cd2c848eb6f14d5646d56c13a4def2ae7ee3ad12122bee960c56f3d271" {
		t.Fatalf("unexpected x64 package: %#v", x64)
	}
	arm64, err := PackageFor("linux", "arm64")
	if err != nil {
		t.Fatalf("PackageFor linux/arm64: %v", err)
	}
	if arm64.Filename != "actions-runner-linux-arm64-2.334.0.tar.gz" || arm64.SHA256 != "f44255bd3e80160eb25f71bc83d06ea025f6908748807a584687b3184759f7e4" {
		t.Fatalf("unexpected arm64 package: %#v", arm64)
	}
}

func TestPackageForUnsupportedNamesSupportedPairs(t *testing.T) {
	_, err := PackageFor("linux", "arm")
	if err == nil || !strings.Contains(err.Error(), "supported packages are linux/x64 and linux/arm64") {
		t.Fatalf("unsupported error = %v", err)
	}
}
