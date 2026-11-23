package identity_test

import (
	"context"
	"errors"
	"maps"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// fakeKeyStore is the key store of the machine kept in memory, with the answers of the real one:
// a missing key is plugin.ErrNoKey.
type fakeKeyStore struct {
	mu        sync.Mutex
	keys      map[string]string
	getFails  error
	setFails  error
	setCalled int
	onGet     func()
}

func newFakeKeyStore() *fakeKeyStore { return &fakeKeyStore{keys: make(map[string]string)} }

func (s *fakeKeyStore) GetKey(_ context.Context, name string) (string, error) {
	if s.onGet != nil {
		defer s.onGet()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getFails != nil {
		return "", s.getFails
	}
	value, ok := s.keys[name]
	if !ok {
		return "", plugindomain.ErrNoKey
	}
	return value, nil
}

func (s *fakeKeyStore) SetKey(_ context.Context, name, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCalled++
	if s.setFails != nil {
		return s.setFails
	}
	s.keys[name] = key
	return nil
}

func (s *fakeKeyStore) DeleteKey(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[name]; !ok {
		return plugindomain.ErrNoKey
	}
	delete(s.keys, name)
	return nil
}

func (s *fakeKeyStore) snapshot() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.keys))
	maps.Copy(out, s.keys)
	return out
}

type fakeNames struct{ name string }

func (n fakeNames) GetInstanceName() string { return n.name }

var errBroken = errors.New("the keychain refuses")
