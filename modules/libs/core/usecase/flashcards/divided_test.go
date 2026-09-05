package flashcards

import (
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"
)

// owing generates what a preset's decks owe of one of its budgets, and the
// budget there is to divide between them. A budget larger than the whole of
// what is owed and one smaller than any single share both fall inside the
// range, so both ends of the division are generated as often as the middle.
func owing(t *rapid.T) (float64, []float64) {
	owes := rapid.SliceOfN(rapid.IntRange(0, 400), 0, 8).Draw(t, "owes")
	out := make([]float64, len(owes))
	for at, one := range owes {
		out[at] = float64(one)
	}
	return float64(rapid.IntRange(0, 1200).Draw(t, "budget")), out
}

// totalled is what a set of decks owes altogether.
func totalled(owes []float64) float64 {
	var out float64
	for _, one := range owes {
		out += one
	}
	return out
}

// A day is handed out whole: what the decks take between them is the budget
// where the budget is the smaller, and everything owed where it is not. What no
// deck could use is not left on the day.
//
// ADR-0041 — a preset's day is divided over the decks it schedules.
func TestADividedDayIsHandedOutWhole(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := owing(t)
		got := totalled(divided(budget, owes))
		want := math.Min(budget, totalled(owes))
		if got != want {
			t.Fatalf("a budget of %v over %v handed out %v, want %v",
				budget, owes, got, want)
		}
	})
}

// No deck takes more of the day than it owes of it, and a deck owing nothing
// takes nothing: a share is what a deck owes, so there is nothing to hand a
// deck that owes nothing.
//
// ADR-0041 — the shares are proportional to what the decks owe.
func TestNoDeckTakesMoreOfTheDayThanItOwes(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := owing(t)
		for at, share := range divided(budget, owes) {
			if share < 0 || share > owes[at] {
				t.Fatalf("a deck owing %v took %v of a budget of %v over %v",
					owes[at], share, budget, owes)
			}
		}
	})
}

// A share stands at the proportion of the day its deck owes, rounded one way or
// the other: three decks owing the same take a third each, and a deck owing
// nine times another's takes nine times the share. What the proportions leave
// over is under one card a deck, so no share is a whole card away from the
// proportion it is drawn from.
//
// ADR-0041 — the shares are proportional, and what they leave over goes by the
// largest fraction.
func TestAShareIsTheProportionOfTheDayItsDeckOwes(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := owing(t)
		total := totalled(owes)
		if total <= 0 || budget >= total {
			return
		}
		for at, share := range divided(budget, owes) {
			exact := budget * owes[at] / total
			if share < math.Floor(exact) || share > math.Ceil(exact) {
				t.Fatalf("a deck owing %v of %v took %v of a budget of %v, "+
					"where its proportion of it is %v",
					owes[at], total, share, budget, exact)
			}
		}
	})
}

// A deck owing more of the day never takes less of it than a deck owing less,
// and decks owing the same take what is left over in the order they are given.
//
// ADR-0041 — a deck owing nine times another's takes nine times the share, and
// decks standing equal take the remainder in the order their paths stand.
func TestADeckOwingMoreNeverTakesLess(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := owing(t)
		out := divided(budget, owes)
		for i := range owes {
			for j := range owes {
				ahead := owes[i] > owes[j] || (owes[i] == owes[j] && i < j)
				if ahead && out[i] < out[j] {
					t.Fatalf("a deck owing %v took %v where one owing %v took %v, "+
						"out of a budget of %v over %v",
						owes[i], out[i], owes[j], out[j], budget, owes)
				}
			}
		}
	})
}

// The same decks owing the same take the same shares between them however they
// are walked. The order says which deck takes a card the proportions left over,
// and never how many cards there are to take.
//
// ADR-0041 — a vault divides the same day however its files are walked.
func TestTheOrderOfTheDecksDoesNotChangeTheShares(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := owing(t)
		places := make([]int, len(owes))
		for at := range places {
			places[at] = at
		}
		walked := make([]float64, len(owes))
		for at, from := range rapid.Permutation(places).Draw(t, "order") {
			walked[at] = owes[from]
		}

		was, now := divided(budget, owes), divided(budget, walked)
		slices.Sort(was)
		slices.Sort(now)
		if !slices.Equal(was, now) {
			t.Fatalf("a budget of %v gives %v over %v, and %v over %v",
				budget, was, owes, now, walked)
		}
	})
}
