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
	var naming []string
	if at != "" {
		naming = append(naming, at)
	}
	return deckNaming(naming, cards, from)
}

// deckNaming is a deck of as many cards, naming these presets in the order it
// names them.
func deckNaming(at []string, cards int, from int) string {
	out := "---\ntype: deck\n"
	if len(at) > 0 {
		out += "links:\n"
		for _, one := range at {
			out += "  - to: " + one + "\n    role: ref\n    type: preset\n"
		}
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

// asked is how many card faces a sitting holds, over every deck.
func asked(sat flashcards.Sitting) int { return len(sat.Asked) }

// The decks naming no preset are one scope, so a second deck of them is a
// second deck of the same day and not a second day's work.
func TestDecksNamingNoPresetShareTheDefaultsBudget(t *testing.T) {
	one := opened(t, map[string]string{
		"Term.md":      term,
		"decks/One.md": deckOf("", 20, 0),
	})
	two := opened(t, map[string]string{
		"Term.md":      term,
		"decks/One.md": deckOf("", 20, 0),
		"decks/Two.md": deckOf("", 20, 100),
	})

	held := history.Defaults().NewADay
	if got := asked(one.sittingAt(t, today, saturday)); got != held {
		t.Fatalf("one deck on the defaults was asked %d cards, want %d", got, held)
	}
	if got := asked(two.sittingAt(t, today, saturday)); got != held {
		t.Errorf("two decks on the defaults were asked %d cards, want %d", got, held)
	}
}

// The decks naming one preset are one scope, and the decks naming none are
// another beside it.
func TestDecksNamingOnePresetShareItsBudget(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":       term,
		"Six.md":        preset("new_a_day: 6\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/One.md":  deckOf("Six", 20, 0),
		"decks/Two.md":  deckOf("Six", 20, 100),
		"decks/Free.md": deckOf("", 20, 200),
	})

	got := byDeck(s.sittingAt(t, today, saturday))
	if got["decks/One.md"]+got["decks/Two.md"] != 6 {
		t.Errorf("the two decks of one preset were asked %d and %d, want six between them",
			got["decks/One.md"], got["decks/Two.md"])
	}
	if got["decks/Free.md"] != history.Defaults().NewADay {
		t.Errorf("the deck naming no preset was asked %d cards, want %d",
			got["decks/Free.md"], history.Defaults().NewADay)
	}
}

// A sitting over the whole vault is the union of the presets it holds, so a
// minute under each of two is two minutes of cards.
func TestTwoPresetsOfAMinuteEachHoldTwoMinutesOfCards(t *testing.T) {
	apart := opened(t, map[string]string{
		"Term.md":      term,
		"First.md":     preset("new_a_day: 50\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"Second.md":    preset("new_a_day: 50\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"decks/One.md": deckOf("First", 20, 0),
		"decks/Two.md": deckOf("Second", 20, 100),
	})
	together := opened(t, map[string]string{
		"Term.md":      term,
		"First.md":     preset("new_a_day: 50\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"decks/One.md": deckOf("First", 20, 0),
		"decks/Two.md": deckOf("First", 20, 100),
	})

	held := int(time.Minute / history.DefaultCost.New)
	if got := asked(together.sittingAt(t, today, saturday)); got != held {
		t.Fatalf("one minute over both decks was asked %d cards, want %d", got, held)
	}
	if got := asked(apart.sittingAt(t, today, saturday)); got != 2*held {
		t.Errorf("a minute under each of two presets was asked %d cards, want %d", got, 2*held)
	}
}

// A day's new cards and a day's reviews are two budgets, so a preset out of the
// one goes on asking the other.
func TestTheNewCardsOfADayAreNotHeldToItsReviews(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":      term,
		"New.md":       preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/New.md": deckOf("New", 6, 0),
	})

	// Two cards answered long enough ago to be owed today, which the preset
	// keeps no reviews for.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	answer(t, before, mark(0), 6*time.Second)
	answer(t, before, mark(1), 6*time.Second)

	sat := s.sittingAt(t, today, saturday)
	if got := unseen(sat); got != 2 {
		t.Errorf("the day was asked %d new cards, want the two it keeps", got)
	}
	if got := asked(sat) - unseen(sat); got != 0 {
		t.Errorf("a day keeping no reviews was asked %d of them", got)
	}
}

// A deck pointed at another preset between sittings is held to the budget of
// the preset it now names.
func TestADeckRepointedBetweenSittingsIsHeldToItsNewPreset(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"Few.md":      preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":     preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("Few", 20, 0),
	})
	if got := unseen(s.sittingAt(t, today, saturday)); got != 2 {
		t.Fatalf("under the preset of two the day was asked %d new cards", got)
	}

	write(t, s, "decks/On.md", deckOf("Many", 20, 0))

	if got := unseen(s.sittingAt(t, today, saturday.AddDate(0, 0, 1))); got != 5 {
		t.Errorf("under the preset of five the day was asked %d new cards, want 5", got)
	}
}

