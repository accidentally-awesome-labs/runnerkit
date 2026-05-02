package state

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
)

// fixtureV1Bytes returns a populated v1 state.json byte payload used across
// migration tests. The bytes are byte-for-byte preserved through the backup
// step, so edits here ripple to TestMigrate_WritesBackupBeforeMutation.
func fixtureV1Bytes(t *testing.T) []byte {
	t.Helper()
	return []byte(`{
  "schema_version": "1",
  "repositories": [
    {
      "repo": {"host":"github.com","owner":"owner","name":"repo","fullName":"owner/repo","full_name":"owner/repo","private":true},
      "auth": {"source":"gh","reference":"gh"},
      "runner": {"name":"runnerkit-owner-repo-local","labels":["self-hosted","runnerkit","runnerkit-owner-repo","linux","x64","persistent"],"mode":"persistent","os":"linux","arch":"x64"},
      "machine": {"kind":"byo-ssh","host_ref":"alice@example.com:22","user":"alice","port":22,"install_path":"/opt/actions-runner/runnerkit-owner-repo-local","work_dir":"/var/lib/runnerkit/work/runnerkit-owner-repo-local","service_name":"actions.runner.runnerkit-owner-repo-local.service"},
      "provider": {"kind":"byo"},
      "cleanup": {"managed_paths":[],"provider_resource_ids":[]},
      "safety": {"code":"ok","allowed":true},
      "runnerkit_version":"test-version",
      "runner_template_version":"2.334.0",
      "service_template_version":"v1",
      "created_at":"2026-04-29T00:00:00Z",
      "updated_at":"2026-04-29T00:00:00Z"
    }
  ]
}`)
}

func writeFixtureV1(t *testing.T, store Store) []byte {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw := fixtureV1Bytes(t)
	if err := os.WriteFile(store.Path(), raw, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return raw
}

func TestMigrate_V1ToV2_ForwardOnly(t *testing.T) {
	// In-memory: parse a v1 state and call Migrate directly. After migration
	// SchemaVersion must be the current SchemaVersion (i.e. "2") and all
	// repository fields must round-trip unchanged.
	now := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	v1 := State{
		SchemaVersion: "1",
		Repositories: []RepositoryState{{
			Repo:                  gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
			Auth:                  AuthReference{Source: "gh", Reference: "gh"},
			Runner:                RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted", "runnerkit"}, Mode: "persistent", OS: "linux", Arch: "x64"},
			Machine:               MachineRef{Kind: "byo-ssh"},
			Provider:              ProviderRef{Kind: "byo"},
			Cleanup:               CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
			Safety:                SafetyMetadata{Code: "ok", Allowed: true},
			RunnerKitVersion:      "test-version",
			RunnerTemplateVersion: "2.334.0",
			CreatedAt:             now,
			UpdatedAt:             now,
		}},
	}

	migrated, err := Migrate(v1)
	if err != nil {
		t.Fatalf("Migrate(v1) returned error: %v", err)
	}
	if migrated.SchemaVersion != SchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", migrated.SchemaVersion, SchemaVersion)
	}
	if SchemaVersion != "2" {
		t.Fatalf("SchemaVersion constant = %q, expected this plan to bump it to \"2\"", SchemaVersion)
	}
	if !reflect.DeepEqual(migrated.Repositories, v1.Repositories) {
		t.Fatalf("Repositories did not round-trip:\n got: %#v\nwant: %#v", migrated.Repositories, v1.Repositories)
	}
}

func TestMigrate_WritesBackupBeforeMutation(t *testing.T) {
	store := NewStore(t.TempDir())
	original := writeFixtureV1(t, store)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load v1 fixture: %v", err)
	}
	if loaded.SchemaVersion != "2" {
		t.Fatalf("Loaded SchemaVersion = %q, want \"2\"", loaded.SchemaVersion)
	}

	// Find the side-by-side backup file.
	dir := filepath.Dir(store.Path())
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var backupName string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, filepath.Base(store.Path())+".backup-v1-") && strings.HasSuffix(name, "Z") {
			backupName = name
			break
		}
	}
	if backupName == "" {
		t.Fatalf("expected sibling backup file matching state.json.backup-v1-*Z; entries: %v", entries)
	}
	backupBytes, err := os.ReadFile(filepath.Join(dir, backupName))
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if !bytes.Equal(backupBytes, original) {
		t.Fatalf("backup bytes differ from original v1 bytes\nbackup:\n%s\noriginal:\n%s", backupBytes, original)
	}

	// New state.json is v2 on disk.
	newRaw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read new state: %v", err)
	}
	if !bytes.Contains(newRaw, []byte(`"schema_version": "2"`)) {
		t.Fatalf("new state.json does not contain schema_version \"2\":\n%s", newRaw)
	}
}

func TestMigrate_RefusesNewerSchema(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := []byte(`{"schema_version":"99","repositories":[]}`)
	if err := os.WriteFile(store.Path(), original, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected error when loading newer schema_version, got nil")
	}
	if !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("expected errors.Is(err, ErrSchemaTooNew); got %v", err)
	}

	// State file unchanged on disk.
	got, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("state.json was mutated by failed Load:\n got: %s\nwant: %s", got, original)
	}

	// No backup file should be written for refuse-newer-schema.
	entries, err := os.ReadDir(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(store.Path())+".backup-") {
			t.Fatalf("unexpected backup file written for refuse-newer-schema path: %s", e.Name())
		}
	}
}

func TestMigrate_Atomic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based atomic guard not portable to windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root makes 0500 dir still writable")
	}
	dir := t.TempDir()
	store := NewStoreAtPath(filepath.Join(dir, "state.json"))
	original := writeFixtureV1(t, store)

	// First call — write backup successfully, then make the dir read-only so
	// the migrated-state save fails. We expect Load to surface the write
	// error AND preserve both the original state.json and the backup file.
	// Strategy: pre-write the backup ourselves to mimic the successful
	// backup step, then chmod the dir 0500 so the atomic save can't rename.
	// (In the real Load flow, the backup is written first; this simulates
	// the failure window between backup write and migrated save.)

	// Run normally first — this exercises the atomic-write helper end-to-end
	// against the existing chmod 0500 directory. We then restore perms and
	// verify the backup file persisted across the failed save.
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatalf("chmod 0500: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	_, loadErr := store.Load()
	if loadErr == nil {
		t.Fatal("expected Load to fail when state dir is read-only, got nil")
	}

	// Restore perms so we can inspect.
	_ = os.Chmod(dir, 0700)

	// Original state.json is preserved (we never replaced it because the
	// migrated save failed).
	got, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("state.json was mutated despite atomic failure")
	}

	// If a backup was attempted before the failed save, it should still
	// be on disk. Either zero backups (if backup itself failed too) or one
	// backup matching the original bytes is acceptable.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "state.json.backup-v1-") {
			continue
		}
		backupBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read backup: %v", err)
		}
		if !bytes.Equal(backupBytes, original) {
			t.Fatalf("backup file %s does not match original bytes", name)
		}
	}
}
