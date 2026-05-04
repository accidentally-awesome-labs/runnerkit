package labels

import (
	"reflect"
	"strings"
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
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

func TestModeConstantsAndEphemeralBuild(t *testing.T) {
	if ModePersistent != "persistent" || ModeEphemeral != "ephemeral" {
		t.Fatalf("mode constants = %q / %q", ModePersistent, ModeEphemeral)
	}
	if DefaultMode != ModePersistent {
		t.Fatalf("DefaultMode = %q, want %q", DefaultMode, ModePersistent)
	}
	set := Build(gh.Repo{Owner: "owner", Name: "repo", FullName: "owner/repo"}, Options{Mode: ModeEphemeral, RunnerName: "runnerkit-owner-repo-ephemeral-abc123"})
	wantLabels := []string{"self-hosted", "runnerkit", "runnerkit-owner-repo", "linux", "x64", "ephemeral"}
	for i, want := range wantLabels {
		if set.Labels[i] != want {
			t.Fatalf("Labels[%d] = %q, want %q (got %v)", i, set.Labels[i], want, set.Labels)
		}
	}
	wantSnippet := "runs-on: [self-hosted, runnerkit, runnerkit-owner-repo, linux, x64, ephemeral]"
	if set.RunsOnYAML != wantSnippet {
		t.Fatalf("RunsOnYAML = %q, want %q", set.RunsOnYAML, wantSnippet)
	}
	if set.RunnerName != "runnerkit-owner-repo-ephemeral-abc123" {
		t.Fatalf("RunnerName = %q", set.RunnerName)
	}
}

func TestRepoScopedLabelReturnsRunnerKitOwnerRepo(t *testing.T) {
	got := RepoScopedLabel(gh.Repo{Owner: "owner", Name: "repo", FullName: "owner/repo"})
	if got != "runnerkit-owner-repo" {
		t.Fatalf("RepoScopedLabel = %q, want runnerkit-owner-repo", got)
	}
	got = RepoScopedLabel(gh.Repo{Owner: "Owner.With Space", Name: "Repo_Name!!", FullName: "Owner.With Space/Repo_Name!!"})
	if got != "runnerkit-owner-with-space-repo-name" {
		t.Fatalf("RepoScopedLabel slug = %q", got)
	}
}

func TestEphemeralRunnerNameAppendsShortID(t *testing.T) {
	got := EphemeralRunnerName(gh.Repo{Owner: "owner", Name: "repo", FullName: "owner/repo"}, "abc123")
	if got != "runnerkit-owner-repo-ephemeral-abc123" {
		t.Fatalf("EphemeralRunnerName = %q, want runnerkit-owner-repo-ephemeral-abc123", got)
	}
}

func TestEphemeralRunnerNameSlugifiesAndCapsLength(t *testing.T) {
	repo := gh.Repo{Owner: strings.Repeat("a", 40), Name: strings.Repeat("b", 40), FullName: strings.Repeat("a", 40) + "/" + strings.Repeat("b", 40)}
	got := EphemeralRunnerName(repo, "abc123")
	if len([]rune(got)) > 63 {
		t.Fatalf("EphemeralRunnerName length = %d, want <=63 (got %q)", len([]rune(got)), got)
	}
	if !strings.HasSuffix(got, "-abc123") {
		t.Fatalf("EphemeralRunnerName missing short id suffix: %q", got)
	}
	if !strings.HasPrefix(got, "runnerkit-") {
		t.Fatalf("EphemeralRunnerName missing prefix: %q", got)
	}
}
