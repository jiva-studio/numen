package mcp_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/mcp"
)

// What the window says about a call is what the tool declared: a call names its
// subject.
func TestAToolSaysWhatOneCallIsAbout(t *testing.T) {
	words, err := mcp.Vocabulary(t.Context(), mcp.Core{})
	if err != nil {
		t.Fatal(err)
	}

	create, served := words["note_create"]
	if !served {
		t.Fatal("no note_create")
	}
	if create.Title == "" {
		t.Error("the tool has no title")
	}
	// A note is called by its title, and that is what the tool requires.
	if create.About != "title" {
		t.Errorf("a call is about %q", create.About)
	}
}

// A call that takes a collection is about the first thing in it, by the name that
// thing carries. A field that may be left out is declared as several types at
// once, and reading the declaration means reading a list of them.
func TestAToolSaysWhatOneOfItsThingsIsCalled(t *testing.T) {
	words, err := mcp.Vocabulary(t.Context(), mcp.Core{})
	if err != nil {
		t.Fatal(err)
	}

	add, served := words["link_add"]
	if !served {
		t.Fatal("no link_add")
	}
	if add.About != "links" {
		t.Errorf("a call is about %q", add.About)
	}
	// A link is named by the note it is written in, which is what a link cannot
	// be added without.
	if add.Inside != "from" {
		t.Errorf("one link is named by %q", add.Inside)
	}
}

// The panel names the note being edited while the call is still being written,
// which it can only do if the note is what the call is declared to be about.
func TestAnEditIsAboutTheNoteItChanges(t *testing.T) {
	words, err := mcp.Vocabulary(t.Context(), mcp.Core{})
	if err != nil {
		t.Fatal(err)
	}

	edit, served := words["note_edit"]
	if !served {
		t.Fatal("no note_edit")
	}
	if edit.Title == "" {
		t.Error("the tool has no title")
	}
	if edit.About != "path" {
		t.Errorf("a call is about %q", edit.About)
	}
}
