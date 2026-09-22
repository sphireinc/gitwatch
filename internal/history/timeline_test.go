package history

import "testing"

func TestFilterEventsByRepositoryKindAndOutcome(t *testing.T) {
	events := []Event{
		{Kind: OperationSuccess, Operation: &OperationRecord{Repository: "/one", Kind: "fetch", Outcome: "success"}},
		{Kind: OperationFailure, Operation: &OperationRecord{Repository: "/two", Kind: "merge", Outcome: "failure"}},
		{Kind: RefreshError, Message: "offline"},
	}
	filtered := FilterEvents(events, TimelineFilter{Repository: "/two", Kind: "merge", Outcome: "failure"})
	if len(filtered) != 1 || filtered[0].Operation == nil || filtered[0].Operation.Repository != "/two" {
		t.Fatalf("filtered = %#v", filtered)
	}
	if got := FilterEvents(events, TimelineFilter{Kind: "refresh"}); len(got) != 1 || got[0].Kind != RefreshError {
		t.Fatalf("kind filter = %#v", got)
	}
}

func TestParseTimelineFilterSupportsFieldSelectors(t *testing.T) {
	filter := ParseTimelineFilter("repo:/repo-a type:merge outcome:success")
	if filter.Repository != "/repo-a" || filter.Kind != "merge" || filter.Outcome != "success" {
		t.Fatalf("parsed timeline filter = %#v", filter)
	}
	if filter := ParseTimelineFilter("/repo-a"); filter.Repository != "/repo-a" {
		t.Fatalf("plain timeline filter = %#v", filter)
	}
}

func TestOperationClassSeparatesBackgroundWorkFromHistoryMutation(t *testing.T) {
	for _, test := range []struct {
		kind, want string
	}{
		{"fetch", "background"},
		{"GitHub checks refresh", "background"},
		{"custom command inspect", "background"},
		{"merge", "history mutation"},
		{"rebase", "history mutation"},
	} {
		if got := OperationClass(test.kind); got != test.want {
			t.Fatalf("OperationClass(%q) = %q, want %q", test.kind, got, test.want)
		}
	}
}
