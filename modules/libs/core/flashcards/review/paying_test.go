package review_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// spending is a day standing at a share of itself for the debt, drawn over the
// whole of what the share may be.
func spending(t *rapid.T) review.Allowance {
	return review.Allowance{Backlog: rapid.IntRange(0, review.AllBacklog).Draw(t, "backlog")}
}

// A day with a card on both sides has spent its share on the debt at every
// point of the run, an odd card going to the debt: after any number of cards,
// what the debt has taken is that number of the share, rounded up.
//
// The count is worked out here from the share alone, so a day at the whole of
// it is all debt, a day at none of it is all new cards, and a day at half of it
// takes them one and one beginning with the debt.
//
// ADR-0038 — backlog is how much of a day goes to what is overdue before
// anything new is offered.
func TestTheDebtTakesItsShareOfEveryPointOfTheRun(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		day := spending(t)
		cards := rapid.IntRange(0, 60).Draw(t, "cards")

		var debt, begun int
		for taken := 1; taken <= cards; taken++ {
			if day.Paying(debt, begun, true, true) {
				debt++
			} else {
				begun++
			}
			// The share of a whole number of cards, rounded up.
			want := (taken*day.Backlog + review.AllBacklog - 1) / review.AllBacklog
			if debt != want {
				t.Fatalf("a day at a share of %d took %d of its first %d cards "+
					"from the debt, want %d", day.Backlog, debt, taken, want)
			}
		}
	})
}

// A side that runs short leaves the rest of the day to the other, so a day is
// never left unspent: whatever the share says, a day holding cards on one side
// alone spends the whole of what it admits on that side.
//
// ADR-0038 — a side that runs short leaves the rest of the day to the other.
func TestASideThatRunsShortLeavesTheDayToTheOther(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		day := spending(t)
		owed := rapid.IntRange(0, 30).Draw(t, "owed")
		fresh := rapid.IntRange(0, 30).Draw(t, "fresh")
		admits := rapid.IntRange(0, 60).Draw(t, "admits")

		var debt, begun int
		for debt+begun < admits && (debt < owed || begun < fresh) {
			if day.Paying(debt, begun, debt < owed, begun < fresh) {
				debt++
			} else {
				begun++
			}
		}
		if got, want := debt+begun, min(admits, owed+fresh); got != want {
			t.Fatalf("a day of %d admitting %d over %d owed and %d unbegun "+
				"spent %d of itself, want %d",
				day.Backlog, admits, owed, fresh, got, want)
		}
		if debt > owed || begun > fresh {
			t.Fatalf("a day took %d of %d owed and %d of %d unbegun",
				debt, owed, begun, fresh)
		}
	})
}

// A budget counting cards charges a card face the first time the day answers
// it and every further showing of it that day is free, so what a day of any
// number of showings costs is the faces it showed. A budget counting showings
// charges every one of them.
//
// ADR-0038 — a budget counts cards, and may be told to count showings.
func TestWhatADayOfShowingsCosts(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		counts := rapid.SampledFrom([]review.Counts{
			review.CountsCards, review.CountsShows,
		}).Draw(t, "counts")
		shows := rapid.SliceOfN(rapid.IntRange(0, 5), 0, 40).Draw(t, "shows")

		faced := make(map[int]bool, len(shows))
		var charged int
		for _, face := range shows {
			if counts.Charges(faced[face]) {
				charged++
			}
			faced[face] = true
		}

		want := len(faced)
		if counts == review.CountsShows {
			want = len(shows)
		}
		if charged != want {
			t.Fatalf("a day counting %s spent %d on the showings %v, want %d",
				counts, charged, shows, want)
		}
	})
}
