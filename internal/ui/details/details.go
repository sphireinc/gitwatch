// Package details builds cached render data for selected repository paths.
package details

import (
	"sync"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

// View is the render-ready metadata for one repository entry.
type View struct {
	Path, PreviousPath, Status, Mode, Hint, SubmoduleState string
	Staged, Unstaged, Conflict, Submodule                  bool
	ObservedAt                                             time.Time
}

// Cache memoizes detail projections for one snapshot generation at a time.
type Cache struct {
	mu         sync.RWMutex
	generation uint64
	values     map[string]View
}

// NewCache creates an empty detail projection cache.
func NewCache() *Cache { return &Cache{values: make(map[string]View)} }

// For returns cached details or builds them for the current snapshot generation.
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

// Build derives user-facing metadata and action hints from an entry.
func Build(snapshot repo.Snapshot, entry repo.Entry) View {
	v := View{Path: string(entry.Path), PreviousPath: string(entry.Original), Status: repo.StatusLabel(entry), Staged: entry.Staged, Unstaged: entry.Unstaged, Conflict: entry.Conflicted, Mode: entry.ModeWork, SubmoduleState: entry.Submodule, ObservedAt: snapshot.ObservedAt}
	v.Submodule = entry.Submodule != "" && entry.Submodule != "N..."
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
