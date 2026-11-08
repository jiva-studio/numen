package connection_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
)

func TestResolveReturnsTheFieldsAsTextWithTheFixedValuesAndNoSecret(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Ollama", "http://localhost:11434/v1", "sk-local")
	if _, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: created.Revision, Name: "Ollama",
		Fields: map[string]any{"baseUrl": "http://localhost:11434/v1", "mode": "fast", "stream": true},
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if resolved.PluginID != "md.numen.connections.llm" || resolved.Declaration.ID != "openai-compatible" || resolved.Name != "Ollama" || resolved.Revision != 2 {
		t.Errorf("resolved = %+v", resolved)
	}
	want := map[string]string{"baseUrl": "http://localhost:11434/v1", "mode": "fast", "stream": "true"}
	if len(resolved.Fields) != len(want) {
		t.Fatalf("fields = %v, want %v", resolved.Fields, want)
	}
	for key, value := range want {
		if resolved.Fields[key] != value {
			t.Errorf("field %s = %q, want %q", key, resolved.Fields[key], value)
		}
	}
}

func TestResolveAddsTheFixedValueOfTheTypeAndNumbersAsText(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	if _, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: created.Revision, Name: "Work", Fields: map[string]any{"parallelRequests": float64(8)},
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Fields["baseUrl"] != "https://openrouter.ai/api/v1" || resolved.Fields["parallelRequests"] != "8" {
		t.Errorf("fields = %v", resolved.Fields)
	}
}

func TestResolveOmitsAStoredFieldTheTypeDoesNotDeclare(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	if err := f.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
		list[0].Fields = map[string]any{"leftover": "x"}
		return list, nil
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, isLeft := resolved.Fields["leftover"]; isLeft {
		t.Errorf("fields = %v", resolved.Fields)
	}
}

func TestResolveRefusesAConnectionThatIsNotInTheList(t *testing.T) {
	f := newFixture(t)
	if _, err := f.connection.Resolve.Execute("c00000099"); !errors.Is(err, connection.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestResolveRefusesAConnectionWhoseTypeNoPluginProvides(t *testing.T) {
	f := newFixture(t)
	if err := f.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
		return append(list, connectionport.Connection{ID: "c00000007", TypeID: retiredID, Name: "Old", Revision: 1}), nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.connection.Resolve.Execute("c00000007"); !errors.Is(err, connection.ErrTypeUnavailable) {
		t.Errorf("err = %v, want ErrTypeUnavailable", err)
	}
}

func TestReadSecretsReturnsTheValueOfEachSecretField(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	secrets, err := f.connection.ReadSecrets.Execute(t.Context(), resolved)
	if err != nil {
		t.Fatal(err)
	}

	if len(secrets) != 1 || secrets["apiKey"] != "sk-or-1" {
		t.Errorf("secrets = %v, want only the key as entered", secrets)
	}
}

func TestReadSecretsIsRefusedWhenARequiredSecretHasNoValue(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	f.keys.values = map[string]string{}
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.connection.ReadSecrets.Execute(t.Context(), resolved); !errors.Is(err, connection.ErrSecretMissing) {
		t.Errorf("err = %v, want ErrSecretMissing", err)
	}
}

func TestReadSecretsOmitsAnOptionalSecretThatHasNoValue(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Ollama", "http://localhost:11434/v1", "")
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	secrets, err := f.connection.ReadSecrets.Execute(t.Context(), resolved)
	if err != nil || len(secrets) != 0 {
		t.Errorf("secrets = %v, %v, want none", secrets, err)
	}
}

func TestReadSecretsReturnsNoKeyForAnAddressItWasNotEnteredFor(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://api.example.com/v1", "sk-for-example")
	if err := f.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
		list[0].Fields = map[string]any{"baseUrl": "https://evil.example.net/v1"}
		return list, nil
	}); err != nil {
		t.Fatal(err)
	}
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	secrets, err := f.connection.ReadSecrets.Execute(t.Context(), resolved)

	if err != nil || len(secrets) != 0 {
		t.Errorf("secrets = %v, %v, want none: the key belongs to another address", secrets, err)
	}
}

// editingKeys changes the connection while the key store is read.
type editingKeys struct {
	*memoryKeys
	change func()
}

func (k editingKeys) GetKey(ctx context.Context, name string) (string, error) {
	k.change()
	return k.memoryKeys.GetKey(ctx, name)
}

func TestReadSecretsIsRefusedWhenTheConnectionChangedWhileTheKeyWasRead(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	editing := connection.NewConnections(f.repository, f.types, editingKeys{memoryKeys: f.keys, change: func() {
		if err := f.repository.ChangeConnections(func(list []connectionport.Connection) ([]connectionport.Connection, error) {
			list[0].Revision++
			return list, nil
		}); err != nil {
			t.Error(err)
		}
	}}, f.uses, nil)

	secrets, err := editing.ReadSecrets.Execute(t.Context(), resolved)

	if !errors.Is(err, connection.ErrStaleRevision) {
		t.Errorf("err = %v, want ErrStaleRevision", err)
	}
	if len(secrets) != 0 {
		t.Errorf("a key was returned for a connection that changed: %v", secrets)
	}
}

func TestReadSecretsIsRefusedWhenTheConnectionWasDeletedWhileTheKeyWasRead(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	deleting := connection.NewConnections(f.repository, f.types, editingKeys{memoryKeys: f.keys, change: func() {
		if err := f.repository.ChangeConnections(func([]connectionport.Connection) ([]connectionport.Connection, error) { return nil, nil }); err != nil {
			t.Error(err)
		}
	}}, f.uses, nil)

	if _, err := deleting.ReadSecrets.Execute(t.Context(), resolved); !errors.Is(err, connection.ErrStaleRevision) {
		t.Errorf("err = %v, want ErrStaleRevision", err)
	}
}

func TestReadSecretsReportsAFailingKeyStore(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	failing := connection.NewConnections(f.repository, f.types, failingKeys{memoryKeys: f.keys}, f.uses, nil)

	if _, err := failing.ReadSecrets.Execute(t.Context(), resolved); !errors.Is(err, connection.ErrKeyStore) {
		t.Errorf("err = %v, want ErrKeyStore", err)
	}
}

type failingKeys struct{ *memoryKeys }

func (failingKeys) GetKey(context.Context, string) (string, error) { return "", errDisk }

func TestNewCallCarriesTheFieldsTheSecretsTheModelAndTheReasoning(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	resolved, err := f.connection.Resolve.Execute(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	call := resolved.NewCall(map[string]string{"apiKey": "sk-or-1"}, "openai/gpt-5", "low")

	if call.Type != "openrouter" || call.Model != "openai/gpt-5" || call.Reasoning != "low" ||
		call.Secrets["apiKey"] != "sk-or-1" || call.Fields["baseUrl"] != "https://openrouter.ai/api/v1" {
		t.Errorf("call = %+v", call)
	}
}
