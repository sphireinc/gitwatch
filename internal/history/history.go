package history

import (
	"fmt"
	"net/url"
	"strings"
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
	At        time.Time
	Kind      Kind
	Path      string
	Message   string
	Duration  time.Duration
	Operation *OperationRecord
}

// OperationRecord describes semantic Git state associated with an activity
// event. It is intentionally bounded and contains no environment or secrets.
type OperationRecord struct {
	Repository      string
	Kind            string
	Args            []string
	Target          string
	OldHead         string
	NewHead         string
	Refs            []string
	Duration        time.Duration
	RecoverySHA     string
	RecoveryRef     string
	RecoverySubject string
	Outcome         string
}

// RedactArgs returns a safe copy suitable for journal storage or rendering.
func RedactArgs(args []string) []string {
	redacted := make([]string, len(args))
	secretNext := false
	for i, arg := range args {
		if secretNext {
			redacted[i] = "<redacted>"
			secretNext = false
			continue
		}
		lower := strings.ToLower(arg)
		if lower == "--password" || lower == "--token" || lower == "--auth-token" || lower == "--header" {
			redacted[i] = arg
			secretNext = true
			continue
		}
		inlineSecret := false
		for _, flag := range []string{"--password=", "--token=", "--auth-token=", "--header="} {
			if strings.HasPrefix(lower, flag) {
				redacted[i] = arg[:len(flag)] + "<redacted>"
				inlineSecret = true
				break
			}
		}
		if inlineSecret {
			continue
		}
		if parsed, err := url.Parse(arg); err == nil && parsed.User != nil {
			parsed.User = nil
			redacted[i] = parsed.String()
			continue
		}
		redacted[i] = arg
	}
	return redacted
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
	l.events = append(l.events, cloneEvent(event))
	if len(l.events) > l.max {
		l.events = l.events[len(l.events)-l.max:]
	}
}
func (l *Log) All() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	events := make([]Event, len(l.events))
	for i, event := range l.events {
		events[i] = cloneEvent(event)
	}
	return events
}

func cloneEvent(event Event) Event {
	if event.Operation != nil {
		operation := *event.Operation
		operation.Args = append([]string(nil), event.Operation.Args...)
		operation.Refs = append([]string(nil), event.Operation.Refs...)
		event.Operation = &operation
	}
	return event
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
