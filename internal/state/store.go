package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Store struct {
	path string
}

func NewStore(baseDir string) Store {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = DefaultBaseDir()
	}
	return Store{path: filepath.Join(baseDir, "state.json")}
}

func NewStoreAtPath(path string) Store {
	return Store{path: path}
}

func (s Store) Path() string { return s.path }

func DefaultBaseDir() string {
	if dir := strings.TrimSpace(os.Getenv("RUNNERKIT_STATE_DIR")); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); dir != "" {
		return filepath.Join(dir, "runnerkit")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".local", "state", "runnerkit")
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "runnerkit", "state")
	}
	return filepath.Join(os.TempDir(), "runnerkit")
}

func (s Store) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

func (s Store) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return NewState(), nil
	}
	if err != nil {
		return State{}, err
	}
	if err := ValidatePersistedJSON(data); err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	// Refuse-to-mutate path: if on-disk schema is newer than this CLI knows,
	// surface ErrSchemaTooNew immediately and write no backup. The state
	// file on disk is left untouched.
	if state.SchemaVersion != "" && cmpVersion(state.SchemaVersion, SchemaVersion) > 0 {
		return State{}, ErrSchemaTooNew
	}
	// Forward migration: when the on-disk schema is older than what the CLI
	// knows, write a side-by-side backup of the ORIGINAL raw bytes BEFORE
	// running Migrate (D-09). The backup MUST persist even if subsequent
	// steps fail.
	migrated := state
	migrating := state.SchemaVersion != "" && cmpVersion(state.SchemaVersion, SchemaVersion) < 0
	if migrating {
		backupPath := fmt.Sprintf("%s.backup-v%s-%s", s.path, state.SchemaVersion, time.Now().UTC().Format("20060102T150405Z"))
		if err := writeBackup(backupPath, data); err != nil {
			return State{}, fmt.Errorf("write state backup: %w", err)
		}
		mig, mErr := Migrate(state)
		if mErr != nil {
			return State{}, mErr
		}
		migrated = mig
		// Persist the migrated state via the existing atomic-write helper so
		// future loads see schema_version >= SchemaVersion. If this save
		// fails, the original file is left in place (atomic rename) and the
		// backup file persists for recovery.
		if err := s.Save(migrated); err != nil {
			return State{}, fmt.Errorf("write migrated state: %w", err)
		}
		return migrated, nil
	}
	return Migrate(state)
}

// writeBackup writes the raw bytes to a side-by-side backup path with mode
// 0600. The backup must exist before any mutation of the original file. We
// intentionally do NOT chmod the parent directory here — the directory was
// already created with 0700 by the Store, and silently relaxing perms
// during a load would surprise users who tightened them externally.
func writeBackup(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (s Store) Save(state State) error {
	if state.SchemaVersion == "" {
		state.SchemaVersion = SchemaVersion
	}
	migrated, err := Migrate(state)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(migrated, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := ValidatePersistedJSON(data); err != nil {
		return err
	}
	return s.writeAtomic(data)
}

func (s Store) SaveRepository(repo RepositoryState, replace bool) error {
	state, err := s.Load()
	if err != nil {
		return err
	}
	if err := state.UpsertRepository(repo, replace); err != nil {
		return err
	}
	return s.Save(state)
}

func (s Store) ListRepositories() ([]RepositoryState, error) {
	state, err := s.Load()
	if err != nil {
		return nil, err
	}
	return state.ListRepositories(), nil
}

func (s Store) UpdateRepository(repo RepositoryState) error {
	state, err := s.Load()
	if err != nil {
		return err
	}
	state.UpdateRepository(repo)
	return s.Save(state)
}

func (s Store) RemoveRepository(fullName string) (bool, error) {
	state, err := s.Load()
	if err != nil {
		return false, err
	}
	removed := state.RemoveRepository(fullName)
	if !removed {
		return false, nil
	}
	return true, s.Save(state)
}

func (s Store) GetRepository(fullName string) (RepositoryState, bool, error) {
	state, err := s.Load()
	if err != nil {
		return RepositoryState{}, false, err
	}
	repo, ok := state.FindRepository(fullName)
	return repo, ok, nil
}

func (s Store) writeAtomic(data []byte) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dir, 0700)
	}
	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d-%d", filepath.Base(s.path), os.Getpid(), time.Now().UnixNano()))
	tmp, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(s.path, 0600)
	}
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}
	return nil
}

func ValidatePersistedJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	return validateNoRawSecretKeys(value, "")
}

func validateNoRawSecretKeys(value any, path string) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := normalizeKey(key)
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if isDeniedSecretKey(normalized) && !isRedactedDisplayString(child) {
				return fmt.Errorf("state contains raw secret field %q", childPath)
			}
			if err := validateNoRawSecretKeys(child, childPath); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range typed {
			if err := validateNoRawSecretKeys(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "-", "_")
	return key
}

func isDeniedSecretKey(key string) bool {
	switch key {
	case "token", "registration_token", "runner_registration_token", "remove_token", "removal_token", "runner_removal_token", "private_key", "ssh_private_key", "provider_credential", "provider_credentials":
		return true
	default:
		return false
	}
}

func isRedactedDisplayString(value any) bool {
	text, ok := value.(string)
	return ok && strings.HasPrefix(text, "<redacted:") && strings.HasSuffix(text, ">")
}
