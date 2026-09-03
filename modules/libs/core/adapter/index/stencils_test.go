package index

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// typed puts one note in, of the kind its file says it is.
func typed(t *testing.T, db *DB, vault domain.Vault, path, title string, kind domain.NoteType) {
	t.Helper()

	n := domain.Note{
		Ref:   domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: 100, ModTime: 1},
		Title: title,
		Type:  kind,
		Body:  title,
	}
	if err := db.Notes().Save(t.Context(), string(vault.ID), []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// stencils is the stencils one vault answers with.
func stencils(t *testing.T, db *DB, vault domain.Vault) []port.Stencil {
	t.Helper()

	found, err := db.NoteQueries().Stencils(t.Context(), string(vault.ID))
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// A stencil is found by what the index holds, and a note beside it is not.
func TestTheStencilsOfAVaultAreTheNotesThatSaySoAndNoOthers(t *testing.T) {
	db := opened(t)
	typed(t, db, first, "stencils/animal.md", "Animal", domain.TypeStencil)
	typed(t, db, first, "stencils/verb.md", "Verb", domain.TypeStencil)
	typed(t, db, first, "decks/mammals.md", "Mammals", domain.TypeDeck)
	typed(t, db, first, "notes/entropy.md", "Entropy", domain.TypeNote)

	found := stencils(t, db, first)

	if len(found) != 2 {
		t.Fatalf("found %d stencils, want 2: %+v", len(found), found)
	}
	// By path, so that two runs over one vault answer in one order.
	if found[0].Path != "stencils/animal.md" || found[1].Path != "stencils/verb.md" {
		t.Errorf("found %+v, want them by path", found)
	}
	if found[0].Title != "Animal" || found[1].Title != "Verb" {
		t.Errorf("found %+v, want the titles the files carry", found)
	}
}

// A note carrying no type is a note, and the stencils of a vault do not include
// it.
func TestANoteWhoseFileSaysNothingIsANote(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/entropy.md", "Entropy")

	if found := stencils(t, db, first); len(found) != 0 {
		t.Errorf("a note that says nothing about itself answered as %+v", found)
	}
	held, err := db.NoteQueries().Types(t.Context(), string(first.ID), []string{"notes/entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if held["notes/entropy.md"] != domain.TypeNote {
		t.Errorf("the index calls it %q, want %q", held["notes/entropy.md"], domain.TypeNote)
	}
}

// One database holds every vault, so a question that forgets its vault answers
// with another vault's stencils and reports nothing wrong.
//
// The two vaults share no word, and both directions are asserted: a leak in
// either is a stencil whose title cannot be mistaken for one of this vault's.
func TestTheStencilsOfOneVaultAreNotAnotherVaultsStencils(t *testing.T) {
	db := opened(t)
	typed(t, db, first, "stencils/animal.md", "Animal", domain.TypeStencil)
	typed(t, db, first, "decks/mammals.md", "Mammals", domain.TypeDeck)
	typed(t, db, second, "patterns/quasar.md", "Quasar", domain.TypeStencil)
	typed(t, db, second, "collections/pulsars.md", "Pulsars", domain.TypeDeck)

	for _, held := range []struct {
		vault domain.Vault
		want  string
	}{
		{first, "stencils/animal.md"},
		{second, "patterns/quasar.md"},
	} {
		found := stencils(t, db, held.vault)
		if len(found) != 1 {
			t.Fatalf("%s answered with %d stencils, want 1: %+v", held.vault.Name, len(found), found)
		}
		if found[0].Path != held.want {
			t.Errorf("%s answered with %s, and it holds %s", held.vault.Name, found[0].Path, held.want)
		}
	}
}

// The type of every entry of one folder is one question, so a file tree tells a
// deck from a note without opening either.
func TestTheTypeOfEveryEntryOfAFolderIsOneQuestion(t *testing.T) {
	db := opened(t)
	typed(t, db, first, "cards/mammals.md", "Mammals", domain.TypeDeck)
	typed(t, db, first, "cards/animal.md", "Animal", domain.TypeStencil)
	typed(t, db, first, "cards/reading.md", "Reading", domain.TypeNote)
	typed(t, db, second, "cards/quasars.md", "Quasars", domain.TypeDeck)

	held, err := db.NoteQueries().Types(t.Context(), string(first.ID), []string{
		"cards/mammals.md", "cards/animal.md", "cards/reading.md",
		"cards/quasars.md", "cards/llama.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]domain.NoteType{
		"cards/mammals.md": domain.TypeDeck,
		"cards/animal.md":  domain.TypeStencil,
		"cards/reading.md": domain.TypeNote,
	} {
		if held[path] != want {
			t.Errorf("%s is %q, want %q", path, held[path], want)
		}
	}
	// A path of another vault and a path the index holds nothing at are both
	// absent, and a listing draws such an entry as the file it is.
	if _, drawn := held["cards/quasars.md"]; drawn {
		t.Error("a folder of one vault was told the type of another vault's note at the same path")
	}
	if _, drawn := held["cards/llama.jpg"]; drawn {
		t.Error("a file the index holds no note at was given a type")
	}
}

// A note saved again as a deck is a deck, and the column carries the change.
func TestSavingANoteAgainWritesTheTypeItNowCarries(t *testing.T) {
	db := opened(t)
	typed(t, db, first, "cards/mammals.md", "Mammals", domain.TypeNote)
	typed(t, db, first, "cards/mammals.md", "Mammals", domain.TypeDeck)

	held, err := db.NoteQueries().Types(t.Context(), string(first.ID), []string{"cards/mammals.md"})
	if err != nil {
		t.Fatal(err)
	}
	if held["cards/mammals.md"] != domain.TypeDeck {
		t.Errorf("the index calls it %q, want %q", held["cards/mammals.md"], domain.TypeDeck)
	}
}
