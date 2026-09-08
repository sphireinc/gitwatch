package blame

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type captureRunner struct {
	args   []string
	output []byte
	err    error
}

func (r *captureRunner) RunBounded(_ context.Context, _ int, args ...string) (git.Result, error) {
	r.args = append([]string(nil), args...)
	return git.Result{Stdout: r.output}, r.err
}

func TestLoadPageUsesBoundedLineRangeAndExactPath(t *testing.T) {
	runner := &captureRunner{output: []byte("abcdef0 1 4 1\nauthor Ada\nauthor-mail <ada@example.test>\nauthor-time 1700000000\nauthor-tz +0000\nfilename weird\tname.txt\n\tcontent\n")}
	page, err := LoadPage(context.Background(), runner, Request{Path: "-weird\tname.txt", Start: 4, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"blame", "--line-porcelain", "-L", "4,13", "--", "-weird\tname.txt"}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("args = %#v, want %#v", runner.args, want)
	}
	if len(page.Lines) != 1 || string(page.Lines[0].Content) != "content" || page.Lines[0].Filename != "weird\tname.txt" {
		t.Fatalf("page = %#v", page)
	}
}

func TestParsePreservesContentAndOriginMetadata(t *testing.T) {
	data := []byte("1234567890abcdef 7 8 1\nauthor Name\nauthor-mail <name@example.test>\nauthor-time 12\nauthor-tz -0500\nfilename old.txt\n\tcontrol\x1b[31mtext\n")
	lines := Parse(data)
	if len(lines) != 1 {
		t.Fatalf("lines = %#v", lines)
	}
	line := lines[0]
	if line.FinalSHA != "1234567890abcdef" || line.OriginalLine != 7 || line.FinalLine != 8 || line.Author != "Name" || line.AuthorTime != 12 || string(line.Content) != "control\x1b[31mtext" {
		t.Fatalf("line = %#v", line)
	}
}

func TestLoadPageAcceptsOutputLimitAsTruncatedPage(t *testing.T) {
	runner := &captureRunner{output: []byte("1234567 1 1\n\tone\n"), err: errors.Join(errors.New("wrapped"), git.ErrOutputLimit)}
	page, err := LoadPage(context.Background(), runner, Request{Path: "file", Start: 1, Limit: 1})
	if err != nil || !page.Truncated || !page.HasMore {
		t.Fatalf("page = %#v, err=%v", page, err)
	}
}

func TestLoadPageRejectsInvalidRequest(t *testing.T) {
	for _, request := range []Request{{Path: ""}, {Path: "file", Start: 0}, {Path: "file\x00name", Start: 1}} {
		if _, err := LoadPage(context.Background(), &captureRunner{}, request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("request %#v error = %v", request, err)
		}
	}
}
