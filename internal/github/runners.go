package github

import (
	"context"
	"net/http"
	"strconv"
)

type Runner struct {
	ID     int64
	Name   string
	OS     string
	Status string
	Busy   bool
	Labels []string
}

type RunnerManager interface {
	CreateRegistrationToken(ctx context.Context, repo Repo) (RunnerToken, error)
	CreateRemovalToken(ctx context.Context, repo Repo) (RunnerToken, error)
	ListRunners(ctx context.Context, repo Repo) ([]Runner, error)
	DeleteRunner(ctx context.Context, repo Repo, runnerID int64) error
}

func (c *Client) ListRunners(ctx context.Context, repo Repo) ([]Runner, error) {
	var payload struct {
		TotalCount int `json:"total_count"`
		Runners    []struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			OS     string `json:"os"`
			Status string `json:"status"`
			Busy   bool   `json:"busy"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		} `json:"runners"`
	}
	if err := c.do(ctx, http.MethodGet, repoPath(repo)+"/actions/runners", nil, &payload); err != nil {
		return nil, err
	}
	runners := make([]Runner, 0, len(payload.Runners))
	for _, item := range payload.Runners {
		labels := make([]string, 0, len(item.Labels))
		for _, label := range item.Labels {
			labels = append(labels, label.Name)
		}
		runners = append(runners, Runner{ID: item.ID, Name: item.Name, OS: item.OS, Status: item.Status, Busy: item.Busy, Labels: labels})
	}
	return runners, nil
}

func (c *Client) DeleteRunner(ctx context.Context, repo Repo, runnerID int64) error {
	return c.do(ctx, http.MethodDelete, repoPath(repo)+"/actions/runners/"+formatRunnerID(runnerID), nil, nil)
}

func FindRunnerByName(runners []Runner, name string) (Runner, bool) {
	for _, runner := range runners {
		if runner.Name == name {
			return runner, true
		}
	}
	return Runner{}, false
}

func formatRunnerID(id int64) string {
	return strconv.FormatInt(id, 10)
}
