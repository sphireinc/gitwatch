package reflog

import (
	"context"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestLoadParsesBoundedReflogRecords(t *testing.T) {
	root := t.TempDir()
	runner := git.NewRunner(root)
	runner.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null"}
	for _, args := range [][]string{{"init", "-b", "main", "--", root}, {"config", "user.name", "reflog-test"}, {"config", "user.email", "reflog@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runner.Run(context.Background(), "commit", "--allow-empty", "-m", "initial recovery point"); err != nil {
		t.Fatal(err)
	}
	entries, err := Load(context.Background(), runner, Request{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].SHA == "" || entries[0].Selector != "HEAD@{0}" || entries[0].Actor != "reflog-test" || entries[0].Action != "commit (initial)" {
		t.Fatalf("reflog entries = %#v", entries)
	}
}

func TestLoadValidatesBoundsAndRefs(t *testing.T) {
	for _, request := range []Request{{Ref: "--bad"}, {Ref: "HEAD", Skip: -1}} {
		if _, err := Load(context.Background(), git.Runner{}, request); err == nil {
			t.Fatalf("request %#v unexpectedly accepted", request)
		}
	}
	if _, err := parse([]byte("sha\x00selector\x00bad\x00actor\x00subject\x00")); err == nil {
		t.Fatal("invalid timestamp accepted")
	}
}
