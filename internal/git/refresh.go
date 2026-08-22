package git

import (
	"context"
	"sync"

	"github.com/sphireinc/git-watch/internal/repo"
)

// SnapshotFunc refreshes repository state for a monotonically increasing generation.
type SnapshotFunc func(context.Context, uint64) (repo.Snapshot, error)

// RefreshCoordinator coalesces refresh requests and publishes completed snapshots.
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

// RefreshResult contains one snapshot refresh and its possible error.
type RefreshResult struct {
	Snapshot repo.Snapshot
	Err      error
}

// NewRefreshCoordinator creates a coordinator backed by fn.
func NewRefreshCoordinator(fn SnapshotFunc) *RefreshCoordinator {
	return &RefreshCoordinator{refresh: fn, results: make(chan RefreshResult, 2), closed: make(chan struct{})}
}

// Results returns the channel on which completed refreshes are published.
func (c *RefreshCoordinator) Results() <-chan RefreshResult { return c.results }

// Done returns a channel that closes when the coordinator shuts down.
func (c *RefreshCoordinator) Done() <-chan struct{} { return c.closed }

// Request schedules a refresh, coalescing requests that arrive while one runs.
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

// Close cancels the active refresh and waits for the coordinator to stop.
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
