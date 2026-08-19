package index

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// noted puts one note in, with the headings given inside it.
func noted(t *testing.T, db *DB, vault domain.Vault, path, title string, headings ...string) {
	t.Helper()

	n := domain.Note{
		Ref:   domain.FileRef{Path: path, Kind: domain.KindNote, Size: 100, MTime: 1},
		Title: title,
		Body:  strings.Join(headings, "\n"),
	}
	for at, heading := range headings {
		n.Headings = append(n.Headings, domain.Heading{Level: 2, Text: heading, Line: at * 2})
	}
	if err := db.Notes().Save(t.Context(), vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

// named is the names one vault answers a query with.
func named(t *testing.T, db *DB, vault domain.Vault, query string) []domain.NameMatch {
	t.Helper()

	found, err := db.NoteQueries().Names(t.Context(), vault.ID, query, 20)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// marked is the name a match stands for, with the runs that matched wrapped in
// brackets, so a test reads what a person would see.
func marked(m domain.NameMatch) string {
	text := m.Title
	if m.Heading != "" {
		text = m.Heading
	}
	runes := []rune(text)
	// The runs come in order and do not overlap, so they are put in from the
	// end and no offset moves before it is used.
	for at := len(m.At) - 1; at >= 0; at-- {
		span := m.At[at]
		runes = append(runes[:span.From],
			append([]rune("["+string(runes[span.From:span.To])+"]"), runes[span.To:]...)...)
	}
	return string(runes)
}

func TestANoteIsFoundByItsOwnTitle(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/entropy.md", "Entropy")

	found := named(t, db, first, "ent")

	if len(found) != 1 {
		t.Fatalf("found %d names, want 1: %+v", len(found), found)
	}
	if found[0].Path != "notes/entropy.md" || found[0].Heading != "" {
		t.Errorf("found %+v, want the note itself", found[0])
	}
	// A word reached by its prefix is marked whole: the index matched the
	// word, and the word is what it has to say.
	if got := marked(found[0]); got != "[Entropy]" {
		t.Errorf("marked %q, want %q — the run that matched is what says why", got, "[Entropy]")
	}
}

func TestANoteIsFoundByAHeadingInsideIt(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/carnot.md", "The Carnot cycle", "Entropy over one cycle")

	found := named(t, db, first, "entropy")

	if len(found) != 1 {
		t.Fatalf("found %d names, want 1: %+v", len(found), found)
	}
	if found[0].Title != "The Carnot cycle" {
		t.Errorf("the note is called %q, want the note the heading stands in", found[0].Title)
	}
	if found[0].Line != 0 {
		t.Errorf("the heading stands on line %d, want 0", found[0].Line)
	}
	if got := marked(found[0]); got != "[Entropy] over one cycle" {
		t.Errorf("marked %q, want the run inside the heading", got)
	}
}

func TestATitleComesBeforeAHeading(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/carnot.md", "The Carnot cycle", "Entropy over one cycle")
	noted(t, db, first, "notes/entropy.md", "Entropy")

	found := named(t, db, first, "entropy")

	if len(found) != 2 {
		t.Fatalf("found %d names, want 2: %+v", len(found), found)
	}
	if found[0].Heading != "" {
		t.Errorf("the first answer is a heading, want the note called Entropy: %+v", found[0])
	}
	if found[1].Heading == "" {
		t.Errorf("the second answer is a title, want the heading: %+v", found[1])
	}
}

func TestOneNoteContributesOnlySoManyHeadings(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/thermo.md", "Thermodynamics",
		"Entropy and heat", "Entropy and work", "Entropy and time", "Entropy and information")

	found := named(t, db, first, "entropy")

	if len(found) != mostHeadingsOfANote {
		t.Fatalf("found %d headings of one note, want %d: %+v",
			len(found), mostHeadingsOfANote, found)
	}
}

// mostHeadingsOfANote is what the adapter caps one note's headings at. It is
// stated here so that the test says what it is asserting.
const mostHeadingsOfANote = 2

// TestOneNoteDoesNotCrowdOutTheOthers is the order the two cuts are made in. A
// note with more matching headings than the whole answer holds is thinned
// first, so the notes ranking below it still reach the person.
func TestOneNoteDoesNotCrowdOutTheOthers(t *testing.T) {
	db := opened(t)

	crowded := make([]string, 0, 40)
	for i := range 40 {
		crowded = append(crowded, "Entropy "+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	noted(t, db, first, "notes/thermo.md", "Thermodynamics", crowded...)
	noted(t, db, first, "notes/engine.md", "Engines", "Entropy of an engine")
	noted(t, db, first, "notes/time.md", "Time", "Entropy and the arrow")

	found := named(t, db, first, "entropy")

	notes := map[string]int{}
	for _, m := range found {
		notes[m.Path]++
	}
	if notes["notes/engine.md"] != 1 || notes["notes/time.md"] != 1 {
		t.Errorf("the other notes answered %d and %d times, want one each: %+v",
			notes["notes/engine.md"], notes["notes/time.md"], found)
	}
	if notes["notes/thermo.md"] != mostHeadingsOfANote {
		t.Errorf("the crowded note answered %d times, want %d",
			notes["notes/thermo.md"], mostHeadingsOfANote)
	}
}

func TestANameGoesWhenTheNoteDoes(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/entropy.md", "Entropy", "Entropy and heat")

	if err := db.Notes().Remove(t.Context(), first.ID, []string{"notes/entropy.md"}); err != nil {
		t.Fatal(err)
	}

	if found := named(t, db, first, "entropy"); len(found) != 0 {
		t.Errorf("found %+v after the note was removed, want nothing", found)
	}
}

func TestAHeadingThatWasTakenOutOfANoteStopsAnswering(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/carnot.md", "The Carnot cycle", "Entropy over one cycle")
	noted(t, db, first, "notes/carnot.md", "The Carnot cycle", "Work over one cycle")

	if found := named(t, db, first, "entropy"); len(found) != 0 {
		t.Errorf("found %+v, want nothing: the heading is no longer in the note", found)
	}
	if found := named(t, db, first, "work"); len(found) != 1 {
		t.Errorf("found %d names for the heading that is there now, want 1", len(found))
	}
}

func TestATitleThatChangedStopsAnsweringUnderTheOldOne(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/entropy.md", "Entropy")
	noted(t, db, first, "notes/entropy.md", "Enthalpy")

	if found := named(t, db, first, "entropy"); len(found) != 0 {
		t.Errorf("found %+v under the title the note no longer carries, want nothing", found)
	}
	if found := named(t, db, first, "enthalpy"); len(found) != 1 {
		t.Errorf("found %d names under the title it now carries, want 1", len(found))
	}
}

func TestANameStaysInsideItsVault(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/entropy.md", "Entropy", "Entropy and heat")
	noted(t, db, second, "notes/engine.md", "Engine", "Engine and work")

	if found := named(t, db, second, "entropy"); len(found) != 0 {
		t.Errorf("the second vault answered %+v about the first, want nothing", found)
	}
	if found := named(t, db, first, "engine"); len(found) != 0 {
		t.Errorf("the first vault answered %+v about the second, want nothing", found)
	}

	// Each answers about itself, so neither is silent for a reason of its own.
	if found := named(t, db, first, "entropy"); len(found) == 0 {
		t.Error("the first vault answered nothing about its own name")
	}
	if found := named(t, db, second, "engine"); len(found) == 0 {
		t.Error("the second vault answered nothing about its own name")
	}
}

func TestOnlyTheLastWordIsMatchedOnItsPrefix(t *testing.T) {
	db := opened(t)
	noted(t, db, first, "notes/one.md", "Entropy of mixing")

	if found := named(t, db, first, "entropy mix"); len(found) != 1 {
		t.Errorf("found %d names for a word still being typed, want 1", len(found))
	}
	if found := named(t, db, first, "entro mixing"); len(found) != 0 {
		t.Errorf("found %+v, want nothing: only the last word is a prefix", found)
	}
}

func TestARunIsCountedTheWayAClientCountsText(t *testing.T) {
	db := opened(t)
	// The emoji is one character to a reader and two code units to a client,
	// so a run counted in characters would land a place early.
	noted(t, db, first, "notes/waving.md", "👋 Entropy")

	found := named(t, db, first, "entropy")

	if len(found) != 1 {
		t.Fatalf("found %d names, want 1", len(found))
	}
	if len(found[0].At) != 1 || found[0].At[0] != (domain.Span{From: 3, To: 10}) {
		t.Errorf("the run is at %+v, want one run from 3 to 10", found[0].At)
	}
}