// What a day has already spent is spent against the preset the deck now names,
// so a deck repointed halfway through a day carries the morning with it.
func TestADeckRepointedInTheMiddleOfADayCarriesTheDaySpent(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"Few.md":      preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":     preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("Few", 20, 0),
	})

	morning := s.run(t, saturday)
	for i := range 2 {
		answer(t, morning, mark(i), 6*time.Second)
	}
	if got := unseen(s.sittingAt(t, today, saturday.Add(time.Hour))); got != 0 {
		t.Fatalf("the preset of two was spent and the day was asked %d new cards", got)
	}

	write(t, s, "decks/On.md", deckOf("Many", 20, 0))

	if got := unseen(s.sittingAt(t, today, saturday.Add(time.Hour))); got != 3 {
		t.Errorf("the day was asked %d new cards, want the five of the new preset less two", got)
	}
}

// A limit edited in the middle of a day holds from that moment, and what the
// day has already spent stands against it either way.
func TestALimitEditedInTheMiddleOfADayHoldsAtOnce(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, mark(i), 6*time.Second)
	}
	evening := saturday.Add(time.Hour)
	if got := unseen(s.sittingAt(t, today, evening)); got != 2 {
		t.Fatalf("five a day with three spent was asked %d new cards, want 2", got)
	}

	write(t, s, "On.md", preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sittingAt(t, today, evening)); got != 5 {
		t.Errorf("raised to eight with three spent, the day was asked %d new cards, want 5", got)
	}

	write(t, s, "On.md", preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sittingAt(t, today, evening)); got != 0 {
		t.Errorf("lowered to two with three spent, the day was asked %d new cards", got)
	}
}

// `counts` edited in the middle of a day is what the day is counted by: the
// showings the day already held are charged for from that moment.
func TestCountsEditedInTheMiddleOfADayCountsTheDayAgain(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md": term,
		"On.md": preset("counts: cards\nnew_a_day: 0\nreviews_a_day: 3\n" +
			"minutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 2, 0),
	})

	before := s.run(t, saturday.AddDate(0, 0, -30))
	answer(t, before, mark(0), 6*time.Second)
	answer(t, before, mark(1), 6*time.Second)

	// One card put to the person three times, which is one card and three
	// showings.
	morning := s.run(t, saturday)
	for range 3 {
		again(t, morning, mark(0), 4*time.Second)
	}

	evening := saturday.Add(2 * time.Hour)
	if got := asked(s.sittingAt(t, today, evening)); got != 2 {
		t.Fatalf("counting in cards the evening was asked %d cards, want 2", got)
	}

	write(t, s, "On.md", preset("counts: shows\nnew_a_day: 0\nreviews_a_day: 3\n"+
		"minutes_a_day: 0\n"))
	if got := asked(s.sittingAt(t, today, evening)); got != 0 {
		t.Errorf("counting in shows the evening was asked %d cards, want none", got)
	}
}

// A preset paused in the middle of a day stops the decks pointing at it, and
// the cards the day had already begun stop with them.
func TestAPresetPausedInTheMiddleOfADayStopsTheCardsItBegan(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 5\nreviews_a_day: 5\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})

	// Two cards the person could not recall, which come round again in the day.
	morning := s.run(t, saturday)
	for i := range 2 {
		again(t, morning, mark(i), 6*time.Second)
	}
	evening := saturday.Add(time.Hour)
	if got := asked(s.sittingAt(t, today, evening)); got == 0 {
		t.Fatal("the running preset was asked nothing")
	}

	write(t, s, "On.md", preset("new_a_day: 0\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := asked(s.sittingAt(t, today, evening)); got != 0 {
		t.Errorf("a paused preset was asked %d cards", got)
	}
}

