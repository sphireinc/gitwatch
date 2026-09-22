// Package atomicreplace contains the platform-specific final step for
// replacing a file that has already been written and closed.
package atomicreplace

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// File replaces target with temporary. On Unix this is the atomic rename
// operation. Windows does not allow Rename to overwrite an existing file, so
// move the old target aside first and restore it if installing the new file
// fails. The caller owns cleanup of temporary.
func File(temporary, target string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(temporary, target)
	}

	old, err := os.CreateTemp(filepath.Dir(target), ".gitwatch-replaced-")
	if err != nil {
		return err
	}
	backup := old.Name()
	if err := old.Close(); err != nil {
		_ = os.Remove(backup)
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}

	if err := os.Rename(target, backup); err != nil {
		if os.IsNotExist(err) {
			return os.Rename(temporary, target)
		}
		return err
	}
	if err := os.Rename(temporary, target); err != nil {
		if restoreErr := os.Rename(backup, target); restoreErr != nil {
			return fmt.Errorf("install replacement: %w (restore original: %v)", err, restoreErr)
		}
		return err
	}
	return os.Remove(backup)
}
