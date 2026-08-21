package diff

import (
	"context"
	"sync"

	"github.com/sphireinc/git-watch/internal/git"
)

type Mode uint8

const (
	Unstaged Mode = iota
	Staged
)

type Result struct {
	Path   []byte
	Mode   Mode
	Text   []byte
	Binary bool
	Err    error
}
type Viewer struct {
	mu      sync.Mutex
	Path    []byte
	Mode    Mode
	Loading bool
	Result  Result
	cancel  context.CancelFunc
}

func (v *Viewer) Open(ctx context.Context, runner git.Runner, path []byte, mode Mode) {
	v.mu.Lock()
	if v.cancel != nil {
		v.cancel()
	}
	loadCtx, cancel := context.WithCancel(ctx)
	v.cancel = cancel
	v.Path = append([]byte(nil), path...)
	v.Mode = mode
	v.Loading = true
	v.Result = Result{Path: append([]byte(nil), path...), Mode: mode}
	v.mu.Unlock()
	go func() {
		d, err := runner.Diff(loadCtx, path, mode == Staged)
		v.mu.Lock()
		defer v.mu.Unlock()
		if loadCtx.Err() != nil {
			return
		}
		v.Loading = false
		v.Result = Result{Path: d.Path, Mode: mode, Text: d.Text, Binary: d.Binary, Err: err}
	}()
}
func (v *Viewer) Close() {
	v.mu.Lock()
	if v.cancel != nil {
		v.cancel()
	}
	v.Loading = false
	v.mu.Unlock()
}
func (v *Viewer) Snapshot() (Result, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.Result, v.Loading
}
