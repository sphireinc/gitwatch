package platform

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// OpenURLCommand returns the platform command used to open an HTTP(S) URL.
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
