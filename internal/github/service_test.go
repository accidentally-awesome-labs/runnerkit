package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/redact"
)

type missingServiceCommandRunner struct{}

func (missingServiceCommandRunner) LookPath(string) (string, error) {
	return "", errors.New("not found")
}
func (missingServiceCommandRunner) Run(context.Context, string, ...string) (string, error) {
	return "", errors.New("not found")
}

func TestServiceVerifyAuthRepositoryAndRedactsTokens(t *testing.T) {
	redactor := redact.New()
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ghp_fixturetoken12345" {
			t.Fatalf("Authorization header = %q, want bearer fixture token", got)
		}
		switch r.URL.Path {
		case "/repos/owner/name":
			seen["repo"] = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"full_name":"owner/name","private":false,"fork":false,"owner":{"login":"owner"},"name":"name"}`))
		case "/repos/owner/name/actions/runners/registration-token":
			seen["registration"] = true
			if r.Method != http.MethodPost {
				t.Fatalf("registration token method = %s, want POST", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "registration-token-fixture-12345", "expires_at": "2026-04-29T03:00:00Z"})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := NewService(ServiceOptions{
		Env:        map[string]string{"RUNNERKIT_GITHUB_TOKEN": "ghp_fixturetoken12345"},
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Redactor:   redactor,
	})
	repo := Repo{Owner: "owner", Name: "name", FullName: "owner/name"}

	status, err := service.VerifyAuth(context.Background(), repo)
	if err != nil {
		t.Fatalf("VerifyAuth returned error: %v", err)
	}
	if !status.OK || status.Source.Kind != "fine-grained-token" {
		t.Fatalf("unexpected status: %#v", status)
	}
	metadata, err := service.Repository(context.Background(), repo)
	if err != nil {
		t.Fatalf("Repository returned error: %v", err)
	}
	if metadata.Private {
		t.Fatalf("Repository returned Private=true, want false from fixture")
	}
	if !seen["repo"] || !seen["registration"] {
		t.Fatalf("expected repo and registration endpoints to be called, seen=%#v", seen)
	}
	redacted := redactor.String("ghp_fixturetoken12345 registration-token-fixture-12345")
	for _, raw := range []string{"ghp_fixturetoken12345", "registration-token-fixture-12345"} {
		if strings.Contains(redacted, raw) {
			t.Fatalf("redactor did not mask %q in %q", raw, redacted)
		}
	}
}

func TestServiceMissingAuthReturnsFineGrainedRemediation(t *testing.T) {
	service := NewService(ServiceOptions{CommandRunner: missingServiceCommandRunner{}, Env: map[string]string{}})
	status, err := service.VerifyAuth(context.Background(), Repo{Owner: "owner", Name: "name", FullName: "owner/name"})
	if err == nil {
		t.Fatal("expected missing auth error")
	}
	if status.OK {
		t.Fatalf("VerifyAuth status OK = true, want false")
	}
	if len(status.Remediation) == 0 || !strings.Contains(status.Remediation[0], "fine-grained token scoped only to owner/name") {
		t.Fatalf("missing fine-grained token remediation: %#v", status.Remediation)
	}
}
