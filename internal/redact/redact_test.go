package redact

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStringRedactsRegisteredValuesAndKnownPatterns(t *testing.T) {
	r := New()
	r.Register(GitHubToken, "registered-gh-token")
	r.Register(RunnerRegistrationToken, "registered-registration-token")
	r.Register(RunnerRemovalToken, "registered-remove-token")
	r.Register(ProviderCredential, "registered-provider-secret")
	r.Register(MachineRef, "machine-123.internal")
	r.Register(MachineRef, "alice@example.com:22")

	input := strings.Join([]string{
		"registered registered-gh-token",
		"classic ghp_example token",
		"fine grained github_pat_example token",
		"runner registration-token-secret value",
		"runner registration-token-secret-logs value",
		"runner remove-token-secret value",
		"runner remove-token-secret-logs value",
		"runner removal-token-secret-logs value",
		"-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----",
		"HCLOUD_TOKEN=provider-secret",
		"HCLOUD_TOKEN=supersecret",
		"registered-provider-secret",
		"Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>",
		"machine-123.internal",
		"alice@example.com:22",
	}, "\n")

	got := r.String(input)
	if !strings.Contains(got, "Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>") {
		t.Fatalf("placeholder credential guidance should not be redacted: %s", got)
	}
	for _, raw := range []string{
		"registered-gh-token",
		"ghp_example",
		"github_pat_example",
		"registration-token-secret",
		"registration-token-secret-logs",
		"remove-token-secret",
		"remove-token-secret-logs",
		"removal-token-secret-logs",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"provider-secret",
		"HCLOUD_TOKEN=supersecret",
		"registered-provider-secret",
		"machine-123.internal",
		"alice@example.com:22",
	} {
		if strings.Contains(got, raw) {
			t.Fatalf("redacted output still contains %q: %s", raw, got)
		}
	}
	for _, replacement := range []string{
		"<redacted:github-token>",
		"<redacted:runner-registration-token>",
		"<redacted:runner-removal-token>",
		"<redacted:ssh-private-key>",
		"<redacted:provider-credential>",
		"<redacted:machine-ref>",
	} {
		if !strings.Contains(got, replacement) {
			t.Fatalf("redacted output missing %q: %s", replacement, got)
		}
	}
}

func TestJSONBytesRedactsTokenLikeFields(t *testing.T) {
	r := New()
	input := []byte(`{"github_token":"plain-token","runner_registration_token":"reg-token","runner_removal_token":"remove-token","private_key":"-----BEGIN OPENSSH PRIVATE KEY-----secret","HCLOUD_TOKEN":"hcloud-secret","nested":{"url":"https://ghp_example@github.com/owner/repo"}}`)
	got := r.JSONBytes(input)
	if !json.Valid(got) {
		t.Fatalf("redacted bytes are not json: %s", got)
	}
	for _, raw := range []string{"plain-token", "reg-token", "remove-token", "OPENSSH", "hcloud-secret", "ghp_example"} {
		if strings.Contains(string(got), raw) {
			t.Fatalf("redacted json still contains %q: %s", raw, got)
		}
	}
}

func TestJSONBytesKeepsSafeProviderStateFields(t *testing.T) {
	r := New()
	input := []byte(`{"provider":{"kind":"none","ids":{}},"cleanup":{"provider_resource_ids":[]},"provider_credential":"raw-secret"}`)
	got := string(r.JSONBytes(input))
	if !strings.Contains(got, `"provider":{"ids":{},"kind":"none"}`) {
		t.Fatalf("safe provider state was redacted unexpectedly: %s", got)
	}
	if !strings.Contains(got, `"provider_resource_ids":[]`) {
		t.Fatalf("safe provider resource IDs field was redacted unexpectedly: %s", got)
	}
	if strings.Contains(got, "raw-secret") || !strings.Contains(got, "<redacted:provider-credential>") {
		t.Fatalf("provider credential was not redacted: %s", got)
	}
}

// TestRedact_SudoPasswordRegistration asserts the new SudoPassword Kind
// (added by Plan 06-06 for Path B interactive sudo password fallback)
// (a) registers a literal value via Register so subsequent String calls
// strip it from output, (b) emits the canonical
// `<redacted:sudo-password>` replacement marker, and (c) auto-redacts
// JSON fields whose key is `sudo_password` regardless of registration.
func TestRedact_SudoPasswordRegistration(t *testing.T) {
	r := New()
	r.Register(SudoPassword, "p@ssw0rd!")

	got := r.String("the password is p@ssw0rd! today")
	if !strings.Contains(got, "<redacted:sudo-password>") {
		t.Fatalf("String() missing sudo-password sentinel: %s", got)
	}
	if strings.Contains(got, "p@ssw0rd!") {
		t.Fatalf("String() leaked raw password: %s", got)
	}

	// JSON-key based redaction: even without prior registration, fields
	// named sudo_password / sudo-password should redact the value.
	r2 := New()
	jsonIn := []byte(`{"sudo_password":"p@ssw0rd!","note":"plain"}`)
	jsonOut := string(r2.JSONBytes(jsonIn))
	if strings.Contains(jsonOut, "p@ssw0rd!") {
		t.Fatalf("JSONBytes leaked raw password (key-based): %s", jsonOut)
	}
	if !strings.Contains(jsonOut, "<redacted:sudo-password>") {
		t.Fatalf("JSONBytes missing sudo-password sentinel (key-based): %s", jsonOut)
	}
	if !strings.Contains(jsonOut, `"note":"plain"`) {
		t.Fatalf("JSONBytes redacted unrelated note field: %s", jsonOut)
	}
}
