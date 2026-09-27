package registry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
	"github.com/sphireinc/git-watch/internal/operations"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
	"github.com/sphireinc/git-watch/internal/watch"
)

func TestEngineUsesBoundedWorkersAndCachesInactiveRepositories(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	var calls atomic.Int32
	engine := NewEngine(2)
	engine.InactiveAfter = time.Hour
	engine.Discover = func(context.Context, string) (git.Discovery, error) {
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		defer active.Add(-1)
		calls.Add(1)
		return git.Discovery{Root: "repo"}, nil
	}
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = func(context.Context, git.Discovery) (int, error) { return 3, nil }
	engine.Remotes = func(context.Context, git.Discovery) (int, error) { return 2, nil }
	engine.Worktrees = func(context.Context, git.Discovery) (int, error) { return 4, nil }
	entries := []Repository{{Path: "one"}, {Path: "two"}, {Path: "three"}}
	if got := engine.Refresh(context.Background(), entries, "one"); len(got) != 3 || peak.Load() > 2 || got[0].Stashes != 3 || got[0].Remotes != 2 || got[0].Worktrees != 4 || got[0].Health.Worktrees != 4 {
		t.Fatalf("unexpected refresh: len=%d peak=%d", len(got), peak.Load())
	}
	if calls.Load() != 3 {
		t.Fatalf("unexpected discovery count: %d", calls.Load())
	}
	entries[1].LastOpened = time.Now().Add(-2 * time.Hour)
	entries[2].LastOpened = time.Now().Add(-2 * time.Hour)
	engine.Refresh(context.Background(), entries, "one")
	if calls.Load() != 4 {
		t.Fatalf("inactive cache was not used: %d", calls.Load())
	}
}

func TestEngineBoundsStatusCacheAcrossRepositoryChurn(t *testing.T) {
	engine := NewEngine(4)
	engine.Stashes, engine.Remotes, engine.Worktrees = nil, nil, nil
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	for index := 0; index < maxCachedRepositories+44; index++ {
		path := fmt.Sprintf("repository-%03d", index)
		if got := engine.Refresh(context.Background(), []Repository{{Path: path}}, "active"); len(got) != 1 {
			t.Fatalf("refresh %q returned %d results", path, len(got))
		}
	}
	if got := len(engine.cache); got != maxCachedRepositories {
		t.Fatalf("status cache size = %d, want %d", got, maxCachedRepositories)
	}
}

func TestInspectGitignoreHealthReportsManagedAndMissingStates(t *testing.T) {
	root := t.TempDir()
	if health := inspectGitignore(root); health.Exists || health.Managed != 0 {
		t.Fatalf("missing health=%+v", health)
	}
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	template, _ := cat.Get("root/Go")
	content, err := managed.EncodeManagedBlock(template.ID, "github/gitignore", cat.Version(), template.ContentSHA256, template.Content, []byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), content, 0644); err != nil {
		t.Fatal(err)
	}
	health := inspectGitignore(root)
	if !health.Exists || health.Managed != 1 || health.Attention != 0 {
		t.Fatalf("managed health=%+v", health)
	}
}

func TestEngineUsesRepositoryRefreshPolicy(t *testing.T) {
	engine := NewEngine(1)
	engine.InactiveAfter = time.Hour
	engine.InactiveAfterFor = func(repository Repository) time.Duration {
		if len(repository.Groups) > 0 && repository.Groups[0] == "fast" {
			return time.Minute
		}
		return time.Hour
	}
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = nil
	engine.Remotes = nil
	entries := []Repository{{Path: "fast", Groups: []string{"fast"}, LastOpened: time.Now().Add(-2 * time.Hour)}, {Path: "slow", LastOpened: time.Now().Add(-30 * time.Minute)}}
	engine.Refresh(context.Background(), entries, "active")
	results := engine.Refresh(context.Background(), entries, "active")
	if !results[0].Skipped || results[1].Skipped {
		t.Fatalf("refresh policy results = %#v", results)
	}
}

