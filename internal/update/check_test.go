package update

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type errTransport struct{ err error }

func (t errTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, t.err
}

func writeCache(t *testing.T, dir string, c CheckedRelease) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "update-check.json"), raw, 0600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}

func readCache(t *testing.T, dir string) CheckedRelease {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "update-check.json"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var c CheckedRelease
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal cache: %v", err)
	}
	return c
}

// TestMaybePrint_JSONMode_Silent: jsonOutput=true must not hit the network
// or print anything.
func TestMaybePrint_JSONMode_Silent(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "")
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://example/v9","published_at":"2026-04-01T00:00:00Z"}`))
	}))
	defer server.Close()

	var errBuf bytes.Buffer
	MaybePrint(true, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC) },
		StateDir:   t.TempDir(),
		APIURL:     server.URL,
	}, &errBuf)
	if errBuf.Len() != 0 {
		t.Fatalf("JSON mode wrote to errOut: %q", errBuf.String())
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("JSON mode triggered %d HTTP calls; want 0", got)
	}
}

// TestMaybePrint_HonorsCache: a fresh cache (<24h) prevents any HTTP call,
// and the cached newer-version notice prints. After 25h, the next call hits
// the server.
func TestMaybePrint_HonorsCache(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "")
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	stateDir := t.TempDir()

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("ETag", `"after-stale"`)
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://example/v9","published_at":"2026-04-01T00:00:00Z"}`))
	}))
	defer server.Close()

	// Pre-write a fresh (< 24h) cache with a newer version than current.
	writeCache(t, stateDir, CheckedRelease{
		Latest:    "v9.9.9",
		URL:       "https://example/v9",
		ETag:      `"abc"`,
		LastCheck: now.Add(-1 * time.Hour),
	})

	var errBuf bytes.Buffer
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return now },
		StateDir:   stateDir,
		APIURL:     server.URL,
	}, &errBuf)
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("cache hit triggered %d HTTP calls; want 0", got)
	}
	if got := errBuf.String(); !bytes.Contains([]byte(got), []byte("v9.9.9")) {
		t.Fatalf("expected notice mentioning v9.9.9; got %q", got)
	}

	// Now bump the cache age past 24h; the next call must hit the server.
	writeCache(t, stateDir, CheckedRelease{
		Latest:    "v9.9.9",
		URL:       "https://example/v9",
		ETag:      `"abc"`,
		LastCheck: now.Add(-25 * time.Hour),
	})
	errBuf.Reset()
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return now },
		StateDir:   stateDir,
		APIURL:     server.URL,
	}, &errBuf)
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("stale cache should hit server once; got %d", got)
	}
}

// TestMaybePrint_NetworkError_Silent: a transport returning an error must
// not panic, must not print, and must not return an error. Cache is left
// intact.
func TestMaybePrint_NetworkError_Silent(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "")
	client := &http.Client{Transport: errTransport{err: errors.New("net unreachable")}}

	var errBuf bytes.Buffer
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: client,
		Now:        func() time.Time { return time.Now() },
		StateDir:   t.TempDir(),
		APIURL:     "https://example.invalid/api",
	}, &errBuf)
	if errBuf.Len() != 0 {
		t.Fatalf("network error wrote to errOut: %q", errBuf.String())
	}
}

// TestMaybePrint_ConditionalGET: first call writes ETag from response into
// the cache. Second call after TTL sends If-None-Match with the cached ETag
// and accepts a 304 by NOT printing a new notice while bumping LastCheck.
func TestMaybePrint_ConditionalGET(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "")
	stateDir := t.TempDir()
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)

	var lastIfNoneMatch atomic.Value
	lastIfNoneMatch.Store("")
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		lastIfNoneMatch.Store(r.Header.Get("If-None-Match"))
		w.Header().Set("ETag", `"abc123"`)
		if n == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tag_name":"v0.1.0","html_url":"https://example/v0","published_at":"2026-04-01T00:00:00Z"}`))
			return
		}
		// Subsequent calls return 304 Not Modified.
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	// First call: no cache, server returns 200 + ETag, version equals current
	// (no notice expected).
	var errBuf bytes.Buffer
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return now },
		StateDir:   stateDir,
		APIURL:     server.URL,
	}, &errBuf)
	cached := readCache(t, stateDir)
	if cached.ETag != `"abc123"` {
		t.Fatalf("first call did not record ETag; cache=%+v", cached)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("first call wrote unexpected output: %q", errBuf.String())
	}

	// Bump cache LastCheck > 24h so the second call refetches with
	// If-None-Match.
	cached.LastCheck = now.Add(-25 * time.Hour)
	writeCache(t, stateDir, cached)

	errBuf.Reset()
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return now },
		StateDir:   stateDir,
		APIURL:     server.URL,
	}, &errBuf)
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("conditional GET did not refetch; hits=%d", got)
	}
	if got := lastIfNoneMatch.Load().(string); got != `"abc123"` {
		t.Fatalf("If-None-Match header = %q; want %q", got, `"abc123"`)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("304 path wrote unexpected output: %q", errBuf.String())
	}
	c := readCache(t, stateDir)
	if !c.LastCheck.Equal(now) {
		t.Fatalf("304 path did not bump LastCheck; got %v, want %v", c.LastCheck, now)
	}
	if c.Latest != "v0.1.0" {
		t.Fatalf("304 path overwrote Latest unexpectedly: %q", c.Latest)
	}
}

// TestMaybePrint_CISkip: $CI being set must short-circuit before any HTTP
// call.
func TestMaybePrint_CISkip(t *testing.T) {
	t.Setenv("CI", "1")
	t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "")
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	defer server.Close()

	var errBuf bytes.Buffer
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return time.Now() },
		StateDir:   t.TempDir(),
		APIURL:     server.URL,
	}, &errBuf)
	if errBuf.Len() != 0 {
		t.Fatalf("CI=1 wrote to errOut: %q", errBuf.String())
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("CI=1 triggered %d HTTP calls; want 0", got)
	}
}

// TestMaybePrint_NoUpdateNotifier: $RUNNERKIT_NO_UPDATE_NOTIFIER must
// short-circuit before any HTTP call.
func TestMaybePrint_NoUpdateNotifier(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "1")
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	defer server.Close()

	var errBuf bytes.Buffer
	MaybePrint(false, "v0.1.0", Deps{
		HTTPClient: server.Client(),
		Now:        func() time.Time { return time.Now() },
		StateDir:   t.TempDir(),
		APIURL:     server.URL,
	}, &errBuf)
	if errBuf.Len() != 0 {
		t.Fatalf("RUNNERKIT_NO_UPDATE_NOTIFIER=1 wrote to errOut: %q", errBuf.String())
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("RUNNERKIT_NO_UPDATE_NOTIFIER=1 triggered %d HTTP calls; want 0", got)
	}
}
