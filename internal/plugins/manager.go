package plugins

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	publicplugin "github.com/sphireinc/git-watch/pkg/plugin"
	"io/fs"
	"os"
	"path/filepath"
)

type Entry struct {
	Manifest            Manifest
	Path                string
	Enabled             bool
	Healthy             bool
	Error               string
	Commands            []publicplugin.CommandSpec
	Panels              []publicplugin.PanelSpec
	Widgets             []publicplugin.StatusWidgetSpec
	Contributions       []publicplugin.Contribution
	GrantedCapabilities []Capability
}

// Probe performs the opt-in process handshake and records only bounded,
// schema-defined output for the host UI.
func Probe(ctx context.Context, host Runtime, entry Entry, supported []Capability) Entry {
	entry.Contributions = nil
	entry.GrantedCapabilities = nil
	if !entry.Enabled || !entry.Healthy {
		return entry
	}
	negotiation, err := host.Handshake(ctx, entry.Manifest, supported)
	if err != nil {
		entry.Healthy = false
		entry.Error = err.Error()
		return entry
	}
	if !negotiation.Accepted {
		entry.Healthy = false
		entry.Error = negotiation.Reason
		return entry
	}
	contributions, decodeErr := DecodeContributions(negotiation.Output, 32)
	if decodeErr != nil {
		entry.Healthy = false
		entry.Error = decodeErr.Error()
		return entry
	}
	entry.GrantedCapabilities = append([]Capability(nil), negotiation.Capabilities...)
	entry.Contributions = filterContributions(contributions, entry.GrantedCapabilities)
	return entry
}

// CanRunMetadataAction reports whether a contribution requests the one
// host-owned provider action currently implemented by gitwatch. The provider
// always uses the active repository and host credentials; plugins supply
// neither URLs nor tokens.
func CanRunMetadataAction(entry Entry, contribution publicplugin.Contribution) bool {
	if !entry.Enabled || !entry.Healthy || entry.Manifest.APIVersion != publicplugin.APIVersion2 {
		return false
	}
	if contribution.SchemaVersion != publicplugin.APIVersion2 || contribution.Kind != "repository_metadata" || !contribution.ReadOnly || contribution.Action == nil {
		return false
	}
	if !contribution.Action.ReadOnly || contribution.Action.Provider != publicplugin.ActionProviderGitHubRepository {
		return false
	}
	return hasCapability(entry.GrantedCapabilities, CapabilityContextAction) && hasCapability(entry.GrantedCapabilities, CapabilityRepositoryMeta)
}

func filterContributions(contributions []publicplugin.Contribution, granted []Capability) []publicplugin.Contribution {
	filtered := make([]publicplugin.Contribution, 0, len(contributions))
	for _, contribution := range contributions {
		if !contribution.ReadOnly {
			continue
		}
		var required []Capability
		switch contribution.Kind {
		case "table":
			required = append(required, CapabilityTable)
		case "detail":
			required = append(required, CapabilityDetail)
		case "notification":
			required = append(required, CapabilityNotification)
		case "repository_metadata":
			required = append(required, CapabilityRepositoryMeta)
		default:
			continue
		}
		if contribution.Action != nil {
			if contribution.Kind != "repository_metadata" || !contribution.Action.ReadOnly {
				continue
			}
			required = append(required, CapabilityContextAction)
		}
		allowed := true
		for _, capability := range required {
			if !hasCapability(granted, capability) {
				allowed = false
				break
			}
		}
		if allowed {
			filtered = append(filtered, contribution)
		}
	}
	return filtered
}

func hasCapability(capabilities []Capability, target Capability) bool {
	for _, capability := range capabilities {
		if capability == target {
			return true
		}
	}
	return false
}

// DecodeContributions extracts bounded, schema-defined contributions from a
// plugin's newline-delimited output. Unknown message types remain ignorable so
// older hosts can safely consume mixed-version plugin output.
func DecodeContributions(data []byte, max int) ([]publicplugin.Contribution, error) {
	if max <= 0 || max > publicplugin.MaxContributionRows {
		max = 32
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), publicplugin.MaxMessageBytes)
	contributions := make([]publicplugin.Contribution, 0)
	for scanner.Scan() {
		message, err := publicplugin.Decode(append(scanner.Bytes(), '\n'))
		if err != nil || message.Type != publicplugin.MessageContribution {
			continue
		}
		var contribution publicplugin.Contribution
		if err := json.Unmarshal(message.Payload, &contribution); err != nil {
			continue
		}
		if err := contribution.Validate(); err != nil {
			continue
		}
		contributions = append(contributions, contribution)
		if len(contributions) >= max {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return contributions, nil
}

func Discover(ctx context.Context, directories []string, max int) ([]Entry, error) {
	if max <= 0 {
		max = 128
	}
	entries := make([]Entry, 0)
	for _, directory := range directories {
		err := filepath.WalkDir(directory, func(path string, item fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if len(entries) >= max {
				return filepath.SkipAll
			}
			if item.Type()&os.ModeSymlink != 0 {
				if item.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if item.IsDir() || item.Name() != "manifest.json" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			manifest, err := DecodeManifest(data)
			entry := Entry{Path: filepath.Dir(path), Enabled: true, Healthy: err == nil}
			if err != nil {
				entry.Error = err.Error()
			} else {
				entry.Manifest = manifest
			}
			entries = append(entries, entry)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func SetEnabled(entries []Entry, id string, enabled bool) []Entry {
	updated := append([]Entry(nil), entries...)
	for i := range updated {
		if updated[i].Manifest.ID == id {
			updated[i].Enabled = enabled
		}
	}
	return updated
}

func StatePath() (string, error) {
	if path := os.Getenv("GITWATCH_PLUGIN_STATE"); path != "" {
		return path, nil
	}
	if root := os.Getenv("XDG_CONFIG_HOME"); root != "" {
		return filepath.Join(root, "gitwatch", "plugins.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gitwatch", "plugins.json"), nil
}

func LoadState(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	state := make(map[string]bool)
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func ApplyState(entries []Entry, state map[string]bool) []Entry {
	updated := append([]Entry(nil), entries...)
	for i := range updated {
		if enabled, ok := state[updated[i].Manifest.ID]; ok {
			updated[i].Enabled = enabled
		}
	}
	return updated
}

func SaveState(path string, entries []Entry) (returnErr error) {
	state := make(map[string]bool)
	for _, entry := range entries {
		if entry.Manifest.ID != "" {
			state[entry.Manifest.ID] = entry.Enabled
		}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".plugins-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer func() {
		if err := os.Remove(temporaryName); err != nil && !errors.Is(err, os.ErrNotExist) && returnErr == nil {
			returnErr = err
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func ValidateEntry(entry Entry) error {
	if entry.Manifest.ID == "" {
		return errors.New("plugin manifest is unavailable")
	}
	return entry.Manifest.Validate()
}