func TestEngineBudgetCancelsSlowStatusSource(t *testing.T) {
	engine := NewEngine(1)
	engine.Budget = 10 * time.Millisecond
	engine.Stashes, engine.Remotes = nil, nil
	engine.Discover = func(ctx context.Context, _ string) (git.Discovery, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return git.Discovery{}, nil
		case <-ctx.Done():
			return git.Discovery{}, ctx.Err()
		}
	}
	result := engine.Refresh(context.Background(), []Repository{{Path: "slow"}}, "active")
	if len(result) != 1 || !errors.Is(result[0].Error, context.DeadlineExceeded) {
		t.Fatalf("slow source result = %#v", result)
	}
}

func TestEngineRecordsAuxiliaryWarningsAndRefreshMetadata(t *testing.T) {
	engine := NewEngine(1)
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{Root: "/repo"}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = func(context.Context, git.Discovery) (int, error) { return 0, errors.New("stash unavailable") }
	engine.Remotes = func(context.Context, git.Discovery) (int, error) { return 0, errors.New("remote unavailable") }
	engine.Worktrees = nil
	results := engine.Refresh(context.Background(), []Repository{{Path: "/repo", Name: "repo"}}, "/repo")
	if len(results) != 1 || len(results[0].Warnings) != 2 || results[0].Refreshed.IsZero() || results[0].Duration < 0 {
		t.Fatalf("refresh metadata = %#v", results)
	}
}

func TestEngineKeepsMixedAdvancedAttentionAcrossTwentyRepositories(t *testing.T) {
	engine := NewEngine(4)
	engine.Stashes, engine.Remotes, engine.Worktrees = nil, nil, nil
	root := t.TempDir()
	repositories := make([]Repository, 20)
	for i := range repositories {
		path := root + fmt.Sprintf("/repo-%02d", i)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		repositories[i] = Repository{Path: path, Name: fmt.Sprintf("repo-%02d", i)}
	}
	engine.Discover = func(_ context.Context, path string) (git.Discovery, error) {
		if path == repositories[7].Path {
			return git.Discovery{}, errors.New("repository disappeared")
		}
		return git.Discovery{Root: path}, nil
	}
	engine.Snapshot = func(_ context.Context, discovery git.Discovery, _ uint64) (repo.Snapshot, error) {
		snapshot := repo.Snapshot{Root: discovery.Root, Branch: repo.Branch{Name: "main"}}
		if discovery.Root == repositories[3].Path {
			operation, err := sequencer.NewState(sequencer.RepositoryID(discovery.Root), 0, sequencer.KindRebase, sequencer.PhaseActive)
			if err != nil {
				return repo.Snapshot{}, err
			}
			snapshot.Operation = &operation
		}
		if discovery.Root == repositories[5].Path {
			snapshot.Counts.Conflicted = 1
		}
		if discovery.Root == repositories[9].Path {
			snapshot.Counts.Untracked = 1
		}
		return snapshot, nil
	}
	results := engine.Refresh(context.Background(), repositories, repositories[0].Path)
	rows := Rows(results)
	if len(rows) != len(repositories) {
		t.Fatalf("mixed attention row count = %d", len(rows))
	}
	byPath := make(map[string]Row, len(rows))
	for _, row := range rows {
		byPath[row.Repository.Path] = row
	}
	if byPath[repositories[3].Path].Attention != "rebase" || byPath[repositories[5].Path].Attention != "conflict" || byPath[repositories[9].Path].Attention != "dirty/diverged" {
		t.Fatalf("mixed attention rows = %#v", rows)
	}
	if byPath[repositories[7].Path].State != "error" || byPath[repositories[0].Path].State == "error" {
		t.Fatalf("broken repository isolation = %#v", rows)
	}
}

