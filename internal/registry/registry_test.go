package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverBoundsAndFindsNestedRepositories(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "one")
	second := filepath.Join(root, "two")
	if err := os.MkdirAll(filepath.Join(first, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, ".git"), []byte("gitdir: elsewhere"), 0600); err != nil {
		t.Fatal(err)
	}
	repositories, err := Discover(context.Background(), []string{root}, Options{MaxRepositories: 1})
	if err != nil || len(repositories) != 1 {
		t.Fatalf("unexpected discovery: %#v, %v", repositories, err)
	}
}

func TestRegistryRoundTripUsesPrivateFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "registry.json")
	want := []Repository{{Path: "/tmp/repo", Name: "repo", Favorite: true}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || len(got) != 1 || !got[0].Favorite {
		t.Fatalf("unexpected registry: %#v, %v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("registry is not private: %v", err)
	}
}

func TestDiscoverSkipsSymlinkedGitMetadata(t *testing.T) {
	root := t.TempDir()
	realGit := filepath.Join(root, "real-git")
	if err := os.MkdirAll(realGit, 0700); err != nil {
		t.Fatal(err)
	}
	repoPath := filepath.Join(root, "linked")
	if err := os.MkdirAll(repoPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realGit, filepath.Join(repoPath, ".git")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	repositories, err := Discover(context.Background(), []string{root}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, repository := range repositories {
		if repository.Path == repoPath {
			t.Fatalf("symlinked .git was discovered: %#v", repositories)
		}
	}
}
