package neighborhood_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/neighborhood"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
)

func reading(
	t *testing.T,
) (neighborhood.Show, func(notes map[string]string) domain.Vault) {
	t.Helper()

	installation := vaulttest.NewInstallation()
	add := func(notes map[string]string) domain.Vault {
		t.Helper()
		return installation.Add(t, notes)
	}

	return neighborhood.NewShow(
		note.NewShowLinks(installation.Queries),
		installation.Queries,
		note.NewRead(installation.Ports.Readers),
	), add
}

func readNeighborhood(t *testing.T, notes map[string]string, deck string) neighborhood.Neighborhood {
	t.Helper()
	u, add := reading(t)
	return getNeighborhood(t, u, add(notes), deck)
}

func getNeighborhood(
	t *testing.T, u neighborhood.Show, v domain.Vault, deck string,
) neighborhood.Neighborhood {
	t.Helper()
	out, err := u.Execute(t.Context(), v, deck)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func getWrittenLinks(j neighborhood.Neighborhood) []string {
	out := make([]string, 0, len(j.Notes))
	for _, one := range j.Notes {
		out = append(out, one.Written)
	}
	return out
}

func paths(j neighborhood.Neighborhood) []string {
	out := make([]string, 0, len(j.Notes))
	for _, one := range j.Notes {
		out = append(out, one.Path)
	}
	return out
}

func TestALinkSavedInsideACardArrivesWithItsText(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
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
	if one.IsBacklink {
		t.Error("the deck points at it, and the answer says otherwise")
	}
	if one.Outcome != note.Ok {
		t.Errorf("outcome %q", one.Outcome)
	}
}

// Every card names the card model it is cut by, so a deck points at its card models.
// A card model is how a card is laid out and not what it was written from.
func TestTheCardModelACardIsSplitByIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"card-models/Word.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n[[card-models/Word]]\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 1 || got[0] != "Migration.md" {
		t.Fatalf("joined to %v", got)
	}
}

// A deck is joined to the notes it was written from. A deck names the preset
// that schedules it and may name another deck, and neither is a note written
// about the material. This holds on both sides: a deck pointing at this one is
// still a deck.
func TestOnlyTheNotesADeckWasSavedFromAreRead(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
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
	// Excluded notes are never read, so none is reported as unread.
	if j.Unread != 0 {
		t.Errorf("%d of them are said to have come without their text", j.Unread)
	}
}

func TestANotePointingAtTheDeckIsNotOneTheDeckPointsAt(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
		"Migration.md":   "# Migration\n\nWorked at with [[decks/Birds]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if j.Notes[0].Path != "Migration.md" {
		t.Fatalf("got %q", j.Notes[0].Path)
	}
	if !j.Notes[0].IsBacklink {
		t.Error("a backlink was reported as something the deck points at")
	}
	if !strings.Contains(j.Notes[0].Body, "Worked at with") {
		t.Errorf("body is %q", j.Notes[0].Body)
	}
}

func TestANoteOnBothSidesIsNamedOnce(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":   "# Migration\n\nDrilled in [[decks/Birds]].\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if j.Notes[0].IsBacklink {
		t.Error("a note on both sides is one the deck points at")
	}
}

func TestTheDeckDoesNotStandInItsOwnList(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
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

func TestADanglingLinkKeepsTheNameItWasSavedBy(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
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
	j := readNeighborhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSee [[Nowhere At All]].\n" +
			"\n## Swallow ^3dkmf936tb\n\n### Word\n\nSee [[Nowhere At All]] again.\n",
	}, "decks/Birds.md")

	if got := getWrittenLinks(j); len(got) != 1 || got[0] != "Nowhere At All" {
		t.Fatalf("joined to %v", got)
	}
}

// A name shared by several notes resolves to the nearest one, and the reader
// cannot detect the ambiguity without the mark.
func TestReadNeighborhood_MarksAmbiguousName(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"decks/Birds.md":     "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"north/Migration.md": "# Migration\n\nOne of the two.\n",
		"south/Migration.md": "# Migration\n\nThe other, answering to the same name.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 {
		t.Fatalf("joined to %v", paths(j))
	}
	if !j.Notes[0].IsAmbiguous {
		t.Errorf("%q answers to a name two notes answer to and is not said to be ambiguous",
			j.Notes[0].Path)
	}
}

// A panel of titles with no content gives no reason, so an unreachable vault is
// an error and not a list of empty entries.
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
	j := readNeighborhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n" +
			"\n![[asset://diagram.png]]\n\nSee [[Migration]].\n",
		"diagram.png":  "not really a picture",
		"Migration.md": "# Migration\n\nBirds go south.\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 1 || j.Notes[0].Path != "Migration.md" {
		t.Fatalf("joined to %v", paths(j))
	}
}

