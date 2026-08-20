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
}

func ParseLog(data []byte) []Commit {
	var commits []Commit
	for _, record := range strings.Split(string(data), "\x1e") {
		fields := strings.SplitN(strings.TrimSuffix(record, "\n"), "\x00", 6)
		if len(fields) < 6 || fields[0] == "" {
			continue
		}
		var parents []string
		if fields[4] != "" {
			parents = strings.Fields(fields[4])
		}
		commits = append(commits, Commit{
			SHA: fields[0], Short: fields[1], Author: fields[2],
			Unix: parseInt(fields[3]), Parents: parents, Subject: fields[5],
		})
	}
	return commits
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
	if limit < 1 {
		limit = 100
	}
	result, err := runner.Run(ctx, "log", "-n", fmtInt(limit), "--format=%H%x00%h%x00%an%x00%at%x00%P%x00%s%x1e")
	if err != nil {
		return nil, err
	}
	return ParseLog(result.Stdout), nil
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
