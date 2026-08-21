package details

import (
	"github.com/sphireinc/git-watch/internal/repo"
	"testing"
)

func TestDetailsAndGenerationCache(t *testing.T) {
	s := repo.Snapshot{Generation: 1}
	e := repo.Entry{Path: repo.Path("new"), Untracked: true}
	c := NewCache()
	v := c.For(s, e)
	if v.Status != "untracked" || v.Hint == "" {
		t.Fatal(v)
	}
	e.Untracked = false
	e.Staged = true
	if c.For(s, e).Status != "untracked" {
		t.Fatal("cache should be generation scoped")
	}
	s.Generation = 2
	if c.For(s, e).Status != "staged" {
		t.Fatal("cache did not invalidate")
	}
}

func TestDetailsExposeSubmoduleState(t *testing.T) {
	view := Build(repo.Snapshot{}, repo.Entry{Path: repo.Path("module"), Submodule: "SC.M", ModeWork: "160000"})
	if !view.Submodule || view.SubmoduleState != "SC.M" || view.Mode != "160000" {
		t.Fatalf("view = %#v", view)
	}
}
