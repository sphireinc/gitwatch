package manage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

// Apply rechecks the planned bytes and replaces the target atomically. The
// caller must request the normal repository status refresh after success.
func Apply(plan domain.MutationPlan) error {
	if plan.Root == "" || plan.Path != ".gitignore" {
		return domain.ErrUnsafeTarget
	}
	path := filepath.Join(plan.Root, plan.Path)
	info, err := os.Lstat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat gitignore: %w", err)
	}
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return domain.ErrUnsafeTarget
	}
	current, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read gitignore: %w", err)
	}
	if err == nil && hash(current) != plan.BeforeSHA256 {
		return domain.ErrConcurrentModification
	}
	if err != nil && plan.BeforeSHA256 != hash(nil) {
		return domain.ErrConcurrentModification
	}
	temporary, err := os.CreateTemp(plan.Root, ".gitignore.gitwatch-*")
	if err != nil {
		return fmt.Errorf("create temporary gitignore: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	permissions := os.FileMode(0644)
	if info != nil {
		permissions = info.Mode().Perm()
	}
	if err := temporary.Chmod(permissions); err != nil {
		temporary.Close()
		return fmt.Errorf("chmod temporary gitignore: %w", err)
	}
	if _, err := temporary.Write(plan.ResultBytes); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary gitignore: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary gitignore: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary gitignore: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace gitignore: %w", err)
	}
	directory, err := os.Open(plan.Root)
	if err == nil {
		syncErr := directory.Sync()
		closeErr := directory.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
