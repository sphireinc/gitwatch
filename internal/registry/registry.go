package registry

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Repository struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	LastOpened time.Time `json:"last_opened,omitempty"`
	Favorite   bool      `json:"favorite,omitempty"`
	Groups     []string  `json:"groups,omitempty"`
}

type Options struct {
	MaxDepth        int
	MaxRepositories int
	IgnoreDirs      []string
}

func Discover(ctx context.Context, roots []string, options Options) ([]Repository, error) {
	if options.MaxDepth <= 0 {
		options.MaxDepth = 4
	}
	if options.MaxRepositories <= 0 {
		options.MaxRepositories = 256
	}
	ignored := map[string]bool{".git": true, ".hg": true, ".svn": true, "node_modules": true}
	for _, name := range options.IgnoreDirs {
		ignored[name] = true
	}
	seen := make(map[string]bool)
	var repositories []Repository
	for _, root := range roots {
		root, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		root = filepath.Clean(root)
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if len(repositories) >= options.MaxRepositories {
				return filepath.SkipAll
			}
			if entry.IsDir() && path != root && ignored[entry.Name()] {
				return filepath.SkipDir
			}
			if !entry.IsDir() && entry.Name() != ".git" {
				return nil
			}
			if !isRepository(path, entry) {
				if entry.IsDir() && depth(root, path) >= options.MaxDepth {
					return filepath.SkipDir
				}
				return nil
			}
			repositoryPath := path
			if entry.Name() == ".git" {
				repositoryPath = filepath.Dir(path)
			}
			repositoryPath, _ = filepath.Abs(repositoryPath)
			if !seen[repositoryPath] {
				seen[repositoryPath] = true
				repositories = append(repositories, Repository{Path: repositoryPath, Name: filepath.Base(repositoryPath)})
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return repositories, nil
}

func isRepository(path string, entry fs.DirEntry) bool {
	if entry.Name() == ".git" {
		return true
	}
	if !entry.IsDir() {
		return false
	}
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && info != nil
}

func depth(root, path string) int {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return 0
	}
	return len(strings.Split(relative, string(filepath.Separator)))
}

func Load(path string) ([]Repository, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var repositories []Repository
	if err := json.Unmarshal(data, &repositories); err != nil {
		return nil, err
	}
	return repositories, nil
}

func Save(path string, repositories []Repository) error {
	data, err := json.MarshalIndent(repositories, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
