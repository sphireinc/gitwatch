package history

import (
	"context"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestRevertConfirmationRequiresExactSHA(t *testing.T) {
	confirmation := RevertConfirmation{SHA: "abc123", Subject: "fix"}
	if confirmation.Accept(" abc123 ") != true || confirmation.Accept("abc") {
		t.Fatal("confirmation did not enforce the exact target")
	}
	if _, err := CheckoutCommit(context.Background(), git.Runner{}, ""); !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("expected missing target, got %v", err)
	}
}

func TestHistoryActionsRejectOptionLikeTargets(t *testing.T) {
	if _, err := CreateBranchAt(context.Background(), git.Runner{}, "feature", "-bad"); !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("expected target validation, got %v", err)
	}
}