func TestEngineRefreshKeepsHundredRepositoriesWithinWorkerBound(t *testing.T) {
	const (
		repositoryCount = 100
		workers         = 8
	)
	engine := NewEngine(workers)
	engine.Stashes, engine.Remotes, engine.Worktrees = nil, nil, nil
	var active atomic.Int32
	var peak atomic.Int32
	engine.Discover = func(ctx context.Context, path string) (git.Discovery, error) {
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		defer active.Add(-1)
		select {
		case <-time.After(time.Millisecond):
			return git.Discovery{Root: path}, nil
		case <-ctx.Done():
			return git.Discovery{}, ctx.Err()
		}
	}
	engine.Snapshot = func(_ context.Context, discovery git.Discovery, _ uint64) (repo.Snapshot, error) {
		return repo.Snapshot{Root: discovery.Root, Branch: repo.Branch{Name: "main"}}, nil
	}
	repositories := make([]Repository, repositoryCount)
	for i := range repositories {
		repositories[i] = Repository{Path: fmt.Sprintf("/repo-%03d", i)}
	}
	results := engine.Refresh(context.Background(), repositories, repositories[0].Path)
	if len(results) != repositoryCount {
		t.Fatalf("refresh result count = %d, want %d", len(results), repositoryCount)
	}
	if got := peak.Load(); got > workers {
		t.Fatalf("peak concurrent refreshes = %d, want <= %d", got, workers)
	}
	seen := make(map[string]struct{}, len(results))
	for index, result := range results {
		if result.Error != nil || result.Snapshot.Root == "" {
			t.Fatalf("result[%d] = %#v", index, result)
		}
		if result.Snapshot.Root != repositories[index].Path {
			t.Fatalf("result[%d] root = %q, want %q", index, result.Snapshot.Root, repositories[index].Path)
		}
		seen[result.Snapshot.Root] = struct{}{}
	}
	if len(seen) != repositoryCount {
		t.Fatalf("distinct refreshed repositories = %d, want %d", len(seen), repositoryCount)
	}
}

