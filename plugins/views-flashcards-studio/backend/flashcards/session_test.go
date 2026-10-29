package flashcards_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// under each of the four is the moment that scheduler brings the card back.
func TestTheWindowsBelowTheFourAreTheCardsOwnSchedulers(t *testing.T) {
	t.Parallel()
	// The share of the cards each deck's preset asks to come back, by the deck.
	shares := map[string]float64{"decks/Roots.md": 0.99, "decks/Terms.md": 0.9}
	s := openVault(t, map[string]string{
		"Term.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
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
	pos := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	record := s.run(t, pos.AddDate(0, 0, -200))
	for _, card := range []string{"k7m2xq9fzp", "3f4g5h6j7k"} {
		on := review.ReviewCardID{Card: card, Template: "Say it"}
		if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}

	session := flashcards.NewSession(
		s.assigning, s.standings, s.schedules,
		s.presets, today, func() time.Time { return pos },
	)
	target, err := session.Execute(t.Context(), s.vault, flashcards.Scope{})
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Queue) != 2 {
		t.Fatalf("%d cards were asked", len(target.Queue))
	}

	observed := map[float64]time.Duration{}
	for _, one := range target.Queue {
		share, named := shares[one.Deck]
		if !named {
			t.Fatalf("a card of %s was asked", one.Deck)
		}
		want := review.NewFSRSAt(share).Next(one.Schedule, pos, review.Good).Due.Sub(pos)
		if one.Ahead[review.Good] != want {
			t.Errorf("%s comes back in %v at a share of %g, and the window says %v",
				one.Deck, want, share, one.Ahead[review.Good])
		}
		observed[share] = want
	}
	if observed[0.99] == observed[0.9] {
		t.Fatal("the two presets leave the card in the same place, and this test says nothing")
	}
}

// The window under a button is the day the card comes back on.
//
// A card's day is chosen in one function: the scheduler works out an interval
// and the preset chooses a day inside the tolerance around it. The button goes
// through the same one, so a preset evening its load names the day it moved the
// card to.
func TestTheWindowsBelowTheFourNameTheDayTheCardReachesBackOn(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		what   string
		isEven bool
	}{{"an even load", true}, {"no even load", false}} {
		s := openVault(t, map[string]string{
			"Term.md": term,
			"On.md": preset(fmt.Sprintf("goal: minutes_a_day\nminutes_a_day: 1440\n"+
				"new_a_day: 0\nreviews_a_day: 9999\neven_load: %t\n", one.isEven)),
			"decks/On.md": deckOf("On", 60, 0),
		})
		// Every review card put into review long ago, so each comes back in days
		// and the placement has a window to move it in.
		for _, days := range []int{40, 25} {
			run := s.run(t, saturday.AddDate(0, 0, -days))
			for i := range 60 {
				answer(t, run, cardID(i), 4*time.Second)
			}
		}

		// One card answered at a time, each a second after the last, so the day
		// stays the one day and nothing but the placement can move a card off
		// the day the scheduler named. What a button names is a day, so the day
		// is what is compared.
		request, differ, worst := 0, 0, time.Duration(0)
		for step := range 60 {
			pos := saturday.Add(time.Duration(step) * time.Second)
			sat := s.sessionAt(t, today, pos)
			if len(sat.Queue) == 0 {
				break
			}
			card := sat.Queue[0]
			response, named := card.Ahead[review.Good]
			if !named {
				t.Fatalf("under %s a card was asked with no window under its buttons", one.what)
			}
			if _, err := s.run(t, pos).Answer(
				t.Context(), card.ID, review.Good, 0,
			); err != nil {
				t.Fatal(err)
			}
			schedules, err := s.schedules.Execute(t.Context(), s.vault)
			if err != nil {
				t.Fatal(err)
			}
			request++
			came, names := schedules[card.ID].Due, pos.Add(response)
			if today.GetName(came) != today.GetName(names) {
				differ++
				if off := came.Sub(names); off > worst || -off > worst {
					worst = max(off, -off)
				}
			}
		}
		if request == 0 {
			t.Fatalf("under %s no card was asked, and this test says nothing", one.what)
		}
		if differ != 0 {
			t.Errorf("under %s %d of %d cards came back somewhere else than their button said, "+
				"the worst by %v", one.what, differ, request, worst)
		}
	}
}

// review cards is the review cards a session asks, in the order it asks them.
func reviewCards(sat flashcards.SessionResult) []review.ReviewCardID {
	out := make([]review.ReviewCardID, 0, len(sat.Queue))
	for _, one := range sat.Queue {
		out = append(out, one.ID)
	}
	return out
}

// getReviewCard is a review card of the one card model these vaults are cut by.
func getReviewCard(pos int) review.ReviewCardID {
	return review.ReviewCardID{Card: cardID(pos), Template: "Say it"}
}

// The debt is paid oldest first, whatever order the cards stand in their deck.
//
// A day that cannot pay all of it answers the review cards waiting longest and
// leaves the rest of the pile standing.
func TestTheDebtIsPaidOldestFirst(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":       term,
		"Two.md":        preset("new_a_day: 0\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"decks/Five.md": deckOf("Two", 5, 0),
	})
	// The cards stand in their deck in the reverse of the order they came round
	// in: the last of them was answered longest ago.
	for i := range 5 {
		answer(t, s.run(t, saturday.AddDate(0, 0, -10+i)), cardID(4-i), 6*time.Second)
	}

	got := reviewCards(s.openSessionByPreset(t, today, saturday, "Two.md"))
	want := []review.ReviewCardID{getReviewCard(4), getReviewCard(3)}
	if !slices.Equal(got, want) {
		t.Errorf("a day of two reviews asked %v, want %v", got, want)
	}
	for _, one := range []int{2, 1, 0} {
		if slices.Contains(got, getReviewCard(one)) {
			t.Errorf("%s was asked, and the day had room for the two older", cardID(one))
		}
	}
}

