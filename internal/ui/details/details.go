package details

import (
	"sync"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

type View struct {
	Path, PreviousPath, Status, Mode, Hint string
	Staged, Unstaged, Conflict, Submodule  bool
	ObservedAt                             time.Time
}

type Cache struct {
	mu         sync.RWMutex
	generation uint64
	values     map[string]View
}

func NewCache() *Cache { return &Cache{values: make(map[string]View)} }

func (c *Cache) For(snapshot repo.Snapshot, entry repo.Entry) View {
	key := string(entry.Path)
	c.mu.RLock()
	if c.generation == snapshot.Generation {
		if v, ok := c.values[key]; ok {
			c.mu.RUnlock()
			return v
		}
	}
	c.mu.RUnlock()
	v := Build(snapshot, entry)
	c.mu.Lock()
	if c.generation != snapshot.Generation {
		c.generation = snapshot.Generation
		c.values = make(map[string]View)
	}
	c.values[key] = v
	c.mu.Unlock()
	return v
}

func Build(snapshot repo.Snapshot, entry repo.Entry) View {
	v := View{Path: string(entry.Path), PreviousPath: string(entry.Original), Status: repo.StatusLabel(entry), Staged: entry.Staged, Unstaged: entry.Unstaged, Conflict: entry.Conflicted, Mode: entry.ModeWork, ObservedAt: snapshot.ObservedAt}
	if entry.Conflicted {
		v.Hint = "Resolve externally, then stage the resolved path"
	} else if entry.Untracked {
		v.Hint = "Space stages this untracked path"
	} else if entry.Staged && !entry.Unstaged {
		v.Hint = "Space unstages this path"
	} else {
		v.Hint = "Space stages this path · d opens diff"
	}
	if v.Mode == "" {
		v.Mode = "regular"
	}
	return v
}
