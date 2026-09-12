package flashcards_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// term is the one stencil the budgeted vaults are cut by: one face, so a card
// is a card face.
const term = "---\ntype: stencil\nfields:\n  - Word\n  - Meaning\n---\n" +
	"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n"

// preset is a preset note carrying these frontmatter lines. Lines naming no
// goal are steered by their retention, which is the goal the card counts close
// the day under.
func preset(front string) string {
	if !strings.Contains(front, "goal:") {
		front = "goal: retention\n" + front
	}
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

// sessionAt is what the vault asks at this instant, held to the day's budgets.
func (s vaulted) sessionAt(t *testing.T, day review.Day, now time.Time) flashcards.SessionResult {
	t.Helper()
	sat, err := flashcards.Session{
		Marks: s.marking, CardFaces: s.standings, Schedules: s.kept,
		Presets: s.presets, Day: day, Now: func() time.Time { return now },
	}.Execute(t.Context(), s.vault, flashcards.Scope{})
	if err != nil {
		t.Fatal(err)
	}
	return sat
}

// byDeck is how many card faces of each deck a session holds.
func byDeck(sat flashcards.SessionResult) map[string]int {
	out := make(map[string]int)
	for _, one := range sat.Queue {
		out[one.Deck]++
	}
	return out
}

// unseen is how many of a session's cards nobody has answered yet.
func unseen(sat flashcards.SessionResult) int {
	out := 0
	for _, one := range sat.Queue {
		if !one.Schedule.IsSeen() {
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
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Slow.md":        preset("goal: minutes_a_day\nnew_a_day: 10\nminutes_a_day: 1\n"),
		"Quick.md":       preset("goal: minutes_a_day\nnew_a_day: 10\nminutes_a_day: 1\n"),
		"decks/Slow.md":  deckOf("Slow", 20, 0),
		"decks/Quick.md": deckOf("Quick", 20, 100),
	})

	// What each preset's cards have cost. Twelve card faces of each answered
	// three times over two months is a history long enough to say what both
	// kinds of answer take.
	for _, days := range []int{90, 80, 75} {
		before := s.run(t, saturday.AddDate(0, 0, -days))
		for i := range 12 {
			answer(t, before, mark(i), 20*time.Second)
			answer(t, before, mark(100+i), 4*time.Second)
		}
	}

	got := byDeck(s.sessionAt(t, today, saturday))
	// Twelve card faces of each stand owed. The minute buys three of the slow
	// ones, and all twelve of the quick ones with room for three unbegun.
	if got["decks/Slow.md"] != 3 {
		t.Errorf("the slow cards were asked %d in a minute, want 3", got["decks/Slow.md"])
	}
	if got["decks/Quick.md"] != 15 {
		t.Errorf("the quick cards were asked %d in a minute, want 15", got["decks/Quick.md"])
	}
}

// The day of the week a budget is read off is the review day being sat, which
// the small hours of the morning belong to the day before.
func TestALightDayIsReadOffTheReviewDay(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"Light.md": preset(
			"new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nload: {sat: 50}\n"),
		"decks/Light.md": deckOf("Light", 20, 0),
	})

	for _, one := range []struct {
		at    time.Time
		cards int
	}{
		// Two in the morning of the Saturday is the Friday's review day, which
		// carries the whole of the load.
		{time.Date(2026, 9, 5, 2, 0, 0, 0, time.Local), 8},
		// And two in the morning of the Sunday is the half Saturday itself.
		{time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local), 4},
	} {
		got := byDeck(s.sessionAt(t, today, one.at))["decks/Light.md"]
		if got != one.cards {
			t.Errorf("at %v the deck was asked %d cards, want %d", one.at, got, one.cards)
		}
	}
}

// A preset aiming at a day schedules through the whole of that review day, and
// stops when the next one opens.
func TestAPresetPastTheDayItAimsAtSchedulesNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"By.md": preset("goal: by_date\nby_date: 2026-09-05\n" +
			"new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/By.md": deckOf("By", 20, 0),
	})

	for _, one := range []struct {
		at    time.Time
		cards int
	}{
		// The last day of it holds all the material still to begin.
		{time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local), 20},
		// The small hours after it are still the day it aims at.
		{time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local), 20},
		{time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local), 0},
	} {
		got := byDeck(s.sessionAt(t, today, one.at))["decks/By.md"]
		if got != one.cards {
			t.Errorf("at %v the deck was asked %d cards, want %d", one.at, got, one.cards)
		}
	}
}

// A second session of the same day takes up where the first left off: what was
// answered since the day opened is off the day's budget.
func TestASecondSessionTakesUpWhereTheFirstLeftOff(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":       term,
		"Five.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Five.md": deckOf("Five", 20, 0),
	})
	if got := unseen(s.sessionAt(t, today, saturday)); got != 5 {
		t.Fatalf("the morning was asked %d new cards, want 5", got)
	}

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, mark(i), 6*time.Second)
	}

	if got := unseen(s.sessionAt(t, today, saturday.Add(2*time.Hour))); got != 2 {
		t.Errorf("the evening was asked %d new cards, want 2", got)
	}
}

