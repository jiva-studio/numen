package review_test

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Only a goal of minutes reads the backlog share, which divides one budget
// between the debt and new material. Under a goal of retention the share
// decides nothing, and a goal of a date covers the whole material by its own
// computation.
func TestWhichSettingsAGoalReads(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime}
	for _, one := range []struct {
		goal       review.Goal
		shouldRead bool
	}{
		{review.GoalMinutes, true},
		{review.GoalRetention, false},
		{review.GoalDate, false},
	} {
		p := review.Defaults()
		p.Goal, p.By, p.Backlog = one.goal, time.Now().AddDate(0, 0, 30), 40
		admits := p.GetAllowance(day, time.Now(), review.Spent{}, 0, 0)

		if got := admits.Limits.Backlog != ""; got != one.shouldRead {
			t.Errorf("under %s the share is read %v, want %v", one.goal, got, one.shouldRead)
		}
		want := 40
		if !one.shouldRead {
			want = review.AllBacklog
		}
		if admits.Backlog != want {
			t.Errorf("under %s the day gives the debt %d, want %d",
				one.goal, admits.Backlog, want)
		}
	}
}

// What the day has already gone through is off what it still admits, so a
// second session takes up where the first left off.
func TestADaysSpendIsOffWhatItStillAdmits(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime}
	p := review.Defaults()
	p.Goal, p.MinutesADay = review.GoalMinutes, 1
	p.NewADay, p.ReviewsADay = 20, 20

	initialAllowance := p.GetAllowance(day, time.Now(), review.Spent{}, 0, 0)
	after := p.GetAllowance(day, time.Now(), review.Spent{
		Answered: 3, New: 3, Reviews: 3, Took: 18 * time.Second,
	}, 0, 0)

	if after.Minutes != initialAllowance.Minutes-18*time.Second {
		t.Errorf("a day of %v with 18s gone still admits %v", initialAllowance.Minutes, after.Minutes)
	}
	if after.New != initialAllowance.New-3 || after.Reviews != initialAllowance.Reviews-3 {
		t.Errorf("a day that has begun 3 and reviewed 3 still admits %d new and %d reviews",
			after.New, after.Reviews)
	}
}

// getBudget is what a preset keeps for a day of the week, read off the day
// that admits it.
func getBudget(p review.Preset, day time.Weekday) review.Budget {
	pos := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	for pos.Weekday() != day {
		pos = pos.AddDate(0, 0, 1)
	}
	return p.GetAllowance(review.Day{Starts: review.DayStartTime}, pos, review.Spent{}, 0, 0).Keeps
}

// The budget a preset keeps on one day is that day of the week's share of it,
// whether or not the days are evened out. A day the preset does not name keeps
// the whole of it, and a day at nothing keeps none.
func TestTheBudgetOfOneDayIsItsShareOfTheLoad(t *testing.T) {
	p := review.Preset{
		MinutesADay: 20, NewADay: 10, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Wednesday: 50, time.Sunday: 0},
	}

	half := getBudget(p, time.Wednesday)
	if want := (review.Budget{New: 5, Reviews: 20, Minutes: 10}); half != want {
		t.Errorf("a day at half the load holds %+v, want %+v", half, want)
	}
	whole := getBudget(p, time.Tuesday)
	if want := (review.Budget{New: 10, Reviews: 40, Minutes: 20}); whole != want {
		t.Errorf("a day the preset does not name holds %+v, want %+v", whole, want)
	}
	if none := getBudget(p, time.Sunday); none != (review.Budget{}) {
		t.Errorf("a day at none of the load holds %+v", none)
	}
}

// A day at none of the load schedules nothing, as a budget of zero does.
func TestADayAtNoneOfTheLoadIsAPause(t *testing.T) {
	p := review.Preset{
		Goal: review.GoalRetention, NewADay: 10, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Sunday: 0},
	}
	pos := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	if pos.Weekday() != time.Sunday {
		t.Fatalf("%v is a %v", pos, pos.Weekday())
	}
	if !p.GetAllowance(ahead, pos, review.Spent{}, 0, 0).IsPaused() {
		t.Error("a day at none of the load is not a pause")
	}
	if p.GetAllowance(ahead, pos.AddDate(0, 0, 1), review.Spent{}, 0, 0).IsPaused() {
		t.Error("the day after it is a pause")
	}
}

// drawAllowance is a day standing at a share of itself for the debt, drawn over
// the whole of what the share may be.
func drawAllowance(t *rapid.T) review.Allowance {
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
// Backlog is how much of a day goes to what is overdue before anything new is
// offered.
func TestTheDebtConsumesItsShareOfEveryPointOfTheRun(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		day := drawAllowance(t)
		cards := rapid.IntRange(0, 60).Draw(t, "cards")

		var debt, begun int
		for totalCards := 1; totalCards <= cards; totalCards++ {
			if day.IsNextCardDebt(debt, begun, true, true) {
				debt++
			} else {
				begun++
			}
			// The share of a whole number of cards, rounded up.
			want := (totalCards*day.Backlog + review.AllBacklog - 1) / review.AllBacklog
			if debt != want {
				t.Fatalf("a day at a share of %d took %d of its first %d cards "+
					"from the debt, want %d", day.Backlog, debt, totalCards, want)
			}
		}
	})
}

// A side that runs short leaves the rest of the day to the other, so a day is
// never left unspent: whatever the share, a day with cards on one side only
// spends all it admits on that side.
func TestASideThatRunsShortPreservesTheDayToTheOther(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		day := drawAllowance(t)
		owed := rapid.IntRange(0, 30).Draw(t, "owed")
		unbegun := rapid.IntRange(0, 30).Draw(t, "unbegun")
		admits := rapid.IntRange(0, 60).Draw(t, "admits")

		var debt, begun int
		for debt+begun < admits && (debt < owed || begun < unbegun) {
			if day.IsNextCardDebt(debt, begun, debt < owed, begun < unbegun) {
				debt++
			} else {
				begun++
			}
		}
		if got, want := debt+begun, min(admits, owed+unbegun); got != want {
			t.Fatalf("a day of %d admitting %d over %d owed and %d unbegun "+
				"spent %d of itself, want %d",
				day.Backlog, admits, owed, unbegun, got, want)
		}
		if debt > owed || begun > unbegun {
			t.Fatalf("a day took %d of %d owed and %d of %d unbegun",
				debt, owed, begun, unbegun)
		}
	})
}

// A budget counting cards charges a review card the first time the day answers
// it and every further showing of it that day is free, so what a day of any
// number of showings costs is the review cards it showed. A budget counting showings
// charges every one of them.
func TestWhatADayOfShowingsCosts(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		counts := rapid.SampledFrom([]review.BudgetUnit{
			review.BudgetUnitCards, review.BudgetUnitShows,
		}).Draw(t, "counts")
		shows := rapid.SliceOfN(rapid.IntRange(0, 5), 0, 40).Draw(t, "shows")

		reviewed := make(map[int]bool, len(shows))
		var charged int
		for _, reviewCard := range shows {
			if counts.IsCharged(reviewed[reviewCard]) {
				charged++
			}
			reviewed[reviewCard] = true
		}

		want := len(reviewed)
		if counts == review.BudgetUnitShows {
			want = len(shows)
		}
		if charged != want {
			t.Fatalf("a day counting %s spent %d on the showings %v, want %d",
				counts, charged, shows, want)
		}
	})
}
