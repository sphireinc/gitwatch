// Package pathhistory loads bounded Git history for one exact path.
package pathhistory

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/history"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
	MaxBytes     = 1 << 20
)

var ErrInvalidPath = errors.New("invalid history path")

type Runner interface {
	RunBounded(context.Context, int, ...string) (git.Result, error)
}

type Request struct {
	Path   string
	Follow bool
	Skip   int
	Limit  int
}

type Entry struct {
	Commit  history.Commit
	Kind    string
	OldPath string
	NewPath string
}

type Page struct {
	Entries []Entry
	HasMore bool
}

func LoadPage(ctx context.Context, runner Runner, request Request) (Page, error) {
	if !validPath(request.Path) || request.Skip < 0 {
		return Page{}, ErrInvalidPath
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	format := "%x1e%H%x00%h%x00%an%x00%at%x00%s\n"
	args := []string{"log", "--skip=" + strconv.Itoa(request.Skip), "-n", strconv.Itoa(limit + 1), "--format=" + format}
	if request.Follow {
		args = append(args, "--follow")
	}
	args = append(args, "--name-status", "-z", "--", request.Path)
	result, err := runner.RunBounded(ctx, MaxBytes, args...)
	truncated := errors.Is(err, git.ErrOutputLimit)
	if err != nil && !truncated {
		return Page{}, err
	}
	entries := parse(result.Stdout)
	page := Page{Entries: entries, HasMore: truncated || len(entries) > limit}
	if len(page.Entries) > limit {
		page.Entries = page.Entries[:limit]
	}
	return page, nil
}

func parse(data []byte) []Entry {
	var entries []Entry
	for _, record := range strings.Split(string(data), "\x1e") {
		record = strings.TrimPrefix(record, "\n")
		if record == "" {
			continue
		}
		headerEnd := strings.IndexByte(record, '\n')
		if headerEnd < 0 {
			continue
		}
		fields := strings.Split(strings.TrimSuffix(record[:headerEnd], "\n"), "\x00")
		if len(fields) < 5 || fields[0] == "" {
			continue
		}
		commit := history.Commit{SHA: fields[0], Short: fields[1], Author: fields[2], Subject: fields[4]}
		if unix, err := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64); err == nil {
			commit.Unix = unix
		}
		paths := strings.Split(record[headerEnd+1:], "\x00")
		for index := 0; index < len(paths); {
			status := strings.TrimSpace(paths[index])
			index++
			if status == "" {
				continue
			}
			if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
				if index+1 >= len(paths) {
					break
				}
				entries = append(entries, Entry{Commit: commit, Kind: status, OldPath: paths[index], NewPath: paths[index+1]})
				index += 2
				continue
			}
			if index < len(paths) {
				entries = append(entries, Entry{Commit: commit, Kind: status, OldPath: paths[index], NewPath: paths[index]})
				index++
			}
		}
	}
	return entries
}

func validPath(path string) bool {
	return path != "" && !strings.ContainsRune(path, '\x00')
}
