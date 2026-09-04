package conflicts

import "testing"

func TestParseIndexPreservesStagesAndWeirdPaths(t *testing.T) {
	data := []byte("100644 base 1\tspace\tunicode-é\x00100644 ours 2\tspace\tunicode-é\x00100644 theirs 3\tspace\tunicode-é\x00")
	got, err := ParseIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || string(got[0].Path) != "space\tunicode-é" {
		t.Fatalf("unexpected conflicts: %+v", got)
	}
	if got[0].Base.OID != "base" || got[0].Ours.OID != "ours" || got[0].Theirs.OID != "theirs" {
		t.Fatalf("stages lost: %+v", got[0])
	}
}

func TestCorrelateHandlesMissingStages(t *testing.T) {
	index, err := ParseIndex([]byte("100644 ours 2\tdeleted\x00100644 theirs 3\tdeleted\x00"))
	if err != nil {
		t.Fatal(err)
	}
	got := Correlate(index, []Status{{Path: []byte("deleted"), XY: "UD", Worktree: "worktree-deleted"}})
	if got[0].Base.OID != "" || got[0].Kind != ModifyDelete || got[0].Resolution != "unmerged" {
		t.Fatalf("missing stage correlation failed: %+v", got[0])
	}
}

func TestParseIndexRejectsMalformedRecord(t *testing.T) {
	if _, err := ParseIndex([]byte("100644 bad 4\tpath\x00")); err == nil {
		t.Fatal("expected invalid stage error")
	}
}
