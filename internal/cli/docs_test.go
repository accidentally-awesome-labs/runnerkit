package cli

import (
	"os"
	"strings"
	"testing"
)

func TestBYOQuickstartDocsContainRequiredCopy(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	quickstart, err := os.ReadFile("../../docs/byo-quickstart.md")
	if err != nil {
		t.Fatalf("read docs/byo-quickstart.md: %v", err)
	}
	combined := string(readme) + "\n" + string(quickstart)
	for _, want := range []string{
		"BYO Persistent Runner Quickstart",
		"BYO persistent runner quickstart",
		"docs/byo-quickstart.md",
		"runnerkit up --repo owner/name --host user@host",
		"Persistent self-hosted runners are intended for trusted private repositories",
		"RunnerKit does not edit or commit workflow YAML for you.",
		"runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]",
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("docs missing %q", want)
		}
	}
}
