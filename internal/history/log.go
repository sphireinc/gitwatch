package history

import (
	"context"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

type Commit struct {
	SHA, Short, Author, Subject string
	Unix                        int64
	Parents                     []string
	Refs                        []string
	Signature                   string
}

func ParseLog(data []byte) []Commit {
	var commits []Commit
	for _, record := range strings.Split(string(data), "\x1e") {
		fields := strings.SplitN(strings.TrimSuffix(record, "\n"), "\x00", 8)
		if len(fields) < 6 || fields[0] == "" {
			continue
		}
		var parents []string
		if fields[4] != "" {
			parents = strings.Fields(fields[4])
		}
		commit := Commit{
			SHA: fields[0], Short: fields[1], Author: fields[2],
			Unix: parseInt(fields[3]), Parents: parents, Subject: fields[5],
		}
		if len(fields) == 8 {
			commit.Refs = splitRefs(fields[5])
			commit.Signature = fields[6]
			commit.Subject = fields[7]
		}
		commits = append(commits, commit)
	}
	return commits
}

func splitRefs(value string) []string {
	var refs []string
	for _, ref := range strings.Split(value, ", ") {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func parseInt(s string) int64 {
	var n int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int64(r-'0')
		}
	}
	return n
}

func LoadLog(ctx context.Context, runner git.Runner, limit int) ([]Commit, error) {
	page, err := LoadPage(ctx, runner, 0, limit)
	return page.Commits, err
}

type Page struct {
	Commits []Commit
	HasMore bool
}

func LoadPage(ctx context.Context, runner git.Runner, skip, limit int) (Page, error) {
	if limit < 1 {
		limit = 100
	}
	if skip < 0 {
		skip = 0
	}
	result, err := runner.Run(ctx, "log", "--skip", fmtInt(skip), "-n", fmtInt(limit+1), "--format=%H%x00%h%x00%an%x00%at%x00%P%x00%D%x00%G?%x00%s%x1e")
	if err != nil {
		return Page{}, err
	}
	commits := ParseLog(result.Stdout)
	page := Page{Commits: commits}
	if len(commits) > limit {
		page.HasMore = true
		page.Commits = commits[:limit]
	}
	return page, nil
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for n > 0 {
		index--
		buffer[index] = byte(n%10) + '0'
		n /= 10
	}
	return string(buffer[index:])
}
