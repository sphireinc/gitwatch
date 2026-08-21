// Package plugin contains the stable, dependency-free wire contract for
// gitwatch plugins. It intentionally imports no internal gitwatch packages.
package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const APIVersion = 1

const (
	MaxMessageBytes = 1 << 20
	MaxFieldBytes   = 256
)

const (
	MessageHandshake = "handshake"
	MessageEvent     = "event"
	MessageCommand   = "command"
	MessagePanel     = "panel"
	MessageWidget    = "status_widget"
)

type Lifecycle string

const (
	LifecycleStart   Lifecycle = "start"
	LifecycleStop    Lifecycle = "stop"
	LifecycleFailure Lifecycle = "failure"
)

type Capability string

const (
	Command        Capability = "command"
	Panel          Capability = "panel"
	StatusWidget   Capability = "status_widget"
	RepositoryRead Capability = "repository_read"
)

type Manifest struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	APIVersion   int             `json:"api_version"`
	Capabilities []Capability    `json:"capabilities"`
	ConfigSchema json.RawMessage `json:"config_schema,omitempty"`
}

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

type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Event struct {
	Lifecycle Lifecycle       `json:"lifecycle"`
	Name      string          `json:"name"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type CommandSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type PanelSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type StatusWidgetSpec struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type ConfigSchema struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

type HandshakeRequest struct {
	APIVersion   int          `json:"api_version"`
	Capabilities []Capability `json:"capabilities"`
}

type HandshakeResponse struct {
	APIVersion   int          `json:"api_version"`
	Accepted     bool         `json:"accepted"`
	Capabilities []Capability `json:"capabilities"`
	Reason       string       `json:"reason,omitempty"`
}

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

func Encode(message Message) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

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
