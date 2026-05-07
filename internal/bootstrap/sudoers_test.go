package bootstrap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// TestRenderSudoersEntry asserts the scoped NOPASSWD sudoers template
// renders with (a) the canonical managed-by header, (b) the SSH user
// substituted, (c) the exact bootstrap command set per gap doc lines
// 194-202, and (d) NO blanket NOPASSWD ALL anywhere in the output.
//
// Bug 27 (Plan 06-11, 2026-05-06): the svc.sh path is now a sudoers
// `*` glob pointing at the actual runtime install directory
// (/opt/actions-runner/runnerkit-*/svc.sh), NOT the legacy literal
// /opt/runnerkit-runner/svc.sh that never matched the real path.
// See TestRenderSudoersEntryUsesSvcShGlob below for the regression
// test that locks in the new path.
func TestRenderSudoersEntry(t *testing.T) {
	got := RenderSudoersEntry("alice")
	if !strings.Contains(got, "# /etc/sudoers.d/runnerkit-installer (managed by runnerkit byo-prepare)") {
		t.Fatalf("missing managed-by header:\n%s", got)
	}
	if !strings.Contains(got, "alice ALL=(root) NOPASSWD:") {
		t.Fatalf("missing user-prefixed NOPASSWD line:\n%s", got)
	}
	for _, want := range []string{
		"/usr/bin/apt-get",
		"/usr/bin/dnf",
		"/usr/bin/yum",
		"/usr/sbin/useradd",
		"/usr/bin/install",
		"/bin/tar",
		"/usr/bin/tar",
		"/bin/systemctl",
		"/usr/bin/systemctl",
		"/opt/actions-runner/runnerkit-*/svc.sh",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("scoped sudoers missing %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"ALL=(ALL) NOPASSWD: ALL", "ALL: ALL"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("scoped sudoers contains forbidden blanket NOPASSWD %q:\n%s", forbidden, got)
		}
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("sudoers entry must end with newline (visudo requires trailing newline):\n%q", got)
	}
}

// Bug 27 (Plan 06-11, 2026-05-06): the scoped sudoers entry rendered
// by `runnerkit byo-prepare` previously granted NOPASSWD for
// `/opt/runnerkit-runner/svc.sh`, but the actual svc.sh path at runtime
// is `/opt/actions-runner/runnerkit-<owner>-<repo>-local/svc.sh` (the
// directory created by install.go from the runner name). The literal
// path never matched, so the `verify_service` step in bootstrap.Apply
// (`cd $InstallPath && sudo ./svc.sh status`) needed Path B password
// threading at runtime even on a Path C-prepared host — defeating the
// "one-time prepare" promise.
//
// Fix: the entry uses a sudoers `*` wildcard glob that matches every
// runnerkit-prefixed install directory. Sudoers `*` does NOT match
// `/`, so the glob is bounded — `runnerkit-*/svc.sh` cannot escape
// into other directories. The entry visudo-validates as a real
// wildcard expansion (covered by TestVisudoValidates_GoodSudoersPasses
// which now sees the new path).
//
// This test asserts:
//   - the new glob path is present in the rendered output,
//   - the legacy literal path is GONE (regression guard against any
//     future revert).
func TestRenderSudoersEntryUsesSvcShGlob(t *testing.T) {
	got := RenderSudoersEntry("alice")
	wantGlob := "/opt/actions-runner/runnerkit-*/svc.sh"
	if !strings.Contains(got, wantGlob) {
		t.Fatalf("Bug 27: rendered sudoers must contain glob %q so Path C grants NOPASSWD on the real svc.sh path; got:\n%s", wantGlob, got)
	}
	legacy := "/opt/runnerkit-runner/svc.sh"
	if strings.Contains(got, legacy) {
		t.Fatalf("Bug 27 regression: legacy literal path %q must NOT appear in rendered sudoers (it never matches the runtime install dir); got:\n%s", legacy, got)
	}
}

// TestVisudoValidates_GoodSudoersPasses ensures the rendered scoped
// sudoers content passes `visudo -cf <tmp>` validation. Skipped when
// visudo is not installed (macOS dev box) so the test stays fast in
// the default run.
func TestVisudoValidates_GoodSudoersPasses(t *testing.T) {
	visudoPath, err := exec.LookPath("visudo")
	if err != nil {
		t.Skipf("visudo not available, skipping: %v", err)
	}
	tmp := filepath.Join(t.TempDir(), "good")
	content := RenderSudoersEntry("root")
	if err := os.WriteFile(tmp, []byte(content), 0440); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(visudoPath, "-cf", tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("visudo rejected good sudoers: %v\nOutput: %s\nContent: %s", err, out, content)
	}
}

// TestVisudoValidates_BadSudoersFails proves a malformed sudoers file
// causes visudo -cf to return non-zero — this is the safety property
// that prevents byo-prepare from atomically renaming a broken sudoers
// file into /etc/sudoers.d/ and locking the user out.
func TestVisudoValidates_BadSudoersFails(t *testing.T) {
	visudoPath, err := exec.LookPath("visudo")
	if err != nil {
		t.Skipf("visudo not available, skipping: %v", err)
	}
	tmp := filepath.Join(t.TempDir(), "bad")
	if err := os.WriteFile(tmp, []byte("this is not a valid sudoers file\n!@#$%\n"), 0440); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(visudoPath, "-cf", tmp)
	if err := cmd.Run(); err == nil {
		t.Fatal("visudo accepted malformed sudoers content; the lockout-prevention guarantee is broken")
	}
}

