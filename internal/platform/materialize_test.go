package platform

import (
	"os"
	"testing"
)

func TestMaterializeFileUsesPrivatePermissionsAndCleansUp(t *testing.T) {
	materialized, err := MaterializeFile("dir/file name.txt", []byte("content"), 100)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(materialized.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 600", info.Mode().Perm())
	}
	if err := os.Remove(materialized.Path); err != nil {
		t.Fatal(err)
	}
	materialized.Cleanup()
	if _, err := os.Stat(materialized.Path); !os.IsNotExist(err) {
		t.Fatalf("materialized path still exists: %v", err)
	}
}

func TestMaterializeFileBoundsContent(t *testing.T) {
	if _, err := MaterializeFile("file", []byte("1234"), 3); err == nil {
		t.Fatal("oversized content was accepted")
	}
}