// A card the day has already charged for comes round again for nothing under
// `counts: cards`, and spends the budget again under `counts: shows`.
func TestACardAnsweredAgainSpendsOneCardOrEveryShow(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		counts string
		cards  int
	}{{"cards", 2}, {"shows", 0}} {
		s := openVault(t, map[string]string{
			"Term.md": term,
			"Two.md": preset("counts: " + one.counts +
				"\nnew_a_day: 0\nreviews_a_day: 2\nminutes_a_day: 0\n"),
			"decks/Two.md": deckOf("Two", 2, 0),
		})

		// Both cards were answered long enough ago to be owed today.
		before := s.run(t, saturday.AddDate(0, 0, -30))
		answer(t, before, mark(0), 6*time.Second)
		answer(t, before, mark(1), 6*time.Second)

		// The first card is put to the person nine times in one session.
		morning := s.run(t, saturday)
		for range 9 {
			again(t, morning, mark(0), 4*time.Second)
		}

		got := byDeck(s.sessionAt(t, today, saturday.Add(2*time.Hour)))["decks/Two.md"]
		if got != one.cards {
			t.Errorf("counting in %s the evening was asked %d cards, want %d",
				one.counts, got, one.cards)
		}
	}
}

// A card face answered in an earlier session today is free when it comes round
// in a later one under `counts: cards`, and is charged again under
// `counts: shows`.
func TestAFaceTheDayHasChargedIsFreeInALaterSession(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		counts string
		cards  int
	}{{"cards", 1}, {"shows", 0}} {
		s := openVault(t, map[string]string{
			"Term.md": term,
			"One.md": preset("counts: " + one.counts +
				"\nnew_a_day: 0\nreviews_a_day: 1\nminutes_a_day: 0\n"),
			"decks/One.md": deckOf("One", 2, 0),
		})

		before := s.run(t, saturday.AddDate(0, 0, -30))
		answer(t, before, mark(0), 6*time.Second)
		answer(t, before, mark(1), 6*time.Second)

		again(t, s.run(t, saturday), mark(0), 4*time.Second)

		got := byDeck(s.sessionAt(t, today, saturday.Add(2*time.Hour)))["decks/One.md"]
		if got != one.cards {
			t.Errorf("counting in %s the evening was asked %d cards, want %d",
				one.counts, got, one.cards)
		}
	}
}

// A day's cards are counted in cards: three new cards that each take three
// steps to settle are the three the day holds, and no fourth is begun.
func TestADaysCardsAreCountedInCardsAndNotShows(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Three.md":       preset("new_a_day: 3\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Three.md": deckOf("Three", 6, 0),
	})
	if got := unseen(s.sessionAt(t, today, saturday)); got != 3 {
		t.Fatalf("the morning was asked %d new cards, want 3", got)
	}

	morning := s.run(t, saturday)
	for i := range 3 {
		for range 3 {
			again(t, morning, mark(i), 4*time.Second)
		}
	}

	sat := s.sessionAt(t, today, saturday.Add(2*time.Hour))
	if got := byDeck(sat)["decks/Three.md"]; got != 3 {
		t.Errorf("the evening was asked %d cards, want the three the day began", got)
	}
	if got := unseen(sat); got != 0 {
		t.Errorf("the evening began %d cards the day had no room for", got)
	}
}

// Each preset holds its own decks to its own budget, and one preset running out
// leaves the others where they were.
func TestTwoPresetsInOneSessionKeepTheirOwnBudgets(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":       term,
		"Few.md":        preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Few.md":  deckOf("Few", 10, 0),
		"decks/Many.md": deckOf("Many", 10, 100),
	})

	got := byDeck(s.sessionAt(t, today, saturday))
	want := map[string]int{"decks/Few.md": 2, "decks/Many.md": 5}
	for deck, cards := range want {
		if got[deck] != cards {
			t.Errorf("%s was asked %d cards, want %d", deck, got[deck], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the session held %v, want %v", got, want)
	}
}

// A preset of no cards a day schedules nothing, and every deck pointing at it
// is empty for the day. A deck scheduled by another preset is untouched.
func TestAPresetOfNoCardsADaySchedulesNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":          term,
		"Paused.md":        preset("new_a_day: 0\nreviews_a_day: 0\n"),
		"decks/Paused.md":  deckOf("Paused", 6, 0),
		"decks/Running.md": deckOf("", 6, 100),
	})

	got := byDeck(s.sessionAt(t, today, saturday))
	if got["decks/Paused.md"] != 0 {
		t.Errorf("a paused preset was asked %d cards", got["decks/Paused.md"])
	}
	if got["decks/Running.md"] != 6 {
		t.Errorf("the deck beside it was asked %d cards, want 6", got["decks/Running.md"])
	}
}

// A day the preset gives half the load carries half the cards, and a preset
// naming that day nothing carries all of them on it.
func TestALightDayCutsTheDaysCards(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"Light.md": preset(
			"new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nload: {sat: 50}\n"),
		"Plain.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Light.md": deckOf("Light", 10, 0),
		"decks/Plain.md": deckOf("Plain", 10, 100),
	})

	got := byDeck(s.sessionAt(t, today, saturday))
	if got["decks/Light.md"] != 4 {
		t.Errorf("a light Saturday was asked %d cards, want 4", got["decks/Light.md"])
	}
	if got["decks/Plain.md"] != 8 {
		t.Errorf("the same Saturday under the whole load was asked %d cards, want 8",
			got["decks/Plain.md"])
	}
}

