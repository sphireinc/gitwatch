package provider

import (
	"testing"
	"time"
)

func TestParseChecksAndDuration(t *testing.T) {
	snapshot, err := ParseChecks([]byte(`{"check_runs":[{"id":42,"name":"build","status":"completed","conclusion":"success","run_attempt":2,"started_at":"2026-08-20T12:00:00Z","completed_at":"2026-08-20T12:02:00Z"},{"name":"lint","status":"in_progress","conclusion":null}]}`))
	if err != nil || snapshot.Passing != 1 || snapshot.Pending != 1 || len(snapshot.Runs) != 2 {
		t.Fatalf("unexpected checks: %#v, %v", snapshot, err)
	}
	if got := snapshot.Runs[0].Duration(); got != 2*time.Minute {
		t.Fatalf("unexpected duration: %v", got)
	}
	if snapshot.Runs[0].ID != 42 || snapshot.Runs[0].Attempt != 2 {
		t.Fatalf("missing check-run identity: %#v", snapshot.Runs[0])
	}
	tooMany := make([]byte, 0, MaxCheckRuns*30)
	tooMany = append(tooMany, `{"check_runs":[`...)
	for i := 0; i < MaxCheckRuns+1; i++ {
		if i > 0 {
			tooMany = append(tooMany, ',')
		}
		tooMany = append(tooMany, `{"name":"run"}`...)
	}
	tooMany = append(tooMany, `]}`...)
	if _, err := ParseChecks(tooMany); err == nil {
		t.Fatal("oversized check-run page was accepted")
	}
}
