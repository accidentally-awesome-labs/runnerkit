package checkliststore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fileSchemaVersion = 1

// Doc is a persisted checklist session.
type Doc struct {
	SchemaVersion int       `json:"schema_version"`
	SessionID     string    `json:"session_id"`
	UpdatedAt     time.Time `json:"updated_at"`
	Steps         []Step    `json:"steps"`
}

// Step mirrors ui.ChecklistStep for JSON persistence.
type Step struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"` // done | active | pending
	Duration string `json:"duration,omitempty"`
}

func sessionsDir(stateBase string) string {
	return filepath.Join(stateBase, "sessions")
}

// SessionPath returns the filesystem path for a session id.
func SessionPath(stateBase, sessionID string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, sessionID)
	return filepath.Join(sessionsDir(stateBase), safe+".json")
}

// Load reads a session file; returns nil doc if not found.
func Load(stateBase, sessionID string) (*Doc, error) {
	p := SessionPath(stateBase, sessionID)
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var d Doc
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Save writes a session file atomically.
func Save(stateBase string, d *Doc) error {
	if d == nil {
		return errors.New("nil doc")
	}
	d.SchemaVersion = fileSchemaVersion
	d.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(sessionsDir(stateBase), 0o700); err != nil {
		return err
	}
	p := SessionPath(stateBase, d.SessionID)
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// BYORegisterSessionID builds a stable id for BYO register/up flows.
func BYORegisterSessionID(repoFullName, hostRef string) string {
	return fmt.Sprintf("byo-%s__%s", repoFullName, hostRef)
}
