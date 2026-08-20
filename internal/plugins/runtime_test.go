package plugins

import (
	"context"
	"errors"
	"testing"
)

func TestRuntimeRejectsInvalidManifest(t *testing.T) {
	_, err := (Runtime{}).Run(context.Background(), Manifest{}, nil)
	if err == nil {
		t.Fatal("invalid manifest was executed")
	}
}

func TestLimitedWriterStopsOversizedOutput(t *testing.T) {
	writer := &limitedWriter{writer: discardWriter{}, limit: 3}
	if _, err := writer.Write([]byte("1234")); !errors.Is(err, ErrOutputLimit) {
		t.Fatalf("expected output limit, got %v", err)
	}
}

type discardWriter struct{}

func (discardWriter) Write(data []byte) (int, error) { return len(data), nil }