func TestRegistryAndOperationEnginesBoundDirectChildProcesses(t *testing.T) {
	const (
		repositoryCount   = 100
		registryWorkers   = 8
		operationWorkers  = 4
		operationCount    = 16
		processCounterEnv = "GITWATCH_REGISTRY_PROCESS_COUNTER"
		processTokenEnv   = "GITWATCH_REGISTRY_PROCESS_TOKENS"
	)
	counterDir := t.TempDir()
	runner := git.Runner{
		Binary: os.Args[0],
		Dir:    counterDir,
		Env: []string{
			processCounterEnv + "=" + counterDir,
			processTokenEnv + "=" + counterDir,
		},
	}
	registryEngine := NewEngine(registryWorkers)
	registryEngine.Stashes, registryEngine.Remotes, registryEngine.Worktrees = nil, nil, nil
	registryEngine.Discover = func(_ context.Context, path string) (git.Discovery, error) {
		return git.Discovery{Root: path}, nil
	}
	registryEngine.Snapshot = func(ctx context.Context, discovery git.Discovery, _ uint64) (repo.Snapshot, error) {
		if err := runRegistryDirectChildProbe(ctx, runner); err != nil {
			return repo.Snapshot{}, err
		}
		return repo.Snapshot{Root: discovery.Root}, nil
	}
	repositories := make([]Repository, repositoryCount)
	for index := range repositories {
		repositories[index] = Repository{Path: fmt.Sprintf("/process-probe-%03d", index)}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stopSampling, sampledPeak := startProcessMarkerSampler(t, counterDir)
	defer stopSampling()
	registryResults := make(chan []StatusResult, 1)
	go func() {
		registryResults <- registryEngine.Refresh(ctx, repositories, repositories[0].Path)
	}()

	operationEngine := operations.New(operationWorkers)
	operationResults := make(chan operations.ResultMsg, operationCount)
	for index := 0; index < operationCount; index++ {
		command := operationEngine.Command(ctx, fmt.Sprintf("process-probe-%02d", index), fmt.Sprintf("repo-%02d", index), "child-process probe", 10*time.Second, func(workCtx context.Context) error {
			return runRegistryDirectChildProbe(workCtx, runner)
		})
		go func() { operationResults <- command() }()
	}
	for index := 0; index < operationCount; index++ {
		select {
		case result := <-operationResults:
			if result.Result.State != operations.Succeeded {
				t.Errorf("operation %q result = %+v", result.Result.ID, result.Result)
			}
		case <-ctx.Done():
			t.Fatalf("waiting for operation child processes: %v", ctx.Err())
		}
	}
	var results []StatusResult
	select {
	case results = <-registryResults:
	case <-ctx.Done():
		t.Fatalf("waiting for registry child processes: %v", ctx.Err())
	}
	if len(results) != repositoryCount {
		t.Fatalf("refresh result count = %d, want %d", len(results), repositoryCount)
	}
	for index, result := range results {
		if result.Error != nil {
			t.Fatalf("refresh result[%d] error = %v", index, result.Error)
		}
	}
	stopSampling()
	active := countActiveProcessMarkers(t, counterDir)
	if active != 0 {
		t.Fatalf("active direct child processes after refresh = %d, want 0", active)
	}
	peak := sampledPeak()
	processBound := registryWorkers + operationWorkers
	if peak < 2 || peak > processBound {
		t.Fatalf("peak direct child processes = %d, want 2..%d", peak, processBound)
	}
	t.Logf("observed peak of %d direct child processes for %d repositories and %d concurrent operations (combined worker cap %d)", peak, repositoryCount, operationCount, processBound)
}

func runRegistryDirectChildProbe(ctx context.Context, runner git.Runner) error {
	result, err := runner.Run(ctx, "-test.run=^TestRegistryDirectChildProcessProbe$")
	if err == nil {
		return nil
	}
	var commandErr *git.CommandError
	if errors.As(err, &commandErr) {
		return fmt.Errorf("child process exit=%d stdout=%q stderr=%q cause=%v: %w", commandErr.Result.ExitCode, commandErr.Result.Stdout, commandErr.Result.Stderr, commandErr.Cause, err)
	}
	return fmt.Errorf("child process stdout=%q stderr=%q: %w", result.Stdout, result.Stderr, err)
}

func TestRefreshCoordinatorBoundsChildrenDuringWatcherEventStorm(t *testing.T) {
	const (
		processCounterEnv = "GITWATCH_REGISTRY_PROCESS_COUNTER"
		processDelayEnv   = "GITWATCH_REGISTRY_PROCESS_DELAY"
		processDelay      = time.Second
		fileCount         = 256
	)
	root := t.TempDir()
	watcher, err := watch.New(root, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	watchCtx, stopWatcher := context.WithCancel(context.Background())
	t.Cleanup(stopWatcher)
	events := watcher.Events(watchCtx)

	counterDir := t.TempDir()
	runner := git.Runner{
		Binary: os.Args[0],
		Dir:    counterDir,
		Env: []string{
			processCounterEnv + "=" + counterDir,
			processDelayEnv + "=" + processDelay.String(),
		},
	}
	started := make(chan struct{}, 2)
	coordinator := git.NewRefreshCoordinator(func(ctx context.Context, generation uint64) (repo.Snapshot, error) {
		started <- struct{}{}
		if _, err := runner.Run(ctx, "-test.run=^TestRegistryDirectChildProcessProbe$"); err != nil {
			return repo.Snapshot{}, err
		}
		return repo.Snapshot{Root: root, Generation: generation}, nil
	})
	t.Cleanup(coordinator.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	coordinator.Request(ctx)
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatalf("initial refresh did not start: %v", ctx.Err())
	}
	processDeadline := time.Now().Add(2 * time.Second)
	for {
		active, _ := readProcessCounter(t, counterDir)
		if active == 1 {
			break
		}
		if time.Now().After(processDeadline) {
			t.Fatalf("initial child process did not become active; active=%d", active)
		}
		time.Sleep(time.Millisecond)
	}

	for index := 0; index < fileCount; index++ {
		path := filepath.Join(root, fmt.Sprintf("storm-%03d", index))
		if err := os.WriteFile(path, []byte("refresh storm"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	eventCount := 0
	eventDuringRefresh := false
	quiet := time.NewTimer(100 * time.Millisecond)
	defer quiet.Stop()
	stormDeadline := time.NewTimer(3 * time.Second)
	defer stormDeadline.Stop()
eventsLoop:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatal("watcher events closed during event storm")
			}
			if event.Err != nil || event.Mode != watch.ModeFS {
				t.Fatalf("event storm produced invalid event: %#v", event)
			}
			eventCount++
			active, _ := readProcessCounter(t, counterDir)
			if active > 0 {
				eventDuringRefresh = true
			}
			coordinator.Request(ctx)
			if !quiet.Stop() {
				select {
				case <-quiet.C:
				default:
				}
			}
			quiet.Reset(100 * time.Millisecond)
		case <-quiet.C:
			if eventCount == 0 {
				t.Fatal("event storm produced no watcher events")
			}
			if !eventDuringRefresh {
				t.Fatal("watcher event storm did not overlap the active child process")
			}
			stopWatcher()
			for range events {
				// Drain any final buffered event and wait for the watcher goroutine
				// to observe cancellation and close its output channel.
			}
			break eventsLoop
		case <-stormDeadline.C:
			t.Fatalf("watcher event storm did not settle: %d events", eventCount)
		case <-ctx.Done():
			t.Fatalf("watcher event storm exceeded context: %v", ctx.Err())
		}
	}

	for index := 0; index < 2; index++ {
		select {
		case result := <-coordinator.Results():
			if result.Err != nil {
				t.Fatalf("refresh %d failed: %v", index+1, result.Err)
			}
			if result.Snapshot.Root != root || result.Snapshot.Generation != uint64(index+1) {
				t.Fatalf("refresh %d snapshot = %+v", index+1, result.Snapshot)
			}
		case <-ctx.Done():
			t.Fatalf("waiting for coalesced refresh %d: %v", index+1, ctx.Err())
		}
	}
	active, peak := readProcessCounter(t, counterDir)
	if active != 0 || peak != 1 {
		t.Fatalf("event-storm child process counts: active=%d peak=%d, want active=0 peak=1", active, peak)
	}
	t.Logf("coalesced %d filesystem events into two refreshes with peak child processes %d", eventCount, peak)
}

func TestRegistryDirectChildProcessProbe(t *testing.T) {
	counterDir := os.Getenv("GITWATCH_REGISTRY_PROCESS_COUNTER")
	if counterDir == "" {
		t.Skip("child-process probe is run only by its parent regression test")
	}
	delay := 40 * time.Millisecond
	if configured := os.Getenv("GITWATCH_REGISTRY_PROCESS_DELAY"); configured != "" {
		parsed, err := time.ParseDuration(configured)
		if err != nil {
			t.Fatalf("parse child-process delay %q: %v", configured, err)
		}
		delay = parsed
	}
	if tokenDir := os.Getenv("GITWATCH_REGISTRY_PROCESS_TOKENS"); tokenDir != "" {
		marker, err := os.CreateTemp(tokenDir, "child-process-*")
		if err != nil {
			t.Fatalf("create child-process marker: %v", err)
		}
		markerPath := marker.Name()
		if err := marker.Close(); err != nil {
			t.Fatalf("close child-process marker: %v", err)
		}
		defer func() {
			if err := os.Remove(markerPath); err != nil {
				t.Errorf("remove child-process marker: %v", err)
			}
		}()
		time.Sleep(delay)
		return
	}
	updateProcessCounter(t, counterDir, 1)
	time.Sleep(delay)
	updateProcessCounter(t, counterDir, -1)
}

func startProcessMarkerSampler(t *testing.T, directory string) (stop func(), peak func() int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var maxActive atomic.Int32
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		sample := func() {
			entries, err := os.ReadDir(directory)
			if err != nil {
				t.Errorf("sample child-process markers: %v", err)
				return
			}
			active := int32(0)
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "child-process-") {
					active++
				}
			}
			for active > maxActive.Load() {
				previous := maxActive.Load()
				if active <= previous || maxActive.CompareAndSwap(previous, active) {
					break
				}
			}
		}
		for {
			sample()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	stop = func() {
		cancel()
		<-done
	}
	peak = func() int { return int(maxActive.Load()) }
	return stop, peak
}

func countActiveProcessMarkers(t *testing.T, directory string) int {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read child-process markers: %v", err)
	}
	active := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "child-process-") {
			active++
		}
	}
	return active
}

