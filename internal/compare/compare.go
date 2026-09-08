// Package compare provides a bounded, read-only arbitrary-revision comparison
// boundary. It resolves refs before loading details so later work is stable
// even if branches move while a view is open.
package compare

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var (
	ErrInvalidRevision = errors.New("invalid comparison revision")
	ErrMissingRevision = errors.New("comparison revision is required")
)

const (
	DefaultMaxFiles      = 2000
	DefaultMaxPatchBytes = 4 << 20
)

// Runner is the minimal Git boundary required by Compare.
type Runner interface {
	Run(context.Context, ...string) (git.Result, error)
	RunBounded(context.Context, int, ...string) (git.Result, error)
}

// Request identifies two revisions and bounded comparison budgets.
type Request struct {
	Left, Right   string
	MaxFiles      int
	MaxPatchBytes int
}

// Revision is an immutable resolved revision.
type Revision struct {
	Ref, SHA string
}

// Metadata contains bounded commit identity displayed in the comparison.
type Metadata struct {
	SHA, Author, Subject, Date string
}

// Change is one changed path. Rename/copy records retain both paths.
type Change struct {
	Status           string
	OldPath, NewPath string
	Added, Removed   int
	Binary           bool
}

// Result is a read-only comparison snapshot.
type Result struct {
	Left, Right         Revision
	LeftMeta, RightMeta Metadata
	Changes             []Change
	Patch               string
	FilesTruncated      bool
	PatchTruncated      bool
}

// FilePatch loads one bounded path-scoped patch from already-resolved sides.
// The path is passed after -- and is never parsed as a revision or option.
func FilePatch(ctx context.Context, runner Runner, result Result, change Change, maxBytes int) (string, bool, error) {
	path := change.NewPath
	if path == "" {
		path = change.OldPath
	}
	if path == "" {
		return "", false, errors.New("comparison change has no path")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxPatchBytes
	}
	patch, err := runner.RunBounded(ctx, maxBytes, "diff", "--no-ext-diff", result.Left.SHA, result.Right.SHA, "--", path)
	truncated := errors.Is(err, git.ErrOutputLimit)
	if err != nil && !truncated {
		return "", false, err
	}
	return string(patch.Stdout), truncated, nil
}

func Compare(ctx context.Context, runner Runner, request Request) (Result, error) {
	left, err := Resolve(ctx, runner, request.Left)
	if err != nil {
		return Result{}, fmt.Errorf("resolve left revision: %w", err)
	}
	right, err := Resolve(ctx, runner, request.Right)
	if err != nil {
		return Result{}, fmt.Errorf("resolve right revision: %w", err)
	}
	maxFiles := request.MaxFiles
	if maxFiles <= 0 {
		maxFiles = DefaultMaxFiles
	}
	result := Result{Left: left, Right: right}
	result.LeftMeta, err = loadMetadata(ctx, runner, left.SHA)
	if err != nil {
		return Result{}, fmt.Errorf("load left metadata: %w", err)
	}
	result.RightMeta, err = loadMetadata(ctx, runner, right.SHA)
	if err != nil {
		return Result{}, fmt.Errorf("load right metadata: %w", err)
	}
	nameStatus, err := runner.Run(ctx, "diff", "--name-status", "-z", left.SHA, right.SHA, "--")
	if err != nil {
		return Result{}, fmt.Errorf("load changed paths: %w", err)
	}
	numstat, err := runner.Run(ctx, "diff", "--numstat", "-z", left.SHA, right.SHA, "--")
	if err != nil {
		return Result{}, fmt.Errorf("load diff stats: %w", err)
	}
	stats := parseNumstat(numstat.Stdout)
	result.Changes, result.FilesTruncated = parseNameStatus(nameStatus.Stdout, stats, maxFiles)
	maxPatch := request.MaxPatchBytes
	if maxPatch <= 0 {
		maxPatch = DefaultMaxPatchBytes
	}
	patch, patchErr := runner.RunBounded(ctx, maxPatch, "diff", "--no-ext-diff", left.SHA, right.SHA, "--")
	result.Patch = string(patch.Stdout)
	result.PatchTruncated = errors.Is(patchErr, git.ErrOutputLimit)
	if patchErr != nil && !result.PatchTruncated {
		return Result{}, fmt.Errorf("load comparison patch: %w", patchErr)
	}
	return result, nil
}

