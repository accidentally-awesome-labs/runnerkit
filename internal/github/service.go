package github

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
)

// ServiceOptions configures the real GitHub service used by production CLI defaults.
type ServiceOptions struct {
	CommandRunner CommandRunner
	Env           map[string]string
	BaseURL       string
	HTTPClient    *http.Client
	Redactor      *redact.Redactor
}

// Service composes auth discovery, repository metadata, and runner-management checks.
type Service struct {
	commandRunner CommandRunner
	env           map[string]string
	baseURL       string
	httpClient    *http.Client
	redactor      *redact.Redactor

	mu         sync.Mutex
	credential *Credential
}

func NewService(opts ServiceOptions) *Service {
	redactor := opts.Redactor
	if redactor == nil {
		redactor = redact.New()
	}
	env := opts.Env
	if env == nil {
		env = map[string]string{"RUNNERKIT_GITHUB_TOKEN": os.Getenv("RUNNERKIT_GITHUB_TOKEN")}
	}
	return &Service{
		commandRunner: opts.CommandRunner,
		env:           env,
		baseURL:       opts.BaseURL,
		httpClient:    opts.HTTPClient,
		redactor:      redactor,
	}
}

func (s *Service) Repository(ctx context.Context, repo Repo) (Repo, error) {
	client, _, err := s.client(ctx)
	if err != nil {
		return Repo{}, err
	}
	return client.Repository(ctx, repo)
}

func (s *Service) VerifyAuth(ctx context.Context, repo Repo) (PermissionStatus, error) {
	client, credential, err := s.client(ctx)
	if err != nil {
		return PermissionStatus{
			OK:          false,
			Required:    []string{"Administration read/write", "Metadata read"},
			Remediation: []string{FineGrainedTokenRemediation(repo)},
		}, err
	}
	return CheckRunnerManagementPermission(ctx, client, repo, credential.Source)
}

func (s *Service) VerifyRunnerManagementRead(ctx context.Context, repo Repo) (PermissionStatus, error) {
	client, credential, err := s.client(ctx)
	if err != nil {
		return PermissionStatus{
			OK:          false,
			Required:    []string{"Administration read/write", "Metadata read"},
			Remediation: []string{FineGrainedTokenRemediation(repo)},
		}, err
	}
	if _, err := client.ListRunners(ctx, repo); err != nil {
		return PermissionStatus{
			OK:          false,
			Source:      credential.Source,
			Required:    []string{"Administration read/write", "Metadata read"},
			Remediation: []string{FineGrainedTokenRemediation(repo)},
		}, err
	}
	return PermissionStatus{
		OK:       true,
		Source:   credential.Source,
		Required: []string{"Administration read/write", "Metadata read"},
	}, nil
}

func (s *Service) CreateRegistrationToken(ctx context.Context, repo Repo) (RunnerToken, error) {
	client, _, err := s.client(ctx)
	if err != nil {
		return RunnerToken{}, err
	}
	return client.CreateRegistrationToken(ctx, repo)
}

func (s *Service) CreateRemovalToken(ctx context.Context, repo Repo) (RunnerToken, error) {
	client, _, err := s.client(ctx)
	if err != nil {
		return RunnerToken{}, err
	}
	return client.CreateRemovalToken(ctx, repo)
}

func (s *Service) ListRunners(ctx context.Context, repo Repo) ([]Runner, error) {
	client, _, err := s.client(ctx)
	if err != nil {
		return nil, err
	}
	return client.ListRunners(ctx, repo)
}

func (s *Service) DeleteRunner(ctx context.Context, repo Repo, runnerID int64) error {
	client, _, err := s.client(ctx)
	if err != nil {
		return err
	}
	return client.DeleteRunner(ctx, repo, runnerID)
}

func (s *Service) client(ctx context.Context) (*Client, Credential, error) {
	credential, err := s.discoverCredential(ctx)
	if err != nil {
		return nil, Credential{}, err
	}
	client := NewClient(ClientOptions{
		BaseURL:    s.baseURL,
		Token:      credential.Token,
		HTTPClient: s.httpClient,
		Redactor:   s.redactor,
	})
	return client, credential, nil
}

func (s *Service) discoverCredential(ctx context.Context) (Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.credential != nil {
		return *s.credential, nil
	}
	credential, err := DiscoverAuth(ctx, AuthOptions{CommandRunner: s.commandRunner, Env: s.env, Redactor: s.redactor})
	if err != nil {
		return Credential{}, err
	}
	s.credential = &credential
	return credential, nil
}
