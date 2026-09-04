package rebaseview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/history"
)

func TestNewBuildsOldestFirstPlanAndKeepsExplicitBase(t *testing.T) {
	m, err := New("feature", "origin/main", []Base{{Label: "origin/main", Ref: "origin/main"}}, []history.Commit{
		{SHA: "new", Author: "A", Subject: "new subject"},
		{SHA: "old", Author: "B", Subject: "old subject"},
	})
	if err != nil {
		t.Fatal(err)
	}
	entries := m.Plan.Entries()
	if len(entries) != 2 || entries[0].SHA() != "old" || entries[1].SHA() != "new" {
		t.Fatalf("plan entries = %#v", entries)
	}
	if !m.StartEnabled() || m.Base.Ref != "origin/main" {
		t.Fatalf("start=%t base=%#v", m.StartEnabled(), m.Base)
	}
}

func TestBaseChoiceAndPublishedWarningAreVisible(t *testing.T) {
	m, err := New("main", "origin/main", []Base{{Label: "upstream", Ref: "origin/main"}, {Label: "parent", Ref: "HEAD~3"}}, []history.Commit{{SHA: "abc", Subject: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	m.Published, m.ReachableRemote, m.BaseMode = true, true, true
	m.Move(1)
	if err := m.SetBase(m.BaseSelected); err != nil || m.Base.Ref != "HEAD~3" {
		t.Fatalf("base=%#v err=%v", m.Base, err)
	}
	view := m.View()
	for _, want := range []string{"WARNING:", "Choose base:", "HEAD~3", "Start: ready"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestStartDisabledWhileLoadingOrInvalid(t *testing.T) {
	m, err := New("main", "", []Base{{Label: "base", Ref: "HEAD~1"}}, []history.Commit{{SHA: "abc", Subject: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	m.Loading = true
	if m.StartEnabled() {
		t.Fatal("loading workspace enabled start")
	}
	m.Loading = false
	m.Error = "plan is invalid"
	if m.StartEnabled() {
		t.Fatal("invalid workspace enabled start")
	}
}

func TestClickProvidesMouseParityForBasePlanAndFooter(t *testing.T) {
	m, err := New("main", "origin/main", []Base{{Label: "upstream", Ref: "origin/main"}, {Label: "parent", Ref: "HEAD~3"}}, []history.Commit{{SHA: "abc", Subject: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	if action, index := m.Click(2, 3, 80, 20); action != MouseChooseBase || index != -1 {
		t.Fatalf("base click = %v, %d", action, index)
	}
	if action, index := m.Click(2, 9, 80, 20); action != MouseSelectPlan || index != 0 {
		t.Fatalf("plan click = %v, %d", action, index)
	}
	m.BaseMode = true
	if action, index := m.Click(2, 10, 80, 20); action != MouseChooseBase || index != 1 {
		t.Fatalf("base choice click = %v, %d", action, index)
	}
	if action, _ := m.Click(2, 19, 80, 20); action != MouseCancel {
		t.Fatalf("cancel click = %v", action)
	}
	if action, _ := m.Click(70, 19, 80, 20); action != MouseStart {
		t.Fatalf("start click = %v", action)
	}
}

func TestMarkedActionsAndRangeMovesRemainValid(t *testing.T) {
	m, err := New("main", "origin/main", []Base{{Label: "upstream", Ref: "origin/main"}}, []history.Commit{
		{SHA: "new", Subject: "new"}, {SHA: "middle", Subject: "middle"}, {SHA: "old", Subject: "old"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.Selected = 0
	m.ToggleMark()
	m.Move(1)
	m.ToggleMark()
	if err := m.ApplyAction("edit", false); err != nil {
		t.Fatal(err)
	}
	if err := m.MoveSelection(1); err != nil {
		t.Fatal(err)
	}
	entries := m.Plan.Entries()
	if entries[1].SHA() != "old" || entries[2].SHA() != "middle" || entries[1].Action() != "edit" || entries[2].Action() != "edit" {
		t.Fatalf("edited/moved entries = %#v", entries)
	}
}
