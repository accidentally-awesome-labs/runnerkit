package labels

import (
	"fmt"
	"regexp"
	"strings"

	gh "github.com/salar/runnerkit/internal/github"
)

const (
	DefaultOS              = "linux"
	DefaultArch            = "x64"
	DefaultMode            = "persistent"
	ExampleRepoScopedLabel = "runnerkit-owner-repo"
	SelfHostedAloneWarning = "Do not use runs-on: self-hosted alone for RunnerKit-managed runners."
	maxGeneratedLabelRunes = 63
	fallbackGeneratedSlug  = "repo"
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
