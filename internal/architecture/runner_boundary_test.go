package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGitFeaturePackagesUseTheTypedRunner keeps Git process execution behind
// internal/git.Runner. Provider authentication, plugin supervision, and
// platform integration are separate process boundaries and are intentionally
// outside this list.
func TestGitFeaturePackagesUseTheTypedRunner(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(filename), "..")
	packages := []string{
		"app", "branches", "commitmodel", "history", "hunks", "operations",
		"patch", "registry", "remotes", "repo", "stash", "worktrees",
	}
	for _, packageName := range packages {
		dir := filepath.Join(root, packageName)
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, imported := range file.Imports {
				if strings.Trim(imported.Path.Value, `"`) == "os/exec" {
					t.Errorf("%s imports os/exec; use internal/git.Runner", path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", packageName, err)
		}
	}
}
