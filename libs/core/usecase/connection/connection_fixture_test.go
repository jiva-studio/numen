package connection_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const (
	openRouterID = "md.numen.connections.llm/openrouter"
	compatibleID = "md.numen.connections.llm/openai-compatible"
	retiredID    = "gone.plugin/x"
)

var errDisk = errors.New("the disk is full")

type memoryRepository struct {
	mu          sync.Mutex
	connections []connectionport.Connection
	failWrite   error
}

func (r *memoryRepository) ListConnections() ([]connectionport.Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.connections), nil
}

func (r *memoryRepository) ChangeConnections(change func([]connectionport.Connection) ([]connectionport.Connection, error)) error {
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

type fixedTypes struct{ types []connectionport.Type }

func (t fixedTypes) ListTypes() []connectionport.Type { return t.types }

func (t fixedTypes) GetType(typeID string) (connectionport.Type, bool) {
	for _, one := range t.types {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connectionport.Type{}, false
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
		return "", pluginport.ErrNoKey
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
		return pluginport.ErrNoKey
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

func number(value float64) *float64 { return &value }

func newTypes() fixedTypes {
	fixed := "https://openrouter.ai/api/v1"
	return fixedTypes{types: []connectionport.Type{
		{PluginID: "md.numen.connections.llm", PluginName: "LLM connections", IsAvailable: true, Declaration: manifest.ConnectionType{
			ID: "openrouter", Title: "OpenRouter", Operations: []string{"complete", "embed"},
			Fields: []manifest.ConnectionField{
				{Key: "baseUrl", Type: "string", Title: "Address", IsAddress: true, FixedValue: &fixed},
				{Key: "apiKey", Type: "secret", Title: "API key", IsRequired: true},
				{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(4), Min: number(1), Max: number(32), Step: number(1)},
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
	types      fixedTypes
	connection *connection.Connections
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{repository: &memoryRepository{}, keys: &memoryKeys{values: map[string]string{}}, types: newTypes()}
	next := 0
	f.connection = connection.NewConnections(f.repository, f.types, f.keys, func() (string, error) {
		next++
		return fmt.Sprintf("c%08x", next), nil
	})
	return f
}

func (f *fixture) createOpenRouter(t *testing.T, name string) connection.Entry {
	t.Helper()
	entry, err := f.connection.Create.Execute(context.Background(), connection.CreateRequest{
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
	entry, err := f.connection.Create.Execute(context.Background(), request)
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
