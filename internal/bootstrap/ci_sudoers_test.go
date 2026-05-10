package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRunnerCISudoersEntry(t *testing.T) {
	got := RenderRunnerCISudoersEntry("runnerkit-runner")
	if !strings.Contains(got, RunnerCISudoersFilePath+" (managed by runnerkit byo-prepare --grant-ci-sudo)") {
		t.Fatalf("missing managed-by header:\n%s", got)
	}
	if !strings.Contains(got, "runnerkit-runner ALL=(root) NOPASSWD:") {
		t.Fatalf("missing service user line:\n%s", got)
	}
	for _, path := range []string{
		"/usr/bin/apt-get",
		"/usr/bin/dnf",
		"/usr/sbin/zypper",
		"/sbin/apk",
	} {
		if !strings.Contains(got, path) {
			t.Fatalf("missing allowlisted path %q:\n%s", path, got)
		}
	}
	for _, forbidden := range []string{"ALL=(ALL) NOPASSWD: ALL", "ALL: ALL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("must not contain blanket NOPASSWD %q:\n%s", forbidden, got)
		}
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("sudoers entry must end with newline:\n%q", got)
	}
}

func TestRemoteRunnerCIVisudoCheckScript_RunsVisudoBeforeMv(t *testing.T) {
	script := RemoteRunnerCIVisudoCheckScript()
	v := strings.Index(script, "visudo -cf")
	m := strings.Index(script, `sudo mv "$TMP"`)
	if v < 0 || m < 0 || v >= m {
		t.Fatalf("visudo -cf must precede mv (lockout-prevention): v=%d m=%d\n%s", v, m, script)
	}
	if !strings.Contains(script, "sudo mktemp ") {
		t.Fatalf("mktemp must use sudo for root-owned tempfile:\n%s", script)
	}
}

func TestVisudoValidates_GoodRunnerCISudoersPasses(t *testing.T) {
	visudoPath, err := exec.LookPath("visudo")
	if err != nil {
		t.Skipf("visudo not available: %v", err)
	}
	tmp := filepath.Join(t.TempDir(), "ci-good")
	content := RenderRunnerCISudoersEntry("runnerkit-runner")
	if err := os.WriteFile(tmp, []byte(content), 0440); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(visudoPath, "-cf", tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("visudo rejected CI sudoers: %v\nOutput: %s\n%s", err, out, content)
	}
}
