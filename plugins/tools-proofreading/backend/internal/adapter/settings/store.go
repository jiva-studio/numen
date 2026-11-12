// Package settings keeps the configuration the host last passed to the plugin.
package settings

import (
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/endpoint"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/usecase/proofread"
)

// Store is how the model is reached and how a text is proofread, as the host last passed it.
type Store struct {
	mu       sync.RWMutex
	endpoint endpoint.Endpoint
	run      proofread.Config
}

// NewStore returns a store that holds the defaults.
func NewStore() *Store {
	return &Store{mu: sync.RWMutex{}, endpoint: endpoint.GetDefault(), run: proofread.DefaultConfig()}
}

// Update takes the values and the secrets the host passes.
func (s *Store) Update(values map[string]string, secrets server.Secrets) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoint = s.endpoint.WithSettings(values)
	if list, carried := server.ReadSecrets(secrets, server.SecretAPIKey); carried {
		s.endpoint.APIKey = ""
		if len(list) > 0 {
			s.endpoint.APIKey = list[0].Value
		}
	}
	s.run = proofread.ParseConfig(values, s.run)
}

// GetEndpoint is the HTTP chat service the configured provider points at.
func (s *Store) GetEndpoint() endpoint.Endpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endpoint
}

// GetRunConfig is how a text is proofread.
func (s *Store) GetRunConfig() proofread.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.run
}
