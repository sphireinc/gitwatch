package patch

import "testing"

func TestParsePatchPreservesHunks(t *testing.T) {
	input := "diff --git a/a.txt b/a.txt\n--- a/a.txt\n+++ b/a.txt\n@@ -1,2 +1,2 @@\n one\n-old\n+new\n\\ No newline at end of file\n"
	files, e := Parse(input)
	if e != nil {
		t.Fatal(e)
	}
	if len(files) != 1 || len(files[0].Hunks) != 1 || files[0].Hunks[0].Lines[2].Kind != Added || !files[0].Hunks[0].Lines[3].NoNewline {
		t.Fatal(files)
	}
}
func TestParseBinaryAndMalformed(t *testing.T) {
	files, e := Parse("diff --git a/a b/b\nBinary files a/a and b/b differ\n")
	if e != nil || !files[0].Binary {
		t.Fatal(files, e)
	}
	if _, e = Parse("diff --git a/a b/b\n@@ bad\n"); e == nil {
		t.Fatal("expected malformed error")
	}
}

func TestParseHeaderWithSpaces(t *testing.T) {
	files, err := Parse("diff --git a/old name.txt b/new name.txt\n@@ -1 +1 @@\n-old\n+new\n")
	if err != nil || len(files) != 1 || files[0].OldPath != "old name.txt" || files[0].NewPath != "new name.txt" {
		t.Fatalf("unexpected paths: %#v, %v", files, err)
	}
}

func TestParseQuotedHeader(t *testing.T) {
	files, err := Parse("diff --git \"a/old\\tname.txt\" \"b/new\\tname.txt\"\n")
	if err != nil || len(files) != 1 || files[0].OldPath != "old\tname.txt" || files[0].NewPath != "new\tname.txt" {
		t.Fatalf("unexpected quoted paths: %#v, %v", files, err)
	}
}

func FuzzParseNeverPanics(f *testing.F) {
	f.Add("diff --git a/a b/b\n@@ -1 +1 @@\n-a\n+b\n")
	f.Add("\x00\\\"@@ malformed")
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = Parse(input)
	})
}
