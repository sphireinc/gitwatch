package catalog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type sourceTransport struct {
	archive []byte
	commit  string
}

func (t sourceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Context().Err() != nil {
		return nil, request.Context().Err()
	}
	if request.URL.Host == "api.github.com" {
		body, _ := json.Marshal(map[string]string{"sha": t.commit})
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	}
	return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(t.archive)), Header: make(http.Header)}, nil
}

func tinyArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	entries := map[string][]byte{"gitignore-abc/LICENSE": []byte("license"), "gitignore-abc/Node.gitignore": []byte("node\n")}
	for name, data := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestOpenFallsBackToEmbeddedWhenCacheIsCorrupt(t *testing.T) {
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "manifest.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	source, err := Open(cache)
	if err != nil {
		t.Fatal(err)
	}
	if source.Kind != SourceEmbedded || source.Catalog == nil {
		t.Fatalf("fallback source=%+v", source)
	}
}

func TestRefreshStoresImmutableCommitAndDoesNotTouchRepository(t *testing.T) {
	cache, repository := t.TempDir(), t.TempDir()
	marker := filepath.Join(repository, "keep.txt")
	if err := os.WriteFile(marker, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	commit := "0123456789012345678901234567890123456789"
	client := &http.Client{Transport: sourceTransport{archive: tinyArchive(t), commit: commit}}
	result, err := Refresh(context.Background(), RefreshConfig{CachePath: cache, Client: client, Now: func() time.Time { return time.Unix(100, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	if result.Commit != commit || result.Source.Kind != SourceCached || result.Source.Commit != commit {
		t.Fatalf("refresh result=%+v", result)
	}
	data, _ := os.ReadFile(marker)
	if string(data) != "unchanged" {
		t.Fatal("repository file changed")
	}
	if _, ok := result.Source.Catalog.Get("root/Node"); !ok {
		t.Fatal("refreshed template missing")
	}
}

func TestRefreshCancellationIsReportedBeforeNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Refresh(ctx, RefreshConfig{CachePath: t.TempDir(), Client: &http.Client{Transport: sourceTransport{commit: "0123456789012345678901234567890123456789"}}})
	if !errors.Is(err, ErrRefreshCancelled) {
		t.Fatalf("cancellation error=%v", err)
	}
}
