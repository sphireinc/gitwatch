package app

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/provider"
)

const maxGitHubMergePreflightFailures = 8

type GitHubMergePreflightFailureCode string

const (
	GitHubMergePreflightInvalidTarget       GitHubMergePreflightFailureCode = "invalid_target"
	GitHubMergePreflightDetailUnavailable   GitHubMergePreflightFailureCode = "detail_unavailable"
	GitHubMergePreflightPullMismatch        GitHubMergePreflightFailureCode = "pull_mismatch"
	GitHubMergePreflightNotOpen             GitHubMergePreflightFailureCode = "pull_not_open"
	GitHubMergePreflightDraft               GitHubMergePreflightFailureCode = "pull_is_draft"
	GitHubMergePreflightMissingHeadSHA      GitHubMergePreflightFailureCode = "missing_head_sha"
	GitHubMergePreflightMergeabilityUnknown GitHubMergePreflightFailureCode = "mergeability_not_clean"
	GitHubMergePreflightChecksUnavailable   GitHubMergePreflightFailureCode = "checks_unavailable"
	GitHubMergePreflightChecksFailing       GitHubMergePreflightFailureCode = "checks_failing"
	GitHubMergePreflightChecksPending       GitHubMergePreflightFailureCode = "checks_pending"
	GitHubMergePreflightReviewsUnavailable  GitHubMergePreflightFailureCode = "reviews_unavailable"
	GitHubMergePreflightChangesRequested    GitHubMergePreflightFailureCode = "changes_requested"
)

// GitHubMergeCriticalState contains only the provider state required to decide
// whether the user may confirm a merge. It is fetched directly for each
// preflight and is never read from the normal workspace caches.
type GitHubMergeCriticalState struct {
	Detail  provider.PullRequestDetail
	Checks  provider.ChecksSnapshot
	Reviews provider.ReviewSnapshot
}

type GitHubMergePreflightFailure struct {
	Code GitHubMergePreflightFailureCode
	Err  error
}

func (f GitHubMergePreflightFailure) Error() string {
	if f.Err != nil {
		return f.Err.Error()
	}
	return string(f.Code)
}

// GitHubMergePreflightMsg is tagged with the repository generation and PR
// number so a handler can reject a late result after switching repositories
// or selecting a different pull request.
type GitHubMergePreflightMsg struct {
	Generation uint64
	PullNumber int
	State      GitHubMergeCriticalState
	CanConfirm bool
	Failures   []GitHubMergePreflightFailure
}

// githubMergePreflight starts a direct, uncached refresh of the provider state
// that is safety-critical to confirming a merge.
func (m Model) githubMergePreflight() tea.Cmd {
	return m.githubMergePreflightWithClient(nil)
}

