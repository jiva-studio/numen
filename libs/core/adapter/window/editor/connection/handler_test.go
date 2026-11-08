package connection

import (
	"context"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	connections "github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

type memoryRepository struct {
	mu          sync.Mutex
	connections []connectionport.Connection
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
	r.connections = updated
	return nil
}

type fixedTypes []connectionport.Type

func (t fixedTypes) ListTypes() []connectionport.Type { return t }

func (t fixedTypes) GetType(typeID string) (connectionport.Type, bool) {
	for _, one := range t {
		if one.GetID() == typeID {
			return one, true
		}
	}
	return connectionport.Type{}, false
}

type memoryKeys struct {
	values map[string]string
	err    error
}

func (k *memoryKeys) GetKey(_ context.Context, name string) (string, error) {
	value, found := k.values[name]
	if !found {
		return "", pluginport.ErrNoKey
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

func number(value float64) *float64 { return &value }

func newHandler(keys *memoryKeys) *Handler {
	fixed := "https://openrouter.ai/api/v1"
	types := fixedTypes{{PluginID: "md.numen.connections.llm", PluginName: "LLM connections", IsAvailable: true, Declaration: manifest.ConnectionType{
		ID: "openrouter", Title: "OpenRouter", Description: "One key for many models.", Operations: []string{"complete", "embed"},
		Fields: []manifest.ConnectionField{
			{Key: "baseUrl", Type: "string", Title: "Address", IsAddress: true, FixedValue: &fixed},
			{Key: "apiKey", Type: "secret", Title: "API key", IsRequired: true},
			{Key: "mode", Type: "enum", Title: "Mode", Default: "fast", Options: manifest.SettingOptions{{Value: "fast"}, {Value: "slow", Title: "Slow"}}},
			{Key: "parallelRequests", Type: "number", Title: "Parallel requests", Default: float64(4), Min: number(1), Max: number(32), Step: number(1)},
		},
	}}}
	next := 0
	return NewHandler(connections.NewConnections(&memoryRepository{}, types, keys, func() (string, error) {
		next++
		return "c0000000" + string(rune('0'+next)), nil
	}))
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
	created, _ := handler.CreateProviderConnection(t.Context(), createRequest("Work"))
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
	created, _ := handler.CreateProviderConnection(t.Context(), createRequest("Work"))
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
	created, _ := handler.CreateProviderConnection(t.Context(), createRequest("Work"))

	response, err := handler.DeleteProviderConnection(t.Context(), connect.NewRequest(&v1.DeleteProviderConnectionRequest{Id: created.Msg.GetConnection().GetId()}))
	if err != nil || response.Msg.GetError() != v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("response = %v, err = %v", response.Msg, err)
	}

	listed, _ := handler.ListProviderConnections(t.Context(), connect.NewRequest(&v1.ListProviderConnectionsRequest{}))
	if len(listed.Msg.GetConnections()) != 0 {
		t.Errorf("connections = %v", listed.Msg.GetConnections())
	}
}
