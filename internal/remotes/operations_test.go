package remotes

import (
	"errors"
	"testing"

	"github.com/jusanchez/gitwatch/internal/git"
)

func TestPullRequiresExplicitStrategy(t *testing.T) {
	_, err := Pull(nil, git.Runner{}, "origin", "main", "")
	if !errors.Is(err, ErrStrategyRequired) {
		t.Fatalf("expected explicit strategy error, got %v", err)
	}
}

func TestRemoteOperationsRejectOptionLikeNames(t *testing.T) {
	_, err := Push(nil, git.Runner{}, "-origin", "main", false)
	if !errors.Is(err, ErrMissingRemote) {
		t.Fatalf("expected remote validation error, got %v", err)
	}
}
