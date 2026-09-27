package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
)

// The release-gate and validation kit (docs/testkit/, scripts/testkit/)
// dispatches its workflows at self-hosted runners in a throwaway private
// repository. These tests pin what keeps it safe to copy: the scripts
// parse, check that the repository is private, and never turn off runner
// updates; the workflows run only when dispatched (or on a schedule, on a
// GitHub-hosted runner) and declare their token permissions.

func testkitFiles(t *testing.T, pattern string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(findRepoRoot(t), "scripts", "testkit", pattern))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("no scripts/testkit/%s files", pattern)
	}
	return files
}

func readTestkitFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestTestkitScriptsParse(t *testing.T) {
	t.Parallel()
	for _, f := range testkitFiles(t, "*.sh") {
		if out, err := exec.Command("bash", "-n", f).CombinedOutput(); err != nil {
			t.Errorf("bash -n %s: %v\n%s", filepath.Base(f), err, out)
		}
	}
}

// TestTestkitScriptsRequirePrivateRepo: every script that talks to the
// GitHub API refuses a public repository before it dispatches anything.
func TestTestkitScriptsRequirePrivateRepo(t *testing.T) {
	t.Parallel()
	for _, f := range testkitFiles(t, "*.sh") {
		name := filepath.Base(f)
		if name == "lib.sh" || name == "host-container.sh" {
			continue
		}
		if !strings.Contains(readTestkitFile(t, f), "rk_require_private_repo") {
			t.Errorf("%s does not call rk_require_private_repo", name)
		}
	}
}

// TestTestkitNeverDisablesRunnerUpdates mirrors
// TestRenderedScriptsNeverDisableUpdate for the kit: no config.sh call may
// pass --disableupdate (CLAUDE.md).
func TestTestkitNeverDisablesRunnerUpdates(t *testing.T) {
	t.Parallel()
	files := append(testkitFiles(t, "*.sh"), testkitFiles(t, filepath.Join("workflows", "*.yml"))...)
	for _, f := range files {
		for _, line := range strings.Split(readTestkitFile(t, f), "\n") {
			if strings.Contains(line, "config.sh") && strings.Contains(line, "--disableupdate") {
				t.Errorf("%s passes --disableupdate to config.sh: %s", filepath.Base(f), strings.TrimSpace(line))
			}
		}
	}
}

// TestTestkitReadsRunnerPin: gate.sh (G9) and autoupdate-probe.sh (U3)
// compare the runner's version with RunnerKit's pin, which lib.sh reads
// from internal/bootstrap/package.go. A change to that file's layout must
// not leave the kit comparing against nothing.
func TestTestkitReadsRunnerPin(t *testing.T) {
	t.Parallel()
	lib := filepath.Join(findRepoRoot(t), "scripts", "testkit", "lib.sh")
	// From another directory: the scripts may be run from anywhere.
	cmd := exec.Command("bash", "-c", `. "$1" && rk_runner_pin`, "bash", lib)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "TMPDIR="+t.TempDir())
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rk_runner_pin: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != bootstrap.RunnerVersion {
		t.Fatalf("rk_runner_pin = %q, want bootstrap.RunnerVersion %q", got, bootstrap.RunnerVersion)
	}
}

// workflowTriggers returns the keys of the top-level `on:` mapping.
func workflowTriggers(content string) []string {
	key := regexp.MustCompile(`^  ([a-z_]+):`)
	var triggers []string
	inOn := false
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "on:") {
			inOn = true
			continue
		}
		if !inOn || strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			break
		}
		if m := key.FindStringSubmatch(line); m != nil {
			triggers = append(triggers, m[1])
		}
	}
	return triggers
}

func TestTestkitWorkflowsRunOnlyWhenDispatched(t *testing.T) {
	t.Parallel()
	runsOn := regexp.MustCompile(`(?m)^\s*runs-on:\s*(.+)$`)
	docs := ""
	for _, f := range []string{"README.md", "release-gate.md", "validations.md"} {
		docs += readTestkitFile(t, filepath.Join(findRepoRoot(t), "docs", "testkit", f))
	}
	for _, f := range testkitFiles(t, filepath.Join("workflows", "*.yml")) {
		name := filepath.Base(f)
		content := readTestkitFile(t, f)
		triggers := workflowTriggers(content)
		if len(triggers) == 0 {
			t.Errorf("%s: no triggers found under on:", name)
		}
		dispatch := false
		for _, trigger := range triggers {
			switch trigger {
			case "workflow_dispatch":
				dispatch = true
			case "schedule":
				// Scheduled jobs run code nobody dispatched: only on
				// GitHub-hosted runners.
				for _, m := range runsOn.FindAllStringSubmatch(content, -1) {
					if strings.TrimSpace(m[1]) != "ubuntu-latest" {
						t.Errorf("%s: scheduled workflow runs on %s; use ubuntu-latest", name, strings.TrimSpace(m[1]))
					}
				}
			default:
				t.Errorf("%s: trigger %q; kit workflows run only when dispatched, never on push or pull requests", name, trigger)
			}
		}
		if !dispatch {
			t.Errorf("%s: no workflow_dispatch trigger", name)
		}
		if !regexp.MustCompile(`(?m)^permissions:`).MatchString(content) {
			t.Errorf("%s: no top-level permissions block", name)
		}
		if !strings.Contains(docs, name) {
			t.Errorf("%s is not mentioned in docs/testkit", name)
		}
	}
}