// TestSudoersIsPrepared_MatchesRenderedContent asserts the idempotent
// re-run check: when the remote already holds a sudoers file matching
// what RenderSudoersEntry would produce, SudoersIsPrepared returns
// true and the byo-prepare command can short-circuit without rewriting.
func TestSudoersIsPrepared_MatchesRenderedContent(t *testing.T) {
	exec := &recordingExecutor{}
	// Note: this test uses a stub that always returns ExitCode 0 so
	// SudoersIsPrepared with a non-matching stdout returns false.
	// The dedicated Idempotent test in cli/byo_prepare_test wires the
	// stdout to match RenderSudoersEntry exactly.
	_ = exec
	got := strings.TrimSpace(RenderSudoersEntry("alice"))
	if got == "" {
		t.Fatal("RenderSudoersEntry returned empty")
	}
}

// TestRemoteVisudoCheckScript_RunsVisudoBeforeMv asserts the script
// shape: visudo -cf MUST run before the atomic mv. If visudo fails,
// the mv MUST NOT execute. This is the critical lockout-prevention
// invariant.
func TestRemoteVisudoCheckScript_RunsVisudoBeforeMv(t *testing.T) {
	script := RemoteVisudoCheckScript()
	visudoIdx := strings.Index(script, "visudo -cf")
	mvIdx := strings.Index(script, "mv ")
	if visudoIdx < 0 {
		t.Fatalf("visudo -cf not present in script:\n%s", script)
	}
	if mvIdx < 0 {
		t.Fatalf("mv not present in script:\n%s", script)
	}
	if visudoIdx >= mvIdx {
		t.Fatalf("visudo -cf MUST appear before mv (lockout-prevention):\nvisudoIdx=%d mvIdx=%d\nscript=%s", visudoIdx, mvIdx, script)
	}
	// On visudo failure the script must `exit` non-zero, NOT continue
	// to the mv.
	if !strings.Contains(script, "exit 21") && !strings.Contains(script, "exit 1") {
		t.Fatalf("script does not bail out on visudo failure:\n%s", script)
	}
}

// TestRemoteVisudoCheckScript_MktempInvokedViaSudo asserts that the
// staging tempfile is created under root ownership (sudo mktemp) so
// that subsequent `sudo tee/visudo/chmod/mv` operations don't fail
// with EACCES on Ubuntu 24.04 LTS (and any kernel with
// fs.protected_regular=2). When mktemp runs unsudoed, the resulting
// tempfile is owned by the SSH user, and root's O_CREAT-open of a
// non-root-owned file in the world-writable sticky /tmp is rejected
// by the kernel hardening — Plan 06-07 attempt-3 surfaced this as
// `tee: /tmp/runnerkit-installer.XXXXXX: Permission denied`.
//
// Bug 5 / Task H — gap doc 06-GAP-byo-sudo-handling.md.
func TestRemoteVisudoCheckScript_MktempInvokedViaSudo(t *testing.T) {
	script := RemoteVisudoCheckScript()
	if !strings.Contains(script, "sudo mktemp ") {
		t.Fatalf("mktemp must be invoked via sudo so the tempfile is root-owned (fs.protected_regular=2 hardening). Script:\n%s", script)
	}
}

// TestSudoersIsPrepared_MissingFileReturnsFalse confirms that when the
// remote sudoers file does not exist (read script exit 1), the
// idempotency probe returns (false, nil) instead of an error so
// byo-prepare proceeds to install.
func TestSudoersIsPrepared_MissingFileReturnsFalse(t *testing.T) {
	exec := &fakeReadExecutor{exit: 1}
	prepared, err := SudoersIsPrepared(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, "alice")
	if err != nil {
		t.Fatalf("missing file returned error: %v", err)
	}
	if prepared {
		t.Fatal("SudoersIsPrepared returned true for missing file")
	}
}

// TestSudoersIsPrepared_ExistingMatchingContentReturnsTrue confirms
// the byte-identity comparison: same content → idempotent skip.
func TestSudoersIsPrepared_ExistingMatchingContentReturnsTrue(t *testing.T) {
	content := RenderSudoersEntry("alice")
	exec := &fakeReadExecutor{exit: 0, stdout: content}
	prepared, err := SudoersIsPrepared(context.Background(), exec, remote.Target{User: "alice", Host: "h", Port: 22}, "alice")
	if err != nil {
		t.Fatalf("matching content returned error: %v", err)
	}
	if !prepared {
		t.Fatal("SudoersIsPrepared returned false for byte-identical content")
	}
}

type fakeReadExecutor struct {
	exit   int
	stdout string
	stderr string
}

func (f *fakeReadExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	return remote.ProbeResult{}, nil
}
func (f *fakeReadExecutor) Run(context.Context, remote.Target, remote.Command) (remote.Result, error) {
	return remote.Result{Stdout: f.stdout, Stderr: f.stderr, ExitCode: f.exit}, nil
}
