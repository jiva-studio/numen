package review_test

import (
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// dividing generates what a preset's decks owe of one of its budgets, and the
// budget there is to divide between them. A budget larger than the whole of
// what is owed and one smaller than any single share both fall inside the
// range, so both ends of the division are generated as often as the middle.
func dividing(t *rapid.T) (int, []int) {
	owes := rapid.SliceOfN(rapid.IntRange(0, 400), 0, 8).Draw(t, "owes")
	return rapid.IntRange(0, 1200).Draw(t, "budget"), owes
}

// totalled is what a set of decks owes altogether.
func totalled(owes []int) int {
	out := 0
	for _, one := range owes {
		out += one
	}
	return out
}

// A day is handed out whole: what the decks take between them is the budget
// where the budget is the smaller, and everything owed where it is not. What no
// deck could use is not left on the day.
//
// A preset's day is divided over the decks it schedules.
func TestTheSharesOfADayAreHandedOutWhole(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := dividing(t)
		got := totalled(review.Shares(budget, owes))
		want := min(budget, totalled(owes))
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
// The shares are proportional to what the decks owe.
func TestNoDeckTakesMoreOfTheDayThanItOwes(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := dividing(t)
		for at, share := range review.Shares(budget, owes) {
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
// The shares are proportional, and what they leave over goes by the largest
// fraction.
func TestAShareIsTheProportionOfTheDayItsDeckOwes(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := dividing(t)
		total := totalled(owes)
		if total <= 0 || budget >= total {
			return
		}
		for at, share := range review.Shares(budget, owes) {
			exact := float64(budget) * float64(owes[at]) / float64(total)
			if float64(share) < math.Floor(exact) || float64(share) > math.Ceil(exact) {
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
// A deck owing nine times another's takes nine times the share, and where the
// fractions stand equal the deck owing more takes the remainder.
func TestADeckOwingMoreNeverTakesLess(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := dividing(t)
		out := review.Shares(budget, owes)
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

// The decks take the same shares between them whatever order they are given in.
// A share is worked out from what its deck owes and from nothing else, so
// renaming a deck's file moves no card from one deck to another.
//
// The shares are proportional to what the decks owe, and where the fractions
// stand equal the deck owing more takes the remainder.
func TestTheOrderDecksAreGivenInDoesNotChangeTheShares(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		budget, owes := dividing(t)
		places := make([]int, len(owes))
		for at := range places {
			places[at] = at
		}
		otherwise := make([]int, len(owes))
		for at, one := range rapid.Permutation(places).Draw(t, "order") {
			otherwise[at] = owes[one]
		}

		was, now := review.Shares(budget, owes), review.Shares(budget, otherwise)
		slices.Sort(was)
		slices.Sort(now)
		if !slices.Equal(was, now) {
			t.Fatalf("a budget of %v over %v was handed out as %v, and as %v "+
				"with the decks given in another order", budget, owes, was, now)
		}
	})
}

// Decks owing quite different amounts stand at the same fraction: half a day
// leaves a deck owing an odd number half a card over, whatever that number is,
// so a deck owing one and a deck owing seven are level. The two cards over go to
// the two decks owing most of those standing level, and not to the decks owing
// one card each.
//
// Where the fractions stand equal the deck owing more takes the remainder.
func TestDecksLevelOnTheFractionAreSplitByWhatTheyOwe(t *testing.T) {
	t.Parallel()
	owes := []int{0, 1, 1, 7, 239, 400}
	want := []int{0, 0, 0, 4, 120, 200}
	if got := review.Shares(324, owes); !slices.Equal(got, want) {
		t.Fatalf("half a day over %v was handed out as %v, want %v",
			owes, got, want)
	}
}
