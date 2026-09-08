package mcp_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// window is what an agent puts a place in front of.
type window struct {
	asked []domain.Place
	drawn []domain.Edit
	went  []domain.Move
	fails error
}

func (w *window) Focus(_ context.Context, at domain.Place) error {
	if w.fails != nil {
		return w.fails
	}
	w.asked = append(w.asked, at)
	return nil
}

// watched is the tools as an agent meets them, with somebody looking at the
// vault.
func watched(t *testing.T, notes map[string]string) (*sdk.ClientSession, *window) {
	t.Helper()
	_, core := built(t, notes)
	looking := &window{}
	core.View = looking
	tells := note.TellEdit(func(ctx context.Context, said domain.Edit) {
		_ = looking.Editing(ctx, said)
	})
	core.Notes.Write.Drawing = tells
	core.Notes.Replace.Drawing = tells
	moving := note.TellMove(func(ctx context.Context, went domain.Move) {
		_ = looking.Moved(ctx, went)
	})
	core.Notes.Move.Drawing = moving
	core.Notes.Rename.Drawing = moving
	return connectedTo(t, core), looking
}

func TestFocusPutsANoteInFrontOfThePerson(t *testing.T) {
	session, looking := watched(t, map[string]string{"notes/Entropy.md": "# Entropy\n"})

	out := call[struct {
		Focused struct {
			Path  string `json:"path"`
			Title string `json:"title"`
		} `json:"focused"`
	}](t, session, "note_focus", map[string]any{"path": "notes/Entropy.md"})

	if out.Focused.Path != "notes/Entropy.md" || out.Focused.Title != "Entropy" {
		t.Errorf("answered with %+v", out.Focused)
	}
	// A note is put in front of the person whole. Nothing about it names a
	// stretch, so nothing is asked for one.
	want := domain.Place{Path: "notes/Entropy.md"}
	if len(looking.asked) != 1 || !reflect.DeepEqual(looking.asked[0], want) {
		t.Errorf("the window was asked for %v", looking.asked)
	}
}

func TestFocusRefusesANoteTheVaultDoesNotHold(t *testing.T) {
	session, looking := watched(t, map[string]string{"notes/Entropy.md": "# Entropy\n"})

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "note_focus", Arguments: map[string]any{"path": "notes/nowhere.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("a note that is not there was shown anyway")
	}
	if len(looking.asked) != 0 {
		t.Errorf("the window was asked for %v", looking.asked)
	}
}

func TestFocusSaysSoWhenTheWindowWouldNot(t *testing.T) {
	_, core := built(t, map[string]string{"notes/Entropy.md": "# Entropy\n"})
	core.View = &window{fails: errors.New("nobody is looking")}
	session := connectedTo(t, core)

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "note_focus", Arguments: map[string]any{"path": "notes/Entropy.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("a window that refused was reported as having shown it")
	}
}

// Without a window there is nobody to show anything to, and the tools an agent
// would call are not there to call.
func TestNoWindowMeansNoTool(t *testing.T) {
	_, core := built(t, map[string]string{"notes/Entropy.md": "# Entropy\n"})
	session := connectedTo(t, core)

	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name == "note_focus" || tool.Name == "source_focus" {
			t.Fatalf("a headless vault serves %s, which needs somebody looking", tool.Name)
		}
	}
}

// library is a vault holding a document beside its notes.
var library = map[string]string{
	"notes/Entropy.md":    "# Entropy\n",
	"library/A Book.epub": "a document nothing here reads",
}

// A person asked to be shown a passage of a book, and what the agent hands over
// is the place: the document, and where in its text to open.
func TestShowPutsAPlaceInFrontOfThePerson(t *testing.T) {
	session, looking := watched(t, library)

	out := call[struct {
		Shown   bool   `json:"shown"`
		Looking string `json:"looking"`
	}](t, session, "source_focus", map[string]any{
		"path": "library/A Book.epub", "start": 1200, "length": 80,
	})

	if !out.Shown {
		t.Errorf("answered with %+v", out)
	}
	want := domain.Place{Path: "library/A Book.epub", Spans: []domain.Span{{From: 1200, To: 1280}}}
	if len(looking.asked) != 1 || !reflect.DeepEqual(looking.asked[0], want) {
		t.Errorf("the window was asked for %v", looking.asked)
	}
}

func TestShowRefusesAPathTheVaultDoesNotHold(t *testing.T) {
	session, looking := watched(t, library)

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      "source_focus",
		Arguments: map[string]any{"path": "library/Nowhere.epub", "start": 0, "length": 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("a document that is not there was shown anyway")
	}
	if len(looking.asked) != 0 {
		t.Errorf("the window was asked for %v", looking.asked)
	}
}

// drawn is what the window was told about a change being made, in the order it
// was told.
func (w *window) Editing(_ context.Context, said domain.Edit) error {
	if w.fails != nil {
		return w.fails
	}
	w.drawn = append(w.drawn, said)
	return nil
}

// went is where the window was told each note moved to.
func (w *window) Moved(_ context.Context, went domain.Move) error {
	if w.fails != nil {
		return w.fails
	}
	w.went = append(w.went, went)
	return nil
}

// A person reading a note that is renamed under them is reading a name with no
// file behind it, and every change made afterwards is made somewhere they are
// not looking. So the window is told where the note went.
func TestRenamingANoteSaysWhereItWent(t *testing.T) {
	s, looking := watched(t, map[string]string{"Entropy.md": "# Entropy\n"})

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_rename", map[string]any{"path": "Entropy.md", "title": "Order"})

	if len(looking.went) != 1 {
		t.Fatalf("the window was told %d times, wanted once", len(looking.went))
	}
	if looking.went[0].From != "Entropy.md" || looking.went[0].To == "" {
		t.Errorf("the note went %+v", looking.went[0])
	}
}
