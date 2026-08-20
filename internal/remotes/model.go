package remotes

import (
	"context"
	"net/url"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

type Remote struct {
	Name          string
	FetchURL      string
	PushURL       string
	Reachable     bool
	LastError     string
	Default       bool
	LastFetchUnix int64
}

func List(ctx context.Context, runner git.Runner) ([]Remote, error) {
	result, err := runner.Run(ctx, "remote")
	if err != nil {
		return nil, err
	}
	var remotes []Remote
	for _, name := range nonEmptyLines(result.Stdout) {
		fetch, fetchErr := runner.Run(ctx, "remote", "get-url", name)
		push, pushErr := runner.Run(ctx, "remote", "get-url", "--push", name)
		remote := Remote{Name: name, Reachable: fetchErr == nil && pushErr == nil}
		if fetchErr != nil {
			remote.LastError = fetchErr.Error()
		} else {
			remote.FetchURL = Redact(string(fetch.Stdout))
		}
		if pushErr == nil {
			remote.PushURL = Redact(string(push.Stdout))
		}
		if fetched, fetchStateErr := runner.Run(ctx, "reflog", "show", "-1", "--format=%ct", "refs/remotes/"+name+"/HEAD"); fetchStateErr == nil {
			remote.LastFetchUnix = parseUnix(fetched.Stdout)
		}
		remotes = append(remotes, remote)
	}
	if len(remotes) > 0 {
		remotes[0].Default = true
	}
	return remotes, nil
}

func parseUnix(data []byte) int64 {
	var value int64
	for _, char := range strings.TrimSpace(string(data)) {
		if char >= '0' && char <= '9' {
			value = value*10 + int64(char-'0')
		}
	}
	return value
}

func Redact(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err == nil && parsed.User != nil {
		parsed.User = url.UserPassword(parsed.User.Username(), "REDACTED")
		return parsed.String()
	}
	if index := strings.IndexAny(raw, "?&"); index >= 0 {
		return raw[:index] + "?REDACTED"
	}
	return raw
}

func nonEmptyLines(data []byte) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}
