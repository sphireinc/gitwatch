package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/sphireinc/git-watch/internal/conflicts"
)

const DefaultConflictContentLimit = 4 << 20

type Content struct {
	OID         string
	Bytes       []byte
	Size        int
	Binary      bool
	InvalidUTF8 bool
	Truncated   bool
	Missing     bool
	Hash        [32]byte
	Regions     int
}

type ConflictContent struct {
	Path   []byte
	Base   Content
	Ours   Content
	Theirs Content
	Result Content
}

// LoadConflictContent loads selected conflict stages by object ID and reads
// the current result with the same bounded policy. It never parses marker text.
func LoadConflictContent(ctx context.Context, runner Runner, conflict conflicts.Conflict, limit int) (ConflictContent, error) {
	if limit <= 0 {
		limit = DefaultConflictContentLimit
	}
	load := func(stage conflicts.Stage) (Content, error) {
		if stage.OID == "" {
			return Content{Missing: true}, nil
		}
		if !validObjectID(stage.OID) {
			return Content{}, fmt.Errorf("invalid conflict object ID %q", stage.OID)
		}
		result, err := runner.RunBounded(ctx, limit, "cat-file", "blob", stage.OID)
		content := Content{OID: stage.OID, Bytes: append([]byte(nil), result.Stdout...), Size: len(result.Stdout)}
		if err != nil {
			if result.ExitCode == 0 {
				content.Truncated = true
				return content, nil
			}
			return Content{}, err
		}
		return classifyContent(content), nil
	}
	base, err := load(conflict.Base)
	if err != nil {
		return ConflictContent{}, err
	}
	ours, err := load(conflict.Ours)
	if err != nil {
		return ConflictContent{}, err
	}
	theirs, err := load(conflict.Theirs)
	if err != nil {
		return ConflictContent{}, err
	}
	result := Content{Missing: true}
	path := filepath.Join(runner.Dir, filepath.FromSlash(string(conflict.Path)))
	root, err := filepath.Abs(runner.Dir)
	if err != nil {
		return ConflictContent{}, err
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return ConflictContent{}, err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ConflictContent{}, fmt.Errorf("conflict path escapes repository root")
	}
	if file, readErr := os.Open(path); readErr == nil {
		data, readDataErr := io.ReadAll(io.LimitReader(file, int64(limit)+1))
		closeErr := file.Close()
		if readDataErr != nil {
			return ConflictContent{}, readDataErr
		}
		if closeErr != nil {
			return ConflictContent{}, closeErr
		}
		if len(data) > limit {
			result = classifyContent(Content{Bytes: append([]byte(nil), data[:limit]...), Size: len(data)})
			result.Truncated = true
		} else {
			result = classifyContent(Content{Bytes: data, Size: len(data)})
		}
		if !result.Binary && !result.InvalidUTF8 && !result.Truncated {
			if document, parseErr := conflicts.ParseRegions(data, 1024); parseErr == nil {
				result.Regions = len(document.Regions)
			}
		}
	} else if !os.IsNotExist(readErr) {
		return ConflictContent{}, readErr
	}
	return ConflictContent{Path: append([]byte(nil), conflict.Path...), Base: base, Ours: ours, Theirs: theirs, Result: result}, nil
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	decoded := make([]byte, len(value)/2)
	_, err := hex.Decode(decoded, []byte(value))
	return err == nil
}

func classifyContent(content Content) Content {
	content.Hash = sha256.Sum256(content.Bytes)
	content.Binary = false
	for _, b := range content.Bytes {
		if b == 0 {
			content.Binary = true
			break
		}
	}
	content.InvalidUTF8 = !content.Binary && !utf8.Valid(content.Bytes)
	return content
}