// A day moved behind us in the middle of a day stops the preset from that
// moment.
func TestADayMovedBehindUsStopsThePresetAtOnce(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md": term,
		"On.md": preset("goal: by_date\nby_date: 2026-09-30\n" +
			"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})
	if got := unseen(s.sittingAt(t, today, saturday)); got != 4 {
		t.Fatalf("a preset aiming ahead was asked %d new cards, want 4", got)
	}

	write(t, s, "On.md", preset("goal: by_date\nby_date: 2026-09-01\n"+
		"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sittingAt(t, today, saturday)); got != 0 {
		t.Errorf("a preset past the day it aims at was asked %d new cards", got)
	}
}

// A goal moved off a day starts the preset again: the day stands in the file
// and is read only while the goal names it.
func TestAGoalMovedOffADayStartsThePresetAgain(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md": term,
		"On.md": preset("goal: by_date\nby_date: 2026-09-01\n" +
			"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})
	if got := unseen(s.sittingAt(t, today, saturday)); got != 0 {
		t.Fatalf("a preset past the day it aims at was asked %d new cards", got)
	}

	write(t, s, "On.md", preset("goal: minutes_a_day\nby_date: 2026-09-01\n"+
		"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sittingAt(t, today, saturday)); got != 4 {
		t.Errorf("a preset steered by its minutes was asked %d new cards, want 4", got)
	}
}

// The budget is whole again when the day of review turns over, and not before:
// the small hours are the evening's day still.
func TestTheBudgetIsWholeAgainWhenTheDayTurnsOver(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"Five.md":     preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("Five", 30, 0),
	})

	// The Friday evening, spent to the last of the day's new cards.
	evening := time.Date(2026, 9, 4, 22, 0, 0, 0, time.Local)
	run := s.run(t, evening)
	for i := range 5 {
		answer(t, run, mark(i), 6*time.Second)
	}

	for _, one := range []struct {
		at    time.Time
		fresh int
	}{
		{evening.Add(time.Hour), 0},
		// Three in the morning is the Friday's review day still.
		{time.Date(2026, 9, 5, 3, 0, 0, 0, time.Local), 0},
		{time.Date(2026, 9, 5, 5, 0, 0, 0, time.Local), 5},
	} {
		if got := unseen(s.sittingAt(t, today, one.at)); got != one.fresh {
			t.Errorf("at %v the day was asked %d new cards, want %d", one.at, got, one.fresh)
		}
	}
}

// A light day sheds onto the day either side of it round the end of the week,
// so a light Sunday is carried by the Saturday before it and the Monday after.
func TestALightSundayShedsOntoTheDaysEitherSideOfIt(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nlight_days: [sun]\n"),
		"decks/On.md": deckOf("On", 40, 0),
	})

	for _, one := range []struct {
		at    time.Time
		cards int
	}{
		{time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local), 10},
		{time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local), 4},
		{time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local), 10},
		{time.Date(2026, 9, 8, 10, 0, 0, 0, time.Local), 8},
	} {
		got := unseen(s.sittingAt(t, today, one.at))
		if got != one.cards {
			t.Errorf("%v was asked %d new cards, want %d", one.at.Weekday(), got, one.cards)
		}
	}
}

// A card face is new on the day it was first answered at all, which is read off
// the times the answers carry and not the order the runs arrived in.
func TestADayIsCountedByTheTimesOfItsAnswersAndNotItsRuns(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md":     term,
		"One.md":      preset("new_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("One", 4, 0),
	})
	yesterday := saturday.AddDate(0, 0, -1)

	// Today's answer to the card, in a run named for yesterday, and yesterday's
	// in a run named for today: the files sort the other way about from the
	// answers in them.
	answer(t, s.runNamed(t, yesterday, saturday.Add(-2*time.Hour)), mark(0), 6*time.Second)
	answer(t, s.runNamed(t, saturday, yesterday), mark(0), 6*time.Second)

	// The card was first answered yesterday, so today's answer to it is a
	// review and the day's one new card is still to be taken.
	if got := unseen(s.sittingAt(t, today, saturday)); got != 1 {
		t.Errorf("the day was asked %d new cards, want the one it keeps", got)
	}
}

// A retention edited in the middle of a day is what the cards are worked out at
// from that moment. The cache stands from before the edit and is thrown away,
// so what a sitting asks is the cards at the target now in force.
func TestARetentionEditedInTheMiddleOfADayChangesWhatIsOwed(t *testing.T) {
	s := opened(t, map[string]string{
		"Term.md": term,
		"On.md": preset("retention: 0.7\nnew_a_day: 0\nreviews_a_day: 50\n" +
			"minutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 3, 0),
	})

	// Three cards taken far enough to be learned, and left a week ago.
	for day := range 3 {
		run := s.run(t, saturday.AddDate(0, 0, day-6))
		for i := range 3 {
			answer(t, run, mark(i), 6*time.Second)
		}
	}
	// The working out under the old target, remembered where the next one will
	// look for it.
	was, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := asked(s.sittingAt(t, today, saturday)); got != 0 {
		t.Fatalf("asking for 0.7 of the cards back, %d were owed today", got)
	}

	write(t, s, "On.md", preset("retention: 0.99\nnew_a_day: 0\nreviews_a_day: 50\n"+
		"minutes_a_day: 0\n"))

	now, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	on := history.CardFace{Card: mark(0), Face: "Say it"}
	if now[on].Due.Equal(was[on].Due) {
		t.Errorf("the target moved and the card still comes round at %v", now[on].Due)
	}
	if got := asked(s.sittingAt(t, today, saturday)); got != 3 {
		t.Errorf("asking for 0.99 of the cards back, %d were owed today, want 3", got)
	}
}
