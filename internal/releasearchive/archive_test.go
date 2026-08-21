package releasearchive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestWriteIsDeterministic(t *testing.T) {
	root := fixture(t)
	timestamp := time.Date(2026, time.August, 21, 12, 30, 0, 0, time.UTC)
	for _, format := range []Format{TarGzip, ZIP} {
		t.Run(string(format), func(t *testing.T) {
			first := filepath.Join(t.TempDir(), "first."+string(format))
			second := filepath.Join(t.TempDir(), "second."+string(format))
			if err := Write(first, root, "gitwatch_1.0.0_test", format, timestamp); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(filepath.Join(root, "README.md"), timestamp.Add(time.Hour), timestamp.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
			if err := Write(second, root, "gitwatch_1.0.0_test", format, timestamp); err != nil {
				t.Fatal(err)
			}
			firstData, err := os.ReadFile(first)
			if err != nil {
				t.Fatal(err)
			}
			secondData, err := os.ReadFile(second)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(firstData, secondData) {
				t.Fatal("archives differ after source timestamp change")
			}
		})
	}
}

func TestTarGzipContentsAreNormalized(t *testing.T) {
	root := fixture(t)
	timestamp := time.Date(2026, time.August, 21, 12, 30, 0, 0, time.UTC)
	output := filepath.Join(t.TempDir(), "release.tar.gz")
	if err := Write(output, root, "release", TarGzip, timestamp); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := gzipReader.Close(); err != nil {
			t.Error(err)
		}
	})
	tarReader := tar.NewReader(gzipReader)
	var names []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
		if !header.ModTime.Equal(timestamp) || header.Uid != 0 || header.Gid != 0 {
			t.Fatalf("unnormalized header: %#v", header)
		}
	}
	want := []string{"release/", "release/README.md", "release/bin/", "release/bin/gitwatch"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %#v, want %#v", names, want)
	}
}

func TestZIPContentsAreNormalized(t *testing.T) {
	root := fixture(t)
	timestamp := time.Date(2026, time.August, 21, 12, 30, 0, 0, time.UTC)
	output := filepath.Join(t.TempDir(), "release.zip")
	if err := Write(output, root, "release", ZIP, timestamp); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	var names []string
	for _, file := range reader.File {
		names = append(names, file.Name)
		if !file.Modified.Equal(timestamp) {
			t.Fatalf("timestamp for %s = %s", file.Name, file.Modified)
		}
	}
	want := []string{"release/", "release/README.md", "release/bin/", "release/bin/gitwatch"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %#v, want %#v", names, want)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("gitwatch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "gitwatch"), []byte("binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
