package flashcards_test

import (
	"fmt"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// term is the one stencil the budgeted vaults are cut by: one face, so a card
// is a card face.
const term = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n"

// preset is a preset note carrying these frontmatter lines.
func preset(front string) string {
	return "---\ntype: preset\n" + front + "---\n\n# A preset\n"
}

// deckOf is a deck of as many cards, pointing at the preset named. A deck
// naming none is written with an empty name.
func deckOf(at string, cards int, from int) string {
	out := "---\ntype: deck\n"
	if at != "" {
		out += "links:\n  - to: " + at + "\n    role: ref\n    type: preset\n"
	}
	out += "---\n"
	for i := range cards {
		mark := fmt.Sprintf("card%06d", from+i)
		out += fmt.Sprintf(
			"\n## Card %d ^%s\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n",
			from+i, mark, from+i, from+i)
	}
	return out
}

// sittingAt is what the vault asks at this instant, held to the day's budgets.
func (s vaulted) sittingAt(t *testing.T, day history.Day, now time.Time) flashcards.Sitting {
	t.Helper()
	sat, err := flashcards.Session{
		Marking: s.marking, Standings: s.standings, Schedules: s.kept,
		Presets: s.presets, Day: day, Now: func() time.Time { return now },
	}.Execute(t.Context(), s.vault, "")
	if err != nil {
		t.Fatal(err)
	}
	return sat
}

// byDeck is how many card faces of each deck a sitting holds.
func byDeck(sat flashcards.Sitting) map[string]int {
	out := make(map[string]int)
	for _, one := range sat.Asked {
		out[one.Deck]++
	}
	return out
}

// unseen is how many of a sitting's cards nobody has answered yet.
func unseen(sat flashcards.Sitting) int {
	out := 0
	for _, one := range sat.Asked {
		if !one.Schedule.Seen() {
			out++
		}
	}
	return out
}

// mark is the card of this place in a deck, as deckOf writes it.
func mark(at int) string { return fmt.Sprintf("card%06d", at) }

// Each preset turns its own minutes into a count from its own answer times, so
// two presets of the same day hold as many cards as their cards cost.
func TestEachPresetIsCostedFromItsOwnAnswers(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Slow.md":        preset("new_a_day: 10\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"Quick.md":       preset("new_a_day: 10\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"decks/Slow.md":  deckOf("Slow", 20, 0),
		"decks/Quick.md": deckOf("Quick", 20, 100),
	})

	// What each preset's cards have cost, answered on a day of their own.
	before := s.run(t, saturday.AddDate(0, 0, -1))
	for i := range 5 {
		answer(t, before, mark(i), 20*time.Second)
		answer(t, before, mark(100+i), 4*time.Second)
	}

	got := byDeck(s.sittingAt(t, today, saturday))
	// A minute of twenty-second cards is three of them, and the same minute of
	// four-second cards is more than the count allows.
	if got["decks/Slow.md"] != 3 {
		t.Errorf("the slow cards were asked %d in a minute, want 3", got["decks/Slow.md"])
	}
	if got["decks/Quick.md"] != 10 {
		t.Errorf("the quick cards were asked %d in a minute, want 10", got["decks/Quick.md"])
	}
}

// The day of the week a budget is read off is the review day being sat, which
// the small hours of the morning belong to the day before.
func TestALightDayIsReadOffTheReviewDay(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Light.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nlight_days: [sat]\n"),
		"decks/Light.md": deckOf("Light", 20, 0),
	})

	for _, one := range []struct {
		at    time.Time
		cards int
	}{
		// Two in the morning of the Saturday is the Friday's review day, which
		// carries what the light Saturday sheds.
		{time.Date(2026, 9, 5, 2, 0, 0, 0, time.Local), 10},
		// And two in the morning of the Sunday is the light Saturday itself.
		{time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local), 4},
	} {
		got := byDeck(s.sittingAt(t, today, one.at))["decks/Light.md"]
		if got != one.cards {
			t.Errorf("at %v the deck was asked %d cards, want %d", one.at, got, one.cards)
		}
	}
}

