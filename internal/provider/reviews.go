package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

const (
	MaxReviewComments = 100
	MaxReviewText     = 16 << 10
)

type ReviewComment struct {
	ID        int64
	Body      string
	Author    string
	Path      string
	Line      int
	Side      string
	DiffHunk  string
	InReplyTo int64
	CreatedAt string
}

type ReviewCommentRequest struct {
	Body      string `json:"body"`
	CommitID  string `json:"commit_id,omitempty"`
	Path      string `json:"path,omitempty"`
	Line      int    `json:"line,omitempty"`
	Side      string `json:"side,omitempty"`
	InReplyTo int64  `json:"in_reply_to,omitempty"`
}

type ReviewCommentClient interface {
	ListReviewComments(context.Context, Repository, int) ([]ReviewComment, error)
	CreateReviewComment(context.Context, Repository, int, ReviewCommentRequest) (ReviewComment, error)
}

type ReviewEvent string

const (
	ReviewEventComment        ReviewEvent = "COMMENT"
	ReviewEventApprove        ReviewEvent = "APPROVE"
	ReviewEventRequestChanges ReviewEvent = "REQUEST_CHANGES"
)

type ReviewSubmission struct {
	Body     string      `json:"body,omitempty"`
	Event    ReviewEvent `json:"event"`
	CommitID string      `json:"commit_id,omitempty"`
}

type ReviewSubmissionResult struct {
	ID      int64
	State   string
	Body    string
	Message string
}

type ReviewDecisionClient interface {
	SubmitReview(context.Context, Repository, int, ReviewSubmission) (ReviewSubmissionResult, error)
}

type MergeMethod string

const (
	MergeMethodMerge  MergeMethod = "merge"
	MergeMethodSquash MergeMethod = "squash"
	MergeMethodRebase MergeMethod = "rebase"
)

type MergeRequest struct {
	Method        MergeMethod `json:"merge_method"`
	ExpectedSHA   string      `json:"sha,omitempty"`
	CommitTitle   string      `json:"commit_title,omitempty"`
	CommitMessage string      `json:"commit_message,omitempty"`
}

type MergeResult struct {
	Merged  bool
	SHA     string
	Message string
}

type MergeClient interface {
	MergePullRequest(context.Context, Repository, int, MergeRequest) (MergeResult, error)
}

// BranchClient deletes a remote branch after an explicitly confirmed provider
// merge. It never changes the local checkout.
type BranchClient interface {
	DeleteBranch(context.Context, Repository, string) error
}

func ParseReviewComments(data []byte) ([]ReviewComment, error) {
	var values []struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Path      string `json:"path"`
		Line      int    `json:"line"`
		Side      string `json:"side"`
		DiffHunk  string `json:"diff_hunk"`
		InReplyTo int64  `json:"in_reply_to_id"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if len(values) > MaxReviewComments {
		return nil, errors.New("review comment page exceeds bound")
	}
	comments := make([]ReviewComment, 0, len(values))
	for _, value := range values {
		if value.ID < 1 || strings.TrimSpace(value.Body) == "" {
			return nil, errors.New("invalid review comment")
		}
		body := value.Body
		if len(body) > MaxReviewText {
			body = body[:MaxReviewText]
		}
		hunk := value.DiffHunk
		if len(hunk) > MaxPatchBytes {
			hunk = hunk[:MaxPatchBytes]
		}
		comments = append(comments, ReviewComment{ID: value.ID, Body: body, Author: value.User.Login, Path: value.Path, Line: value.Line, Side: value.Side, DiffHunk: hunk, InReplyTo: value.InReplyTo, CreatedAt: value.CreatedAt})
	}
	return comments, nil
}

func (r ReviewCommentRequest) Validate() error {
	if strings.TrimSpace(r.Body) == "" || len(r.Body) > MaxReviewText {
		return errors.New("review comment body is required and bounded")
	}
	if r.InReplyTo < 0 || r.Line < 0 || len(r.Path) > 1024 || len(r.CommitID) > 256 {
		return errors.New("review comment target is invalid")
	}
	if r.Side != "" && r.Side != "LEFT" && r.Side != "RIGHT" {
		return errors.New("review comment side is invalid")
	}
	return nil
}

func (r ReviewSubmission) Validate() error {
	if r.Event != ReviewEventComment && r.Event != ReviewEventApprove && r.Event != ReviewEventRequestChanges {
		return errors.New("unsupported review event")
	}
	if len(r.Body) > MaxReviewText || len(r.CommitID) > 256 {
		return errors.New("review submission exceeds bounds")
	}
	if r.Event == ReviewEventRequestChanges && strings.TrimSpace(r.Body) == "" {
		return errors.New("request-changes review requires a body")
	}
	return nil
}

func (r MergeRequest) Validate() error {
	if r.Method != MergeMethodMerge && r.Method != MergeMethodSquash && r.Method != MergeMethodRebase {
		return errors.New("unsupported pull request merge method")
	}
	if len(r.ExpectedSHA) > 256 || len(r.CommitTitle) > MaxReviewText || len(r.CommitMessage) > MaxReviewText {
		return errors.New("pull request merge request exceeds bounds")
	}
	return nil
}
