package hetzner

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestResolveTokenPrefersHCLOUDToken(t *testing.T) {
	source, err := ResolveToken(map[string]string{EnvHCLOUDToken: "hcloud-secret", EnvHetznerCloudToken: "alias-secret"})
	if err != nil {
		t.Fatalf("ResolveToken returned error: %v", err)
	}
	if source.Source != EnvHCLOUDToken || source.Token != "hcloud-secret" {
		t.Fatalf("unexpected source: %#v", source)
	}
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal token source: %v", err)
	}
	if strings.Contains(string(encoded), "hcloud-secret") || !strings.Contains(string(encoded), `"source":"HCLOUD_TOKEN"`) {
		t.Fatalf("token value persisted in JSON: %s", encoded)
	}
}

func TestResolveTokenUsesAliasWhenPrimaryMissing(t *testing.T) {
	source, err := ResolveToken(map[string]string{EnvHetznerCloudToken: "alias-secret"})
	if err != nil {
		t.Fatalf("ResolveToken returned error: %v", err)
	}
	if source.Source != EnvHetznerCloudToken || source.Token != "alias-secret" {
		t.Fatalf("unexpected source: %#v", source)
	}
}

func TestResolveTokenMissingRemediation(t *testing.T) {
	source, err := ResolveToken(map[string]string{})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if source != (TokenSource{}) {
		t.Fatalf("expected no token source when missing, got %#v", source)
	}
	var missing *MissingTokenError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingTokenError, got %T", err)
	}
	joined := strings.Join(missing.Remediation, "\n")
	for _, want := range []string{"Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>", "Re-run runnerkit up --repo owner/name --cloud hetzner"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("remediation missing %q: %#v", want, missing.Remediation)
		}
	}
}
