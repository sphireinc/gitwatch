// Package rebase parses and transforms Git interactive-rebase todo plans.
// Plans are data only: this package never invokes Git or edits files.
package rebase

import (
	"errors"
	"fmt"
	"strings"
)

const maxPlanBytes = 4 << 20

// Action is a Git interactive-rebase commit action.
type Action string

const (
	Pick   Action = "pick"
	Reword Action = "reword"
	Edit   Action = "edit"
	Squash Action = "squash"
	Fixup  Action = "fixup"
	Drop   Action = "drop"
)

func (a Action) valid() bool {
	switch a {
	case Pick, Reword, Edit, Squash, Fixup, Drop:
		return true
	default:
		return false
	}
}

// EntryKind identifies a parsed todo record.
type EntryKind uint8

const (
	CommitEntry EntryKind = iota
	CommentEntry
	BlankEntry
	DirectiveEntry
)

// Entry is an immutable view of one todo record. Use Plan methods for copies.
type Entry struct {
	kind          EntryKind
	action        Action
	sha           string
	subject       string
	raw           string
	originalIndex int
	changed       bool
}

func (e Entry) Kind() EntryKind    { return e.kind }
func (e Entry) Action() Action     { return e.action }
func (e Entry) SHA() string        { return e.sha }
func (e Entry) Subject() string    { return e.subject }
func (e Entry) Raw() string        { return e.raw }
func (e Entry) OriginalIndex() int { return e.originalIndex }

// Plan is an immutable parsed todo plan.
type Plan struct {
	entries         []Entry
	trailingNewline bool
}

// Entries returns a copy of the plan's records.
func (p Plan) Entries() []Entry { return append([]Entry(nil), p.entries...) }

// ParseError identifies a malformed editable todo record.
type ParseError struct {
	Line         int
	Text, Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("rebase todo line %d: %s: %q", e.Line, e.Reason, e.Text)
}

// Parse converts Git todo text into an immutable plan and retains unknown lines.
func Parse(input string) (Plan, error) {
	if len(input) > maxPlanBytes {
		return Plan{}, fmt.Errorf("rebase todo exceeds %d-byte limit", maxPlanBytes)
	}
	plan := Plan{trailingNewline: strings.HasSuffix(input, "\n")}
	if input == "" {
		return plan, nil
	}
	lines := strings.Split(input, "\n")
	if plan.trailingNewline {
		lines = lines[:len(lines)-1]
	}
	for index, line := range lines {
		entry, err := parseEntry(line, index)
		if err != nil {
			return Plan{}, err
		}
		plan.entries = append(plan.entries, entry)
	}
	return plan, nil
}

func parseEntry(line string, index int) (Entry, error) {
	entry := Entry{raw: line, originalIndex: index}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		entry.kind = BlankEntry
		return entry, nil
	}
	if strings.HasPrefix(trimmed, "#") {
		entry.kind = CommentEntry
		return entry, nil
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		entry.kind = BlankEntry
		return entry, nil
	}
	action := Action(fields[0])
	if !action.valid() {
		entry.kind = DirectiveEntry
		return entry, nil
	}
	if len(fields) < 2 {
		return Entry{}, &ParseError{Line: index + 1, Text: line, Reason: "commit action has no object name"}
	}
	position := strings.Index(line, fields[1]) + len(fields[1])
	entry.kind, entry.action, entry.sha, entry.subject = CommitEntry, action, fields[1], strings.TrimLeft(line[position:], " \t")
	return entry, nil
}

// Render preserves unchanged records, comments, blanks, directives, and the
// original trailing-newline choice. Changed commits use canonical todo syntax.
func (p Plan) Render() string {
	if len(p.entries) == 0 {
		return ""
	}
	var builder strings.Builder
	for index, entry := range p.entries {
		if index > 0 {
			builder.WriteByte('\n')
		}
		if entry.kind == CommitEntry && entry.changed {
			builder.WriteString(string(entry.action))
			builder.WriteByte(' ')
			builder.WriteString(entry.sha)
			if entry.subject != "" {
				builder.WriteByte(' ')
				builder.WriteString(entry.subject)
			}
		} else {
			builder.WriteString(entry.raw)
		}
	}
	if p.trailingNewline {
		builder.WriteByte('\n')
	}
	return builder.String()
}

