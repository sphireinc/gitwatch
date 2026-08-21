// Package releasearchive writes deterministic gitwatch release archives.
package releasearchive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

// Format identifies a supported release archive format.
type Format string

const (
	// TarGzip writes a gzip-compressed POSIX tar archive.
	TarGzip Format = "tar.gz"
	// ZIP writes a ZIP archive.
	ZIP Format = "zip"
)

type entry struct {
	abs  string
	name string
	info fs.FileInfo
}

// Write creates an archive of root under the top-level archive name. File
// ordering, ownership, modes, and timestamps are normalized for reproducible
// output. Root must contain only directories and regular files.
func Write(output, root, name string, format Format, timestamp time.Time) (err error) {
	if output == "" || root == "" || name == "" {
		return fmt.Errorf("output, root, and name are required")
	}
	if path.Base(name) != name || name == "." || name == ".." {
		return fmt.Errorf("invalid archive root name %q", name)
	}
	entries, err := collect(root, name)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer func() {
		if closeErr := out.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close archive: %w", closeErr)
		}
		if err != nil {
			_ = os.Remove(output)
		}
	}()

	timestamp = timestamp.UTC().Truncate(time.Second)
	switch format {
	case TarGzip:
		err = writeTarGzip(out, entries, timestamp)
	case ZIP:
		err = writeZIP(out, entries, timestamp)
	default:
		err = fmt.Errorf("unsupported archive format %q", format)
	}
	return err
}

func collect(root, name string) ([]entry, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve archive root: %w", err)
	}
	var entries []entry
	err = filepath.WalkDir(root, func(filename string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported archive entry %s with mode %s", filename, info.Mode())
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		archiveName := name
		if relative != "." {
			archiveName = path.Join(name, filepath.ToSlash(relative))
		}
		if info.IsDir() {
			archiveName += "/"
		}
		entries = append(entries, entry{abs: filename, name: archiveName, info: info})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect archive entries: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, nil
}

func normalizedMode(info fs.FileInfo) fs.FileMode {
	if info.IsDir() {
		return fs.ModeDir | 0o755
	}
	if info.Mode()&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

func writeTarGzip(output io.Writer, entries []entry, timestamp time.Time) error {
	gzipWriter, err := gzip.NewWriterLevel(output, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	gzipWriter.ModTime = timestamp
	gzipWriter.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	for _, item := range entries {
		header := &tar.Header{
			Name:       item.name,
			Mode:       int64(normalizedMode(item.info).Perm()),
			Size:       item.info.Size(),
			ModTime:    timestamp,
			AccessTime: time.Time{},
			ChangeTime: time.Time{},
			Typeflag:   tar.TypeReg,
			Format:     tar.FormatPAX,
		}
		if item.info.IsDir() {
			header.Size = 0
			header.Typeflag = tar.TypeDir
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("write tar header %s: %w", item.name, err)
		}
		if item.info.Mode().IsRegular() {
			if err := copyFile(tarWriter, item.abs); err != nil {
				return fmt.Errorf("write tar entry %s: %w", item.name, err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	return nil
}

func writeZIP(output io.Writer, entries []entry, timestamp time.Time) error {
	zipWriter := zip.NewWriter(output)
	for _, item := range entries {
		header := &zip.FileHeader{Name: item.name, Method: zip.Deflate, Modified: timestamp}
		header.SetMode(normalizedMode(item.info))
		if item.info.IsDir() {
			header.Method = zip.Store
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("write zip header %s: %w", item.name, err)
		}
		if item.info.Mode().IsRegular() {
			if err := copyFile(writer, item.abs); err != nil {
				return fmt.Errorf("write zip entry %s: %w", item.name, err)
			}
		}
	}
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("close zip writer: %w", err)
	}
	return nil
}

func copyFile(output io.Writer, filename string) (err error) {
	input, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := input.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	_, err = io.Copy(output, input)
	return err
}
