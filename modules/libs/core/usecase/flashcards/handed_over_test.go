package flashcards

import (
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

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
// this order: the card faces the session takes, by name.
//
// The order stands for the order the vault was walked in. Each deck's own cards
// keep the order they stand in, which is the deck's and not the walk's. A
// budget of no minutes is a day that does not close on them.
func walking(load []deckLoad, order []int, keeps review.Budget) map[review.CardFaceID]bool {
	limits := review.Limits{New: review.ClosedNew, Reviews: review.ClosedReviews}
	if keeps.Minutes > 0 {
		limits.Minutes = review.ClosedMinutes
	}
	day := &budgets{
		under: make(map[review.CardFaceID]string),
		left: map[string]*allowance{"Preset.md": {
			admits: review.Allowance{
				Keeps: keeps, New: keeps.New, Reviews: keeps.Reviews,
				Minutes: time.Duration(keeps.Minutes * float64(time.Minute)),
				Limits:  limits,
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

// handedOver is how many cards each deck was handed of a day of this budget.
func handedOver(load []deckLoad, keeps review.Budget) map[string]int {
	places := make([]int, len(load))
	for at := range places {
		places[at] = at
	}
	out := make(map[string]int, len(load))
	for face := range walking(load, places, keeps) {
		for _, one := range load {
			if strings.HasPrefix(face.Card, one.path+"/") {
				out[one.path]++
			}
		}
	}
	return out
}

// A day of one minute over a deck of one unbegun card and a deck of five: a
// card costs twenty seconds, so the shares of the minute — ten seconds and
// fifty — buy two cards in the larger deck and none in the smaller. The twenty
// seconds nobody could spend buy a third card in the larger deck, which is
// where four cards are still standing, and not the one card of the deck whose
// name sorts first.
//
// What no deck can use is offered round again, to the deck still holding most.
func TestWhatNoDeckCouldUseGoesToTheDeckHoldingMost(t *testing.T) {
	t.Parallel()
	keeps := review.Budget{New: 6, Minutes: 1}
	one, five := "decks/a.md", "decks/b.md"

	got := handedOver([]deckLoad{{path: one, fresh: 1}, {path: five, fresh: 5}}, keeps)
	if want := (map[string]int{five: 3}); !maps.Equal(got, want) {
		t.Fatalf("a day of one minute handed over %v, want %v", got, want)
	}
	// The same two decks under each other's names hand over the same cards.
	got = handedOver([]deckLoad{{path: five, fresh: 1}, {path: one, fresh: 5}}, keeps)
	if want := (map[string]int{one: 3}); !maps.Equal(got, want) {
		t.Fatalf("a day of one minute over the same decks renamed handed over "+
			"%v, want %v", got, want)
	}
}

// A vault hands over the same cards however its files are walked. The decks are
// handed their shares in the order their paths stand, so the order a walk
// reached them in decides nothing.
//
// A vault divides the same day however its files are walked.
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
