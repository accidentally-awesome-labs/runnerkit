package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// UserConfig is persisted next to state.json (same base directory).
type UserConfig struct {
	DoctorIgnoreFindingIDs []string `json:"doctor_ignore_finding_ids,omitempty"`
}

func userConfigPath(stateBaseDir string) string {
	return filepath.Join(stateBaseDir, "config.json")
}

// LoadUserConfig reads config.json; missing file yields empty config.
func LoadUserConfig(stateBaseDir string) (UserConfig, error) {
	p := userConfigPath(stateBaseDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UserConfig{}, nil
		}
		return UserConfig{}, err
	}
	var c UserConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return UserConfig{}, err
	}
	return c, nil
}

// SaveUserConfig writes config.json atomically.
func SaveUserConfig(stateBaseDir string, c UserConfig) error {
	if err := os.MkdirAll(stateBaseDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	p := userConfigPath(stateBaseDir)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func mergeUniqueStrings(base []string, add []string) []string {
	seen := map[string]bool{}
	for _, s := range base {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		seen[s] = true
	}
	out := append([]string(nil), base...)
	for _, s := range add {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
