package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A card is scheduled by the preset of the deck it stands in, so the window
// under each of the four is the moment that scheduler brings the card back.
func TestTheWindowsUnderTheFourAreTheCardsOwnSchedulers(t *testing.T) {
	// The share of the cards each deck's preset asks to come back, by the deck.
	shares := map[string]float64{"decks/Roots.md": 0.99, "decks/Terms.md": 0.9}
	s := opened(t, map[string]string{
		"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"Tight.md": "---\ntype: preset\ngoal: retention\nretention: 0.99\n---\n\n# Tight\n",
		"decks/Roots.md": "---\ntype: deck\nlinks:\n" +
			"  - to: Tight\n    role: ref\n    type: preset\n---\n" +
			"\n## Root ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nbhu\n\n### Meaning\n\nto be\n",
		"decks/Terms.md": "---\ntype: deck\n---\n" +
			"\n## Term ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nsandhi\n\n### Meaning\n\njoining\n",
	})

	// Both cards are put into review, so each stands where its own scheduler
	// left it and the four are asked of a schedule and not of a blank.
	at := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	record := s.run(t, at.AddDate(0, 0, -200))
	for _, card := range []string{"k7m2xq9fzp", "3f4g5h6j7k"} {
		on := history.CardFace{Card: card, Face: "Say it"}
		if _, err := record.Answer(t.Context(), on, history.Good, 0); err != nil {
			t.Fatal(err)
		}
	}

	sitting := flashcards.Session{
		Marking: s.marking, Standings: s.standings, Schedules: s.kept,
		Presets: s.presets, Day: today, Now: func() time.Time { return at },
	}
	held, err := sitting.Execute(t.Context(), s.vault, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Asked) != 2 {
		t.Fatalf("%d cards were asked", len(held.Asked))
	}

	seen := map[float64]time.Duration{}
	for _, one := range held.Asked {
		share, named := shares[one.Deck]
		if !named {
			t.Fatalf("a card of %s was asked", one.Deck)
		}
		want := history.NewFSRSAt(share).Next(one.Schedule, at, history.Good).Due.Sub(at)
		if one.Ahead[history.Good] != want {
			t.Errorf("%s comes back in %v at a share of %g, and the window says %v",
				one.Deck, want, share, one.Ahead[history.Good])
		}
		seen[share] = want
	}
	if seen[0.99] == seen[0.9] {
		t.Fatal("the two presets leave the card in the same place, and this test says nothing")
	}
}
