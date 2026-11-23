package identity_test

import (
	"context"
	"errors"
	"maps"
	"sync"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
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
}

func newFakeKeyStore() *fakeKeyStore { return &fakeKeyStore{keys: make(map[string]string)} }

func (s *fakeKeyStore) GetKey(_ context.Context, name string) (string, error) {
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

// fakeKitSaver stands where the person chooses a file: it keeps every kit it is given, or answers
// that no file was chosen, or fails.
type fakeKitSaver struct {
	mu        sync.Mutex
	kits      []identitydomain.Kit
	isRefused bool
	fails     error
}

func (s *fakeKitSaver) SaveKit(_ context.Context, kit identitydomain.Kit) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fails != nil {
		return false, s.fails
	}
	if s.isRefused {
		return false, nil
	}
	s.kits = append(s.kits, kit)
	return true, nil
}

func (s *fakeKitSaver) getLast() (identitydomain.Kit, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.kits) == 0 {
		return identitydomain.Kit{}, false
	}
	return s.kits[len(s.kits)-1], true
}
