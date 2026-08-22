package operations

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrDuplicate = errors.New("operation already running")

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
}
type Work func(context.Context) error
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
}

func New(limit int) *Engine {
	if limit < 1 {
		limit = 1
	}
	return &Engine{limit: make(chan struct{}, limit), repos: make(map[string]*sync.Mutex), active: make(map[string]context.CancelFunc), waiters: make(map[string]chan Result), results: make(chan Result, limit), latest: make(map[string]Result)}
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
	_, err := e.submit(parent, id, repo, name, timeout, work)
	return err
}

func (e *Engine) submit(parent context.Context, id, repo, name string, timeout time.Duration, work Work) (chan Result, error) {
	e.mu.Lock()
	if _, ok := e.active[id]; ok {
		e.mu.Unlock()
		return nil, ErrDuplicate
	}
	ctx, cancel := context.WithCancel(parent)
	e.active[id] = cancel
	waiter := make(chan Result, 1)
	e.waiters[id] = waiter
	e.latest[id] = Result{ID: id, Repo: repo, Name: name, State: Pending, Queued: time.Now()}
	lock := e.repos[repo]
	if lock == nil {
		lock = &sync.Mutex{}
		e.repos[repo] = lock
	}
	e.mu.Unlock()
	go e.run(ctx, id, repo, name, timeout, lock, work)
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
	return func() ResultMsg {
		waiter, err := e.submit(ctx, id, repo, name, timeout, work)
		if err != nil {
			return ResultMsg{Result: Result{ID: id, Repo: repo, Name: name, State: Failed, Err: err}}
		}
		result := <-waiter
		return ResultMsg{Result: result}
	}
}
func (e *Engine) run(ctx context.Context, id, repo, name string, timeout time.Duration, repoLock *sync.Mutex, work Work) {
	started := time.Now()
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	timerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result := Result{ID: id, Repo: repo, Name: name, State: Running, Queued: started, Started: started}
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
	if len(e.history) > 32 {
		e.history = e.history[len(e.history)-32:]
	}
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
