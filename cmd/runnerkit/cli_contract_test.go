package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

// C-03 binary CLI contract (A-01 / A-03): build the real binary once and
// exercise it as a user would. v1.3.3 exited non-zero with an empty stderr
// for every failing row below, and --version printed nothing.

const contractVersion = "9.9.9-test"

var (
	contractBinOnce sync.Once
	contractBinPath string
	contractBinErr  error
	contractBinOut  []byte
)

func TestMain(m *testing.M) {
	code := testsupport.MainWithIsolatedStateDir(m.Run)
	if contractBinPath != "" {
		_ = os.RemoveAll(filepath.Dir(contractBinPath))
	}
	os.Exit(code)
}

func contractBinary(t *testing.T) string {
	t.Helper()
	contractBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "runnerkit-contract-")
		if err != nil {
			contractBinErr = err
			return
		}
		name := "runnerkit"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		contractBinPath = filepath.Join(dir, name)
		goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
		if _, err := os.Stat(goTool); err != nil {
			goTool = "go"
		}
		cmd := exec.Command(goTool, "build", "-o", contractBinPath, "-ldflags", "-X main.version="+contractVersion, ".")
		contractBinOut, contractBinErr = cmd.CombinedOutput()
	})
	if contractBinErr != nil {
		t.Fatalf("build runnerkit: %v\n%s", contractBinErr, contractBinOut)
	}
	return contractBinPath
}

type contractResult struct {
	stdout string
	stderr string
	code   int
}

// runContract runs the binary in an empty CWD with an isolated state dir and
// no update check, then asserts nothing was written into the CWD (P1-3).
func runContract(t *testing.T, args ...string) contractResult {
	t.Helper()
	return runContractIn(t, t.TempDir(), nil, args...)
}

// runContractIn is runContract with a caller-chosen state dir (so a row can
// seed state.json first) and extra environment entries, which override the
// defaults because exec uses the last value of a duplicated key.
func runContractIn(t *testing.T, stateDir string, extraEnv []string, args ...string) contractResult {
	t.Helper()
	bin := contractBinary(t)
	cwd := t.TempDir()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader("")
	cmd.Env = append(os.Environ(),
		"RUNNERKIT_STATE_DIR="+stateDir,
		"RUNNERKIT_NO_UPDATE_NOTIFIER=1",
		"CI=1",
		"NO_COLOR=1",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	entries, readErr := os.ReadDir(cwd)
	if readErr != nil {
		t.Fatalf("read cwd: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("runnerkit %v wrote into the CWD: %v", args, entries)
	}
	return contractResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

func TestCLIContract_ErrorsAreNeverSilent(t *testing.T) {
	rows := []struct {
		args       []string
		wantCode   int
		wantStderr []string
	}{
		{[]string{"bogus"}, 2, []string{`unknown command "bogus"`}},
		{[]string{"stauts"}, 2, []string{`unknown command "stauts"`, "Did you mean this?", "status"}},
		{[]string{"status", "--bogus"}, 2, []string{"invalid_flag", "unknown flag: --bogus", "runnerkit status --help"}},
		{[]string{"status", "--json=maybe"}, 2, []string{"invalid_flag", `invalid argument "maybe"`}},
		{[]string{"register", "--extra-packages", "x"}, 2, []string{"invalid_flag", "unknown flag: --extra-packages"}},
		{[]string{"byo-prepare", "--host", "x"}, 2, []string{"removed in v1.0.8", "release notes"}},
	}
	for _, row := range rows {
		t.Run(strings.Join(row.args, "_"), func(t *testing.T) {
			res := runContract(t, row.args...)
			if res.code == 0 {
				t.Fatalf("expected non-zero exit; stdout=%q stderr=%q", res.stdout, res.stderr)
			}
			if strings.TrimSpace(res.stderr) == "" {
				t.Fatalf("non-zero exit %d with empty stderr (stdout=%q)", res.code, res.stdout)
			}
			if res.code != row.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr=%q", res.code, row.wantCode, res.stderr)
			}
			for _, want := range row.wantStderr {
				if !strings.Contains(res.stderr, want) {
					t.Fatalf("stderr missing %q:\n%s", want, res.stderr)
				}
			}
		})
	}
}

func TestCLIContract_RenderedErrorsPrintOnce(t *testing.T) {
	res := runContract(t, "status", "--bogus")
	if n := strings.Count(res.stderr, "unknown flag: --bogus"); n != 1 {
		t.Fatalf("flag error printed %d times:\n%s", n, res.stderr)
	}
	res = runContract(t, "byo-prepare")
	if n := strings.Count(res.stderr, "removed in v1.0.8"); n != 1 {
		t.Fatalf("tombstone printed %d times:\n%s", n, res.stderr)
	}
	if strings.Contains(res.stderr, "runnerkit: ") {
		t.Fatalf("rendered error echoed again by main:\n%s", res.stderr)
	}
}

func TestCLIContract_JSONErrorEnvelopes(t *testing.T) {
	for _, row := range []struct {
		args []string
		code string
	}{
		{[]string{"--json", "bogus"}, "cli_usage"},
		{[]string{"--json", "status", "--bogus"}, "invalid_flag"},
		{[]string{"--json", "byo-prepare", "--host", "x"}, "command_removed"},
	} {
		res := runContract(t, row.args...)
		if res.code != 2 {
			t.Fatalf("%v: exit %d, want 2", row.args, res.code)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(res.stdout)), &payload); err != nil {
			t.Fatalf("%v: stdout is not a single JSON object: %v\n%s", row.args, err, res.stdout)
		}
		errObj, _ := payload["error"].(map[string]any)
		if payload["ok"] != false || errObj["code"] != row.code {
			t.Fatalf("%v: unexpected envelope %#v", row.args, payload)
		}
	}
}

func TestCLIContract_Version(t *testing.T) {
	res := runContract(t, "--version")
	if res.code != 0 {
		t.Fatalf("--version exit %d; stderr=%q", res.code, res.stderr)
	}
	if res.stdout != "runnerkit "+contractVersion+"\n" {
		t.Fatalf("--version stdout = %q, want %q", res.stdout, "runnerkit "+contractVersion+"\n")
	}
}

// A-03 / C-03: `doctor --ignore` persists config.json to the state dir, never
// the CWD (v1.3.x wrote ./config.json). PATH is emptied and the token unset
// so the row makes no SSH or GitHub calls: ssh and gh are simply not found.
func TestCLIContract_DoctorIgnoreStaysOutOfCWD(t *testing.T) {
	stateDir := t.TempDir()
	if err := state.NewStore(stateDir).Save(testsupport.StateWithRepository(testsupport.HealthyRepositoryState())); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	env := []string{"PATH=" + t.TempDir(), "HOME=" + t.TempDir(), "RUNNERKIT_GITHUB_TOKEN=", "GH_TOKEN=", "GITHUB_TOKEN="}
	res := runContractIn(t, stateDir, env, "doctor", "--repo", testsupport.TestRepoFullName, "--ignore", "host_mem_low")
	if res.code != 0 && strings.TrimSpace(res.stderr) == "" {
		t.Fatalf("doctor exit %d with empty stderr (stdout=%q)", res.code, res.stdout)
	}
	raw, err := os.ReadFile(filepath.Join(stateDir, "config.json"))
	if err != nil {
		t.Fatalf("doctor --ignore did not write config.json to the state dir: %v\nstdout=%s\nstderr=%s", err, res.stdout, res.stderr)
	}
	if !strings.Contains(string(raw), "host_mem_low") {
		t.Fatalf("config.json missing ignored id: %s", raw)
	}
}