// A session over several presets is one session however many times it is asked
// for: the same request over the same vault is the same cards in the same
// order, review card by review card.
func TestASessionOverSeveralPresetsIsTheSameSessionTwice(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":    term,
		"One.md":     preset("new_a_day: 2\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"Two.md":     preset("new_a_day: 3\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"Three.md":   preset("new_a_day: 1\nreviews_a_day: 2\nminutes_a_day: 0\n"),
		"decks/A.md": deckOf("One", 6, 0),
		"decks/B.md": deckOf("Two", 6, 100),
		"decks/C.md": deckOf("Three", 6, 200),
		"decks/D.md": deckOf("One", 6, 300),
	})
	// A debt under each preset, so every one of them has both sides of its day
	// to spend and none of them is handed its cards in one lump.
	for _, days := range []int{-9, -8} {
		record := s.run(t, saturday.AddDate(0, 0, days))
		for _, from := range []int{0, 100, 200, 300} {
			for i := range 2 {
				answer(t, record, cardID(from+i), 6*time.Second)
			}
		}
	}

	first := reviewCards(s.sessionAt(t, today, saturday))
	if len(first) == 0 {
		t.Fatal("the session asked nothing")
	}
	for range 5 {
		if again := reviewCards(s.sessionAt(t, today, saturday)); !slices.Equal(first, again) {
			t.Fatalf("the session asked %v, and the same request asked %v", first, again)
		}
	}
}

// openSession is what the vault asks at this instant when the session is opened
// over this deck or this preset.
func (s vaulted) openSession(
	t *testing.T, day review.Day, now time.Time, pos flashcards.Scope,
) (flashcards.SessionResult, error) {
	t.Helper()
	return flashcards.NewSession(
		s.assigning, s.standings, s.schedules,
		s.presets, day, func() time.Time { return now },
	).Execute(t.Context(), s.vault, pos)
}

// openSessionByPreset is the session over one preset, and a fatal error where
func (s vaulted) openSessionByPreset(
	t *testing.T, day review.Day, now time.Time, preset string,
) flashcards.SessionResult {
	t.Helper()
	sat, err := s.openSession(t, day, now, flashcards.Scope{Preset: preset, IsNamed: true})
	if err != nil {
		t.Fatal(err)
	}
	return sat
}

// Pressing a preset sits to the cards of every deck pointing at it, and to no
// card of another preset.
func TestASessionOverAPresetAsksTheDecksThatPointAtIt(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":         term,
		"Steady.md":       preset("new_a_day: 20\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Other.md":        preset("new_a_day: 20\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md":  deckOf("Steady", 3, 0),
		"decks/Trees.md":  deckOf("Steady", 4, 100),
		"decks/Rivers.md": deckOf("Other", 5, 200),
	})

	got := byDeck(s.openSessionByPreset(t, today, saturday, "Steady.md"))
	want := map[string]int{"decks/Birds.md": 3, "decks/Trees.md": 4}
	for deck, cards := range want {
		if got[deck] != cards {
			t.Errorf("%s was asked %d cards, want %d", deck, got[deck], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the session held %v, want %v", got, want)
	}
}

// The decks naming no preset are a preset of their own, and the session over it
// is opened by naming no note.
func TestASessionOverThePresetOfTheDecksNamingNone(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Steady.md":      preset("new_a_day: 20\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Loose.md": deckOf("", 3, 0),
		"decks/Birds.md": deckOf("Steady", 4, 100),
	})

	got := byDeck(s.openSessionByPreset(t, today, saturday, ""))
	if len(got) != 1 || got["decks/Loose.md"] != 3 {
		t.Errorf("the session held %v, want three cards of decks/Loose.md", got)
	}
}

// The cards a preset offers are what its own allowance admits for the day, and
// the decks under it share that one budget.
func TestSession_PresetBudgetLimitsSessionCards(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Five.md":        preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md": deckOf("Five", 10, 0),
		"decks/Trees.md": deckOf("Five", 10, 100),
	})

	if got := countQueue(s.openSessionByPreset(t, today, saturday, "Five.md")); got != 5 {
		t.Errorf("a preset of five new cards a day offered %d", got)
	}
}

// A second session over the same preset takes up where the first left off: what
// the day has spent is off the allowance the tile was drawn from.
func TestASecondSessionOverAPresetConsumesUpWhereTheFirstLeftOff(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":       term,
		"Five.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Five.md": deckOf("Five", 20, 0),
	})
	if got := unseen(s.openSessionByPreset(t, today, saturday, "Five.md")); got != 5 {
		t.Fatalf("the morning was asked %d new cards, want 5", got)
	}

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, cardID(i), 6*time.Second)
	}

	if got := unseen(s.openSessionByPreset(t, today, saturday.Add(2*time.Hour), "Five.md")); got != 2 {
		t.Errorf("the evening was asked %d new cards, want 2", got)
	}
}

// A deck and a preset are two answers to which cards were meant, and the
// session puts the question back.
func TestNamingADeckAndAPresetTogetherIsRefused(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Steady.md":      preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md": deckOf("Steady", 3, 0),
	})

	_, err := s.openSession(t, today, saturday, flashcards.Scope{
		Deck: "decks/Birds.md", Preset: "Steady.md", IsNamed: true,
	})
	if !errors.Is(err, flashcards.ErrBothNamed) {
		t.Fatalf("a deck and a preset together were answered with %v", err)
	}
}

