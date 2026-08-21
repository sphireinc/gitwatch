package watch

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
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
	interval time.Duration
}

func NewPoller(root string, interval time.Duration) Poller {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return Poller{root: root, interval: interval}
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
		previous, err := p.signature()
		if err != nil && !emit(Event{At: time.Now(), Path: p.root, Mode: ModePoll, Err: err}) {
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				next, err := p.signature()
				if err != nil {
					if !emit(Event{At: time.Now(), Path: p.root, Mode: ModePoll, Err: err}) {
						return
					}
					continue
				}
				if next != previous {
					previous = next
					if !emit(Event{At: time.Now(), Path: p.root, Mode: ModePoll}) {
						return
					}
				}
			}
		}
	}()
	return out
}

func (p Poller) signature() (string, error) {
	var newest int64
	var count int
	err := filepath.Walk(p.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return os.ErrInvalid
		}
		if info.ModTime().UnixNano() > newest {
			newest = info.ModTime().UnixNano()
		}
		count++
		if count > 10000 {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return time.Unix(0, newest).UTC().String() + ":" + strconv.Itoa(count), nil
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
