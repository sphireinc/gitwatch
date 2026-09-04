package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sphireinc/git-watch/internal/gitignore/sync"
)

type SourceKind string

const (
	SourceEmbedded SourceKind = "embedded"
	SourceCached   SourceKind = "cached"
)

type Source struct {
	Catalog   *Catalog
	Kind      SourceKind
	Commit    string
	SyncedAt  time.Time
	CachePath string
}

// UseBundled returns the guaranteed offline source.
func UseBundled() (Source, error) {
	cat, err := Default()
	if err != nil {
		return Source{}, err
	}
	return Source{Catalog: cat, Kind: SourceEmbedded, Commit: cat.Version()}, nil
}

var ErrCacheInvalid = errors.New("cached gitignore catalog is invalid")
var ErrRefreshCancelled = errors.New("gitignore catalog refresh cancelled")

// DefaultCacheDir returns a user cache location, never a repository path.
func DefaultCacheDir() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "gitwatch", "gitignore-catalog"), nil
}

// Open chooses the newest valid cache snapshot and falls back to embedded on
// every cache error. The embedded catalog is always returned if available.
func Open(cachePath string) (Source, error) {
	result, err := UseBundled()
	if err != nil {
		return Source{}, err
	}
	if cachePath == "" {
		return result, nil
	}
	cached, loadErr := LoadFS(os.DirFS(cachePath))
	if loadErr != nil {
		return result, nil
	}
	metadata, metaErr := readMetadata(cachePath)
	if metaErr != nil {
		return result, nil
	}
	if metadata.Commit != cached.Version() {
		return result, nil
	}
	if metadata.SyncedAt.After(result.SyncedAt) {
		result = Source{Catalog: cached, Kind: SourceCached, Commit: cached.Version(), SyncedAt: metadata.SyncedAt, CachePath: cachePath}
	}
	return result, nil
}

type metadata struct {
	SyncedAt time.Time `json:"synced_at"`
	Commit   string    `json:"commit"`
}

func readMetadata(path string) (metadata, error) {
	data, err := os.ReadFile(filepath.Join(path, "metadata.json"))
	if err != nil {
		return metadata{}, err
	}
	var value metadata
	if err := json.Unmarshal(data, &value); err != nil || value.Commit == "" {
		return metadata{}, ErrCacheInvalid
	}
	return value, nil
}

type RefreshConfig struct {
	CachePath, Repository          string
	Timeout                        time.Duration
	MaxArchiveBytes, MaxEntryBytes int64
	Client                         *http.Client
	Now                            func() time.Time
}
type RefreshResult struct {
	Source   Source
	Commit   string
	SyncedAt time.Time
}

// Refresh resolves a branch to an immutable commit, fetches exactly that
// archive, validates it with the maintainer sync parser, and atomically swaps
// the cache directory. It never touches repository files.
func Refresh(ctx context.Context, cfg RefreshConfig) (RefreshResult, error) {
	if err := ctx.Err(); err != nil {
		return RefreshResult{}, fmt.Errorf("%w: %v", ErrRefreshCancelled, err)
	}
	if cfg.Repository == "" {
		cfg.Repository = sync.DefaultRepository
	}
	if cfg.CachePath == "" {
		var err error
		cfg.CachePath, err = DefaultCacheDir()
		if err != nil {
			return RefreshResult{}, err
		}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{}
	}
	refreshCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	commit, err := ResolveCommit(refreshCtx, cfg.Client, cfg.Repository)
	if err != nil {
		return RefreshResult{}, err
	}
	assets, _, err := sync.Fetch(refreshCtx, cfg.Client, sync.Config{Repository: cfg.Repository, Commit: commit, MaxArchiveBytes: cfg.MaxArchiveBytes, MaxEntryBytes: cfg.MaxEntryBytes})
	if err != nil {
		return RefreshResult{}, err
	}
	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now()
	}
	manifest, err := sync.BuildManifest(sync.Config{Repository: cfg.Repository, Commit: commit, SyncedAt: now}, assets)
	if err != nil {
		return RefreshResult{}, err
	}
	if err := writeCache(cfg.CachePath, manifest, commit, now, assets); err != nil {
		return RefreshResult{}, err
	}
	source, err := Open(cfg.CachePath)
	if err != nil {
		return RefreshResult{}, err
	}
	return RefreshResult{Source: source, Commit: commit, SyncedAt: now}, nil
}

func ResolveCommit(ctx context.Context, client *http.Client, repository string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repository+"/commits/main", nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve upstream commit: %s", response.Status)
	}
	var value struct {
		SHA string `json:"sha"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 16<<10))
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	if err := sync.ValidateCommit(value.SHA); err != nil {
		return "", err
	}
	return value.SHA, nil
}

func writeCache(path string, manifest []byte, commit string, syncedAt time.Time, catalogAssets []sync.Asset) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".gitignore-catalog-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if err := os.MkdirAll(filepath.Join(temporary, "catalog"), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(temporary, "manifest.json"), manifest, 0600); err != nil {
		return err
	}
	metadataBytes, err := json.Marshal(metadata{Commit: commit, SyncedAt: syncedAt})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(temporary, "metadata.json"), metadataBytes, 0600); err != nil {
		return err
	}
	for _, asset := range catalogAssets {
		target := filepath.Join(temporary, "catalog", filepath.FromSlash(asset.Template.ID.String()+".gitignore"))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(target, asset.Content, 0600); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(temporary, "LICENSE.github-gitignore"), []byte("validated upstream license\n"), 0600); err != nil {
		return err
	}
	old := path + ".old"
	_ = os.RemoveAll(old)
	if err := os.Rename(path, old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		if _, statErr := os.Stat(old); statErr == nil {
			_ = os.Rename(old, path)
		}
		return err
	}
	_ = os.RemoveAll(old)
	return nil
}
