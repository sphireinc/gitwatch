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

// APIVersion2 is the first extensible contribution protocol. API-1 remains
// frozen; clients should advertise versions they understand and use the
// highest version selected by the host.
const APIVersion2 = 2

var SupportedAPIVersions = []int{APIVersion, APIVersion2}

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
	// MessageContribution carries schema-defined actions and data extensions.
	MessageContribution = "contribution"
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
	// ContextAction allows a plugin to register a host-rendered contextual action.
	ContextAction Capability = "context_action"
	// TableContribution allows a plugin to provide bounded tabular data.
	TableContribution Capability = "table"
	// DetailContribution allows a plugin to provide bounded detail fields.
	DetailContribution Capability = "detail"
	// Notification allows a plugin to request a host-rendered notification.
	Notification Capability = "notification"
	// RepositoryMetadata allows read-only repository metadata extensions.
	RepositoryMetadata Capability = "repository_metadata"
	// Process, Network, and GitMutation are explicit opt-in permissions.
	Process     Capability = "process"
	Network     Capability = "network"
	GitMutation Capability = "git_mutation"
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
	if m.ID == "" || m.Name == "" || m.Version == "" || (m.APIVersion != APIVersion && m.APIVersion != APIVersion2) {
		return errors.New("invalid plugin manifest")
	}
	seen := make(map[Capability]bool, len(m.Capabilities))
	for _, capability := range m.Capabilities {
		if !knownCapability(capability) {
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

func knownCapability(capability Capability) bool {
	switch capability {
	case Command, Panel, StatusWidget, RepositoryRead, ContextAction,
		TableContribution, DetailContribution, Notification, RepositoryMetadata,
		Process, Network, GitMutation:
		return true
	default:
		return false
	}
}

// Message is the newline-delimited JSON envelope exchanged by hosts and plugins.
type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewHandshake creates the API-1 handshake request used by plugin hosts.
func NewHandshake(capabilities []Capability) Message {
	payload, _ := json.Marshal(HandshakeRequest{APIVersion: APIVersion, Capabilities: capabilities})
	return Message{Type: MessageHandshake, Payload: payload}
}

// NewHandshakeVersions advertises an ordered set of versions for negotiation.
func NewHandshakeVersions(versions []int, capabilities []Capability) Message {
	payload, _ := json.Marshal(HandshakeRequest{APIVersion: APIVersion, Versions: versions, Capabilities: capabilities})
	return Message{Type: MessageHandshake, Payload: payload}
}

// NewEvent creates a lifecycle/event message with a bounded opaque payload.
func NewEvent(id string, event Event) (Message, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return Message{}, err
	}
	return Message{Type: MessageEvent, ID: id, Payload: payload}, nil
}

// NewCommand, NewPanel, and NewWidget create public extension messages.
func NewCommand(id string, spec CommandSpec) (Message, error) {
	return newPayloadMessage(MessageCommand, id, spec)
}
func NewPanel(id string, spec PanelSpec) (Message, error) {
	return newPayloadMessage(MessagePanel, id, spec)
}
func NewWidget(id string, spec StatusWidgetSpec) (Message, error) {
	return newPayloadMessage(MessageWidget, id, spec)
}

// NewContribution creates a schema-defined API-2 contribution message. The
// host renders this data; plugins never provide executable UI code.
func NewContribution(id string, contribution Contribution) (Message, error) {
	if err := contribution.Validate(); err != nil {
		return Message{}, err
	}
	return newPayloadMessage(MessageContribution, id, contribution)
}

// WireError is a structured, non-sensitive plugin error payload.
type WireError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newPayloadMessage(kind, id string, payload any) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{Type: kind, ID: id, Payload: data}, nil
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

type ActionSpec struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context,omitempty"`
	ReadOnly    bool   `json:"read_only"`
}

type TableColumn struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type TableRow map[string]string

// Contribution is deliberately data-only. Bounds prevent a plugin from
// turning a single message into an unbounded UI or memory workload.
type Contribution struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	Title         string            `json:"title"`
	Description   string            `json:"description,omitempty"`
	Action        *ActionSpec       `json:"action,omitempty"`
	Columns       []TableColumn     `json:"columns,omitempty"`
	Rows          []TableRow        `json:"rows,omitempty"`
	Fields        map[string]string `json:"fields,omitempty"`
	ReadOnly      bool              `json:"read_only"`
}

const (
	MaxContributionRows   = 256
	MaxContributionFields = 64
	MaxContributionText   = 4096
)

func (c Contribution) Validate() error {
	if c.SchemaVersion < 1 || c.SchemaVersion > APIVersion2 || c.Kind == "" || c.Title == "" {
		return errors.New("invalid plugin contribution")
	}
	if len(c.Title) > MaxContributionText || len(c.Description) > MaxContributionText {
		return fmt.Errorf("plugin contribution text exceeds %d bytes", MaxContributionText)
	}
	if len(c.Rows) > MaxContributionRows || len(c.Fields) > MaxContributionFields {
		return errors.New("plugin contribution exceeds collection limits")
	}
	if len(c.Columns) > MaxContributionFields {
		return errors.New("plugin contribution has too many columns")
	}
	for _, value := range append([]string{c.Title, c.Description}, contributionValues(c)...) {
		if hasControl(value) {
			return errors.New("plugin contribution contains terminal control data")
		}
	}
	return nil
}

func contributionValues(c Contribution) []string {
	values := make([]string, 0, len(c.Fields)+len(c.Rows)*len(c.Columns))
	for _, column := range c.Columns {
		values = append(values, column.ID, column.Title)
	}
	for key, value := range c.Fields {
		values = append(values, key, value)
	}
	for _, row := range c.Rows {
		for key, value := range row {
			values = append(values, key, value)
		}
	}
	if c.Action != nil {
		values = append(values, c.Action.ID, c.Action.Title, c.Action.Description, c.Action.Context)
	}
	return values
}

func hasControl(value string) bool {
	for _, r := range value {
		if r < 0x20 && r != '\t' && r != '\n' {
			return true
		}
		if r == 0x7f || r == 0x1b {
			return true
		}
	}
	return false
}

// ConfigSchema contains the JSON-schema fields exposed by a plugin manifest.
type ConfigSchema struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

// HandshakeRequest is sent by the host to negotiate an API version and capability grant.
type HandshakeRequest struct {
	APIVersion   int          `json:"api_version"`
	Versions     []int        `json:"versions,omitempty"`
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

// NegotiateVersions selects the highest mutually supported version. Unknown
// capabilities are intentionally degraded away for versioned negotiation;
// the legacy Negotiate function retains API-1's strict behavior.
func NegotiateVersions(requestedVersions []int, requested, supported []Capability) HandshakeResponse {
	version := 0
	for _, requestedVersion := range requestedVersions {
		for _, supportedVersion := range SupportedAPIVersions {
			if requestedVersion == supportedVersion && requestedVersion > version {
				version = requestedVersion
			}
		}
	}
	if version == 0 {
		return HandshakeResponse{APIVersion: APIVersion, Reason: "no mutually supported plugin API version"}
	}
	allowed := make(map[Capability]bool, len(supported))
	for _, capability := range supported {
		if knownCapability(capability) {
			allowed[capability] = true
		}
	}
	granted := make([]Capability, 0, len(requested))
	seen := make(map[Capability]bool)
	for _, capability := range requested {
		if allowed[capability] && !seen[capability] {
			granted = append(granted, capability)
			seen[capability] = true
		}
	}
	return HandshakeResponse{APIVersion: version, Accepted: true, Capabilities: granted}
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
