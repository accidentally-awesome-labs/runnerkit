package github

import (
	"context"
	"time"

	"github.com/salar/runnerkit/internal/redact"
)

type RunnerToken struct {
	Token     string
	ExpiresAt time.Time
	Kind      redact.Kind
}

type RunnerTokenProvider interface {
	CreateRegistrationToken(ctx context.Context, repo Repo) (RunnerToken, error)
	CreateRemovalToken(ctx context.Context, repo Repo) (RunnerToken, error)
}

func CheckRunnerManagementPermission(ctx context.Context, provider RunnerTokenProvider, repo Repo, source AuthSource) (PermissionStatus, error) {
	_, err := provider.CreateRegistrationToken(ctx, repo)
	if err != nil {
		return PermissionStatus{
			OK:          false,
			Source:      source,
			Required:    []string{"Administration read/write", "Metadata read"},
			Remediation: []string{FineGrainedTokenRemediation(repo)},
		}, err
	}
	return PermissionStatus{
		OK:       true,
		Source:   source,
		Required: []string{"Administration read/write", "Metadata read"},
	}, nil
}
