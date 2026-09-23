//go:build windows

package git

import (
	"os/exec"
	"time"
)

func configureCommandCancellation(command *exec.Cmd) {
	// CommandContext terminates the direct Git process on Windows. Bound pipe
	// draining if a child process inherited stdout/stderr and outlives it.
	command.WaitDelay = 250 * time.Millisecond
}
