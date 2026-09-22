// Package health derives bounded repository health from authoritative local
// state. It deliberately avoids reducing unrelated concerns to one score.
package health

import (
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

type Severity string

const (
	SeverityHealthy  Severity = "healthy"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Summary struct {
	Severity        Severity
	Dirty           int
	Conflicts       int
	Ahead           int
	Behind          int
	Unpushed        int
	Stashes         int
	Worktrees       int
	ActiveOperation string
	SubmoduleIssues int
	Attention       []string
	FreshAt         time.Time
	Source          string
}

func Compute(snapshot repo.Snapshot, stashes, worktrees int, warnings []string) Summary {
	return ComputeWithSubmoduleIssues(snapshot, stashes, worktrees, CountSubmoduleIssues(snapshot), warnings)
}

// ComputeWithSubmoduleIssues derives health while allowing callers that load
// submodule metadata separately to provide its authoritative issue count.
func ComputeWithSubmoduleIssues(snapshot repo.Snapshot, stashes, worktrees, submoduleIssues int, warnings []string) Summary {
	summary := Summary{Dirty: snapshot.Counts.Staged + snapshot.Counts.Unstaged + snapshot.Counts.Untracked, Conflicts: snapshot.Counts.Conflicted, Ahead: snapshot.Branch.Ahead, Behind: snapshot.Branch.Behind, Unpushed: snapshot.Branch.Ahead, Stashes: stashes, Worktrees: worktrees, SubmoduleIssues: submoduleIssues, FreshAt: snapshot.ObservedAt, Source: "git status"}
	if snapshot.Operation != nil {
		summary.ActiveOperation = snapshot.Operation.Kind().String()
	}
	if summary.Conflicts > 0 {
		summary.Severity = SeverityCritical
		summary.Attention = append(summary.Attention, "conflicts")
	} else {
		summary.Severity = SeverityHealthy
	}
	if summary.ActiveOperation != "" {
		summary.Severity = maxSeverity(summary.Severity, SeverityWarning)
		summary.Attention = append(summary.Attention, summary.ActiveOperation)
	}
	if summary.Dirty > 0 || summary.Ahead > 0 || summary.Behind > 0 {
		summary.Severity = maxSeverity(summary.Severity, SeverityInfo)
	}
	if summary.SubmoduleIssues > 0 {
		summary.Severity = maxSeverity(summary.Severity, SeverityWarning)
		summary.Attention = append(summary.Attention, "submodules")
	}
	if len(warnings) > 0 {
		summary.Severity = maxSeverity(summary.Severity, SeverityWarning)
		for _, warning := range warnings {
			if strings.TrimSpace(warning) != "" {
				summary.Attention = append(summary.Attention, warning)
			}
		}
	}
	return summary
}

// CountSubmoduleIssues counts dirty submodule status fields from Git's
// porcelain-v2 snapshot. A four-character all-dot value is clean; any other
// populated submodule field represents changed, modified, or untracked
// submodule content.
func CountSubmoduleIssues(snapshot repo.Snapshot) int {
	count := 0
	for _, entry := range snapshot.Entries {
		value := strings.TrimPrefix(entry.Submodule, "S")
		if value == "" || strings.Trim(value, ".") == "" {
			continue
		}
		count++
	}
	return count
}

func maxSeverity(left, right Severity) Severity {
	rank := func(value Severity) int {
		switch value {
		case SeverityCritical:
			return 3
		case SeverityWarning:
			return 2
		case SeverityInfo:
			return 1
		default:
			return 0
		}
	}
	if rank(right) > rank(left) {
		return right
	}
	return left
}
