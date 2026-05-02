package errcodes

import (
	"os"
	"strings"
)

const defaultDocsBase = "https://github.com/salar/runnerkit/blob/main/docs"

// URL returns the canonical troubleshooting URL for a Code. Honors
// $RUNNERKIT_DOCS_BASE for static-site hosting (e.g., runnerkit.dev/docs).
//
// GitHub blob URLs need the ".md" suffix BEFORE the anchor:
//
//	https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001
//
// Static site URLs typically strip the suffix:
//
//	https://runnerkit.dev/docs/troubleshooting/auth#rkd-auth-001
//
// Detection rule: if the docs base URL contains "/blob/", we keep the .md
// suffix; otherwise we strip it. This matches the two known hosting modes
// without needing a separate config flag.
func URL(c Code) string {
	base := strings.TrimRight(os.Getenv("RUNNERKIT_DOCS_BASE"), "/")
	if base == "" {
		base = defaultDocsBase
	}
	file := c.File
	if !strings.Contains(base, "/blob/") {
		file = strings.TrimSuffix(file, ".md")
	}
	return base + "/troubleshooting/" + file + "#" + c.Anchor
}

// FormatLine returns "RKD-XXX-NNN: <Title>\nSee: <URL>" — the canonical CLI
// emit shape per D-15. Callers should use this any time a Code is reported.
func FormatLine(c Code) string {
	return c.ID + ": " + c.Title + "\nSee: " + URL(c)
}
