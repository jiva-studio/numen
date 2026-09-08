package editor_test

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
		Path:   "cards",
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
	if !answer.Msg.GetUnlevelled() {
		t.Error("the stencil is answered as findable and search does not hold it")
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
	if !answer.Msg.GetUnlevelled() {
		t.Error("the deck is answered as findable and search does not hold it")
	}
}

// TestANoteMadeWhenTheIndexWouldNotComeLevelIsAnswered. A note is made through
// its own use case, and it says the same thing a stencil does.
func TestANoteMadeWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	f := quitting(t, nil, nil)
	scanned(t, f)
	f.opened.API.Notes.Create.Index = jammed

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
	}))
	if err != nil {
		t.Fatalf("the note is on disk and the client was told %v", err)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note is filed at %q", path)
	}
	if !answer.Msg.GetUnlevelled() {
		t.Error("the note is answered as findable and search does not hold it")
	}
}

// TestAFileRemovedWhenTheIndexWouldNotComeLevelIsAnswered. The file is in the
// trash and search still answers about it, and what its links now reach is
// reported all the same.
func TestAFileRemovedWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
		"Entropy.md": "---\ntitle: Entropy\nlinks:\n  - to: Ontology\n    role: parent\n---\n\n" +
			"# Entropy\n",
	})
	scanned(t, f)
	f.opened.API.Notes.Remove.Index = jammed

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Ontology.md",
	}))
	if err != nil {
		t.Fatalf("the note is in the trash and the client was told %v", err)
	}
	if answer.Msg.GetTrashed() == "" {
		t.Error("the note went to the trash and the client was told nothing of where")
	}
	if !answer.Msg.GetUnlevelled() {
		t.Error("the removal is answered as followed and search still holds the note")
	}
	if len(answer.Msg.GetDangling()) != 1 {
		t.Errorf("the links now reaching nothing are %q", answer.Msg.GetDangling())
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
