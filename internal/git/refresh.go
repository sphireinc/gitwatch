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
}

type RefreshResult struct {
	Snapshot repo.Snapshot
	Err      error
}

func NewRefreshCoordinator(fn SnapshotFunc) *RefreshCoordinator {
	return &RefreshCoordinator{refresh: fn, results: make(chan RefreshResult, 2), closed: make(chan struct{})}
}
func (c *RefreshCoordinator) Results() <-chan RefreshResult { return c.results }

func (c *RefreshCoordinator) Request(ctx context.Context) {
	c.mu.Lock()
	if c.active {
		c.dirty = true
		c.mu.Unlock()
		return
	}
	c.active = true
	c.mu.Unlock()
	go c.run(ctx)
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
		}
		c.mu.Lock()
		if !c.dirty {
			c.active = false
			c.mu.Unlock()
			return
		}
		c.dirty = false
		c.mu.Unlock()
	}
}

func (c *RefreshCoordinator) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}
