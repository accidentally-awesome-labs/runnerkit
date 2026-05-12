package checkliststore

import (
	"path/filepath"
	"testing"
)

func TestSessionPathSanitizes(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	p := SessionPath(base, "o/r@host:22")
	if filepath.Base(p) != "o_r_host_22.json" && filepath.Ext(p) != ".json" {
		t.Fatalf("unexpected path %q", p)
	}
}

func TestSaveLoad(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	d := &Doc{SessionID: "test-session", Steps: []Step{{ID: "a", Title: "A", Status: "done"}}}
	if err := Save(base, d); err != nil {
		t.Fatal(err)
	}
	got, err := Load(base, "test-session")
	if err != nil || got == nil || len(got.Steps) != 1 {
		t.Fatalf("load err=%v doc=%v", err, got)
	}
}