// Under a goal of minutes the minutes close the day: a day of one minute holds
// three answers at the twenty seconds a new card costs.
func TestTheMinutesCloseTheDayUnderAGoalOfMinutes(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Short.md":       preset("goal: minutes_a_day\nnew_a_day: 10\nminutes_a_day: 1\n"),
		"decks/Short.md": deckOf("Short", 10, 0),
	})

	held := int(time.Minute / review.DefaultCost.New)
	got := byDeck(s.sessionAt(t, today, saturday))
	if got["decks/Short.md"] != held {
		t.Errorf("a day of one minute was asked %d cards, want %d", got["decks/Short.md"], held)
	}
}

// The goal names the budget that closes the day, and every other budget takes
// no part: moved to either end of what a preset may hold, a budget the goal
// does not name leaves the day asking the same cards.
func TestTheBudgetTheGoalDoesNotNameMovesNothing(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		what  string
		goal  string
		aside []string
		cards int
	}{
		// A minute of twenty-second cards is three of them.
		{"minutes", "goal: minutes_a_day\nminutes_a_day: 1\n", []string{
			"new_a_day: 0\nreviews_a_day: 0\n",
			"new_a_day: 9999\nreviews_a_day: 9999\n",
		}, 3},
		// Four new cards a day.
		{"retention", "goal: retention\nnew_a_day: 4\nreviews_a_day: 0\n", []string{
			"minutes_a_day: 0\n",
			"minutes_a_day: 1440\n",
		}, 4},
		// Forty cards over the ten days to the day it aims at, under a rule the
		// ten days can meet.
		{"a date", "goal: by_date\nby_date: 2026-09-14\nlearned: retention\n", []string{
			"new_a_day: 0\nreviews_a_day: 0\nminutes_a_day: 0\n",
			"new_a_day: 9999\nreviews_a_day: 9999\nminutes_a_day: 1440\n",
		}, 4},
	} {
		for _, aside := range one.aside {
			s := openVault(t, map[string]string{
				"Term.md":     term,
				"On.md":       preset(one.goal + aside),
				"decks/On.md": deckOf("On", 40, 0),
			})

			got := byDeck(s.sessionAt(t, today, saturday))["decks/On.md"]
			if got != one.cards {
				t.Errorf("steered by its %s under %q the day was asked %d cards, want %d",
					one.what, aside, got, one.cards)
			}
		}
	}
}

// A preset steered by its minutes goes on asking while the minutes last, and
// the reviews it keeps none of do not close it. Beside it the decks naming no
// preset are asked their own day.
func TestAPresetSteeredByItsMinutesKeepsAskingOnNoReviews(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"Steady.md": preset("goal: minutes_a_day\nminutes_a_day: 34\n" +
			"new_a_day: 12\nreviews_a_day: 0\n"),
		"decks/Steady.md": deckOf("Steady", 20, 0),
		"decks/Loose.md":  deckOf("", 20, 100),
	})

	// Six cards answered long enough ago to be owed today, and nothing answered
	// today.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	for i := range 6 {
		answer(t, before, mark(i), 6*time.Second)
	}

	sat := s.sessionAt(t, today, saturday)
	got := byDeck(sat)
	if got["decks/Steady.md"] != 20 {
		t.Errorf("the preset of no reviews was asked %d cards, want its 20",
			got["decks/Steady.md"])
	}
	if owed := countQueue(sat) - unseen(sat); owed != 6 {
		t.Errorf("the day was asked %d cards it owed, want 6", owed)
	}
	if got["decks/Loose.md"] != 20 {
		t.Errorf("the deck naming no preset was asked %d cards, want its 20",
			got["decks/Loose.md"])
	}
}

// A goal of a date paces the day: the material still to begin, over the days
// left to begin it in. Neither the minutes nor the counts cut it short.
func TestAGoalOfADatePacesTheDayOverTheDaysLeft(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		// Ten days from the Saturday to the day it aims at, counting both, and
		// a card is learned the day it is answered, so every one of them is a
		// day a card can be begun on.
		"By.md": preset("goal: by_date\nby_date: 2026-09-14\nlearned: retention\n" +
			"new_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 1\n"),
		"decks/By.md": deckOf("By", 40, 0),
	})

	if got := unseen(s.sessionAt(t, today, saturday)); got != 4 {
		t.Errorf("forty cards over ten days was asked %d a day, want 4", got)
	}
}

// What is owed is what the session asks: the same numbers, deck by deck.
func TestWhatIsOwedIsWhatTheSessionAsks(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Few.md":         preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Few.md":   deckOf("Few", 10, 0),
		"decks/Loose.md": deckOf("", 10, 100),
	})
	now := func() time.Time { return saturday }

	sat := byDeck(s.sessionAt(t, today, saturday))
	owing, err := s.newCountCardsDueAt(today, now).Execute(t.Context(), s.vault)
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

// countQueue is how many card faces a session holds, over every deck.
func countQueue(sat flashcards.SessionResult) int { return len(sat.Queue) }

