// Package sync imports a commit-pinned github/gitignore archive into stable
// offline assets. It has no dependency on the TUI or Git process runner.
package sync

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

const DefaultRepository = "github/gitignore"

var (
	ErrInvalidCommit           = errors.New("commit must be exactly 40 hexadecimal characters")
	ErrArchiveTooLarge         = errors.New("gitignore archive exceeds size limit")
	ErrUnsafeArchivePath       = errors.New("unsafe archive path")
	ErrUnsupportedArchiveEntry = errors.New("unsupported archive entry")
	ErrHTTPStatus              = errors.New("upstream archive returned an unexpected HTTP status")
	ErrHashMismatch            = errors.New("catalog asset hash mismatch")
	ErrMissingLicense          = errors.New("upstream archive does not contain LICENSE")
)

type Config struct {
	Repository      string
	Commit          string
	ArchiveURL      string
	MaxArchiveBytes int64
	MaxEntryBytes   int64
	SyncedAt        time.Time
}

type Asset struct {
	Template domain.Template
	Content  []byte
}

type Manifest struct {
	Repository string          `json:"repository"`
	Commit     string          `json:"commit"`
	SyncedAt   string          `json:"synced_at"`
	Templates  []ManifestEntry `json:"templates"`
}

type ManifestEntry struct {
	ID         domain.TemplateID       `json:"id"`
	SourcePath string                  `json:"source_path"`
	Category   domain.TemplateCategory `json:"category"`
	SHA256     string                  `json:"sha256"`
	Bytes      int                     `json:"bytes"`
}

func ValidateCommit(commit string) error {
	if len(commit) != 40 {
		return ErrInvalidCommit
	}
	for _, r := range commit {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return ErrInvalidCommit
		}
	}
	return nil
}

func Fetch(ctx context.Context, client *http.Client, cfg Config) ([]Asset, []byte, error) {
	if err := ValidateCommit(cfg.Commit); err != nil {
		return nil, nil, err
	}
	if cfg.Repository == "" {
		cfg.Repository = DefaultRepository
	}
	if cfg.ArchiveURL == "" {
		cfg.ArchiveURL = "https://github.com/" + cfg.Repository + "/archive/" + cfg.Commit + ".tar.gz"
	}
	if cfg.MaxArchiveBytes <= 0 {
		cfg.MaxArchiveBytes = 64 << 20
	}
	if cfg.MaxEntryBytes <= 0 {
		cfg.MaxEntryBytes = 4 << 20
	}
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ArchiveURL, nil)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: %s", ErrHTTPStatus, res.Status)
	}
	limited := io.LimitReader(res.Body, cfg.MaxArchiveBytes+1)
	archive, err := io.ReadAll(limited)
	if err != nil {
		return nil, nil, err
	}
	if int64(len(archive)) > cfg.MaxArchiveBytes {
		return nil, nil, ErrArchiveTooLarge
	}
	return ParseArchive(archive, cfg)
}

func ParseArchive(data []byte, cfg Config) ([]Asset, []byte, error) {
	if err := ValidateCommit(cfg.Commit); err != nil {
		return nil, nil, err
	}
	if cfg.MaxEntryBytes <= 0 {
		cfg.MaxEntryBytes = 4 << 20
	}
	zr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, nil, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	var assets []Asset
	var license []byte
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		clean, err := archivePath(header.Name)
		if err != nil {
			return nil, nil, err
		}
		if header.Typeflag == tar.TypeDir || clean == "" {
			continue
		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink || header.Typeflag != tar.TypeReg {
			return nil, nil, fmt.Errorf("%w: %s (type %q)", ErrUnsupportedArchiveEntry, clean, header.Typeflag)
		}
		if header.Size < 0 || header.Size > cfg.MaxEntryBytes {
			return nil, nil, ErrArchiveTooLarge
		}
		content, err := io.ReadAll(io.LimitReader(tr, cfg.MaxEntryBytes+1))
		if err != nil {
			return nil, nil, err
		}
		if int64(len(content)) != header.Size || int64(len(content)) > cfg.MaxEntryBytes {
			return nil, nil, ErrArchiveTooLarge
		}
		if clean == "LICENSE" {
			license = append([]byte(nil), content...)
			continue
		}
		category, source, ok := classify(clean)
		if !ok {
			continue
		}
		id, err := domain.NewTemplateID(category, strings.TrimSuffix(source, ".gitignore"))
		if err != nil {
			return nil, nil, err
		}
		assets = append(assets, Asset{Template: domain.Template{ID: id, Name: strings.TrimSuffix(path.Base(source), ".gitignore"), Category: category, SourcePath: clean, ContentSHA256: digest(content)}, Content: append([]byte(nil), content...)})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Template.ID < assets[j].Template.ID })
	if license == nil {
		return nil, nil, ErrMissingLicense
	}
	return assets, license, nil
}

func BuildManifest(cfg Config, assets []Asset) ([]byte, error) {
	if err := ValidateCommit(cfg.Commit); err != nil {
		return nil, err
	}
	if cfg.Repository == "" {
		cfg.Repository = DefaultRepository
	}
	stamp := cfg.SyncedAt.UTC().Format(time.RFC3339)
	entries := make([]ManifestEntry, 0, len(assets))
	for _, asset := range assets {
		if asset.Template.ID == "" || digest(asset.Content) != asset.Template.ContentSHA256 {
			return nil, fmt.Errorf("%w: %s", ErrHashMismatch, asset.Template.ID)
		}
		entries = append(entries, ManifestEntry{ID: asset.Template.ID, SourcePath: asset.Template.SourcePath, Category: asset.Template.Category, SHA256: asset.Template.ContentSHA256, Bytes: len(asset.Content)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return json.MarshalIndent(Manifest{Repository: cfg.Repository, Commit: strings.ToLower(cfg.Commit), SyncedAt: stamp, Templates: entries}, "", "  ")
}

func archivePath(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return "", ErrUnsafeArchivePath
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", ErrUnsafeArchivePath
		}
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", ErrUnsafeArchivePath
	}
	parts := strings.Split(clean, "/")
	if len(parts) == 1 && parts[0] != "" {
		return "", nil
	}
	if len(parts) < 2 || parts[0] == "" {
		return "", ErrUnsafeArchivePath
	}
	return strings.Join(parts[1:], "/"), nil
}

func classify(value string) (domain.TemplateCategory, string, bool) {
	if !strings.HasSuffix(value, ".gitignore") {
		return "", "", false
	}
	switch {
	case !strings.Contains(value, "/"):
		return domain.CategoryRoot, value, true
	case strings.HasPrefix(value, "Global/"):
		return domain.CategoryGlobal, strings.TrimPrefix(value, "Global/"), true
	case strings.HasPrefix(value, "community/"):
		return domain.CategoryCommunity, strings.TrimPrefix(value, "community/"), true
	default:
		return "", "", false
	}
}

func digest(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
