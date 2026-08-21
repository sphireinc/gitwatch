package watch

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"time"
)

type RequestedMode string

const (
	RequestedAuto RequestedMode = "auto"
	RequestedFS   RequestedMode = "fs"
	RequestedPoll RequestedMode = "poll"
)

func ParseMode(value string) (RequestedMode, bool) {
	switch RequestedMode(value) {
	case RequestedAuto, RequestedFS, RequestedPoll:
		return RequestedMode(value), true
	default:
		return RequestedAuto, false
	}
}

type Poller struct {
	root     string
	metadata []string
	interval time.Duration
}

func NewPoller(root string, interval time.Duration) Poller {
	return NewPollerWithMetadata(root, nil, interval)
}

// NewPollerWithMetadata includes Git directories outside the worktree in its
// bounded change signature.
func NewPollerWithMetadata(root string, metadata []string, interval time.Duration) Poller {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return Poller{root: filepath.Clean(root), metadata: uniquePaths(metadata), interval: interval}
}

func (p Poller) Events(ctx context.Context) <-chan Event {
	out := make(chan Event, 1)
	go func() {
		defer close(out)
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		emit := func(event Event) bool {
			select {
			case out <- event:
				return true
			case <-ctx.Done():
				return false
			}
		}
		_, err := p.signature()
		if err != nil && !emit(Event{At: time.Now(), Path: p.root, Mode: ModePoll, Err: err}) {
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := p.signature()
				if err != nil {
					if !emit(Event{At: time.Now(), Path: p.root, Mode: ModePoll, Err: err}) {
						return
					}
					continue
				}
				if !emit(Event{At: time.Now(), Path: p.root, Mode: ModePoll}) {
					return
				}
			}
		}
	}()
	return out
}

func (p Poller) signature() (string, error) {
	hash := fnv.New64a()
	var count int
	add := func(path string, info os.FileInfo) {
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00%d\x00%d\x00", path, info.Mode(), info.Size(), info.ModTime().UnixNano())
		count++
	}
	if err := walkBounded(p.root, 10000, func(path string, info os.FileInfo) error {
		if info.IsDir() && path != p.root && filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}
		add(path, info)
		return nil
	}); err != nil {
		return "", err
	}
	for _, directory := range p.metadata {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(directory)
		if err != nil {
			return "", err
		}
		add(directory, info)
		for _, entry := range entries {
			path := filepath.Join(directory, entry.Name())
			entryInfo, infoErr := entry.Info()
			if infoErr != nil {
				return "", infoErr
			}
			add(path, entryInfo)
		}
		refs := filepath.Join(directory, "refs")
		if info, statErr := os.Stat(refs); statErr == nil && info.IsDir() {
			if err := walkBounded(refs, 10000, func(path string, info os.FileInfo) error {
				if path != refs {
					add(path, info)
				}
				return nil
			}); err != nil {
				return "", err
			}
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return "", statErr
		}
	}
	return fmt.Sprintf("%x:%d", hash.Sum64(), count), nil
}

func walkBounded(root string, limit int, visit func(string, os.FileInfo) error) error {
	seen := 0
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return os.ErrInvalid
		}
		if seen >= limit {
			return filepath.SkipAll
		}
		seen++
		return visit(path, info)
	})
}

func SelectMode(requested RequestedMode, fsAvailable bool) (Mode, error) {
	if requested == RequestedPoll {
		return ModePoll, nil
	}
	if requested == RequestedFS && !fsAvailable {
		return "", os.ErrNotExist
	}
	if fsAvailable {
		return ModeFS, nil
	}
	return ModePoll, nil
}
