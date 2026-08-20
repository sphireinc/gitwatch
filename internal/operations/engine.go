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

type Result struct {
	ID, Repo, Name    string
	State             State
	Err               error
	Started, Finished time.Time
}
type Work func(context.Context) error
type Engine struct {
	mu      sync.Mutex
	limit   chan struct{}
	repos   map[string]*sync.Mutex
	active  map[string]context.CancelFunc
	results chan Result
}

func New(limit int) *Engine {
	if limit < 1 {
		limit = 1
	}
	return &Engine{limit: make(chan struct{}, limit), repos: make(map[string]*sync.Mutex), active: make(map[string]context.CancelFunc), results: make(chan Result, limit)}
}
func (e *Engine) Results() <-chan Result { return e.results }
func (e *Engine) Submit(parent context.Context, id, repo, name string, timeout time.Duration, work Work) error {
	e.mu.Lock()
	if _, ok := e.active[id]; ok {
		e.mu.Unlock()
		return ErrDuplicate
	}
	ctx, cancel := context.WithCancel(parent)
	e.active[id] = cancel
	lock := e.repos[repo]
	if lock == nil {
		lock = &sync.Mutex{}
		e.repos[repo] = lock
	}
	e.mu.Unlock()
	go e.run(ctx, id, repo, name, timeout, lock, work)
	return nil
}
func (e *Engine) Cancel(id string) {
	e.mu.Lock()
	if c := e.active[id]; c != nil {
		c()
	}
	e.mu.Unlock()
}
func (e *Engine) run(ctx context.Context, id, repo, name string, timeout time.Duration, repoLock *sync.Mutex, work Work) {
	started := time.Now()
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	timerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result := Result{ID: id, Repo: repo, Name: name, State: Running, Started: started}
	select {
	case e.limit <- struct{}{}:
	case <-timerCtx.Done():
		result.State = Cancelled
		result.Err = timerCtx.Err()
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
	} else if errors.Is(timerCtx.Err(), context.Canceled) {
		result.State = Cancelled
		result.Err = timerCtx.Err()
	} else if err != nil {
		result.State = Failed
	} else {
		result.State = Succeeded
	}
	e.finish(result)
}
func (e *Engine) finish(r Result) { e.mu.Lock(); delete(e.active, r.ID); e.mu.Unlock(); e.results <- r }
