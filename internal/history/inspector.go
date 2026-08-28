package history

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var ErrInvalidRef = errors.New("invalid history ref")

func ResolveRef(ctx context.Context, runner git.Runner, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\r\n\x00") {
		return "", ErrInvalidRef
	}
	result, err := runner.Run(ctx, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(result.Stdout)), nil
}

type Inspector struct {
	Commit  Commit
	Parent  string
	Files   []string
	Stats   []FileStat
	Diff    string
	Loading bool
	Error   error
}

type FileStat struct {
	Path           string
	Added, Deleted int
	Binary         bool
}

func (i Inspector) Summary() string {
	return i.Commit.Short + " · " + i.Commit.Author + " · " + i.Commit.Subject
}

// Inspect loads the changed paths and patch for a commit. The optional parent
// makes merge inspection explicit and produces a parent-relative diff.
func Inspect(ctx context.Context, runner git.Runner, sha, parent string) (Inspector, error) {
	return InspectPath(ctx, runner, sha, parent, "")
}

func InspectPath(ctx context.Context, runner git.Runner, sha, parent, path string) (Inspector, error) {
	if strings.TrimSpace(sha) == "" {
		return Inspector{}, git.ErrCommandFailed
	}
	if strings.HasPrefix(strings.TrimSpace(sha), "-") || strings.ContainsAny(path, "\r\n\x00") {
		return Inspector{}, git.ErrCommandFailed
	}
	args := []string{"show", "--format=", "--numstat", "-z", "--no-renames", sha}
	if parent != "" {
		args = []string{"diff", "--numstat", "-z", "--no-renames", parent, sha}
	}
	if path != "" {
		args = append(args, "--", path)
	}
	pathsResult, err := runner.Run(ctx, args...)
	if err != nil {
		return Inspector{Error: err}, err
	}
	patchArgs := []string{"show", "--format=", "--no-ext-diff", sha}
	if parent != "" {
		patchArgs = []string{"diff", "--no-ext-diff", parent, sha}
	}
	if path != "" {
		patchArgs = append(patchArgs, "--", path)
	}
	patchResult, err := runner.Run(ctx, patchArgs...)
	if err != nil {
		return Inspector{Files: statPaths(pathsResult.Stdout), Stats: parseStats(pathsResult.Stdout), Error: err}, err
	}
	return Inspector{Commit: Commit{SHA: sha}, Parent: parent, Files: statPaths(pathsResult.Stdout), Stats: parseStats(pathsResult.Stdout), Diff: string(patchResult.Stdout)}, nil
}

func statPaths(data []byte) []string {
	var lines []string
	for _, stat := range parseStats(data) {
		if stat.Path != "" {
			lines = append(lines, stat.Path)
		}
	}
	return lines
}

func parseStats(data []byte) []FileStat {
	if strings.IndexByte(string(data), 0) >= 0 {
		return parseStatsZ(data)
	}
	var stats []FileStat
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		stat := FileStat{Path: fields[2]}
		if fields[0] == "-" || fields[1] == "-" {
			stat.Binary = true
		} else {
			stat.Added = int(parseInt(fields[0]))
			stat.Deleted = int(parseInt(fields[1]))
		}
		stats = append(stats, stat)
	}
	return stats
}

func parseStatsZ(data []byte) []FileStat {
	var stats []FileStat
	for _, record := range strings.Split(string(data), "\x00") {
		fields := strings.SplitN(record, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		stat := FileStat{Path: fields[2]}
		if fields[0] == "-" || fields[1] == "-" {
			stat.Binary = true
		} else {
			stat.Added = int(parseInt(fields[0]))
			stat.Deleted = int(parseInt(fields[1]))
		}
		stats = append(stats, stat)
	}
	return stats
}
