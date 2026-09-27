//go:build windows

package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func configureProcessCancellation(command *exec.Cmd) {
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
	command.WaitDelay = 250 * time.Millisecond
}
