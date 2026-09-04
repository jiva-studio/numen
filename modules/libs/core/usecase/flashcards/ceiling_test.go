package flashcards_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// theCeiling is the minutes_a_day the presets below ask for.
const theCeiling = 20 * time.Minute

// A goal is a ceiling over every deck the preset schedules.
//
// Decks are added and taken away; the goal stands. A person who asked for
// twenty minutes gets twenty minutes, whether one deck names the preset or
// five do.
func TestAGoalIsACeilingOverEveryDeckOfThePreset(t *testing.T) {
	t.Parallel()
	for _, decks := range []int{1, 2, 5} {
		files := map[string]string{
			"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
				"new_a_day: 9999\nreviews_a_day: 9999\n---\n\n# Sanskrit\n",
		}
		for d := range decks {
			deck := "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\n"
			for i := range 60 {
				deck += fmt.Sprintf(
					"\n## Word %d.%d ^c%02d%06d\n\n[[Term]]\n\n### Word\n\nW\n\n### Meaning\n\nM\n",
					d, i, d, i)
			}
			files[fmt.Sprintf("decks/D%d.md", d)] = deck
		}
		files["Term.md"] = vault["Term.md"]

		s := opened(t, files)
		now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
		if took := s.minutes(t, today, now); took != theCeiling {
			t.Errorf("%d decks: the day ran %v, and the goal asks for %v", decks, took, theCeiling)
		}
	}
}

// minutes is how long the whole day of review took, driven the way a person
// drives it.
func (s vaulted) minutes(t *testing.T, day review.Day, now time.Time) time.Duration {
	t.Helper()
	var out time.Duration
	for range 200 {
		sat, err := flashcards.Session{
			Marking: s.marking, Standings: s.standings, Schedules: s.kept,
			Presets: s.presets, Day: day, Now: func() time.Time { return now },
		}.Execute(t.Context(), s.vault, flashcards.Scope{})
		if err != nil || len(sat.Asked) == 0 {
			return out
		}
		record := s.run(t, now)
		for _, one := range sat.Asked {
			cost := review.DefaultCost.Review
			if !one.Schedule.Seen() {
				cost = review.DefaultCost.New
			}
			out += cost
			answer(t, record, one.CardFace.Card, cost)
		}
	}
	t.Fatal("the day never ran out")
	return 0
}

// And the same ceiling holds when a person sits to one deck at a time.
func TestSittingDeckByDeckStaysUnderTheOneCeiling(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
			"new_a_day: 9999\nreviews_a_day: 9999\n---\n\n# Sanskrit\n",
		"Term.md": vault["Term.md"],
	}
	for d := range 3 {
		deck := "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\n"
		for i := range 60 {
			deck += fmt.Sprintf(
				"\n## Word %d.%d ^c%02d%06d\n\n[[Term]]\n\n### Word\n\nW\n\n### Meaning\n\nM\n",
				d, i, d, i)
		}
		files[fmt.Sprintf("decks/D%d.md", d)] = deck
	}

	s := opened(t, files)
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
	var out time.Duration
	for d := range 3 {
		over := flashcards.Scope{Deck: fmt.Sprintf("decks/D%d.md", d)}
		for range 200 {
			sat, err := flashcards.Session{
				Marking: s.marking, Standings: s.standings, Schedules: s.kept,
				Presets: s.presets, Day: today, Now: func() time.Time { return now },
			}.Execute(t.Context(), s.vault, over)
			if err != nil || len(sat.Asked) == 0 {
				break
			}
			record := s.run(t, now)
			for _, one := range sat.Asked {
				cost := review.DefaultCost.Review
				if !one.Schedule.Seen() {
					cost = review.DefaultCost.New
				}
				out += cost
				answer(t, record, one.CardFace.Card, cost)
			}
		}
		if out > theCeiling {
			t.Fatalf("after deck %d the day has run %v, and the goal asks for %v",
				d, out, theCeiling)
		}
	}
	if out != theCeiling {
		t.Errorf("deck by deck the day ran %v, and the goal asks for %v", out, theCeiling)
	}
}

// One deck is handed no more than the whole day is.
//
// A preset is the scope of its own budget, and entering one of its decks does
// not open a budget of that deck's own.
func TestOneDeckIsHandedNoMoreThanTheDayHolds(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"Sanskrit.md": "---\ntype: preset\ngoal: retention\nnew_a_day: 30\n" +
			"reviews_a_day: 30\nretention: 0.9\n---\n\n# Sanskrit\n",
		"Term.md": vault["Term.md"],
	}
	for d := range 3 {
		deck := "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\n"
		for i := range 40 {
			deck += fmt.Sprintf(
				"\n## Word %d.%d ^c%02d%06d\n\n[[Term]]\n\n### Word\n\nW\n\n### Meaning\n\nM\n",
				d, i, d, i)
		}
		files[fmt.Sprintf("decks/D%d.md", d)] = deck
	}

	s := opened(t, files)
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
	sits := func(over flashcards.Scope) flashcards.Sitting {
		sat, err := flashcards.Session{
			Marking: s.marking, Standings: s.standings, Schedules: s.kept,
			Presets: s.presets, Day: today, Now: func() time.Time { return now },
		}.Execute(t.Context(), s.vault, over)
		if err != nil {
			t.Fatal(err)
		}
		return sat
	}

	whole := 0
	for _, held := range byDeck(sits(flashcards.Scope{})) {
		whole += held
	}
	if whole != 30 {
		t.Errorf("the whole vault hands over %d card faces, and the day holds 30", whole)
	}
	for d := range 3 {
		deck := fmt.Sprintf("decks/D%d.md", d)
		if got := len(sits(flashcards.Scope{Deck: deck}).Asked); got > 30 {
			t.Errorf("%s alone hands over %d card faces, and the day holds 30", deck, got)
		}
	}
}
