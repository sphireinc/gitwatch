package history

import "testing"

func TestParseLog(t *testing.T) {
	value := ParseLog([]byte("abc\x00abc\x00Alice\x001700000000\x00p1 p2\x00subject\x1e"))
	if len(value) != 1 || len(value[0].Parents) != 2 || value[0].Subject != "subject" {
		t.Fatal(value)
	}
}

func TestParseLogMetadata(t *testing.T) {
	value := ParseLog([]byte("abc\x00abc\x00Alice\x001700000000\x00p1\x00HEAD, main\x00G\x00subject\x1e"))
	if len(value) != 1 || len(value[0].Refs) != 2 || value[0].Signature != "G" || value[0].Subject != "subject" {
		t.Fatalf("unexpected metadata: %#v", value)
	}
}

func TestParseLogPreservesNewlineInSubject(t *testing.T) {
	value := ParseLog([]byte("abc\x00abc\x00Alice\x001700000000\x00\x00subject\ncontinued\x1e"))
	if len(value) != 1 || value[0].Subject != "subject\ncontinued" {
		t.Fatalf("unexpected commit: %#v", value)
	}
}
