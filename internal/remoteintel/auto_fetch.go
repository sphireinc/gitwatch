// Package remoteintel provides bounded, opt-in remote-awareness scheduling.
// It never performs a history-changing operation: callers supply a fetch-only
// callback and remain responsible for the typed Git boundary.
package remoteintel

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var ErrActiveOperation = errors.New("repository has an active history operation")

type Config struct {
	Enabled        bool
	Interval       time.Duration
	Jitter         time.Duration
	BackoffBase    time.Duration
	BackoffMax     time.Duration
	Workers        int
	GroupIntervals map[string]time.Duration
}

type Repository struct {
	Path            string
	Groups          []string
	ActiveOperation bool
}

type Result struct {
	Repository   string
	Status       string
	Attempt      int
	Err          error
	Started      time.Time
	Finished     time.Time
	Next         time.Time
	FailureClass string
}

type FetchFunc func(context.Context, string) error

type Scheduler struct {
	config Config
	mu     sync.Mutex
	next   map[string]time.Time
	fails  map[string]int
}

func New(config Config) *Scheduler {
	if config.Interval <= 0 {
		config.Interval = 30 * time.Minute
	}
	if config.BackoffBase <= 0 {
		config.BackoffBase = time.Minute
	}
	if config.BackoffMax <= 0 {
		config.BackoffMax = 30 * time.Minute
	}
	if config.BackoffMax < config.BackoffBase {
		config.BackoffMax = config.BackoffBase
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	return &Scheduler{config: config, next: make(map[string]time.Time), fails: make(map[string]int)}
}

func (s *Scheduler) Config() Config { return s.config }

// RunOnce schedules only repositories due at now. It uses a fixed worker pool,
// deterministic per-repository jitter, cancellation-aware waits, and bounded
// exponential backoff. Active history operations are never fetched.
func (s *Scheduler) RunOnce(ctx context.Context, repositories []Repository, now time.Time, fetch FetchFunc) []Result {
	if s == nil || !s.config.Enabled || fetch == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	type job struct {
		repository Repository
		attempt    int
	}
	jobs := make(chan job)
	results := make(chan Result, len(repositories))
	workers := s.config.Workers
	if workers > len(repositories) {
		workers = len(repositories)
	}
	if workers == 0 {
		return nil
	}
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for item := range jobs {
				results <- s.run(ctx, now, item.repository, item.attempt, fetch)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, repository := range repositories {
			if repository.Path == "" || repository.ActiveOperation {
				if repository.Path != "" {
					results <- Result{Repository: repository.Path, Status: "skipped-active"}
				}
				continue
			}
			s.mu.Lock()
			due := s.next[repository.Path]
			attempt := s.fails[repository.Path] + 1
			s.mu.Unlock()
			if !due.IsZero() && due.After(now) {
				continue
			}
			select {
			case jobs <- job{repository: repository, attempt: attempt}:
			case <-ctx.Done():
				return
			}
		}
	}()
	group.Wait()
	close(results)
	output := make([]Result, 0, len(repositories))
	for result := range results {
		output = append(output, result)
	}
	return output
}

func (s *Scheduler) run(ctx context.Context, now time.Time, repository Repository, attempt int, fetch FetchFunc) Result {
	started := time.Now()
	result := Result{Repository: repository.Path, Status: "running", Attempt: attempt, Started: started}
	if err := ctx.Err(); err != nil {
		result.Status, result.Err = "cancelled", err
		result.Finished = time.Now()
		return result
	}
	if delay := s.jitter(repository.Path); delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			result.Status, result.Err = "cancelled", ctx.Err()
			result.Finished = time.Now()
			return result
		}
	}
	result.Err = fetch(ctx, repository.Path)
	result.Finished = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if errors.Is(result.Err, ErrActiveOperation) {
		result.Status = "skipped-active"
		result.Err = nil
		result.Next = now.Add(s.intervalFor(repository))
		s.next[repository.Path] = result.Next
		return result
	}
	if result.Err != nil {
		result.Status = "failed"
		result.FailureClass = ClassifyError(result.Err)
		s.fails[repository.Path] = attempt
		backoff := s.config.BackoffBase
		for i := 1; i < attempt && backoff < s.config.BackoffMax; i++ {
			backoff *= 2
		}
		if backoff > s.config.BackoffMax {
			backoff = s.config.BackoffMax
		}
		result.Next = now.Add(backoff)
		s.next[repository.Path] = result.Next
		return result
	}
	result.Status = "fetched"
	delete(s.fails, repository.Path)
	result.Next = now.Add(s.intervalFor(repository))
	s.next[repository.Path] = result.Next
	return result
}

// ClassifyError provides stable dashboard language without treating provider
// or transport stderr as authoritative state.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "rate limit"), strings.Contains(lower, "too many requests"), strings.Contains(lower, "429"):
		return "rate-limited"
	case strings.Contains(lower, "auth"), strings.Contains(lower, "permission"), strings.Contains(lower, "denied"), strings.Contains(lower, "could not read username"), strings.Contains(lower, "401"), strings.Contains(lower, "403"):
		return "authentication"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "network"), strings.Contains(lower, "offline"), strings.Contains(lower, "connection"), strings.Contains(lower, "could not resolve"):
		return "offline"
	default:
		return "remote-error"
	}
}

func (s *Scheduler) intervalFor(repository Repository) time.Duration {
	interval := s.config.Interval
	for _, group := range repository.Groups {
		if candidate, ok := s.config.GroupIntervals[group]; ok && candidate > 0 && candidate < interval {
			interval = candidate
		}
	}
	return interval
}

func (s *Scheduler) jitter(repository string) time.Duration {
	if s.config.Jitter <= 0 {
		return 0
	}
	hash := sha256.Sum256([]byte(repository))
	var value uint64
	for _, byteValue := range hash[:8] {
		value = value<<8 | uint64(byteValue)
	}
	return time.Duration(value % uint64(s.config.Jitter))
}

func (r Result) Error() string {
	if r.Err == nil {
		return ""
	}
	return fmt.Sprintf("%s: %v", r.Status, r.Err)
}
