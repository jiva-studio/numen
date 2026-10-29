package api_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
)

func TestFillCollectionsEncodesEmptyCollections(t *testing.T) {
	response := api.ReadDeckResponse{Deck: &api.Deck{Cards: []*api.Card{{}}}}
	api.FillCollections(&response)

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"deck":{"path":"","title":"","preamble":"","cards":[{"cardId":"","heading":"","cardModelLink":"","preamble":"","values":[]}],"sections":[],"problems":[],"tail":""},"bound":0}`
	if string(encoded) != want {
		t.Fatalf("encoded response\n got %s\nwant %s", encoded, want)
	}
}

func TestFillCollectionsFillsMaps(t *testing.T) {
	settings := api.Settings{}
	api.FillCollections(&settings)
	if settings.Load == nil {
		t.Fatal("Load is nil")
	}
}
