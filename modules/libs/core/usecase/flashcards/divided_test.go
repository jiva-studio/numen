package flashcards

import (
	"fmt"
	"maps"
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
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

// deckLoad is one deck of a preset and what it holds: cards that are owed, and
// cards nobody has begun.
type deckLoad struct {
	path        string
	owed, fresh int
}

// loading generates the decks of one preset, each under a path of its own.
func loading(t *rapid.T) []deckLoad {
	decks := rapid.IntRange(1, 5).Draw(t, "decks")
	out := make([]deckLoad, decks)
	for at := range out {
		out[at] = deckLoad{
			path:  fmt.Sprintf("decks/%c.md", 'A'+at),
			owed:  rapid.IntRange(0, 8).Draw(t, "owed"),
			fresh: rapid.IntRange(0, 8).Draw(t, "fresh"),
		}
	}
	return out
}

// walking is what a day of this budget hands over, with the decks reached in
// this order: the card faces the sitting takes, by name.
//
// The order stands for the order the vault was walked in. Each deck's own cards
// keep the order they stand in, which is the deck's and not the walk's.
func walking(load []deckLoad, order []int, keeps review.Budget) map[review.CardFaceID]bool {
	day := &budgets{
		under: make(map[review.CardFaceID]string),
		left: map[string]*allowance{"Preset.md": {
			admits: review.Allowance{
				Keeps: keeps, New: keeps.New, Reviews: keeps.Reviews,
				Closes:  review.Closes{New: review.ClosedNew, Reviews: review.ClosedReviews},
				Backlog: review.AllBacklog,
			},
			cost:   review.DefaultCost,
			counts: review.CountsCards,
		}},
		decks: make(map[string]string),
		faced: make(map[review.CardFaceID]bool),
		sat:   make(map[string]review.Spent),
	}

	var owed, fresh []CardFace
	for _, at := range order {
		one := load[at]
		day.decks[one.path] = "Preset.md"
		for card := range one.owed {
			face := CardFace{Deck: one.path, ID: review.CardFaceID{
				Card: fmt.Sprintf("%s/owed/%d", one.path, card),
			}}
			day.under[face.ID] = "Preset.md"
			owed = append(owed, face)
		}
		for card := range one.fresh {
			face := CardFace{Deck: one.path, ID: review.CardFaceID{
				Card: fmt.Sprintf("%s/fresh/%d", one.path, card),
			}}
			day.under[face.ID] = "Preset.md"
			fresh = append(fresh, face)
		}
	}

	took := day.spends(owed, fresh)
	out := make(map[review.CardFaceID]bool)
	for at, one := range owed {
		if took.owed[at] {
			out[one.ID] = true
		}
	}
	for at, one := range fresh {
		if took.fresh[at] {
			out[one.ID] = true
		}
	}
	return out
}

// A vault hands over the same cards however its files are walked. The decks are
// handed their shares in the order their paths stand, so the order a walk
// reached them in decides nothing.
//
// ADR-0041 — a vault divides the same day however its files are walked.
func TestTheWalkOfAVaultDoesNotChangeWhatADayHandsOver(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		load := loading(t)
		keeps := review.Budget{
			New:     rapid.IntRange(0, 20).Draw(t, "new"),
			Reviews: rapid.IntRange(0, 20).Draw(t, "reviews"),
		}
		places := make([]int, len(load))
		for at := range places {
			places[at] = at
		}

		was := walking(load, places, keeps)
		now := walking(load, rapid.Permutation(places).Draw(t, "walk"), keeps)
		if !maps.Equal(was, now) {
			t.Fatalf("a day of %d new and %d reviews over %v hands over %v "+
				"walked in order and %v walked otherwise",
				keeps.New, keeps.Reviews, load, was, now)
		}
	})
}
