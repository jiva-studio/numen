package mcp_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

// What a call does to the vault is written out beside the tools rather than
// read off a schema, so a tool renamed without its entry says nothing.
func TestEveryToolServedSaysWhatItDoes(t *testing.T) {
	// The tools a window brings are served only where there is one, and the
	// tools for the list of vaults only where there is a list, so a vault with
	// somebody looking at it and an installation holding several are asked as
	// well.
	for _, core := range []mcp.Core{{}, {View: &window{}}, onTheList(t).core} {
		words, err := mcp.Vocabulary(t.Context(), core)
		if err != nil {
			t.Fatal(err)
		}
		for name, said := range words {
			if said.Kind == port.StepCalling {
				t.Errorf("%s says nothing about what it does", name)
			}
		}
	}
}

// The arguments an edit is drawn from are the ones it is declared with. A name
// that drifted would leave a change nobody could place.
func TestAnEditNamesTheArgumentsItReplacesTextWith(t *testing.T) {
	words, err := mcp.Vocabulary(t.Context(), mcp.Core{})
	if err != nil {
		t.Fatal(err)
	}
	edit := words["note_edit"]
	if edit.Stood != "stood" || edit.Becomes != "becomes" {
		t.Fatalf("an edit replaces %q with %q", edit.Stood, edit.Becomes)
	}
	for _, tool := range []string{"note_write", "note_read"} {
		if words[tool].Stood != "" || words[tool].Becomes != "" {
			t.Errorf("%s claims to replace a stretch", tool)
		}
	}
}
