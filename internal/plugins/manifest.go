package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const APIVersion = 1
const APIVersion2 = 2

type Capability string

const (
	CapabilityCommand        Capability = "command"
	CapabilityPanel          Capability = "panel"
	CapabilityStatusWidget   Capability = "status_widget"
	CapabilityRepositoryRead Capability = "repository_read"
	CapabilityContextAction  Capability = "context_action"
	CapabilityTable          Capability = "table"
	CapabilityDetail         Capability = "detail"
	CapabilityNotification   Capability = "notification"
	CapabilityRepositoryMeta Capability = "repository_metadata"
	CapabilityProcess        Capability = "process"
	CapabilityNetwork        Capability = "network"
	CapabilityGitMutation    Capability = "git_mutation"
)

// DefaultCapabilities is the host's schema-only capability surface. A plugin
// still receives only the subset it declares and the negotiated response
// confirms.
var DefaultCapabilities = []Capability{
	CapabilityCommand, CapabilityPanel, CapabilityStatusWidget, CapabilityRepositoryRead,
	CapabilityContextAction, CapabilityTable, CapabilityDetail, CapabilityNotification,
	CapabilityRepositoryMeta, CapabilityProcess, CapabilityNetwork, CapabilityGitMutation,
}

type Manifest struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	APIVersion   int          `json:"api_version"`
	Executable   string       `json:"executable"`
	Capabilities []Capability `json:"capabilities"`
}

func (m Manifest) Validate() error {
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`).MatchString(m.ID) {
		return errors.New("plugin id must use lowercase identifier characters")
	}
	if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Version) == "" {
		return errors.New("plugin name and version are required")
	}
	if m.APIVersion != APIVersion && m.APIVersion != APIVersion2 {
		return fmt.Errorf("unsupported plugin API version %d", m.APIVersion)
	}
	if strings.TrimSpace(m.Executable) == "" {
		return errors.New("plugin executable is required")
	}
	seen := make(map[Capability]bool)
	for _, capability := range m.Capabilities {
		if seen[capability] {
			return fmt.Errorf("duplicate plugin capability %q", capability)
		}
		seen[capability] = true
	}
	return nil
}

func DecodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

type Negotiation struct {
	APIVersion   int
	Accepted     bool
	Capabilities []Capability
	Reason       string
	Output       []byte
}

func Negotiate(manifest Manifest, supported []Capability) Negotiation {
	if err := manifest.Validate(); err != nil {
		return Negotiation{Reason: err.Error()}
	}
	allowed := make(map[Capability]bool, len(supported))
	for _, capability := range supported {
		allowed[capability] = true
	}
	var capabilities []Capability
	for _, capability := range manifest.Capabilities {
		if allowed[capability] {
			capabilities = append(capabilities, capability)
		}
	}
	return Negotiation{APIVersion: manifest.APIVersion, Accepted: true, Capabilities: capabilities}
}
