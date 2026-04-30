package hetzner

import (
	"os"
	"strings"
)

const (
	EnvHCLOUDToken       = "HCLOUD_TOKEN"
	EnvHetznerCloudToken = "HETZNER_CLOUD_TOKEN"
)

type TokenSource struct {
	Source string `json:"source"`
	Token  string `json:"-"`
}

type MissingTokenError struct {
	Remediation []string
}

func (e *MissingTokenError) Error() string {
	return "Hetzner credentials are missing"
}

func ResolveToken(env map[string]string) (TokenSource, error) {
	if env == nil {
		env = map[string]string{
			EnvHCLOUDToken:       os.Getenv(EnvHCLOUDToken),
			EnvHetznerCloudToken: os.Getenv(EnvHetznerCloudToken),
		}
	}
	if token := strings.TrimSpace(env[EnvHCLOUDToken]); token != "" {
		return TokenSource{Source: EnvHCLOUDToken, Token: token}, nil
	}
	if token := strings.TrimSpace(env[EnvHetznerCloudToken]); token != "" {
		return TokenSource{Source: EnvHetznerCloudToken, Token: token}, nil
	}
	return TokenSource{}, &MissingTokenError{Remediation: []string{
		"Export HCLOUD_TOKEN=<token from Hetzner Cloud Console>",
		"Re-run runnerkit up --repo owner/name --cloud hetzner",
	}}
}
