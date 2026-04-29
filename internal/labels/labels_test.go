package labels

import (
	"reflect"
	"strings"
	"testing"

	gh "github.com/salar/runnerkit/internal/github"
)

func TestBuildOwnerRepoDefaultLabelsAndSnippet(t *testing.T) {
	set := Build(gh.Repo{Owner: "owner", Name: "repo", FullName: "owner/repo"}, Options{})
	wantLabels := []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "persistent"}
	if !reflect.DeepEqual(set.Labels, wantLabels) {
		t.Fatalf("Labels = %#v, want %#v", set.Labels, wantLabels)
	}
	if set.RunnerName != "runnerkit-owner-repo-local" {
		t.Fatalf("RunnerName = %q", set.RunnerName)
	}
	wantSnippet := "runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, persistent]"
	if set.RunsOnYAML != wantSnippet {
		t.Fatalf("RunsOnYAML = %q, want %q", set.RunsOnYAML, wantSnippet)
	}
	if !strings.Contains(set.Warning, "Do not use runs-on: self-hosted alone for RunnerKit-managed runners.") {
		t.Fatalf("missing self-hosted-alone warning: %#v", set)
	}
}

func TestBuildSlugsGeneratedRepoLabel(t *testing.T) {
	set := Build(gh.Repo{Owner: "Owner.With Space", Name: "Repo_Name!!", FullName: "Owner.With Space/Repo_Name!!"}, Options{OS: "linux", Arch: "arm64", Mode: "ephemeral"})
	if set.Labels[2] != "runnerkit-owner-with-space-repo-name" {
		t.Fatalf("repo label = %q", set.Labels[2])
	}
	if set.RunnerName != "runnerkit-owner-with-space-repo-name-local" {
		t.Fatalf("runner name = %q", set.RunnerName)
	}
	if !strings.Contains(set.RunsOnYAML, "arm64") || !strings.Contains(set.RunsOnYAML, "ephemeral") {
		t.Fatalf("snippet missing option labels: %q", set.RunsOnYAML)
	}
}