// githubMergePreflightWithClient allows deterministic httptest-backed
// coverage. A nil override uses the normal GitHub token sources.
func (m Model) githubMergePreflightWithClient(clientOverride *provider.GitHubClient) tea.Cmd {
	generation := m.repositoryGeneration
	repository := m.GitHub.Repository
	pullNumber := m.GitHub.Pull.Number
	ctx := m.commandContext()
	tokenEnv := m.GitHubTokenEnv
	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}

	return func() tea.Msg {
		msg := GitHubMergePreflightMsg{Generation: generation, PullNumber: pullNumber}
		if repository.Owner == "" || repository.Name == "" || pullNumber < 1 {
			msg.addFailure(GitHubMergePreflightInvalidTarget, errors.New("GitHub repository or pull request is unavailable"))
			return msg
		}

		client := provider.GitHubClient{TokenSource: provider.FallbackToken{Sources: []provider.TokenSource{provider.CLIToken{}, provider.EnvironmentToken(tokenEnv)}}}
		if clientOverride != nil {
			client = *clientOverride
		}

		detail, err := client.PullRequestDetail(ctx, repository, pullNumber)
		if err != nil {
			msg.addFailure(GitHubMergePreflightDetailUnavailable, err)
			return msg
		}
		msg.State.Detail = detail
		if detail.Number != pullNumber {
			msg.addFailure(GitHubMergePreflightPullMismatch, fmt.Errorf("provider returned PR #%d for requested PR #%d", detail.Number, pullNumber))
			return msg
		}
		if !strings.EqualFold(strings.TrimSpace(detail.State), "open") {
			msg.addFailure(GitHubMergePreflightNotOpen, errors.New("pull request is not open"))
		}
		if detail.Draft {
			msg.addFailure(GitHubMergePreflightDraft, errors.New("draft pull requests cannot be merged"))
		}
		if !validGitHubHeadSHA(detail.HeadSHA) {
			msg.addFailure(GitHubMergePreflightMissingHeadSHA, errors.New("fresh pull request head SHA is missing or invalid"))
		} else {
			checks, checksErr := client.Checks(ctx, repository, detail.HeadSHA)
			if checksErr != nil {
				msg.addFailure(GitHubMergePreflightChecksUnavailable, checksErr)
			} else {
				msg.State.Checks = checks
				if checks.Failing > 0 {
					msg.addFailure(GitHubMergePreflightChecksFailing, fmt.Errorf("%d observed check(s) failed", checks.Failing))
				}
				if checks.Pending > 0 {
					msg.addFailure(GitHubMergePreflightChecksPending, fmt.Errorf("%d observed check(s) are still pending", checks.Pending))
				}
			}
		}
		if !strings.EqualFold(strings.TrimSpace(detail.Mergeable), "clean") {
			msg.addFailure(GitHubMergePreflightMergeabilityUnknown, fmt.Errorf("fresh mergeability is %q, not clean", detail.Mergeable))
		}

		reviews, reviewsErr := client.Reviews(ctx, repository, pullNumber)
		if reviewsErr != nil {
			msg.addFailure(GitHubMergePreflightReviewsUnavailable, reviewsErr)
		} else {
			msg.State.Reviews = reviews
			if reviews.Changes > 0 {
				msg.addFailure(GitHubMergePreflightChangesRequested, fmt.Errorf("%d review(s) request changes", reviews.Changes))
			}
		}
		msg.CanConfirm = len(msg.Failures) == 0
		return msg
	}
}

func (msg *GitHubMergePreflightMsg) addFailure(code GitHubMergePreflightFailureCode, err error) {
	if len(msg.Failures) >= maxGitHubMergePreflightFailures {
		return
	}
	msg.Failures = append(msg.Failures, GitHubMergePreflightFailure{Code: code, Err: err})
}

func validGitHubHeadSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// invalidateGitHubMutationCaches prevents the next workspace load from
// reusing state made stale by a review, reply, merge, branch deletion, or
// workflow action. InvalidateAll also stops an in-flight cached read from
// repopulating these caches after the mutation has completed.
func (m *Model) invalidateGitHubMutationCaches() {
	if m == nil {
		return
	}
	if m.GitHubCache != nil {
		branch := m.Snapshot.Branch.Name
		if branch == "" {
			branch = m.GitHub.Branch
		}
		if branch != "" {
			m.GitHubCache.Invalidate(m.GitHub.Repository, branch)
		}
	}
	if m.GitHubPullsCache != nil {
		m.GitHubPullsCache.InvalidateAll()
	}
	if m.GitHubDetailsCache != nil {
		m.GitHubDetailsCache.InvalidateAll()
	}
	if m.GitHubCommentsCache != nil {
		m.GitHubCommentsCache.InvalidateAll()
	}
	if m.GitHubChecksCache != nil {
		m.GitHubChecksCache.InvalidateAll()
	}
	if m.GitHubReviewsCache != nil {
		m.GitHubReviewsCache.InvalidateAll()
	}
	if m.GitHubIssuesCache != nil {
		m.GitHubIssuesCache.InvalidateAll()
	}
	if m.GitHubReleasesCache != nil {
		m.GitHubReleasesCache.InvalidateAll()
	}
	if m.GitHubWorkflowsCache != nil {
		m.GitHubWorkflowsCache.InvalidateAll()
	}
}
