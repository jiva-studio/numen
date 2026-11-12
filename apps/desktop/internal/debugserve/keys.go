//go:build numen_debug

package debugserve

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// KeyStore keeps the keys of plugin secrets in memory for the life of the process. A test server
// keeps its keys here, and the keychain of the machine is never read or written.
type KeyStore struct {
	mu   sync.Mutex
	keys map[string]string
}

// NewKeyStore returns a KeyStore that holds no key.
func NewKeyStore() *KeyStore {
	return &KeyStore{keys: map[string]string{}}
}

// GetKey returns the key stored under the name.
func (s *KeyStore) GetKey(_ context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[name]
	if !ok {
		return "", fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
	}
	return key, nil
}

// SetKey stores the key under the name and replaces the one already there.
func (s *KeyStore) SetKey(_ context.Context, name, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[name] = key
	return nil
}

// DeleteKey removes the key stored under the name.
func (s *KeyStore) DeleteKey(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[name]; !ok {
		return fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
	}
	delete(s.keys, name)
	return nil
}
