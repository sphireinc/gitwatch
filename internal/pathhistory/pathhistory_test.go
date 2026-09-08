package pathhistory

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type captureRunner struct {
	args []string
	data []byte
}

func (r *captureRunner) RunBounded(_ context.Context, _ int, args ...string) (git.Result, error) {
	r.args = append([]string(nil), args...)
	return git.Result{Args: args, Stdout: r.data}, nil
}

func TestLoadPageUsesExplicitPathAndFollowAndParsesRename(t *testing.T) {
	runner := &captureRunner{data: []byte("\x1eabc\x00abc\x00Author\x001700000000\x00rename\nR100\x00old name\x00new\tname\x00")}
	page, err := LoadPage(context.Background(), runner, Request{Path: "-leading\tname", Follow: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.Entries[0].Kind != "R100" || page.Entries[0].OldPath != "old name" || page.Entries[0].NewPath != "new\tname" {
		t.Fatalf("path history = %#v", page.Entries)
	}
	want := []string{"log", "--skip=0", "-n", "11", "--format=%x1e%H%x00%h%x00%an%x00%at%x00%s\n", "--follow", "--name-status", "-z", "--", "-leading\tname"}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("path history argv = %#v, want %#v", runner.args, want)
	}
}

func TestLoadPageRejectsUnsafePathAndClampsLimit(t *testing.T) {
	runner := &captureRunner{}
	if _, err := LoadPage(context.Background(), runner, Request{Path: "bad\x00path"}); err != ErrInvalidPath {
		t.Fatalf("invalid path error = %v", err)
	}
	if _, err := LoadPage(context.Background(), runner, Request{Path: "file", Limit: MaxLimit + 1}); err != nil {
		t.Fatal(err)
	}
	if runner.args[2] != "-n" || runner.args[3] != strconv.Itoa(MaxLimit+1) {
		t.Fatalf("clamped args = %#v", runner.args)
	}
}
