package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"

	publicplugin "github.com/jusanchez/gitwatch/pkg/plugin"
)

const MaxOutputBytes = 1 << 20

var ErrOutputLimit = errors.New("plugin output exceeded limit")

type Runtime struct {
	OutputLimit int64
}

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

func (r Runtime) Handshake(ctx context.Context, manifest Manifest, supported []Capability) (Negotiation, error) {
	negotiation := Negotiate(manifest, supported)
	if !negotiation.Accepted {
		return negotiation, nil
	}
	capabilities := make([]publicplugin.Capability, len(negotiation.Capabilities))
	for i, capability := range negotiation.Capabilities {
		capabilities[i] = publicplugin.Capability(capability)
	}
	payload, err := json.Marshal(publicplugin.HandshakeRequest{APIVersion: APIVersion, Capabilities: capabilities})
	if err != nil {
		return Negotiation{}, err
	}
	message, err := publicplugin.Encode(publicplugin.Message{Type: "handshake", Payload: payload})
	if err != nil {
		return Negotiation{}, err
	}
	result, err := r.Run(ctx, manifest, message)
	if err != nil {
		return Negotiation{}, err
	}
	response, err := publicplugin.Decode(result.Stdout)
	if err != nil || response.Type != "handshake" {
		return Negotiation{}, errors.New("invalid plugin handshake response")
	}
	var handshake publicplugin.HandshakeResponse
	if err := json.Unmarshal(response.Payload, &handshake); err != nil {
		return Negotiation{}, errors.New("invalid plugin handshake payload")
	}
	if handshake.APIVersion != APIVersion || !handshake.Accepted {
		return Negotiation{Reason: handshake.Reason}, nil
	}
	negotiation.Capabilities = negotiation.Capabilities[:0]
	for _, capability := range handshake.Capabilities {
		negotiation.Capabilities = append(negotiation.Capabilities, Capability(capability))
	}
	return negotiation, nil
}

func (r Runtime) Run(ctx context.Context, manifest Manifest, input []byte) (Result, error) {
	if err := manifest.Validate(); err != nil {
		return Result{}, err
	}
	limit := r.OutputLimit
	if limit <= 0 {
		limit = MaxOutputBytes
	}
	command := exec.CommandContext(ctx, manifest.Executable, "--gitwatch-plugin")
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedWriter{writer: &stdout, limit: limit}
	command.Stderr = &limitedWriter{writer: &stderr, limit: limit}
	err := command.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

type limitedWriter struct {
	writer io.Writer
	limit  int64
	wrote  int64
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if int64(len(data))+w.wrote > w.limit {
		remaining := w.limit - w.wrote
		if remaining > 0 {
			_, _ = w.writer.Write(data[:remaining])
		}
		w.wrote = w.limit
		return 0, ErrOutputLimit
	}
	n, err := w.writer.Write(data)
	w.wrote += int64(n)
	return n, err
}
