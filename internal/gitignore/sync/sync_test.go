package sync

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func archive(t *testing.T, entries map[string]struct {
	kind byte
	body []byte
}) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	tw := tar.NewWriter(zw)
	for name, entry := range entries {
		h := &tar.Header{Name: name, Mode: 0644, Size: int64(len(entry.body)), Typeflag: entry.kind}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestParseArchiveSelectsAndSortsTemplates(t *testing.T) {
	data := archive(t, map[string]struct {
		kind byte
		body []byte
	}{
		"gitignore-abc/LICENSE":                         {kind: tar.TypeReg, body: []byte("CC0")},
		"gitignore-abc/community/Java/Gradle.gitignore": {kind: tar.TypeReg, body: []byte("gradle")},
		"gitignore-abc/Global/macOS.gitignore":          {kind: tar.TypeReg, body: []byte("mac")},
		"gitignore-abc/Node.gitignore":                  {kind: tar.TypeReg, body: []byte("node")},
		"gitignore-abc/.github/workflows/ci.yml":        {kind: tar.TypeReg, body: []byte("ci")},
	})
	assets, license, err := ParseArchive(data, Config{Commit: "0123456789012345678901234567890123456789"})
	if err != nil {
		t.Fatal(err)
	}
	if string(license) != "CC0" || len(assets) != 3 {
		t.Fatalf("license/assets = %q/%d", license, len(assets))
	}
	if assets[0].Template.ID.String() != "community/Java/Gradle" || assets[2].Template.ID.String() != "root/Node" {
		t.Fatalf("unsorted assets: %+v", assets)
	}
	manifest, err := BuildManifest(Config{Commit: "0123456789012345678901234567890123456789", SyncedAt: time.Unix(0, 0)}, assets)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte(`"synced_at": "1970-01-01T00:00:00Z"`)) {
		t.Fatal("manifest timestamp not deterministic")
	}
}

func TestParseArchiveRejectsTraversalAndSymlink(t *testing.T) {
	commit := "0123456789012345678901234567890123456789"
	for _, name := range []string{"gitignore-abc/../Node.gitignore", "/Node.gitignore", "gitignore-abc/Node\\evil.gitignore"} {
		data := archive(t, map[string]struct {
			kind byte
			body []byte
		}{name: {kind: tar.TypeReg, body: []byte("x")}, "gitignore-abc/LICENSE": {kind: tar.TypeReg, body: []byte("CC0")}})
		_, _, err := ParseArchive(data, Config{Commit: commit})
		if !errors.Is(err, ErrUnsafeArchivePath) {
			t.Errorf("%q error=%v", name, err)
		}
	}
	_, _, err := ParseArchive(archive(t, map[string]struct {
		kind byte
		body []byte
	}{"gitignore-abc/Node.gitignore": {kind: tar.TypeSymlink}}), Config{Commit: commit})
	if !errors.Is(err, ErrUnsupportedArchiveEntry) {
		t.Fatalf("symlink error=%v", err)
	}
}

func TestManifestDetectsHashMismatch(t *testing.T) {
	_, err := BuildManifest(Config{Commit: "0123456789012345678901234567890123456789"}, []Asset{{Template: structTemplate("root/Node", "wrong"), Content: []byte("node")}})
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("error=%v", err)
	}
}

func structTemplate(id, sum string) domain.Template {
	parsed, err := domain.ParseTemplateID(id)
	if err != nil {
		panic(err)
	}
	return domain.Template{ID: parsed, ContentSHA256: sum}
}