func updateProcessCounter(t *testing.T, directory string, delta int) {
	t.Helper()
	lockPath := filepath.Join(directory, "lock")
	deadline := time.Now().Add(5 * time.Second)
	var lockFile *os.File
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			lockFile = file
			break
		}
		if !errors.Is(err, os.ErrExist) {
			t.Fatalf("acquire process-counter lock: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for process-counter lock")
		}
		time.Sleep(time.Millisecond)
	}
	defer func() {
		if err := lockFile.Close(); err != nil {
			t.Errorf("close process-counter lock: %v", err)
		}
		if err := os.Remove(lockPath); err != nil {
			t.Errorf("release process-counter lock: %v", err)
		}
	}()

	active, _ := readProcessCounter(t, directory)
	active += delta
	if active < 0 {
		t.Fatalf("active direct child process count became negative: %d", active)
	}
	_, peak := readProcessCounter(t, directory)
	if active > peak {
		peak = active
	}
	if err := os.WriteFile(filepath.Join(directory, "counter"), []byte(strconv.Itoa(active)), 0o600); err != nil {
		t.Fatalf("write active child-process count: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "peak"), []byte(strconv.Itoa(peak)), 0o600); err != nil {
		t.Fatalf("write peak child-process count: %v", err)
	}
}

func readProcessCounter(t *testing.T, directory string) (active, peak int) {
	t.Helper()
	read := func(name string) int {
		content, err := os.ReadFile(filepath.Join(directory, name))
		if errors.Is(err, os.ErrNotExist) {
			return 0
		}
		if err != nil {
			t.Fatalf("read %s child-process count: %v", name, err)
		}
		value, err := strconv.Atoi(string(content))
		if err != nil {
			t.Fatalf("parse %s child-process count %q: %v", name, content, err)
		}
		return value
	}
	return read("counter"), read("peak")
}

