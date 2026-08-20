// Package plugin contains the stable, dependency-free wire contract for
// gitwatch plugins. It intentionally imports no internal gitwatch packages.
package plugin

import (
	"encoding/json"
	"errors"
)

const APIVersion = 1

type Capability string

const (
	Command        Capability = "command"
	Panel          Capability = "panel"
	StatusWidget   Capability = "status_widget"
	RepositoryRead Capability = "repository_read"
)

type Manifest struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	APIVersion   int          `json:"api_version"`
	Capabilities []Capability `json:"capabilities"`
}

func (m Manifest) Validate() error {
	if m.ID == "" || m.Name == "" || m.Version == "" || m.APIVersion != APIVersion {
		return errors.New("invalid plugin manifest")
	}
	return nil
}

type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func Encode(message Message) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Decode(data []byte) (Message, error) {
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return Message{}, err
	}
	if message.Type == "" {
		return Message{}, errors.New("plugin message type is required")
	}
	return message, nil
}
