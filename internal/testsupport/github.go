package testsupport

import (
	"context"
	"strings"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
)

type GitHubService struct {
	Repo       gh.Repo
	Permission gh.PermissionStatus

	RegistrationToken gh.RunnerToken
	RemovalToken      gh.RunnerToken
	Runners           []gh.Runner

	RepositoryErr                 error
	VerifyAuthErr                 error
	VerifyRunnerManagementReadErr error
	CreateRegistrationTokenErr    error
	CreateRemovalTokenErr         error
	ListRunnersErr                error
	DeleteRunnerErr               error

	RepositoryCalls                 int
	VerifyAuthCalls                 int
	VerifyRunnerManagementReadCalls int
	CreateRegistrationTokenCalls    int
	CreateRemovalTokenCalls         int
	ListRunnersCalls                int
	DeleteRunnerCalls               int

	DeletedRunnerIDs []int64
	LastRepositoryIn gh.Repo
	LastAuthIn       gh.Repo
	LastDeleteRepo   gh.Repo
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
	if repo.Host == "" {
		repo.Host = "github.com"
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

func (s *GitHubService) VerifyRunnerManagementRead(_ context.Context, repo gh.Repo) (gh.PermissionStatus, error) {
	s.VerifyRunnerManagementReadCalls++
	s.LastAuthIn = repo
	if s.VerifyRunnerManagementReadErr != nil {
		return s.Permission, s.VerifyRunnerManagementReadErr
	}
	if s.Permission.Source.Kind == "" {
		s.Permission.Source = gh.AuthSource{Kind: "gh", Reference: "gh"}
	}
	if !s.Permission.OK && len(s.Permission.Remediation) == 0 {
		s.Permission.OK = true
	}
	return s.Permission, nil
}

func (s *GitHubService) CreateRegistrationToken(_ context.Context, _ gh.Repo) (gh.RunnerToken, error) {
	s.CreateRegistrationTokenCalls++
	if s.CreateRegistrationTokenErr != nil {
		return gh.RunnerToken{}, s.CreateRegistrationTokenErr
	}
	if s.RegistrationToken.Token != "" {
		return s.RegistrationToken, nil
	}
	return gh.RunnerToken{Token: strings.Join([]string{"registration", "token", "testsupport"}, "-"), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (s *GitHubService) CreateRemovalToken(_ context.Context, _ gh.Repo) (gh.RunnerToken, error) {
	s.CreateRemovalTokenCalls++
	if s.CreateRemovalTokenErr != nil {
		return gh.RunnerToken{}, s.CreateRemovalTokenErr
	}
	if s.RemovalToken.Token != "" {
		return s.RemovalToken, nil
	}
	return gh.RunnerToken{Token: strings.Join([]string{"removal", "token", "testsupport"}, "-"), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (s *GitHubService) ListRunners(_ context.Context, _ gh.Repo) ([]gh.Runner, error) {
	s.ListRunnersCalls++
	if s.ListRunnersErr != nil {
		return nil, s.ListRunnersErr
	}
	out := make([]gh.Runner, len(s.Runners))
	copy(out, s.Runners)
	return out, nil
}

func (s *GitHubService) DeleteRunner(_ context.Context, repo gh.Repo, runnerID int64) error {
	s.DeleteRunnerCalls++
	s.LastDeleteRepo = repo
	s.DeletedRunnerIDs = append(s.DeletedRunnerIDs, runnerID)
	if s.DeleteRunnerErr != nil {
		return s.DeleteRunnerErr
	}
	return nil
}
