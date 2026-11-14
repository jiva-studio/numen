package connection_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection/connectiontest"
)

func retiredConnection() connectiondomain.Connection {
	return connectiondomain.Connection{ID: "c0000000a", TypeID: retiredID, Name: "Old", Revision: 1, Fields: map[string]any{"removedField": "v"}}
}

func TestDeleteRemovesTheConnectionAndItsSecrets(t *testing.T) {
	f := newFixture(t)
	keep := f.createOpenRouter(t, "Keep")
	removedConnection := f.createOpenRouter(t, "Gone")

	if _, err := f.connection.Delete.Execute(t.Context(), removedConnection.ID); err != nil {
		t.Fatal(err)
	}

	stored := f.listStoredConnections(t)
	if len(stored) != 1 || stored[0].ID != keep.ID {
		t.Errorf("stored = %+v", stored)
	}
	if !slices.Equal(f.keys.listNames(), []string{"connection/" + keep.ID + "/apiKey"}) {
		t.Errorf("key store = %v", f.keys.listNames())
	}
}

func TestDeleteOfAnUnknownConnectionIsNotFound(t *testing.T) {
	f := newFixture(t)
	if _, err := f.connection.Delete.Execute(t.Context(), "c0000000f"); !errors.Is(err, connectiondomain.ErrNotFound) {
		t.Errorf("err = %v, want connectiondomain.ErrNotFound", err)
	}
}

func TestDeleteRestoresTheEntryWhenASecretCannotBeRemoved(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	failing := &failingDelete{memoryKeys: f.keys}
	f.connection = connectiontest.New(f.repository, f.types, failing, f.uses, func() (string, error) { return "c0000000b", nil })

	_, err := f.connection.Delete.Execute(t.Context(), created.ID)

	if !errors.Is(err, connectiondomain.ErrKeyStore) {
		t.Fatalf("err = %v, want connectiondomain.ErrKeyStore", err)
	}
	if stored := f.listStoredConnections(t); len(stored) != 1 {
		t.Errorf("the connection is gone and its key stays: %+v", stored)
	}
}

type failingDelete struct{ *memoryKeys }

func (k *failingDelete) DeleteKey(context.Context, string) error { return errDisk }

func TestListReturnsTheConnectionsWithoutAnyValueOfASecret(t *testing.T) {
	f := newFixture(t)
	f.createOpenRouter(t, "Work")
	f.createCompatible(t, "Ollama", "http://localhost:11434/v1", "")

	entries, err := f.connection.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 || entries[0].Name != "Work" || entries[1].Name != "Ollama" {
		t.Fatalf("entries = %+v", entries)
	}
	if !slices.Equal(entries[0].SecretKeysSet, []string{"apiKey"}) || len(entries[1].SecretKeysSet) != 0 {
		t.Errorf("secret keys = %v, %v", entries[0].SecretKeysSet, entries[1].SecretKeysSet)
	}
	if _, isListed := entries[0].Fields["apiKey"]; isListed {
		t.Errorf("fields = %+v", entries[0].Fields)
	}
}

func TestListSkipsAKeyWhoseAddressChangedInTheFile(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://a.example/v1", "key-for-a")
	f.repository.connections[0].Fields["baseUrl"] = "https://attacker.example/v1"

	entries, err := f.connection.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(entries[0].SecretKeysSet) != 0 {
		t.Errorf("set = %v: the key of %s follows an address edited by hand", entries[0].SecretKeysSet, created.ID)
	}
}

func TestListSkipsAKeyWhoseTypeChangedInTheFile(t *testing.T) {
	f := newFixture(t)
	f.createCompatible(t, "Server", "https://a.example/v1", "key-for-a")
	f.repository.connections[0].TypeID = openRouterID
	f.repository.connections[0].Fields = map[string]any{}

	entries := f.listEntries(t)

	if len(entries[0].SecretKeysSet) != 0 {
		t.Errorf("set = %v", entries[0].SecretKeysSet)
	}
}

func TestListMarksAConnectionOfAnUnavailableType(t *testing.T) {
	f := newFixture(t)
	f.repository.connections = append(f.repository.connections, retiredConnection())

	entries, err := f.connection.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 || entries[0].IsTypeAvailable || entries[0].Name != "Old" {
		t.Errorf("entries = %+v", entries)
	}
}

func TestListMarksAConnectionWhoseRequiredFieldIsMissingAsIncomplete(t *testing.T) {
	f := newFixture(t)
	f.createOpenRouter(t, "Work")
	f.repository.connections = append(f.repository.connections, connectiondomain.Connection{
		ID: "c0000000c", TypeID: compatibleID, Name: "No address", Revision: 1, Fields: map[string]any{},
	})
	delete(f.keys.values, "connection/c00000001/apiKey")

	entries := f.listEntries(t)

	if entries[0].IsComplete || entries[1].IsComplete {
		t.Errorf("complete = %v, %v: a missing key or address leaves a connection incomplete", entries[0].IsComplete, entries[1].IsComplete)
	}
}

func TestListShowsOnlyTheFieldsTheTypeDeclares(t *testing.T) {
	f := newFixture(t)
	f.repository.connections = append(f.repository.connections, connectiondomain.Connection{
		ID: "c0000000d", TypeID: compatibleID, Name: "Server", Revision: 1,
		Fields: map[string]any{"baseUrl": "https://a.example/v1", "removedField": "old"},
	})

	entries := f.listEntries(t)

	if len(entries[0].Fields) != 1 || entries[0].Fields["baseUrl"] != "https://a.example/v1" {
		t.Errorf("fields = %+v", entries[0].Fields)
	}
	if !entries[0].IsComplete {
		t.Error("an undeclared stored field made the connection incomplete or invalid")
	}
}

func TestListTypesReturnsTheTypesOfThePlugins(t *testing.T) {
	f := newFixture(t)

	types := f.connection.ListTypes.Execute().Types

	if len(types) != 2 || types[0].GetID() != openRouterID || types[1].GetID() != compatibleID {
		t.Errorf("types = %+v", types)
	}
}
