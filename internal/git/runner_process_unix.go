//go:build !windows

package git

import (
	"github.com/sphireinc/git-watch/internal/platform"
	"os/exec"
)

func configureCommandCancellation(command *exec.Cmd) {
	platform.ConfigureProcessCancellation(command)
}
