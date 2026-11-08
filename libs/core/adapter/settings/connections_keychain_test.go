package settings_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
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
		return "", pluginport.ErrNoKey
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

func (oneType) ListTypes() []connectionport.Type { return []connectionport.Type{keychainType()} }

func (oneType) GetType(typeID string) (connectionport.Type, bool) {
	if typeID == keychainType().GetID() {
		return keychainType(), true
	}
	return connectionport.Type{}, false
}

func keychainType() connectionport.Type {
	return connectionport.Type{PluginID: "md.numen.connections.llm", PluginName: "LLM", IsAvailable: true, Declaration: manifest.ConnectionType{
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
		func() (container.Settings, error) { return openAt(path) },
		func() {})
	next := 0
	no := settings.NewConnectionRows(settings.ConnectionRowsSource{
		Connections: repository, Plugins: func() []pluginport.PluginInfo { return nil },
	})
	connections := connection.NewConnections(repository, oneType{}, keys, no, func() (string, error) {
		next++
		return "c0000000" + string(rune('0'+next)), nil
	})
	const key = "sk-secret-value-123"

	created, err := connections.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: keychainType().GetID(), Name: "Server",
		Fields:  map[string]any{"baseUrl": "https://api.example.com/v1"},
		Secrets: map[string]string{"apiKey": key},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connections.Update.Execute(t.Context(), connection.UpdateRequest{
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

func TestTheKeyStoreNamesOfConnectionsAndSecretsNeverCollide(t *testing.T) {
	secretIDs := []string{"s0a1b2c3d", "sffffffff", "s00000000"}
	connectionIDs := []string{"c0a1b2c3d", "cffffffff", "c00000000"}
	names := map[string]bool{}
	for _, id := range secretIDs {
		names[secret.GetEntryName(id)] = true
	}
	for _, id := range connectionIDs {
		for _, field := range []string{"apiKey", "key", ""} {
			name := connection.GetEntryName(id, field)
			if names[name] {
				t.Errorf("the name %q is used for a secret and a connection", name)
			}
			if !strings.HasPrefix(name, "connection/") || strings.HasPrefix(secret.GetEntryName(id), "connection/") {
				t.Errorf("name = %q", name)
			}
		}
	}
	if secret.GetEntryName("connection/c0a1b2c3d/apiKey") == connection.GetEntryName("c0a1b2c3d", "apiKey") {
		t.Error("a secret id can build the name of a connection entry")
	}
}
