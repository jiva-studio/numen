package connection

import (
	"context"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/connectiontest"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	connections "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type memoryRepository struct {
	mu          sync.Mutex
	connections []connection.Connection
}

func (r *memoryRepository) ListConnections() ([]connection.Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.connections), nil
}

func (r *memoryRepository) ChangeConnections(change func([]connection.Connection) ([]connection.Connection, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	updated, err := change(slices.Clone(r.connections))
	if err != nil {
		return err
	}
	r.connections = updated
	return nil
}

type fixedTypes []connection.Type

func (t fixedTypes) ListTypes() []connection.Type { return t }

func (t fixedTypes) GetType(typeID string) (connection.Type, bool) {
	for _, one := range t {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connection.Type{}, false
}

type memoryKeys struct {
	values map[string]string
	err    error
}

func (k *memoryKeys) GetKey(_ context.Context, name string) (string, error) {
	value, found := k.values[name]
	if !found {
		return "", plugin.ErrNoKey
	}
	return value, nil
}

func (k *memoryKeys) SetKey(_ context.Context, name, key string) error {
	if k.err != nil {
		return k.err
	}
	k.values[name] = key
	return nil
}

func (k *memoryKeys) DeleteKey(_ context.Context, name string) error {
	delete(k.values, name)
	return nil
}

type fixedUses map[string][]connection.Use

func (u fixedUses) ListUses(connectionID string) ([]connection.Use, error) {
	return u[connectionID], nil
}

func (u fixedUses) RemoveRows(string) error { return nil }

func (u fixedUses) Refresh(context.Context, string) error { return nil }

func newHandler(keys *memoryKeys) *Handler { return newHandlerWithUses(keys, fixedUses{}) }

func newHandlerWithUses(keys *memoryKeys, uses connectionport.Uses) *Handler {
	fixed := "https://openrouter.ai/api/v1"
	types := fixedTypes{{PluginID: "md.numen.connections.llm", PluginName: "LLM connections", IsAvailable: true, Declaration: manifest.ConnectionType{
		ID: "openrouter", Title: "OpenRouter", Description: "One key for many models.", Operations: []string{"complete", "embed"},
		Fields: []manifest.ConnectionField{
			{Key: "baseUrl", Type: "string", Title: "Address", IsAddress: true, FixedValue: &fixed},
			{Key: "apiKey", Type: "secret", Title: "API key", IsRequired: true},
			{Key: "mode", Type: "enum", Title: "Mode", Default: "fast", Options: manifest.SettingOptions{{Value: "fast"}, {Value: "slow", Title: "Slow"}}},
			{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(4), Min: new(1.0), Max: new(32.0), Step: new(1.0)},
		},
	}}}
	next := 0
	return NewHandler(useCasesOf(connectiontest.New(&memoryRepository{}, types, keys, uses, func() (string, error) {
		next++
		return "c0000000" + string(rune('0'+next)), nil
	})))
}

func useCasesOf(all *connectiontest.Connections) UseCases {
	return UseCases{Create: all.Create, Update: all.Update, Delete: all.Delete, List: all.List, ListTypes: all.ListTypes}
}

func newKeys() *memoryKeys { return &memoryKeys{values: map[string]string{}} }

func createRequest(name string) *connect.Request[v1.CreateProviderConnectionRequest] {
	return connect.NewRequest(&v1.CreateProviderConnectionRequest{
		TypeId: "md.numen.connections.llm/openrouter", Name: name,
		Fields:  map[string]*structpb.Value{"parallelRequests": structpb.NewNumberValue(8)},
		Secrets: map[string]string{"apiKey": "sk-or-1"},
	})
}

func TestListProviderConnectionTypesReturnsTheFieldsOfEachType(t *testing.T) {
	response, err := newHandler(newKeys()).ListProviderConnectionTypes(t.Context(), connect.NewRequest(&v1.ListProviderConnectionTypesRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if len(response.Msg.GetTypes()) != 1 {
		t.Fatalf("types = %v", response.Msg.GetTypes())
	}
	one := response.Msg.GetTypes()[0]
	if one.GetTypeId() != "md.numen.connections.llm/openrouter" || one.GetPluginName() != "LLM connections" || !one.GetIsAvailable() {
		t.Errorf("type = %v", one)
	}
	if !slices.Equal(one.GetOperations(), []string{"complete", "embed"}) || len(one.GetFields()) != 4 {
		t.Fatalf("type = %v", one)
	}
	address, key, mode, parallel := one.GetFields()[0], one.GetFields()[1], one.GetFields()[2], one.GetFields()[3]
	if !address.GetIsAddress() || address.GetFixedValue() != "https://openrouter.ai/api/v1" {
		t.Errorf("address = %v", address)
	}
	if key.GetType() != "secret" || !key.GetIsRequired() {
		t.Errorf("key = %v", key)
	}
	options := mode.GetOptions().GetValues()
	if len(options) != 2 || options[0].GetStringValue() != "fast" || options[1].GetStructValue().GetFields()["title"].GetStringValue() != "Slow" {
		t.Errorf("options = %v", options)
	}
	if parallel.GetDefault().GetNumberValue() != 4 || parallel.GetMin() != 1 || parallel.GetMax() != 32 || parallel.GetStep() != 1 {
		t.Errorf("parallel = %v", parallel)
	}
}

func TestCreateProviderConnectionReturnsTheConnectionWithoutTheSecret(t *testing.T) {
	handler := newHandler(newKeys())

	response, err := handler.CreateProviderConnection(t.Context(), createRequest(""))
	if err != nil {
		t.Fatal(err)
	}

	one := response.Msg.GetConnection()
	if response.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("error = %v: %s", response.Msg.GetError(), response.Msg.GetDetail())
	}
	if one.GetName() != "OpenRouter" || one.GetRevision() != 1 || !one.GetIsTypeAvailable() || !one.GetIsComplete() {
		t.Errorf("connection = %v", one)
	}
	if !slices.Equal(one.GetSecretKeysSet(), []string{"apiKey"}) || one.GetFields()["parallelRequests"].GetNumberValue() != 8 {
		t.Errorf("connection = %v", one)
	}
	if _, found := one.GetFields()["apiKey"]; found {
		t.Errorf("the key is among the fields: %v", one.GetFields())
	}
}

func TestListProviderConnectionsReturnsTheCreatedConnections(t *testing.T) {
	handler := newHandler(newKeys())
	if _, err := handler.CreateProviderConnection(t.Context(), createRequest("Work")); err != nil {
		t.Fatal(err)
	}

	response, err := handler.ListProviderConnections(t.Context(), connect.NewRequest(&v1.ListProviderConnectionsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if len(response.Msg.GetConnections()) != 1 || response.Msg.GetConnections()[0].GetName() != "Work" {
		t.Errorf("connections = %v", response.Msg.GetConnections())
	}
}

func TestAServedFailureIsAnErrorCodeAndNotAStatus(t *testing.T) {
	handler := newHandler(newKeys())
	created := mustCreate(t, handler)
	id := created.Msg.GetConnection().GetId()

	taken, err := handler.CreateProviderConnection(t.Context(), createRequest("work"))
	if err != nil || taken.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NAME_TAKEN || taken.Msg.GetConnection() != nil {
		t.Errorf("create: response = %v, err = %v", taken.Msg, err)
	}
	stale, err := handler.UpdateProviderConnection(t.Context(), connect.NewRequest(&v1.UpdateProviderConnectionRequest{Id: id, Revision: 7, Name: "Home"}))
	if err != nil || stale.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_STALE_REVISION {
		t.Errorf("update: response = %v, err = %v", stale.Msg, err)
	}
	missing, err := handler.DeleteProviderConnection(t.Context(), connect.NewRequest(&v1.DeleteProviderConnectionRequest{Id: "c0000000f"}))
	if err != nil || missing.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NOT_FOUND {
		t.Errorf("delete: response = %v, err = %v", missing.Msg, err)
	}
	invalid, err := handler.CreateProviderConnection(t.Context(), connect.NewRequest(&v1.CreateProviderConnectionRequest{
		TypeId: "md.numen.connections.llm/openrouter", Fields: map[string]*structpb.Value{"nope": structpb.NewStringValue("x")},
		Secrets: map[string]string{"apiKey": "k"},
	}))
	if err != nil || invalid.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_INVALID_FIELD || invalid.Msg.GetDetail() != `the type has no field "nope"` {
		t.Errorf("invalid: response = %v, err = %v", invalid.Msg, err)
	}
	unavailable, err := handler.CreateProviderConnection(t.Context(), connect.NewRequest(&v1.CreateProviderConnectionRequest{TypeId: "gone.plugin/x"}))
	if err != nil || unavailable.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_TYPE_UNAVAILABLE {
		t.Errorf("unavailable: response = %v, err = %v", unavailable.Msg, err)
	}
}

func TestAKeyStoreFailureIsAnErrorCode(t *testing.T) {
	keys := newKeys()
	keys.err = context.DeadlineExceeded

	response, err := newHandler(keys).CreateProviderConnection(t.Context(), createRequest(""))

	if err != nil || response.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_KEY_STORE_FAILED {
		t.Errorf("response = %v, err = %v", response.Msg, err)
	}
}

func TestUpdateProviderConnectionChangesTheNameAndTheRevision(t *testing.T) {
	handler := newHandler(newKeys())
	created := mustCreate(t, handler)
	one := created.Msg.GetConnection()

	response, err := handler.UpdateProviderConnection(t.Context(), connect.NewRequest(&v1.UpdateProviderConnectionRequest{
		Id: one.GetId(), Revision: one.GetRevision(), Name: "Home",
		Fields: map[string]*structpb.Value{"parallelRequests": structpb.NewNumberValue(2)},
	}))
	if err != nil {
		t.Fatal(err)
	}

	updated := response.Msg.GetConnection()
	if updated.GetName() != "Home" || updated.GetRevision() != 2 || updated.GetFields()["parallelRequests"].GetNumberValue() != 2 {
		t.Errorf("connection = %v", updated)
	}
}

func TestDeleteProviderConnectionRemovesTheConnection(t *testing.T) {
	handler := newHandler(newKeys())
	created := mustCreate(t, handler)

	response, err := handler.DeleteProviderConnection(t.Context(), connect.NewRequest(&v1.DeleteProviderConnectionRequest{Id: created.Msg.GetConnection().GetId()}))
	if err != nil || response.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("response = %v, err = %v", response.Msg, err)
	}

	listed, err := handler.ListProviderConnections(t.Context(), connect.NewRequest(&v1.ListProviderConnectionsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetConnections()) != 0 {
		t.Errorf("connections = %v", listed.Msg.GetConnections())
	}
}

func mustCreate(t *testing.T, handler *Handler) *connect.Response[v1.CreateProviderConnectionResponse] {
	t.Helper()
	created, err := handler.CreateProviderConnection(t.Context(), createRequest("Work"))
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func TestListProviderConnectionsNamesThePluginSettingsThatUseEachConnection(t *testing.T) {
	uses := fixedUses{"c00000001": {{PluginID: "md.numen.tools.proofreading", PluginName: "Proofreading", Operation: "complete"}}}
	handler := newHandlerWithUses(newKeys(), uses)
	mustCreate(t, handler)

	listed, err := handler.ListProviderConnections(t.Context(), connect.NewRequest(&v1.ListProviderConnectionsRequest{}))

	if err != nil {
		t.Fatal(err)
	}
	got := listed.Msg.GetConnections()[0].GetUses()
	if len(got) != 1 || got[0].GetPluginId() != "md.numen.tools.proofreading" || got[0].GetPluginName() != "Proofreading" || got[0].GetOperation() != "complete" {
		t.Errorf("uses = %v", got)
	}
}

func TestDeleteProviderConnectionNamesThePluginSettingsWhoseRowsWereRemoved(t *testing.T) {
	uses := fixedUses{"c00000001": {{PluginID: "md.numen.engines.embeddings", PluginName: "Embeddings", Operation: "embed"}}}
	handler := newHandlerWithUses(newKeys(), uses)
	created := mustCreate(t, handler)

	response, err := handler.DeleteProviderConnection(t.Context(), connect.NewRequest(&v1.DeleteProviderConnectionRequest{Id: created.Msg.GetConnection().GetId()}))

	if err != nil {
		t.Fatal(err)
	}
	got := response.Msg.GetRemovedUses()
	if len(got) != 1 || got[0].GetPluginId() != "md.numen.engines.embeddings" || got[0].GetOperation() != "embed" {
		t.Errorf("removed uses = %v", got)
	}
}
