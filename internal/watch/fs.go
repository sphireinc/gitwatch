package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Mode string

const (
	ModeFS   Mode = "fs"
	ModePoll Mode = "poll"
)

type Event struct {
	At   time.Time
	Path string
	Mode Mode
	Err  error
}

type Watcher struct {
	root     string
	debounce time.Duration
	fs       *fsnotify.Watcher
	seen     map[string]struct{}
	mu       sync.Mutex
}

func New(root string, debounce time.Duration) (*Watcher, error) {
	if debounce <= 0 {
		debounce = 75 * time.Millisecond
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{root: root, debounce: debounce, fs: fw, seen: make(map[string]struct{})}
	if err := w.addTree(root); err != nil {
		_ = fw.Close()
		return nil, err
	}
	return w, nil
}

func (w *Watcher) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root && filepath.Base(path) == ".git" {
			if err := w.fs.Add(path); err != nil {
				return err
			}
			w.seen[path] = struct{}{}
			return filepath.SkipDir
		}
		if entry.IsDir() && !w.skip(path) {
			if err := w.fs.Add(path); err != nil {
				return err
			}
			w.seen[path] = struct{}{}
		}
		return nil
	})
}

func (w *Watcher) skip(path string) bool {
	clean := filepath.Clean(path)
	return clean != w.root && filepath.Base(clean) == ".git"
}

func (w *Watcher) Events(ctx context.Context) <-chan Event {
	out := make(chan Event, 8)
	go func() {
		defer close(out)
		timer := time.NewTimer(time.Hour)
		if !timer.Stop() {
			<-timer.C
		}
		pending := false
		var last string
		flush := func() {
			if !pending {
				return
			}
			pending = false
			select {
			case out <- Event{At: time.Now(), Path: last, Mode: ModeFS}:
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
				last = event.Name
				pending = true
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() && !w.skip(event.Name) {
						_ = w.fs.Add(event.Name)
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
