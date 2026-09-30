package provider

import (
	"strings"
	"testing"
	"time"
)

func TestParseChecksAndDuration(t *testing.T) {
	snapshot, err := ParseChecks([]byte(`{"total_count":2,"check_runs":[{"id":42,"name":"build","status":"completed","conclusion":"success","run_attempt":2,"started_at":"2026-08-20T12:00:00Z","completed_at":"2026-08-20T12:02:00Z"},{"name":"lint","status":"in_progress","conclusion":null}]}`))
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
	tooMany = append(tooMany, `{"total_count":101,"check_runs":[`...)
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

func TestParseChecksRejectsIncompletePageEvenWhenReturnedRunFailed(t *testing.T) {
	run := `{"id":1,"name":"required?","status":"completed","conclusion":"failure"}`
	page := `{"total_count":101,"check_runs":[` + strings.TrimSuffix(strings.Repeat(run+",", MaxCheckRuns), ",") + `]}`
	if _, err := ParseChecks([]byte(page)); err == nil {
		t.Fatal("incomplete check page was accepted despite a failed observed check")
	}
}

func TestParseChecksRequiresCompleteCount(t *testing.T) {
	for _, body := range []string{`{"check_runs":[]}`, `{"total_count":2,"check_runs":[]}`, `{"total_count":0,"check_runs":[{"name":"extra"}]}`, "null"} {
		if _, err := ParseChecks([]byte(body)); err == nil {
			t.Errorf("invalid or incomplete checks response accepted: %s", body)
		}
	}
}
