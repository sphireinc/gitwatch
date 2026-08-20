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

func TestPushTagRequiresSafeExplicitTag(t *testing.T) {
	_, err := PushTag(nil, git.Runner{}, "origin", "-v1")
	if !errors.Is(err, ErrMissingTag) {
		t.Fatalf("expected tag validation error, got %v", err)
	}
}

func TestPushSetUpstreamRejectsMissingRemote(t *testing.T) {
	_, err := PushSetUpstream(nil, git.Runner{}, "", "main")
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
