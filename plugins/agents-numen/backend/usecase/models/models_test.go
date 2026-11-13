package models_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"testing/synctest"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/models"
)

// fakeModels are the connections that serve llm.chat and the models each lists by connection id. A
// connection without models in the map cannot list them, and one in slow answers once its context ends.
type fakeModels struct {
	connections []conversation.ChatConnection
	models      map[string][]conversation.ModelOption
	slow        map[string]bool
	err         error
}

func (f *fakeModels) ListConnections(context.Context) ([]conversation.ChatConnection, error) {
	return f.connections, f.err
}

func (f *fakeModels) ListModels(ctx context.Context, chat conversation.ChatConnection) ([]conversation.ModelOption, error) {
	if f.slow[chat.ID] {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	listed, isListed := f.models[chat.ID]
	if !isListed {
		return nil, errors.New("the service did not answer")
	}
	return listed, nil
}

func listOptions(t *testing.T, source *fakeModels) models.Listing {
	t.Helper()
	lister, err := models.NewLister(source)
	if err != nil {
		t.Fatal(err)
	}
	listing, err := lister.ListOptions(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return listing
}

func TestTheOptionsAreTheModelsOfEveryChatConnectionInOrder(t *testing.T) {
	listing := listOptions(t, &fakeModels{
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
	if !reflect.DeepEqual(listing.Options, want) || listing.IsPartial {
		t.Errorf("listing = %+v, want %+v whole", listing, want)
	}
}

func TestAConnectionWhoseModelsCannotBeListedIsLeftOutOfAPartialListing(t *testing.T) {
	listing := listOptions(t, &fakeModels{
		connections: []conversation.ChatConnection{
			{ID: "c-meadow", Name: "Meadow", ModelField: "model"},
			{ID: "c-silent", Name: "Silent", ModelField: "model"},
		},
		models: map[string][]conversation.ModelOption{"c-meadow": {{ID: "wren-2", Title: "Wren 2"}}},
	})

	if !reflect.DeepEqual(listing.Options, []models.Option{{Value: "c-meadow|wren-2", Title: "Meadow · Wren 2"}}) || !listing.IsPartial {
		t.Errorf("listing = %+v, want Meadow alone, partial", listing)
	}
}

func TestAConnectionWithNoFieldOfModelsIsLeftOutOfAWholeListing(t *testing.T) {
	listing := listOptions(t, &fakeModels{
		connections: []conversation.ChatConnection{
			{ID: "c-meadow", Name: "Meadow", ModelField: "model"},
			{ID: "c-quarry", Name: "Quarry", ModelField: ""},
		},
		models: map[string][]conversation.ModelOption{
			"c-meadow": {{ID: "wren-2", Title: "Wren 2"}},
			"c-quarry": {{ID: "flint", Title: "Flint"}},
		},
	})

	if !reflect.DeepEqual(listing.Options, []models.Option{{Value: "c-meadow|wren-2", Title: "Meadow · Wren 2"}}) || listing.IsPartial {
		t.Errorf("listing = %+v, want Meadow alone, whole", listing)
	}
}

func TestASlowConnectionIsGivenUpAfterItsLimitAndTheOthersAreListed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		listing := listOptions(t, &fakeModels{
			connections: []conversation.ChatConnection{
				{ID: "c-stall", Name: "Stall", ModelField: "model"},
				{ID: "c-harbor", Name: "Harbor", ModelField: "model"},
			},
			models: map[string][]conversation.ModelOption{"c-harbor": {{ID: "gull-7b", Title: "Gull 7B"}}},
			slow:   map[string]bool{"c-stall": true},
		})

		if !reflect.DeepEqual(listing.Options, []models.Option{{Value: "c-harbor|gull-7b", Title: "Harbor · Gull 7B"}}) || !listing.IsPartial {
			t.Errorf("listing = %+v, want Harbor alone, partial", listing)
		}
	})
}

func TestTheOptionsOfAChoiceNameTheConnectionAndTheModel(t *testing.T) {
	listing := listOptions(t, &fakeModels{
		connections: []conversation.ChatConnection{{ID: "md.numen.connections.llm/ollama/Desk|Lab", Name: "Desk", ModelField: "model"}},
		models:      map[string][]conversation.ModelOption{"md.numen.connections.llm/ollama/Desk|Lab": {{ID: "qwen|2.5", Title: "Qwen"}}},
	})

	choice, isChoice := conversation.ParseChoice(listing.Options[0].Value)
	if !isChoice || choice.ConnectionID != "md.numen.connections.llm/ollama/Desk|Lab" || choice.ModelID != "qwen|2.5" {
		t.Errorf("the value %q names %+v", listing.Options[0].Value, choice)
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
