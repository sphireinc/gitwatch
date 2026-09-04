package managed

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func TestManagedBlockRoundTripAndNewlineAdaptation(t *testing.T) {
	id := domain.TemplateID("root/Node")
	encoded, err := EncodeManagedBlock(id, "github/gitignore", "commit", "sha256", []byte("# comment\nnode_modules/\n"), []byte("\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseManagedBlock(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.TemplateID != id || parsed.Source != "github/gitignore" || !bytes.Equal(parsed.Body, []byte("# comment\r\nnode_modules/\r\n")) {
		t.Fatalf("parsed=%+v body=%q", parsed, parsed.Body)
	}
}

func TestManagedBlockRejectsUnknownVersionAndMismatchedID(t *testing.T) {
	unknown := []byte("# >>> gitwatch:gitignore begin format=2 id=root/Node source=x commit=x hash=x\nbody\n# <<< gitwatch:gitignore end format=2 id=root/Node\n")
	if !errors.Is(mustParseError(unknown), ErrUnknownFormat) {
		t.Fatal("unknown format accepted")
	}
	mismatch := []byte("# >>> gitwatch:gitignore begin format=1 id=root/Node source=x commit=x hash=x\nbody\n# <<< gitwatch:gitignore end format=1 id=root/Go\n")
	if !errors.Is(mustParseError(mismatch), ErrMismatchedID) {
		t.Fatal("mismatched IDs accepted")
	}
}

func TestAdjacentBlocksRemainIndependentlyParseable(t *testing.T) {
	first, _ := EncodeManagedBlock("root/Node", "x", "a", "h1", []byte("*.node\n"), []byte("\n"))
	second, _ := EncodeManagedBlock("root/Go", "x", "b", "h2", []byte("*.go\n"), []byte("\n"))
	for _, input := range [][]byte{first, second} {
		block, err := ParseManagedBlock(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := Validate(block); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Contains(first, []byte("*.node")) || !bytes.Contains(second, []byte("*.go")) {
		t.Fatal("block body missing")
	}
}

func mustParseError(input []byte) error { _, err := ParseManagedBlock(input); return err }
