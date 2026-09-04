package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExternalTool is an executable plus already-tokenized arguments. {path} is
// replaced inside individual argv tokens; no shell parser is ever involved.
type ExternalTool struct {
	Executable string
	Args       []string
}

func (t ExternalTool) Command(path, dir string, environment []string) (*exec.Cmd, error) {
	if strings.TrimSpace(t.Executable) == "" {
		return nil, fmt.Errorf("external tool executable is required")
	}
	if path == "" {
		return nil, fmt.Errorf("external tool path is required")
	}
	args := make([]string, 0, len(t.Args)+1)
	found := false
	for _, token := range t.Args {
		if strings.Contains(token, "{path}") {
			found = true
		}
		args = append(args, strings.ReplaceAll(token, "{path}", path))
	}
	if !found {
		args = append(args, path)
	}
	command := exec.Command(t.Executable, args...)
	command.Dir = dir
	if len(environment) > 0 {
		command.Env = append(os.Environ(), environment...)
	}
	return command, nil
}
