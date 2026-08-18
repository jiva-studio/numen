package mcp_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// window is what an agent puts a note in front of.
type window struct {
	asked []string
	drawn []domain.Editing
	fails error
}

func (w *window) Focus(_ context.Context, path string) error {
	if w.fails != nil {
		return w.fails
	}
	w.asked = append(w.asked, path)
	return nil
}

// watched is the tools as an agent meets them, with somebody looking at the
// vault.
func watched(t *testing.T, notes map[string]string) (*sdk.ClientSession, *window) {
	t.Helper()
	_, core := built(t, notes)
	looking := &window{}
	core.View = looking
	tells := note.Telling(func(ctx context.Context, said domain.Editing) {
		_ = looking.Editing(ctx, said)
	})
	core.Write.Telling = tells
	core.Replace.Telling = tells
	return connectedTo(t, core), looking
}

func TestFocusPutsANoteInFrontOfThePerson(t *testing.T) {
	session, looking := watched(t, map[string]string{"notes/entropy.md": "# Entropy\n"})

	out := call[struct {
		Focused struct {
			Path  string `json:"path"`
			Title string `json:"title"`
		} `json:"focused"`
	}](t, session, "note_focus", map[string]any{"path": "notes/entropy.md"})

	if out.Focused.Path != "notes/entropy.md" || out.Focused.Title != "Entropy" {
		t.Errorf("answered with %+v", out.Focused)
	}
	if len(looking.asked) != 1 || looking.asked[0] != "notes/entropy.md" {
		t.Errorf("the window was asked for %v", looking.asked)
	}
}

func TestFocusRefusesANoteTheVaultDoesNotHold(t *testing.T) {
	session, looking := watched(t, map[string]string{"notes/entropy.md": "# Entropy\n"})

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
	_, core := built(t, map[string]string{"notes/entropy.md": "# Entropy\n"})
	core.View = &window{fails: errors.New("nobody is looking")}
	session := connectedTo(t, core)

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "note_focus", Arguments: map[string]any{"path": "notes/entropy.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("a window that refused was reported as having shown it")
	}
}

// Without a window there is nobody to show a note to, and the tool an agent
// would call is not there to call.
func TestNoWindowMeansNoTool(t *testing.T) {
	_, core := built(t, map[string]string{"notes/entropy.md": "# Entropy\n"})
	session := connectedTo(t, core)

	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name == "note_focus" {
			t.Fatal("a headless vault serves a tool that needs somebody looking")
		}
	}
}

// drawn is what the window was told about a change being made, in the order it
// was told.
func (w *window) Editing(_ context.Context, said domain.Editing) error {
	if w.fails != nil {
		return w.fails
	}
	w.drawn = append(w.drawn, said)
	return nil
}
