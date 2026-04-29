package github

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/salar/runnerkit/internal/redact"
)

type OSCommandRunner struct{}

func (OSCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (OSCommandRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type AuthOptions struct {
	CommandRunner CommandRunner
	Redactor      *redact.Redactor
	Env           map[string]string
}

type Credential struct {
	Token  string
	Source AuthSource
}

func DiscoverAuth(ctx context.Context, opts AuthOptions) (Credential, error) {
	if opts.Redactor == nil {
		opts.Redactor = redact.New()
	}
	if opts.CommandRunner != nil {
		if _, err := opts.CommandRunner.LookPath("gh"); err == nil {
			// Prefer the GitHub CLI credential: gh auth token
			token, err := opts.CommandRunner.Run(ctx, "gh", "auth", "token")
			if err == nil {
				token = strings.TrimSpace(token)
				if token != "" {
					opts.Redactor.Register(redact.GitHubToken, token)
					return Credential{Token: token, Source: AuthSource{Kind: "gh", Reference: "gh"}}, nil
				}
			}
		}
	}
	if token := strings.TrimSpace(opts.Env["RUNNERKIT_GITHUB_TOKEN"]); token != "" {
		opts.Redactor.Register(redact.GitHubToken, token)
		return Credential{Token: token, Source: AuthSource{Kind: "fine-grained-token", Reference: "RUNNERKIT_GITHUB_TOKEN"}}, nil
	}
	return Credential{}, errors.New("GitHub authentication not found")
}

func FineGrainedTokenRemediation(repo Repo) string {
	fullName := repo.FullName
	if fullName == "" && repo.Owner != "" && repo.Name != "" {
		fullName = repo.Owner + "/" + repo.Name
	}
	if fullName == "" {
		fullName = "owner/name"
	}
	return "Create a fine-grained token scoped only to " + fullName + " with repository Administration read/write and Metadata read, then pass it with RUNNERKIT_GITHUB_TOKEN for this command."
}
