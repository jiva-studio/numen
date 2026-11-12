package connection_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

const (
	openRouterID = "md.numen.connections.llm/openrouter"
	compatibleID = "md.numen.connections.llm/openai-compatible"
	retiredID    = "gone.plugin/x"
)

var errDisk = errors.New("the disk is full")

type memoryRepository struct {
	mu          sync.Mutex
	connections []connectiondomain.Connection
	failWrite   error
}

func (r *memoryRepository) ListConnections() ([]connectiondomain.Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.connections), nil
}

func (r *memoryRepository) ChangeConnections(change func([]connectiondomain.Connection) ([]connectiondomain.Connection, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	updated, err := change(slices.Clone(r.connections))
	if err != nil {
		return err
	}
	if r.failWrite != nil {
		return r.failWrite
	}
	r.connections = updated
	return nil
}

type fixedTypes struct{ types []connectiondomain.Type }

func (t fixedTypes) ListTypes() []connectiondomain.Type { return t.types }

func (t fixedTypes) GetType(typeID string) (connectiondomain.Type, bool) {
	for _, one := range t.types {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connectiondomain.Type{}, false
}

type memoryKeys struct {
	mu      sync.Mutex
	values  map[string]string
	failSet error
}

func (k *memoryKeys) GetKey(_ context.Context, name string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	value, found := k.values[name]
	if !found {
		return "", plugindomain.ErrNoKey
	}
	return value, nil
}

func (k *memoryKeys) SetKey(_ context.Context, name, key string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failSet != nil {
		return k.failSet
	}
	k.values[name] = key
	return nil
}

func (k *memoryKeys) DeleteKey(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, found := k.values[name]; !found {
		return plugindomain.ErrNoKey
	}
	delete(k.values, name)
	return nil
}

func (k *memoryKeys) listNames() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	names := make([]string, 0, len(k.values))
	for name := range k.values {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func newTypes() fixedTypes {
	fixed := "https://openrouter.ai/api/v1"
	return fixedTypes{types: []connectiondomain.Type{
		{PluginID: "md.numen.connections.llm", PluginName: "LLM connections", IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "openrouter", Title: "OpenRouter", Operations: []string{"complete", "embed"},
			Fields: []manifest.ConnectionField{
				{Key: "baseUrl", Type: "string", Title: "Address", IsAddress: true, FixedValue: &fixed},
				{Key: "apiKey", Type: "secret", Title: "API key", IsRequired: true},
				{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(4), Min: new(1.0), Max: new(32.0), Step: new(1.0)},
			},
		}},
		{PluginID: "md.numen.connections.llm", PluginName: "LLM connections", IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "openai-compatible", Title: "OpenAI-compatible server", Operations: []string{"complete", "embed"},
			Fields: []manifest.ConnectionField{
				{Key: "baseUrl", Type: "string", Title: "Address", IsAddress: true, IsRequired: true},
				{Key: "apiKey", Type: "secret", Title: "API key"},
				{Key: "mode", Type: "enum", Title: "Mode", Options: manifest.SettingOptions{{Value: "fast"}, {Value: "slow"}}},
				{Key: "stream", Type: "boolean", Title: "Stream"},
			},
		}},
	}}
}

type fixture struct {
	repository *memoryRepository
	keys       *memoryKeys
	uses       *memoryUses
	types      fixedTypes
	connection *connectiontest.Connections
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{repository: &memoryRepository{}, keys: &memoryKeys{values: map[string]string{}}, uses: &memoryUses{}, types: newTypes()}
	next := 0
	f.connection = connectiontest.New(f.repository, f.types, f.keys, f.uses, func() (string, error) {
		next++
		return fmt.Sprintf("c%08x", next), nil
	})
	return f
}

func (f *fixture) createOpenRouter(t *testing.T, name string) connection.Entry {
	t.Helper()
	entry, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Name: name, Secrets: map[string]string{"apiKey": "sk-or-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func (f *fixture) createCompatible(t *testing.T, name, address, key string) connection.Entry {
	t.Helper()
	request := connection.CreateRequest{TypeID: compatibleID, Name: name, Fields: map[string]any{"baseUrl": address}}
	if key != "" {
		request.Secrets = map[string]string{"apiKey": key}
	}
	entry, err := f.connection.Create.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func hasValueContaining(values map[string]string, part string) bool {
	for _, value := range values {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}

func (f *fixture) listStoredConnections(t *testing.T) []connectiondomain.Connection {
	t.Helper()
	stored, err := f.repository.ListConnections()
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func (f *fixture) listEntries(t *testing.T) []connection.Entry {
	t.Helper()
	entries, err := f.connection.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

// memoryUses stands for the settings of rows of the plugins: uses lists them by connection, and the
// calls the use cases make are recorded.
type memoryUses struct {
	mu          sync.Mutex
	uses        map[string][]connectiondomain.Use
	removed     []string
	refreshed   []string
	failRemove  error
	failRefresh error
}

func (u *memoryUses) ListUses(connectionID string) ([]connectiondomain.Use, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.uses[connectionID]), nil
}

func (u *memoryUses) RemoveRows(connectionID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.failRemove != nil {
		return u.failRemove
	}
	u.removed = append(u.removed, connectionID)
	delete(u.uses, connectionID)
	return nil
}

func (u *memoryUses) Refresh(_ context.Context, connectionID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.refreshed = append(u.refreshed, connectionID)
	return u.failRefresh
}

func (u *memoryUses) use(connectionID string, uses ...connectiondomain.Use) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.uses == nil {
		u.uses = map[string][]connectiondomain.Use{}
	}
	u.uses[connectionID] = uses
}
