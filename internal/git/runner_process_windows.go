//go:build windows

package git

import (
	"os/exec"

	"github.com/sphireinc/git-watch/internal/platform"
)

func configureCommandCancellation(command *exec.Cmd) {
	platform.ConfigureProcessCancellation(command)
}