// Validate rejects transformations Git cannot execute safely.
func (p Plan) Validate() error {
	commits := 0
	for index, entry := range p.entries {
		if entry.kind != CommitEntry {
			continue
		}
		if entry.sha == "" || !entry.action.valid() {
			return fmt.Errorf("invalid commit entry at plan index %d", index)
		}
		if commits == 0 && (entry.action == Squash || entry.action == Fixup) {
			return errors.New("first commit entry cannot use squash or fixup")
		}
		commits++
	}
	return nil
}

// ChangeAction returns a copy with one commit action changed.
func (p Plan) ChangeAction(index int, action Action) (Plan, error) {
	if !action.valid() {
		return Plan{}, fmt.Errorf("unsupported rebase action %q", action)
	}
	if index < 0 || index >= len(p.entries) || p.entries[index].kind != CommitEntry {
		return Plan{}, fmt.Errorf("plan index %d is not an editable commit", index)
	}
	result := p.clone()
	result.entries[index].action = action
	result.entries[index].changed = action != p.entries[index].action
	if err := result.Validate(); err != nil {
		return Plan{}, err
	}
	return result, nil
}

// MoveEntry returns a copy with one record moved to destination.
func (p Plan) MoveEntry(from, to int) (Plan, error) { return p.MoveRange(from, from, to) }

// MoveRange returns a copy with the inclusive record range moved before the
// destination index. Every record, including unknown directives, is retained.
func (p Plan) MoveRange(start, end, destination int) (Plan, error) {
	if start < 0 || end < start || end >= len(p.entries) || destination < 0 || destination > len(p.entries) {
		return Plan{}, errors.New("rebase move range is out of bounds")
	}
	if destination >= start && destination <= end+1 {
		return Plan{}, errors.New("rebase move destination overlaps selected range")
	}
	result := p.clone()
	selected := append([]Entry(nil), result.entries[start:end+1]...)
	remaining := append([]Entry(nil), result.entries[:start]...)
	remaining = append(remaining, result.entries[end+1:]...)
	if destination > end {
		destination -= end - start + 1
	}
	if destination > len(remaining) {
		return Plan{}, errors.New("rebase move destination is out of bounds")
	}
	result.entries = append(append(append([]Entry(nil), remaining[:destination]...), selected...), remaining[destination:]...)
	if err := result.Validate(); err != nil {
		return Plan{}, err
	}
	return result, nil
}

// SquashTarget returns the preceding commit target, ignoring intervening
// comments and directives.
func (p Plan) SquashTarget(index int) (int, error) {
	if index < 0 || index >= len(p.entries) || p.entries[index].kind != CommitEntry {
		return -1, fmt.Errorf("plan index %d is not an editable commit", index)
	}
	for index--; index >= 0; index-- {
		if p.entries[index].kind == CommitEntry {
			return index, nil
		}
	}
	return -1, errors.New("first commit entry has no squash/fixup target")
}

// LogicalGroups groups each commit with following squash/fixup commits.
func (p Plan) LogicalGroups() [][]Entry {
	groups := make([][]Entry, 0)
	currentCommitGroup := -1
	for _, entry := range p.entries {
		if entry.kind == CommitEntry {
			if (entry.action == Squash || entry.action == Fixup) && currentCommitGroup >= 0 {
				groups[currentCommitGroup] = append(groups[currentCommitGroup], entry)
				continue
			}
			groups = append(groups, []Entry{entry})
			currentCommitGroup = len(groups) - 1
			continue
		}
		if currentCommitGroup >= 0 {
			groups[currentCommitGroup] = append(groups[currentCommitGroup], entry)
			continue
		}
		groups = append(groups, []Entry{entry})
	}
	return groups
}

func (p Plan) clone() Plan {
	return Plan{entries: append([]Entry(nil), p.entries...), trailingNewline: p.trailingNewline}
}