func TestEngineCancelledRefreshesSettleWithoutGoroutineGrowth(t *testing.T) {
	engine := NewEngine(4)
	engine.Stashes, engine.Remotes, engine.Worktrees = nil, nil, nil
	engine.Discover = func(ctx context.Context, path string) (git.Discovery, error) {
		select {
		case <-ctx.Done():
			return git.Discovery{}, ctx.Err()
		case <-time.After(time.Second):
			return git.Discovery{Root: path}, nil
		}
	}
	repositories := make([]Repository, 64)
	for index := range repositories {
		repositories[index] = Repository{Path: fmt.Sprintf("/cancelled-%03d", index)}
	}
	runtime.GC()
	baseline := runtime.NumGoroutine()
	for iteration := 0; iteration < 20; iteration++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		results := engine.Refresh(ctx, repositories, "")
		if len(results) > len(repositories) {
			t.Fatalf("cancelled refresh returned %d results for %d repositories", len(results), len(repositories))
		}
		for _, result := range results {
			if result.Repository.Path == "" {
				t.Fatal("cancelled refresh returned an unscoped result")
			}
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > baseline+2 && time.Now().Before(deadline) {
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > baseline+2 {
		t.Fatalf("cancelled refresh goroutines grew from %d to %d", baseline, got)
	}
}
