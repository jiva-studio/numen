package webui_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Two processes write the index, so a write that reached the vault may find the
// index held open by the other one. The file is on disk either way, and a
// person told their save failed loses the fingerprint the next save presents.

// jammed is an index that will not come level with what was written.
func jammed(context.Context, domain.Vault, []string) error {
	return errors.New("the index is held open elsewhere")
}

// unwritten is a vault with those notes in it and an index that will not come
// level with anything written to it: a client asking the way the editor's
// window does, and a client asking the way its card editor does.
func unwritten(t *testing.T, notes map[string]string) (*going, *cutting) {
	t.Helper()

	f := quitting(t, nil, notes)
	scanned(t, f)
	f.opened.API.Saves.Index = jammed
	f.opened.API.Cuts.Index = jammed

	route, handler := numenv1connect.NewCardsServiceHandler(f.opened.API)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return f, &cutting{
		client: numenv1connect.NewCardsServiceClient(server.Client(), server.URL),
		root:   f.root,
	}
}

func TestASaveTheIndexWouldNotComeLevelWithIsAnswered(t *testing.T) {
	f, _ := unwritten(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	answer, err := f.client.Write(t.Context(), connect.NewRequest(&v1.WriteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetChanged() || answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("saving answered %+v", answer.Msg)
	}
	// Without it the next save has nothing to present and is answered as a note
	// somebody else touched.
	if answer.Msg.GetAt() == nil {
		t.Error("the save landed and the client was handed no fingerprint")
	}
	if held := onDisk(t, f.root, "Note.md"); !strings.Contains(held, "what the person typed") {
		t.Errorf("the note on disk is %q", held)
	}
}

func TestADeckWrittenWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	_, f := unwritten(t, map[string]string{
		"Animal.md":  animal,
		"Animals.md": threeCards,
	})

	read := deck(t, f, "Animals.md")
	answer, err := f.client.WriteDeck(t.Context(), connect.NewRequest(&v1.WriteDeckRequest{
		Path:     "Animals.md",
		Preamble: read.GetDeck().GetPreamble(),
		Sections: read.GetDeck().GetSections(),
		Cards:    read.GetDeck().GetCards()[:1],
		Tail:     read.GetDeck().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetChanged() || answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("writing the deck answered %+v", answer.Msg)
	}
	if answer.Msg.GetAt() == nil {
		t.Error("the deck was written and the client was handed no fingerprint")
	}
	if held := onDisk(t, f.root, "Animals.md"); strings.Contains(held, "Alpaca") {
		t.Errorf("the deck on disk is %q", held)
	}
}

func TestAStencilWrittenWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	_, f := unwritten(t, map[string]string{"cards/Animal.md": animal})

	read := stencil(t, f, "cards/Animal.md")
	faces := read.GetStencil().GetFaces()
	faces[0].Back = "{{Height}} at the shoulder"

	answer, err := f.client.WriteStencil(t.Context(), connect.NewRequest(&v1.WriteStencilRequest{
		Path:     "cards/Animal.md",
		Fields:   read.GetStencil().GetFields(),
		Faces:    faces,
		Preamble: read.GetStencil().GetPreamble(),
		Tail:     read.GetStencil().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetChanged() || answer.Msg.GetRefusal() != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("writing the stencil answered %+v", answer.Msg)
	}
	if answer.Msg.GetAt() == nil {
		t.Error("the stencil was written and the client was handed no fingerprint")
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); !strings.Contains(held, "at the shoulder") {
		t.Errorf("the stencil on disk is %q", held)
	}
}
