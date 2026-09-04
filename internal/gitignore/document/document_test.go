package document

import (
	"bytes"
	"errors"
	"testing"
)

func TestParseRenderIsLossless(t *testing.T) {
	input := append([]byte{0xef, 0xbb, 0xbf}, []byte("# comment\r\n\\#escaped\n!keep\\ rule\rblank\r\nno-final")...)
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc.Render(), input) || !doc.HasBOM || doc.FinalNewline || doc.DominantNewline == "" {
		t.Fatal("document was not lossless")
	}
	if doc.Lines[0].Kind != Comment || doc.Lines[1].Kind != Rule || doc.Lines[2].Kind != Rule || doc.Lines[3].Kind != Rule {
		t.Fatalf("unexpected line kinds: %+v", doc.Lines)
	}
	if got, _ := doc.Span(0, 1); !bytes.Equal(got, append([]byte{0xef, 0xbb, 0xbf}, []byte("# comment\r\n\\#escaped\n")...)) {
		t.Fatalf("span=%q", got)
	}
}

func TestManagedMarkersStrictAndNested(t *testing.T) {
	input := []byte("# gitwatch:begin template=root/Node version=1\r\n*.node\r\n# gitwatch:end template=root/Node\r\n# gitwatch:begin template=root/PHP version=1\n# gitwatch:begin template=root/Go version=1\n# gitwatch:end template=root/PHP\n")
	doc, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Blocks) != 3 || !doc.Blocks[0].Valid || doc.Blocks[1].Valid || doc.Blocks[2].Valid {
		t.Fatalf("blocks=%+v", doc.Blocks)
	}
}

func TestBinaryRejected(t *testing.T) {
	_, err := Parse([]byte("ok\x00bad"))
	if !errors.Is(err, ErrBinary) {
		t.Fatalf("error=%v", err)
	}
}

func FuzzParseRender(f *testing.F) {
	for _, seed := range [][]byte{nil, []byte("a\r\nb\n"), []byte{0xef, 0xbb, 0xbf, '#', 'x'}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if bytes.IndexByte(input, 0) >= 0 {
			return
		}
		doc, err := Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(doc.Render(), input) {
			t.Fatalf("render changed input")
		}
	})
}
