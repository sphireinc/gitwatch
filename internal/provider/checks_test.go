package provider

import (
	"testing"
	"time"
)

func TestParseChecksAndDuration(t *testing.T) {
	snapshot, err := ParseChecks([]byte(`{"check_runs":[{"name":"build","status":"completed","conclusion":"success","started_at":"2026-08-20T12:00:00Z","completed_at":"2026-08-20T12:02:00Z"},{"name":"lint","status":"in_progress","conclusion":null}]}`))
	if err != nil || snapshot.Passing != 1 || snapshot.Pending != 1 || len(snapshot.Runs) != 2 {
		t.Fatalf("unexpected checks: %#v, %v", snapshot, err)
	}
	if got := snapshot.Runs[0].Duration(); got != 2*time.Minute {
		t.Fatalf("unexpected duration: %v", got)
	}
}
