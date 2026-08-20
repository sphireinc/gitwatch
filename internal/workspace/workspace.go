package workspace

import (
	"context"
	"sync"
)

type View string

const (
	Status       View = "status"
	Commit       View = "commit"
	Stashes      View = "stashes"
	Branches     View = "branches"
	Log          View = "log"
	Remotes      View = "remotes"
	GitHub       View = "github"
	Plugins      View = "plugins"
	Repositories View = "repositories"
)

type Breadcrumb struct {
	Label string
	View  View
}
type ModalOwner string
type JobState uint8

const (
	JobIdle JobState = iota
	JobRunning
	JobSucceeded
	JobFailed
	JobCancelled
)

type Job struct {
	ID, Name string
	State    JobState
	Err      error
	Cancel   context.CancelFunc
}

type Model struct {
	mu          sync.RWMutex
	Current     View
	Breadcrumbs []Breadcrumb
	Modal       ModalOwner
	Jobs        map[string]Job
}

func New() *Model {
	return &Model{Current: Status, Breadcrumbs: []Breadcrumb{{Label: "Status", View: Status}}, Jobs: make(map[string]Job)}
}
func (m *Model) Navigate(view View, label string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Current = view
	m.Breadcrumbs = append(m.Breadcrumbs, Breadcrumb{Label: label, View: view})
}
func (m *Model) Back() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Breadcrumbs) > 1 {
		m.Breadcrumbs = m.Breadcrumbs[:len(m.Breadcrumbs)-1]
		m.Current = m.Breadcrumbs[len(m.Breadcrumbs)-1].View
	}
}
func (m *Model) SetModal(owner ModalOwner) { m.mu.Lock(); defer m.mu.Unlock(); m.Modal = owner }
func (m *Model) StartJob(ctx context.Context, id, name string) context.Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	jobCtx, cancel := context.WithCancel(ctx)
	m.Jobs[id] = Job{ID: id, Name: name, State: JobRunning, Cancel: cancel}
	return jobCtx
}
func (m *Model) FinishJob(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.Jobs[id]
	j.Err = err
	if err == context.Canceled {
		j.State = JobCancelled
	} else if err != nil {
		j.State = JobFailed
	} else {
		j.State = JobSucceeded
	}
	m.Jobs[id] = j
}
func (m *Model) Snapshot() (View, []Breadcrumb, ModalOwner, map[string]Job) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jobs := make(map[string]Job, len(m.Jobs))
	for k, v := range m.Jobs {
		jobs[k] = v
	}
	return m.Current, append([]Breadcrumb(nil), m.Breadcrumbs...), m.Modal, jobs
}
