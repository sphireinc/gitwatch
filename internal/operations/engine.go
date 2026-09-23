package operations

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrDuplicate = errors.New("operation already running")
var ErrNotRetryable = errors.New("operation is not marked replayable")

const retainedHistoryLimit = 32

type State uint8

const (
	Pending State = iota
	Running
	Succeeded
	Failed
	Cancelled
	TimedOut
)

// String returns the stable user-facing lifecycle label.
func (s State) String() string {
	switch s {
	case Pending:
		return "queued"
	case Running:
		return "running"
	case Succeeded:
		return "completed"
	case Failed:
		return "failed"
	case Cancelled:
		return "canceled"
	case TimedOut:
		return "timed out"
	default:
		return "unknown"
	}
}

type Result struct {
	ID, Repo, Name            string
	State                     State
	Err                       error
	Cause                     string
	Queued, Started, Finished time.Time
	Retryable                 bool
}
type Work func(context.Context) error
type Options struct{ Retryable bool }
type retrySpec struct {
	repo, name string
	timeout    time.Duration
	work       Work
	retryable  bool
}
type ResultMsg struct{ Result Result }
type Engine struct {
	mu      sync.Mutex
	limit   chan struct{}
	repos   map[string]*sync.Mutex
	active  map[string]context.CancelFunc
	waiters map[string]chan Result
	results chan Result
	latest  map[string]Result
	history []Result
	retry   map[string]retrySpec
}

func New(limit int) *Engine {
	if limit < 1 {
		limit = 1
	}
	return &Engine{limit: make(chan struct{}, limit), repos: make(map[string]*sync.Mutex), active: make(map[string]context.CancelFunc), waiters: make(map[string]chan Result), results: make(chan Result, limit), latest: make(map[string]Result), retry: make(map[string]retrySpec)}
}
func (e *Engine) Results() <-chan Result { return e.results }

// Snapshot returns copies of active and recently completed lifecycle records.
func (e *Engine) Snapshot() []Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := append([]Result(nil), e.history...)
	for _, item := range e.latest {
		if item.State == Pending || item.State == Running {
			result = append(result, item)
		}
	}
	return result
}
func (e *Engine) Submit(parent context.Context, id, repo, name string, timeout time.Duration, work Work) error {
	_, err := e.submit(parent, id, repo, name, timeout, work, Options{})
	return err
}

func (e *Engine) SubmitWithOptions(parent context.Context, id, repo, name string, timeout time.Duration, work Work, options Options) error {
	_, err := e.submit(parent, id, repo, name, timeout, work, options)
	return err
}

func (e *Engine) submit(parent context.Context, id, repo, name string, timeout time.Duration, work Work, options Options) (chan Result, error) {
	e.mu.Lock()
	if _, ok := e.active[id]; ok {
		e.mu.Unlock()
		return nil, ErrDuplicate
	}
	ctx, cancel := context.WithCancel(parent)
	e.active[id] = cancel
	waiter := make(chan Result, 1)
	e.waiters[id] = waiter
	e.retry[id] = retrySpec{repo: repo, name: name, timeout: timeout, work: work, retryable: options.Retryable}
	e.latest[id] = Result{ID: id, Repo: repo, Name: name, State: Pending, Queued: time.Now(), Retryable: options.Retryable}
	lock := e.repos[repo]
	if lock == nil {
		lock = &sync.Mutex{}
		e.repos[repo] = lock
	}
	e.mu.Unlock()
	go e.run(ctx, id, repo, name, timeout, lock, work, options.Retryable)
	return waiter, nil
}

// Cancel requests cancellation and reports whether the operation was active.
func (e *Engine) Cancel(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c := e.active[id]; c != nil {
		c()
		return true
	}
	return false
}

// Command adapts one operation result to a Bubble Tea-compatible command.
// The blocking wait happens in the command goroutine, never in Update or View.
func (e *Engine) Command(ctx context.Context, id, repo, name string, timeout time.Duration, work Work) func() ResultMsg {
	return e.CommandWithOptions(ctx, id, repo, name, timeout, work, Options{})
}

