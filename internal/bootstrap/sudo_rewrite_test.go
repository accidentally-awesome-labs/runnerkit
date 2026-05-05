package bootstrap

import (
	"strings"
	"testing"
)

// Bug 6 / Task I — gap doc 06-GAP-byo-sudo-handling.md.
//
// Plan 06-07 attempt-4 (2026-05-05) surfaced a sudo-rewrite mangling
// bug in both internal/bootstrap/install.go::wrapSudoCommand and
// internal/cli/byo_prepare.go::runByoPrepareInstall. Both call
// strings.ReplaceAll(script, "sudo ", "sudo -S "), which incorrectly
// matches the trailing 5 characters of "visudo " and rewrites them to
// "visudo -S " — sending an unknown -S option to visudo. Result:
//
//   visudo: invalid option -- 'S'
//
// The fix is to use a word-boundary-anchored rewrite so "visudo " and
// "sudoers" remain untouched while standalone "sudo " invocations are
// upgraded to "sudo -S ".

func TestRewriteSudoForPasswordPipe_StandaloneSudoIsRewritten(t *testing.T) {
	t.Parallel()
	in := "sudo apt-get update\nsudo systemctl restart foo\n"
	out := RewriteSudoForPasswordPipe(in)
	if !strings.Contains(out, "sudo -S apt-get update") {
		t.Fatalf("standalone `sudo apt-get` was not rewritten:\n%s", out)
	}
	if !strings.Contains(out, "sudo -S systemctl restart foo") {
		t.Fatalf("standalone `sudo systemctl` was not rewritten:\n%s", out)
	}
}

func TestRewriteSudoForPasswordPipe_VisudoIsNotMangled(t *testing.T) {
	t.Parallel()
	in := "if ! sudo visudo -cf \"$TMP\"; then\n  exit 21\nfi\n"
	out := RewriteSudoForPasswordPipe(in)
	if strings.Contains(out, "visudo -S") {
		t.Fatalf("`visudo -S` token leaked — naive ReplaceAll mangling not fixed:\n%s", out)
	}
	if !strings.Contains(out, "sudo -S visudo -cf") {
		t.Fatalf("expected `sudo -S visudo -cf`, got:\n%s", out)
	}
}

func TestRewriteSudoForPasswordPipe_SudoersIsNotMangled(t *testing.T) {
	t.Parallel()
	in := "echo \"sudoers entry installed\" >&2\n"
	out := RewriteSudoForPasswordPipe(in)
	if out != in {
		t.Fatalf("`sudoers` substring should not be touched (it is not followed by space):\nin=%q\nout=%q", in, out)
	}
}

func TestRewriteSudoForPasswordPipe_AtStartOfLineAndAfterSemicolon(t *testing.T) {
	t.Parallel()
	in := "sudo a\nfoo; sudo b\n\tsudo c\n"
	out := RewriteSudoForPasswordPipe(in)
	for _, want := range []string{"sudo -S a", "sudo -S b", "sudo -S c"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestRewriteSudoForPasswordPipe_RemoteVisudoCheckScriptHasNoMangle(t *testing.T) {
	t.Parallel()
	rewritten := RewriteSudoForPasswordPipe(RemoteVisudoCheckScript())
	if strings.Contains(rewritten, "visudo -S") {
		t.Fatalf("RemoteVisudoCheckScript mangled by rewrite — visudo received -S:\n%s", rewritten)
	}
	if !strings.Contains(rewritten, "sudo -S mktemp ") {
		t.Fatalf("expected `sudo -S mktemp ` in rewritten script:\n%s", rewritten)
	}
	if !strings.Contains(rewritten, "sudo -S visudo -cf ") {
		t.Fatalf("expected `sudo -S visudo -cf ` in rewritten script:\n%s", rewritten)
	}
}
