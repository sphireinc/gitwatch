// Package security contains filesystem and display boundaries for gitignore
// data. It treats repository documents and upstream catalog data as untrusted.
package security

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

const DefaultMaxDocumentBytes int64 = 8 << 20

var (
	ErrUnsafeTarget = errors.New("gitignore target is outside the repository root")
	ErrTooLarge     = errors.New("gitignore file exceeds the interactive size limit")
	ErrBinary       = errors.New("gitignore file is binary or contains invalid text")
)

// Target returns the only supported gitignore target after proving it remains
// inside root. The feature never accepts a caller-provided relative path.
func Target(root string) (string, error) {
	if root == "" {
		return "", ErrUnsafeTarget
	}
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", ErrUnsafeTarget
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", ErrUnsafeTarget
	}
	target := filepath.Join(root, ".gitignore")
	rel, err := filepath.Rel(root, target)
	if err != nil || rel != ".gitignore" {
		return "", ErrUnsafeTarget
	}
	return target, nil
}

// ReadDocument reads a regular, non-symlink text document within root with a
// bounded allocation. A missing file is returned as an empty document.
func ReadDocument(root string, maxBytes int64) ([]byte, bool, error) {
	path, err := Target(root)
	if err != nil {
		return nil, false, err
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDocumentBytes
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, true, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, false, ErrUnsafeTarget
	}
	if info.Size() > maxBytes {
		return nil, false, ErrTooLarge
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	data, err := readBounded(file, maxBytes)
	closeErr := file.Close()
	if err != nil {
		return nil, false, err
	}
	if closeErr != nil {
		return nil, false, closeErr
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return nil, false, ErrBinary
	}
	return data, false, nil
}

func readBounded(file *os.File, maxBytes int64) ([]byte, error) {
	data := make([]byte, 0, minInt64(maxBytes, 64<<10))
	buffer := make([]byte, 32<<10)
	for {
		count, err := file.Read(buffer)
		data = append(data, buffer[:count]...)
		if int64(len(data)) > maxBytes {
			return nil, ErrTooLarge
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return data, nil
			}
			return nil, err
		}
	}
}

func minInt64(a, b int64) int {
	if a < b {
		return int(a)
	}
	return int(b)
}
