package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListRunnersParsesRepositoryRunnerInventory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/name/actions/runners" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"total_count":1,"runners":[{"id":123,"name":"runnerkit-owner-repo-local","os":"linux","status":"online","busy":false,"labels":[{"name":"self-hosted"},{"name":"runnerkit"}]}]}`))
	}))
	defer server.Close()
	client := NewClient(ClientOptions{BaseURL: server.URL})
	runners, err := client.ListRunners(context.Background(), Repo{Owner: "owner", Name: "name", FullName: "owner/name"})
	if err != nil {
		t.Fatalf("ListRunners returned error: %v", err)
	}
	if len(runners) != 1 || runners[0].ID != 123 || runners[0].Name != "runnerkit-owner-repo-local" || runners[0].Status != "online" || len(runners[0].Labels) != 2 {
		t.Fatalf("unexpected runners: %#v", runners)
	}
	if runner, ok := FindRunnerByName(runners, "runnerkit-owner-repo-local"); !ok || runner.ID != 123 {
		t.Fatalf("FindRunnerByName failed: %#v %t", runner, ok)
	}
}

func TestClientDeleteRunnerUsesRepositoryRunnerEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/repos/owner/name/actions/runners/123" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := NewClient(ClientOptions{BaseURL: server.URL})
	if err := client.DeleteRunner(context.Background(), Repo{Owner: "owner", Name: "name", FullName: "owner/name"}, 123); err != nil {
		t.Fatalf("DeleteRunner returned error: %v", err)
	}
}
