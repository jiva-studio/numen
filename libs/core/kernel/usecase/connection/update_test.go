package connection_test

import (
	"errors"
	"slices"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func TestUpdateRenamesAConnectionAndAddsOneToItsRevision(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")

	updated, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: created.Revision, Name: "Home",
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated.Name != "Home" || updated.Revision != 2 {
		t.Errorf("updated = %+v", updated)
	}
	if !slices.Equal(updated.SecretKeysSet, []string{"apiKey"}) {
		t.Errorf("a secret that the request does not name was lost: %v", updated.SecretKeysSet)
	}
}

func TestUpdateAddsOneToTheRevisionForAKeyWriteAlone(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")

	updated, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Work", Secrets: map[string]string{"apiKey": "sk-or-2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated.Revision != 2 {
		t.Errorf("revision = %d", updated.Revision)
	}
	if !hasValueContaining(f.keys.values, ":sk-or-2") || hasValueContaining(f.keys.values, ":sk-or-1") {
		t.Errorf("key store = %v", f.keys.values)
	}
}

func TestUpdateIsRefusedWhenTheRevisionIsOlderThanTheStoredOne(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	if _, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "First window",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Second window", Secrets: map[string]string{"apiKey": "sk-or-9"},
	})

	if !errors.Is(err, connectiondomain.ErrStaleRevision) {
		t.Fatalf("err = %v, want connectiondomain.ErrStaleRevision", err)
	}
	stored := f.listStoredConnections(t)
	if stored[0].Name != "First window" || stored[0].Revision != 2 {
		t.Errorf("stored = %+v", stored[0])
	}
	if hasValueContaining(f.keys.values, "sk-or-9") {
		t.Errorf("a refused update wrote a key: %v", f.keys.values)
	}
}

func TestUpdateOfAnUnknownConnectionIsNotFound(t *testing.T) {
	f := newFixture(t)
	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: "c0000000f", Revision: 1, Name: "X"})
	if !errors.Is(err, connectiondomain.ErrNotFound) {
		t.Errorf("err = %v, want connectiondomain.ErrNotFound", err)
	}
}

func TestUpdateRefusesATakenName(t *testing.T) {
	f := newFixture(t)
	f.createOpenRouter(t, "Work")
	other := f.createOpenRouter(t, "Home")

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: other.ID, Revision: 1, Name: "work"})

	if !errors.Is(err, connection.ErrNameTaken) {
		t.Errorf("err = %v, want ErrNameTaken", err)
	}
}

func TestUpdateKeepsTheNameWhenTheRequestNamesTheSameOneInOtherCase(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")

	updated, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: created.ID, Revision: 1, Name: "WORK"})

	if err != nil || updated.Name != "WORK" {
		t.Errorf("updated = %+v, err = %v", updated, err)
	}
}

func TestUpdateOfTheAddressClearsTheSecretsBeforeItStoresTheNewOnes(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://a.example/v1", "key-for-a")

	cleared, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Server", Fields: map[string]any{"baseUrl": "https://b.example/v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.SecretKeysSet) != 0 || len(f.keys.listNames()) != 0 {
		t.Fatalf("set = %v, key store = %v: the key of the old address stayed", cleared.SecretKeysSet, f.keys.listNames())
	}

	renewed, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 2, Name: "Server", Fields: map[string]any{"baseUrl": "https://c.example/v1"},
		Secrets: map[string]string{"apiKey": "key-for-c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(renewed.SecretKeysSet, []string{"apiKey"}) || !hasValueContaining(f.keys.values, ":key-for-c") {
		t.Errorf("set = %v, key store = %v", renewed.SecretKeysSet, f.keys.values)
	}
}

func TestUpdateKeepsTheSecretsWhenTheAddressIsTheSame(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://a.example/v1", "key-for-a")

	updated, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Renamed", Fields: map[string]any{"baseUrl": "https://a.example/v1"},
	})

	if err != nil || !slices.Equal(updated.SecretKeysSet, []string{"apiKey"}) {
		t.Errorf("updated = %+v, err = %v", updated, err)
	}
}

func TestUpdateClearsASecretWithAnEmptyValue(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://a.example/v1", "key-for-a")

	updated, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Server", Fields: map[string]any{"baseUrl": "https://a.example/v1"},
		Secrets: map[string]string{"apiKey": ""},
	})

	if err != nil || len(updated.SecretKeysSet) != 0 || len(f.keys.listNames()) != 0 {
		t.Errorf("updated = %+v, key store = %v, err = %v", updated, f.keys.listNames(), err)
	}
}

func TestUpdateKeepsAStoredFieldTheTypeNoLongerDeclares(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://a.example/v1", "")
	f.repository.connections[0].Fields["removedField"] = "old"

	updated, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Server", Fields: map[string]any{"baseUrl": "https://a.example/v1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, isListed := updated.Fields["removedField"]; isListed {
		t.Errorf("the window got a field the type does not declare: %+v", updated.Fields)
	}
	if f.repository.connections[0].Fields["removedField"] != "old" {
		t.Errorf("stored = %+v: the undeclared field was dropped", f.repository.connections[0].Fields)
	}
}

func TestUpdateRefusesAnAddressThatAKeyMustNotBeSentTo(t *testing.T) {
	f := newFixture(t)
	created := f.createCompatible(t, "Server", "https://a.example/v1", "key-for-a")

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Server", Fields: map[string]any{"baseUrl": "http://metadata.google.internal"},
		Secrets: map[string]string{"apiKey": "k"},
	})

	if !errors.Is(err, connection.ErrInvalidField) {
		t.Errorf("err = %v, want ErrInvalidField", err)
	}
}

func TestUpdateRestoresKeysOnListWriteFailure(t *testing.T) {
	f := newFixture(t)
	created := f.createOpenRouter(t, "Work")
	before := f.keys.values["connection/"+created.ID+"/apiKey"]
	f.repository.failWrite = errDisk

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{
		ID: created.ID, Revision: 1, Name: "Work", Secrets: map[string]string{"apiKey": "sk-or-2"},
	})

	if !errors.Is(err, errDisk) {
		t.Fatalf("err = %v", err)
	}
	if f.keys.values["connection/"+created.ID+"/apiKey"] != before {
		t.Errorf("key store = %v: the old key was not restored", f.keys.values)
	}
}

func TestUpdateRejectsAnUnavailableType(t *testing.T) {
	f := newFixture(t)
	f.repository.connections = append(f.repository.connections, retiredConnection())

	_, err := f.connection.Update.Execute(t.Context(), connection.UpdateRequest{ID: "c0000000a", Revision: 1, Name: "Old"})

	if !errors.Is(err, connectiondomain.ErrTypeUnavailable) {
		t.Errorf("err = %v, want connectiondomain.ErrTypeUnavailable", err)
	}
}
