package redact

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type Kind string

const (
	GitHubToken             Kind = "github-token"
	RunnerRegistrationToken Kind = "runner-registration-token"
	RunnerRemovalToken      Kind = "runner-removal-token"
	SSHPrivateKey           Kind = "ssh-private-key"
	ProviderCredential      Kind = "provider-credential"
	MachineRef              Kind = "machine-ref"
)

type registeredValue struct {
	kind  Kind
	value string
}

type patternFilter struct {
	kind Kind
	re   *regexp.Regexp
}

// Redactor masks sensitive values and known token-like patterns before output.
type Redactor struct {
	mu       sync.RWMutex
	values   []registeredValue
	patterns []patternFilter
}

func New() *Redactor {
	return &Redactor{
		patterns: []patternFilter{
			{kind: GitHubToken, re: regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+)\b`)},
			{kind: RunnerRegistrationToken, re: regexp.MustCompile(`\bregistration-token-[A-Za-z0-9._-]+\b`)},
			{kind: RunnerRemovalToken, re: regexp.MustCompile(`\b(?:remove|removal)-token-[A-Za-z0-9._-]+\b`)},
			{kind: SSHPrivateKey, re: regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)},
			{kind: ProviderCredential, re: regexp.MustCompile(`\bHCLOUD_TOKEN=[^\s]+`)},
		},
	}
}

func (r *Redactor) Register(kind Kind, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values = append(r.values, registeredValue{kind: kind, value: value})
	sort.SliceStable(r.values, func(i, j int) bool {
		return len(r.values[i].value) > len(r.values[j].value)
	})
}

func (r *Redactor) String(input string) string {
	if r == nil {
		return input
	}
	r.mu.RLock()
	values := append([]registeredValue(nil), r.values...)
	patterns := append([]patternFilter(nil), r.patterns...)
	r.mu.RUnlock()

	output := input
	for _, item := range values {
		output = strings.ReplaceAll(output, item.value, replacement(item.kind))
	}
	for _, item := range patterns {
		output = item.re.ReplaceAllString(output, replacement(item.kind))
	}
	return output
}

func (r *Redactor) JSONBytes(input []byte) []byte {
	if r == nil {
		return input
	}
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return []byte(r.String(string(input)))
	}
	value = r.sanitizeJSON(value, "")
	encoded, err := marshalNoEscape(value)
	if err != nil {
		return []byte(r.String(string(input)))
	}
	return []byte(r.String(string(encoded)))
}

func (r *Redactor) sanitizeJSON(value any, key string) any {
	if kind, ok := sensitiveKindForKey(key); ok {
		return replacement(kind)
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = r.sanitizeJSON(v, k)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, v := range typed {
			out[i] = r.sanitizeJSON(v, key)
		}
		return out
	case string:
		return r.String(typed)
	default:
		return value
	}
}

func sensitiveKindForKey(key string) (Kind, bool) {
	lower := strings.ToLower(key)
	lower = strings.ReplaceAll(lower, "-", "_")
	if lower == "" || lower == "redactions_applied" {
		return "", false
	}
	if strings.Contains(lower, "registration") && strings.Contains(lower, "token") {
		return RunnerRegistrationToken, true
	}
	if (strings.Contains(lower, "removal") || strings.Contains(lower, "remove")) && strings.Contains(lower, "token") {
		return RunnerRemovalToken, true
	}
	if strings.Contains(lower, "private_key") || (strings.Contains(lower, "ssh") && strings.Contains(lower, "key")) {
		return SSHPrivateKey, true
	}
	if strings.Contains(lower, "github") && strings.Contains(lower, "token") {
		return GitHubToken, true
	}
	if strings.Contains(lower, "hcloud") || strings.Contains(lower, "provider") || strings.Contains(lower, "credential") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "api_token") || strings.Contains(lower, "api_key") {
		return ProviderCredential, true
	}
	if lower == "token" || strings.HasSuffix(lower, "_token") {
		return GitHubToken, true
	}
	if strings.Contains(lower, "machine_ref") {
		return MachineRef, true
	}
	return "", false
}

func replacement(kind Kind) string {
	switch kind {
	case GitHubToken:
		return "<redacted:github-token>"
	case RunnerRegistrationToken:
		return "<redacted:runner-registration-token>"
	case RunnerRemovalToken:
		return "<redacted:runner-removal-token>"
	case SSHPrivateKey:
		return "<redacted:ssh-private-key>"
	case ProviderCredential:
		return "<redacted:provider-credential>"
	case MachineRef:
		return "<redacted:machine-ref>"
	default:
		return "<redacted>"
	}
}

func marshalNoEscape(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