// The decks naming no preset are one scope, so a second deck of them is a
// second deck of the same day and not a second day's work.
func TestDecksNamingNoPresetShareTheDefaultsBudget(t *testing.T) {
	t.Parallel()
	one := openVault(t, map[string]string{
		"Term.md":      term,
		"decks/One.md": deckOf("", 80, 0),
	})
	two := openVault(t, map[string]string{
		"Term.md":      term,
		"decks/One.md": deckOf("", 80, 0),
		"decks/Two.md": deckOf("", 80, 100),
	})

	held := int(time.Duration(review.Defaults().MinutesADay) *
		time.Minute / review.DefaultCost.New)
	if got := countQueue(one.sessionAt(t, today, saturday)); got != held {
		t.Fatalf("one deck on the defaults was asked %d cards, want %d", got, held)
	}
	if got := countQueue(two.sessionAt(t, today, saturday)); got != held {
		t.Errorf("two decks on the defaults were asked %d cards, want %d", got, held)
	}
}

// The decks naming one preset are one scope, and the decks naming none are
// another beside it.
func TestDecksNamingOnePresetShareItsBudget(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":       term,
		"Six.md":        preset("new_a_day: 6\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/One.md":  deckOf("Six", 20, 0),
		"decks/Two.md":  deckOf("Six", 20, 100),
		"decks/Free.md": deckOf("", 20, 200),
	})

	got := byDeck(s.sessionAt(t, today, saturday))
	if got["decks/One.md"]+got["decks/Two.md"] != 6 {
		t.Errorf("the two decks of one preset were asked %d and %d, want six between them",
			got["decks/One.md"], got["decks/Two.md"])
	}
	if got["decks/Free.md"] != 20 {
		t.Errorf("the deck naming no preset was asked %d cards, want its 20",
			got["decks/Free.md"])
	}
}

// A session over the whole vault is the union of the presets it holds, so a
// minute under each of two is two minutes of cards.
func TestTwoPresetsOfAMinuteEachHoldTwoMinutesOfCards(t *testing.T) {
	t.Parallel()
	apart := openVault(t, map[string]string{
		"Term.md":      term,
		"First.md":     preset("goal: minutes_a_day\nnew_a_day: 50\nminutes_a_day: 1\n"),
		"Second.md":    preset("goal: minutes_a_day\nnew_a_day: 50\nminutes_a_day: 1\n"),
		"decks/One.md": deckOf("First", 20, 0),
		"decks/Two.md": deckOf("Second", 20, 100),
	})
	together := openVault(t, map[string]string{
		"Term.md":      term,
		"First.md":     preset("goal: minutes_a_day\nnew_a_day: 50\nminutes_a_day: 1\n"),
		"decks/One.md": deckOf("First", 20, 0),
		"decks/Two.md": deckOf("First", 20, 100),
	})

	held := int(time.Minute / review.DefaultCost.New)
	if got := countQueue(together.sessionAt(t, today, saturday)); got != held {
		t.Fatalf("one minute over both decks was asked %d cards, want %d", got, held)
	}
	if got := countQueue(apart.sessionAt(t, today, saturday)); got != 2*held {
		t.Errorf("a minute under each of two presets was asked %d cards, want %d", got, 2*held)
	}
}

// A day's new cards and a day's reviews are two budgets, so a preset out of the
// one goes on asking the other.
func TestTheNewCardsOfADayAreNotHeldToItsReviews(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":      term,
		"New.md":       preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/New.md": deckOf("New", 6, 0),
	})

	// Two cards answered long enough ago to be owed today, which the preset
	// keeps no reviews for.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	answer(t, before, mark(0), 6*time.Second)
	answer(t, before, mark(1), 6*time.Second)

	sat := s.sessionAt(t, today, saturday)
	if got := unseen(sat); got != 2 {
		t.Errorf("the day was asked %d new cards, want the two it keeps", got)
	}
	if got := countQueue(sat) - unseen(sat); got != 0 {
		t.Errorf("a day keeping no reviews was asked %d of them", got)
	}
}

// A deck pointed at another preset between sessions is held to the budget of
// the preset it now names.
func TestADeckRepointedBetweenSessionsIsHeldToItsNewPreset(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"Few.md":      preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":     preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("Few", 20, 0),
	})
	if got := unseen(s.sessionAt(t, today, saturday)); got != 2 {
		t.Fatalf("under the preset of two the day was asked %d new cards", got)
	}

	write(t, s, "decks/On.md", deckOf("Many", 20, 0))

	if got := unseen(s.sessionAt(t, today, saturday.AddDate(0, 0, 1))); got != 5 {
		t.Errorf("under the preset of five the day was asked %d new cards, want 5", got)
	}
}

// What a day has already spent is spent against the preset the deck now names,
// so a deck repointed halfway through a day carries the morning with it.
func TestADeckRepointedInTheMiddleOfADayCarriesTheDaySpent(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"Few.md":      preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":     preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("Few", 20, 0),
	})

	morning := s.run(t, saturday)
	for i := range 2 {
		answer(t, morning, mark(i), 6*time.Second)
	}
	if got := unseen(s.sessionAt(t, today, saturday.Add(time.Hour))); got != 0 {
		t.Fatalf("the preset of two was spent and the day was asked %d new cards", got)
	}

	write(t, s, "decks/On.md", deckOf("Many", 20, 0))

	if got := unseen(s.sessionAt(t, today, saturday.Add(time.Hour))); got != 3 {
		t.Errorf("the day was asked %d new cards, want the five of the new preset less two", got)
	}
}

