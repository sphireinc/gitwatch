package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// MaterializedFile is a private, bounded temporary file for an external tool.
// Cleanup is idempotent and removes the private directory containing it.
type MaterializedFile struct {
	Path    string
	cleanup func()
}

// MaterializeFile writes content into a private 0700 directory and 0600 file.
// It refuses content larger than maxBytes so an external handoff cannot turn
// a bounded Git read into an unbounded retained copy.
func MaterializeFile(name string, content []byte, maxBytes int) (MaterializedFile, error) {
	if name == "" {
		name = "content"
	}
	if maxBytes <= 0 || len(content) > maxBytes {
		return MaterializedFile{}, fmt.Errorf("materialized file exceeds configured size limit")
	}
	directory, err := os.MkdirTemp("", ".gitwatch-tool-")
	if err != nil {
		return MaterializedFile{}, err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		_ = os.RemoveAll(directory)
		return MaterializedFile{}, err
	}
	path := filepath.Join(directory, filepath.Base(name))
	if err := os.WriteFile(path, content, 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return MaterializedFile{}, err
	}
	return MaterializedFile{Path: path, cleanup: func() { _ = os.RemoveAll(directory) }}, nil
}

// Cleanup removes the materialized file and its private directory.
func (f *MaterializedFile) Cleanup() {
	if f == nil || f.cleanup == nil {
		return
	}
	f.cleanup()
	f.cleanup = nil
}
