package watch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Mode string

const (
	ModeFS   Mode = "fs"
	ModePoll Mode = "poll"
)

type Event struct {
	At        time.Time
	Path      string
	Mode      Mode
	Operation string
	Err       error
}

type Watcher struct {
	root     string
	debounce time.Duration
	fs       *fsnotify.Watcher
	watched  map[string]struct{}
	metadata map[string]struct{}
}

func New(root string, debounce time.Duration) (*Watcher, error) {
	return NewWithMetadata(root, nil, debounce)
}

// NewWithMetadata watches the worktree plus the Git directories that may live
// outside it for linked worktrees and submodules.
func NewWithMetadata(root string, metadata []string, debounce time.Duration) (*Watcher, error) {
	if debounce <= 0 {
		debounce = 75 * time.Millisecond
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		root:     filepath.Clean(root),
		debounce: debounce,
		fs:       fw,
		watched:  make(map[string]struct{}),
		metadata: make(map[string]struct{}),
	}
	if err := w.addTree(w.root); err != nil {
		return nil, errors.Join(err, fw.Close())
	}
	for _, directory := range uniquePaths(metadata) {
		w.metadata[directory] = struct{}{}
		parent := filepath.Dir(directory)
		if parent != directory {
			if err := w.add(parent); err != nil {
				return nil, errors.Join(err, fw.Close())
			}
		}
		if err := w.add(directory); err != nil {
			return nil, errors.Join(err, fw.Close())
		}
		refs := filepath.Join(directory, "refs")
		if info, err := os.Stat(refs); err == nil && info.IsDir() {
			if err := w.addTree(refs); err != nil {
				return nil, errors.Join(err, fw.Close())
			}
		} else if err != nil && !os.IsNotExist(err) {
			return nil, errors.Join(err, fw.Close())
		}
	}
	return w, nil
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, value := range paths {
		if value == "" {
			continue
		}
		clean := filepath.Clean(value)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}
	return unique
}

func (w *Watcher) add(directory string) error {
	directory = filepath.Clean(directory)
	if _, ok := w.watched[directory]; ok {
		return nil
	}
	if err := w.fs.Add(directory); err != nil {
		return err
	}
	w.watched[directory] = struct{}{}
	return nil
}

func (w *Watcher) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root && filepath.Base(path) == ".git" {
			if err := w.add(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}
		if entry.IsDir() && !w.skip(path) {
			if err := w.add(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func (w *Watcher) skip(path string) bool {
	clean := filepath.Clean(path)
	return clean != w.root && filepath.Base(clean) == ".git"
}

func (w *Watcher) shouldWatchTree(path string) bool {
	_, metadata := w.metadata[filepath.Clean(path)]
	return metadata || !w.skip(path)
}

func (w *Watcher) isMetadataPath(path string) bool {
	clean := filepath.Clean(path)
	for directory := range w.metadata {
		if clean == directory || strings.HasPrefix(clean, directory+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (w *Watcher) Events(ctx context.Context) <-chan Event {
	out := make(chan Event, 8)
	go func() {
		defer close(out)
		timer := time.NewTimer(time.Hour)
		defer timer.Stop()
		if !timer.Stop() {
			<-timer.C
		}
		pending := false
		var last, lastOperation string
		flush := func() {
			if !pending {
				return
			}
			pending = false
			select {
			case out <- Event{At: time.Now(), Path: last, Mode: ModeFS, Operation: lastOperation}:
			case <-ctx.Done():
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				flush()
			case event, ok := <-w.fs.Events:
				if !ok {
					return
				}
				// On kqueue platforms, opening Git metadata such as the index can
				// surface as a CHMOD event even when no mode or content changed. A
				// status refresh necessarily reads that metadata, so forwarding the
				// read artifact would create an endless watcher/refresh loop. Real
				// metadata writes include WRITE/CREATE/REMOVE/RENAME and worktree
				// CHMOD events remain authoritative mode-change hints.
				if event.Op == fsnotify.Chmod && w.isMetadataPath(event.Name) {
					continue
				}
				last = event.Name
				lastOperation = event.Op.String()
				pending = true
				if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
					w.removeWatchedTree(event.Name)
				}
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() && w.shouldWatchTree(event.Name) {
						if err := w.addTree(event.Name); err != nil {
							select {
							case out <- Event{At: time.Now(), Path: event.Name, Mode: ModeFS, Operation: event.Op.String(), Err: err}:
							case <-ctx.Done():
								return
							}
						}
					}
				}
				timer.Reset(w.debounce)
			case err, ok := <-w.fs.Errors:
				if !ok {
					return
				}
				select {
				case out <- Event{At: time.Now(), Mode: ModeFS, Err: err}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

func (w *Watcher) removeWatchedTree(root string) {
	root = filepath.Clean(root)
	prefix := root + string(filepath.Separator)
	for watched := range w.watched {
		if watched == root || strings.HasPrefix(watched, prefix) {
			delete(w.watched, watched)
		}
	}
}

func (w *Watcher) Close() error { return w.fs.Close() }

func IsGitMetadata(path string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	for i, part := range parts {
		if part == ".git" && i == len(parts)-2 {
			return true
		}
	}
	return false
}
