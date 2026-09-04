package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// learnedHour is an hour well inside a review day, on a day every preset here
// carries the whole of its load.
var learnedHour = time.Date(2026, 9, 2, 10, 0, 0, 0, time.Local)

// A vault of three decks: two on a preset that counts a card learned while it
// is likely to be recalled, and one naming no preset at all. A card answered a
// moment ago is learned under the first rule and not under the second, so the
// same answer counts differently in the decks either side of it.
var learnedVault = map[string]string{
	"Term.md":        term,
	"Recall.md":      preset("learned: retention\nretention: 0.9\n"),
	"decks/One.md":   deckOf("Recall", 3, 0),
	"decks/Two.md":   deckOf("Recall", 2, 10),
	"decks/Three.md": deckOf("", 2, 20),
}

// How much of a deck stands learned is counted under the rule the preset that
// deck names counts by, and two decks on one preset are counted under the one
// rule.
func TestADecksLearnedFacesAreCountedByItsOwnPresetsRule(t *testing.T) {
	t.Parallel()
	s := opened(t, learnedVault)

	run := s.run(t, learnedHour)
	answer(t, run, "card000000", 5*time.Second)
	answer(t, run, "card000010", 5*time.Second)
	answer(t, run, "card000020", 5*time.Second)

	owing, err := s.owedAt(today, func() time.Time { return learnedHour.Add(time.Minute) }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]flashcards.DeckCardsDue{
		// The rule is a chance of recall, and a card answered a minute ago is
		// as likely to come back as a card gets.
		"decks/One.md": {Deck: "decks/One.md", Faces: 3, Learned: 1},
		"decks/Two.md": {Deck: "decks/Two.md", Faces: 2, Learned: 1},
		// The defaults count by an interval of three weeks, which a card
		// answered once is nowhere near.
		"decks/Three.md": {Deck: "decks/Three.md", Faces: 2, Learned: 0},
	}
	for _, one := range owing.Decks {
		if one.Learned != want[one.Deck].Learned || one.Faces != want[one.Deck].Faces {
			t.Errorf("%s stands at %d learned of %d, want %d of %d",
				one.Deck, one.Learned, one.Faces,
				want[one.Deck].Learned, want[one.Deck].Faces)
		}
	}
	if len(owing.Decks) != len(want) {
		t.Errorf("the vault came to %d decks, want %d", len(owing.Decks), len(want))
	}
}

// A card nobody has answered is learned under neither rule, so a vault nobody
// has sat down to stands at nothing learned.
func TestADeckNobodyHasAnsweredStandsAtNothingLearned(t *testing.T) {
	t.Parallel()
	s := opened(t, learnedVault)

	owing, err := s.owedAt(today, func() time.Time { return learnedHour }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range owing.Decks {
		if one.Learned != 0 {
			t.Errorf("%s stands at %d learned before anything was answered",
				one.Deck, one.Learned)
		}
	}
}

// A deck holding no cards is counted for nothing: it has no card face to stand
// learned.
func TestADeckOfNoCardsIsCountedForNothing(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Recall.md":      preset("learned: retention\nretention: 0.9\n"),
		"decks/Empty.md": deckOf("Recall", 0, 0),
		"decks/One.md":   deckOf("Recall", 1, 0),
	})

	owing, err := s.owedAt(today, func() time.Time { return learnedHour }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range owing.Decks {
		if one.Deck == "decks/Empty.md" {
			t.Errorf("a deck of no cards came to %+v", one)
		}
	}
}

// How many of a deck's card faces nobody has begun is counted off the same
// pass, so a deck every face of which is unbegun says so on its own row.
func TestADecksUnbegunFacesAreCounted(t *testing.T) {
	t.Parallel()
	s := opened(t, learnedVault)

	before, err := s.owedAt(today, func() time.Time { return learnedHour }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range before.Decks {
		if one.Unbegun != one.Faces {
			t.Errorf("%s stands at %d unbegun of %d, want all of them",
				one.Deck, one.Unbegun, one.Faces)
		}
	}

	run := s.run(t, learnedHour)
	answer(t, run, "card000000", 5*time.Second)

	after, err := s.owedAt(today, func() time.Time { return learnedHour.Add(time.Minute) }).
		Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"decks/One.md": 2, "decks/Two.md": 2, "decks/Three.md": 2}
	for _, one := range after.Decks {
		if one.Unbegun != want[one.Deck] {
			t.Errorf("%s stands at %d unbegun, want %d", one.Deck, one.Unbegun, want[one.Deck])
		}
	}
}
