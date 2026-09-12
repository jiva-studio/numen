package flashcards_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// reading opens one index and hands back the use case that reads it, together
// with a way to write another vault into it. One index for every vault is what
// the application runs, so a scoping test has somewhere to leak to.
func reading(
	t *testing.T,
) (flashcards.ShowNeighbourhood, func(notes map[string]string) domain.Vault) {
	t.Helper()

	db, err := index.Open(t.Context(), indexfile.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := vaults.Scan{
		Readers: filesystem.VaultReaders{}, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.NoteQueries(), Maintenance: db.Maintenance(),
	}
	add := func(notes map[string]string) domain.Vault {
		t.Helper()
		v := testsupport.NewVault(t, notes)
		if _, err := scan.Execute(t.Context(), v); err != nil {
			t.Fatal(err)
		}
		return v
	}

	return flashcards.ShowNeighbourhood{
		Linked: note.ShowLinks{Links: db.NoteQueries()},
		Notes:  db.NoteQueries(),
		Reads:  note.Read{Readers: filesystem.VaultReaders{}},
	}, add
}

// around is one vault of notes, and what its deck turns out to be joined to.
func around(t *testing.T, notes map[string]string, deck string) flashcards.Neighbourhood {
	t.Helper()
	u, add := reading(t)
	return getNeighbourhood(t, u, add(notes), deck)
}

func getNeighbourhood(
	t *testing.T, u flashcards.ShowNeighbourhood, v domain.Vault, deck string,
) flashcards.Neighbourhood {
	t.Helper()
	out, err := u.Execute(t.Context(), v, deck)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// getWrittenLinks is how each entry was addressed, which is all a link that
// reached no note ever has.
func getWrittenLinks(j flashcards.Neighbourhood) []string {
	out := make([]string, 0, len(j.Notes))
	for _, one := range j.Notes {
		out = append(out, one.Written)
	}
	return out
}

func paths(j flashcards.Neighbourhood) []string {
	out := make([]string, 0, len(j.Notes))
	for _, one := range j.Notes {
		out = append(out, one.Path)
	}
	return out
}

func TestALinkWrittenInsideACardArrivesWithItsText(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	one := j.Notes[0]
	if one.Path != "Migration.md" || one.Title != "Migration" {
		t.Errorf("got %q titled %q", one.Path, one.Title)
	}
	if !strings.Contains(one.Body, "Birds go south") {
		t.Errorf("body is %q", one.Body)
	}
	if one.Backlink {
		t.Error("the deck points at it, and the answer says otherwise")
	}
	if one.Outcome != note.Ok {
		t.Errorf("outcome %q", one.Outcome)
	}
}

// Every card names the stencil it is cut by, so a deck points at its stencils.
// A stencil is how a card is laid out and not what it was written from.
func TestTheStencilACardIsCutByIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"stencils/Word.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n[[stencils/Word]]\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 1 || got[0] != "Migration.md" {
		t.Fatalf("joined to %v", got)
	}
}

// What a deck is joined to is the notes it was written from. A deck names the
// preset that schedules it and may name another deck, and neither is a note
// somebody wrote about the material. It holds on both sides: a deck pointing
// at this one is a deck all the same.
func TestOnlyTheNotesADeckWasWrittenFromAreRead(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n---\n" +
			"\n# Sanskrit\n\nTwenty minutes a day.\n",
		"decks/Roots.md": "---\ntype: deck\n---\n\n# Roots\n\nThe verbs.\n",
		"decks/Songs.md": "---\ntype: deck\n---\n\n# Songs\n\nSung over [[decks/Birds]].\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Sanskrit\n    role: ref\n    type: preset\n" +
			"  - to: decks/Roots\n    role: ref\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nSee [[Migration]] and [[Feathers]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
		"Feathers.md":  "# Feathers\n\nWhat a wing is made of.\n",
	}, "decks/Birds.md")

	want := []string{"Migration.md", "Feathers.md"}
	got := paths(j)
	if len(got) != len(want) {
		t.Fatalf("joined to %v, want %v", got, want)
	}
	for _, path := range want {
		if !slices.Contains(got, path) {
			t.Errorf("joined to %v, want %v", got, want)
		}
	}
	// What was left out was never on its way to being read, so nothing is
	// reported as text a person did not get.
	if j.Unread != 0 {
		t.Errorf("%d of them are said to have come without their text", j.Unread)
	}
}

