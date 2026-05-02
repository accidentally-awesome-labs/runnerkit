package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// CheckedRelease is the cached payload from /releases/latest.
type CheckedRelease struct {
	Latest      string    `json:"latest"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	ETag        string    `json:"etag"`
	LastCheck   time.Time `json:"last_check"`
}

const (
	cacheFileName = "update-check.json"
	cacheTTL      = 24 * time.Hour
	defaultAPIURL = "https://api.github.com/repos/salar/runnerkit/releases/latest"
	httpTimeout   = 5 * time.Second
)

// Deps lets tests inject HTTP, time, and cache dir.
type Deps struct {
	HTTPClient *http.Client
	Now        func() time.Time
	StateDir   string
	// APIURL overrides the GitHub Releases API endpoint. Empty falls back
	// to the production API.
	APIURL string
}

// MaybePrint emits a single non-blocking notice line to errOut if a newer
// release exists. Silent on any error path. Honors:
//   - jsonOutput == true        -> silent (Phase 1 contract)
//   - $CI set                   -> silent (gh CLI convention)
//   - $RUNNERKIT_NO_UPDATE_NOTIFIER set -> silent (per-user opt-out)
//   - last check < 24h ago      -> use cached value, no HTTP
//   - network error             -> silent
//   - response is same tag as current -> silent
//   - non-200/304 response      -> silent
func MaybePrint(jsonOutput bool, currentVersion string, deps Deps, errOut io.Writer) {
	if jsonOutput {
		return
	}
	if os.Getenv("CI") != "" {
		return
	}
	if os.Getenv("RUNNERKIT_NO_UPDATE_NOTIFIER") != "" {
		return
	}
	if deps.HTTPClient == nil {
		deps.HTTPClient = &http.Client{Timeout: httpTimeout}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	apiTarget := deps.APIURL
	if apiTarget == "" {
		apiTarget = defaultAPIURL
	}

	cachePath := ""
	if deps.StateDir != "" {
		cachePath = filepath.Join(deps.StateDir, cacheFileName)
	}

	cached := loadCache(cachePath)
	now := deps.Now()
	var latest CheckedRelease

	// Use cache when fresh.
	if !cached.LastCheck.IsZero() && now.Sub(cached.LastCheck) < cacheTTL {
		latest = cached
	} else {
		// Fetch with conditional GET.
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiTarget, nil)
		if err != nil {
			return
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if cached.ETag != "" {
			req.Header.Set("If-None-Match", cached.ETag)
		}
		resp, err := deps.HTTPClient.Do(req)
		if err != nil {
			return // silent on no-net per D-06
		}
		defer func() { _ = resp.Body.Close() }()
		switch resp.StatusCode {
		case http.StatusNotModified:
			// 304 — payload unchanged; refresh LastCheck.
			cached.LastCheck = now
			saveCache(cachePath, cached)
			latest = cached
		case http.StatusOK:
			var payload struct {
				TagName     string    `json:"tag_name"`
				HTMLURL     string    `json:"html_url"`
				PublishedAt time.Time `json:"published_at"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				return
			}
			latest = CheckedRelease{
				Latest:      payload.TagName,
				URL:         payload.HTMLURL,
				PublishedAt: payload.PublishedAt,
				ETag:        resp.Header.Get("ETag"),
				LastCheck:   now,
			}
			saveCache(cachePath, latest)
		default:
			return // silent on non-200/304
		}
	}

	if latest.Latest == "" {
		return
	}
	if !IsNewer(currentVersion, latest.Latest) {
		return
	}
	fmt.Fprintf(errOut, "runnerkit %s available (you have %s). Run `runnerkit upgrade` for instructions.\n",
		latest.Latest, currentVersion)
}

func loadCache(path string) CheckedRelease {
	if path == "" {
		return CheckedRelease{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return CheckedRelease{}
	}
	var c CheckedRelease
	if err := json.Unmarshal(raw, &c); err != nil {
		return CheckedRelease{}
	}
	return c
}

func saveCache(path string, c CheckedRelease) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return
	}
	// Atomic-ish write: tmp + rename to avoid the cache file race.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}
