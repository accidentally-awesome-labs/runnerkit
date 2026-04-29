package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/salar/runnerkit/internal/redact"
)

const (
	githubAcceptHeader     = "application/vnd.github+json"
	githubAPIVersionHeader = "2022-11-28"
)

type ClientOptions struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Redactor   *redact.Redactor
}

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	redactor   *redact.Redactor
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return fmt.Sprintf("github API returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("github API returned status %d: %s", e.StatusCode, e.Message)
}

func IsPermissionDenied(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

func NewClient(opts ClientOptions) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	parsed, err := url.Parse(base)
	if err != nil {
		parsed, _ = url.Parse("https://api.github.com")
	}
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	redactor := opts.Redactor
	if redactor == nil {
		redactor = redact.New()
	}
	if opts.Token != "" {
		redactor.Register(redact.GitHubToken, opts.Token)
	}
	return &Client{baseURL: parsed, token: opts.Token, httpClient: client, redactor: redactor}
}

func (c *Client) Repository(ctx context.Context, repo Repo) (Repo, error) {
	var payload struct {
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
		Fork     bool   `json:"fork"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}
	if err := c.do(ctx, http.MethodGet, repoPath(repo), nil, &payload); err != nil {
		return Repo{}, err
	}
	if payload.FullName != "" {
		repo.FullName = payload.FullName
	}
	if payload.Owner.Login != "" {
		repo.Owner = payload.Owner.Login
	}
	if payload.Name != "" {
		repo.Name = payload.Name
	}
	if repo.Host == "" {
		repo.Host = "github.com"
	}
	repo.Private = payload.Private
	repo.Fork = payload.Fork
	return repo, nil
}

func (c *Client) CreateRegistrationToken(ctx context.Context, repo Repo) (RunnerToken, error) {
	return c.createRunnerToken(ctx, repo, "registration-token", redact.RunnerRegistrationToken)
}

func (c *Client) CreateRemovalToken(ctx context.Context, repo Repo) (RunnerToken, error) {
	return c.createRunnerToken(ctx, repo, "remove-token", redact.RunnerRemovalToken)
}

func (c *Client) createRunnerToken(ctx context.Context, repo Repo, endpoint string, kind redact.Kind) (RunnerToken, error) {
	var payload struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := c.do(ctx, http.MethodPost, repoPath(repo)+"/actions/runners/"+endpoint, bytes.NewReader([]byte("{}")), &payload); err != nil {
		return RunnerToken{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil {
		return RunnerToken{}, err
	}
	c.redactor.Register(kind, payload.Token)
	return RunnerToken{Token: payload.Token, ExpiresAt: expiresAt, Kind: kind}, nil
}

func (c *Client) do(ctx context.Context, method string, apiPath string, body io.Reader, into any) error {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(c.baseURL.Path, apiPath)
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", githubAcceptHeader)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersionHeader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorPayload struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(responseBody, &errorPayload)
		return &APIError{StatusCode: resp.StatusCode, Message: c.redactor.String(errorPayload.Message)}
	}
	if into == nil {
		return nil
	}
	return json.Unmarshal(responseBody, into)
}

func repoPath(repo Repo) string {
	owner := repo.Owner
	name := repo.Name
	if (owner == "" || name == "") && repo.FullName != "" {
		parts := strings.SplitN(repo.FullName, "/", 2)
		if len(parts) == 2 {
			owner, name = parts[0], parts[1]
		}
	}
	return "/repos/" + pathEscape(owner) + "/" + pathEscape(name)
}

func pathEscape(value string) string {
	return strings.ReplaceAll(url.PathEscape(value), "+", "%20")
}
