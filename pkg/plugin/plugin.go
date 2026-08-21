// Package plugin contains the stable, dependency-free wire contract for
// gitwatch plugins. It intentionally imports no internal gitwatch packages.
package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// APIVersion is the plugin wire-protocol version supported by this SDK.
const APIVersion = 1

const (
	// MaxMessageBytes is the largest encoded protocol message accepted by Decode.
	MaxMessageBytes = 1 << 20
	// MaxFieldBytes is the largest message type or identifier accepted by Decode.
	MaxFieldBytes = 256
)

const (
	// MessageHandshake identifies a host-plugin capability negotiation message.
	MessageHandshake = "handshake"
	// MessageEvent identifies a plugin lifecycle or repository event.
	MessageEvent = "event"
	// MessageCommand identifies a command extension payload.
	MessageCommand = "command"
	// MessagePanel identifies a panel extension payload.
	MessagePanel = "panel"
	// MessageWidget identifies a status-widget extension payload.
	MessageWidget = "status_widget"
)

// Lifecycle identifies a plugin process lifecycle transition.
type Lifecycle string

const (
	// LifecycleStart indicates that a plugin started.
	LifecycleStart Lifecycle = "start"
	// LifecycleStop indicates that a plugin stopped normally.
	LifecycleStop Lifecycle = "stop"
	// LifecycleFailure indicates that a plugin stopped because of a failure.
	LifecycleFailure Lifecycle = "failure"
)

// Capability identifies an extension surface requested by a plugin.
type Capability string

const (
	// Command allows a plugin to expose command-palette actions.
	Command Capability = "command"
	// Panel allows a plugin to expose a bounded panel payload.
	Panel Capability = "panel"
	// StatusWidget allows a plugin to expose status-bar content.
	StatusWidget Capability = "status_widget"
	// RepositoryRead allows a plugin to receive read-only repository context.
	RepositoryRead Capability = "repository_read"
)

// Manifest declares a plugin's identity, protocol version, and requested capabilities.
type Manifest struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	APIVersion   int             `json:"api_version"`
	Capabilities []Capability    `json:"capabilities"`
	ConfigSchema json.RawMessage `json:"config_schema,omitempty"`
}

// Validate checks that the manifest is compatible with this SDK and internally consistent.
func (m Manifest) Validate() error {
	if m.ID == "" || m.Name == "" || m.Version == "" || m.APIVersion != APIVersion {
		return errors.New("invalid plugin manifest")
	}
	seen := make(map[Capability]bool, len(m.Capabilities))
	for _, capability := range m.Capabilities {
		if capability != Command && capability != Panel && capability != StatusWidget && capability != RepositoryRead {
			return fmt.Errorf("unsupported plugin capability %q", capability)
		}
		if seen[capability] {
			return fmt.Errorf("duplicate plugin capability %q", capability)
		}
		seen[capability] = true
	}
	if len(m.ConfigSchema) > MaxMessageBytes {
		return fmt.Errorf("plugin config schema exceeds %d bytes", MaxMessageBytes)
	}
	return nil
}

// Message is the newline-delimited JSON envelope exchanged by hosts and plugins.
type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Event describes a plugin lifecycle event with an optional opaque payload.
type Event struct {
	Lifecycle Lifecycle       `json:"lifecycle"`
	Name      string          `json:"name"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// CommandSpec describes a command extension exposed by a plugin.
type CommandSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// PanelSpec describes a panel extension exposed by a plugin.
type PanelSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// StatusWidgetSpec describes a status widget exposed by a plugin.
type StatusWidgetSpec struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// ConfigSchema contains the JSON-schema fields exposed by a plugin manifest.
type ConfigSchema struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

// HandshakeRequest is sent by the host to negotiate an API version and capability grant.
type HandshakeRequest struct {
	APIVersion   int          `json:"api_version"`
	Capabilities []Capability `json:"capabilities"`
}

// HandshakeResponse reports whether negotiation succeeded and which capabilities were accepted.
type HandshakeResponse struct {
	APIVersion   int          `json:"api_version"`
	Accepted     bool         `json:"accepted"`
	Capabilities []Capability `json:"capabilities"`
	Reason       string       `json:"reason,omitempty"`
}

// Negotiate validates a requested API version and capability set against host support.
func Negotiate(apiVersion int, requested, supported []Capability) HandshakeResponse {
	if apiVersion != APIVersion {
		return HandshakeResponse{APIVersion: APIVersion, Reason: fmt.Sprintf("unsupported plugin API version %d", apiVersion)}
	}
	allowed := make(map[Capability]bool, len(supported))
	for _, capability := range supported {
		allowed[capability] = true
	}
	granted := make([]Capability, 0, len(requested))
	for _, capability := range requested {
		if !allowed[capability] {
			return HandshakeResponse{APIVersion: APIVersion, Reason: fmt.Sprintf("capability %q is not supported", capability)}
		}
		granted = append(granted, capability)
	}
	return HandshakeResponse{APIVersion: APIVersion, Accepted: true, Capabilities: granted}
}

// Encode serializes a message as one newline-terminated JSON record.
func Encode(message Message) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Decode validates and decodes one JSON protocol message.
func Decode(data []byte) (Message, error) {
	if len(data) > MaxMessageBytes {
		return Message{}, fmt.Errorf("plugin message exceeds %d bytes", MaxMessageBytes)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return Message{}, errors.New("plugin message is empty")
	}
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return Message{}, err
	}
	if message.Type == "" {
		return Message{}, errors.New("plugin message type is required")
	}
	if len(message.Type) > MaxFieldBytes || len(message.ID) > MaxFieldBytes {
		return Message{}, fmt.Errorf("plugin message field exceeds %d bytes", MaxFieldBytes)
	}
	return message, nil
}