func Resolve(ctx context.Context, runner Runner, ref string) (Revision, error) {
	if !validRevision(ref) {
		if strings.TrimSpace(ref) == "" {
			return Revision{}, ErrMissingRevision
		}
		return Revision{}, ErrInvalidRevision
	}
	result, err := runner.Run(ctx, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return Revision{}, err
	}
	sha := strings.TrimSpace(string(result.Stdout))
	if sha == "" || strings.ContainsAny(sha, " \t\r\n\x00") {
		return Revision{}, fmt.Errorf("invalid resolved SHA")
	}
	return Revision{Ref: ref, SHA: sha}, nil
}

func loadMetadata(ctx context.Context, runner Runner, sha string) (Metadata, error) {
	result, err := runner.Run(ctx, "show", "-s", "--format=%H%x00%an%x00%s%x00%aI", sha)
	if err != nil {
		return Metadata{}, err
	}
	parts := strings.Split(strings.TrimSuffix(string(result.Stdout), "\n"), "\x00")
	if len(parts) != 4 || parts[0] == "" {
		return Metadata{}, errors.New("malformed commit metadata")
	}
	return Metadata{SHA: parts[0], Author: parts[1], Subject: parts[2], Date: parts[3]}, nil
}

type stat struct {
	Added, Removed int
	Binary         bool
}

func parseNumstat(data []byte) map[string]stat {
	values := strings.Split(string(data), "\x00")
	result := make(map[string]stat)
	for index := 0; index < len(values); index++ {
		if values[index] == "" {
			continue
		}
		fields := strings.SplitN(values[index], "\t", 3)
		if len(fields) != 3 {
			continue
		}
		added, removed := 0, 0
		binary := fields[0] == "-" || fields[1] == "-"
		if !binary {
			added, _ = strconv.Atoi(fields[0])
			removed, _ = strconv.Atoi(fields[1])
		}
		value := stat{Added: added, Removed: removed, Binary: binary}
		result[fields[2]] = value
		// Rename/copy numstat records emit the second path as the next NUL
		// token. Mapping it too keeps the result useful regardless of which
		// side the name-status parser selects for display.
		if index+1 < len(values) && !strings.Contains(values[index+1], "\t") && values[index+1] != "" {
			result[values[index+1]] = value
			index++
		}
	}
	return result
}

func parseNameStatus(data []byte, stats map[string]stat, limit int) ([]Change, bool) {
	values := strings.Split(string(data), "\x00")
	changes := make([]Change, 0, min(limit, len(values)/2))
	for index := 0; index < len(values) && len(changes) < limit; {
		status := values[index]
		index++
		if status == "" {
			continue
		}
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if index+1 >= len(values) {
				break
			}
			oldPath, newPath := values[index], values[index+1]
			index += 2
			value := stats[newPath]
			changes = append(changes, Change{Status: status, OldPath: oldPath, NewPath: newPath, Added: value.Added, Removed: value.Removed, Binary: value.Binary})
			continue
		}
		if index >= len(values) {
			break
		}
		path := values[index]
		index++
		value := stats[path]
		changes = append(changes, Change{Status: status, OldPath: path, NewPath: path, Added: value.Added, Removed: value.Removed, Binary: value.Binary})
	}
	return changes, countNameStatus(data) > len(changes)
}

func countNameStatus(data []byte) int {
	values := strings.Split(string(data), "\x00")
	count := 0
	for index := 0; index < len(values); {
		if values[index] == "" {
			index++
			continue
		}
		count++
		if strings.HasPrefix(values[index], "R") || strings.HasPrefix(values[index], "C") {
			index += 3
		} else {
			index += 2
		}
	}
	return count
}

func validRevision(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, " \t\r\n\x00")
}
