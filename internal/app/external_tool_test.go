package app

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/compare"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/repo"
)

func TestConfiguredEditorUsesSelectedPathAndRefreshesAfterExit(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("space name.txt")}})
	m.rebuildStatusFileTree()
	m.EditorTool = platform.ExternalTool{Executable: "editor", Args: []string{"--file={path}"}}
	updated, command := m.Update(key("ctrl+e"))
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.Status != "editor active" {
		t.Fatalf("editor route: command nil=%v state=%v status=%q", command == nil, m.State, m.Status)
	}
	updated, refresh := m.Update(ExternalToolFinishedMsg{Name: "editor", Repository: m.repositoryGeneration})
	m = updated.(Model)
	if refresh == nil || m.State != StateReady || m.Status != "editor exited; refreshing authoritative status" {
		t.Fatalf("editor completion: refresh nil=%v state=%v status=%q", refresh == nil, m.State, m.Status)
	}
}

func TestHistoricalToolReadyMaterializesAndStartsConfiguredTool(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.Compare.Result = compare.Result{
		Left:    compare.Revision{SHA: "0123456789012345678901234567890123456789"},
		Right:   compare.Revision{SHA: "abcdefabcdefabcdefabcdefabcdefabcdefabcd"},
		Changes: []compare.Change{{NewPath: "historical file.txt"}},
	}
	m.Compare.Selected = 0
	m.EditorTool = platform.ExternalTool{Executable: "editor", Args: []string{"{path}"}}
	updated, command := m.Update(HistoricalToolReadyMsg{
		Name: "editor", Repository: m.repositoryGeneration, Tool: m.EditorTool,
		Path: "historical file.txt", Content: []byte("old content"),
	})
	m = updated.(Model)
	if command == nil || m.State != StateOperationPending || m.Status != "editor active" {
		t.Fatalf("historical tool route: command nil=%v state=%v status=%q", command == nil, m.State, m.Status)
	}
	if message := command(); message == nil {
		t.Fatal("historical tool command returned no completion message")
	}
}

func TestConfiguredToolIsNotAvailableForDirectorySelection(t *testing.T) {
	m := New()
	m.Discovery.Root = t.TempDir()
	m.Files.SetEntries([]repo.Entry{{Path: repo.Path("dir/file.txt")}})
	m.rebuildStatusFileTree()
	m.StatusTreeMode = true
	m.rebuildStatusFileTree()
	m.FileTree.Selected = 0
	m.EditorTool = platform.ExternalTool{Executable: "editor"}
	updated, command := m.Update(key("ctrl+e"))
	m = updated.(Model)
	if command != nil || m.Status != "editor: select a file row" {
		t.Fatalf("directory editor route: command nil=%v status=%q", command == nil, m.Status)
	}
}
