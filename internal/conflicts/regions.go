package conflicts

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrStaleDocument = errors.New("conflict document changed externally")

type Choice uint8

const (
	ChoiceOurs Choice = iota
	ChoiceTheirs
	ChoiceBoth
	ChoiceManual
)

type Region struct {
	Start, End         int
	Ours, Base, Theirs []byte
}

type Document struct {
	Bytes   []byte
	Hash    [32]byte
	Regions []Region
}

// ParseRegions identifies conflict markers without treating them as Git
// identity. Index stages remain the source of truth for object identity.
func ParseRegions(data []byte, maxRegions int) (Document, error) {
	if maxRegions <= 0 {
		maxRegions = 1024
	}
	document := Document{Bytes: append([]byte(nil), data...), Hash: sha256.Sum256(data)}
	var active *Region
	var oursStart, baseStart, theirsStart int
	for offset := 0; offset < len(data); {
		lineEnd := bytes.IndexByte(data[offset:], '\n')
		if lineEnd < 0 {
			lineEnd = len(data)
		} else {
			lineEnd += offset + 1
		}
		line := data[offset:lineEnd]
		trimmed := bytes.TrimSuffix(line, []byte("\n"))
		trimmed = bytes.TrimSuffix(trimmed, []byte("\r"))
		switch {
		case active == nil && bytes.HasPrefix(trimmed, []byte("<<<<<<<")):
			if len(document.Regions) >= maxRegions {
				return Document{}, fmt.Errorf("conflict region limit exceeded")
			}
			active = &Region{Start: offset}
			oursStart = lineEnd
		case active != nil && bytes.HasPrefix(trimmed, []byte("|||||||")):
			active.Ours = append([]byte(nil), data[oursStart:offset]...)
			baseStart = lineEnd
		case active != nil && bytes.HasPrefix(trimmed, []byte("=======")):
			if baseStart > 0 {
				active.Base = append([]byte(nil), data[baseStart:offset]...)
			} else {
				active.Ours = append([]byte(nil), data[oursStart:offset]...)
			}
			theirsStart = lineEnd
		case active != nil && bytes.HasPrefix(trimmed, []byte(">>>>>>>")):
			active.Theirs = append([]byte(nil), data[theirsStart:offset]...)
			active.End = lineEnd
			document.Regions = append(document.Regions, *active)
			active = nil
			oursStart, baseStart, theirsStart = 0, 0, 0
		}
		offset = lineEnd
	}
	if active != nil {
		return Document{}, fmt.Errorf("unterminated conflict region")
	}
	return document, nil
}

func (d Document) Apply(index int, choice Choice, manual []byte, current []byte) ([]byte, error) {
	if sha256.Sum256(current) != d.Hash {
		return nil, ErrStaleDocument
	}
	if index < 0 || index >= len(d.Regions) {
		return nil, fmt.Errorf("conflict region %d is out of bounds", index)
	}
	region := d.Regions[index]
	var replacement []byte
	switch choice {
	case ChoiceOurs:
		replacement = region.Ours
	case ChoiceTheirs:
		replacement = region.Theirs
	case ChoiceBoth:
		replacement = append(append([]byte(nil), region.Ours...), region.Theirs...)
	case ChoiceManual:
		replacement = manual
	default:
		return nil, fmt.Errorf("unknown conflict choice %d", choice)
	}
	result := make([]byte, 0, len(current)-(region.End-region.Start)+len(replacement))
	result = append(result, current[:region.Start]...)
	result = append(result, replacement...)
	result = append(result, current[region.End:]...)
	return result, nil
}

// AtomicWrite replaces one conflict file while preserving its existing mode.
func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".gitwatch-conflict-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(mode.Perm()); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

type UndoStack struct {
	entries [][]byte
	limit   int
}

func NewUndoStack(limit int) *UndoStack {
	if limit <= 0 {
		limit = 32
	}
	return &UndoStack{limit: limit}
}
func (u *UndoStack) Push(data []byte) {
	u.entries = append(u.entries, append([]byte(nil), data...))
	if len(u.entries) > u.limit {
		u.entries = u.entries[len(u.entries)-u.limit:]
	}
}
func (u *UndoStack) Undo() ([]byte, bool) {
	if len(u.entries) == 0 {
		return nil, false
	}
	index := len(u.entries) - 1
	data := u.entries[index]
	u.entries = u.entries[:index]
	return append([]byte(nil), data...), true
}
