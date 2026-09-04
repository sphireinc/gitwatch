package repo

import (
	"time"

	"github.com/sphireinc/git-watch/internal/sequencer"
)

// Path preserves a repository path as bytes so unusual names survive parsing.
type Path []byte

// Bytes returns an owned copy of the path bytes.
func (p Path) Bytes() []byte  { return append([]byte(nil), p...) }
func (p Path) String() string { return string(p) }

// Entry is an immutable projection of one Git status path.
type Entry struct {
	Path       Path
	Original   Path
	Kind       byte
	XY         string
	Staged     bool
	Unstaged   bool
	Untracked  bool
	Conflicted bool
	Deleted    bool
	Renamed    bool
	Copied     bool
	ModeHead   string
	ModeIndex  string
	ModeWork   string
	Submodule  string
}

// ConflictType returns the human-readable conflict classification.
func (e Entry) ConflictType() string {
	if !e.Conflicted {
		return ""
	}
	switch e.XY {
	case "DD":
		return "both deleted"
	case "AU":
		return "added by us"
	case "UD":
		return "deleted by them"
	case "UA":
		return "added by them"
	case "DU":
		return "deleted by us"
	case "AA":
		return "both added"
	case "UU":
		return "both modified"
	default:
		return "unmerged"
	}
}

// Branch contains HEAD, upstream, and divergence metadata.
type Branch struct {
	Name     string
	OID      string
	Upstream string
	Ahead    int
	Behind   int
	Detached bool
	Unborn   bool
}

// Counts summarizes status categories in a snapshot.
type Counts struct{ Staged, Unstaged, Untracked, Conflicted, Added, Deleted int }

// Snapshot is the authoritative repository state used by the UI.
type Snapshot struct {
	Root            string
	GitDir          string
	Branch          Branch
	Entries         []Entry
	Counts          Counts
	Generation      uint64
	ObservedAt      time.Time
	RefreshDuration time.Duration
	// Operation is a Git-derived durable operation projection observed during
	// this same refresh generation. It is nil when no operation is active.
	Operation            *sequencer.State
	OperationDiagnostics []string
}

// Clone returns an independent snapshot copy safe for model ownership.
func (s Snapshot) Clone() Snapshot {
	s.Entries = append([]Entry(nil), s.Entries...)
	for i := range s.Entries {
		s.Entries[i].Path = append(Path(nil), s.Entries[i].Path...)
		s.Entries[i].Original = append(Path(nil), s.Entries[i].Original...)
	}
	if s.Operation != nil {
		operation := *s.Operation
		s.Operation = &operation
	}
	s.OperationDiagnostics = append([]string(nil), s.OperationDiagnostics...)
	return s
}

// StatusLabel returns the compact status symbol for an entry.
func StatusLabel(e Entry) string {
	switch {
	case e.Conflicted:
		return "conflict"
	case e.Untracked:
		return "untracked"
	case e.Renamed:
		return "renamed"
	case e.Deleted:
		return "deleted"
	case e.Staged && e.Unstaged:
		return "staged + modified"
	case e.Staged:
		return "staged"
	case e.Unstaged:
		return "modified"
	default:
		return "clean"
	}
}