// A limit edited in the middle of a day holds from that moment, and what the
// day has already spent stands against it either way.
func TestALimitEditedInTheMiddleOfADayHoldsAtOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, mark(i), 6*time.Second)
	}
	evening := saturday.Add(time.Hour)
	if got := unseen(s.sessionAt(t, today, evening)); got != 2 {
		t.Fatalf("five a day with three spent was asked %d new cards, want 2", got)
	}

	write(t, s, "On.md", preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sessionAt(t, today, evening)); got != 5 {
		t.Errorf("raised to eight with three spent, the day was asked %d new cards, want 5", got)
	}

	write(t, s, "On.md", preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sessionAt(t, today, evening)); got != 0 {
		t.Errorf("lowered to two with three spent, the day was asked %d new cards", got)
	}
}

// `counts` edited in the middle of a day is what the day is counted by: the
// showings the day already held are charged for from that moment.
func TestCountsEditedInTheMiddleOfADayCountsTheDayAgain(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	if got := countQueue(s.sessionAt(t, today, evening)); got != 2 {
		t.Fatalf("counting in cards the evening was asked %d cards, want 2", got)
	}

	write(t, s, "On.md", preset("counts: shows\nnew_a_day: 0\nreviews_a_day: 3\n"+
		"minutes_a_day: 0\n"))
	if got := countQueue(s.sessionAt(t, today, evening)); got != 0 {
		t.Errorf("counting in shows the evening was asked %d cards, want none", got)
	}
}

// A preset paused in the middle of a day stops the decks pointing at it, and
// the cards the day had already begun stop with them.
func TestAPresetPausedInTheMiddleOfADayStopsTheCardsItBegan(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	if got := countQueue(s.sessionAt(t, today, evening)); got == 0 {
		t.Fatal("the running preset was asked nothing")
	}

	write(t, s, "On.md", preset("new_a_day: 0\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := countQueue(s.sessionAt(t, today, evening)); got != 0 {
		t.Errorf("a paused preset was asked %d cards", got)
	}
}

// A day moved behind us in the middle of a day stops the preset from that
// moment.
func TestADayMovedBehindUsStopsThePresetAtOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"On.md": preset("goal: by_date\nby_date: 2026-09-30\nlearned: retention\n" +
			"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})
	// Twenty cards over the twenty-six days to the day it aims at.
	if got := unseen(s.sessionAt(t, today, saturday)); got != 1 {
		t.Fatalf("a preset aiming ahead was asked %d new cards, want 1", got)
	}

	write(t, s, "On.md", preset("goal: by_date\nby_date: 2026-09-01\nlearned: retention\n"+
		"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sessionAt(t, today, saturday)); got != 0 {
		t.Errorf("a preset past the day it aims at was asked %d new cards", got)
	}
}

// A goal moved off a day starts the preset again: the day stands in the file
// and is read only while the goal names it.
func TestAGoalMovedOffADayStartsThePresetAgain(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"On.md": preset("goal: by_date\nby_date: 2026-09-01\n" +
			"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})
	if got := unseen(s.sessionAt(t, today, saturday)); got != 0 {
		t.Fatalf("a preset past the day it aims at was asked %d new cards", got)
	}

	write(t, s, "On.md", preset("goal: retention\nby_date: 2026-09-01\n"+
		"new_a_day: 4\nreviews_a_day: 0\nminutes_a_day: 0\n"))
	if got := unseen(s.sessionAt(t, today, saturday)); got != 4 {
		t.Errorf("a preset steered by its retention was asked %d new cards, want 4", got)
	}
}

// The budget is whole again when the day of review turns over, and not before:
// the small hours are the evening's day still.
func TestTheBudgetIsWholeAgainWhenTheDayTurnsOver(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
		if got := unseen(s.sessionAt(t, today, one.at)); got != one.fresh {
			t.Errorf("at %v the day was asked %d new cards, want %d", one.at, got, one.fresh)
		}
	}
}

// A day carrying none of the load is asked no card, and the day after it opens
// on the whole of the budget.
func TestADayCarryingNoneOfTheLoadIsAskedNothing(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nload: {sun: 0}\n"),
		"decks/On.md": deckOf("On", 40, 0),
	})

	for _, one := range []struct {
		at    time.Time
		cards int
	}{
		{time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local), 8},
		{time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local), 0},
		{time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local), 8},
	} {
		got := unseen(s.sessionAt(t, today, one.at))
		if got != one.cards {
			t.Errorf("%v was asked %d new cards, want %d", one.at.Weekday(), got, one.cards)
		}
	}
}

// A card face is new on the day it was first answered at all, which is read off
// the times the answers carry and not the order the runs arrived in.
func TestADayIsCountedByTheTimesOfItsAnswersAndNotItsRuns(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	if got := unseen(s.sessionAt(t, today, saturday)); got != 1 {
		t.Errorf("the day was asked %d new cards, want the one it keeps", got)
	}
}

