package history

import (
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
)

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
	old := map[string]repo.Entry{}
	for _, e := range oldSnapshot.Entries {
		old[string(e.Path)] = e
	}
	out := []Event{}
	for _, e := range newSnapshot.Entries {
		p := string(e.Path)
		previous, ok := old[p]
		if !ok {
			out = append(out, Event{At: newSnapshot.ObservedAt, Kind: FileModified, Path: p})
			continue
		}
		if e.Staged && !previous.Staged {
			out = append(out, Event{At: newSnapshot.ObservedAt, Kind: FileStaged, Path: p})
		}
		if !e.Staged && previous.Staged {
			out = append(out, Event{At: newSnapshot.ObservedAt, Kind: FileUnstaged, Path: p})
		}
	}
	for p := range old {
		found := false
		for _, e := range newSnapshot.Entries {
			if string(e.Path) == p {
				found = true
				break
			}
		}
		if !found {
			out = append(out, Event{At: newSnapshot.ObservedAt, Kind: FileRemoved, Path: p})
		}
	}
	if oldSnapshot.Branch.Name != newSnapshot.Branch.Name {
		out = append(out, Event{At: newSnapshot.ObservedAt, Kind: BranchChanged, Message: newSnapshot.Branch.Name})
	}
	return out
}
