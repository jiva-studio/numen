// Package settings keeps the configuration the host last passed to the plugin.
package settings

import (
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/preview"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/refresh"
)

const (
	refreshMinutesKey     = "refreshMinutes"
	defaultFolderKey      = "defaultFolder"
	defaultRefreshMinutes = 10
)

// Store is the settings of the plugin, as the host last passed them.
type Store struct {
	mu       sync.RWMutex
	settings refresh.Settings
	folder   string
}

// NewStore returns a store that holds the defaults.
func NewStore() *Store {
	return &Store{
		mu:       sync.RWMutex{},
		settings: refresh.Settings{Preview: preview.DefaultConfig(), RefreshMinutes: defaultRefreshMinutes},
		folder:   "",
	}
}

// Update takes the values the host passes. A refresh delay that is not a whole number of zero or more
// keeps the value before.
func (s *Store) Update(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.Preview = preview.ParseConfig(values, s.settings.Preview)
	if value, isSet := values[refreshMinutesKey]; isSet {
		if minutes, err := strconv.Atoi(value); err == nil && minutes >= 0 {
			s.settings.RefreshMinutes = minutes
		}
	}
	if folder, isSet := values[defaultFolderKey]; isSet {
		s.folder = folder
	}
}

func (s *Store) GetConfig() preview.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Preview
}

// GetSettings returns the numbers of a preview and the refresh delay.
func (s *Store) GetSettings() refresh.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// GetDefaultFolder returns the folder a saved search is created in when the call names none; empty is the
// vault folder.
func (s *Store) GetDefaultFolder() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.folder
}
