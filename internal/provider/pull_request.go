package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

const MaxPullRequests = 100
const MaxPullRequestFiles = 100
const MaxPullRequestCommits = 100
const MaxPatchBytes = 64 << 10

func ValidateCheckoutRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" || len(ref) > 256 || strings.HasPrefix(ref, "-") || strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") || strings.Contains(ref, "..") || strings.ContainsAny(ref, "~^:?*[\\\r\n\x00 \t\x1b") {
		return errors.New("invalid provider checkout ref")
	}
	return nil
}

type PullRequest struct {
	Number      int
	Title       string
	State       string
	Draft       bool
	URL         string
	Base        string
	Head        string
	HeadSHA     string
	Mergeable   string
	ReviewState string
	Checks      Checks
	Comments    int
	Reviews     int
	Body        string
}

type PullRequestCommit struct {
	SHA     string
	Message string
	Author  string
}

type PullRequestFile struct {
	Path      string
	Status    string
	Additions int
	Deletions int
	Changes   int
	Patch     string
}

type PullRequestDetail struct {
	PullRequest
	Commits []PullRequestCommit
	Files   []PullRequestFile
}

type Checks struct {
	Total   int
	Passing int
	Failing int
	Pending int
}

type ReviewSnapshot struct {
	Approved  int
	Changes   int
	Commented int
}

func (s ReviewSnapshot) State() string {
	if s.Changes > 0 {
		return "changes requested"
	}
	if s.Approved > 0 {
		return "approved"
	}
	if s.Commented > 0 {
		return "commented"
	}
	return "pending"
}

func ParseReviews(data []byte) (ReviewSnapshot, error) {
	var reviews []struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(data, &reviews); err != nil {
		return ReviewSnapshot{}, err
	}
	snapshot := ReviewSnapshot{}
	for _, review := range reviews {
		switch strings.ToUpper(review.State) {
		case "APPROVED":
			snapshot.Approved++
		case "CHANGES_REQUESTED":
			snapshot.Changes++
		case "COMMENTED":
			snapshot.Commented++
		}
	}
	return snapshot, nil
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
			SHA string `json:"sha"`
		} `json:"head"`
		Mergeable string `json:"mergeable_state"`
		Comments  int    `json:"comments"`
		Reviews   int    `json:"review_comments"`
		Body      string `json:"body"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return PullRequest{}, err
	}
	if value.Number < 1 || value.Title == "" {
		return PullRequest{}, errors.New("invalid pull request response")
	}
	if len(value.Body) > MaxPatchBytes {
		value.Body = value.Body[:MaxPatchBytes]
	}
	return PullRequest{Number: value.Number, Title: value.Title, State: value.State, Draft: value.Draft, URL: value.HTMLURL, Base: value.Base.Ref, Head: value.Head.Ref, HeadSHA: value.Head.SHA, Mergeable: value.Mergeable, Comments: value.Comments, Reviews: value.Reviews, Body: value.Body}, nil
}

type PullRequestClient interface {
	PullRequest(context.Context, Repository, string) (PullRequest, error)
}

// PullRequestListClient is the provider-neutral bounded listing surface used
// by a future PR workspace. Providers must not make local Git depend on it.
type PullRequestListClient interface {
	ListPullRequests(context.Context, Repository, int, int) ([]PullRequest, error)
}

type PullRequestDetailClient interface {
	PullRequestDetail(context.Context, Repository, int) (PullRequestDetail, error)
}

type PullRequestCreateRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft"`
}

type PullRequestCreator interface {
	CreatePullRequest(context.Context, Repository, PullRequestCreateRequest) (PullRequest, error)
}

func ParsePullRequestCommits(data []byte) ([]PullRequestCommit, error) {
	var values []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if len(values) > MaxPullRequestCommits {
		return nil, errors.New("pull request commit page exceeds bound")
	}
	commits := make([]PullRequestCommit, 0, len(values))
	for _, value := range values {
		if value.SHA == "" {
			return nil, errors.New("invalid pull request commit")
		}
		message := value.Commit.Message
		if len(message) > MaxPatchBytes {
			message = message[:MaxPatchBytes]
		}
		commits = append(commits, PullRequestCommit{SHA: value.SHA, Message: message, Author: value.Commit.Author.Name})
	}
	return commits, nil
}

func ParsePullRequestFiles(data []byte) ([]PullRequestFile, error) {
	var values []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Changes   int    `json:"changes"`
		Patch     string `json:"patch"`
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if len(values) > MaxPullRequestFiles {
		return nil, errors.New("pull request file page exceeds bound")
	}
	files := make([]PullRequestFile, 0, len(values))
	for _, value := range values {
		if value.Filename == "" {
			return nil, errors.New("invalid pull request file")
		}
		if len(value.Patch) > MaxPatchBytes {
			value.Patch = value.Patch[:MaxPatchBytes]
		}
		files = append(files, PullRequestFile{Path: value.Filename, Status: value.Status, Additions: value.Additions, Deletions: value.Deletions, Changes: value.Changes, Patch: value.Patch})
	}
	return files, nil
}

// ParsePullRequests decodes one bounded provider page.
func ParsePullRequests(data []byte) ([]PullRequest, error) {
	var values []json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if len(values) > MaxPullRequests {
		return nil, errors.New("pull request page exceeds bound")
	}
	pulls := make([]PullRequest, 0, len(values))
	for _, value := range values {
		pull, err := ParsePullRequest(value)
		if err != nil {
			return nil, err
		}
		pulls = append(pulls, pull)
	}
	return pulls, nil
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
	for len(c.items) > maxProviderCacheEntries {
		oldestKey := ""
		var oldest time.Time
		for candidate, item := range c.items {
			if oldestKey == "" || item.At.Before(oldest) {
				oldestKey, oldest = candidate, item.At
			}
		}
		if oldestKey == "" {
			break
		}
		delete(c.items, oldestKey)
	}
	c.mu.Unlock()
	return value, nil
}
