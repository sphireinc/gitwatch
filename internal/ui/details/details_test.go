package details

import (
	"testing"
	"github.com/jusanchez/gitwatch/internal/repo"
)

func TestDetailsAndGenerationCache(t *testing.T) { s:=repo.Snapshot{Generation:1}; e:=repo.Entry{Path:repo.Path("new"),Untracked:true}; c:=NewCache(); v:=c.For(s,e); if v.Status!="untracked" || v.Hint=="" { t.Fatal(v) }; e.Untracked=false; e.Staged=true; if c.For(s,e).Status!="untracked" { t.Fatal("cache should be generation scoped") }; s.Generation=2; if c.For(s,e).Status!="staged" { t.Fatal("cache did not invalidate") } }