// A link that resolves to nothing is retained only for a name that resolved
// earlier. An address that never named a note is not such a name, and displaying
// it as a missing note reports a question the vault does not have.
func TestAnAddressThatNamesNoNoteIsNotADanglingNote(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
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
	// wrong note and not nothing.
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

	j := getNeighborhood(t, u, v, "decks/Birds.md")
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

	j := getNeighborhood(t, u, v, "decks/Birds.md")
	if len(j.Notes) != 2 {
		t.Fatalf("joined to %v", paths(j))
	}
	byPath := map[string]neighborhood.Neighbor{}
	for _, one := range j.Notes {
		byPath[one.Path] = one
	}
	removed := byPath["Migration.md"]
	if removed.Outcome != note.Missing || removed.Body != "" {
		t.Errorf("the deleted note came back as %+v", removed)
	}
	if !strings.Contains(byPath["Feathers.md"].Body, "Barbs on a shaft") {
		t.Errorf("the note beside it came back as %+v", byPath["Feathers.md"])
	}
}

func TestPastTheThirtiethNoteTheTextIsLeftUnread(t *testing.T) {
	t.Parallel()
	const pointing = neighborhood.MostRead + 5
	notes := map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
	}
	for i := range pointing {
		name := fmt.Sprintf("Note %02d", i)
		notes[name+".md"] = "# " + name + "\n\nDrilled in [[decks/Birds]].\n"
	}
	j := readNeighborhood(t, notes, "decks/Birds.md")

	if len(j.Notes) != pointing {
		t.Fatalf("joined to %d notes", len(j.Notes))
	}
	if j.Unread != 5 {
		t.Errorf("said %d came without their text", j.Unread)
	}
	for i, one := range j.Notes {
		read := one.Body != ""
		if want := i < neighborhood.MostRead; read != want {
			t.Errorf("note %d of %d: read=%v", i, len(j.Notes), read)
		}
		if one.Title == "" {
			t.Errorf("note %d came back unnamed", i)
		}
	}
}

func TestADeckJoinedToNothingIsAnEmptyAnswer(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"decks/Birds.md": "---\ntype: deck\n---\n\n# The ones with feathers\n",
	}, "decks/Birds.md")

	if len(j.Notes) != 0 || j.Unread != 0 {
		t.Fatalf("got %+v", j)
	}
}

func TestOneVaultsDeckIsNotJoinedToAnothersNotes(t *testing.T) {
	t.Parallel()
	// Both vaults file their deck at the same path, so a query that forgets its
	u, add := reading(t)
	feathers := add(map[string]string{
		"decks/Deck.md": "---\ntype: deck\n---\n\nCut from [[Migration]].\n",
		"Migration.md":  "# Migration\n\nBirds go south when the days shorten.\n",
	})
	sums := add(map[string]string{
		"decks/Deck.md": "---\ntype: deck\n---\n\nCut from [[Arithmetic]].\n",
		"Arithmetic.md": "# Arithmetic\n\nRemainders divide integers evenly.\n",
	})

	first := getNeighborhood(t, u, feathers, "decks/Deck.md")
	if len(first.Notes) != 1 || first.Notes[0].Path != "Migration.md" {
		t.Fatalf("the first vault is joined to %v", paths(first))
	}
	if strings.Contains(first.Notes[0].Body, "Remainders") {
		t.Errorf("the first vault read the second's note: %q", first.Notes[0].Body)
	}

	second := getNeighborhood(t, u, sums, "decks/Deck.md")
	if len(second.Notes) != 1 || second.Notes[0].Path != "Arithmetic.md" {
		t.Fatalf("the second vault is joined to %v", paths(second))
	}
	if strings.Contains(second.Notes[0].Body, "days shorten") {
		t.Errorf("the second vault read the first's note: %q", second.Notes[0].Body)
	}
}

// A deck points at the preset that schedules it, so the preset is among the
// notes the deck is joined to. A preset defines the schedule and is not a source
// of the cards.
func TestThePresetADeckIsScheduledByIsNotSomethingToRead(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"presets/Steady.md": "---\ntype: preset\ngoal: retention\nretention: 0.95\n---\n" +
			"\n# Steady\n\nA slower climb through the same cards.\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"presets/Steady\"\n    role: ref\n    type: preset\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n" +
			"\n### Meaning\n\nA bird that sleeps flying. See [[Migration]].\n",
		"Migration.md": "# Migration\n\nBirds go south when the days shorten.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 1 || got[0] != "Migration.md" {
		t.Fatalf("joined to %v", got)
	}
}

// A preset written into a note that points back at the deck is read like any
// other note: the deck's own link determines which preset schedules it.
func TestAPresetIsLeftOutOfEveryCardsPanel(t *testing.T) {
	t.Parallel()
	j := readNeighborhood(t, map[string]string{
		"presets/Steady.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n---\n" +
			"\n# Steady\n",
		"decks/Birds.md": "---\ntype: deck\nlinks:\n" +
			"  - to: \"presets/Steady\"\n    role: ref\n    type: preset\n---\n" +
			"\n## Swift ^k7m2xq9fzp\n\n### Word\n\nSwift\n\n### Meaning\n\nA bird.\n",
	}, "decks/Birds.md")

	if got := paths(j); len(got) != 0 {
		t.Fatalf("joined to %v", got)
	}
}
