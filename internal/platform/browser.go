package platform

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

func OpenURLCommand(raw string) (*exec.Cmd, error) {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, fmt.Errorf("unsupported URL")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", raw), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", raw), nil
	default:
		return exec.Command("xdg-open", raw), nil
	}
}
