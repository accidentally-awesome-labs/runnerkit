package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var repoTargetPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var scpGitHubRemotePattern = regexp.MustCompile(`^(?:[^@]+@)?github\.com:([^/]+)/(.+)$`)

const repoTargetRemediation = "Pass --repo owner/name from a GitHub repository."

type CommandRunner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type TargetError struct {
	Message     string
	Remediation []string
}

func (e *TargetError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewTargetError(message string) *TargetError {
	return &TargetError{Message: message, Remediation: []string{repoTargetRemediation}}
}

type ResolveOptions struct {
	Repo          string
	RemoteName    string
	CommandRunner CommandRunner
}

type Resolution struct {
	Repo              Repo
	Source            string
	NeedsConfirmation bool
}

func ParseRepo(raw string) (Repo, error) {
	raw = strings.TrimSpace(raw)
	if !repoTargetPattern.MatchString(raw) {
		return Repo{}, NewTargetError(fmt.Sprintf("%q is not a GitHub repository in owner/name form", raw))
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Repo{}, NewTargetError("repository owner and name are required")
	}
	return Repo{Host: "github.com", Owner: parts[0], Name: parts[1], FullName: parts[0] + "/" + parts[1]}, nil
}

func ParseRemote(raw string) (Repo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Repo{}, NewTargetError("git remote URL is empty")
	}
	if match := scpGitHubRemotePattern.FindStringSubmatch(raw); match != nil {
		return repoFromRemoteParts("github.com", match[1], match[2])
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return Repo{}, NewTargetError("git remote URL is not a supported GitHub remote")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" {
		return Repo{}, NewTargetError("git remote host is not github.com")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 {
		return Repo{}, NewTargetError("git remote path must be owner/name")
	}
	return repoFromRemoteParts(host, parts[0], parts[1])
}

func ResolveTarget(ctx context.Context, opts ResolveOptions) (Resolution, error) {
	if strings.TrimSpace(opts.Repo) != "" {
		repo, err := ParseRepo(opts.Repo)
		if err != nil {
			return Resolution{}, err
		}
		return Resolution{Repo: repo, Source: "flag", NeedsConfirmation: false}, nil
	}
	if opts.CommandRunner == nil {
		return Resolution{}, NewTargetError("repository scope is required before auth or state actions apply")
	}
	remoteName := opts.RemoteName
	if remoteName == "" {
		remoteName = "origin"
	}
	remote, err := opts.CommandRunner.Run(ctx, "git", "remote", "get-url", remoteName)
	if err != nil {
		return Resolution{}, NewTargetError("could not read git remote " + remoteName)
	}
	repo, err := ParseRemote(remote)
	if err != nil {
		return Resolution{}, err
	}
	return Resolution{Repo: repo, Source: "git-remote", NeedsConfirmation: true}, nil
}

func repoFromRemoteParts(host, owner, name string) (Repo, error) {
	name = strings.TrimSuffix(name, ".git")
	candidate := owner + "/" + name
	if !repoTargetPattern.MatchString(candidate) {
		return Repo{}, NewTargetError("git remote path must be owner/name")
	}
	repo, err := ParseRepo(candidate)
	if err != nil {
		return Repo{}, err
	}
	repo.Host = host
	return repo, nil
}

func IsTargetError(err error) bool {
	var targetErr *TargetError
	return errors.As(err, &targetErr)
}

func TargetRemediation(err error) []string {
	var targetErr *TargetError
	if errors.As(err, &targetErr) && len(targetErr.Remediation) > 0 {
		return targetErr.Remediation
	}
	return []string{repoTargetRemediation}
}
