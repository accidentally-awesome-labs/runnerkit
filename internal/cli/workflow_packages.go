package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// aptInstallRe matches `apt-get install` and `apt install` lines in
// workflow run blocks, with optional sudo prefix and -y/--yes flags.
// It captures everything after `install` (plus flags) as a single
// group so we can split individual package names out.
var aptInstallRe = regexp.MustCompile(
	`(?:sudo\s+)?apt(?:-get)?\s+install\s+` +
		`(?:-[yq]+\s+|--yes\s+|--quiet\s+|--no-install-recommends\s+|-\S+\s+)*` +
		`(.+)`)

// scanWorkflowExtraPackages reads all .yml and .yaml files under
// <projectRoot>/.github/workflows/ and extracts package names from
// apt-get install / apt install commands found in run: blocks.
//
// Returns a deduplicated, order-stable list of package names suitable
// for --extra-packages. Returns nil (not error) when the workflow dir
// doesn't exist or contains no apt install lines.
func scanWorkflowExtraPackages(projectRoot string) []string {
	workflowDir := filepath.Join(projectRoot, ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var packages []string

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(workflowDir, name))
		if err != nil {
			continue
		}
		for _, pkg := range extractAptPackages(string(data)) {
			if !seen[pkg] && isValidPackageName(pkg) {
				seen[pkg] = true
				packages = append(packages, pkg)
			}
		}
	}
	return packages
}

// extractAptPackages finds all apt-get/apt install commands in raw
// workflow file content and returns the individual package names.
// Handles shell line continuations (trailing backslash).
func extractAptPackages(content string) []string {
	// Join backslash-continued lines before scanning so multi-line
	// apt-get install commands are matched as a single logical line.
	joined := joinContinuations(content)

	var packages []string
	for _, line := range strings.Split(joined, "\n") {
		line = strings.TrimSpace(line)

		matches := aptInstallRe.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}
		for _, token := range strings.Fields(matches[1]) {
			token = strings.TrimSpace(token)
			if strings.HasPrefix(token, "-") {
				continue
			}
			if strings.ContainsAny(token, "|>&;$()") {
				break
			}
			if isValidPackageName(token) {
				packages = append(packages, token)
			}
		}
	}
	return packages
}

// joinContinuations merges lines ending with \ into a single line.
func joinContinuations(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	var buf strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmed, "\\") {
			buf.WriteString(strings.TrimSuffix(trimmed, "\\"))
			buf.WriteByte(' ')
		} else {
			buf.WriteString(line)
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
	}
	return strings.Join(out, "\n")
}
