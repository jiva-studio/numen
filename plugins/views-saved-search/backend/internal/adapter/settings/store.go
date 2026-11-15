// Package settings keeps the configuration the host last passed to the plugin.
package settings

import (
	"sync"

	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

// Store is the numbers of a preview, as the host last passed them.
type Store struct {
	mu     sync.RWMutex
	config preview.Config
}

// NewStore returns a store that holds the defaults.
func NewStore() *Store {
	return &Store{mu: sync.RWMutex{}, config: preview.DefaultConfig()}
}

// Update takes the values the host passes.
func (s *Store) Update(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = preview.ParseConfig(values, s.config)
}

// GetConfig returns the defaults until the host passes values.
func (s *Store) GetConfig() preview.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}
