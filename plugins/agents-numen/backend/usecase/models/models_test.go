package models_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/models"
)

// fakeModels are the connections that serve llm.chat and the models each lists by connection id. A
// connection without models in the map cannot list them.
type fakeModels struct {
	connections []conversation.ChatConnection
	models      map[string][]conversation.ModelOption
	err         error
}

func (f *fakeModels) ListConnections(context.Context) ([]conversation.ChatConnection, error) {
	return f.connections, f.err
}

func (f *fakeModels) ListModels(_ context.Context, chat conversation.ChatConnection) ([]conversation.ModelOption, error) {
	listed, isListed := f.models[chat.ID]
	if !isListed {
		return nil, errors.New("the service did not answer")
	}
	return listed, nil
}

func listOptions(t *testing.T, source *fakeModels) []models.Option {
	t.Helper()
	lister, err := models.NewLister(source)
	if err != nil {
		t.Fatal(err)
	}
	options, err := lister.ListOptions(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return options
}

func TestTheOptionsAreTheModelsOfEveryChatConnectionInOrder(t *testing.T) {
	options := listOptions(t, &fakeModels{
		connections: []conversation.ChatConnection{
			{ID: "c-meadow", Name: "Meadow", ModelField: "model"},
			{ID: "c-harbor", Name: "Harbor", ModelField: "model"},
		},
		models: map[string][]conversation.ModelOption{
			"c-meadow": {{ID: "wren-2", Title: "Wren 2"}, {ID: "heron-mini", Title: ""}},
			"c-harbor": {{ID: "gull-7b", Title: "Gull 7B"}},
		},
	})

	want := []models.Option{
		{Value: "c-meadow|wren-2", Title: "Meadow · Wren 2"},
		{Value: "c-meadow|heron-mini", Title: "Meadow · heron-mini"},
		{Value: "c-harbor|gull-7b", Title: "Harbor · Gull 7B"},
	}
	if !reflect.DeepEqual(options, want) {
		t.Errorf("options = %+v, want %+v", options, want)
	}
}

func TestAConnectionWhoseModelsCannotBeListedOrThatListsNoneIsLeftOut(t *testing.T) {
	options := listOptions(t, &fakeModels{
		connections: []conversation.ChatConnection{
			{ID: "c-meadow", Name: "Meadow", ModelField: "model"},
			{ID: "c-silent", Name: "Silent", ModelField: "model"},
			{ID: "c-quarry", Name: "Quarry", ModelField: ""},
		},
		models: map[string][]conversation.ModelOption{
			"c-meadow": {{ID: "wren-2", Title: "Wren 2"}},
			"c-quarry": {{ID: "flint", Title: "Flint"}},
		},
	})

	if !reflect.DeepEqual(options, []models.Option{{Value: "c-meadow|wren-2", Title: "Meadow · Wren 2"}}) {
		t.Errorf("options = %+v", options)
	}
}

func TestTheOptionsOfAChoiceNameTheConnectionAndTheModel(t *testing.T) {
	options := listOptions(t, &fakeModels{
		connections: []conversation.ChatConnection{{ID: "md.numen.connections.llm/ollama/Desk|Lab", Name: "Desk", ModelField: "model"}},
		models:      map[string][]conversation.ModelOption{"md.numen.connections.llm/ollama/Desk|Lab": {{ID: "qwen|2.5", Title: "Qwen"}}},
	})

	choice, isChoice := conversation.ParseChoice(options[0].Value)
	if !isChoice || choice.ConnectionID != "md.numen.connections.llm/ollama/Desk|Lab" || choice.ModelID != "qwen|2.5" {
		t.Errorf("the value %q names %+v", options[0].Value, choice)
	}
}

func TestAListingOfTheConnectionsThatFailsIsTheError(t *testing.T) {
	failure := errors.New("the core is gone")
	lister, err := models.NewLister(&fakeModels{err: failure})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := lister.ListOptions(t.Context()); !errors.Is(err, failure) {
		t.Errorf("err = %v, want the failure of the listing", err)
	}
}
