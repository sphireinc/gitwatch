package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	publicplugin "github.com/sphireinc/git-watch/pkg/plugin"
)

const MaxOutputBytes = 1 << 20

var ErrOutputLimit = errors.New("plugin output exceeded limit")
var ErrCapabilityDenied = errors.New("plugin capability was not granted")
var ErrPluginTimeout = errors.New("plugin execution timed out")
var ErrPluginCancelled = errors.New("plugin execution cancelled")
var ErrPluginProtocol = errors.New("invalid plugin protocol")

type Runtime struct {
	OutputLimit int64
}

type Supervision struct {
	MaxRestarts int
	Backoff     time.Duration
}

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
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
	message, err := publicplugin.Encode(publicplugin.Message{Type: publicplugin.MessageHandshake, Payload: payload})
	if err != nil {
		return Negotiation{}, err
	}
	result, err := r.RunWithCapabilities(ctx, manifest, message, negotiation.Capabilities)
	if err != nil {
		return Negotiation{}, err
	}
	response, err := publicplugin.Decode(result.Stdout)
	if err != nil || response.Type != publicplugin.MessageHandshake {
		return Negotiation{}, ErrPluginProtocol
	}
	var handshake publicplugin.HandshakeResponse
	if err := json.Unmarshal(response.Payload, &handshake); err != nil {
		return Negotiation{}, ErrPluginProtocol
	}
	if handshake.APIVersion != APIVersion || !handshake.Accepted {
		return Negotiation{Reason: handshake.Reason}, nil
	}
	granted := make(map[Capability]bool, len(negotiation.Capabilities))
	for _, capability := range negotiation.Capabilities {
		granted[capability] = true
	}
	negotiation.Capabilities = negotiation.Capabilities[:0]
	for _, capability := range handshake.Capabilities {
		if !granted[Capability(capability)] {
			return Negotiation{}, fmt.Errorf("%w: %s", ErrCapabilityDenied, capability)
		}
		negotiation.Capabilities = append(negotiation.Capabilities, Capability(capability))
	}
	return negotiation, nil
}

func (r Runtime) Run(ctx context.Context, manifest Manifest, input []byte) (Result, error) {
	return r.RunWithCapabilities(ctx, manifest, input, manifest.Capabilities)
}

func (r Runtime) RunWithCapabilities(ctx context.Context, manifest Manifest, input []byte, grants []Capability) (Result, error) {
	if err := manifest.Validate(); err != nil {
		return Result{}, err
	}
	allowed := make(map[Capability]bool, len(grants))
	for _, grant := range grants {
		allowed[grant] = true
	}
	for _, capability := range manifest.Capabilities {
		if !allowed[capability] {
			return Result{}, fmt.Errorf("%w: %s", ErrCapabilityDenied, capability)
		}
	}
	limit := r.OutputLimit
	if limit <= 0 {
		limit = MaxOutputBytes
	}
	started := time.Now()
	command := exec.CommandContext(ctx, manifest.Executable, "--gitwatch-plugin")
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedWriter{writer: &stdout, limit: limit}
	command.Stderr = &limitedWriter{writer: &stderr, limit: limit}
	err := command.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Duration: time.Since(started)}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return result, fmt.Errorf("%w: %v", ErrPluginTimeout, ctx.Err())
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return result, fmt.Errorf("%w: %v", ErrPluginCancelled, ctx.Err())
		}
		return result, err
	}
	return result, nil
}

func (r Runtime) Supervise(ctx context.Context, manifest Manifest, input []byte, grants []Capability, policy Supervision) (Result, error) {
	if policy.MaxRestarts < 0 {
		policy.MaxRestarts = 0
	}
	if policy.Backoff < 0 {
		policy.Backoff = 0
	}
	for attempt := 0; ; attempt++ {
		result, err := r.RunWithCapabilities(ctx, manifest, input, grants)
		if err == nil || attempt >= policy.MaxRestarts || ctx.Err() != nil {
			return result, err
		}
		if policy.Backoff > 0 {
			timer := time.NewTimer(policy.Backoff)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return result, ctx.Err()
			case <-timer.C:
			}
		}
	}
}

type limitedWriter struct {
	writer io.Writer
	limit  int64
	wrote  int64
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if int64(len(data))+w.wrote > w.limit {
		remaining := w.limit - w.wrote
		if remaining <= 0 {
			return 0, ErrOutputLimit
		}
		n, err := w.writer.Write(data[:remaining])
		w.wrote += int64(n)
		if err != nil {
			return n, err
		}
		return n, ErrOutputLimit
	}
	n, err := w.writer.Write(data)
	w.wrote += int64(n)
	return n, err
}
