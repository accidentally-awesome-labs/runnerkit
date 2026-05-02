package errcodes

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// docsRoot returns the absolute path to the docs/troubleshooting directory.
// It uses runtime.Caller so the path is robust regardless of which directory
// `go test` is run from.
func docsRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "troubleshooting")
}

// readDocFile reads a docs/troubleshooting/<file> and returns its bytes.
func readDocFile(t *testing.T, file string) []byte {
	t.Helper()
	path := filepath.Join(docsRoot(t), file)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// TestEveryCodeHasDocAnchor walks every Code in Registry and asserts the
// matching docs/troubleshooting/<File> contains the literal anchor
// `<a name="<Anchor>"></a>`. Per CONTEXT D-15, anchors are stable across
// renames; missing anchors break the CLI's `See: <URL>` contract.
func TestEveryCodeHasDocAnchor(t *testing.T) {
	for _, c := range Registry {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			content := readDocFile(t, c.File)
			needle := `<a name="` + c.Anchor + `"></a>`
			if !strings.Contains(string(content), needle) {
				t.Fatalf("docs/troubleshooting/%s missing anchor %q for code %s", c.File, needle, c.ID)
			}
		})
	}
}

// TestCodesAreUnique asserts no two Codes share the same ID, and no two share
// the same (File, Anchor) pair. Both invariants are required: duplicate IDs
// would make CLI emission ambiguous; duplicate anchors would mean two codes
// resolve to the same docs section.
func TestCodesAreUnique(t *testing.T) {
	seenIDs := map[string]Code{}
	seenAnchors := map[string]Code{}
	for _, c := range Registry {
		if prev, dup := seenIDs[c.ID]; dup {
			t.Fatalf("duplicate Code.ID %q: %+v vs %+v", c.ID, prev, c)
		}
		seenIDs[c.ID] = c

		key := c.File + "#" + c.Anchor
		if prev, dup := seenAnchors[key]; dup {
			t.Fatalf("duplicate (File, Anchor) %q: %+v vs %+v", key, prev, c)
		}
		seenAnchors[key] = c
	}
}

// TestURL_RespectsEnvOverride asserts URL() honors the RUNNERKIT_DOCS_BASE
// env override when set, and falls back to the default GitHub blob URL when
// unset. Crucially, the GitHub blob URL keeps the ".md" suffix BEFORE the
// anchor; the static-site URL strips it.
func TestURL_RespectsEnvOverride(t *testing.T) {
	// Override path: static-site hosting — `.md` suffix is stripped.
	t.Setenv("RUNNERKIT_DOCS_BASE", "https://runnerkit.dev/docs")
	got := URL(AuthPublicRepoBlocked)
	want := "https://runnerkit.dev/docs/troubleshooting/auth#rkd-auth-001"
	if got != want {
		t.Fatalf("URL(AuthPublicRepoBlocked) with override = %q, want %q", got, want)
	}

	// Trailing slash on the base must be tolerated.
	t.Setenv("RUNNERKIT_DOCS_BASE", "https://runnerkit.dev/docs/")
	got = URL(BootRunnerVersionStale)
	want = "https://runnerkit.dev/docs/troubleshooting/bootstrap#rkd-boot-002"
	if got != want {
		t.Fatalf("URL(BootRunnerVersionStale) with trailing-slash base = %q, want %q", got, want)
	}

	// Unset path: default GitHub blob URL — `.md` suffix is kept.
	if err := os.Unsetenv("RUNNERKIT_DOCS_BASE"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	got = URL(AuthPublicRepoBlocked)
	want = "https://github.com/salar/runnerkit/blob/main/docs/troubleshooting/auth.md#rkd-auth-001"
	if got != want {
		t.Fatalf("URL(AuthPublicRepoBlocked) default = %q, want %q", got, want)
	}

	// FormatLine sanity: contains both the ID and the URL.
	line := FormatLine(AuthPublicRepoBlocked)
	if !strings.Contains(line, "RKD-AUTH-001:") || !strings.Contains(line, "See: ") || !strings.Contains(line, "rkd-auth-001") {
		t.Fatalf("FormatLine(AuthPublicRepoBlocked) = %q", line)
	}
}

// TestEachComponentHasMinimumOneEntry asserts each of the 6 component files
// (auth.md, ssh.md, bootstrap.md, github.md, provider.md, cleanup.md)
// contains at least one Code from Registry whose File matches.
func TestEachComponentHasMinimumOneEntry(t *testing.T) {
	required := []string{"auth.md", "ssh.md", "bootstrap.md", "github.md", "provider.md", "cleanup.md"}
	for _, file := range required {
		var any bool
		for _, c := range Registry {
			if c.File == file {
				any = true
				break
			}
		}
		if !any {
			t.Fatalf("no Code in Registry with File=%q (D-14: each of the 6 component files must have at least one entry)", file)
		}
		// Sanity: file exists on disk too.
		path := filepath.Join(docsRoot(t), file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("docs/troubleshooting/%s missing on disk: %v", file, err)
		}
	}
}

// TestEntriesFollowSymptomDiagnosisFix asserts every Code's docs section
// contains all three literal headings: `### Symptom`, `### Diagnosis`,
// `### Fix` (D-17). The "section" for a code spans from its `<a name=>`
// anchor down to the next `<a name=` anchor (or EOF).
func TestEntriesFollowSymptomDiagnosisFix(t *testing.T) {
	// Group codes by file so we open each file exactly once.
	byFile := map[string][]Code{}
	for _, c := range Registry {
		byFile[c.File] = append(byFile[c.File], c)
	}
	for file, codes := range byFile {
		content := string(readDocFile(t, file))
		for _, c := range codes {
			c := c
			t.Run(c.ID, func(t *testing.T) {
				start := strings.Index(content, `<a name="`+c.Anchor+`"></a>`)
				if start < 0 {
					t.Fatalf("%s: anchor %q not found", file, c.Anchor)
				}
				rest := content[start:]
				// Skip past this anchor line, then find the next anchor (any).
				skipFirst := len(`<a name="` + c.Anchor + `"></a>`)
				tail := rest[skipFirst:]
				nextAnchor := strings.Index(tail, `<a name="`)
				var section string
				if nextAnchor < 0 {
					section = rest
				} else {
					section = rest[:skipFirst+nextAnchor]
				}
				for _, heading := range []string{"### Symptom", "### Diagnosis", "### Fix"} {
					if !strings.Contains(section, heading) {
						t.Fatalf("%s entry %s: section missing %q heading\nsection:\n%s", file, c.ID, heading, section)
					}
				}
			})
		}
	}
}
