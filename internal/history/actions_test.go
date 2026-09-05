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

func TestRevertPlanPreservesExplicitOrder(t *testing.T) {
	plan := RevertPlan{Commits: []string{"oldest", "newest"}}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	confirmation := RevertConfirmation{SHA: "oldest newest"}
	if !confirmation.Accept("oldest newest") {
		t.Fatal("ordered confirmation rejected")
	}
	if err := (RevertPlan{Commits: []string{"--bad"}}).Validate(); !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("unsafe plan error = %v", err)
	}
}
