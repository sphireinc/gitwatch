package git

import (
	"bytes"
	"fmt"
	"strings"
)

// Status is the parsed repository-wide porcelain-v2 status response.
type Status struct {
	BranchHead string
	BranchOID  string
	Upstream   string
	Ahead      int
	Behind     int
	Entries    []StatusEntry
}

// StatusEntry is one porcelain-v2 path record with byte-preserved paths.
type StatusEntry struct {
	Kind        byte
	XY          string
	Submodule   string
	ModeHead    string
	ModeIndex   string
	ModeWork    string
	OIDHead     string
	OIDIndex    string
	RenameScore string
	Path        []byte
	OrigPath    []byte
}

// ParseError identifies malformed machine-readable Git output.
type ParseError struct {
	Record []byte
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("porcelain v2 record: %s (%q)", e.Reason, e.Record)
}

// ParseStatus parses NUL-delimited porcelain-v2 status output.
func ParseStatus(data []byte) (Status, error) {
	var status Status
	for len(data) > 0 {
		i := bytes.IndexByte(data, 0)
		if i < 0 {
			return Status{}, &ParseError{Record: data, Reason: "record is not NUL terminated"}
		}
		record := data[:i]
		data = data[i+1:]
		if len(record) == 0 {
			continue
		}
		if record[0] == '#' {
			if err := parseHeader(&status, string(record)); err != nil {
				return Status{}, err
			}
			continue
		}
		entry, _, err := parseEntry(record)
		if err != nil {
			return Status{}, err
		}
		if entry.Kind == '2' {
			j := bytes.IndexByte(data, 0)
			if j < 0 {
				return Status{}, &ParseError{Record: record, Reason: "rename record missing NUL-delimited original path"}
			}
			entry.OrigPath = append([]byte(nil), data[:j]...)
			data = data[j+1:]
		}
		status.Entries = append(status.Entries, entry)
	}
	return status, nil
}

func parseHeader(status *Status, record string) error {
	fields := strings.SplitN(record, " ", 3)
	if len(fields) < 3 || fields[0] != "#" {
		return &ParseError{Record: []byte(record), Reason: "malformed header"}
	}
	switch fields[1] {
	case "branch.head":
		status.BranchHead = fields[2]
	case "branch.oid":
		status.BranchOID = fields[2]
	case "branch.upstream":
		status.Upstream = fields[2]
	case "branch.ab":
		parts := strings.Fields(fields[2])
		if len(parts) != 2 || len(parts[0]) < 2 || len(parts[1]) < 2 || parts[0][0] != '+' || parts[1][0] != '-' {
			return &ParseError{Record: []byte(record), Reason: "malformed branch divergence"}
		}
		var err error
		if _, err = fmt.Sscanf(parts[0], "+%d", &status.Ahead); err != nil {
			return &ParseError{Record: []byte(record), Reason: "invalid ahead count"}
		}
		if _, err = fmt.Sscanf(parts[1], "-%d", &status.Behind); err != nil {
			return &ParseError{Record: []byte(record), Reason: "invalid behind count"}
		}
	}
	return nil
}

func parseEntry(record []byte) (StatusEntry, []byte, error) {
	if len(record) < 3 {
		return StatusEntry{}, nil, &ParseError{Record: record, Reason: "record too short"}
	}
	kind := record[0]
	if kind != '1' && kind != '2' && kind != 'u' && kind != '?' && kind != '!' {
		return StatusEntry{}, nil, &ParseError{Record: record, Reason: "unknown record type"}
	}
	if kind == '?' || kind == '!' {
		if len(record) < 3 || record[1] != ' ' {
			return StatusEntry{}, nil, &ParseError{Record: record, Reason: "malformed untracked/ignored record"}
		}
		return StatusEntry{Kind: kind, Path: append([]byte(nil), record[2:]...)}, nil, nil
	}
	needed := 9
	switch kind {
	case '2':
		needed = 10
	case 'u':
		// Unmerged porcelain-v2 records contain four modes, three object IDs,
		// and the path: `u xy sub m1 m2 m3 mW h1 h2 h3 path`.
		needed = 11
	}
	fields := bytes.SplitN(record, []byte(" "), needed)
	if len(fields) < needed {
		return StatusEntry{}, nil, &ParseError{Record: record, Reason: "missing fields"}
	}
	e := StatusEntry{Kind: kind, XY: string(fields[1]), Submodule: string(fields[2]), ModeHead: string(fields[3]), ModeIndex: string(fields[4]), ModeWork: string(fields[5]), OIDHead: string(fields[6]), OIDIndex: string(fields[7]), Path: append([]byte(nil), fields[needed-1]...)}
	if kind == 'u' {
		e.OIDHead = string(fields[7])
		e.OIDIndex = string(fields[8])
		e.Path = append([]byte(nil), fields[10]...)
	}
	if kind == '2' {
		e.RenameScore = string(fields[8])
		e.Path = append([]byte(nil), fields[9]...)
		return e, nil, nil
	}
	return e, nil, nil
}
