// Package localsource lists the plugins in the plugin folders of this machine, the disabled ones too.
package localsource

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

// Name is the name of the source in its entries.
const Name = "local"

// Settings answers which plugins are switched on.
type Settings interface {
	// IsEnabled reports whether the settings of the plugin leave it enabled.
	IsEnabled(pluginID string) bool
}

// SettingsFunc is a function that answers as Settings does.
type SettingsFunc func(pluginID string) bool

// IsEnabled calls the function.
func (f SettingsFunc) IsEnabled(pluginID string) bool { return f(pluginID) }

// Source reads every plugin folder under the directories.
type Source struct {
	dirs     []string
	settings Settings
}

// New returns the source over the plugin directories. The enabled flag of each plugin comes from
// the settings.
func New(dirs []string, settings Settings) *Source {
	return &Source{dirs: dirs, settings: settings}
}

// Name is Name.
func (s *Source) Name() string { return Name }

// List returns an entry for each valid plugin found. A folder that cannot be read is reported in
// the error, which joins one error for each, and does not hide the plugins beside it.
func (s *Source) List(ctx context.Context) ([]catalog.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	found, problems := manifestjson.Discover(s.dirs)
	entries := make([]catalog.Entry, 0, len(found))
	for _, plugin := range found {
		state := catalog.StateInstalled
		if !s.settings.IsEnabled(plugin.Manifest.ID) {
			state = catalog.StateDisabled
		}
		entries = append(entries, catalog.NewEntry(Name, state, plugin.Manifest))
	}
	failures := make([]error, 0, len(problems))
	for _, problem := range problems {
		failures = append(failures, fmt.Errorf("%s: %w", problem.Dir, problem.Err))
	}
	return entries, errors.Join(failures...)
}

// Get returns the entry of the plugin. A plugin that is not in the folders is catalog.ErrNotFound.
func (s *Source) Get(ctx context.Context, id string) (catalog.Entry, error) {
	entries, err := s.List(ctx)
	for _, entry := range entries {
		if entry.ID == id {
			return entry, err
		}
	}
	return catalog.Entry{}, errors.Join(fmt.Errorf("plugin %q: %w", id, catalog.ErrNotFound), err)
}
