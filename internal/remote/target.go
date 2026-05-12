package remote

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Target describes a BYO SSH target without embedding secret material.
type Target struct {
	Host    string
	User    string
	Port    int
	KeyPath string
	Raw     string
}

// ParseTarget parses user@host, user@host:port, or ssh://user@host:port.
func ParseTarget(raw string, defaultPort int) (Target, error) {
	raw = strings.TrimSpace(raw)
	if defaultPort == 0 {
		defaultPort = 22
	}
	if defaultPort < 1 || defaultPort > 65535 {
		return Target{}, fmt.Errorf("invalid default SSH port %d", defaultPort)
	}
	if raw == "" {
		return Target{}, fmt.Errorf("SSH target is required")
	}
	if strings.HasPrefix(raw, "ssh://") {
		return parseURLTarget(raw, defaultPort)
	}
	at := strings.LastIndex(raw, "@")
	if at <= 0 {
		return Target{}, fmt.Errorf("SSH target must include user@host")
	}
	user := strings.TrimSpace(raw[:at])
	hostPort := strings.TrimSpace(raw[at+1:])
	if user == "" {
		return Target{}, fmt.Errorf("SSH target missing user")
	}
	if hostPort == "" {
		return Target{}, fmt.Errorf("SSH target missing host")
	}
	host := hostPort
	port := defaultPort
	if strings.Contains(hostPort, ":") {
		parts := strings.Split(hostPort, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return Target{}, fmt.Errorf("invalid SSH target host or port")
		}
		host = parts[0]
		parsedPort, err := parsePort(parts[1])
		if err != nil {
			return Target{}, err
		}
		port = parsedPort
	}
	if strings.TrimSpace(host) == "" {
		return Target{}, fmt.Errorf("SSH target missing host")
	}
	return Target{Host: host, User: user, Port: port, Raw: raw}, nil
}

func parseURLTarget(raw string, defaultPort int) (Target, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Target{}, err
	}
	if parsed.Scheme != "ssh" {
		return Target{}, fmt.Errorf("unsupported SSH target scheme %q", parsed.Scheme)
	}
	user := parsed.User.Username()
	if strings.TrimSpace(user) == "" {
		return Target{}, fmt.Errorf("SSH target missing user")
	}
	host := parsed.Hostname()
	if strings.TrimSpace(host) == "" {
		return Target{}, fmt.Errorf("SSH target missing host")
	}
	port := defaultPort
	if parsed.Port() != "" {
		parsedPort, err := parsePort(parsed.Port())
		if err != nil {
			return Target{}, err
		}
		port = parsedPort
	}
	return Target{Host: host, User: user, Port: port, Raw: raw}, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("SSH target port must be numeric")
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("SSH target port must be between 1 and 65535")
	}
	return port, nil
}

func (t Target) Address() string {
	return fmt.Sprintf("%s:%d", t.Host, t.Port)
}

func (t Target) Display() string {
	return fmt.Sprintf("%s@%s:%d", t.User, t.Host, t.Port)
}

// CanonicalHostKey parses raw (user@host or user@host:port) with fallbackPort
// when the port is omitted, then returns the normalized Display() form for
// comparisons (SEED-002 list --host filtering).
func CanonicalHostKey(raw string, fallbackPort int) (string, error) {
	t, err := ParseTarget(strings.TrimSpace(raw), fallbackPort)
	if err != nil {
		return "", err
	}
	return t.Display(), nil
}
