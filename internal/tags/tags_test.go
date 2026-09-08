package tags

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type tagRunner struct {
	tagOutput    []byte
	remoteOutput []byte
}

func (r tagRunner) Run(ctx context.Context, args ...string) (git.Result, error) {
	if len(args) > 0 && args[len(args)-1] == "refs/tags" {
		return git.Result{Args: args, Stdout: r.tagOutput}, nil
	}
	return git.Result{Args: args, Stdout: r.remoteOutput}, nil
}

func (r tagRunner) RunBounded(ctx context.Context, maxBytes int, args ...string) (git.Result, error) {
	return r.Run(ctx, args...)
}

func TestLoadPreservesLightweightAnnotatedAndRemoteMetadata(t *testing.T) {
	tagOutput := strings.Join([]string{
		"v1.0.0", "111", "commit", "", "", "", "", "", "first release",
		"v2.0.0", "222", "tag", "333", "commit", "Release Bot", "bot@example.test", "2026-09-05 12:00:00 +0000", "second release",
		"",
	}, "\x00")
	remoteOutput := strings.Join([]string{"origin/tags/v2.0.0", "222", "upstream/tags/v9.0.0", "999", ""}, "\x00")
	snapshot, err := Load(context.Background(), tagRunner{tagOutput: []byte(tagOutput), remoteOutput: []byte(remoteOutput)}, LoadRequest{Repository: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tags) != 2 || snapshot.Truncated {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	lightweight, annotated := snapshot.Tags[0], snapshot.Tags[1]
	if lightweight.Kind != Lightweight || lightweight.TargetID != "111" || lightweight.Signature != SignatureUnknown {
		t.Fatalf("lightweight tag = %+v", lightweight)
	}
	if annotated.Kind != Annotated || annotated.TargetID != "333" || annotated.TargetKind != "commit" || annotated.TaggerName != "Release Bot" || annotated.RemotePresence != RemotePresent || len(annotated.RemoteNames) != 1 || annotated.RemoteNames[0] != "origin" {
		t.Fatalf("annotated tag = %+v", annotated)
	}
}

func TestLoadBoundsTagsAndRejectsMalformedRecords(t *testing.T) {
	data := []byte(strings.Join([]string{"v1", "111", "commit", "", "", "", "", "", "one", "v2", "222", "commit", "", "", "", "", "", "two", ""}, "\x00"))
	snapshot, err := Load(context.Background(), tagRunner{tagOutput: data}, LoadRequest{Repository: "/repo", Limits: Limits{MaxTags: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tags) != 1 || !snapshot.Truncated {
		t.Fatalf("bounded snapshot = %+v", snapshot)
	}
	_, _, err = parseTagRefs([]byte("bad\x00record\x00"), 10)
	if err != ErrMalformedRecord {
		t.Fatalf("malformed error = %v", err)
	}
}

func TestVerifyIsOnDemandAndRejectsUnsafeNames(t *testing.T) {
	state, err := Verify(context.Background(), tagRunner{}, "v2.0.0")
	if err != nil || state != SignatureValid {
		t.Fatalf("verify = state=%q err=%v", state, err)
	}
	if _, err := Verify(context.Background(), tagRunner{}, "-bad"); err != ErrInvalidName {
		t.Fatalf("unsafe tag verification error = %v", err)
	}
}

func TestLoadThousandsOfTagsWithinBound(t *testing.T) {
	fields := make([]string, 0, 2000*9)
	for i := 0; i < 2000; i++ {
		fields = append(fields, fmt.Sprintf("v%04d", i), fmt.Sprintf("%040d", i), "commit", "", "", "", "", "", "fixture tag")
	}
	fields = append(fields, "")
	snapshot, err := Load(context.Background(), tagRunner{tagOutput: []byte(strings.Join(fields, "\x00"))}, LoadRequest{Repository: "/repo", Limits: Limits{MaxTags: 2048}})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tags) != 2000 || snapshot.Truncated || snapshot.Tags[1999].Name != "v1999" {
		t.Fatalf("large tag fixture = count=%d truncated=%v last=%+v", len(snapshot.Tags), snapshot.Truncated, snapshot.Tags[len(snapshot.Tags)-1])
	}
}
