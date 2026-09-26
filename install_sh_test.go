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

// TestInstallShSudoersMatchesTemplate runs install.sh's render_sudoers
// in bash and requires its output to be byte-identical to
// bootstrap.RenderSudoersEntry (P0-1). The previous test compared only
// the header line, so install.sh silently fell 16 command paths behind
// the Go template and password-sudo BYO hosts prepared by it could not
// bootstrap. Regenerate with `go generate ./internal/bootstrap`.
func TestInstallShSudoersMatchesTemplate(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	got := renderInstallShSudoers(t, "alice")
	want := bootstrap.RenderSudoersEntry("alice")
	if got != want {
		t.Fatalf("install.sh render_sudoers diverges from bootstrap.RenderSudoersEntry (run `go generate ./internal/bootstrap`)\nmissing from install.sh: %v\nextra in install.sh: %v\n--- install.sh ---\n%s--- template ---\n%s",
			sudoersPathsMissing(want, got), sudoersPathsMissing(got, want), got, want)
	}
}

// TestInstallShSudoersPassesVisudo validates the rendered fragment with
// visudo -cf, as install.sh itself does before installing it.
func TestInstallShSudoersPassesVisudo(t *testing.T) {
	t.Parallel()
	visudo, err := exec.LookPath("visudo")
	if err != nil {
		t.Skip("visudo not available")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	path := filepath.Join(t.TempDir(), "runnerkit-installer")
	if err := os.WriteFile(path, []byte(renderInstallShSudoers(t, "alice")), 0o440); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(visudo, "-cf", path).CombinedOutput(); err != nil {
		t.Fatalf("visudo -cf rejected install.sh sudoers fragment: %v\n%s", err, out)
	}
}

// TestInstallShSudoersBlockIsGenerated fails when install.sh was edited
// by hand between the generator markers (the CI equivalent of
// `go generate ./... && git diff --exit-code`).
func TestInstallShSudoersBlockIsGenerated(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(findRepoRoot(t), "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	regenerated, err := bootstrap.ReplaceInstallShSudoersBlock(string(b))
	if err != nil {
		t.Fatal(err)
	}
	if regenerated != string(b) {
		t.Fatal("install.sh sudoers block is stale; run `go generate ./internal/bootstrap`")
	}
}

// renderInstallShSudoers extracts the render_sudoers function from
// install.sh (the script itself requires root and writes /etc) and
// runs it in bash for user.
func renderInstallShSudoers(t *testing.T, user string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(findRepoRoot(t), "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var fn []string
	in := false
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "render_sudoers() {") {
			in = true
		}
		if in {
			fn = append(fn, line)
			if line == "}" {
				break
			}
		}
	}
	if len(fn) == 0 || fn[len(fn)-1] != "}" {
		t.Fatal("install.sh: could not find a top-level render_sudoers() { ... } function")
	}
	script := "set -euo pipefail\n" + strings.Join(fn, "\n") + "\nrender_sudoers \"$1\"\n"
	cmd := exec.Command("bash", "-c", script, "bash", user)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("render_sudoers %s: %v\n%s", user, err, stderr.String())
	}
	return string(out)
}

// sudoersPathsMissing lists absolute command paths present in a but
// not in b.
func sudoersPathsMissing(a, b string) []string {
	have := map[string]bool{}
	for _, p := range sudoersPaths(b) {
		have[p] = true
	}
	var missing []string
	for _, p := range sudoersPaths(a) {
		if !have[p] {
			missing = append(missing, p)
		}
	}
	return missing
}

func sudoersPaths(s string) []string {
	var paths []string
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, field := range strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ' ' || r == '\\' }) {
			if strings.HasPrefix(field, "/") {
				paths = append(paths, field)
			}
		}
	}
	return paths
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
