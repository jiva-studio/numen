// Package settings keeps the configuration the host last passed to the plugin.
package settings

import (
	"sync"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/usecase/proofread"
)

// Store is how a text is proofread, as the host last passed it.
type Store struct {
	mu  sync.RWMutex
	run proofread.Config
}

// NewStore returns a store that holds the defaults.
func NewStore() *Store {
	return &Store{mu: sync.RWMutex{}, run: proofread.DefaultConfig()}
}

// Update takes the values the host passes.
func (s *Store) Update(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.run = proofread.ParseConfig(values, s.run)
}

// GetRunConfig is how a text is proofread.
func (s *Store) GetRunConfig() proofread.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.run
}
