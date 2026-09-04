package rebase

import (
	"strings"
	"testing"
)

const goldenTodo = "# Rebase plan\n\npick aaa111 first subject\nexec echo keep-me\nreword bbb222 second subject\nlabel branch-point\nunknown-command opaque payload\nsquash ccc333 third subject\n"

func TestParseRenderPreservesGoldenTodo(t *testing.T) {
	plan, err := Parse(goldenTodo)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Render(); got != goldenTodo {
		t.Fatalf("round trip changed todo:\n%s", got)
	}
	entries := plan.Entries()
	if len(entries) != 8 || entries[0].Kind() != CommentEntry || entries[1].Kind() != BlankEntry || entries[2].Action() != Pick || entries[3].Kind() != DirectiveEntry || entries[4].SHA() != "bbb222" || entries[5].Raw() != "label branch-point" {
		t.Fatalf("parsed entries = %#v", entries)
	}
	if entries[2].OriginalIndex() != 2 || entries[2].Subject() != "first subject" {
		t.Fatalf("commit metadata = %#v", entries[2])
	}
}

func TestChangeActionIsPureAndCanonicalizesOnlyChangedCommit(t *testing.T) {
	plan, err := Parse("pick aaa first\n  # preserved comment\nedit bbb second\n")
	if err != nil {
		t.Fatal(err)
	}
	changed, err := plan.ChangeAction(2, Fixup)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Render() != "pick aaa first\n  # preserved comment\nedit bbb second\n" {
		t.Fatal("original plan was mutated")
	}
	if got := changed.Render(); got != "pick aaa first\n  # preserved comment\nfixup bbb second\n" {
		t.Fatalf("changed plan = %q", got)
	}
	if _, err := changed.ChangeAction(0, Fixup); err == nil || !strings.Contains(err.Error(), "first commit") {
		t.Fatalf("invalid first fixup was accepted: %v", err)
	}
}

func TestMoveRangeRetainsUnknownDirectivesAndValidatesBounds(t *testing.T) {
	plan, err := Parse("pick aaa first\nunknown keep\npick bbb second\nfixup ccc third\n")
	if err != nil {
		t.Fatal(err)
	}
	moved, err := plan.MoveRange(2, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := moved.Render(); got != "pick bbb second\nfixup ccc third\npick aaa first\nunknown keep\n" {
		t.Fatalf("moved plan = %q", got)
	}
	if len(moved.Entries()) != len(plan.Entries()) {
		t.Fatal("move dropped entries")
	}
	if _, err := plan.MoveEntry(0, 1); err == nil {
		t.Fatal("overlapping move was accepted")
	}
	if _, err := plan.MoveRange(-1, 1, 0); err == nil {
		t.Fatal("out-of-bounds move was accepted")
	}
}

func TestSquashTargetAndLogicalGroups(t *testing.T) {
	plan, err := Parse("# header\npick aaa first\ncomment directive\nsquash bbb second\nfixup ccc third\npick ddd fourth\n")
	if err != nil {
		t.Fatal(err)
	}
	target, err := plan.SquashTarget(3)
	if err != nil || target != 1 {
		t.Fatalf("squash target=%d err=%v", target, err)
	}
	if _, err := plan.SquashTarget(1); err == nil {
		t.Fatal("first commit was accepted as squash target")
	}
	groups := plan.LogicalGroups()
	if len(groups) != 3 || len(groups[1]) != 4 || groups[1][0].SHA() != "aaa" || groups[1][3].Action() != Fixup {
		t.Fatalf("logical groups = %#v", groups)
	}
}

func TestParseRejectsMalformedEditableActionAndBoundsInput(t *testing.T) {
	if _, err := Parse("pick\n"); err == nil {
		t.Fatal("malformed pick was accepted")
	}
	if _, err := Parse(strings.Repeat("x", maxPlanBytes+1)); err == nil {
		t.Fatal("oversized todo was accepted")
	}
	plan, err := Parse("pick aaa first\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.ChangeAction(1, Drop); err == nil {
		t.Fatal("invalid index was accepted")
	}
}