func (e *Engine) CommandWithOptions(ctx context.Context, id, repo, name string, timeout time.Duration, work Work, options Options) func() ResultMsg {
	return func() ResultMsg {
		waiter, err := e.submit(ctx, id, repo, name, timeout, work, options)
		if err != nil {
			return ResultMsg{Result: Result{ID: id, Repo: repo, Name: name, State: Failed, Err: err}}
		}
		result := <-waiter
		return ResultMsg{Result: result}
	}
}
func (e *Engine) Retry(ctx context.Context, id string) (chan Result, error) {
	e.mu.Lock()
	if _, active := e.active[id]; active {
		e.mu.Unlock()
		return nil, ErrDuplicate
	}
	spec, ok := e.retry[id]
	latest, hasLatest := e.latest[id]
	e.mu.Unlock()
	if !ok || !spec.retryable || !hasLatest || (latest.State != Failed && latest.State != Cancelled && latest.State != TimedOut) {
		return nil, ErrNotRetryable
	}
	return e.submit(ctx, id, spec.repo, spec.name, spec.timeout, spec.work, Options{Retryable: true})
}

func (e *Engine) RetryCommand(ctx context.Context, id string) func() ResultMsg {
	return func() ResultMsg {
		waiter, err := e.Retry(ctx, id)
		if err != nil {
			return ResultMsg{Result: Result{ID: id, State: Failed, Err: err}}
		}
		return ResultMsg{Result: <-waiter}
	}
}

func (e *Engine) run(ctx context.Context, id, repo, name string, timeout time.Duration, repoLock *sync.Mutex, work Work, retryable bool) {
	started := time.Now()
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	timerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result := Result{ID: id, Repo: repo, Name: name, State: Running, Queued: started, Started: started, Retryable: retryable}
	e.setLatest(result)
	select {
	case e.limit <- struct{}{}:
	case <-timerCtx.Done():
		result.State = Cancelled
		result.Err = timerCtx.Err()
		result.Cause = causeFor(result.Err)
		result.Finished = time.Now()
		e.finish(result)
		return
	}
	repoLock.Lock()
	err := work(timerCtx)
	repoLock.Unlock()
	<-e.limit
	result.Finished = time.Now()
	result.Err = err
	if errors.Is(timerCtx.Err(), context.DeadlineExceeded) {
		result.State = TimedOut
		result.Err = timerCtx.Err()
		result.Cause = "deadline exceeded"
	} else if errors.Is(timerCtx.Err(), context.Canceled) {
		result.State = Cancelled
		result.Err = timerCtx.Err()
		result.Cause = "canceled by user or shutdown"
	} else if err != nil {
		result.State = Failed
		result.Cause = "operation failed"
	} else {
		result.State = Succeeded
		result.Cause = "completed"
	}
	e.finish(result)
}
func (e *Engine) finish(r Result) {
	e.mu.Lock()
	delete(e.active, r.ID)
	waiter := e.waiters[r.ID]
	delete(e.waiters, r.ID)
	e.latest[r.ID] = r
	e.history = append(e.history, r)
	if len(e.history) > retainedHistoryLimit {
		e.history = e.history[len(e.history)-retainedHistoryLimit:]
	}
	e.pruneRetainedLocked()
	e.mu.Unlock()
	if waiter != nil {
		waiter <- r
	}
	// Result observers are best-effort; a command-specific waiter above is the
	// authoritative delivery path and must never be blocked by an idle observer.
	select {
	case e.results <- r:
	default:
	}
}

// pruneRetainedLocked keeps retry metadata and completed snapshots bounded by
// the same retention policy as the visible operation history. Active entries
// are retained even before they reach history so Snapshot remains live.
func (e *Engine) pruneRetainedLocked() {
	retained := make(map[string]struct{}, len(e.history)+len(e.active))
	for _, result := range e.history {
		retained[result.ID] = struct{}{}
	}
	for id := range e.active {
		retained[id] = struct{}{}
	}
	for id := range e.latest {
		if _, ok := retained[id]; !ok {
			delete(e.latest, id)
		}
	}
	for id := range e.retry {
		if _, ok := retained[id]; !ok {
			delete(e.retry, id)
		}
	}
}

func (e *Engine) setLatest(result Result) {
	e.mu.Lock()
	e.latest[result.ID] = result
	e.mu.Unlock()
}

func causeFor(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline exceeded"
	}
	return "canceled by user or shutdown"
}