// A preset no deck points at has nothing to sit to, and says so.
func TestASessionOverAPresetNothingPointsAtIsRefused(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Steady.md":      preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Lonely.md":      preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md": deckOf("Steady", 3, 0),
	})

	_, err := s.openSession(t, today, saturday, flashcards.Scope{Preset: "Lonely.md", IsNamed: true})
	if !errors.Is(err, flashcards.ErrSchedulesNothing) {
		t.Fatalf("a preset nothing points at was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "no deck") {
		t.Errorf("the reason shown is %q", err)
	}
}

func TestASessionOverAPausedPresetIsRefused(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":         term,
		"Paused.md":       preset("new_a_day: 0\nreviews_a_day: 0\n"),
		"decks/Paused.md": deckOf("Paused", 6, 0),
	})

	_, err := s.openSession(t, today, saturday, flashcards.Scope{Preset: "Paused.md", IsNamed: true})
	if !errors.Is(err, flashcards.ErrSchedulesNothing) {
		t.Fatalf("a paused preset was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Errorf("the reason shown is %q", err)
	}
}

// whole of the allowance, and the second is told why there is nothing left.
func TestASessionOverAPresetWhoseDayIsSpentIsRefused(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"Two.md": preset(
			"new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\ncounts: shows\n"),
		"decks/Two.md":  deckOf("Two", 6, 0),
		"decks/Rest.md": deckOf("", 6, 100),
	})
	if got := unseen(s.openSessionByPreset(t, today, saturday, "Two.md")); got != 2 {
		t.Fatalf("the morning was asked %d new cards, want 2", got)
	}

	morning := s.run(t, saturday)
	for i := range 2 {
		answer(t, morning, cardID(i), 6*time.Second)
	}

	_, err := s.openSession(t, today, saturday.Add(2*time.Hour), flashcards.Scope{Preset: "Two.md", IsNamed: true})
	if !errors.Is(err, flashcards.ErrSchedulesNothing) {
		t.Fatalf("a spent day was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "spent") {
		t.Errorf("the reason shown is %q", err)
	}
}

// term is the one card model the budgeted vaults are cut by: one review card, so a card
// is a review card.
const term = "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
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
func deckOf(pos string, cards int, from int) string {
	var naming []string
	if pos != "" {
		naming = append(naming, pos)
	}
	return deckNaming(naming, cards, from)
}

// deckNaming is a deck of as many cards, naming these presets in the order it
// names them.
func deckNaming(pos []string, cards int, from int) string {
	out := "---\ntype: deck\n"
	if len(pos) > 0 {
		out += "links:\n"
		for _, one := range pos {
			out += "  - to: " + one + "\n    role: ref\n    type: preset\n"
		}
	}
	out += "---\n"
	for i := range cards {
		id := fmt.Sprintf("card%06d", from+i)
		out += fmt.Sprintf(
			"\n## Card %d ^%s\n\n[[Term]]\n\n### Word\n\nw%d\n\n### Meaning\n\nm%d\n",
			from+i, id, from+i, from+i)
	}
	return out
}

func (s vaulted) sessionAt(t *testing.T, day review.Day, now time.Time) flashcards.SessionResult {
	t.Helper()
	sat, err := flashcards.NewSession(
		s.assigning, s.standings, s.schedules,
		s.presets, day, func() time.Time { return now },
	).Execute(t.Context(), s.vault, flashcards.Scope{})
	if err != nil {
		t.Fatal(err)
	}
	return sat
}

// byDeck is how many review cards of each deck a session holds.
func byDeck(sat flashcards.SessionResult) map[string]int {
	out := make(map[string]int)
	for _, one := range sat.Queue {
		out[one.Deck]++
	}
	return out
}

func unseen(sat flashcards.SessionResult) int {
	out := 0
	for _, one := range sat.Queue {
		if one.Schedule.IsNew() {
			out++
		}
	}
	return out
}

// cardID is the card id of this place in a deck, as deckOf writes it.
func cardID(pos int) string { return fmt.Sprintf("card%06d", pos) }

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

	// What each preset's cards have cost. Twelve review cards of each answered
	// three times over two months is a history long enough to say what both
	// kinds of answer take.
	for _, days := range []int{90, 80, 75} {
		before := s.run(t, saturday.AddDate(0, 0, -days))
		for i := range 12 {
			answer(t, before, cardID(i), 20*time.Second)
			answer(t, before, cardID(100+i), 4*time.Second)
		}
	}

	got := byDeck(s.sessionAt(t, today, saturday))
	// Twelve review cards of each stand owed. The minute buys three of the slow
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
		targetTime time.Time
		cards      int
	}{
		// Two in the morning of the Saturday is the Friday's review day, which
		// carries the whole of the load.
		{time.Date(2026, 9, 5, 2, 0, 0, 0, time.Local), 8},
		// And two in the morning of the Sunday is the half Saturday itself.
		{time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local), 4},
	} {
		got := byDeck(s.sessionAt(t, today, one.targetTime))["decks/Light.md"]
		if got != one.cards {
			t.Errorf("at %v the deck was asked %d cards, want %d", one.targetTime, got, one.cards)
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
		targetTime time.Time
		cards      int
	}{
		// The last day of it holds all the material still to begin.
		{time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local), 20},
		// The small hours after it are still the day it aims at.
		{time.Date(2026, 9, 6, 2, 0, 0, 0, time.Local), 20},
		{time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local), 0},
	} {
		got := byDeck(s.sessionAt(t, today, one.targetTime))["decks/By.md"]
		if got != one.cards {
			t.Errorf("at %v the deck was asked %d cards, want %d", one.targetTime, got, one.cards)
		}
	}
}

