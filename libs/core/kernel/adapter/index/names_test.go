package index

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func saveNamedNote(t *testing.T, db *DB, vault domain.Vault, path, title string, headings ...string) {
	t.Helper()

	n := domain.Note{
		FileStat: domain.FileStat{Path: path, Kind: domain.KindNote, Size: 100, ModTime: getWalkedTime()},
		Title:    title,
		Body:     strings.Join(headings, "\n"),
	}
	for item, heading := range headings {
		n.Headings = append(n.Headings, domain.Heading{Level: 2, Text: heading, Line: item * 2})
	}
	if err := saveCut(t.Context(), db, vault.ID, []domain.Note{n}); err != nil {
		t.Fatal(err)
	}
}

func getNames(t *testing.T, db *DB, vault domain.Vault, query string) []domain.NameMatch {
	t.Helper()

	found, err := db.NoteQueries().GetNameMatches(t.Context(), vault.ID, query, 20)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func describeMatch(m domain.NameMatch) string {
	text := m.Title
	if m.Heading != "" {
		text = m.Heading
	}
	runes := []rune(text)
	// The runs come in order and do not overlap, so they are put in from the
	// end and no offset moves before it is used.
	for _, span := range slices.Backward(m.Ranges) {

		runes = append(runes[:span.Start],
			append([]rune("["+string(runes[span.Start:span.End])+"]"), runes[span.End:]...)...)
	}
	return string(runes)
}

func TestANoteIsFoundByItsOwnTitle(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/entropy.md", "Entropy")

	found := getNames(t, db, getFirstVault(), "ent")

	if len(found) != 1 {
		t.Fatalf("found %d names, want 1: %+v", len(found), found)
	}
	if found[0].Path != "notes/entropy.md" || found[0].Heading != "" {
		t.Errorf("found %+v, want the note itself", found[0])
	}
	// A word reached by its prefix is marked whole: the index matched the
	// word, and the word is what it has to say.
	if got := describeMatch(found[0]); got != "[Entropy]" {
		t.Errorf("marked %q, want %q — the run that matched is what says why", got, "[Entropy]")
	}
}

func TestANoteIsFoundByAHeadingInsideIt(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/carnot.md", "The Carnot cycle", "Entropy over one cycle")

	found := getNames(t, db, getFirstVault(), "entropy")

	if len(found) != 1 {
		t.Fatalf("found %d names, want 1: %+v", len(found), found)
	}
	if found[0].Title != "The Carnot cycle" {
		t.Errorf("got note %q, want heading parent note", found[0].Title)
	}
	if found[0].Line != 0 {
		t.Errorf("got heading line %d, want 0", found[0].Line)
	}
	if got := describeMatch(found[0]); got != "[Entropy] over one cycle" {
		t.Errorf("marked %q, want the run inside the heading", got)
	}
}

func TestATitleReachesBeforeAHeading(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/carnot.md", "The Carnot cycle", "Entropy over one cycle")
	saveNamedNote(t, db, getFirstVault(), "notes/entropy.md", "Entropy")

	found := getNames(t, db, getFirstVault(), "entropy")

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
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/thermo.md", "Thermodynamics",
		"Entropy and heat", "Entropy and work", "Entropy and time", "Entropy and information")

	found := getNames(t, db, getFirstVault(), "entropy")

	if len(found) != mostHeadingsOfANote {
		t.Fatalf("found %d headings of one note, want %d: %+v",
			len(found), mostHeadingsOfANote, found)
	}
}

// mostHeadingsOfANote is the cap the adapter applies to one note's headings. It
// is stated here so the test shows the value it asserts against.
const mostHeadingsOfANote = 2

