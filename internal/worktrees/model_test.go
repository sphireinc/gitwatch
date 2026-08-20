package worktrees

import "testing"

func TestParse(t *testing.T) {
	v := Parse([]byte("worktree /tmp/main\nHEAD abc\nbranch refs/heads/main\n\nworktree /tmp/other\nHEAD def\ndetached\n"))
	if len(v) != 2 || v[1].Branch != "" || !v[1].Detached {
		t.Fatal(v)
	}
}