// A second session of the same day takes up where the first left off: what was
// answered since the day opened is off the day's budget.
func TestASecondSessionConsumesUpWhereTheFirstLeftOff(t *testing.T) {
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
		answer(t, morning, cardID(i), 6*time.Second)
	}

	if got := unseen(s.sessionAt(t, today, saturday.Add(2*time.Hour))); got != 2 {
		t.Errorf("the evening was asked %d new cards, want 2", got)
	}
}

// A card the day has already charged for comes round again for nothing under
// `counts: cards`, and spends the budget again under `counts: shows`.
func TestSession_CountsCardsOrShowsForRepeatedCard(t *testing.T) {
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
		answer(t, before, cardID(0), 6*time.Second)
		answer(t, before, cardID(1), 6*time.Second)

		// The first card is put to the person nine times in one session.
		morning := s.run(t, saturday)
		for range 9 {
			answerAgain(t, morning, cardID(0), 4*time.Second)
		}

		got := byDeck(s.sessionAt(t, today, saturday.Add(2*time.Hour)))["decks/Two.md"]
		if got != one.cards {
			t.Errorf("counting in %s the evening was asked %d cards, want %d",
				one.counts, got, one.cards)
		}
	}
}

// A review card answered in an earlier session today is free when it comes round
// in a later one under `counts: cards`, and is charged again under
// `counts: shows`.
func TestACardTheDayHasChargedIsFreeInALaterSession(t *testing.T) {
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
		answer(t, before, cardID(0), 6*time.Second)
		answer(t, before, cardID(1), 6*time.Second)

		answerAgain(t, s.run(t, saturday), cardID(0), 4*time.Second)

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
			answerAgain(t, morning, cardID(i), 4*time.Second)
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
func TestALightDaySplitsTheDaysCards(t *testing.T) {
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
func TestTheMinutesCloseTheDayBelowAGoalOfMinutes(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Short.md":       preset("goal: minutes_a_day\nnew_a_day: 10\nminutes_a_day: 1\n"),
		"decks/Short.md": deckOf("Short", 10, 0),
	})

	target := int(time.Minute / review.DefaultCost.New)
	got := byDeck(s.sessionAt(t, today, saturday))
	if got["decks/Short.md"] != target {
		t.Errorf("a day of one minute was asked %d cards, want %d", got["decks/Short.md"], target)
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
		answer(t, before, cardID(i), 6*time.Second)
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

// countQueue is how many review cards a session holds, over every deck.
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

	target := int(time.Duration(review.Defaults().MinutesADay) *
		time.Minute / review.DefaultCost.New)
	if got := countQueue(one.sessionAt(t, today, saturday)); got != target {
		t.Fatalf("one deck on the defaults was asked %d cards, want %d", got, target)
	}
	if got := countQueue(two.sessionAt(t, today, saturday)); got != target {
		t.Errorf("two decks on the defaults were asked %d cards, want %d", got, target)
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

	target := int(time.Minute / review.DefaultCost.New)
	if got := countQueue(together.sessionAt(t, today, saturday)); got != target {
		t.Fatalf("one minute over both decks was asked %d cards, want %d", got, target)
	}
	if got := countQueue(apart.sessionAt(t, today, saturday)); got != 2*target {
		t.Errorf("a minute under each of two presets was asked %d cards, want %d", got, 2*target)
	}
}

// A day's new cards and a day's reviews are two budgets, so a preset out of the
// one goes on asking the other.
func TestSession_NewCardBudgetIndependentOfReviews(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":      term,
		"New.md":       preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/New.md": deckOf("New", 6, 0),
	})

	// Two cards answered long enough ago to be owed today, which the preset
	// keeps no reviews for.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	answer(t, before, cardID(0), 6*time.Second)
	answer(t, before, cardID(1), 6*time.Second)

	sat := s.sessionAt(t, today, saturday)
	if got := unseen(sat); got != 2 {
		t.Errorf("the day was asked %d new cards, want the two it keeps", got)
	}
	if got := countQueue(sat) - unseen(sat); got != 0 {
		t.Errorf("a day keeping no reviews was asked %d of them", got)
	}
}

// the preset it now names.
func TestSession_RepointedDeckUsesNewPreset(t *testing.T) {
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
		answer(t, morning, cardID(i), 6*time.Second)
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
func TestALimitEditedInTheMiddleOfADayContainsAtOnce(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 20, 0),
	})

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, cardID(i), 6*time.Second)
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
	answer(t, before, cardID(0), 6*time.Second)
	answer(t, before, cardID(1), 6*time.Second)

	// One card put to the person three times, which is one card and three
	// showings.
	morning := s.run(t, saturday)
	for range 3 {
		answerAgain(t, morning, cardID(0), 4*time.Second)
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
		answerAgain(t, morning, cardID(i), 6*time.Second)
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
func TestADayMovedPriorUsStopsThePresetAtOnce(t *testing.T) {
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
		answer(t, run, cardID(i), 6*time.Second)
	}

	for _, one := range []struct {
		targetTime time.Time
		newCount   int
	}{
		{evening.Add(time.Hour), 0},
		// Three in the morning is the Friday's review day still.
		{time.Date(2026, 9, 5, 3, 0, 0, 0, time.Local), 0},
		{time.Date(2026, 9, 5, 5, 0, 0, 0, time.Local), 5},
	} {
		if got := unseen(s.sessionAt(t, today, one.targetTime)); got != one.newCount {
			t.Errorf("at %v the day was asked %d new cards, want %d", one.targetTime, got, one.newCount)
		}
	}
}

