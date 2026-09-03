package mcp_test

import (
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// tabbed is one tab of the window, as an agent is told about it.
type tabbed struct {
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	Title string `json:"title"`
	Where string `json:"where"`
	Front bool   `json:"front"`
}

// attending is the tools as an agent meets them, with a window saying what the
// person has open.
func attending(t *testing.T, open domain.Attention) *sdk.ClientSession {
	t.Helper()
	_, core := built(t, map[string]string{"notes/Entropy.md": "# Entropy\n"})
	core.Attending = func() domain.Attention { return open }
	return connectedTo(t, core)
}

// tabs is what window_tabs answers.
func tabs(t *testing.T, session *sdk.ClientSession) struct {
	Tabs []tabbed `json:"tabs"`
	Says string   `json:"says"`
} {
	t.Helper()
	return call[struct {
		Tabs []tabbed `json:"tabs"`
		Says string   `json:"says"`
	}](t, session, "window_tabs", map[string]any{})
}

func TestWindowTabsAnswersWithEveryTabAndMarksTheOneInFront(t *testing.T) {
	session := attending(t, domain.Attention{
		Front: "two",
		Tabs: []domain.Tab{
			{ID: "one", Kind: domain.TabPlex, Path: "Main 222.md", Title: "Main 222"},
			{ID: "two", Kind: domain.TabRecording, Path: "730707BG.LON.mp3",
				Title: "730707BG.LON.mp3", At: 754000, Of: 3494000},
		},
	})

	out := tabs(t, session)

	if len(out.Tabs) != 2 || out.Tabs[0].Front || !out.Tabs[1].Front {
		t.Fatalf("answered with %+v", out.Tabs)
	}
	if out.Tabs[1].Where != "12:34 of its 58:14 written down" {
		t.Errorf("the recording stands at %q", out.Tabs[1].Where)
	}
	want := `the recording "730707BG.LON.mp3" at 730707BG.LON.mp3 is in front of them, ` +
		`with 12:34 of its 58:14 written down`
	if out.Says != want {
		t.Errorf("says %q", out.Says)
	}
}

func TestWindowTabsSaysWhereInADocumentThePersonIs(t *testing.T) {
	session := attending(t, domain.Attention{
		Front: "one",
		Tabs: []domain.Tab{
			{ID: "one", Kind: domain.TabDocument, Path: "library/A Book.pdf",
				Title: "A Book.pdf", At: 3, Of: 40},
		},
	})

	out := tabs(t, session)

	want := `the document "A Book.pdf" at library/A Book.pdf is in front of them, ` +
		`open at page 3 of 40`
	if out.Says != want {
		t.Errorf("says %q", out.Says)
	}
}

// A window is free to open a kind of tab nothing here has words for, and such a
// tab is named by its own kind.
func TestWindowTabsNamesAKindItHasNoWordsForAndNoNote(t *testing.T) {
	session := attending(t, domain.Attention{
		Front: "two",
		Tabs: []domain.Tab{
			{ID: "one", Kind: domain.TabNote, Path: "notes/Entropy.md", Title: "Entropy"},
			{ID: "two", Kind: "kaleidoscope", Title: "Colours"},
		},
	})

	out := tabs(t, session)

	if out.Says != `a tab of kind "kaleidoscope" is in front of them` {
		t.Errorf("says %q", out.Says)
	}
}

func TestWindowTabsSaysSoWhereNothingIsOpen(t *testing.T) {
	session := attending(t, domain.Attention{})

	out := tabs(t, session)

	if len(out.Tabs) != 0 || out.Says != "the window has nothing open" {
		t.Errorf("answered with %+v, saying %q", out.Tabs, out.Says)
	}
}

// A build nobody is sitting at serves the vault and nothing about a window.
func TestWindowTabsIsNotServedWhereNoWindowSays(t *testing.T) {
	_, core := built(t, map[string]string{"notes/Entropy.md": "# Entropy\n"})
	session := connectedTo(t, core)

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "window_tabs"})
	if err == nil && !res.IsError {
		t.Fatal("the tool was served")
	}
}
