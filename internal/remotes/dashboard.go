package remotes

import "time"

// JobState describes the lifecycle state of a remote operation.
type JobState string

const (
	// JobQueued means the operation is waiting to run.
	JobQueued JobState = "queued"
	// JobRunning means the operation is currently executing.
	JobRunning JobState = "running"
	// JobSuccess means the operation completed successfully.
	JobSuccess JobState = "success"
	// JobFailed means the operation completed with an error.
	JobFailed JobState = "failed"
	// JobCanceled means the operation was canceled before completion.
	JobCanceled JobState = "canceled"
)

// Job records the observable state of one remote operation.
type Job struct {
	ID, Operation, Remote string
	State                 JobState
	Progress              string
	Started, Finished     time.Time
	Updated               time.Time
	Error                 string
}

// Activity is a bounded history entry for a remote operation.
type Activity struct {
	At                 time.Time
	Operation, Message string
	Success            bool
	Duration           time.Duration
}

type Dashboard struct {
	Remotes       []Remote
	Jobs          []Job
	Activity      []Activity
	CurrentBranch string
	Ahead, Behind int
	Now           time.Time
	StaleAfter    time.Duration
}

// Stale reports whether the dashboard has no recent data for remote.
func (d Dashboard) Stale(remote Remote) bool {
	if d.StaleAfter <= 0 || remote.LastFetchUnix == 0 || d.Now.IsZero() {
		return false
	}
	return d.Now.Sub(time.Unix(remote.LastFetchUnix, 0)) >= d.StaleAfter
}

// ActiveJobs returns a copy of jobs that have not reached a terminal state.
func (d Dashboard) ActiveJobs() []Job {
	var jobs []Job
	for _, job := range d.Jobs {
		if job.State == JobQueued || job.State == JobRunning {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// RecentActivity returns up to limit newest activity entries.
func (d Dashboard) RecentActivity(limit int) []Activity {
	if limit < 1 || limit >= len(d.Activity) {
		return append([]Activity(nil), d.Activity...)
	}
	start := len(d.Activity) - limit
	return append([]Activity(nil), d.Activity[start:]...)
}
