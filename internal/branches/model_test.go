package branches

import (
	"errors"
	"testing"

	"github.com/jusanchez/gitwatch/internal/git"
)

func TestParse(t *testing.T) {
	v := Parse([]byte("main\x00abc\x00origin/main\x00*\nfeature\x00def\x00\x00 \n"))
	if len(v) != 2 || !v[0].Current || v[1].Remote {
		t.Fatal(v)
	}
}

func TestDeleteGuardsCurrentBranchAndExactConfirmation(t *testing.T) {
	branch := Branch{Name: "feature", Current: false}
	if _, err := Delete(nil, git.Runner{}, branch, DeletePrompt("feature", false), "wrong"); !errors.Is(err, ErrConfirmation) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	branch.Current = true
	if _, err := Delete(nil, git.Runner{}, branch, DeletePrompt("feature", false), "feature"); !errors.Is(err, ErrCurrentBranch) {
		t.Fatalf("expected current branch error, got %v", err)
	}
}