// A preset aiming at a day schedules through the whole of that review day, and
// stops when the next one opens.
func TestAPresetPastTheDayItAimsAtSchedulesNothing(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md": term,
		"By.md": preset("goal: by_date\nby_date: 2026-09-05\n" +
			"new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/By.md": deckOf("By", 20, 0),
	})

	for _, one := range []struct {
		at    time.Time
		cards int
	}{
		{time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local), 5},
		// The small hours after it are still the day it aims at.
		{time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local), 5},
		{time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local), 0},
	} {
		got := byDeck(s.sittingAt(t, today, one.at))["decks/By.md"]
		if got != one.cards {
			t.Errorf("at %v the deck was asked %d cards, want %d", one.at, got, one.cards)
		}
	}
}

// A second sitting of the same day takes up where the first left off: what was
// answered since the day opened is off the day's budget.
func TestASecondSittingTakesUpWhereTheFirstLeftOff(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":       term,
		"Five.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Five.md": deckOf("Five", 20, 0),
	})
	if got := unseen(s.sittingAt(t, today, saturday)); got != 5 {
		t.Fatalf("the morning was asked %d new cards, want 5", got)
	}

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, mark(i), 6*time.Second)
	}

	if got := unseen(s.sittingAt(t, today, saturday.Add(2*time.Hour))); got != 2 {
		t.Errorf("the evening was asked %d new cards, want 2", got)
	}
}

// A card the day has already charged for comes round again for nothing under
// `counts: cards`, and spends the budget again under `counts: shows`.
func TestACardAnsweredAgainSpendsOneCardOrEveryShow(t *testing.T) {
	for _, one := range []struct {
		counts string
		cards  int
	}{{"cards", 2}, {"shows", 0}} {
		s := opened(t, map[string]string{
			"Term.md": term,
			"Two.md": preset("counts: " + one.counts +
				"\nnew_a_day: 0\nreviews_a_day: 2\nminutes_a_day: 0\n"),
			"decks/Two.md": deckOf("Two", 2, 0),
		})

		// Both cards were answered long enough ago to be owed today.
		before := s.run(t, saturday.AddDate(0, 0, -30))
		answer(t, before, mark(0), 6*time.Second)
		answer(t, before, mark(1), 6*time.Second)

		// The first card is put to the person nine times in one sitting.
		morning := s.run(t, saturday)
		for range 9 {
			again(t, morning, mark(0), 4*time.Second)
		}

		got := byDeck(s.sittingAt(t, today, saturday.Add(2*time.Hour)))["decks/Two.md"]
		if got != one.cards {
			t.Errorf("counting in %s the evening was asked %d cards, want %d",
				one.counts, got, one.cards)
		}
	}
}

// A card face answered in an earlier sitting today is free when it comes round
// in a later one under `counts: cards`, and is charged again under
// `counts: shows`.
func TestAFaceTheDayHasChargedIsFreeInALaterSitting(t *testing.T) {
	for _, one := range []struct {
		counts string
		cards  int
	}{{"cards", 1}, {"shows", 0}} {
		s := opened(t, map[string]string{
			"Term.md": term,
			"One.md": preset("counts: " + one.counts +
				"\nnew_a_day: 0\nreviews_a_day: 1\nminutes_a_day: 0\n"),
			"decks/One.md": deckOf("One", 2, 0),
		})

		before := s.run(t, saturday.AddDate(0, 0, -30))
		answer(t, before, mark(0), 6*time.Second)
		answer(t, before, mark(1), 6*time.Second)

		again(t, s.run(t, saturday), mark(0), 4*time.Second)

		got := byDeck(s.sittingAt(t, today, saturday.Add(2*time.Hour)))["decks/One.md"]
		if got != one.cards {
			t.Errorf("counting in %s the evening was asked %d cards, want %d",
				one.counts, got, one.cards)
		}
	}
}

