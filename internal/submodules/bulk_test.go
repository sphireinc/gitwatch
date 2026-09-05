package submodules

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/git"
)

type boundedBulkRunner struct {
	mu        sync.Mutex
	active    int
	maxActive int
	failPath  string
}

func (r *boundedBulkRunner) Run(ctx context.Context, args ...string) (git.Result, error) {
	r.mu.Lock()
	r.active++
	if r.active > r.maxActive {
		r.maxActive = r.active
	}
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.active--
		r.mu.Unlock()
	}()
	select {
	case <-time.After(5 * time.Millisecond):
	case <-ctx.Done():
		return git.Result{Args: append([]string(nil), args...)}, ctx.Err()
	}
	if len(args) > 0 && args[len(args)-1] == r.failPath {
		return git.Result{Args: append([]string(nil), args...)}, errors.New("injected module failure")
	}
	return git.Result{Args: append([]string(nil), args...)}, nil
}

func (r *boundedBulkRunner) maxConcurrent() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxActive
}

func TestBulkDeduplicatesAndSkipsBeyondHardModuleLimit(t *testing.T) {
	repository := t.TempDir()
	outcome := Bulk(context.Background(), git.NewRunner(repository), BulkRequest{
		Repository: repository,
		Paths:      []string{"one", "one", "two"},
		Action:     BulkUpdate,
		Workers:    99,
		MaxModules: 1,
	})
	if len(outcome.Items) != 2 || outcome.Items[0].Path != "one" || outcome.Items[1].State != ItemSkipped || outcome.Items[1].Outcome.Err != ErrBulkLimit {
		t.Fatalf("bulk bound = %+v", outcome.Items)
	}
	if outcome.Items[0].State != ItemFailed {
		t.Fatalf("invalid repository result = %+v", outcome.Items[0])
	}
}

func TestBulkCancellationMarksQueuedItemsWithoutStartingGit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome := Bulk(ctx, git.Runner{Binary: "missing-git-for-cancel-test"}, BulkRequest{
		Repository: t.TempDir(), Paths: []string{"one", "two"}, Action: BulkSync, Workers: 2,
	})
	if !outcome.Cancelled || len(outcome.Items) != 2 {
		t.Fatalf("cancelled bulk = %+v", outcome)
	}
	for _, item := range outcome.Items {
		if item.State != ItemCancelled {
			t.Fatalf("cancelled item = %+v", item)
		}
	}
}

func TestBulkFailureIsolatedFromSuccessfulSelectedModule(t *testing.T) {
	ctx := context.Background()
	child := t.TempDir()
	childRunner := git.NewRunner(child)
	gitMustRun(t, ctx, childRunner, "init", "-b", "main", "--", child)
	if err := os.WriteFile(filepath.Join(child, "README"), []byte("child\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRun(t, ctx, childRunner, "add", "--", "README")
	gitMustRun(t, ctx, childRunner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "child")
	parent := t.TempDir()
	runner := git.NewRunner(parent)
	runner.Env = []string{"GIT_ALLOW_PROTOCOL=file"}
	gitMustRun(t, ctx, runner, "init", "-b", "main", "--", parent)
	added := Add(ctx, runner, AddRequest{Repository: parent, Path: "nested", URL: "file://" + child})
	if added.Err != nil {
		t.Fatalf("add = %+v", added)
	}
	gitMustRun(t, ctx, runner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "add-submodule")
	outcome := Bulk(ctx, runner, BulkRequest{Repository: parent, Paths: []string{"missing", "nested"}, Action: BulkInitialize, Workers: 2})
	if len(outcome.Items) != 2 || outcome.Items[0].State != ItemFailed || outcome.Items[1].State != ItemSucceeded {
		t.Fatalf("failure isolation = %+v", outcome.Items)
	}
}

func TestBulkTwentyFourModulesFailureIsolationAndWorkerBound(t *testing.T) {
	paths := make([]string, 24)
	for i := range paths {
		paths[i] = filepath.Join("module", string(rune('a'+i)))
	}
	runner := &boundedBulkRunner{failPath: paths[7]}
	outcome := Bulk(context.Background(), runner, BulkRequest{
		Repository: t.TempDir(), Paths: paths, Action: BulkUpdate, Workers: 99, MaxModules: 24,
	})
	if runner.maxConcurrent() > maxBulkWorkers {
		t.Fatalf("worker bound exceeded: max active=%d limit=%d", runner.maxConcurrent(), maxBulkWorkers)
	}
	if len(outcome.Items) != len(paths) || outcome.Items[7].State != ItemFailed {
		t.Fatalf("injected failure isolation = items=%d failed=%+v", len(outcome.Items), outcome.Items[7])
	}
	for i, item := range outcome.Items {
		if i != 7 && item.State != ItemSucceeded {
			t.Fatalf("module %d did not continue after isolated failure: %+v", i, item)
		}
	}
}
