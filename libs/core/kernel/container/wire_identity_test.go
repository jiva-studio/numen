package container

import (
	"context"
	"sync"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

type memoryKeys map[string]string

func (m memoryKeys) GetKey(_ context.Context, name string) (string, error) {
	if value, ok := m[name]; ok {
		return value, nil
	}
	return "", plugindomain.ErrNoKey
}

func (m memoryKeys) SetKey(_ context.Context, name, key string) error { m[name] = key; return nil }

func (m memoryKeys) DeleteKey(_ context.Context, name string) error {
	if _, ok := m[name]; !ok {
		return plugindomain.ErrNoKey
	}
	delete(m, name)
	return nil
}

func TestOpenIdentityUseCases_ServesNothingWhereTheInstallationHasNoKeyStore(t *testing.T) {
	t.Parallel()
	if got := openIdentityUseCases(pluginInstall{}, vaultWiring{Clock: testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))}, &Resources{}); got != nil {
		t.Errorf("openIdentityUseCases() = %+v, want nil", got)
	}
}

func TestOpenIdentityUseCases_KeepsAnIdentityInTheKeyStoreOfTheInstallation(t *testing.T) {
	t.Parallel()
	keys := memoryKeys{}
	res := &Resources{ChainStore: &memoryChain{}, progress: identity.NewProgress()}
	use := openIdentityUseCases(pluginInstall{KeyStore: keys}, vaultWiring{Clock: testsupport.NewClockAt(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))}, res)
	if use == nil || use.Get == nil || use.GenerateWords == nil || use.Create == nil || use.Delete == nil {
		t.Fatalf("openIdentityUseCases() = %+v, want all four use cases", use)
	}
	words, err := use.GenerateWords.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	created, err := use.Create.Execute(t.Context(), words)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := keys["identity"]; !ok {
		t.Errorf("the key store holds %v, want the entry \"identity\"", keys)
	}
	if created.DeviceName == "" {
		t.Error("the device has no name: the licensing adapter is not wired as the name of the device")
	}
}

// memoryChain is the chain file of a device kept in memory.
type memoryChain struct {
	mu    sync.Mutex
	chain []byte
}

func (m *memoryChain) Read(context.Context) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chain == nil {
		return nil, identitydomain.ErrNoChain
	}
	return append([]byte(nil), m.chain...), nil
}

func (m *memoryChain) Write(_ context.Context, chain []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chain = append([]byte(nil), chain...)
	return nil
}

func (m *memoryChain) Delete(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chain = nil
	return nil
}
