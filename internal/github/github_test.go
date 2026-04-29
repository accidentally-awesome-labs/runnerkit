package github

import (
	"context"
	"testing"
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
