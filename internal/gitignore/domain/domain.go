// Package domain contains the repository-independent domain for gitignore
// management. It deliberately has no UI, filesystem, Git, or network imports.
package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type RepositoryID string

type TemplateCategory string

const (
	CategoryRoot      TemplateCategory = "root"
	CategoryGlobal    TemplateCategory = "global"
	CategoryCommunity TemplateCategory = "community"
)

func (c TemplateCategory) valid() bool {
	return c == CategoryRoot || c == CategoryGlobal || c == CategoryCommunity
}

// TemplateID is based on the upstream relative path, never a display name.
type TemplateID string

func NewTemplateID(category TemplateCategory, relativePath string) (TemplateID, error) {
	if !category.valid() {
		return "", fmt.Errorf("invalid template category %q", category)
	}
	if err := validateRelativePath(relativePath); err != nil {
		return "", err
	}
	return TemplateID(string(category) + "/" + relativePath), nil
}

func ParseTemplateID(value string) (TemplateID, error) {
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("template ID must be category/relative-path: %q", value)
	}
	return NewTemplateID(TemplateCategory(parts[0]), parts[1])
}

func (id TemplateID) String() string { return string(id) }

func (id TemplateID) Category() TemplateCategory {
	parts := strings.SplitN(string(id), "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return TemplateCategory(parts[0])
}

func (id TemplateID) RelativePath() string {
	parts := strings.SplitN(string(id), "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func (id TemplateID) MarshalJSON() ([]byte, error) { return json.Marshal(string(id)) }

func (id *TemplateID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseTemplateID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func validateRelativePath(value string) error {
	if value == "" || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "\\") {
		return fmt.Errorf("invalid template relative path %q", value)
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid template relative path %q", value)
		}
	}
	return nil
}

type Template struct {
	ID            TemplateID       `json:"id"`
	Name          string           `json:"name"`
	Category      TemplateCategory `json:"category"`
	SourcePath    string           `json:"source_path"`
	ContentSHA256 string           `json:"content_sha256"`
}

type MatchKind string

const (
	ManagedExact        MatchKind = "managed_exact"
	UnmanagedFull       MatchKind = "unmanaged_full"
	Partial             MatchKind = "partial"
	Absent              MatchKind = "absent"
	InvalidManagedBlock MatchKind = "invalid_managed_block"
)

func (k MatchKind) Full() bool { return k == ManagedExact || k == UnmanagedFull }

type ManagedBlock struct {
	TemplateID    TemplateID `json:"template_id"`
	Start         int        `json:"start"`
	End           int        `json:"end"`
	ContentSHA256 string     `json:"content_sha256"`
}

func (b ManagedBlock) Valid() bool {
	return b.TemplateID != "" && b.Start >= 0 && b.End >= b.Start && b.ContentSHA256 != ""
}

type TemplateMatch struct {
	TemplateID TemplateID    `json:"template_id"`
	Kind       MatchKind     `json:"kind"`
	Block      *ManagedBlock `json:"block,omitempty"`
	Warning    string        `json:"warning,omitempty"`
}

// Owned is intentionally true only for a valid managed block. A full match in
// user-authored content never grants gitwatch permission to remove it.
func (m TemplateMatch) Owned() bool {
	return m.Kind == ManagedExact && m.Block != nil && m.Block.Valid()
}

type NewlineStyle string

const (
	NewlineNone  NewlineStyle = "none"
	NewlineLF    NewlineStyle = "lf"
	NewlineCRLF  NewlineStyle = "crlf"
	NewlineMixed NewlineStyle = "mixed"
)

type DocumentSnapshot struct {
	Repository   RepositoryID `json:"repository"`
	Root         string       `json:"root"`
	Path         string       `json:"path"`
	Bytes        []byte       `json:"bytes"`
	SHA256       string       `json:"sha256"`
	Newline      NewlineStyle `json:"newline"`
	HasBOM       bool         `json:"has_bom"`
	FinalNewline bool         `json:"final_newline"`
	Permissions  uint32       `json:"permissions"`
}

func NewDocumentSnapshot(repository RepositoryID, root, path string, content []byte, permissions uint32) (DocumentSnapshot, error) {
	if repository == "" || root == "" || path == "" || !safePath(path) {
		return DocumentSnapshot{}, ErrUnsafeTarget
	}
	return DocumentSnapshot{Repository: repository, Root: root, Path: path, Bytes: append([]byte(nil), content...), SHA256: hash(content), Newline: newlineStyle(content), HasBOM: bytes.HasPrefix(content, []byte{0xef, 0xbb, 0xbf}), FinalNewline: len(content) > 0 && (content[len(content)-1] == '\n' || content[len(content)-1] == '\r'), Permissions: permissions}, nil
}

func (s DocumentSnapshot) Clone() DocumentSnapshot {
	s.Bytes = append([]byte(nil), s.Bytes...)
	return s
}

type MutationKind string

const (
	MutationCreate MutationKind = "create"
	MutationAppend MutationKind = "append"
	MutationRemove MutationKind = "remove"
	MutationUpdate MutationKind = "update"
)

type Edit struct {
	Start       int        `json:"start"`
	End         int        `json:"end"`
	Replacement []byte     `json:"replacement"`
	TemplateID  TemplateID `json:"template_id,omitempty"`
}

type MutationPlan struct {
	Repository   RepositoryID `json:"repository"`
	Root         string       `json:"root"`
	Path         string       `json:"path"`
	Kind         MutationKind `json:"kind"`
	BeforeSHA256 string       `json:"before_sha256"`
	BeforeBytes  []byte       `json:"before_bytes"`
	Newline      NewlineStyle `json:"newline"`
	Selected     []TemplateID `json:"selected"`
	Edits        []Edit       `json:"edits"`
	ResultBytes  []byte       `json:"result_bytes"`
	Warnings     []string     `json:"warnings,omitempty"`
}

func NewMutationPlan(snapshot DocumentSnapshot, kind MutationKind, selected []TemplateID, edits []Edit, result []byte, warnings []string) (MutationPlan, error) {
	if snapshot.Repository == "" || snapshot.Root == "" || snapshot.Path == "" || snapshot.SHA256 == "" || !safePath(snapshot.Path) {
		return MutationPlan{}, ErrUnsafeTarget
	}
	if kind == "" {
		return MutationPlan{}, errors.New("mutation kind is required")
	}
	p := MutationPlan{Repository: snapshot.Repository, Root: snapshot.Root, Path: snapshot.Path, Kind: kind, BeforeSHA256: snapshot.SHA256, BeforeBytes: append([]byte(nil), snapshot.Bytes...), Newline: snapshot.Newline, Selected: append([]TemplateID(nil), selected...), Edits: cloneEdits(edits), ResultBytes: append([]byte(nil), result...), Warnings: append([]string(nil), warnings...)}
	return p, nil
}

func (p MutationPlan) Clone() MutationPlan {
	p.BeforeBytes = append([]byte(nil), p.BeforeBytes...)
	p.Selected = append([]TemplateID(nil), p.Selected...)
	p.Edits = cloneEdits(p.Edits)
	p.ResultBytes = append([]byte(nil), p.ResultBytes...)
	p.Warnings = append([]string(nil), p.Warnings...)
	return p
}

func cloneEdits(in []Edit) []Edit {
	out := append([]Edit(nil), in...)
	for i := range out {
		out[i].Replacement = append([]byte(nil), out[i].Replacement...)
	}
	return out
}

type RepositoryGitignoreState struct {
	Repository RepositoryID     `json:"repository"`
	Root       string           `json:"root"`
	Snapshot   DocumentSnapshot `json:"snapshot"`
	Matches    []TemplateMatch  `json:"matches"`
	ReadOnly   bool             `json:"read_only"`
}

var (
	ErrConcurrentModification    = errors.New("gitignore changed after preview")
	ErrUnsafeTarget              = errors.New("unsafe or symlink gitignore target")
	ErrMalformedManagedBlock     = errors.New("malformed gitwatch managed block")
	ErrUnknownTemplate           = errors.New("unknown gitignore template")
	ErrCatalogUnavailable        = errors.New("gitignore catalog unavailable")
	ErrReadOnlyRepository        = errors.New("repository is read-only")
	ErrAmbiguousUnmanagedRemoval = errors.New("unmanaged template removal is ambiguous")
)

func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

func safePath(value string) bool {
	return value == ".gitignore" || (value != "" && !strings.HasPrefix(value, "/") && !strings.Contains(value, "\\") && !strings.Contains(value, "../") && !strings.Contains(value, "/.."))
}

func newlineStyle(value []byte) NewlineStyle {
	lf := bytes.Contains(value, []byte{'\n'})
	crlf := bytes.Contains(value, []byte{'\r', '\n'})
	if !lf {
		return NewlineNone
	}
	if crlf && bytes.Count(value, []byte{'\r', '\n'}) == bytes.Count(value, []byte{'\n'}) {
		return NewlineCRLF
	}
	if !crlf {
		return NewlineLF
	}
	return NewlineMixed
}
