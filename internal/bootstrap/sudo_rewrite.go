package bootstrap

import "regexp"

// sudoTokenRe matches a standalone `sudo ` invocation: the literal
// "sudo " preceded by a word boundary, so embedded substrings like
// "visudo " (where the preceding char is "i", a word char) are left
// untouched.
var sudoTokenRe = regexp.MustCompile(`\bsudo `)

// RewriteSudoForPasswordPipe rewrites every standalone `sudo `
// invocation in script to `sudo -S ` so that each sudo call accepts
// its password from the pipeline's stdin (the caller piples the
// password into the wrapped script via `printf | { ... }`).
//
// Bug 6 (Plan 06-07 attempt-4, 2026-05-05) — the previous
// implementation used strings.ReplaceAll(script, "sudo ", "sudo -S "),
// which silently mangled "visudo " into "visudo -S " because
// strings.ReplaceAll matches anywhere in the string regardless of
// word boundaries. visudo then rejected -S with
// "visudo: invalid option -- 'S'" and the byo-prepare flow aborted.
// The word-boundary regex preserves "visudo " (preceded by "i", a
// word character → no boundary) while still rewriting standalone
// "sudo " at line starts, after whitespace, and after delimiters
// like ";" or "&&".
func RewriteSudoForPasswordPipe(script string) string {
	return sudoTokenRe.ReplaceAllString(script, "sudo -S ")
}
