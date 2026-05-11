package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
)

func TestInstallShSyntax(t *testing.T) {
	t.Parallel()
	repoRoot := findRepoRoot(t)
	script := filepath.Join(repoRoot, "install.sh")
	out, err := exec.Command("bash", "-n", script).CombinedOutput()
	if err != nil {
		t.Fatalf("bash -n install.sh: %v\n%s", err, out)
	}
}

func TestInstallShSudoersMatchesGoTemplate(t *testing.T) {
	t.Parallel()
	repoRoot := findRepoRoot(t)
	b, err := os.ReadFile(filepath.Join(repoRoot, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	want := bootstrap.RenderSudoersEntry("alice")
	// First line must match so SudoersIsPrepared stays consistent.
	first := strings.Split(strings.TrimSpace(want), "\n")[0]
	if !strings.Contains(got, first) {
		t.Fatalf("install.sh header diverges from bootstrap.RenderSudoersEntry:\nwant line %q\nscript excerpt:\n%s", first, got[:min(400, len(got))])
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Caller failed")
	}
	// file is .../install_sh_test.go at repo root.
	return filepath.Dir(file)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