// A day carrying none of the load is asked no card, and the day after it opens
// on the whole of the budget.
func TestSession_ZeroLoadDayYieldsNoCards(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":     term,
		"On.md":       preset("new_a_day: 8\nreviews_a_day: 0\nminutes_a_day: 0\nload: {sun: 0}\n"),
		"decks/On.md": deckOf("On", 40, 0),
	})

	for _, one := range []struct {
		targetTime time.Time
		cards      int
	}{
		{time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local), 8},
		{time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local), 0},
		{time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local), 8},
	} {
		got := unseen(s.sessionAt(t, today, one.targetTime))
		if got != one.cards {
			t.Errorf("%v was asked %d new cards, want %d", one.targetTime.Weekday(), got, one.cards)
		}
	}
}

// A review card is new on the day it was first answered at all, which is read off
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
	answer(t, s.runNamed(t, yesterday, saturday.Add(-2*time.Hour)), cardID(0), 6*time.Second)
	answer(t, s.runNamed(t, saturday, yesterday), cardID(0), 6*time.Second)

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
			answer(t, run, cardID(i), 6*time.Second)
		}
	}
	// The working out under the old target, remembered where the next one will
	// look for it.
	was, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got := countQueue(s.sessionAt(t, today, saturday)); got != 0 {
		t.Fatalf("asking for 0.7 of the cards back, %d were owed today", got)
	}

	write(t, s, "On.md", preset("retention: 0.99\nnew_a_day: 0\nreviews_a_day: 50\n"+
		"minutes_a_day: 0\n"))

	now, err := s.schedules.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	on := review.ReviewCardID{Card: cardID(0), Template: "Say it"}
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
			saturday.AddDate(0, 0, day).Format(review.DayLayout)),
		cards: 40,
		gave:  newSpentToday(20),
	}, {
		what: "counting showings, on a day already partly spent",
		front: fmt.Sprintf("goal: by_date\nby_date: %s\nlearned: interval\ninterval: 5\n"+
			"counts: shows\nnew_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 1\n",
			saturday.AddDate(0, 0, day).Format(review.DayLayout)),
		cards: 40,
		gave:  newSpentToday(20),
	}, {
		what: "a date",
		front: fmt.Sprintf("goal: by_date\nby_date: %s\nlearned: interval\ninterval: 5\n"+
			"new_a_day: 1\nreviews_a_day: 0\nminutes_a_day: 1\n",
			saturday.AddDate(0, 0, day).Format(review.DayLayout)),
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

			read, err := s.presets.GetForDeck(t.Context(), s.vault, "decks/On.md")
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

			pointReviews := curve.Points[curve.Now.Index].Reviews
			reviewCards := s.countDayCards(t, today, getNextAdmittedDay(read.Settings, today, saturday))
			if pointReviews != float64(reviewCards) {
				t.Errorf("the curve draws %v and the day hands over %d", pointReviews, reviewCards)
			}
		})
	}
}

// getNextAdmittedDay is the next day of review this preset admits, counting
// from the day holding now, which is the day the curve draws. A preset admitting
// no day at all is answered with the day it was asked about.
func getNextAdmittedDay(p review.Preset, day review.Day, now time.Time) time.Time {
	for range 8 {
		if !p.GetAllowance(day, now, review.Spent{}, 0, 0).IsPaused() {
			return now
		}
		now = day.GetEnd(now)
	}
	return now
}

// answersAt is as many review cards answered this way, this many days before the
// day the vault is sat.
func answersAt(cards, days int, r review.Rating) func(*testing.T, vaulted) {
	return func(t *testing.T, s vaulted) {
		t.Helper()
		before := s.run(t, saturday.AddDate(0, 0, days))
		for i := range cards {
			answerWith(t, before, cardID(i), r, 6*time.Second)
		}
	}
}

// newSpentToday is a day a person is already partway through: review cards owed
// from a month back, some of them answered twice since the day opened.
func newSpentToday(cards int) func(*testing.T, vaulted) {
	return func(t *testing.T, s vaulted) {
		t.Helper()
		answersAt(cards, -30, review.Good)(t, s)
		this := s.run(t, saturday.Add(-time.Hour))
		for i := range 4 {
			answerAgain(t, this, cardID(i), 5*time.Second)
			answer(t, this, cardID(i), 5*time.Second)
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
	// Twenty review cards answered long enough ago to be owed today, and twenty
	// nobody has begun.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	for i := range 20 {
		answer(t, before, cardID(i), 6*time.Second)
	}
	return s
}

// `backlog` says how much of a day goes to the debt before anything unbegun is
// offered. It is a share of the day and not a budget of its own, so it is read
// against the one pot a goal of minutes keeps.
func TestTheBacklogShareReturnsWhatTheDayIsSpentOn(t *testing.T) {
	t.Parallel()
	// Four minutes a day over twenty review cards owed and twenty unbegun, which
	// is more of each than the day can carry.
	const day = "goal: minutes_a_day\nminutes_a_day: 4\n"

	all := s0(t, newBacklogVault(t, day+"backlog: 100\n"))
	if all.owed != 20 || all.newCount == 0 {
		t.Errorf("giving the debt all of the day asked %+v, want every card owed and "+
			"the rest of the day on new ones", all)
	}
	none := s0(t, newBacklogVault(t, day+"backlog: 0\n"))
	if none.newCount != 20 || none.owed == 0 {
		t.Errorf("giving the debt none of the day asked %+v, want every new card and "+
			"the rest of the day on owed ones", none)
	}
	// An odd day gives the extra card to the debt, so the two are never more
	// than one apart.
	half := s0(t, newBacklogVault(t, day+"backlog: 50\n"))
	if half.owed-half.newCount < 0 || half.owed-half.newCount > 1 {
		t.Errorf("splitting the day evenly asked %+v", half)
	}
	if half.owed == 0 || half.newCount == 0 {
		t.Errorf("splitting the day evenly left one side of it unspent: %+v", half)
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
	if was.owed != 20 {
		t.Errorf("the debt was not paid first: %+v", was)
	}
}

// A side that runs short leaves the rest of the day to the other, so a day is
// never left part spent because one half of it had nothing to offer.
func TestASideThatRunsShortPreservesTheDayToTheOther(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md": term,
		"On.md": preset("goal: retention\nbacklog: 50\n" +
			"new_a_day: 20\nreviews_a_day: 20\nminutes_a_day: 0\n"),
		"decks/On.md": deckOf("On", 40, 0),
	})
	// Two review cards owed, and thirty-eight nobody has begun.
	before := s.run(t, saturday.AddDate(0, 0, -30))
	for i := range 2 {
		answer(t, before, cardID(i), 6*time.Second)
	}

	sat := s.sessionAt(t, today, saturday)
	newCards := unseen(sat)
	if owed := countQueue(sat) - newCards; owed != 2 || newCards != 20 {
		t.Errorf("the day asked %d owed and %d new, want the two owed and its twenty new",
			owed, newCards)
	}
}

