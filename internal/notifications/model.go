// Package notifications stores bounded, session-local user notifications.
package notifications

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

// Level controls the visual severity of a notification.
type Level string

const (
	// Info is an informational notification.
	Info    Level = "info"
	Success Level = "success"
	Warning Level = "warning"
	Error   Level = "error"
)

// Kind identifies the event that produced a notification.
type Kind string

const (
	// JobComplete identifies a completed background operation.
	JobComplete   Kind = "job_complete"
	Conflict      Kind = "conflict"
	HookFailure   Kind = "hook_failure"
	PushFailure   Kind = "push_failure"
	RemoteStale   Kind = "remote_stale"
	PluginFailure Kind = "plugin_failure"
)

// Notification is one user-visible event with dismissal state.
type Notification struct {
	ID        string
	Kind      Kind
	Level     Level
	Title     string
	Message   string
	At        time.Time
	Attention bool
	Dismissed bool
}

// Model maintains bounded notifications and attention counts.
type Model struct {
	mu     sync.RWMutex
	max    int
	quiet  bool
	nextID uint64
	items  []Notification
}

func New(max int, quiet bool) *Model {
	if max <= 0 {
		max = 100
	}
	return &Model{max: max, quiet: quiet}
}

// Add appends a notification and returns its generated stable ID.
func (m *Model) Add(notification Notification) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	if notification.At.IsZero() {
		notification.At = time.Now()
	}
	notification.ID = formatID(m.nextID)
	if m.quiet {
		notification.Attention = false
	}
	m.items = append(m.items, notification)
	if len(m.items) > m.max {
		m.items = m.items[len(m.items)-m.max:]
	}
	return notification.ID
}

// Dismiss marks a notification inactive and reports whether it was found.
func (m *Model) Dismiss(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.items {
		if m.items[i].ID == id {
			if m.items[i].Dismissed {
				return false
			}
			m.items[i].Dismissed = true
			m.items[i].Attention = false
			return true
		}
	}
	return false
}

// Items returns a copy of all retained notifications.
func (m *Model) Items() []Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Notification(nil), m.items...)
}

// Active returns a copy of notifications that still require attention.
func (m *Model) Active() []Notification {
	items := m.Items()
	items = filter(items, func(item Notification) bool { return !item.Dismissed })
	return items
}

// Attention returns the number of active notifications.
func (m *Model) Attention() int {
	count := 0
	for _, item := range m.Active() {
		if item.Attention {
			count++
		}
	}
	return count
}

// SortNewest returns notifications ordered from newest to oldest.
func SortNewest(items []Notification) []Notification {
	sorted := append([]Notification(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At.After(sorted[j].At) })
	return sorted
}

func formatID(value uint64) string { return "notification-" + strconv.FormatUint(value, 10) }

func filter(items []Notification, keep func(Notification) bool) []Notification {
	filtered := make([]Notification, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