// A day's cards are counted in cards: three new cards that each take three
// steps to settle are the three the day holds, and no fourth is begun.
func TestADaysCardsAreCountedInCardsAndNotShows(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Three.md":       preset("new_a_day: 3\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Three.md": deckOf("Three", 6, 0),
	})
	if got := unseen(s.sittingAt(t, today, saturday)); got != 3 {
		t.Fatalf("the morning was asked %d new cards, want 3", got)
	}

	morning := s.run(t, saturday)
	for i := range 3 {
		for range 3 {
			again(t, morning, mark(i), 4*time.Second)
		}
	}

	sat := s.sittingAt(t, today, saturday.Add(2*time.Hour))
	if got := byDeck(sat)["decks/Three.md"]; got != 3 {
		t.Errorf("the evening was asked %d cards, want the three the day began", got)
	}
	if got := unseen(sat); got != 0 {
		t.Errorf("the evening began %d cards the day had no room for", got)
	}
}

// Each preset holds its own decks to its own budget, and one preset running out
// leaves the others where they were.
func TestTwoPresetsInOneSittingKeepTheirOwnBudgets(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":       term,
		"Few.md":        preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Few.md":  deckOf("Few", 10, 0),
		"decks/Many.md": deckOf("Many", 10, 100),
	})

	got := byDeck(s.sittingAt(t, today, saturday))
	want := map[string]int{"decks/Few.md": 2, "decks/Many.md": 5}
	for deck, cards := range want {
		if got[deck] != cards {
			t.Errorf("%s was asked %d cards, want %d", deck, got[deck], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the sitting held %v, want %v", got, want)
	}
}

// A preset of no cards a day schedules nothing, and every deck pointing at it
// is empty for the day. A deck scheduled by another preset is untouched.
func TestAPresetOfNoCardsADaySchedulesNothing(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":          term,
		"Paused.md":        preset("new_a_day: 0\nreviews_a_day: 0\n"),
		"decks/Paused.md":  deckOf("Paused", 6, 0),
		"decks/Running.md": deckOf("", 6, 100),
	})

	got := byDeck(s.sittingAt(t, today, saturday))
	if got["decks/Paused.md"] != 0 {
		t.Errorf("a paused preset was asked %d cards", got["decks/Paused.md"])
	}
	if got["decks/Running.md"] != 6 {
		t.Errorf("the deck beside it was asked %d cards, want 6", got["decks/Running.md"])
	}
}

// A day the preset names light carries half the load, and a preset naming none
// carries all of it on the same day.
func TestALightDayCutsTheDaysCards(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Light.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nlight_days: [sat]\n"),
		"Plain.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Light.md": deckOf("Light", 10, 0),
		"decks/Plain.md": deckOf("Plain", 10, 100),
	})

	got := byDeck(s.sittingAt(t, today, saturday))
	if got["decks/Light.md"] != 4 {
		t.Errorf("a light Saturday was asked %d cards, want 4", got["decks/Light.md"])
	}
	if got["decks/Plain.md"] != 8 {
		t.Errorf("the same Saturday under no light day was asked %d cards, want 8",
			got["decks/Plain.md"])
	}
}

// Whichever budget runs out first closes the preset for the day: a day of one
// minute holds three answers at the twenty seconds a new card costs, and the
// ten cards the count allows are never reached.
func TestTheMinutesBudgetClosesTheDayBeforeTheCount(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Short.md":       preset("new_a_day: 10\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"decks/Short.md": deckOf("Short", 10, 0),
	})

	held := int(time.Minute / history.DefaultCost.New)
	got := byDeck(s.sittingAt(t, today, saturday))
	if got["decks/Short.md"] != held {
		t.Errorf("a day of one minute was asked %d cards, want %d", got["decks/Short.md"], held)
	}
}

// What is owed is what the sitting asks: the same numbers, deck by deck.
func TestWhatIsOwedIsWhatTheSittingAsks(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Few.md":         preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Few.md":   deckOf("Few", 10, 0),
		"decks/Loose.md": deckOf("", 10, 100),
	})
	now := func() time.Time { return saturday }

	sat := byDeck(s.sittingAt(t, today, saturday))
	owing, err := s.owedAt(today, now).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	// Two cards under the preset that keeps two, and the ten the defaults keep.
	if owing.New != 12 || owing.Due != 0 {
		t.Errorf("the vault owes %d new and %d due, want 12 and 0", owing.New, owing.Due)
	}
	for _, one := range owing.Decks {
		if one.New+one.Due != sat[one.Deck] {
			t.Errorf("%s owes %d and is asked %d", one.Deck, one.New+one.Due, sat[one.Deck])
		}
	}
}
