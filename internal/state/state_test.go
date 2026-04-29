package state

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
)

func TestStoreSavesVersionedSecretFreeStateAtomically(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 4, 29, 2, 30, 0, 0, time.UTC)
	state := State{Repositories: []RepositoryState{{
		Repo: gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth: AuthReference{Source: "gh", Reference: "gh"},
		Runner: RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"}, Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine: MachineRef{Kind: "placeholder", InstallPath: ""},
		Provider: ProviderRef{Kind: "none", IDs: map[string]string{}},
		Cleanup: CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety: SafetyMetadata{Code: "ok", Allowed: true},
		RunnerKitVersion: "test-version",
		CreatedAt: now,
		UpdatedAt: now,
	}}}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("state file was not written: %v", err)
	}
	if !bytes.Contains(data, []byte(`"schema_version": "1"`)) {
		t.Fatalf("state file missing schema_version 1:\n%s", data)
	}
	for _, forbidden := range []string{"registration_token", "private_key"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("persisted JSON contains forbidden key %q:\n%s", forbidden, data)
		}
	}
	fileInfo, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if runtime.GOOS != "windows" && fileInfo.Mode().Perm() != 0600 {
		t.Fatalf("state file mode = %v, want 0600", fileInfo.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatalf("stat state dir: %v", err)
	}
	if runtime.GOOS != "windows" && dirInfo.Mode().Perm() != 0700 {
		t.Fatalf("state dir mode = %v, want 0700", dirInfo.Mode().Perm())
	}
}

func TestStoreRejectsRawSecretFields(t *testing.T) {
	badJSON := []byte(`{"schema_version":"1","repositories":[],"token":"ghp_raw","private_key":"raw-key","provider_credential":"raw-provider"}`)
	if err := ValidatePersistedJSON(badJSON); err == nil {
		t.Fatal("expected raw secret keys to be rejected")
	}

	store := NewStore(t.TempDir())
	state := State{Repositories: []RepositoryState{{
		Repo: gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth: AuthReference{Source: "env", Reference: "RUNNERKIT_GITHUB_TOKEN"},
		Runner: RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted"}},
		Safety: SafetyMetadata{Code: "ok", Allowed: true},
	}}}
	if err := store.Save(state); err != nil {
		t.Fatalf("safe auth reference should not be treated as a persisted token: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read saved state: %v", err)
	}
	if strings.Contains(string(data), "registration_token") || strings.Contains(string(data), "private_key") {
		t.Fatalf("saved state contains forbidden secret field:\n%s", data)
	}
}

func TestMigrateAcceptsSchemaVersionOne(t *testing.T) {
	migrated, err := Migrate(State{SchemaVersion: SchemaVersion})
	if err != nil {
		t.Fatalf("Migrate v1 returned error: %v", err)
	}
	if migrated.SchemaVersion != SchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", migrated.SchemaVersion, SchemaVersion)
	}
}

func TestProjectConfigPathUsesRunnerKitConfigYAML(t *testing.T) {
	path := ProjectConfigPath("/work/project")
	if !strings.HasSuffix(path, filepath.Join(".runnerkit", "config.yaml")) {
		t.Fatalf("ProjectConfigPath() = %q, want .runnerkit/config.yaml", path)
	}
}