func TestANotePointingAtTheDeckIsNotOneTheDeckPointsAt(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
		"Migration.md":   "# Migration\n\nWorked at with [[decks/Birds]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if j.Notes[0].Path != "Migration.md" {
		t.Fatalf("got %q", j.Notes[0].Path)
	}
	if !j.Notes[0].Backlink {
		t.Error("a backlink was reported as something the deck points at")
	}
	if !strings.Contains(j.Notes[0].Body, "Worked at with") {
		t.Errorf("body is %q", j.Notes[0].Body)
	}
}

func TestANoteOnBothSidesIsNamedOnce(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":   "# Migration\n\nDrilled in [[decks/Birds]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if j.Notes[0].Backlink {
		t.Error("a note on both sides is one the deck points at")
	}
}

func TestTheDeckDoesNotStandInItsOwnList(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\nGathered in [[decks/Birds]] itself.\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nFiled under [[decks/Birds]].\n" +
			"\n### Meaning\n\nSee [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 || j.Notes[0].Path != "Migration.md" {
		t.Fatalf("joined to %v", paths(j))
	}
}

func TestADanglingLinkKeepsTheNameItWasWrittenBy(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nSee [[Nowhere At All]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	one := j.Notes[0]
	if one.Written != "Nowhere At All" {
		t.Errorf("written %q", one.Written)
	}
	if one.Path != "" || one.Body != "" || one.Title != "" {
		t.Errorf("a dangling link came back with %+v", one)
	}
}

func TestOneNameThatCameLooseIsNamedOnce(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSee [[Nowhere At All]].\n" +
			"\n## Swallow ^3dkmf936tb\n\n### Word\n\nSee [[Nowhere At All]] again.\n",
	}, "decks/Birds.md")

	if got := getWrittenLinks(j); len(got) != 1 || got[0] != "Nowhere At All" {
		t.Fatalf("joined to %v", got)
	}
}

// A name several notes answer to resolves to the nearest, and a person reading
// the wrong note has no other way to find out.
func TestANameSeveralNotesAnswerToIsSaidToBeAmbiguous(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md":     "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"north/Migration.md": "# Migration\n\nOne of the two.\n",
		"south/Migration.md": "# Migration\n\nThe other, answering to the same name.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if !j.Notes[0].Ambiguous {
		t.Errorf("%q answers to a name two notes answer to and is not said to be ambiguous",
			j.Notes[0].Path)
	}
}

// A panel of titles with nothing under any of them says nothing about why, so
// the vault being out of reach is an error and not thirty silent entries.
func TestAVaultOutOfReachIsAnError(t *testing.T) {
	t.Parallel()
	u, add := reading(t)
	v := add(map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":   "# Migration\n\nBirds go south.\n",
	})
	if err := os.RemoveAll(v.Path); err != nil {
		t.Fatal(err)
	}

	if _, err := u.Execute(t.Context(), v, "decks/Birds.md"); err == nil {
		t.Error("a vault that is not there was read around without complaint")
	}
}

func TestAnAttachmentIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n![[asset://diagram.png]]\n\nSee [[Migration]].\n",
		"diagram.png":  "not really a picture",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 || j.Notes[0].Path != "Migration.md" {
		t.Fatalf("joined to %v", paths(j))
	}
}

// A name that came loose is the whole reason a link resolving to nothing is
// kept. An address that never named a note is not a name that came loose, and
// drawing one as a note that is missing says the vault has a question in it
// where it has none.
func TestAnAddressThatNamesNoNoteIsNotADanglingNote(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"https://example.org/birds\"\n    role: ref\n---\n" +
			"\n![[asset://diagram.png]]\n\nSee [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	for _, one := range j.Notes {
		if one.Path == "" && one.Written != "Migration" {
			t.Errorf("%q is drawn as a note whose name came loose", one.Written)
		}
	}
	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", getWrittenLinks(j))
	}
}

func TestALinkIntoAnotherVaultIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	// An identifier finds its note wherever it is, and where it is may be a
	// vault this reading has no reader for. Both vaults file a note at the same
	// path, so a reading that ignores which vault the link landed in shows the
	// wrong note rather than nothing.
	const id = "01M02DTC80PABQQW3XS3XWDVHW"
	u, add := reading(t)
	v := add(map[string]string{
		"decks/Birds.md": "---\ntype: deck\nlinks:\n  - to: \"note://" + id +
			"\"\n    role: jump\n---\n\n# The ones with feathers\n",
		"elsewhere.md": "# A namesake at home\n",
	})
	add(map[string]string{
		"elsewhere.md": "---\nid: " + id + "\n---\n\n# In the other vault\n",
	})

	j := getNeighbourhood(t, u, v, "decks/Birds.md")
	if len(j.Notes) != 0 {
		t.Fatalf("joined to %v", paths(j))
	}
}

func TestANoteDeletedAfterTheScanDoesNotSinkTheOthers(t *testing.T) {
	t.Parallel()
	u, add := reading(t)
	v := add(map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nSee [[Migration]] and [[Feathers]].\n",
		"Migration.md":   "# Migration\n\nBirds go south.\n",
		"Feathers.md":    "# Feathers\n\nBarbs on a shaft.\n",
	})
	if err := os.Remove(filepath.Join(v.Path, "Migration.md")); err != nil {
		t.Fatal(err)
	}

	j := getNeighbourhood(t, u, v, "decks/Birds.md")
	if len(j.Notes) != 2 {
		t.Fatalf("joined to %v", paths(j))
	}
	byPath := map[string]flashcards.Neighbour{}
	for _, one := range j.Notes {
		byPath[one.Path] = one
	}
	gone := byPath["Migration.md"]
	if gone.Outcome != note.Missing || gone.Body != "" {
		t.Errorf("the deleted note came back as %+v", gone)
	}
	if !strings.Contains(byPath["Feathers.md"].Body, "Barbs on a shaft") {
		t.Errorf("the note beside it came back as %+v", byPath["Feathers.md"])
	}
}

func TestPastTheThirtiethNoteTheTextIsLeftUnread(t *testing.T) {
	t.Parallel()
	const pointing = flashcards.MostRead + 5
	notes := map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
	}
	for i := range pointing {
		name := fmt.Sprintf("Note %02d", i)
		notes[name+".md"] = "# " + name + "\n\nDrilled in [[decks/Birds]].\n"
	}
	j := around(t, notes, "decks/Birds.md")

	if len(j.Notes) != pointing {
		t.Fatalf("joined to %d notes", len(j.Notes))
	}
	if j.Unread != 5 {
		t.Errorf("said %d came without their text", j.Unread)
	}
	for i, one := range j.Notes {
		read := one.Body != ""
		if want := i < flashcards.MostRead; read != want {
			t.Errorf("note %d of %d: read=%v", i, len(j.Notes), read)
		}
		if one.Title == "" {
			t.Errorf("note %d came back unnamed", i)
		}
	}
}

func TestADeckJoinedToNothingIsAnEmptyAnswer(t *testing.T) {
	t.Parallel()
	j := around(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 0 || j.Unread != 0 {
		t.Fatalf("got %+v", j)
	}
}

func TestOneVaultsDeckIsNotJoinedToAnothersNotes(t *testing.T) {
	t.Parallel()
	// Both vaults file their deck at the same path, so a query that forgets its
	// vault answers with the other one's notes rather than with nothing.
	u, add := reading(t)
	feathers := add(map[string]string{
		"decks/Deck.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":  "# Migration\n\nBirds go south when the days shorten.\n",
	})
	sums := add(map[string]string{
		"decks/Deck.md": "---\ntype: deck\n---\n\nCut from [[Arithmetic]].\n",
		"Arithmetic.md": "# Arithmetic\n\nRemainders divide integers evenly.\n",
	})

	first := getNeighbourhood(t, u, feathers, "decks/Deck.md")
	if len(first.Notes) != 1 || first.Notes[0].Path != "Migration.md" {
		t.Fatalf("the first vault is joined to %v", paths(first))
	}
	if strings.Contains(first.Notes[0].Body, "Remainders") {
		t.Errorf("the first vault read the second's note: %q", first.Notes[0].Body)
	}

	second := getNeighbourhood(t, u, sums, "decks/Deck.md")
	if len(second.Notes) != 1 || second.Notes[0].Path != "Arithmetic.md" {
		t.Fatalf("the second vault is joined to %v", paths(second))
	}
	if strings.Contains(second.Notes[0].Body, "days shorten") {
		t.Errorf("the second vault read the first's note: %q", second.Notes[0].Body)
	}
}
