package settings_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"
	usecases "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type mapKeys struct {
	mu     sync.Mutex
	values map[string]string
}

func (k *mapKeys) GetKey(_ context.Context, name string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	value, found := k.values[name]
	if !found {
		return "", plugin.ErrNoKey
	}
	return value, nil
}

func (k *mapKeys) SetKey(_ context.Context, name, key string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.values[name] = key
	return nil
}

func (k *mapKeys) DeleteKey(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.values, name)
	return nil
}

type oneType struct{}

func (oneType) ListTypes() []connection.Type { return []connection.Type{keychainType()} }

func (oneType) GetType(typeID string) (connection.Type, bool) {
	if typeID == keychainType().GetID() {
		return keychainType(), true
	}
	return connection.Type{}, false
}

func keychainType() connection.Type {
	return connection.Type{PluginID: "md.numen.connections.llm", PluginName: "LLM", IsAvailable: true, Declaration: manifest.ConnectionType{
		ID: "openai-compatible", Title: "OpenAI-compatible server", Operations: []string{"complete"},
		Fields: []manifest.ConnectionField{
			{Key: "baseUrl", Type: "string", Title: "Address", IsAddress: true, IsRequired: true},
			{Key: "apiKey", Type: "secret", Title: "API key"},
		},
	}}
}

func TestTheKeyOfAConnectionIsInTheKeyStoreAndNotInTheFile(t *testing.T) {
	path := write(t, `{}`)
	keys := &mapKeys{values: map[string]string{}}
	repository := settings.NewConnectionRepository(settings.NewFiles(),
		func() (string, error) { return path, nil },
		func() (settings.File, error) { return openAt(path) },
		func() {})
	next := 0
	no := settings.NewConnectionRows(settings.ConnectionRowsSource{
		Connections: repository, Plugins: func() []plugin.Info { return nil },
	})
	connections := connectiontest.New(repository, oneType{}, keys, no, func() (string, error) {
		next++
		return "c0000000" + string(rune('0'+next)), nil
	})
	const key = "sk-secret-value-123"

	created, err := connections.Create.Execute(t.Context(), usecases.CreateRequest{
		TypeID: keychainType().GetID(), Name: "Server",
		Fields:  map[string]any{"baseUrl": "https://api.example.com/v1"},
		Secrets: map[string]string{"apiKey": key},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connections.Update.Execute(t.Context(), usecases.UpdateRequest{
		ID: created.ID, Revision: created.Revision, Name: "Server",
		Fields:  map[string]any{"baseUrl": "https://api.example.com/v1"},
		Secrets: map[string]string{"apiKey": key + "-renewed"},
	}); err != nil {
		t.Fatal(err)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bytes), "sk-secret-value") || strings.Contains(string(bytes), "apiKey") {
		t.Errorf("the file holds a key or its field:\n%s", bytes)
	}
	if !strings.Contains(string(bytes), `"baseUrl":"https://api.example.com/v1"`) {
		t.Errorf("the address is missing from the file:\n%s", bytes)
	}
	entry, err := keys.GetKey(t.Context(), "connection/"+created.ID+"/apiKey")
	if err != nil || !strings.HasSuffix(entry, ":"+key+"-renewed") {
		t.Errorf("key store entry = %q, %v", entry, err)
	}
}
