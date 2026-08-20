package history

import (
	"context"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

type Inspector struct {
	Commit  Commit
	Files   []string
	Diff    string
	Loading bool
	Error   error
}

func (i Inspector) Summary() string {
	return i.Commit.Short + " · " + i.Commit.Author + " · " + i.Commit.Subject
}

// Inspect loads the changed paths and patch for a commit. The optional parent
// makes merge inspection explicit and produces a parent-relative diff.
func Inspect(ctx context.Context, runner git.Runner, sha, parent string) (Inspector, error) {
	if strings.TrimSpace(sha) == "" {
		return Inspector{}, git.ErrCommandFailed
	}
	args := []string{"show", "--format=", "--name-only", "--no-renames", sha}
	if parent != "" {
		args = []string{"diff", "--name-only", parent, sha}
	}
	pathsResult, err := runner.Run(ctx, args...)
	if err != nil {
		return Inspector{Error: err}, err
	}
	patchArgs := []string{"show", "--format=", "--no-ext-diff", sha}
	if parent != "" {
		patchArgs = []string{"diff", "--no-ext-diff", parent, sha}
	}
	patchResult, err := runner.Run(ctx, patchArgs...)
	if err != nil {
		return Inspector{Files: nonEmptyLines(pathsResult.Stdout), Error: err}, err
	}
	return Inspector{Files: nonEmptyLines(pathsResult.Stdout), Diff: string(patchResult.Stdout)}, nil
}

func nonEmptyLines(data []byte) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
