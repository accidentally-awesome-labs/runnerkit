package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	hcloud "github.com/hetznercloud/hcloud-go/hcloud"
)

// fakeVerifier implements verifierClient for unit tests. It returns the
// resource for the first `return404After` lookups, then a NotFound API
// error so the polling loop sees the resource disappear.
//
// `alwaysFound` overrides return404After and forces every call to
// report the resource as still present — the timeout-failure case.
type fakeVerifier struct {
	serverHits     int
	return404After int
	alwaysFound    bool
}

func (f *fakeVerifier) GetServerByID(_ context.Context, _ int) (*hcloud.Server, error) {
	f.serverHits++
	if f.alwaysFound {
		return &hcloud.Server{ID: 1}, nil
	}
	if f.serverHits > f.return404After {
		return nil, hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"}
	}
	return &hcloud.Server{ID: 1}, nil
}

func (f *fakeVerifier) GetSSHKeyByID(_ context.Context, _ int) (*hcloud.SSHKey, error) {
	return nil, hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"}
}

func (f *fakeVerifier) GetPrimaryIPByID(_ context.Context, _ int) (*hcloud.PrimaryIP, error) {
	return nil, hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"}
}

func (f *fakeVerifier) GetFirewallByID(_ context.Context, _ int) (*hcloud.Firewall, error) {
	return nil, hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"}
}

func writeFixtureState(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	// Minimal state.json shape that matches the partial-unmarshal in
	// extractCloudIDs(): a single repository with one cloud server ID.
	const fixture = `{
  "schema_version": "2",
  "repositories": [
    {
      "provider": {
        "cloud": {
          "server_id": "1"
        }
      }
    }
  ]
}`
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestDestroyVerify_Timeout locks D-12 gate 2:
//   - Success path: a fake that returns the server for N polls, then
//     ErrorCodeNotFound, must let run() return nil after at least N
//     poll attempts.
//   - Failure path: a fake that always returns the server must cause
//     run() to return a non-nil error within RUNNERKIT_SMOKE_TIMEOUT
//     seconds.
func TestDestroyVerify_Timeout(t *testing.T) {
	statePath := writeFixtureState(t)

	t.Run("returns nil after resource 404s within timeout", func(t *testing.T) {
		t.Setenv("RUNNERKIT_SMOKE_STATE_FILE", statePath)
		t.Setenv("RUNNERKIT_SMOKE_TIMEOUT", "30") // generous; the 404 should land on poll #4
		fake := &fakeVerifier{return404After: 3}
		if err := run(context.Background(), fake); err != nil {
			t.Fatalf("expected nil after 404; got %v", err)
		}
		if fake.serverHits < 3 {
			t.Fatalf("expected at least 3 polls before 404; got %d", fake.serverHits)
		}
	})

	t.Run("returns timeout error when resource never disappears", func(t *testing.T) {
		t.Setenv("RUNNERKIT_SMOKE_STATE_FILE", statePath)
		t.Setenv("RUNNERKIT_SMOKE_TIMEOUT", "2") // tight bound; loop must give up
		fake := &fakeVerifier{alwaysFound: true}
		err := run(context.Background(), fake)
		if err == nil {
			t.Fatal("expected timeout error when resource never disappears; got nil")
		}
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
			t.Fatalf("error should mention timeout/deadline; got %q", err.Error())
		}
	})
}
