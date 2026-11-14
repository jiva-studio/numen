// Package settings keeps the configuration the host last passed to the plugin.
package settings

import (
	"sync"

	domainsettings "github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/settings"
)

// Store is how a text is proofread, as the host last passed it.
type Store struct {
	mu  sync.RWMutex
	run domainsettings.Config
}

// NewStore returns a store that holds the defaults.
func NewStore() *Store {
	return &Store{mu: sync.RWMutex{}, run: domainsettings.DefaultConfig()}
}

// Update takes the values the host passes.
func (s *Store) Update(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.run = domainsettings.ParseConfig(values, s.run)
}

// GetRunConfig is how a text is proofread.
func (s *Store) GetRunConfig() domainsettings.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.run
}
