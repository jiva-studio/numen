package flashcards_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A card is scheduled by the preset of the deck it stands in, so the window
// under each of the four is the moment that scheduler brings the card back.
func TestTheWindowsUnderTheFourAreTheCardsOwnSchedulers(t *testing.T) {
	t.Parallel()
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
		on := review.CardFaceID{Card: card, Face: "Say it"}
		if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}

	sitting := flashcards.Session{
		Marking: s.marking, Standings: s.standings, Schedules: s.kept,
		Presets: s.presets, Day: today, Now: func() time.Time { return at },
	}
	held, err := sitting.Execute(t.Context(), s.vault, flashcards.Over{})
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
		want := review.NewFSRSAt(share).Next(one.Schedule, at, review.Good).Due.Sub(at)
		if one.Ahead[review.Good] != want {
			t.Errorf("%s comes back in %v at a share of %g, and the window says %v",
				one.Deck, want, share, one.Ahead[review.Good])
		}
		seen[share] = want
	}
	if seen[0.99] == seen[0.9] {
		t.Fatal("the two presets leave the card in the same place, and this test says nothing")
	}
}

// The window under a button is the day the card comes back on.
//
// A card's day is chosen in one function: the scheduler works out an interval
// and the preset chooses a day inside the tolerance around it. The button goes
// through the same one, so a preset evening its load names the day it moved the
// card to.
func TestTheWindowsUnderTheFourNameTheDayTheCardComesBackOn(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		what string
		even bool
	}{{"an even load", true}, {"no even load", false}} {
		s := opened(t, map[string]string{
			"Term.md": term,
			"On.md": preset(fmt.Sprintf("goal: minutes_a_day\nminutes_a_day: 1440\n"+
				"new_a_day: 0\nreviews_a_day: 9999\neven_load: %t\n", one.even)),
			"decks/On.md": deckOf("On", 60, 0),
		})
		// Every card face put into review long ago, so each comes back in days
		// and the placement has a window to move it in.
		for _, days := range []int{40, 25} {
			run := s.run(t, saturday.AddDate(0, 0, -days))
			for i := range 60 {
				answer(t, run, mark(i), 4*time.Second)
			}
		}

		// One card answered at a time, each a second after the last, so the day
		// stays the one day and nothing but the placement can move a card off
		// the day the scheduler named. What a button names is a day, so the day
		// is what is compared.
		asked, differ, worst := 0, 0, time.Duration(0)
		for step := range 60 {
			at := saturday.Add(time.Duration(step) * time.Second)
			sat := s.sittingAt(t, today, at)
			if len(sat.Asked) == 0 {
				break
			}
			card := sat.Asked[0]
			said, named := card.Ahead[review.Good]
			if !named {
				t.Fatalf("under %s a card was asked with no window under its buttons", one.what)
			}
			if _, err := s.run(t, at).Answer(
				t.Context(), card.CardFace, review.Good, 0,
			); err != nil {
				t.Fatal(err)
			}
			schedules, err := s.kept.Execute(t.Context(), s.vault)
			if err != nil {
				t.Fatal(err)
			}
			asked++
			came, names := schedules[card.CardFace].Due, at.Add(said)
			if today.Names(came) != today.Names(names) {
				differ++
				if off := came.Sub(names); off > worst || -off > worst {
					worst = max(off, -off)
				}
			}
		}
		if asked == 0 {
			t.Fatalf("under %s no card was asked, and this test says nothing", one.what)
		}
		if differ != 0 {
			t.Errorf("under %s %d of %d cards came back somewhere else than their button said, "+
				"the worst by %v", one.what, differ, asked, worst)
		}
	}
}
