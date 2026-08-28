# Task 109 — Parse colorized commit-tree output into safe segments

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Task 108

## Objective

Create a pure parser that converts Git's colored graph output into safe
presentation segments while preserving graph topology and text meaning.

## Requirements

Represent graph glyphs/spacing, commit hash, punctuation, ref decorations,
subject, relative date, author, and unknown/plain text in display order. Recognize
Git SGR and reset markers, map only known roles, and drop unknown SGR values.
Strip C0/C1 controls, OSC sequences, cursor controls, and malformed escapes from
repository-controlled text. Preserve Unicode, display width, topology spaces,
long text, and bounded truncation. Handle markers across lines, redundant or
missing resets, and incomplete output.

Do not infer fields from subject text. Field boundaries must come from the Git
format contract. Keep the parser independent of Bubble Tea and terminal I/O.

## Tests and acceptance

Add table-driven and fuzz/property tests for linear/merge/decorated histories,
Unicode and wide glyphs, all supported color markers, malformed/unknown escape
sequences, injection attempts in subjects/ref names/authors, truncation, and
display-width calculations. The parser must never panic, emit unbounded output,
or return unsafe terminal controls.

**Status:** Complete

## Completion summary

Added the pure `internal/ui/committree` parser. It converts Git SGR markers to
semantic hash, decoration, date, author, and plain segments while preserving
graph/text order and Unicode. OSC, CSI, C0/C1, unknown, and malformed control
sequences are removed. Table-driven and fuzz tests cover normal, hostile, and
malformed inputs without terminal I/O or Bubble Tea coupling.
