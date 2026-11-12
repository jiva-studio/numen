package connection_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	connectiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func TestCreateKeepsASecretInTheKeyStoreAndNotInTheList(t *testing.T) {
	f := newFixture(t)

	entry := f.createOpenRouter(t, "Work")

	if entry.ID != "c00000001" || entry.Revision != 1 || entry.Name != "Work" {
		t.Errorf("entry = %+v", entry)
	}
	names := f.keys.listNames()
	if !slices.Equal(names, []string{"connection/c00000001/apiKey"}) {
		t.Fatalf("key store names = %v", names)
	}
	if !strings.HasSuffix(f.keys.values[names[0]], ":sk-or-1") {
		t.Errorf("the key store entry = %q, want a fingerprint and the key", f.keys.values[names[0]])
	}
	stored := f.listStored(t)
	if len(stored) != 1 || len(stored[0].Fields) != 0 {
		t.Errorf("stored = %+v, want no field: the key is in the key store, the address is fixed", stored)
	}
	if !slices.Equal(entry.SecretKeysSet, []string{"apiKey"}) || !entry.IsComplete || !entry.IsTypeAvailable {
		t.Errorf("entry = %+v", entry)
	}
}

func TestCreateStoresNoSecretThatIsEmpty(t *testing.T) {
	f := newFixture(t)

	entry := f.createCompatible(t, "Ollama", "http://localhost:11434/v1", "")

	if len(f.keys.listNames()) != 0 || len(entry.SecretKeysSet) != 0 {
		t.Errorf("key store = %v, set = %v", f.keys.listNames(), entry.SecretKeysSet)
	}
	if entry.Fields["baseUrl"] != "http://localhost:11434/v1" {
		t.Errorf("fields = %+v", entry.Fields)
	}
}

func TestCreateMakesANameFromTheTitleOfTheTypeAndANumber(t *testing.T) {
	f := newFixture(t)

	first := f.createOpenRouter(t, "")
	second := f.createOpenRouter(t, "")
	third := f.createOpenRouter(t, "  ")

	if first.Name != "OpenRouter" || second.Name != "OpenRouter 2" || third.Name != "OpenRouter 3" {
		t.Errorf("names = %q, %q, %q", first.Name, second.Name, third.Name)
	}
	if _, err := f.connection.Delete.Execute(t.Context(), second.ID); err != nil {
		t.Fatal(err)
	}
	if again := f.createOpenRouter(t, ""); again.Name != "OpenRouter 2" {
		t.Errorf("the lowest free number was not used: %q", again.Name)
	}
}

func TestCreateTrimsTheNameAndRefusesATakenOneIgnoringCase(t *testing.T) {
	f := newFixture(t)
	first := f.createOpenRouter(t, "  Work ")
	if first.Name != "Work" {
		t.Fatalf("name = %q", first.Name)
	}

	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Name: "WORK", Secrets: map[string]string{"apiKey": "k"},
	})

	if !errors.Is(err, connection.ErrNameTaken) {
		t.Errorf("err = %v, want ErrNameTaken", err)
	}
	if got := f.keys.listNames(); len(got) != 1 {
		t.Errorf("a refused create left key store entries: %v", got)
	}
}

func TestCreateRefusesANameOfTheWrongShape(t *testing.T) {
	f := newFixture(t)
	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Name: strings.Repeat("n", 65), Secrets: map[string]string{"apiKey": "k"},
	})
	if !errors.Is(err, connection.ErrInvalidField) {
		t.Errorf("err = %v, want ErrInvalidField", err)
	}
}

func TestCreateRefusesATypeNoPluginProvides(t *testing.T) {
	f := newFixture(t)
	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{TypeID: retiredID})
	if !errors.Is(err, connectiondomain.ErrTypeUnavailable) {
		t.Errorf("err = %v, want connectiondomain.ErrTypeUnavailable", err)
	}
}

