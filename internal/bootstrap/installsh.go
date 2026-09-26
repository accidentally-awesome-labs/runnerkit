package bootstrap

//go:generate go run gen_installsh.go -path ../../install.sh

import (
	"fmt"
	"strings"
)

// Markers delimiting the generated sudoers heredoc inside install.sh's
// render_sudoers(). Everything between them is rewritten by
// `go generate ./internal/bootstrap` (gen_installsh.go).
const (
	InstallShSudoersBeginMarker = "# BEGIN runnerkit-sudoers"
	InstallShSudoersEndMarker   = "# END runnerkit-sudoers"
)

// installShUserPlaceholder stands in for the SSH user while the
// template is escaped for the heredoc, then becomes `${u}`, the
// render_sudoers argument.
const installShUserPlaceholder = "RUNNERKIT_INSTALL_SH_USER_PLACEHOLDER"

// InstallShSudoersBlock renders the lines that go between the markers
// in install.sh: an unquoted heredoc that prints exactly
// RenderSudoersEntry(u) for the shell variable ${u}.
//
// P0-1 (v1.3.4): install.sh carried a hand-copied allowlist that fell
// 16 paths behind RenderSudoersEntry, so password-sudo BYO hosts
// prepared by install.sh could not bootstrap. Generating the block from
// the one template keeps them identical; TestInstallShSudoersMatchesTemplate
// runs the shell function and compares the full body.
func InstallShSudoersBlock() (string, error) {
	entry := RenderSudoersEntry(installShUserPlaceholder)
	if strings.Count(entry, installShUserPlaceholder) != 1 {
		return "", fmt.Errorf("sudoers template must reference the user exactly once")
	}
	if !strings.HasSuffix(entry, "\n") {
		return "", fmt.Errorf("sudoers template must end with a newline to round-trip through a heredoc")
	}
	// Unquoted heredoc: only \, $ and ` are special.
	escaped := strings.NewReplacer(`\`, `\\`, `$`, `\$`, "`", "\\`").Replace(entry)
	escaped = strings.Replace(escaped, installShUserPlaceholder, "${u}", 1)
	for _, line := range strings.Split(escaped, "\n") {
		if line == "EOF" {
			return "", fmt.Errorf("sudoers template contains the heredoc delimiter line EOF")
		}
	}
	var b strings.Builder
	b.WriteString("\t" + InstallShSudoersBeginMarker + " (generated from bootstrap.RenderSudoersEntry by `go generate ./internal/bootstrap`; do not edit)\n")
	b.WriteString("\tcat <<EOF\n")
	b.WriteString(escaped)
	b.WriteString("EOF\n")
	b.WriteString("\t" + InstallShSudoersEndMarker + "\n")
	return b.String(), nil
}

// ReplaceInstallShSudoersBlock returns src (install.sh) with the lines
// from the begin marker through the end marker replaced by
// InstallShSudoersBlock. It is idempotent.
func ReplaceInstallShSudoersBlock(src string) (string, error) {
	block, err := InstallShSudoersBlock()
	if err != nil {
		return "", err
	}
	lines := strings.SplitAfter(src, "\n")
	begin, end := -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, InstallShSudoersBeginMarker):
			if begin != -1 {
				return "", fmt.Errorf("install.sh has more than one %q marker", InstallShSudoersBeginMarker)
			}
			begin = i
		case trimmed == InstallShSudoersEndMarker:
			if end != -1 {
				return "", fmt.Errorf("install.sh has more than one %q marker", InstallShSudoersEndMarker)
			}
			end = i
		}
	}
	if begin == -1 || end == -1 || end < begin {
		return "", fmt.Errorf("install.sh must contain %q followed by %q", InstallShSudoersBeginMarker, InstallShSudoersEndMarker)
	}
	return strings.Join(lines[:begin], "") + block + strings.Join(lines[end+1:], ""), nil
}
