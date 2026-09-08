// Package blame loads bounded, machine-readable line ownership from Git.
package blame

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

const (
	DefaultLimit = 200
	MaxLimit     = 1000
	MaxBytes     = 2 << 20
)

var ErrInvalidRequest = errors.New("invalid blame request")

type Runner interface {
	RunBounded(context.Context, int, ...string) (git.Result, error)
}

type Request struct {
	Path  string
	Start int
	Limit int
}

type Line struct {
	FinalSHA     string
	OriginalSHA  string
	FinalLine    int
	OriginalLine int
	NumLines     int
	Author       string
	AuthorMail   string
	AuthorTime   int64
	AuthorTZ     string
	Filename     string
	Content      []byte
}

type Page struct {
	Lines     []Line
	Start     int
	HasMore   bool
	Truncated bool
}

func LoadPage(ctx context.Context, runner Runner, request Request) (Page, error) {
	if request.Path == "" || strings.ContainsRune(request.Path, '\x00') || request.Start < 1 {
		return Page{}, ErrInvalidRequest
	}
	limit := request.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	end := request.Start + limit - 1
	result, err := runner.RunBounded(ctx, MaxBytes, "blame", "--line-porcelain", "-L", fmt.Sprintf("%d,%d", request.Start, end), "--", request.Path)
	truncated := errors.Is(err, git.ErrOutputLimit)
	if err != nil && !truncated {
		return Page{}, err
	}
	lines := Parse(result.Stdout)
	return Page{Lines: lines, Start: request.Start, HasMore: truncated || len(lines) == limit, Truncated: truncated}, nil
}

func Parse(data []byte) []Line {
	var lines []Line
	var current *Line
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		if len(raw) > 0 && raw[0] == '\t' {
			if current != nil {
				current.Content = append([]byte(nil), raw[1:]...)
				lines = append(lines, *current)
				current = nil
			}
			continue
		}
		if len(raw) == 0 {
			continue
		}
		fields := bytes.Fields(raw)
		if len(fields) >= 3 && isSHA(fields[0]) {
			finalLine, okFinal := parseInt(fields[2])
			originalLine, okOriginal := parseInt(fields[1])
			if !okFinal || !okOriginal {
				continue
			}
			current = &Line{FinalSHA: string(fields[0]), OriginalSHA: string(fields[0]), FinalLine: finalLine, OriginalLine: originalLine, NumLines: 1}
			if len(fields) >= 4 {
				if count, ok := parseInt(fields[3]); ok {
					current.NumLines = count
				}
			}
			continue
		}
		if current == nil {
			continue
		}
		key, value, ok := splitHeader(raw)
		if !ok {
			continue
		}
		switch key {
		case "author":
			current.Author = string(value)
		case "author-mail":
			current.AuthorMail = string(value)
		case "author-time":
			if timestamp, ok := parseInt(value); ok {
				current.AuthorTime = int64(timestamp)
			}
		case "author-tz":
			current.AuthorTZ = string(value)
		case "filename":
			current.Filename = string(value)
		}
	}
	return lines
}

func splitHeader(raw []byte) (string, []byte, bool) {
	index := bytes.IndexByte(raw, ' ')
	if index < 1 {
		return "", nil, false
	}
	return string(raw[:index]), raw[index+1:], true
}

func parseInt(value []byte) (int, bool) {
	n, err := strconv.Atoi(string(value))
	return n, err == nil
}

func isSHA(value []byte) bool {
	if len(value) < 7 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9') && !(char >= 'a' && char <= 'f') && !(char >= 'A' && char <= 'F') {
			return false
		}
	}
	return true
}
