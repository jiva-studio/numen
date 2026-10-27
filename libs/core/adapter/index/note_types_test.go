package index

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// saveTypedNote puts one note in, of the kind its file says it is.
func saveTypedNote(t *testing.T, db *DB, vault domain.Vault, path, title string, kind domain.NoteType) {
	t.Helper()

	n := domain.Note{
		FileStat: domain.FileStat{Path: path, Kind: domain.KindNote, Size: 100, ModTime: walked},
		Title:    title,
		Type:     kind,
		Body:     title,
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

// The notes of one type in one vault leave out the notes of that type in another vault.
//
// The two vaults share no word, and both directions are asserted: a leak in either is a note whose
// path cannot be mistaken for one of this vault's.
func TestTheNotesOfATypeInOneVaultAreNotAnotherVaultsNotes(t *testing.T) {
	db := openDB(t)
	saveTypedNote(t, db, first, "sheets/animal.md", "Animal", formType)
	saveTypedNote(t, db, first, "samples/mammals.md", "Mammals", sampleType)
	saveTypedNote(t, db, second, "patterns/quasar.md", "Quasar", formType)
	saveTypedNote(t, db, second, "collections/pulsars.md", "Pulsars", sampleType)

	for _, target := range []struct {
		vault domain.Vault
		want  string
	}{
		{first, "sheets/animal.md"},
		{second, "patterns/quasar.md"},
	} {
		found, err := db.NoteQueries().GetPathsOfType(t.Context(), target.vault.ID, formType)
		if err != nil {
			t.Fatal(err)
		}
		if len(found) != 1 || found[0] != target.want {
			t.Errorf("%s answered with %v, and it holds %s", target.vault.Name, found, target.want)
		}
	}
}

// The type of every entry of one folder is one question, so a file tree tells a
// typed note from a plain note without opening either.
func TestTheTypeOfEveryEntryOfAFolderIsOneQuestion(t *testing.T) {
	db := openDB(t)
	saveTypedNote(t, db, first, "mixed/mammals.md", "Mammals", sampleType)
	saveTypedNote(t, db, first, "mixed/animal.md", "Animal", formType)
	saveTypedNote(t, db, first, "mixed/reading.md", "Reading", domain.TypeNote)
	saveTypedNote(t, db, second, "mixed/quasars.md", "Quasars", sampleType)

	target, err := db.NoteQueries().GetTypes(t.Context(), first.ID, []string{
		"mixed/mammals.md", "mixed/animal.md", "mixed/reading.md",
		"mixed/quasars.md", "mixed/llama.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]domain.NoteType{
		"mixed/mammals.md": sampleType,
		"mixed/animal.md":  formType,
		"mixed/reading.md": domain.TypeNote,
	} {
		if target[path] != want {
			t.Errorf("%s is %q, want %q", path, target[path], want)
		}
	}
	// A path of another vault and a path the index holds nothing at are both
	// absent, and a listing draws such an entry as the file it is.
	if _, hasType := target["mixed/quasars.md"]; hasType {
		t.Error("folder received note type from different vault at same path")
	}
	if _, renderedVal := target["mixed/llama.jpg"]; renderedVal {
		t.Error("a file the index holds no note at was given a type")
	}
}

// A note saved again with another type carries that type, and the column holds the change.
func TestSavingANoteAgainWritesTheTypeItNowCarries(t *testing.T) {
	db := openDB(t)
	saveTypedNote(t, db, first, "mixed/mammals.md", "Mammals", domain.TypeNote)
	saveTypedNote(t, db, first, "mixed/mammals.md", "Mammals", sampleType)

	target, err := db.NoteQueries().GetTypes(t.Context(), first.ID, []string{"mixed/mammals.md"})
	if err != nil {
		t.Fatal(err)
	}
	if target["mixed/mammals.md"] != sampleType {
		t.Errorf("the index calls it %q, want %q", target["mixed/mammals.md"], sampleType)
	}
}