func TestCreateRejectsInvalidFieldValues(t *testing.T) {
	cases := map[string]connection.CreateRequest{
		"an undeclared field":     {TypeID: compatibleID, Fields: map[string]any{"baseUrl": "https://a.example", "nope": "x"}},
		"a secret sent as field":  {TypeID: compatibleID, Fields: map[string]any{"baseUrl": "https://a.example", "apiKey": "k"}},
		"a fixed value as field":  {TypeID: openRouterID, Fields: map[string]any{"baseUrl": "https://evil.example"}, Secrets: map[string]string{"apiKey": "k"}},
		"an undeclared secret":    {TypeID: compatibleID, Fields: map[string]any{"baseUrl": "https://a.example"}, Secrets: map[string]string{"other": "k"}},
		"a text for a boolean":    {TypeID: compatibleID, Fields: map[string]any{"baseUrl": "https://a.example", "stream": "yes"}},
		"a value not in options":  {TypeID: compatibleID, Fields: map[string]any{"baseUrl": "https://a.example", "mode": "medium"}},
		"a number over the limit": {TypeID: openRouterID, Fields: map[string]any{"parallelRequests": float64(33)}, Secrets: map[string]string{"apiKey": "k"}},
		"a number off the step":   {TypeID: openRouterID, Fields: map[string]any{"parallelRequests": 2.5}, Secrets: map[string]string{"apiKey": "k"}},
		"a number as a text":      {TypeID: openRouterID, Fields: map[string]any{"parallelRequests": "4"}, Secrets: map[string]string{"apiKey": "k"}},
	}
	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			if _, err := f.connection.Create.Execute(t.Context(), request); !errors.Is(err, connection.ErrInvalidField) {
				t.Errorf("err = %v, want ErrInvalidField", err)
			}
			if len(f.keys.listNames()) != 0 {
				t.Errorf("key store = %v", f.keys.listNames())
			}
		})
	}
}

func TestCreateRequiresEveryRequiredField(t *testing.T) {
	f := newFixture(t)
	for name, request := range map[string]connection.CreateRequest{
		"no key":        {TypeID: openRouterID},
		"an empty key":  {TypeID: openRouterID, Secrets: map[string]string{"apiKey": ""}},
		"no address":    {TypeID: compatibleID},
		"a blank value": {TypeID: compatibleID, Fields: map[string]any{"baseUrl": ""}},
	} {
		if _, err := f.connection.Create.Execute(t.Context(), request); !errors.Is(err, connection.ErrInvalidField) {
			t.Errorf("%s: err = %v, want ErrInvalidField", name, err)
		}
	}
}

func TestCreateKeepsNeitherAFixedValueNorADefaultInTheList(t *testing.T) {
	f := newFixture(t)

	f.createOpenRouter(t, "")

	stored := f.listStored(t)
	if _, found := stored[0].Fields["baseUrl"]; found {
		t.Errorf("the fixed value was stored: %+v", stored[0].Fields)
	}
	if _, found := stored[0].Fields["parallelRequests"]; found {
		t.Errorf("a default was stored: %+v", stored[0].Fields)
	}
}

func TestCreateRemovesKeyStoreEntriesOnListWriteFailure(t *testing.T) {
	f := newFixture(t)
	f.repository.failWrite = errDisk

	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Secrets: map[string]string{"apiKey": "k"},
	})

	if !errors.Is(err, errDisk) {
		t.Errorf("err = %v", err)
	}
	if got := f.keys.listNames(); len(got) != 0 {
		t.Errorf("key store = %v", got)
	}
}

func TestCreateWritesNoListEntryWhenTheKeyStoreFails(t *testing.T) {
	f := newFixture(t)
	f.keys.failSet = errDisk

	_, err := f.connection.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Secrets: map[string]string{"apiKey": "k"},
	})

	if !errors.Is(err, connectiondomain.ErrKeyStore) {
		t.Errorf("err = %v, want connectiondomain.ErrKeyStore", err)
	}
	if stored := f.listStored(t); len(stored) != 0 {
		t.Errorf("stored = %+v", stored)
	}
}

func TestCreateOfASecretFailsWhenTheMachineHasNoKeyStore(t *testing.T) {
	f := newFixture(t)
	noKeys := connectiontest.New(f.repository, f.types, nil, f.uses, func() (string, error) { return "c00000001", nil })

	_, err := noKeys.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: openRouterID, Secrets: map[string]string{"apiKey": "k"},
	})
	if !errors.Is(err, connectiondomain.ErrKeyStore) {
		t.Errorf("err = %v, want connectiondomain.ErrKeyStore", err)
	}

	local, err := noKeys.Create.Execute(t.Context(), connection.CreateRequest{
		TypeID: compatibleID, Fields: map[string]any{"baseUrl": "http://localhost:11434/v1"},
	})
	if err != nil || local.ID == "" {
		t.Errorf("a connection with no secret needs no key store: %v", err)
	}
}

func TestCreateSkipsAnIDThatIsTaken(t *testing.T) {
	f := newFixture(t)
	ids := []string{"c00000001", "c00000001", "c00000002"}
	f.connection = connectiontest.New(f.repository, f.types, f.keys, f.uses, func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	f.createOpenRouter(t, "A")

	second := f.createOpenRouter(t, "B")

	if second.ID != "c00000002" {
		t.Errorf("id = %q", second.ID)
	}
}
