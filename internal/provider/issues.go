package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

const (
	MaxIssues      = 100
	MaxIssueText   = 16 << 10
	MaxIssueLabels = 20
)

type Issue struct {
	Number int
	Title  string
	Body   string
	State  string
	URL    string
	Author string
	Labels []string
}

type IssueCreateRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

type IssueClient interface {
	ListIssues(context.Context, Repository, string, int, int) ([]Issue, error)
	CreateIssue(context.Context, Repository, IssueCreateRequest) (Issue, error)
}

func ParseIssues(data []byte) ([]Issue, error) {
	var values []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		URL    string `json:"html_url"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		PullRequest json.RawMessage `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if len(values) > MaxIssues {
		return nil, errors.New("issue page exceeds bound")
	}
	issues := make([]Issue, 0, len(values))
	for _, value := range values {
		if len(value.PullRequest) > 0 {
			continue
		}
		if value.Number < 1 || strings.TrimSpace(value.Title) == "" {
			return nil, errors.New("invalid issue")
		}
		body := value.Body
		if len(body) > MaxIssueText {
			body = body[:MaxIssueText]
		}
		labels := make([]string, 0, len(value.Labels))
		for _, label := range value.Labels {
			if strings.TrimSpace(label.Name) != "" {
				labels = append(labels, label.Name)
			}
		}
		if len(labels) > MaxIssueLabels {
			return nil, errors.New("issue labels exceed bound")
		}
		issues = append(issues, Issue{Number: value.Number, Title: value.Title, Body: body, State: value.State, URL: value.URL, Author: value.User.Login, Labels: labels})
	}
	return issues, nil
}

func (r IssueCreateRequest) Validate() error {
	if strings.TrimSpace(r.Title) == "" || len(r.Title) > MaxIssueText {
		return errors.New("issue title is required and bounded")
	}
	if len(r.Body) > MaxIssueText || len(r.Labels) > MaxIssueLabels {
		return errors.New("issue request exceeds bound")
	}
	for _, label := range r.Labels {
		if strings.TrimSpace(label) == "" || len(label) > 256 {
			return errors.New("invalid issue label")
		}
	}
	return nil
}