// A goal of a date carries the whole material by its own reckoning, so the
func TestAGoalOfADateReadsNoBacklogShare(t *testing.T) {
	t.Parallel()
	for _, share := range []string{"backlog: 0\n", "backlog: 100\n"} {
		s := newBacklogVault(t, "goal: by_date\nby_date: 2026-09-14\nlearned: retention\n"+share+
			"new_a_day: 1\nreviews_a_day: 1\nminutes_a_day: 0\n")

		sat := s.sessionAt(t, today, saturday)
		newCards := unseen(sat)
		// Twenty owed, and twenty unbegun over the ten days to the day it aims
		// at, which is two a day.
		if owed := countQueue(sat) - newCards; owed != 20 || newCards != 2 {
			t.Errorf("under %q the day asked %d owed and %d new, want 20 and 2",
				share, owed, newCards)
		}
	}
}

// what a session came to, of each kind.
type session struct{ owed, newCount int }

func s0(t *testing.T, s vaulted) session {
	t.Helper()
	sat := s.sessionAt(t, today, saturday)
	newCards := unseen(sat)
	return session{owed: countQueue(sat) - newCards, newCount: newCards}
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

	elapsed := make(map[string]map[int]int, 2)
	for name, day := range map[string]string{"minutes": minutes, "retention": target} {
		elapsed[name] = make(map[int]int)
		for _, share := range []int{100, 50, 0} {
			one := s0(t, newBacklogVault(t, fmt.Sprintf("%sbacklog: %d\n", day, share)))
			elapsed[name][share] = one.owed
		}
	}
	if elapsed["minutes"][100] == elapsed["minutes"][0] {
		t.Errorf("a day of one pot asked %d of the debt at every share", elapsed["minutes"][100])
	}
	if elapsed["retention"][100] != elapsed["retention"][0] ||
		elapsed["retention"][100] != elapsed["retention"][50] {
		t.Errorf("a day of a count for each side asked %v of the debt, and the share "+
			"decides nothing there", elapsed["retention"])
	}
}

// What a day's budget is spent on decides what a count charges, and leaves the
// minutes alone.
//
// A review card the day has already answered comes round again for no count where
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
		// Every review card answered long enough ago to be owed today, at six
		// seconds an answer, so a minute holds ten of them.
		before := s.run(t, saturday.AddDate(0, 0, -30))
		for i := range 40 {
			answer(t, before, cardID(i), 6*time.Second)
		}

		elapsed := time.Duration(0)
		request := 0
		for range 4 {
			sat := s.sessionAt(t, today, saturday.Add(elapsed))
			if len(sat.Queue) == 0 {
				break
			}
			provided := s.run(t, saturday.Add(elapsed))
			for _, one := range sat.Queue {
				answerAgain(t, provided, one.ID.Card, 6*time.Second)
				elapsed += 6 * time.Second
				request++
			}
		}
		if elapsed > 90*time.Second {
			t.Errorf("counting %s, a day of one minute asked %d review cards and ran %v",
				counts, request, elapsed)
		}
	}
}

// countDayCards is how many review cards one whole day of review hands over,
// driven the way a person drives it: a session at a time until the day has
// nothing left to ask, answering everything each of them holds.
//
// A card the day comes back to is the one card, so a review card is counted once
// however many sessions show it. Each answer takes what a projection costs its
// kind at, so driving the day does not move the day's own arithmetic under it.
func (s vaulted) countDayCards(t *testing.T, day review.Day, now time.Time) int {
	t.Helper()
	reviewCards := make(map[review.ReviewCardID]bool)
	for range 100 {
		sat := s.sessionAt(t, day, now)
		if len(sat.Queue) == 0 {
			return len(reviewCards)
		}
		record := s.run(t, now)
		for _, one := range sat.Queue {
			took := review.DefaultCost.Review
			if one.Schedule.IsNew() {
				took = review.DefaultCost.New
			}
			reviewCards[one.ID] = true
			answer(t, record, one.ID.Card, took)
		}
	}
	t.Fatal("the day went on asking and never ran out")
	return 0
}

// theCeiling is the minutes_a_day the presets below ask for.
const theCeiling = 20 * time.Minute

// A goal is a ceiling over every deck the preset schedules.
//
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

		s := openVault(t, files)
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
		sat, err := flashcards.NewSession(
			s.assigning, s.standings, s.schedules,
			s.presets, day, func() time.Time { return now },
		).Execute(t.Context(), s.vault, flashcards.Scope{})
		if err != nil || len(sat.Queue) == 0 {
			return out
		}
		record := s.run(t, now)
		for _, one := range sat.Queue {
			cost := review.DefaultCost.Review
			if one.Schedule.IsNew() {
				cost = review.DefaultCost.New
			}
			out += cost
			answer(t, record, one.ID.Card, cost)
		}
	}
	t.Fatal("the day never ran out")
	return 0
}

