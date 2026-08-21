package git

import (
	"context"
	"sync"

	"github.com/sphireinc/git-watch/internal/repo"
)

type SnapshotFunc func(context.Context, uint64) (repo.Snapshot, error)

type RefreshCoordinator struct {
	mu         sync.Mutex
	active     bool
	dirty      bool
	generation uint64
	refresh    SnapshotFunc
	results    chan RefreshResult
	closed     chan struct{}
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

type RefreshResult struct {
	Snapshot repo.Snapshot
	Err      error
}

func NewRefreshCoordinator(fn SnapshotFunc) *RefreshCoordinator {
	return &RefreshCoordinator{refresh: fn, results: make(chan RefreshResult, 2), closed: make(chan struct{})}
}
func (c *RefreshCoordinator) Results() <-chan RefreshResult { return c.results }
func (c *RefreshCoordinator) Done() <-chan struct{}         { return c.closed }

func (c *RefreshCoordinator) Request(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return
	default:
	}
	if c.active {
		c.dirty = true
		c.mu.Unlock()
		return
	}
	c.active = true
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.wg.Add(1)
	c.mu.Unlock()
	go func() {
		defer c.wg.Done()
		defer cancel()
		c.run(runCtx)
	}()
}

func (c *RefreshCoordinator) run(ctx context.Context) {
	for {
		c.mu.Lock()
		c.generation++
		generation := c.generation
		c.mu.Unlock()
		snapshot, err := c.refresh(ctx, generation)
		select {
		case c.results <- RefreshResult{Snapshot: snapshot, Err: err}:
		case <-ctx.Done():
		case <-c.closed:
		}
		c.mu.Lock()
		select {
		case <-c.closed:
			c.active = false
			c.dirty = false
			c.cancel = nil
			c.mu.Unlock()
			return
		default:
		}
		if !c.dirty {
			c.active = false
			c.cancel = nil
			c.mu.Unlock()
			return
		}
		c.dirty = false
		c.mu.Unlock()
	}
}

func (c *RefreshCoordinator) Close() {
	c.mu.Lock()
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.wg.Wait()
}
