package platform

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	mu sync.Mutex
	w  io.Writer
}

func NewLogger(path string) (*Logger, error) {
	if path == "" {
		return &Logger{w: io.Discard}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{w: f}, nil
}
func (l *Logger) Printf(format string, args ...any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	message := SafeText(RedactSecrets(fmt.Sprintf(format, args...)))
	_, err := fmt.Fprintf(l.w, "%s %s\n", time.Now().UTC().Format(time.RFC3339Nano), message)
	return err
}
