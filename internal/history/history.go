package history

import (
	"fmt"
	"sync"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

type Kind string

const (
	FileModified     Kind = "file modified"
	FileStaged       Kind = "file staged"
	FileUnstaged     Kind = "file unstaged"
	FileRemoved      Kind = "file removed"
	BranchChanged    Kind = "branch changed"
	WatchFallback    Kind = "watcher fallback"
	RefreshError     Kind = "refresh error"
	OperationSuccess Kind = "operation success"
	OperationFailure Kind = "operation failure"
	ChangesCoalesced Kind = "changes coalesced"
)

const maxDiffEvents = 100

type Event struct {
	At       time.Time
	Kind     Kind
	Path     string
	Message  string
	Duration time.Duration
}
type Log struct {
	mu     sync.RWMutex
	max    int
	events []Event
}

func New(max int) *Log {
	if max <= 0 {
		max = 100
	}
	return &Log{max: max}
}
func (l *Log) Add(event Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
	if len(l.events) > l.max {
		l.events = l.events[len(l.events)-l.max:]
	}
}
func (l *Log) All() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]Event(nil), l.events...)
}
func Diff(oldSnapshot, newSnapshot repo.Snapshot) []Event {
	old := make(map[string]repo.Entry, len(oldSnapshot.Entries))
	for _, e := range oldSnapshot.Entries {
		old[string(e.Path)] = e
	}
	newPaths := make(map[string]struct{}, len(newSnapshot.Entries))
	out := make([]Event, 0, min(maxDiffEvents, len(oldSnapshot.Entries)+len(newSnapshot.Entries)+1))
	omitted := 0
	add := func(event Event) {
		if len(out) < maxDiffEvents-1 {
			out = append(out, event)
			return
		}
		omitted++
	}
	for _, e := range newSnapshot.Entries {
		p := string(e.Path)
		newPaths[p] = struct{}{}
		previous, ok := old[p]
		if !ok {
			add(Event{At: newSnapshot.ObservedAt, Kind: FileModified, Path: p})
			continue
		}
		if e.Staged && !previous.Staged {
			add(Event{At: newSnapshot.ObservedAt, Kind: FileStaged, Path: p})
		}
		if !e.Staged && previous.Staged {
			add(Event{At: newSnapshot.ObservedAt, Kind: FileUnstaged, Path: p})
		}
	}
	for p := range old {
		if _, found := newPaths[p]; !found {
			add(Event{At: newSnapshot.ObservedAt, Kind: FileRemoved, Path: p})
		}
	}
	if oldSnapshot.Branch.Name != newSnapshot.Branch.Name {
		add(Event{At: newSnapshot.ObservedAt, Kind: BranchChanged, Message: newSnapshot.Branch.Name})
	}
	if omitted > 0 {
		out = append(out, Event{At: newSnapshot.ObservedAt, Kind: ChangesCoalesced, Message: fmt.Sprintf("%d additional changes", omitted)})
	}
	return out
}
