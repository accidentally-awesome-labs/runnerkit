package cli

import (
	"path/filepath"
	"testing"
)

func TestMergeUniqueStrings(t *testing.T) {
	t.Parallel()
	got := mergeUniqueStrings([]string{"a"}, []string{"a", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%v", got)
	}
}

func TestUserConfigRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c := UserConfig{DoctorIgnoreFindingIDs: []string{"runner_version_stale"}}
	if err := SaveUserConfig(dir, c); err != nil {
		t.Fatal(err)
	}
	got, err := LoadUserConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DoctorIgnoreFindingIDs) != 1 {
		t.Fatalf("%+v", got)
	}
	if filepath.Base(userConfigPath(dir)) != "config.json" {
		t.Fatalf("path %s", userConfigPath(dir))
	}
}
