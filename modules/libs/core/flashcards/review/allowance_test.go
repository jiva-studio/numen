package review_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// The share of the day that goes to the debt is read where one pot is spent
// between the two, and the goal says so where it says which budget closes its
// day.
//
// A goal of retention keeps a count for each side, so each is held to its own
// and the share decides nothing. A goal of a date carries the whole material by
// its own reckoning.
func TestWhichSettingsAGoalReads(t *testing.T) {
	day := review.Day{Starts: review.DayStarts}
	for _, one := range []struct {
		goal  review.Goal
		reads bool
	}{
		{review.GoalMinutes, true},
		{review.GoalRetention, false},
		{review.GoalDate, false},
	} {
		p := review.Defaults()
		p.Goal, p.By, p.Backlog = one.goal, time.Now().AddDate(0, 0, 30), 40
		admits := p.Admits(day, time.Now(), review.Spent{}, 0, 0)

		if got := admits.Limits.Backlog != ""; got != one.reads {
			t.Errorf("under %s the share is read %v, want %v", one.goal, got, one.reads)
		}
		want := 40
		if !one.reads {
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
	day := review.Day{Starts: review.DayStarts}
	p := review.Defaults()
	p.Goal, p.MinutesADay = review.GoalMinutes, 1
	p.NewADay, p.ReviewsADay = 20, 20

	fresh := p.Admits(day, time.Now(), review.Spent{}, 0, 0)
	after := p.Admits(day, time.Now(), review.Spent{
		Answered: 3, New: 3, Reviews: 3, Took: 18 * time.Second,
	}, 0, 0)

	if after.Minutes != fresh.Minutes-18*time.Second {
		t.Errorf("a day of %v with 18s gone still admits %v", fresh.Minutes, after.Minutes)
	}
	if after.New != fresh.New-3 || after.Reviews != fresh.Reviews-3 {
		t.Errorf("a day that has begun 3 and reviewed 3 still admits %d new and %d reviews",
			after.New, after.Reviews)
	}
}

// keeps is what a preset keeps for a day of the week, read off the day that
// admits it.
func keeps(p review.Preset, day time.Weekday) review.Budget {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	for at.Weekday() != day {
		at = at.AddDate(0, 0, 1)
	}
	return p.Admits(review.Day{Starts: review.DayStarts}, at, review.Spent{}, 0, 0).Keeps
}

// The budget a preset keeps on one day is that day of the week's share of it,
// whether or not the days are evened out. A day the preset does not name keeps
// the whole of it, and a day at nothing keeps none.
func TestTheBudgetOfOneDayIsItsShareOfTheLoad(t *testing.T) {
	p := review.Preset{
		MinutesADay: 20, NewADay: 10, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Wednesday: 50, time.Sunday: 0},
	}

	half := keeps(p, time.Wednesday)
	if want := (review.Budget{New: 5, Reviews: 20, Minutes: 10}); half != want {
		t.Errorf("a day at half the load holds %+v, want %+v", half, want)
	}
	whole := keeps(p, time.Tuesday)
	if want := (review.Budget{New: 10, Reviews: 40, Minutes: 20}); whole != want {
		t.Errorf("a day the preset does not name holds %+v, want %+v", whole, want)
	}
	if none := keeps(p, time.Sunday); none != (review.Budget{}) {
		t.Errorf("a day at none of the load holds %+v", none)
	}
}

// A day at none of the load schedules nothing, as a budget of zero does.
func TestADayAtNoneOfTheLoadIsAPause(t *testing.T) {
	p := review.Preset{
		Goal: review.GoalRetention, NewADay: 10, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Sunday: 0},
	}
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	if at.Weekday() != time.Sunday {
		t.Fatalf("%v is a %v", at, at.Weekday())
	}
	if !p.Admits(ahead, at, review.Spent{}, 0, 0).IsPaused() {
		t.Error("a day at none of the load is not a pause")
	}
	if p.Admits(ahead, at.AddDate(0, 0, 1), review.Spent{}, 0, 0).IsPaused() {
		t.Error("the day after it is a pause")
	}
}
