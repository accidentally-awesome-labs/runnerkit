package github

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/salar/runnerkit/internal/redact"
	"github.com/salar/runnerkit/internal/ui"
)

func TestParseRepoExplicitOwnerName(t *testing.T) {
	repo, err := ParseRepo("owner/name")
	if err != nil {
		t.Fatalf("ParseRepo returned error: %v", err)
	}
	if repo.Host != "github.com" || repo.Owner != "owner" || repo.Name != "name" || repo.FullName != "owner/name" {
		t.Fatalf("unexpected repo: %#v", repo)
	}
}

func TestParseRemoteGitHubURLs(t *testing.T) {
	tests := map[string]string{
		"https://github.com/owner/name.git":     "owner/name",
		"git@github.com:owner/name.git":         "owner/name",
		"ssh://git@github.com/owner/name.git":   "owner/name",
		"https://github.com/owner/name":         "owner/name",
		"git@github.com:owner.with-dots/name_1": "owner.with-dots/name_1",
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			repo, err := ParseRemote(raw)
			if err != nil {
				t.Fatalf("ParseRemote returned error: %v", err)
			}
			if repo.FullName != want || repo.Host != "github.com" {
				t.Fatalf("ParseRemote(%q) = %#v, want %s", raw, repo, want)
			}
		})
	}
}

func TestParseRemoteRejectsInvalidHost(t *testing.T) {
	if _, err := ParseRemote("https://example.com/owner/name.git"); err == nil {
		t.Fatal("expected invalid host error")
	}
}

func TestResolveTargetMissingRepo(t *testing.T) {
	if _, err := ResolveTarget(context.Background(), ResolveOptions{}); err == nil {
		t.Fatal("expected missing repository error")
	}
}

const fixtureRegistrationToken = "registration-token-fixture-12345"
const fixtureRemovalToken = "remove-token-fixture-12345"
const fixtureGitHubToken = "ghp_fixturetoken12345"

type fakeCommandRunnerForAuth struct {
	lookPathErr error
	output      string
	commands    []string
}

func (f *fakeCommandRunnerForAuth) LookPath(name string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	return "/usr/bin/" + name, nil
}

func (f *fakeCommandRunnerForAuth) Run(_ context.Context, name string, args ...string) (string, error) {
	f.commands = append(f.commands, name+" "+strings.Join(args, " "))
	return f.output, nil
}

func TestDiscoverAuthUsesGhFirstAndRegistersToken(t *testing.T) {
	redactor := redact.New()
	runner := &fakeCommandRunnerForAuth{output: fixtureGitHubToken + "\n"}
	credential, err := DiscoverAuth(context.Background(), AuthOptions{CommandRunner: runner, Redactor: redactor})
	if err != nil {
		t.Fatalf("DiscoverAuth returned error: %v", err)
	}
	if credential.Source.Kind != "gh" || credential.Source.Reference != "gh" {
		t.Fatalf("unexpected source: %#v", credential.Source)
	}
	if len(runner.commands) != 1 || runner.commands[0] != "gh auth token" {
		t.Fatalf("DiscoverAuth did not use gh auth token first: %#v", runner.commands)
	}
	if strings.Contains(redactor.String("token="+fixtureGitHubToken), fixtureGitHubToken) {
		t.Fatalf("github token was not registered for redaction")
	}
}

func TestFineGrainedTokenRemediationCopy(t *testing.T) {
	repo := Repo{FullName: "owner/name"}
	want := "Create a fine-grained token scoped only to owner/name with repository Administration read/write and Metadata read, then pass it with RUNNERKIT_GITHUB_TOKEN for this command."
	if got := FineGrainedTokenRemediation(repo); got != want {
		t.Fatalf("remediation = %q, want %q", got, want)
	}
}

func TestClientCreatesRegistrationTokenWithRESTHeadersAndRedaction(t *testing.T) {
	redactor := redact.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/name/actions/runners/registration-token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Fatalf("missing Accept header: %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Fatalf("missing API version header: %q", r.Header.Get("X-GitHub-Api-Version"))
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"token": fixtureRegistrationToken, "expires_at": "2026-04-29T03:00:00Z"})
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL, Token: fixtureGitHubToken, HTTPClient: server.Client(), Redactor: redactor})
	token, err := client.CreateRegistrationToken(context.Background(), Repo{Owner: "owner", Name: "name", FullName: "owner/name"})
	if err != nil {
		t.Fatalf("CreateRegistrationToken returned error: %v", err)
	}
	if token.Token != fixtureRegistrationToken || token.Kind != redact.RunnerRegistrationToken {
		t.Fatalf("unexpected token: %#v", token)
	}
	if strings.Contains(redactor.String("runner token "+fixtureRegistrationToken), fixtureRegistrationToken) {
		t.Fatalf("registration token was not registered for redaction")
	}
}

func TestClientPermissionDeniedOnRegistrationToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Resource not accessible by integration"})
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL, Token: fixtureGitHubToken, HTTPClient: server.Client(), Redactor: redact.New()})
	_, err := client.CreateRegistrationToken(context.Background(), Repo{Owner: "owner", Name: "name", FullName: "owner/name"})
	if !IsPermissionDenied(err) {
		t.Fatalf("expected permission denied error, got %v", err)
	}
}

func TestClientCreatesRemovalTokenAndRedactsFixture(t *testing.T) {
	redactor := redact.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/name/actions/runners/remove-token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"token": fixtureRemovalToken, "expires_at": "2026-04-29T03:00:00Z"})
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL, Token: fixtureGitHubToken, HTTPClient: server.Client(), Redactor: redactor})
	if _, err := client.CreateRemovalToken(context.Background(), Repo{Owner: "owner", Name: "name", FullName: "owner/name"}); err != nil {
		t.Fatalf("CreateRemovalToken returned error: %v", err)
	}
	if strings.Contains(redactor.String("runner token "+fixtureRemovalToken), fixtureRemovalToken) {
		t.Fatalf("removal token was not registered for redaction")
	}
}

func TestRunnerTokenFixtureTextNeverAppearsInRenderedOutput(t *testing.T) {
	redactor := redact.New()
	redactor.Register(redact.RunnerRegistrationToken, fixtureRegistrationToken)
	var out bytes.Buffer
	renderer := ui.NewRenderer(&out, &out, ui.FormatJSON, ui.TerminalCapabilities{Width: 80}, redactor)
	if err := renderer.JSON(map[string]any{"ok": true, "runner_registration_token": fixtureRegistrationToken}); err != nil {
		t.Fatalf("render json: %v", err)
	}
	if strings.Contains(out.String(), fixtureRegistrationToken) {
		t.Fatalf("fixture token appeared in rendered output: %s", out.String())
	}
}
