package testsupport

import (
	"context"

	gh "github.com/salar/runnerkit/internal/github"
)

type GitHubService struct {
	Repo             gh.Repo
	Permission       gh.PermissionStatus
	RepositoryErr    error
	VerifyAuthErr    error
	RepositoryCalls  int
	VerifyAuthCalls  int
	LastRepositoryIn gh.Repo
	LastAuthIn       gh.Repo
}

func (s *GitHubService) Repository(_ context.Context, repo gh.Repo) (gh.Repo, error) {
	s.RepositoryCalls++
	s.LastRepositoryIn = repo
	if s.RepositoryErr != nil {
		return gh.Repo{}, s.RepositoryErr
	}
	if s.Repo.FullName != "" {
		return s.Repo, nil
	}
	return repo, nil
}

func (s *GitHubService) VerifyAuth(_ context.Context, repo gh.Repo) (gh.PermissionStatus, error) {
	s.VerifyAuthCalls++
	s.LastAuthIn = repo
	if s.VerifyAuthErr != nil {
		return s.Permission, s.VerifyAuthErr
	}
	if s.Permission.Source.Kind == "" {
		s.Permission.Source = gh.AuthSource{Kind: "gh", Reference: "gh"}
	}
	if !s.Permission.OK && len(s.Permission.Remediation) == 0 {
		s.Permission.OK = true
	}
	return s.Permission, nil
}
