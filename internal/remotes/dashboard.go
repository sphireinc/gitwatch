package remotes

import "time"

type JobState string

const (
	JobQueued   JobState = "queued"
	JobRunning  JobState = "running"
	JobSuccess  JobState = "success"
	JobFailed   JobState = "failed"
	JobCanceled JobState = "canceled"
)

type Job struct {
	ID, Operation, Remote string
	State                 JobState
	Progress              string
	Started, Finished     time.Time
	Updated               time.Time
	Error                 string
}

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

func (d Dashboard) Stale(remote Remote) bool {
	if d.StaleAfter <= 0 || remote.LastFetchUnix == 0 || d.Now.IsZero() {
		return false
	}
	return d.Now.Sub(time.Unix(remote.LastFetchUnix, 0)) >= d.StaleAfter
}

func (d Dashboard) ActiveJobs() []Job {
	var jobs []Job
	for _, job := range d.Jobs {
		if job.State == JobQueued || job.State == JobRunning {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (d Dashboard) RecentActivity(limit int) []Activity {
	if limit < 1 || limit >= len(d.Activity) {
		return append([]Activity(nil), d.Activity...)
	}
	start := len(d.Activity) - limit
	return append([]Activity(nil), d.Activity[start:]...)
}