// And the same ceiling holds when a person sits to one deck at a time.
func TestSessionDeckByDeckStaysBelowTheOneCeiling(t *testing.T) {
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

	s := openVault(t, files)
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
	var out time.Duration
	for d := range 3 {
		cardScope := flashcards.Scope{Deck: fmt.Sprintf("decks/D%d.md", d)}
		for range 200 {
			sat, err := flashcards.NewSession(
				s.assigning, s.standings, s.schedules,
				s.presets, today, func() time.Time { return now },
			).Execute(t.Context(), s.vault, cardScope)
			if err != nil || len(sat.Queue) == 0 {
				break
			}
			record := s.run(t, now)
			for _, one := range sat.Queue {
				cost := review.DefaultCost.Review
				if one.Schedule.IsNew() {
					cost = review.DefaultCost.New
				}
				out += cost
				answer(t, record, one.ID.Card, cost)
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
func TestOneDeckIsHandedNoMoreThanTheDayContains(t *testing.T) {
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

	s := openVault(t, files)
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
	runSession := func(scope flashcards.Scope) flashcards.SessionResult {
		sat, err := flashcards.NewSession(
			s.assigning, s.standings, s.schedules,
			s.presets, today, func() time.Time { return now },
		).Execute(t.Context(), s.vault, scope)
		if err != nil {
			t.Fatal(err)
		}
		return sat
	}

	whole := 0
	for _, target := range byDeck(runSession(flashcards.Scope{})) {
		whole += target
	}
	if whole != 30 {
		t.Errorf("the whole vault hands over %d review cards, and the day holds 30", whole)
	}
	for d := range 3 {
		deck := fmt.Sprintf("decks/D%d.md", d)
		if got := len(runSession(flashcards.Scope{Deck: deck}).Queue); got > 30 {
			t.Errorf("%s alone hands over %d review cards, and the day holds 30", deck, got)
		}
	}
}

// when is the instant the divided days below are worked out at.
var when = time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)

// counts is a preset whose day is closed by a count of cards nobody has begun.
const counts = "goal: retention\nnew_a_day: 30\nreviews_a_day: 0\n"

// deckAt is the path of the deck standing at this place.
func deckAt(pos int) string { return fmt.Sprintf("decks/D%d.md", pos) }

// newDeckVault is a vault of one preset and as many decks, each holding the
// cards it is given and none of them answered.
func newDeckVault(t testing.TB, front string, decks ...int) vaulted {
	t.Helper()
	files := map[string]string{"Term.md": term, "Steady.md": preset(front)}
	for item, cards := range decks {
		files[deckAt(item)] = deckOf("Steady", cards, item*1000)
	}
	return openVault(t, files)
}

func (s vaulted) rows(t *testing.T, now time.Time) map[string]int {
	t.Helper()
	out, err := s.newCountCardsDueAt(today, func() time.Time { return now }).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	target := make(map[string]int, len(out.Decks))
	for _, one := range out.Decks {
		target[one.Deck] = one.Due + one.New
	}
	return target
}

// pressDeck is how many review cards pressing one deck hands over.
func (s vaulted) pressDeck(t *testing.T, now time.Time, deck string) int {
	t.Helper()
	sat, err := s.openSession(t, today, now, flashcards.Scope{Deck: deck})
	if err != nil {
		t.Fatal(err)
	}
	return len(sat.Queue)
}

// reviewDeckSession answers everything one deck hands over, session again until it hands
// over nothing, and says how many review cards went through. A card the day comes
// back to is the one card.
func (s vaulted) reviewDeckSession(t *testing.T, now time.Time, deck string) int {
	t.Helper()
	reviewCards := make(map[review.ReviewCardID]bool)
	for range 100 {
		sat, err := s.openSession(t, today, now, flashcards.Scope{Deck: deck})
		if err != nil {
			t.Fatal(err)
		}
		if len(sat.Queue) == 0 {
			return len(reviewCards)
		}
		record := s.run(t, now)
		for _, one := range sat.Queue {
			took := review.DefaultCost.Review
			if one.Schedule.IsNew() {
				took = review.DefaultCost.New
			}
			reviewCards[one.ID] = true
			answer(t, record, one.ID.Card, took)
		}
	}
	t.Fatal("the deck went on asking and never ran out")
	return 0
}

// Decks owing the same take the day in equal parts.
//
// Adding a deck does not add to the day's work; it spreads the same work over
// more decks.
func TestDecksOwingTheSameDivideTheDayEqually(t *testing.T) {
	t.Parallel()
	s := newDeckVault(t, counts, 40, 40, 40)
	got := s.rows(t, when)
	for item := range 3 {
		if got[deckAt(item)] != 10 {
			t.Errorf("%s holds %d of a day of 30 over three equal decks, and a third is 10",
				deckAt(item), got[deckAt(item)])
		}
	}
}

// A deck owing nine times another's takes nine times the share.
func TestADeckOwingMoreConsumesTheLargerShare(t *testing.T) {
	t.Parallel()
	s := newDeckVault(t, counts, 90, 10)
	got := s.rows(t, when)
	if got[deckAt(0)] != 27 || got[deckAt(1)] != 3 {
		t.Errorf("a deck owing 90 beside one owing 10 holds %d and %d of a day of 30, and nine times the share is 27 and 3",
			got[deckAt(0)], got[deckAt(1)])
	}
}

// A deck that cannot use a whole share leaves the rest to the others, and the
// day still spends what it holds.
//
// Twenty minutes buys sixty cards nobody has begun. The deck of one card holds
// too small a share to buy even that card, and what the shares leave over buys
// the cards the other two could not afford out of their own.
func TestADeckThatCannotUseItsSharePreservesItToTheOthers(t *testing.T) {
	t.Parallel()
	s := newDeckVault(t, "goal: minutes_a_day\nminutes_a_day: 20\n"+
		"new_a_day: 9999\nreviews_a_day: 9999\n", 60, 60, 1)
	got := s.rows(t, when)
	if got[deckAt(2)] != 0 {
		t.Errorf("%s holds %d out of a share that buys no card", deckAt(2), got[deckAt(2)])
	}
	for item := range 2 {
		if got[deckAt(item)] < 29 {
			t.Errorf("%s holds %d of the sixty the two of them divide", deckAt(item), got[deckAt(item)])
		}
	}
	whole := got[deckAt(0)] + got[deckAt(1)] + got[deckAt(2)]
	if whole != 60 {
		t.Errorf("the day hands over %d and twenty minutes buys 60", whole)
	}
}

// A deck holding nothing takes no share of the day.
func TestADeckContainingNothingConsumesNoShare(t *testing.T) {
	t.Parallel()
	s := newDeckVault(t, counts, 0, 40, 40)
	got := s.rows(t, when)
	if got[deckAt(0)] != 0 {
		t.Errorf("a deck of no cards holds %d", got[deckAt(0)])
	}
	if got[deckAt(1)] != 15 || got[deckAt(2)] != 15 {
		t.Errorf("two decks beside an empty one hold %d and %d of a day of 30, and half is 15",
			got[deckAt(1)], got[deckAt(2)])
	}
}

// One deck takes the whole day, which is what it did before any of it was
// divided.
func TestOneDeckConsumesTheWholeDay(t *testing.T) {
	t.Parallel()
	s := newDeckVault(t, counts, 40)
	if got := s.rows(t, when)[deckAt(0)]; got != 30 {
		t.Errorf("the one deck of the vault holds %d of a day of 30", got)
	}
}

// The minutes are divided the same way the counts are.
func TestTheMinutesOfADayAreDividedOverTheDecksToo(t *testing.T) {
	t.Parallel()
	s := newDeckVault(t, "goal: minutes_a_day\nminutes_a_day: 20\n"+
		"new_a_day: 9999\nreviews_a_day: 9999\n", 60, 60, 60)
	got := s.rows(t, when)
	// A third of twenty minutes buys twenty cards nobody has begun.
	for item := range 3 {
		if got[deckAt(item)] != 20 {
			t.Errorf("%s holds %d of twenty minutes over three equal decks, and a third buys 20 cards",
				deckAt(item), got[deckAt(item)])
		}
	}
}

func TestTheDeckRowIsWhatPressingTheDeckPassesOver(t *testing.T) {
	t.Parallel()
	for _, shape := range []struct {
		front string
		decks []int
	}{
		{counts, []int{40, 40, 40}},
		{counts, []int{90, 10}},
		{counts, []int{2, 40, 40}},
		{counts, []int{0, 40, 40}},
		{counts, []int{40}},
		{counts, []int{5, 5, 5, 5, 5}},
		{"goal: minutes_a_day\nminutes_a_day: 20\nnew_a_day: 9999\nreviews_a_day: 9999\n",
			[]int{60, 12, 3}},
		{"goal: by_date\nby_date: 2026-06-01\nlearned: retention\n", []int{30, 10, 10}},
	} {
		t.Run(fmt.Sprint(shape.decks), func(t *testing.T) {
			s := newDeckVault(t, shape.front, shape.decks...)
			got := s.rows(t, when)
			for item := range shape.decks {
				deck := deckAt(item)
				if pressed := s.pressDeck(t, when, deck); pressed != got[deck] {
					t.Errorf("%s is at %d on the overview and yields %d when pressed",
						deck, got[deck], pressed)
				}
			}
		})
	}
}

// Session deck by deck spends the same day whichever deck is sat first, and no
// deck's own share grows because another was sat before it.
func TestSessionTheDecksInAnyOrderSpendsTheOneDay(t *testing.T) {
	t.Parallel()
	for _, order := range [][]int{{0, 1, 2}, {2, 1, 0}, {1, 0, 2}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			s := newDeckVault(t, counts, 40, 40, 40)
			whole, each := 0, make(map[string]int)
			for _, item := range order {
				took := s.reviewDeckSession(t, when, deckAt(item))
				each[deckAt(item)] = took
				whole += took
			}
			if whole != 30 {
				t.Errorf("session the decks in the order %v spends %d of a day of 30", order, whole)
			}
			for entry := range 3 {
				if each[deckAt(entry)] != 10 {
					t.Errorf("%s spent %d of a day of 30 over three equal decks, and a third is 10",
						deckAt(entry), each[deckAt(entry)])
				}
			}
		})
	}
}

// The division turns on what each deck owes and not on where the deck stands,
// so the same vault divides the same day whatever order its files are walked
// in.
func TestTheDivisionDoesNotTurnOnWhereADeckExists(t *testing.T) {
	t.Parallel()
	first := newDeckVault(t, counts, 90, 10).rows(t, when)
	second := newDeckVault(t, counts, 10, 90).rows(t, when)
	if first[deckAt(0)] != second[deckAt(1)] || first[deckAt(1)] != second[deckAt(0)] {
		t.Errorf("a deck of 90 beside one of 10 holds %d and %d, and the other way round %d and %d",
			first[deckAt(0)], first[deckAt(1)], second[deckAt(0)], second[deckAt(1)])
	}
}
