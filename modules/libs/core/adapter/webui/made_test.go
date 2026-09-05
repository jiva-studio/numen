package webui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// A stencil, a deck and a preset are filed under the name they were given, and
// nothing numbers a second attempt: making one twice is refused as occupied. So
// a file that landed and an index that would not follow has to come back with
// its path, or the person is left with a note they cannot see, cannot open and
// cannot make again.

// unmade is a vault whose index will not come level with anything made in it,
// asked the way the window's card editor asks.
func unmade(t *testing.T, notes map[string]string) *cutting {
	t.Helper()

	f := quitting(t, nil, notes)
	scanned(t, f)
	f.opened.API.Cards.Create.Index = jammed

	route, handler := numenv1connect.NewCardsServiceHandler(f.opened.API)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &cutting{
		client: numenv1connect.NewCardsServiceClient(server.Client(), server.URL),
		root:   f.root,
	}
}

func TestAStencilTheIndexWouldNotComeLevelWithIsAnsweredWithItsPath(t *testing.T) {
	f := unmade(t, nil)

	answer, err := f.client.CreateStencil(t.Context(), connect.NewRequest(&v1.CreateStencilRequest{
		Title:  "Animal",
		Folder: "cards",
		Fields: []string{"Name", "Height"},
	}))
	if err != nil {
		t.Fatalf("the stencil is on disk and the client was told %v", err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the stencil was made and answered %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "cards/Animal.md" {
		t.Fatalf("the stencil is filed at %q", path)
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); held == "" {
		t.Error("the stencil the client was given a path to holds nothing")
	}
}

func TestADeckTheIndexWouldNotComeLevelWithIsAnsweredWithItsPath(t *testing.T) {
	f := unmade(t, nil)

	answer, err := f.client.CreateDeck(t.Context(), connect.NewRequest(&v1.CreateDeckRequest{
		Title: "Animals",
	}))
	if err != nil {
		t.Fatalf("the deck is on disk and the client was told %v", err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the deck was made and answered %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "Animals.md" {
		t.Fatalf("the deck is filed at %q", path)
	}
}

// TestAStencilTheIndexWouldNotComeLevelWithCannotBeMadeAgain is why the path
// has to come back: the name is taken from the moment the file lands.
func TestAStencilTheIndexWouldNotComeLevelWithCannotBeMadeAgain(t *testing.T) {
	f := unmade(t, nil)

	made := connect.NewRequest(&v1.CreateStencilRequest{Title: "Animal", Fields: []string{"Name"}})
	if _, err := f.client.CreateStencil(t.Context(), made); err != nil {
		t.Fatal(err)
	}

	again, err := f.client.CreateStencil(t.Context(), connect.NewRequest(&v1.CreateStencilRequest{
		Title:  "Animal",
		Fields: []string{"Name"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := again.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_OCCUPIED {
		t.Fatalf("making the stencil a second time answered %v", refusal)
	}
}
