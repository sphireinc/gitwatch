// Package conflicts models the unmerged index without depending on Git's
// localized human-readable diagnostics.
package conflicts

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

type Stage struct {
	Mode string
	OID  string
}

type Kind string

const (
	BothModified Kind = "both-modified"
	AddAdd       Kind = "add-add"
	ModifyDelete Kind = "modify-delete"
	Other        Kind = "other"
)

type Conflict struct {
	Path       []byte
	Kind       Kind
	Base       Stage
	Ours       Stage
	Theirs     Stage
	StatusXY   string
	Worktree   string
	Resolution string
}

func (c Conflict) Bytes() []byte { return append([]byte(nil), c.Path...) }

func ParseIndex(data []byte) ([]Conflict, error) {
	byPath := make(map[string]*Conflict)
	for len(data) > 0 {
		i := bytes.IndexByte(data, 0)
		if i < 0 {
			return nil, fmt.Errorf("unmerged index record is not NUL terminated")
		}
		record, rest := data[:i], data[i+1:]
		data = rest
		tab := bytes.IndexByte(record, '\t')
		if tab < 0 {
			return nil, fmt.Errorf("unmerged index record missing tab: %q", record)
		}
		fields := bytes.Fields(record[:tab])
		if len(fields) != 3 {
			return nil, fmt.Errorf("unmerged index record has %d fields", len(fields))
		}
		stage, err := strconv.Atoi(string(fields[2]))
		if err != nil || stage < 1 || stage > 3 {
			return nil, fmt.Errorf("invalid unmerged index stage %q", fields[2])
		}
		key := string(record[tab+1:])
		conflict := byPath[key]
		if conflict == nil {
			conflict = &Conflict{Path: append([]byte(nil), record[tab+1:]...)}
			byPath[key] = conflict
		}
		value := Stage{Mode: string(fields[0]), OID: string(fields[1])}
		switch stage {
		case 1:
			conflict.Base = value
		case 2:
			conflict.Ours = value
		case 3:
			conflict.Theirs = value
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make([]Conflict, 0, len(paths))
	for _, path := range paths {
		result = append(result, *byPath[path])
	}
	return result, nil
}

// Correlate adds status-derived working-tree information while retaining all
// index stages, including intentionally missing stages.
func Correlate(index []Conflict, statuses []Status) []Conflict {
	result := append([]Conflict(nil), index...)
	for i := range result {
		for _, status := range statuses {
			if bytes.Equal(result[i].Path, status.Path) {
				result[i].StatusXY, result[i].Worktree = status.XY, status.Worktree
				result[i].Kind = classify(status.XY, result[i])
				result[i].Resolution = resolution(result[i])
				break
			}
		}
	}
	return result
}

type Status struct {
	Path         []byte
	XY, Worktree string
}

func classify(xy string, c Conflict) Kind {
	switch xy {
	case "UU":
		return BothModified
	case "AA":
		return AddAdd
	case "UD", "DU", "UA", "AU", "DD":
		return ModifyDelete
	}
	if c.Base.OID == "" || c.Ours.OID == "" || c.Theirs.OID == "" {
		return ModifyDelete
	}
	return Other
}

func resolution(c Conflict) string {
	if c.StatusXY == "" {
		return "unmerged"
	}
	if c.StatusXY[0] == '.' && c.StatusXY[1] == '.' {
		return "resolved"
	}
	return "unmerged"
}
