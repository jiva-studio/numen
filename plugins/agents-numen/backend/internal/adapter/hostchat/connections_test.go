package hostchat

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

// fakeRows are the rows of llm.chat of the plugin: Meadow lists its models in the field model, and Quarry
// lists none.
type fakeRows struct {
	operations []string
	asked      []connection.RowOptionsRequest
}

func (f *fakeRows) ListRows(_ context.Context, operation string) ([]connection.Row, error) {
	f.operations = append(f.operations, operation)
	return []connection.Row{
		{ConnectionID: "c-meadow", ConnectionName: "Meadow", Fields: []connection.RowField{
			{Key: "tier", Title: "Tier", OptionsFrom: "", IsRequired: false},
			{Key: "model", Title: "Model", OptionsFrom: "llm.models", IsRequired: true},
		}},
		{ConnectionID: "c-quarry", ConnectionName: "Quarry", Fields: nil},
	}, nil
}

func (f *fakeRows) ListRowOptions(_ context.Context, request connection.RowOptionsRequest) ([]connection.Option, error) {
	f.asked = append(f.asked, request)
	if request.ConnectionID != "c-meadow" {
		return nil, errors.New("not among the rows")
	}
	return []connection.Option{{Value: "wren-2", Title: "Wren 2", Group: "Birds"}}, nil
}

func TestTheConnectionsAreTheRowsOfChatWithTheFieldThatListsTheirModels(t *testing.T) {
	rows := &fakeRows{}

	connections, err := NewConnections(rows).ListConnections(t.Context())

	want := []conversation.ChatConnection{{ID: "c-meadow", Name: "Meadow", ModelField: "model"}, {ID: "c-quarry", Name: "Quarry", ModelField: ""}}
	if err != nil || !reflect.DeepEqual(connections, want) || !reflect.DeepEqual(rows.operations, []string{"llm.chat"}) {
		t.Errorf("connections = %+v, %v after asking for %v", connections, err, rows.operations)
	}
}

func TestTheModelsOfAConnectionAreTheValuesOfItsFieldOfModels(t *testing.T) {
	rows := &fakeRows{}

	listed, err := NewConnections(rows).ListModels(t.Context(), conversation.ChatConnection{ID: "c-meadow", Name: "Meadow", ModelField: "model"})

	if err != nil || !reflect.DeepEqual(listed, []conversation.ModelOption{{ID: "wren-2", Title: "Wren 2"}}) {
		t.Fatalf("models = %+v, %v", listed, err)
	}
	if asked := rows.asked[0]; asked.Operation != "llm.chat" || asked.ConnectionID != "c-meadow" || asked.Field != "model" {
		t.Errorf("asked = %+v", asked)
	}
}
