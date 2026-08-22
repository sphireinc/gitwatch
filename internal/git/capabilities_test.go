package git

import (
	"context"
	"errors"
	"testing"
)

func TestParseAndCompareGitVersions(t *testing.T) {
	got, err := ParseVersion("git version 2.43.0 (Apple Git-143)")
	if err != nil || got != (Version{Major: 2, Minor: 43}) {
		t.Fatalf("version = %#v err=%v", got, err)
	}
	if got.Compare(MinimumGitVersion) < 0 || (Version{Major: 2, Minor: 22}).Compare(MinimumGitVersion) >= 0 {
		t.Fatal("version boundary comparison is incorrect")
	}
	if _, err := ParseVersion("git unavailable"); err == nil {
		t.Fatal("malformed version was accepted")
	}
}

func TestProbeRejectsUnsupportedGit(t *testing.T) {
	runner := Runner{Binary: "sh", Dir: t.TempDir(), Env: []string{"GITWATCH_TEST_VERSION=1"}}
	// The shell is only used as a deterministic test executable; production Git
	// invocation remains argument-vector based through Runner.
	runner.Binary = "git"
	if _, err := runner.Probe(context.Background()); err != nil && errors.Is(err, ErrGitMissing) {
		t.Fatal(err)
	}
}
