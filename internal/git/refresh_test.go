package git

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
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
	for i := 0; i < 10000; i++ {
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

func TestRefreshCoordinatorCloseStopsNewWork(t *testing.T) {
	called := make(chan struct{}, 1)
	c := NewRefreshCoordinator(func(context.Context, uint64) (repo.Snapshot, error) {
		called <- struct{}{}
		return repo.Snapshot{}, nil
	})
	c.Close()
	c.Request(context.Background())
	select {
	case <-called:
		t.Fatal("closed coordinator started refresh work")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-c.Done():
	default:
		t.Fatal("coordinator did not close its done channel")
	}
}

func TestRefreshCoordinatorCloseCancelsAndWaitsForActiveWork(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	c := NewRefreshCoordinator(func(ctx context.Context, _ uint64) (repo.Snapshot, error) {
		close(started)
		<-ctx.Done()
		close(finished)
		return repo.Snapshot{}, ctx.Err()
	})
	c.Request(context.Background())
	<-started
	c.Close()
	select {
	case <-finished:
	default:
		t.Fatal("close returned before active refresh exited")
	}
}