// A retention edited in the middle of a day is what the cards are worked out at
// from that moment. The cache stands from before the edit and is thrown away,
// so what a session asks is the cards at the target now in force.
func TestARetentionEditedInTheMiddleOfADayChangesWhatIsOwed(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
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
	if got := countQueue(s.sessionAt(t, today, saturday)); got != 0 {
		t.Fatalf("asking for 0.7 of the cards back, %d were owed today", got)
	}

	write(t, s, "On.md", preset("retention: 0.99\nnew_a_day: 0\nreviews_a_day: 50\n"+
		"minutes_a_day: 0\n"))

	now, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	on := review.CardFaceID{Card: mark(0), Face: "Say it"}
	if now[on].Due.Equal(was[on].Due) {
		t.Errorf("the target moved and the card still comes round at %v", now[on].Due)
	}
	if got := countQueue(s.sessionAt(t, today, saturday)); got != 3 {
		t.Errorf("asking for 0.99 of the cards back, %d were owed today, want 3", got)
	}
}

// The session and the curve are one arithmetic.
//
// What a day admits is worked out in one place, so the count the curve draws at
// the value the preset holds is the count the sessions of that day hand a person.
// That day is the next one the preset admits: a day at none of the load is no
// session at all, so the curve draws the day after it.
//
// The day is driven the way a person drives it, a session at a time until it has
// nothing left to ask. A card the day comes back to is in a later session than
// the one that first showed it, so a single session is a batch of that day and
// not the day.
func TestTheSessionAndTheCurveAgreeOnTheDay(t *testing.T) {
	t.Parallel()
	// A day of the week the vault's own preset is read on, so a light Saturday
	// and a dead Saturday are read where a person meets them.
	const day = 90

	for _, one := range []struct {
		what  string
		front string
		cards int
		gave  func(t *testing.T, s vaulted)
	}{{
		what:  "caught up",
		front: "goal: minutes_a_day\nminutes_a_day: 3\nnew_a_day: 0\n",
		cards: 20,
		gave:  answersAt(20, -1, review.Good),
	}, {
		what:  "a month away",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 5\n",
		cards: 40,
		gave:  answersAt(25, -30, review.Good),
	}, {
		what:  "all new",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n",
		cards: 40,
	}, {
		what:  "answered once",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n",
		cards: 40,
		gave:  answersAt(40, -20, review.Good),
	}, {
		what:  "one card",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n",
		cards: 1,
	}, {
		what:  "an empty deck",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n",
		cards: 0,
	}, {
		what:  "all Again",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Again),
	}, {
		what:  "all Easy",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Easy),
	}, {
		what:  "a huge backlog",
		front: "goal: minutes_a_day\nminutes_a_day: 5\nnew_a_day: 6\n",
		cards: 120,
		gave:  answersAt(120, -60, review.Good),
	}, {
		what:  "a budget too small",
		front: "goal: minutes_a_day\nminutes_a_day: 1\nnew_a_day: 6\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Good),
	}, {
		what:  "a budget larger than the material",
		front: "goal: minutes_a_day\nminutes_a_day: 1440\nnew_a_day: 9999\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Good),
	}, {
		what:  "a light day",
		front: "goal: minutes_a_day\nminutes_a_day: 4\nnew_a_day: 6\nload: {sat: 50}\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Good),
	}, {
		what:  "a dead day",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\nload: {sat: 0}\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Good),
	}, {
		what: "every day dead",
		front: "goal: minutes_a_day\nminutes_a_day: 2\nnew_a_day: 6\n" +
			"load: {mon: 0, tue: 0, wed: 0, thu: 0, fri: 0, sat: 0, sun: 0}\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Good),
	}, {
		what:  "paused",
		front: "goal: minutes_a_day\nminutes_a_day: 0\nnew_a_day: 6\n",
		cards: 40,
		gave:  answersAt(20, -30, review.Good),
	}, {
		what:  "a day already partly spent",
		front: "goal: minutes_a_day\nminutes_a_day: 3\nnew_a_day: 6\n",
		cards: 40,
		gave:  newSpentToday(20),
	}, {
		// The counts close a day steered by a date, and the minutes never do.
		what: "counting cards, on a day already partly spent",
		front: fmt.Sprintf("goal: by_date\nby_date: %s\nlearned: interval\ninterval: 5\n"+
			"counts: cards\nnew_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 1\n",
			saturday.AddDate(0, 0, day).Format(review.Named)),
		cards: 40,
		gave:  newSpentToday(20),
	}, {
		what: "counting showings, on a day already partly spent",
		front: fmt.Sprintf("goal: by_date\nby_date: %s\nlearned: interval\ninterval: 5\n"+
			"counts: shows\nnew_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 1\n",
			saturday.AddDate(0, 0, day).Format(review.Named)),
		cards: 40,
		gave:  newSpentToday(20),
	}, {
		what: "a date",
		front: fmt.Sprintf("goal: by_date\nby_date: %s\nlearned: interval\ninterval: 5\n"+
			"new_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 1\n",
			saturday.AddDate(0, 0, day).Format(review.Named)),
		cards: 40,
		gave:  answersAt(15, -30, review.Good),
	}} {
		t.Run(one.what, func(t *testing.T) {
			s := openVault(t, map[string]string{
				"Term.md":     term,
				"On.md":       preset(one.front),
				"decks/On.md": deckOf("On", one.cards, 0),
			})
			if one.gave != nil {
				one.gave(t, s)
			}

			read, err := s.presets.Of(t.Context(), s.vault, "decks/On.md")
			if err != nil {
				t.Fatal(err)
			}
			curve, err := s.curves(saturday).Execute(
				t.Context(), s.vault, read.Path, read.Settings)
			if err != nil {
				t.Fatal(err)
			}
			if curve.Now.Index < 0 {
				t.Fatalf("the value the preset holds stands nowhere on the grid %v", curve.Grid)
			}

			drawn := curve.Points[curve.Now.Index].Reviews
			faces := s.through(t, today, admession(read.Settings, today, saturday))
			if drawn != float64(faces) {
				t.Errorf("the curve draws %v and the day hands over %d", drawn, faces)
			}
		})
	}
}

