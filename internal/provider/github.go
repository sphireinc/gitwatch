package provider

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var ErrNoToken = errors.New("no GitHub token configured")

type Repository struct {
	Host  string
	Owner string
	Name  string
}

type TokenSource interface {
	Token() (string, error)
}

type CLIToken struct {
	Binary string
}

func (c CLIToken) Token() (string, error) {
	binary := c.Binary
	if binary == "" {
		binary = "gh"
	}
	command := exec.CommandContext(context.Background(), binary, "auth", "token")
	output, err := command.Output()
	if err != nil {
		return "", ErrNoToken
	}
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

type EnvironmentToken string

func (e EnvironmentToken) Token() (string, error) {
	name := string(e)
	if name == "" {
		name = "GITHUB_TOKEN"
	}
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", ErrNoToken
	}
	return value, nil
}

func ParseGitHubRemote(raw string) (Repository, bool) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "git@github.com:") {
		return parsePath("github.com", strings.TrimPrefix(raw, "git@github.com:"))
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return Repository{}, false
	}
	return parsePath(parsed.Hostname(), parsed.Path)
}

func parsePath(host, path string) (Repository, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		return Repository{}, false
	}
	name := strings.TrimSuffix(parts[1], ".git")
	if parts[0] == "" || name == "" {
		return Repository{}, false
	}
	return Repository{Host: host, Owner: parts[0], Name: name}, true
}
