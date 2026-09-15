package editor_test

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
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// Two processes write the index, so a write that reached the vault may find the
// index held open by the other one. The write is answered, and the answer says
// the index is behind: the file is on disk and search does not hold what it now
// says.

// refuseToLevel is an index that will not come level with what was written.
func refuseToLevel(context.Context, domain.Vault, []string) error {
	return errors.New("the index is held open elsewhere")
}

// unwritten is a vault with those notes in it and an index that will not come
// level with anything written to it: a client asking the way the editor's
// window does, and a client asking the way its card editor does.
func unwritten(t *testing.T, notes map[string]string) (*going, *cutting) {
	t.Helper()

	f := openWindow(t, nil, notes)
	waitForScan(t, f)
	f.opened.API.Notes.Write.Index = refuseToLevel
	f.opened.API.Cards.Write.Index = refuseToLevel

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

	answer, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("saving answered %+v", answer.Msg)
	}
	// Without it the next save has nothing to present and is answered as a note
	// somebody else touched.
	if answer.Msg.GetAt() == nil {
		t.Error("the save landed and the client was handed no fingerprint")
	}
	if !answer.Msg.GetUnlevelled() {
		t.Error("the save is answered as findable and search does not hold it")
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
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing the deck answered %+v", answer.Msg)
	}
	if answer.Msg.GetAt() == nil {
		t.Error("the deck was written and the client was handed no fingerprint")
	}
	if !answer.Msg.GetUnlevelled() {
		t.Error("the deck is answered as findable and search does not hold it")
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
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing the stencil answered %+v", answer.Msg)
	}
	if answer.Msg.GetAt() == nil {
		t.Error("the stencil was written and the client was handed no fingerprint")
	}
	if !answer.Msg.GetUnlevelled() {
		t.Error("the stencil is answered as findable and search does not hold it")
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); !strings.Contains(held, "at the shoulder") {
		t.Errorf("the stencil on disk is %q", held)
	}
}

// TestASaveTheIndexCameLevelWithSaysNothingIsBehind is the control. A flag
// every save carries says nothing, and a client shown a warning after every
// save learns to ignore it.
func TestASaveTheIndexCameLevelWithSaysNothingIsBehind(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	waitForScan(t, f)

	answer, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetUnlevelled() {
		t.Error("a save the index came level with is answered as behind")
	}
	if at := findLevellingTask(t, f); at != nil {
		t.Errorf("a save that went through stands in the list as %+v", at)
	}
}

// TestASaveTheIndexWouldNotComeLevelWithStandsInTheList. A person saving a
// second note has no reason to look at what the first one answered, and the
// list of what is being done is where they are.
func TestASaveTheIndexWouldNotComeLevelWithStandsInTheList(t *testing.T) {
	f, _ := unwritten(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	save := func() {
		t.Helper()
		if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
			Path: "Note.md",
			Body: "what the person typed\n",
		})); err != nil {
			t.Fatal(err)
		}
	}

	save()
	at := findLevellingTask(t, f)
	if at == nil || at.GetError() == "" {
		t.Fatalf("what a person is shown after a save search cannot see is %+v", at)
	}

	// The index is behind until something puts it right, and the save that does
	// is what says so. A list nothing leaves is a list nobody reads.
	f.opened.API.Notes.Write.Index = func(context.Context, domain.Vault, []string) error { return nil }
	save()
	if at := findLevellingTask(t, f); at != nil {
		t.Errorf("the index came level and a person is still shown %+v", at)
	}
}

// findLevellingTask is what the list of what is being done says about the index, asked
// for the way the window asks for it. The first message of the stream is the
// list as it stands, so nothing here waits on anything.
func findLevellingTask(t *testing.T, f *going) *v1.Task {
	t.Helper()

	listening, over := context.WithCancel(t.Context())
	defer over()
	tasks, err := f.drawn.WatchTasks(listening, connect.NewRequest(&v1.WatchTasksRequest{
		Window: wire.Editor,
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer tasks.Close()
	if !tasks.Receive() {
		t.Fatalf("the tasks stream never opened: %v", tasks.Err())
	}
	for _, at := range tasks.Msg().GetTasks() {
		if at.GetId() == "levelling the index" {
			return at
		}
	}
	return nil
}
