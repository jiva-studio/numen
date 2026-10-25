package index

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// saveTypedNote puts one note in, of the kind its file says it is.
func saveTypedNote(t *testing.T, db *DB, vault domain.Vault, path, title string, kind domain.NoteType) {
	t.Helper()

	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: 100, ModTime: walked},
		Title:       title,
		Type:        kind,
		Body:        title,
	}
	if err := saveCut(t.Context(), db, vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// A note carrying no type is a note.
func TestANoteWhoseFileReturnsNothingIsANote(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, first, "notes/entropy.md", "Entropy")

	target, err := db.NoteQueries().GetTypes(t.Context(), first.ID, []string{"notes/entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if target["notes/entropy.md"] != domain.TypeNote {
		t.Errorf("the index calls it %q, want %q", target["notes/entropy.md"], domain.TypeNote)
	}
}

// notes of a type and reports nothing wrong.
//
// The two vaults share no word, and both directions are asserted: a leak in either is a note whose
// path cannot be mistaken for one of this vault's.
func TestTheNotesOfATypeInOneVaultAreNotAnotherVaultsNotes(t *testing.T) {
	db := openDB(t)
	saveTypedNote(t, db, first, "sheets/animal.md", "Animal", stencilType)
	saveTypedNote(t, db, first, "decks/mammals.md", "Mammals", deckType)
	saveTypedNote(t, db, second, "patterns/quasar.md", "Quasar", stencilType)
	saveTypedNote(t, db, second, "collections/pulsars.md", "Pulsars", deckType)

	for _, target := range []struct {
		vault domain.Vault
		want  string
	}{
		{first, "sheets/animal.md"},
		{second, "patterns/quasar.md"},
	} {
		found, err := db.NoteQueries().GetPathsOfType(t.Context(), target.vault.ID, stencilType)
		if err != nil {
			t.Fatal(err)
		}
		if len(found) != 1 || found[0] != target.want {
			t.Errorf("%s answered with %v, and it holds %s", target.vault.Name, found, target.want)
		}
	}
}

// The type of every entry of one folder is one question, so a file tree tells a
// deck from a note without opening either.
func TestTheTypeOfEveryEntryOfAFolderIsOneQuestion(t *testing.T) {
	db := openDB(t)
	saveTypedNote(t, db, first, "cards/mammals.md", "Mammals", deckType)
	saveTypedNote(t, db, first, "cards/animal.md", "Animal", stencilType)
	saveTypedNote(t, db, first, "cards/reading.md", "Reading", domain.TypeNote)
	saveTypedNote(t, db, second, "cards/quasars.md", "Quasars", deckType)

	target, err := db.NoteQueries().GetTypes(t.Context(), first.ID, []string{
		"cards/mammals.md", "cards/animal.md", "cards/reading.md",
		"cards/quasars.md", "cards/llama.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]domain.NoteType{
		"cards/mammals.md": deckType,
		"cards/animal.md":  stencilType,
		"cards/reading.md": domain.TypeNote,
	} {
		if target[path] != want {
			t.Errorf("%s is %q, want %q", path, target[path], want)
		}
	}
	// A path of another vault and a path the index holds nothing at are both
	// absent, and a listing draws such an entry as the file it is.
	if _, hasCard := target["cards/quasars.md"]; hasCard {
		t.Error("folder received note type from different vault at same path")
	}
	if _, renderedVal := target["cards/llama.jpg"]; renderedVal {
		t.Error("a file the index holds no note at was given a type")
	}
}

// A note saved again as a deck is a deck, and the column carries the change.
func TestSavingANoteAgainWritesTheTypeItNowCarries(t *testing.T) {
	db := openDB(t)
	saveTypedNote(t, db, first, "cards/mammals.md", "Mammals", domain.TypeNote)
	saveTypedNote(t, db, first, "cards/mammals.md", "Mammals", deckType)

	target, err := db.NoteQueries().GetTypes(t.Context(), first.ID, []string{"cards/mammals.md"})
	if err != nil {
		t.Fatal(err)
	}
	if target["cards/mammals.md"] != deckType {
		t.Errorf("the index calls it %q, want %q", target["cards/mammals.md"], deckType)
	}
}
