package platform

import "os/exec"

// ConfigureProcessCancellation makes context cancellation terminate the whole
// child-process tree and bounds pipe draining after the process is stopped.
func ConfigureProcessCancellation(command *exec.Cmd) {
	configureProcessCancellation(command)
}
