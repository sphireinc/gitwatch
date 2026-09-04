// Command gitignore-sync creates deterministic offline gitignore assets from
// one explicitly pinned github/gitignore commit.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gitignoresync "github.com/sphireinc/git-watch/internal/gitignore/sync"
)

func main() {
	commit := flag.String("commit", "", "40-hex upstream commit (required)")
	repository := flag.String("repo", gitignoresync.DefaultRepository, "upstream repository")
	archiveURL := flag.String("archive-url", "", "archive URL override, primarily for tests")
	out := flag.String("out", "internal/gitignore/assets", "generated asset directory")
	skipSymlinks := flag.Bool("skip-symlinks", false, "skip upstream symlink aliases")
	flag.Parse()
	if *commit == "" {
		fatal("--commit is required")
	}
	assets, license, err := gitignoresync.Fetch(context.Background(), nil, gitignoresync.Config{Repository: *repository, Commit: *commit, ArchiveURL: *archiveURL, SyncedAt: syncTime(), SkipSymlinks: *skipSymlinks})
	if err != nil {
		fatal("fetch archive: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(*out, "catalog"), 0755); err != nil {
		fatal("create output: %v", err)
	}
	for _, asset := range assets {
		name := filepath.Join(*out, "catalog", filepath.FromSlash(asset.Template.ID.String()+".gitignore"))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			fatal("create template directory: %v", err)
		}
		if err := os.WriteFile(name, asset.Content, 0644); err != nil {
			fatal("write template: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(*out, "LICENSE.github-gitignore"), license, 0644); err != nil {
		fatal("write license: %v", err)
	}
	manifest, err := gitignoresync.BuildManifest(gitignoresync.Config{Repository: *repository, Commit: *commit, SyncedAt: syncTime()}, assets)
	if err != nil {
		fatal("build manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(*out, "manifest.json"), append(manifest, '\n'), 0644); err != nil {
		fatal("write manifest: %v", err)
	}
}

func syncTime() time.Time {
	if value := os.Getenv("SOURCE_DATE_EPOCH"); value != "" {
		var seconds int64
		if _, err := fmt.Sscan(value, &seconds); err == nil {
			return time.Unix(seconds, 0).UTC()
		}
	}
	return time.Unix(0, 0).UTC()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gitignore-sync: "+format+"\n", args...)
	os.Exit(2)
}