// admession is the next day of review this preset admits, counting from the day
// holding now, which is the day the curve draws. A preset admession no day at
// all is answered with the day it was asked about.
func admession(p review.Preset, day review.Day, now time.Time) time.Time {
	for range 8 {
		if !p.Admits(day, now, review.Spent{}, 0, 0).IsPaused() {
			return now
		}
		now = day.GetEnd(now)
	}
	return now
}

// answersAt is as many card faces answered this way, this many days before the
// day the vault is sat.
func answersAt(cards, days int, r review.Rating) func(*testing.T, vaulted) {
	return func(t *testing.T, s vaulted) {
		t.Helper()
		before := s.run(t, saturday.AddDate(0, 0, days))
		for i := range cards {
			answerWith(t, before, mark(i), r, 6*time.Second)
		}
	}
}

// newSpentToday is a day a person is already partway through: card faces owed
// from a month back, some of them answered twice since the day opened.
func newSpentToday(cards int) func(*testing.T, vaulted) {
	return func(t *testing.T, s vaulted) {
		t.Helper()
		answersAt(cards, -30, review.Good)(t, s)
		this := s.run(t, saturday.Add(-time.Hour))
		for i := range 4 {
			again(t, this, mark(i), 5*time.Second)
			answer(t, this, mark(i), 5*time.Second)
		}
	}
}

// newBacklogVault is a vault whose preset has a debt before it and material it
// has not begun, so a day has both to choose between.
func newBacklogVault(t *testing.T, front string) vaulted {
	t.Helper()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset(front),
		"decks/On.md": deckOf("On", 40, 0),
	})
	// Twenty card faces answered long enough ago to be owed today, and twenty
	// nobody has begun.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	for i := range 20 {
		answer(t, before, mark(i), 6*time.Second)
	}
	return s
}

// `backlog` says how much of a day goes to the debt before anything unbegun is
// offered. It is a share of the day and not a budget of its own, so it is read
// against the one pot a goal of minutes keeps.
func TestTheBacklogShareSaysWhatTheDayIsSpentOn(t *testing.T) {
	t.Parallel()
	// Four minutes a day over twenty card faces owed and twenty unbegun, which
	// is more of each than the day can carry.
	const day = "goal: minutes_a_day\nminutes_a_day: 4\n"

	all := s0(t, newBacklogVault(t, day+"backlog: 100\n"))
	if all.seen != 20 || all.fresh == 0 {
		t.Errorf("giving the debt all of the day asked %+v, want every card owed and "+
			"the rest of the day on new ones", all)
	}
	none := s0(t, newBacklogVault(t, day+"backlog: 0\n"))
	if none.fresh != 20 || none.seen == 0 {
		t.Errorf("giving the debt none of the day asked %+v, want every new card and "+
			"the rest of the day on owed ones", none)
	}
	// An odd day gives the extra card to the debt, so the two are never more
	// than one apart.
	half := s0(t, newBacklogVault(t, day+"backlog: 50\n"))
	if half.seen-half.fresh < 0 || half.seen-half.fresh > 1 {
		t.Errorf("splession the day evenly asked %+v", half)
	}
	if half.seen == 0 || half.fresh == 0 {
		t.Errorf("splession the day evenly left one side of it unspent: %+v", half)
	}
}

// A preset naming no share pays the debt first, which is what every preset
// written before the setting existed does.
func TestAPresetNamingNoBacklogShareIsUnchanged(t *testing.T) {
	t.Parallel()
	const day = "goal: minutes_a_day\nminutes_a_day: 4\n"

	was := s0(t, newBacklogVault(t, day+"backlog: 100\n"))
	if got := s0(t, newBacklogVault(t, day)); got != was {
		t.Errorf("a preset naming no share asked %+v, and one naming a hundred %+v", got, was)
	}
	if was.seen != 20 {
		t.Errorf("the debt was not paid first: %+v", was)
	}
}

// A side that runs short leaves the rest of the day to the other, so a day is
// never left part spent because one half of it had nothing to offer.
func TestASideThatRunsShortLeavesTheDayToTheOther(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"On.md": preset("goal: retention\nbacklog: 50\n" +
			"new_a_day: 20\nreviews_a_day: 20\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 40, 0),
	})
	// Two card faces owed, and thirty-eight nobody has begun.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	for i := range 2 {
		answer(t, before, mark(i), 6*time.Second)
	}

	sat := s.sessionAt(t, today, saturday)
	fresh := unseen(sat)
	if seen := countQueue(sat) - fresh; seen != 2 || fresh != 20 {
		t.Errorf("the day asked %d owed and %d new, want the two owed and its twenty new",
			seen, fresh)
	}
}

