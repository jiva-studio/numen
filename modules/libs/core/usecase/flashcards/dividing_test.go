package flashcards_test

import (
	"fmt"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// when is the instant the divided days below are worked out at.
var when = time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)

// counts is a preset whose day is closed by a count of cards nobody has begun.
const counts = "goal: retention\nnew_a_day: 30\nreviews_a_day: 0\n"

// deckAt is the path of the deck standing at this place.
func deckAt(at int) string { return fmt.Sprintf("decks/D%d.md", at) }

// dividing is a vault of one preset and as many decks, each holding the cards
// it is given and none of them answered.
func dividing(t testing.TB, front string, decks ...int) vaulted {
	t.Helper()
	files := map[string]string{"Term.md": term, "Steady.md": preset(front)}
	for at, cards := range decks {
		files[deckAt(at)] = deckOf("Steady", cards, at*1000)
	}
	return opened(t, files)
}

// rows is what the front door says each deck holds at this instant.
func (s vaulted) rows(t *testing.T, now time.Time) map[string]int {
	t.Helper()
	out, err := s.owedAt(today, func() time.Time { return now }).Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	held := make(map[string]int, len(out.Decks))
	for _, one := range out.Decks {
		held[one.Deck] = one.Due + one.New
	}
	return held
}

// presses is how many card faces pressing one deck hands over.
func (s vaulted) presses(t *testing.T, now time.Time, deck string) int {
	t.Helper()
	sat, err := s.over(t, today, now, flashcards.OverDeck(deck))
	if err != nil {
		t.Fatal(err)
	}
	return len(sat.Asked)
}

// sits answers everything one deck hands over, sitting again until it hands
// over nothing, and says how many card faces went through. A card the day comes
// back to is the one card.
func (s vaulted) sits(t *testing.T, now time.Time, deck string) int {
	t.Helper()
	faces := make(map[history.CardFace]bool)
	for range 100 {
		sat, err := s.over(t, today, now, flashcards.OverDeck(deck))
		if err != nil {
			t.Fatal(err)
		}
		if len(sat.Asked) == 0 {
			return len(faces)
		}
		record := s.run(t, now)
		for _, one := range sat.Asked {
			took := history.DefaultCost.Review
			if !one.Schedule.Seen() {
				took = history.DefaultCost.New
			}
			faces[one.CardFace] = true
			answer(t, record, one.CardFace.Card, took)
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
	s := dividing(t, counts, 40, 40, 40)
	got := s.rows(t, when)
	for at := range 3 {
		if got[deckAt(at)] != 10 {
			t.Errorf("%s holds %d of a day of 30 over three equal decks, and a third is 10",
				deckAt(at), got[deckAt(at)])
		}
	}
}

// A deck owing nine times another's takes nine times the share.
func TestADeckOwingMoreTakesTheLargerShare(t *testing.T) {
	t.Parallel()
	s := dividing(t, counts, 90, 10)
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
func TestADeckThatCannotUseItsShareLeavesItToTheOthers(t *testing.T) {
	t.Parallel()
	s := dividing(t, "goal: minutes_a_day\nminutes_a_day: 20\n"+
		"new_a_day: 9999\nreviews_a_day: 9999\n", 60, 60, 1)
	got := s.rows(t, when)
	if got[deckAt(2)] != 0 {
		t.Errorf("%s holds %d out of a share that buys no card", deckAt(2), got[deckAt(2)])
	}
	for at := range 2 {
		if got[deckAt(at)] < 29 {
			t.Errorf("%s holds %d of the sixty the two of them divide", deckAt(at), got[deckAt(at)])
		}
	}
	whole := got[deckAt(0)] + got[deckAt(1)] + got[deckAt(2)]
	if whole != 60 {
		t.Errorf("the day hands over %d and twenty minutes buys 60", whole)
	}
}

// A deck holding nothing takes no share of the day.
func TestADeckHoldingNothingTakesNoShare(t *testing.T) {
	t.Parallel()
	s := dividing(t, counts, 0, 40, 40)
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
func TestOneDeckTakesTheWholeDay(t *testing.T) {
	t.Parallel()
	s := dividing(t, counts, 40)
	if got := s.rows(t, when)[deckAt(0)]; got != 30 {
		t.Errorf("the one deck of the vault holds %d of a day of 30", got)
	}
}

// The minutes are divided the same way the counts are.
func TestTheMinutesOfADayAreDividedOverTheDecksToo(t *testing.T) {
	t.Parallel()
	s := dividing(t, "goal: minutes_a_day\nminutes_a_day: 20\n"+
		"new_a_day: 9999\nreviews_a_day: 9999\n", 60, 60, 60)
	got := s.rows(t, when)
	// A third of twenty minutes buys twenty cards nobody has begun.
	for at := range 3 {
		if got[deckAt(at)] != 20 {
			t.Errorf("%s holds %d of twenty minutes over three equal decks, and a third buys 20 cards",
				deckAt(at), got[deckAt(at)])
		}
	}
}

// What the front door says of a deck is what pressing that deck hands over.
func TestTheDeckRowIsWhatPressingTheDeckHandsOver(t *testing.T) {
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
			s := dividing(t, shape.front, shape.decks...)
			got := s.rows(t, when)
			for at := range shape.decks {
				deck := deckAt(at)
				if pressed := s.presses(t, when, deck); pressed != got[deck] {
					t.Errorf("%s stands at %d on the front door and hands over %d when pressed",
						deck, got[deck], pressed)
				}
			}
		})
	}
}

// Sitting deck by deck spends the same day whichever deck is sat first, and no
// deck's own share grows because another was sat before it.
func TestSittingTheDecksInAnyOrderSpendsTheOneDay(t *testing.T) {
	t.Parallel()
	for _, order := range [][]int{{0, 1, 2}, {2, 1, 0}, {1, 0, 2}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			s := dividing(t, counts, 40, 40, 40)
			whole, each := 0, make(map[string]int)
			for _, at := range order {
				took := s.sits(t, when, deckAt(at))
				each[deckAt(at)] = took
				whole += took
			}
			if whole != 30 {
				t.Errorf("sitting the decks in the order %v spends %d of a day of 30", order, whole)
			}
			for at := range 3 {
				if each[deckAt(at)] != 10 {
					t.Errorf("%s spent %d of a day of 30 over three equal decks, and a third is 10",
						deckAt(at), each[deckAt(at)])
				}
			}
		})
	}
}

// The division turns on what each deck owes and not on where the deck stands,
// so the same vault divides the same day whatever order its files are walked
// in.
func TestTheDivisionDoesNotTurnOnWhereADeckStands(t *testing.T) {
	t.Parallel()
	first := dividing(t, counts, 90, 10).rows(t, when)
	second := dividing(t, counts, 10, 90).rows(t, when)
	if first[deckAt(0)] != second[deckAt(1)] || first[deckAt(1)] != second[deckAt(0)] {
		t.Errorf("a deck of 90 beside one of 10 holds %d and %d, and the other way round %d and %d",
			first[deckAt(0)], first[deckAt(1)], second[deckAt(0)], second[deckAt(1)])
	}
}
