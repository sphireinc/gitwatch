package remotes

import (
	"testing"
	"time"
)

func TestDashboardStalenessAndActivityWindows(t *testing.T) {
	now := time.Unix(1_000, 0)
	dashboard := Dashboard{Now: now, StaleAfter: time.Minute, Activity: []Activity{{Operation: "fetch"}, {Operation: "pull"}}}
	remote := Remote{LastFetchUnix: 900}
	if !dashboard.Stale(remote) {
		t.Fatal("old fetch was not marked stale")
	}
	jobs := Dashboard{Jobs: []Job{{State: JobSuccess}, {State: JobRunning}, {State: JobQueued}}}.ActiveJobs()
	if len(jobs) != 2 {
		t.Fatalf("unexpected active jobs: %#v", jobs)
	}
	if recent := dashboard.RecentActivity(1); len(recent) != 1 || recent[0].Operation != "pull" {
		t.Fatalf("unexpected recent activity: %#v", recent)
	}
}