// A goal of a date carries the whole material by its own reckoning, so the
// share takes no part and the value stands in the file untouched.
func TestAGoalOfADateReadsNoBacklogShare(t *testing.T) {
	t.Parallel()
	for _, share := range []string{"backlog: 0\n", "backlog: 100\n"} {
		s := newBacklogVault(t, "goal: by_date\nby_date: 2026-09-14\nlearned: retention\n"+share+
			"new_a_day: 1\nreviews_a_day: 1\nminutes_a_day: 0\n")

		sat := s.sessionAt(t, today, saturday)
		fresh := unseen(sat)
		// Twenty owed, and twenty unbegun over the ten days to the day it aims
		// at, which is two a day.
		if seen := countQueue(sat) - fresh; seen != 20 || fresh != 2 {
			t.Errorf("under %q the day asked %d owed and %d new, want 20 and 2",
				share, seen, fresh)
		}
	}
}

// what a session came to, of each kind.
type session struct{ seen, fresh int }

func s0(t *testing.T, s vaulted) session {
	t.Helper()
	sat := s.sessionAt(t, today, saturday)
	fresh := unseen(sat)
	return session{seen: countQueue(sat) - fresh, fresh: fresh}
}

// The share of the day that goes to the debt moves the day where the day is one
// pot, and nowhere else.
//
// A goal of minutes spends one pot between the debt and the material it has not
// begun, so the share decides how much of each is asked. A goal of retention
// holds each side to a count of its own, so the order the two are drawn in
// cannot move either total.
func TestTheBacklogShareMovesOnlyADaySpentFromOnePot(t *testing.T) {
	t.Parallel()
	const minutes = "goal: minutes_a_day\nminutes_a_day: 4\n"
	// Counts larger than either side holds, so nothing but the share could
	// close the day.
	const target = "goal: retention\nnew_a_day: 20\nreviews_a_day: 20\nminutes_a_day: 0\n"

	spent := make(map[string]map[int]int, 2)
	for name, day := range map[string]string{"minutes": minutes, "retention": target} {
		spent[name] = make(map[int]int)
		for _, share := range []int{100, 50, 0} {
			one := s0(t, newBacklogVault(t, fmt.Sprintf("%sbacklog: %d\n", day, share)))
			spent[name][share] = one.seen
		}
	}
	if spent["minutes"][100] == spent["minutes"][0] {
		t.Errorf("a day of one pot asked %d of the debt at every share", spent["minutes"][100])
	}
	if spent["retention"][100] != spent["retention"][0] ||
		spent["retention"][100] != spent["retention"][50] {
		t.Errorf("a day of a count for each side asked %v of the debt, and the share "+
			"decides nothing there", spent["retention"])
	}
}

// What a day's budget is spent on decides what a count charges, and leaves the
// minutes alone.
//
// A card face the day has already answered comes round again for no count where
// the preset counts cards, and the minutes go on it as they go on every answer.
// A day of a minute spent four times over is a day nobody asked for.
func TestTheMinutesCloseTheDayWhicheverWayThePresetCounts(t *testing.T) {
	t.Parallel()
	for _, counts := range []string{"cards", "shows"} {
		s := openVault(t, map[string]string{
			"Term.md": term,
			"On.md": preset("goal: minutes_a_day\nminutes_a_day: 1\n" +
				"new_a_day: 0\nreviews_a_day: 0\ncounts: " + counts + "\n"),
			"decks/On.md": deckOf("On", 40, 0),
		})
		// Every card face answered long enough ago to be owed today, at six
		// seconds an answer, so a minute holds ten of them.
		before := s.run(t, saturday.AddDate(0, 0, -30))
		for i := range 40 {
			answer(t, before, mark(i), 6*time.Second)
		}

		spent := time.Duration(0)
		asked := 0
		for range 4 {
			sat := s.sessionAt(t, today, saturday.Add(spent))
			if len(sat.Queue) == 0 {
				break
			}
			given := s.run(t, saturday.Add(spent))
			for _, one := range sat.Queue {
				again(t, given, one.ID.Card, 6*time.Second)
				spent += 6 * time.Second
				asked++
			}
		}
		if spent > 90*time.Second {
			t.Errorf("counting %s, a day of one minute asked %d card faces and ran %v",
				counts, asked, spent)
		}
	}
}

// through is how many card faces one whole day of review hands over, driven the
// way a person drives it: a session at a time until the day has nothing left to
// ask, answering everything each of them holds.
//
// A card the day comes back to is the one card, so a face is counted once
// however many sessions show it. Each answer takes what a projection costs its
// kind at, so driving the day does not move the day's own arithmetic under it.
func (s vaulted) through(t *testing.T, day review.Day, now time.Time) int {
	t.Helper()
	faces := make(map[review.CardFaceID]bool)
	for range 100 {
		sat := s.sessionAt(t, day, now)
		if len(sat.Queue) == 0 {
			return len(faces)
		}
		record := s.run(t, now)
		for _, one := range sat.Queue {
			took := review.DefaultCost.Review
			if !one.Schedule.IsSeen() {
				took = review.DefaultCost.New
			}
			faces[one.ID] = true
			answer(t, record, one.ID.Card, took)
		}
	}
	t.Fatal("the day went on asking and never ran out")
	return 0
}
