package state

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
)

func TestStoreSavesVersionedSecretFreeStateAtomically(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 4, 29, 2, 30, 0, 0, time.UTC)
	state := State{Repositories: []RepositoryState{{
		Repo:             gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth:             AuthReference{Source: "gh", Reference: "gh"},
		Runner:           RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"}, Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine:          MachineRef{Kind: "placeholder", InstallPath: ""},
		Provider:         ProviderRef{Kind: "none", IDs: map[string]string{}},
		Cleanup:          CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety:           SafetyMetadata{Code: "ok", Allowed: true},
		RunnerKitVersion: "test-version",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("state file was not written: %v", err)
	}
	if !bytes.Contains(data, []byte(`"schema_version": "2"`)) {
		t.Fatalf("state file missing schema_version 2:\n%s", data)
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
		Repo:   gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth:   AuthReference{Source: "env", Reference: "RUNNERKIT_GITHUB_TOKEN"},
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

func TestLoadBackwardsCompatibleStateWithoutHostKeyFields(t *testing.T) {
	store := NewStore(t.TempDir())
	json := `{"schema_version":"1","repositories":[{"repo":{"fullName":"owner/repo","full_name":"owner/repo"},"auth":{"source":"gh","reference":"gh"},"runner":{"name":"runnerkit-owner-repo-local","labels":["self-hosted"],"mode":"persistent","os":"linux","arch":"x64"},"machine":{"kind":"phase1-placeholder","host_ref":"alice@example.com"},"provider":{"kind":"none"},"cleanup":{"managed_paths":[],"provider_resource_ids":[]},"safety":{"code":"ok","allowed":true},"runnerkit_version":"test","created_at":"2026-04-29T00:00:00Z","updated_at":"2026-04-29T00:00:00Z"}]}`
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), []byte(json), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Repositories) != 1 || loaded.Repositories[0].Machine.HostKeyFingerprint != "" {
		t.Fatalf("unexpected migrated state: %#v", loaded)
	}
}

func TestMachineRefRoundTripsBYOFields(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 4, 29, 3, 0, 0, 0, time.UTC)
	state := NewState()
	state.Repositories = []RepositoryState{{
		Repo:     gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth:     AuthReference{Source: "gh", Reference: "gh"},
		Runner:   RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted"}, Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine:  MachineRef{Kind: "byo-ssh", HostRef: "alice@example.com:22", User: "alice", Port: 22, HostKeyFingerprint: "SHA256:fakehostfingerprint", InstallPath: "/opt/actions-runner/runnerkit-owner-repo-local", WorkDir: "/var/lib/runnerkit/work/runnerkit-owner-repo-local", ServiceName: "actions.runner.runnerkit-owner-repo-local.service", HostKeyAcceptedAt: &now},
		Provider: ProviderRef{Kind: "byo"},
		Cleanup:  CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety:   SafetyMetadata{Code: "ok", Allowed: true},
	}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	machine := loaded.Repositories[0].Machine
	if machine.Port != 22 || machine.HostKeyFingerprint != "SHA256:fakehostfingerprint" || machine.InstallPath == "" || machine.WorkDir == "" || machine.ServiceName == "" {
		t.Fatalf("MachineRef did not round-trip BYO fields: %#v", machine)
	}
}

func TestSafetyMetadataPersistsSafetyProfile(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 5, 2, 1, 0, 0, 0, time.UTC)
	state := NewState()
	state.Repositories = []RepositoryState{{
		Repo:             gh.Repo{Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false},
		Auth:             AuthReference{Source: "gh", Reference: "gh"},
		Runner:           RunnerIdentity{Name: "runnerkit-owner-name-ephemeral-abc123", Labels: []string{"self-hosted"}, Mode: "ephemeral", OS: "linux", Arch: "x64"},
		Machine:          MachineRef{Kind: "cloud-ssh"},
		Provider:         ProviderRef{Kind: "hetzner"},
		Cleanup:          CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety:           SafetyMetadata{Code: "ok", Allowed: true, SafetyProfile: "ephemeral-cloud"},
		RunnerKitVersion: "test-version",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(data), `"safety_profile": "ephemeral-cloud"`) {
		t.Fatalf("expected safety_profile field in serialized state:\n%s", string(data))
	}

	// Backwards compatible: old state without safety_profile must still load.
	json := `{"schema_version":"1","repositories":[{"repo":{"fullName":"owner/old"},"auth":{"source":"gh","reference":"gh"},"runner":{"name":"runnerkit-owner-old-local","labels":["self-hosted"],"mode":"persistent","os":"linux","arch":"x64"},"machine":{"kind":"byo-ssh"},"provider":{"kind":"byo"},"cleanup":{"managed_paths":[],"provider_resource_ids":[]},"safety":{"code":"ok","allowed":true},"runnerkit_version":"test","created_at":"2026-04-29T00:00:00Z","updated_at":"2026-04-29T00:00:00Z"}]}`
	store2 := NewStore(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(store2.Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store2.Path(), []byte(json), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store2.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Repositories) != 1 || loaded.Repositories[0].Safety.SafetyProfile != "" {
		t.Fatalf("backwards-compatible load: %#v", loaded.Repositories)
	}
}

func TestEphemeralMetadataPersistsAndIsBackwardsCompatible(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 5, 2, 18, 30, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)
	state := NewState()
	state.Repositories = []RepositoryState{{
		Repo:    gh.Repo{Host: "github.com", Owner: "owner", Name: "name", FullName: "owner/name", Private: false},
		Auth:    AuthReference{Source: "gh", Reference: "gh"},
		Runner:  RunnerIdentity{Name: "runnerkit-owner-name-ephemeral-abc123", Labels: []string{"self-hosted", "ephemeral"}, Mode: "ephemeral", OS: "linux", Arch: "x64"},
		Machine: MachineRef{Kind: "byo-ssh", ServiceName: "runnerkit-ephemeral.runnerkit-owner-name-ephemeral-abc123.service"},
		Provider: ProviderRef{Kind: "byo"},
		Cleanup: CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety:  SafetyMetadata{Code: "ok", Allowed: true, SafetyProfile: "ephemeral-byo"},
		Ephemeral: EphemeralMetadata{
			Enabled:         true,
			TTL:             "24h",
			ExpiresAt:       &expires,
			LogArchivePath:  "/var/lib/runnerkit/ephemeral/runnerkit-owner-name-ephemeral-abc123/logs",
			FinalizerStatus: "pending",
			CleanupCommand:  "runnerkit down --repo owner/name",
		},
		RunnerKitVersion: "test-version",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	for _, want := range []string{
		`"ephemeral"`,
		`"enabled": true`,
		`"ttl": "24h"`,
		`"expires_at"`,
		`"log_archive_path": "/var/lib/runnerkit/ephemeral/runnerkit-owner-name-ephemeral-abc123/logs"`,
		`"finalizer_status": "pending"`,
		`"cleanup_command": "runnerkit down --repo owner/name"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("ephemeral metadata missing %q:\n%s", want, string(data))
		}
	}

	// Backwards compatible: state without the ephemeral key still loads.
	json := `{"schema_version":"1","repositories":[{"repo":{"fullName":"owner/legacy"},"auth":{"source":"gh","reference":"gh"},"runner":{"name":"runnerkit-owner-legacy-local","labels":["self-hosted"],"mode":"persistent","os":"linux","arch":"x64"},"machine":{"kind":"byo-ssh"},"provider":{"kind":"byo"},"cleanup":{"managed_paths":[],"provider_resource_ids":[]},"safety":{"code":"ok","allowed":true},"runnerkit_version":"test","created_at":"2026-04-29T00:00:00Z","updated_at":"2026-04-29T00:00:00Z"}]}`
	store2 := NewStore(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(store2.Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store2.Path(), []byte(json), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store2.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Repositories) != 1 || loaded.Repositories[0].Ephemeral.Enabled {
		t.Fatalf("backwards-compat load: %#v", loaded.Repositories)
	}
}

func TestProjectConfigPathUsesRunnerKitConfigYAML(t *testing.T) {
	path := ProjectConfigPath("/work/project")
	if !strings.HasSuffix(path, filepath.Join(".runnerkit", "config.yaml")) {
		t.Fatalf("ProjectConfigPath() = %q, want .runnerkit/config.yaml", path)
	}
}

func TestRepositoryStateListUpdateRemoveAndOperationCheckpointPersistence(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 4, 29, 4, 0, 0, 0, time.UTC)
	repo := RepositoryState{
		Repo:       gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth:       AuthReference{Source: "gh", Reference: "gh"},
		Runner:     RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted"}},
		Machine:    MachineRef{Kind: "byo-ssh"},
		Provider:   ProviderRef{Kind: "byo"},
		Cleanup:    CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{}},
		Safety:     SafetyMetadata{Code: "ok", Allowed: true},
		CreatedAt:  now,
		UpdatedAt:  now,
		Operations: []OperationCheckpoint{{Command: "down", Artifact: "github_runner", Status: "pending", Message: "github_cleanup_pending", UpdatedAt: now}},
	}
	if err := store.Save(State{Repositories: []RepositoryState{repo}}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	repos, err := store.ListRepositories()
	if err != nil || len(repos) != 1 {
		t.Fatalf("ListRepositories() = %d, %v", len(repos), err)
	}
	repo.Cleanup.Notes = []string{"remote_cleanup_pending"}
	repo.UpdatedAt = now.Add(time.Minute)
	if err := store.UpdateRepository(repo); err != nil {
		t.Fatalf("UpdateRepository returned error: %v", err)
	}
	loaded, found, err := store.GetRepository("owner/repo")
	if err != nil || !found {
		t.Fatalf("GetRepository after update found=%v err=%v", found, err)
	}
	if len(loaded.Operations) != 1 || loaded.Operations[0].Message != "github_cleanup_pending" || loaded.Cleanup.Notes[0] != "remote_cleanup_pending" {
		t.Fatalf("operation checkpoint or cleanup notes did not persist: %#v", loaded)
	}
	removed, err := store.RemoveRepository("owner/repo")
	if err != nil || !removed {
		t.Fatalf("RemoveRepository removed=%v err=%v", removed, err)
	}
	repos, err = store.ListRepositories()
	if err != nil || len(repos) != 0 {
		t.Fatalf("ListRepositories after remove = %d, %v", len(repos), err)
	}
}

func TestCloudInventorySerializesProviderIdentityAndNoSecrets(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 5, 1, 0, 30, 0, 0, time.UTC)
	cloud := CloudInventory{
		Provider:          "hetzner",
		ServerID:          "srv-123",
		ServerName:        "runnerkit-owner-repo-local",
		ServerStatus:      "provisioning",
		Region:            "fsn1",
		ServerType:        "cpx22",
		Image:             "ubuntu-24.04",
		PublicIPv4:        "203.0.113.10",
		PublicIPv6:        "2001:db8::10",
		PrimaryIPv4ID:     "ipv4-123",
		PrimaryIPv6ID:     "ipv6-123",
		SSHKeyID:          "key-123",
		SSHKeyName:        "runnerkit-owner-repo-local-ssh-key",
		SSHKeyFingerprint: "SHA256:sshkeyfingerprint",
		FirewallID:        "fw-123",
		FirewallName:      "runnerkit-owner-repo-local-firewall",
		Tags:              map[string]string{"runnerkit": "true", "managed": "true", "mode": "persistent"},
		CostProfile: CostProfileRef{
			Provider:             "hetzner",
			Region:               "fsn1",
			ServerType:           "cpx22",
			Image:                "ubuntu-24.04",
			EstimatedHourlyCost:  "approx €0.0081/hour",
			EstimatedMonthlyCost: "approx €4.90/month",
			Caveat:               "Estimated cost is approximate.",
		},
		CloudInitVersion: "runnerkit-cloud-init-v2",
	}
	state := State{Repositories: []RepositoryState{{
		Repo:       gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: "owner/repo", Private: true},
		Auth:       AuthReference{Source: "gh", Reference: "gh"},
		Runner:     RunnerIdentity{Name: "runnerkit-owner-repo-local", Labels: []string{"self-hosted", "runnerkit"}, Mode: "persistent", OS: "linux", Arch: "x64"},
		Machine:    MachineRef{Kind: "cloud-ssh", HostRef: "runnerkit-admin@203.0.113.10:22", User: "runnerkit-admin", Port: 22},
		Provider:   ProviderRef{Kind: "hetzner", Name: "hetzner", Region: "fsn1", Profile: "cpx22", IDs: map[string]string{"server": "srv-123"}, ResourceIDs: map[string]string{"server": "srv-123", "ssh_key": "key-123", "firewall": "fw-123", "primary_ipv4": "ipv4-123"}, Tags: cloud.Tags, Cloud: cloud},
		Cleanup:    CleanupMetadata{ManagedPaths: []string{}, ProviderResourceIDs: []string{"server:srv-123", "ssh_key:key-123", "firewall:fw-123", "primary_ipv4:ipv4-123"}, Notes: []string{"cloud_provision_pending"}},
		Safety:     SafetyMetadata{Code: "ok", Allowed: true},
		Operations: []OperationCheckpoint{{Command: "up", Artifact: "provider", Status: "pending", Message: "cloud_provision_pending", UpdatedAt: now}},
		CreatedAt:  now,
		UpdatedAt:  now,
	}}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	text := string(data)
	for _, want := range []string{`"provider": "hetzner"`, `"server_id": "srv-123"`, `"ssh_key_id": "key-123"`, `"firewall_id": "fw-123"`, `"primary_ipv4_id": "ipv4-123"`, `"cost_profile"`, `"provider_resource_ids"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("serialized cloud state missing %s:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"HCLOUD_TOKEN", "HETZNER_CLOUD_TOKEN", "fake-provider-token", "BEGIN OPENSSH PRIVATE KEY"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("serialized cloud state leaked %q:\n%s", forbidden, text)
		}
	}
}
