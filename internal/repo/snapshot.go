package repo

import "time"

type Path []byte

func (p Path) Bytes() []byte  { return append([]byte(nil), p...) }
func (p Path) String() string { return string(p) }

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
}

type Branch struct {
	Name     string
	OID      string
	Upstream string
	Ahead    int
	Behind   int
	Detached bool
	Unborn   bool
}

type Counts struct{ Staged, Unstaged, Untracked, Conflicted, Added, Deleted int }

type Snapshot struct {
	Root            string
	GitDir          string
	Branch          Branch
	Entries         []Entry
	Counts          Counts
	Generation      uint64
	ObservedAt      time.Time
	RefreshDuration time.Duration
}

func (s Snapshot) Clone() Snapshot {
	s.Entries = append([]Entry(nil), s.Entries...)
	for i := range s.Entries {
		s.Entries[i].Path = append(Path(nil), s.Entries[i].Path...)
		s.Entries[i].Original = append(Path(nil), s.Entries[i].Original...)
	}
	return s
}

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
