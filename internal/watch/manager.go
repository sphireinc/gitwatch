package watch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Manager owns one repository refresh-hint source. Filesystem mode includes
// low-frequency reconciliation and degrades to polling after runtime errors.
type Manager struct {
	root           string
	metadata       []string
	requested      RequestedMode
	pollInterval   time.Duration
	reconciliation time.Duration
	events         chan Event
	done           chan struct{}
	cancel         context.CancelFunc

	mu       sync.Mutex
	mode     Mode
	fs       *Watcher
	closed   bool
	closeErr error
}

// Start creates a repository watcher. In auto mode, filesystem setup failures
// return a polling manager plus a warning. Forced filesystem setup failures
// return an error and no manager so the caller can make degradation visible.
func Start(parent context.Context, root string, requested RequestedMode, pollInterval, reconciliation, debounce time.Duration) (*Manager, error) {
	return StartWithMetadata(parent, root, nil, requested, pollInterval, reconciliation, debounce)
}

// StartWithMetadata starts a manager that also observes linked Git metadata
// directories outside the worktree.
func StartWithMetadata(parent context.Context, root string, metadata []string, requested RequestedMode, pollInterval, reconciliation, debounce time.Duration) (*Manager, error) {
	if parent == nil {
		return nil, fmt.Errorf("watch parent context is nil")
	}
	if requested == "" {
		requested = RequestedAuto
	}
	if _, ok := ParseMode(string(requested)); !ok {
		return nil, fmt.Errorf("invalid watch mode %q", requested)
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	if reconciliation <= 0 {
		reconciliation = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(parent)
	manager := &Manager{
		root:           root,
		metadata:       append([]string(nil), metadata...),
		requested:      requested,
		pollInterval:   pollInterval,
		reconciliation: reconciliation,
		events:         make(chan Event, 8),
		done:           make(chan struct{}),
		cancel:         cancel,
	}

	if requested == RequestedPoll {
		manager.mode = ModePoll
		go manager.run(ctx)
		return manager, nil
	}
	filesystem, err := NewWithMetadata(root, metadata, debounce)
	if err != nil {
		if requested == RequestedFS {
			cancel()
			return nil, fmt.Errorf("start filesystem watcher: %w", err)
		}
		manager.mode = ModePoll
		go manager.run(ctx)
		return manager, fmt.Errorf("filesystem watcher unavailable; using polling: %w", err)
	}
	manager.mode = ModeFS
	manager.fs = filesystem
	go manager.run(ctx)
	return manager, nil
}

// Events returns refresh hints and watcher degradation diagnostics.
func (m *Manager) Events() <-chan Event { return m.events }

// Mode returns the currently active hint source.
func (m *Manager) Mode() Mode {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode
}

func (m *Manager) run(ctx context.Context) {
	defer close(m.done)
	defer close(m.events)
	defer func() {
		if err := m.closeFilesystem(); err != nil {
			m.recordCloseError(err)
		}
	}()
	if m.Mode() == ModeFS {
		if err := m.runFilesystem(ctx); err == nil || errors.Is(err, context.Canceled) {
			return
		} else {
			if closeErr := m.closeFilesystem(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close failed watcher: %w", closeErr))
			}
			m.setMode(ModePoll)
			if !m.emit(ctx, Event{At: time.Now(), Path: m.root, Mode: ModePoll, Err: fmt.Errorf("filesystem watcher failed; using polling: %w", err)}) {
				return
			}
		}
	}
	m.runPolling(ctx)
}

func (m *Manager) runFilesystem(ctx context.Context) error {
	m.mu.Lock()
	filesystem := m.fs
	m.mu.Unlock()
	if filesystem == nil {
		return fmt.Errorf("filesystem watcher is unavailable")
	}
	events := filesystem.Events(ctx)
	ticker := time.NewTicker(m.reconciliation)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !m.emit(ctx, Event{At: time.Now(), Path: m.root, Mode: ModeFS}) {
				return ctx.Err()
			}
		case event, ok := <-events:
			if !ok {
				return fmt.Errorf("filesystem event stream closed")
			}
			if event.Err != nil {
				return event.Err
			}
			if !m.emit(ctx, event) {
				return ctx.Err()
			}
		}
	}
}

func (m *Manager) runPolling(ctx context.Context) {
	for event := range NewPollerWithMetadata(m.root, m.metadata, m.pollInterval).Events(ctx) {
		if !m.emit(ctx, event) {
			return
		}
	}
}

func (m *Manager) emit(ctx context.Context, event Event) bool {
	select {
	case m.events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

func (m *Manager) setMode(mode Mode) {
	m.mu.Lock()
	m.mode = mode
	m.mu.Unlock()
}

func (m *Manager) closeFilesystem() error {
	m.mu.Lock()
	filesystem := m.fs
	m.fs = nil
	m.mu.Unlock()
	if filesystem == nil {
		return nil
	}
	return filesystem.Close()
}

func (m *Manager) recordCloseError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	m.closeErr = errors.Join(m.closeErr, err)
	m.mu.Unlock()
}

// Close cancels watcher and polling goroutines and waits for their exit.
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		<-m.done
		m.mu.Lock()
		err := m.closeErr
		m.mu.Unlock()
		return err
	}
	m.closed = true
	m.cancel()
	m.mu.Unlock()
	err := m.closeFilesystem()
	m.recordCloseError(err)
	<-m.done
	m.mu.Lock()
	err = m.closeErr
	m.mu.Unlock()
	return err
}
