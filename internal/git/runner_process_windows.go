//go:build windows

package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func configureCommandCancellation(command *exec.Cmd) {
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
			killCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			killer := exec.CommandContext(
				killCtx,
				filepath.Join(systemRoot, "System32", "taskkill.exe"),
				"/PID", strconv.Itoa(command.Process.Pid), "/T", "/F",
			)
			if err := killer.Run(); err == nil {
				return nil
			}
		}
		return command.Process.Kill()
	}
	// Git may spawn a remote helper that inherits stdout/stderr. Kill the
	// process tree on cancellation and bound any remaining pipe draining.
	command.WaitDelay = 250 * time.Millisecond
}
