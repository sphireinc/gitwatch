package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MinimumGitVersion is the oldest Git release supported by gitwatch.
// Git 2.23 introduced the restore and switch commands used by guarded flows.
var MinimumGitVersion = Version{Major: 2, Minor: 23}

// Version is a comparable Git semantic version. Patch suffixes are ignored.
type Version struct {
	Major int
	Minor int
	Patch int
}

// String formats a Git version for diagnostics.
func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// Compare compares v with other, returning -1, 0, or 1.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	return compareInt(v.Patch, other.Patch)
}

// ParseVersion extracts a version from `git version ...` output.
func ParseVersion(output string) (Version, error) {
	match := regexp.MustCompile(`(?m)git version ([0-9]+)\.([0-9]+)(?:\.([0-9]+))?`).FindStringSubmatch(output)
	if len(match) == 0 {
		return Version{}, fmt.Errorf("unrecognized Git version output %q", strings.TrimSpace(output))
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch := 0
	if match[3] != "" {
		patch, _ = strconv.Atoi(match[3])
	}
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// Capabilities describes commands and output contracts available in a Git session.
type Capabilities struct {
	Version           Version
	Restore           bool
	Switch            bool
	WorktreePorcelain bool
	StashFormat       bool
	BranchFormat      bool
	DiffNoIndex       bool
	TrackingMetadata  bool
}

// Probe runs one version command and derives the cached capability set.
func (r Runner) Probe(ctx context.Context) (Capabilities, error) {
	result, err := r.Run(ctx, "--version")
	if err != nil {
		return Capabilities{}, err
	}
	version, err := ParseVersion(string(result.Stdout))
	if err != nil {
		return Capabilities{}, err
	}
	if version.Compare(MinimumGitVersion) < 0 {
		return Capabilities{Version: version}, fmt.Errorf("Git %s is unsupported; gitwatch requires Git %s or newer: %w", version, MinimumGitVersion, ErrUnsupportedGit)
	}
	return Capabilities{
		Version: version, Restore: version.Compare(Version{Major: 2, Minor: 23}) >= 0,
		Switch:            version.Compare(Version{Major: 2, Minor: 23}) >= 0,
		WorktreePorcelain: version.Compare(Version{Major: 2, Minor: 15}) >= 0,
		StashFormat:       version.Compare(Version{Major: 2, Minor: 7}) >= 0,
		BranchFormat:      version.Compare(Version{Major: 2, Minor: 7}) >= 0,
		DiffNoIndex:       true, TrackingMetadata: version.Compare(Version{Major: 2, Minor: 13}) >= 0,
	}, nil
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
