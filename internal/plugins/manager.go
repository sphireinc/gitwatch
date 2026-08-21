package plugins

import (
	"context"
	"encoding/json"
	"errors"
	publicplugin "github.com/sphireinc/git-watch/pkg/plugin"
	"io/fs"
	"os"
	"path/filepath"
)

type Entry struct {
	Manifest Manifest
	Path     string
	Enabled  bool
	Healthy  bool
	Error    string
	Commands []publicplugin.CommandSpec
	Panels   []publicplugin.PanelSpec
	Widgets  []publicplugin.StatusWidgetSpec
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
