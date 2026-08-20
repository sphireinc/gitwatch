package git

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jusanchez/gitwatch/internal/repo"
)

func TestRefreshCoordinatorCoalesces(t *testing.T) {
	var mu sync.Mutex
	active, maxActive, calls := 0, 0, 0
	started := make(chan struct{}, 4)
	fn := func(context.Context, uint64) (repo.Snapshot, error) {
		mu.Lock()
		active++
		calls++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		started <- struct{}{}
		time.Sleep(15 * time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		return repo.Snapshot{}, nil
	}
	c := NewRefreshCoordinator(fn)
	ctx := context.Background()
	c.Request(ctx)
	<-started
	for i := 0; i < 20; i++ {
		c.Request(ctx)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-c.Results():
		case <-time.After(time.Second):
			t.Fatal("refresh did not complete")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if maxActive != 1 || calls != 2 {
		t.Fatalf("active=%d calls=%d", maxActive, calls)
	}
}
