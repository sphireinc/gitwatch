package plugins

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Entry struct {
	Manifest Manifest
	Path     string
	Enabled  bool
	Healthy  bool
	Error    string
}

func Discover(ctx context.Context, directories []string, max int) ([]Entry, error) {
	if max <= 0 {
		max = 128
	}
	entries := make([]Entry, 0)
	for _, directory := range directories {
		err := filepath.WalkDir(directory, func(path string, item fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if len(entries) >= max {
				return filepath.SkipAll
			}
			if item.Type()&os.ModeSymlink != 0 {
				if item.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if item.IsDir() || item.Name() != "manifest.json" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			manifest, err := DecodeManifest(data)
			entry := Entry{Path: filepath.Dir(path), Enabled: true, Healthy: err == nil}
			if err != nil {
				entry.Error = err.Error()
			} else {
				entry.Manifest = manifest
			}
			entries = append(entries, entry)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func SetEnabled(entries []Entry, id string, enabled bool) []Entry {
	updated := append([]Entry(nil), entries...)
	for i := range updated {
		if updated[i].Manifest.ID == id {
			updated[i].Enabled = enabled
		}
	}
	return updated
}

func ValidateEntry(entry Entry) error {
	if entry.Manifest.ID == "" {
		return errors.New("plugin manifest is unavailable")
	}
	return entry.Manifest.Validate()
}
