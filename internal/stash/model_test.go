package stash

import (
	"context"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestParse(t *testing.T) {
	v := Parse([]byte("stash@{0} abc 1700000000 On main: save work\n"))
	if len(v) != 1 || v[0].Ref != "stash@{0}" || v[0].Unix != 1700000000 {
		t.Fatal(v)
	}
}

func TestMutationRejectsUnsafeReferences(t *testing.T) {
	if _, err := Apply(context.Background(), git.Runner{}, "-evil"); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("expected invalid ref, got %v", err)
	}
	if _, err := Branch(context.Background(), git.Runner{}, "-branch", "stash@{0}"); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("expected invalid branch name, got %v", err)
	}
}
