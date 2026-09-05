// Package tags models repository tags as bounded, selectable objects.
package tags

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/git"
)

const (
	defaultMaxTags      = 1024
	defaultMaxOutput    = 4 << 20
	maxTagNameBytes     = 1024
	maxTagMessageBytes  = 4096
	maxTaggerFieldBytes = 512
)

// Kind identifies whether a tag directly names its target or wraps an object.
type Kind string

const (
	Lightweight Kind = "lightweight"
	Annotated   Kind = "annotated"
)

// SignatureState is deliberately Unknown until a bounded, on-demand verify.
type SignatureState string

const (
	SignatureUnknown  SignatureState = "unknown"
	SignatureUnsigned SignatureState = "unsigned"
	SignatureValid    SignatureState = "valid"
	SignatureInvalid  SignatureState = "invalid"
)

// RemotePresence summarizes matching refs without claiming that signatures or
// remote connectivity have been verified.
type RemotePresence string

const (
	RemoteUnknown RemotePresence = "unknown"
	RemoteAbsent  RemotePresence = "absent"
	RemotePresent RemotePresence = "present"
)

// Tag is the bounded presentation/domain record for one refs/tags entry.
type Tag struct {
	Name           string
	ObjectID       string
	TargetID       string
	TargetKind     string
	Kind           Kind
	TaggerName     string
	TaggerEmail    string
	TaggedAt       time.Time
	Message        string
	Signature      SignatureState
	RemotePresence RemotePresence
	RemoteNames    []string
}

// Limits bounds both retained refs and command output.
type Limits struct {
	MaxTags        int
	MaxOutputBytes int
}

func (l Limits) normalized() Limits {
	if l.MaxTags <= 0 {
		l.MaxTags = defaultMaxTags
	}
	if l.MaxOutputBytes <= 0 {
		l.MaxOutputBytes = defaultMaxOutput
	}
	return l
}

// LoadRequest scopes one tag snapshot to a repository.
type LoadRequest struct {
	Repository string
	Limits     Limits
}

// Runner is the bounded command surface required by the tag loader.
type Runner interface {
	RunBounded(context.Context, int, ...string) (git.Result, error)
}

// Snapshot is an ordered, bounded tag view.
type Snapshot struct {
	Repository string
	Tags       []Tag
	Truncated  bool
}

var (
	ErrRepositoryRequired = errors.New("tag repository is required")
	ErrMalformedRecord    = errors.New("malformed tag ref record")
)

const tagFormat = "%(refname:short)%00%(objectname)%00%(objecttype)%00%(*objectname)%00%(*objecttype)%00%(taggername)%00%(taggeremail)%00%(taggerdate:iso8601)%00%(contents:subject)"
const remoteFormat = "%(refname:short)%00%(objectname)"

// Load uses NUL-delimited for-each-ref output and never verifies every tag.
// Signature verification belongs to a later, explicitly selected operation.
func Load(ctx context.Context, runner Runner, request LoadRequest) (Snapshot, error) {
	limits := request.Limits.normalized()
	if strings.TrimSpace(request.Repository) == "" {
		return Snapshot{}, ErrRepositoryRequired
	}
	result, err := runner.RunBounded(ctx, limits.MaxOutputBytes, "for-each-ref", "--sort=refname", "--format="+tagFormat, "refs/tags")
	if err != nil {
		return Snapshot{}, fmt.Errorf("load tags: %w", err)
	}
	tags, truncated, err := parseTagRefs(result.Stdout, limits.MaxTags)
	if err != nil {
		return Snapshot{}, err
	}
	remoteNames, err := remoteTagRefs(ctx, runner, limits.MaxOutputBytes)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load remote tag refs: %w", err)
	}
	for i := range tags {
		matching := remoteNames[tags[i].Name]
		if len(matching) == 0 {
			tags[i].RemotePresence = RemoteAbsent
			continue
		}
		tags[i].RemotePresence = RemotePresent
		tags[i].RemoteNames = append([]string(nil), matching...)
	}
	return Snapshot{Repository: request.Repository, Tags: tags, Truncated: truncated}, nil
}

func parseTagRefs(data []byte, maxTags int) ([]Tag, bool, error) {
	if maxTags <= 0 {
		maxTags = defaultMaxTags
	}
	fields := splitNULFields(data)
	tags := make([]Tag, 0, minInt(len(fields)/9, maxTags))
	truncated := false
	for offset := 0; offset < len(fields); offset += 9 {
		if len(tags) >= maxTags {
			truncated = true
			break
		}
		if offset+9 > len(fields) || fields[offset] == "" || fields[offset+1] == "" || fields[offset+2] == "" {
			return nil, false, ErrMalformedRecord
		}
		if len(fields[offset]) > maxTagNameBytes || len(fields[offset+5]) > maxTaggerFieldBytes || len(fields[offset+6]) > maxTaggerFieldBytes || len(fields[offset+8]) > maxTagMessageBytes {
			return nil, false, ErrMalformedRecord
		}
		kind := Lightweight
		targetID, targetKind := fields[offset+1], fields[offset+2]
		if fields[offset+2] == "tag" {
			kind = Annotated
			if fields[offset+3] != "" {
				targetID = fields[offset+3]
			}
			if fields[offset+4] != "" {
				targetKind = fields[offset+4]
			}
		}
		tag := Tag{Name: fields[offset], ObjectID: fields[offset+1], TargetID: targetID, TargetKind: targetKind, Kind: kind, TaggerName: fields[offset+5], TaggerEmail: fields[offset+6], Signature: SignatureUnknown, RemotePresence: RemoteUnknown, Message: fields[offset+8]}
		if fields[offset+7] != "" {
			parsed, err := time.Parse("2006-01-02 15:04:05 -0700", fields[offset+7])
			if err != nil {
				return nil, false, fmt.Errorf("parse tag %q date: %w", fields[offset], err)
			}
			tag.TaggedAt = parsed
		}
		tags = append(tags, tag)
	}
	if len(fields)/9 > maxTags {
		truncated = true
	}
	return tags, truncated, nil
}

func remoteTagRefs(ctx context.Context, runner Runner, maxOutput int) (map[string][]string, error) {
	result, err := runner.RunBounded(ctx, maxOutput, "for-each-ref", "--sort=refname", "--format="+remoteFormat, "refs/remotes")
	if err != nil {
		return nil, err
	}
	remoteTags := make(map[string][]string)
	fields := splitNULFields(result.Stdout)
	for offset := 0; offset+2 <= len(fields); offset += 2 {
		if !strings.Contains(fields[offset], "/tags/") {
			continue
		}
		parts := strings.SplitN(fields[offset], "/tags/", 2)
		if parts[1] == "" {
			continue
		}
		remote := parts[0]
		remoteTags[parts[1]] = append(remoteTags[parts[1]], remote)
	}
	return remoteTags, nil
}

func splitNULFields(data []byte) []string {
	trimmed := strings.TrimRight(string(data), "\x00\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\x00")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
