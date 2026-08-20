package provider

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type PullRequest struct {
	Number      int
	Title       string
	State       string
	Draft       bool
	URL         string
	Base        string
	Head        string
	Mergeable   string
	ReviewState string
	Checks      Checks
	Comments    int
	Reviews     int
}

type Checks struct {
	Total   int
	Passing int
	Failing int
	Pending int
}

func ParsePullRequest(data []byte) (PullRequest, error) {
	var value struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
		HTMLURL string `json:"html_url"`
		Base    struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Mergeable string `json:"mergeable_state"`
		Comments  int    `json:"comments"`
		Reviews   int    `json:"review_comments"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return PullRequest{}, err
	}
	if value.Number < 1 || value.Title == "" {
		return PullRequest{}, errors.New("invalid pull request response")
	}
	return PullRequest{Number: value.Number, Title: value.Title, State: value.State, Draft: value.Draft, URL: value.HTMLURL, Base: value.Base.Ref, Head: value.Head.Ref, Mergeable: value.Mergeable, Comments: value.Comments, Reviews: value.Reviews}, nil
}

type PullRequestClient interface {
	PullRequest(context.Context, Repository, string) (PullRequest, error)
}

type cachedPullRequest struct {
	Value PullRequest
	At    time.Time
}

type PullRequestCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]cachedPullRequest
}

func NewPullRequestCache(ttl time.Duration) *PullRequestCache {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &PullRequestCache{ttl: ttl, items: make(map[string]cachedPullRequest)}
}

func (c *PullRequestCache) Get(ctx context.Context, client PullRequestClient, repository Repository, branch string) (PullRequest, error) {
	key := repository.Host + "/" + repository.Owner + "/" + repository.Name + "@" + branch
	now := time.Now()
	c.mu.Lock()
	if item, ok := c.items[key]; ok && now.Sub(item.At) < c.ttl {
		c.mu.Unlock()
		return item.Value, nil
	}
	c.mu.Unlock()
	value, err := client.PullRequest(ctx, repository, branch)
	if err != nil {
		return PullRequest{}, err
	}
	c.mu.Lock()
	c.items[key] = cachedPullRequest{Value: value, At: now}
	c.mu.Unlock()
	return value, nil
}