// TestOneNoteDoesNotCrowdOutTheOthers checks the order of the two cuts. A note
// with more matching headings than the whole result holds is thinned first, so
// notes ranked below it still appear in the result.
func TestOneNoteDoesNotCrowdOutTheOthers(t *testing.T) {
	db := openDB(t)

	crowded := make([]string, 0, 40)
	for i := range 40 {
		crowded = append(crowded, "Entropy "+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	saveNamedNote(t, db, getFirstVault(), "notes/thermo.md", "Thermodynamics", crowded...)
	saveNamedNote(t, db, getFirstVault(), "notes/engine.md", "Engines", "Entropy of an engine")
	saveNamedNote(t, db, getFirstVault(), "notes/time.md", "Time", "Entropy and the arrow")

	found := getNames(t, db, getFirstVault(), "entropy")

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
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/entropy.md", "Entropy", "Entropy and heat")

	if err := db.Notes().Remove(t.Context(), getFirstVault().ID, []string{"notes/entropy.md"}); err != nil {
		t.Fatal(err)
	}

	if found := getNames(t, db, getFirstVault(), "entropy"); len(found) != 0 {
		t.Errorf("found %+v after the note was removed, want nothing", found)
	}
}

func TestAHeadingThatWasTakenOutOfANoteStopsAnswering(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/carnot.md", "The Carnot cycle", "Entropy over one cycle")
	saveNamedNote(t, db, getFirstVault(), "notes/carnot.md", "The Carnot cycle", "Work over one cycle")

	if found := getNames(t, db, getFirstVault(), "entropy"); len(found) != 0 {
		t.Errorf("found %+v, want nothing: the heading is no longer in the note", found)
	}
	if found := getNames(t, db, getFirstVault(), "work"); len(found) != 1 {
		t.Errorf("found %d names for the heading that is there now, want 1", len(found))
	}
}

func TestATitleThatChangedStopsAnsweringBelowTheOldOne(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/entropy.md", "Entropy")
	saveNamedNote(t, db, getFirstVault(), "notes/entropy.md", "Enthalpy")

	if found := getNames(t, db, getFirstVault(), "entropy"); len(found) != 0 {
		t.Errorf("found %+v under the title the note no longer carries, want nothing", found)
	}
	if found := getNames(t, db, getFirstVault(), "enthalpy"); len(found) != 1 {
		t.Errorf("found %d names under the title it now carries, want 1", len(found))
	}
}

func TestANameStaysInsideItsVault(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/entropy.md", "Entropy", "Entropy and heat")
	saveNamedNote(t, db, getSecond(), "notes/engine.md", "Engine", "Engine and work")

	if found := getNames(t, db, getSecond(), "entropy"); len(found) != 0 {
		t.Errorf("the second vault answered %+v about the first, want nothing", found)
	}
	if found := getNames(t, db, getFirstVault(), "engine"); len(found) != 0 {
		t.Errorf("the first vault answered %+v about the second, want nothing", found)
	}

	// Each answers about itself, so neither is silent for a reason of its own.
	if found := getNames(t, db, getFirstVault(), "entropy"); len(found) == 0 {
		t.Error("the first vault answered nothing about its own name")
	}
	if found := getNames(t, db, getSecond(), "engine"); len(found) == 0 {
		t.Error("the second vault answered nothing about its own name")
	}
}

func TestOnlyTheLastWordIsMatchedOnItsPrefix(t *testing.T) {
	db := openDB(t)
	saveNamedNote(t, db, getFirstVault(), "notes/one.md", "Entropy of mixing")

	if found := getNames(t, db, getFirstVault(), "entropy mix"); len(found) != 1 {
		t.Errorf("found %d names for a word still being typed, want 1", len(found))
	}
	if found := getNames(t, db, getFirstVault(), "entro mixing"); len(found) != 0 {
		t.Errorf("found %+v, want nothing: only the last word is a prefix", found)
	}
}

func TestARunIsCountedTheWayAClientCountsText(t *testing.T) {
	db := openDB(t)
	// The emoji is one character to a reader and two code units to a client,
	// so a run counted in characters would land a place early.
	saveNamedNote(t, db, getFirstVault(), "notes/waving.md", "👋 Entropy")

	found := getNames(t, db, getFirstVault(), "entropy")

	if len(found) != 1 {
		t.Fatalf("found %d names, want 1", len(found))
	}
	if len(found[0].Ranges) != 1 || found[0].Ranges[0] != (domain.UnitRange{Start: 3, End: 10}) {
		t.Errorf("the run is at %+v, want one run from 3 to 10", found[0].Ranges)
	}
}
