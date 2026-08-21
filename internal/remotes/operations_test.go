package remotes

import (
	"context"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestPullRequiresExplicitStrategy(t *testing.T) {
	_, err := Pull(context.Background(), git.Runner{}, "origin", "main", "")
	if !errors.Is(err, ErrStrategyRequired) {
		t.Fatalf("expected explicit strategy error, got %v", err)
	}
}

func TestRemoteOperationsRejectOptionLikeNames(t *testing.T) {
	_, err := Push(context.Background(), git.Runner{}, "-origin", "main", false)
	if !errors.Is(err, ErrMissingRemote) {
		t.Fatalf("expected remote validation error, got %v", err)
	}
}

func TestPushTagRequiresSafeExplicitTag(t *testing.T) {
	_, err := PushTag(context.Background(), git.Runner{}, "origin", "-v1")
	if !errors.Is(err, ErrMissingTag) {
		t.Fatalf("expected tag validation error, got %v", err)
	}
}

func TestPushSetUpstreamRejectsMissingRemote(t *testing.T) {
	_, err := PushSetUpstream(context.Background(), git.Runner{}, "", "main")
	if !errors.Is(err, ErrMissingRemote) {
		t.Fatalf("expected remote validation error, got %v", err)
	}
}

func TestParseRemoteSHA(t *testing.T) {
	if got := parseRemoteSHA([]byte("abc123\trefs/heads/main\n")); got != "abc123" {
		t.Fatalf("remote SHA = %q", got)
	}
	if got := parseRemoteSHA(nil); got != "" {
		t.Fatalf("empty remote SHA = %q", got)
	}
}
