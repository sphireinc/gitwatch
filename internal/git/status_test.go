package git

import "testing"

func TestParseStatusAllRecordKinds(t *testing.T) {
	data := []byte("# branch.head main\x00# branch.oid abc\x00# branch.upstream origin/main\x00# branch.ab +2 -1\x001 .M N... 100644 100644 100644 abc def file with space\x002 R. N... 100644 100644 100644 abc def R100 new name\x00old name\x00u UU N... 100644 100644 100644 100644 111 222 333 conflict\x00? odd\tname\x00! ignored\x00")
	got, err := ParseStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.BranchHead != "main" || got.Ahead != 2 || got.Behind != 1 || len(got.Entries) != 5 {
		t.Fatalf("unexpected status: %+v", got)
	}
	if string(got.Entries[1].OrigPath) != "old name" || string(got.Entries[2].Path) != "conflict" {
		t.Fatalf("rename/conflict parsing failed: %+v", got.Entries)
	}
}

func TestParseStatusRejectsMalformed(t *testing.T) {
	for _, data := range [][]byte{[]byte("1 .M\x00"), []byte("x foo\x00"), []byte("? missing-nul")} {
		if _, err := ParseStatus(data); err == nil {
			t.Errorf("expected parse error for %q", data)
		}
	}
}
