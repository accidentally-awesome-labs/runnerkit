package labels

import (
	"fmt"
	"regexp"
	"strings"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
)

const (
	DefaultOS  = "linux"
	DefaultArch = "x64"

	// ModePersistent and ModeEphemeral are the canonical mode label values
	// users see in the workflow `runs-on` snippet. They mirror the values
	// in internal/runmode so labels stay loosely coupled and tests in this
	// package do not import runmode.
	ModePersistent = "persistent"
	ModeEphemeral  = "ephemeral"

	// DefaultMode preserves backwards-compatible persistent BYO/cloud
	// behavior: runner name `runnerkit-owner-repo-local` and the persistent
	// label set. Phase 5 must not change persistent defaults.
	DefaultMode = ModePersistent

	ExampleRepoScopedLabel = "runnerkit-owner-repo"
	SelfHostedAloneWarning = "Do not use runs-on: self-hosted alone for RunnerKit-managed runners."
	maxGeneratedLabelRunes = 63
	fallbackGeneratedSlug  = "repo"

	// ephemeralRunnerSeparator separates the runnerkit repo-scoped label
	// from the short id in EphemeralRunnerName.
	ephemeralRunnerSeparator = "-ephemeral-"
)

type Options struct {
	OS          string
	Arch        string
	Mode        string
	RunnerName  string
	ExtraLabels []string
}

type LabelSet struct {
	RunnerName string
	Labels     []string
	RunsOnYAML string
	Warning    string
}

func Build(repo gh.Repo, opts Options) LabelSet {
	owner, name := repo.Owner, repo.Name
	if (owner == "" || name == "") && repo.FullName != "" {
		parts := strings.SplitN(repo.FullName, "/", 2)
		if len(parts) == 2 {
			owner, name = parts[0], parts[1]
		}
	}
	osLabel := defaultString(opts.OS, DefaultOS)
	archLabel := defaultString(opts.Arch, DefaultArch)
	modeLabel := defaultString(opts.Mode, DefaultMode)
	repoLabel := capGeneratedLabel("runnerkit-" + slug(owner) + "-" + slug(name))
	if repoLabel == "runnerkit--" || repoLabel == "runnerkit-" || repoLabel == "" {
		repoLabel = "runnerkit-" + fallbackGeneratedSlug
	}
	labels := []string{"self-hosted", "runnerkit", repoLabel, slug(osLabel), slug(archLabel), slug(modeLabel)}
	for _, extra := range opts.ExtraLabels {
		if label := slug(extra); label != "" {
			labels = append(labels, capGeneratedLabel(label))
		}
	}
	runnerName := opts.RunnerName
	if runnerName == "" {
		// Future collision hook: append a deterministic suffix only after GitHub duplicate detection exists.
		runnerName = repoLabel + "-local"
	}
	return LabelSet{RunnerName: runnerName, Labels: labels, RunsOnYAML: WorkflowSnippet(labels), Warning: SelfHostedAloneWarning}
}

func WorkflowSnippet(labels []string) string {
	return fmt.Sprintf("runs-on: [%s]", strings.Join(labels, ", "))
}

// RepoScopedLabel returns the canonical RunnerKit repo-scoped label value
// `runnerkit-owner-repo` for the supplied repo. It applies the same slug
// and length capping rules as Build so callers can derive label values
// (for ephemeral runner names, status output, etc.) without rebuilding
// the full label set.
func RepoScopedLabel(repo gh.Repo) string {
	owner, name := repo.Owner, repo.Name
	if (owner == "" || name == "") && repo.FullName != "" {
		parts := strings.SplitN(repo.FullName, "/", 2)
		if len(parts) == 2 {
			owner, name = parts[0], parts[1]
		}
	}
	label := capGeneratedLabel("runnerkit-" + slug(owner) + "-" + slug(name))
	if label == "runnerkit--" || label == "runnerkit-" || label == "" {
		label = "runnerkit-" + fallbackGeneratedSlug
	}
	return label
}

// EphemeralRunnerName builds a runner name suitable for ephemeral
// registration: `runnerkit-owner-repo-ephemeral-<short-id>`. The combined
// length is capped at the GitHub-runner-name limit while still preserving
// the short id suffix so multiple ephemeral runners avoid collisions.
//
// Persistent runner names continue to use Build's `runnerkit-owner-repo-local`
// suffix; this helper exists only for ephemeral mode.
func EphemeralRunnerName(repo gh.Repo, shortID string) string {
	repoLabel := RepoScopedLabel(repo)
	suffix := ephemeralRunnerSeparator + slug(shortID)
	// Reserve room for the suffix so the short id is always preserved.
	maxRepoRunes := maxGeneratedLabelRunes - len([]rune(suffix))
	if maxRepoRunes < 0 {
		maxRepoRunes = 0
	}
	if len([]rune(repoLabel)) > maxRepoRunes {
		runes := []rune(repoLabel)
		repoLabel = strings.Trim(string(runes[:maxRepoRunes]), "-")
	}
	return repoLabel + suffix
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonAlnum.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}

func capGeneratedLabel(value string) string {
	if len([]rune(value)) <= maxGeneratedLabelRunes {
		return value
	}
	runes := []rune(value)
	value = string(runes[:maxGeneratedLabelRunes])
	return strings.Trim(value, "-")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
