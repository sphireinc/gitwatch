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
	if path == "" {
		return nil, fmt.Errorf("external tool path is required")
	}
	return t.CommandValues(map[string]string{"path": path}, dir, environment)
}

// CommandValues expands only named argv-token placeholders. It never invokes
// a shell, and each expanded token remains one process argument.
func (t ExternalTool) CommandValues(values map[string]string, dir string, environment []string) (*exec.Cmd, error) {
	if strings.TrimSpace(t.Executable) == "" {
		return nil, fmt.Errorf("external tool executable is required")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("external tool values are required")
	}
	args := make([]string, 0, len(t.Args)+1)
	foundPath := false
	for _, token := range t.Args {
		if strings.Contains(token, "{path}") {
			foundPath = true
		}
		for name, value := range values {
			if value == "" {
				continue
			}
			token = strings.ReplaceAll(token, "{"+name+"}", value)
		}
		args = append(args, token)
	}
	if !foundPath {
		if path := values["path"]; path != "" {
			args = append(args, path)
		}
	}
	command := exec.Command(t.Executable, args...)
	command.Dir = dir
	if len(environment) > 0 {
		command.Env = append(os.Environ(), environment...)
	}
	return command, nil
}
