package index

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// divided puts one note in, divided by the headings given at the levels given.
func divided(t *testing.T, db *DB, vault domain.Vault, path string, headings ...domain.Heading) {
	t.Helper()

	text := make([]string, 0, len(headings))
	for _, h := range headings {
		text = append(text, h.Text)
	}
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: 100, ModTime: walked},
		Title:       path,
		Body:        strings.Join(text, "\n"),
		Headings:    headings,
	}
	if err := db.Notes().Save(t.Context(), vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// divisions is what the index says the notes asked about are divided into.
func divisions(
	t *testing.T,
	db *DB,
	vault domain.Vault,
	paths ...string,
) map[string][]domain.Heading {
	t.Helper()

	found, err := db.NoteQueries().Headings(t.Context(), vault.ID, paths)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestAHeadingComesBackWithItsLevelAndItsLine(t *testing.T) {
	db := opened(t)
	divided(t, db, first, "notes/entropy.md",
		domain.Heading{Level: 1, Text: "What it is", Line: 0},
		domain.Heading{Level: 2, Text: "Where it came from", Line: 4},
	)

	found := divisions(t, db, first, "notes/entropy.md")["notes/entropy.md"]
	if len(found) != 2 {
		t.Fatalf("headings = %d, want 2", len(found))
	}
	if found[0].Level != 1 || found[0].Text != "What it is" || found[0].Line != 0 {
		t.Errorf("first = %+v", found[0])
	}
	if found[1].Level != 2 || found[1].Text != "Where it came from" || found[1].Line != 4 {
		t.Errorf("second = %+v", found[1])
	}
}

func TestHeadingsComeBackInTheOrderTheyStand(t *testing.T) {
	db := opened(t)
	divided(t, db, first, "notes/entropy.md",
		domain.Heading{Level: 1, Text: "Last", Line: 12},
		domain.Heading{Level: 1, Text: "First", Line: 2},
		domain.Heading{Level: 1, Text: "Middle", Line: 7},
	)

	var read []string
	for _, h := range divisions(t, db, first, "notes/entropy.md")["notes/entropy.md"] {
		read = append(read, h.Text)
	}
	want := []string{"First", "Middle", "Last"}
	if strings.Join(read, ",") != strings.Join(want, ",") {
		t.Errorf("headings = %v, want %v", read, want)
	}
}

func TestANoteWithNoHeadingsIsAbsent(t *testing.T) {
	db := opened(t)
	divided(t, db, first, "notes/plain.md")

	if found := divisions(t, db, first, "notes/plain.md"); len(found) != 0 {
		t.Errorf("found = %v, want nothing", found)
	}
}

func TestAPathThatNamesNothingIsAbsent(t *testing.T) {
	db := opened(t)
	divided(t, db, first, "notes/entropy.md",
		domain.Heading{Level: 1, Text: "What it is", Line: 0},
	)

	found := divisions(t, db, first, "notes/entropy.md", "notes/gone.md")
	if len(found) != 1 {
		t.Errorf("found = %v, want the one note that is there", found)
	}
}

func TestEachNoteAskedAboutIsAnsweredForItself(t *testing.T) {
	db := opened(t)
	divided(t, db, first, "notes/one.md", domain.Heading{Level: 1, Text: "Of one", Line: 0})
	divided(t, db, first, "notes/two.md", domain.Heading{Level: 1, Text: "Of two", Line: 0})

	found := divisions(t, db, first, "notes/one.md", "notes/two.md")
	if found["notes/one.md"][0].Text != "Of one" || found["notes/two.md"][0].Text != "Of two" {
		t.Errorf("found = %v", found)
	}
}

func TestAskingAboutNoNotesAsksTheDatabaseNothing(t *testing.T) {
	db := opened(t)

	if found := divisions(t, db, first); len(found) != 0 {
		t.Errorf("found = %v, want nothing", found)
	}
}
