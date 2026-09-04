package review_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// counted is a table of the days of review, loaded with as many card faces on
// each of these many days past an instant.
func counted(d review.Day, at time.Time, on map[int]int) *review.DueByDay {
	out := review.Spreading(d)
	for day, cards := range on {
		for range cards {
			out.Holds(at.AddDate(0, 0, day))
		}
	}
	return out
}

// A card goes on the heaviest day of the window around the interval it was sent
// away for: the day whose share of the load stands highest over what already
// falls on it. A day at none of the load weighs nothing and takes no card.
func TestACardGoesOnTheHeaviestDayOfItsWindow(t *testing.T) {
	day := review.Day{Starts: review.DayStarts, In: time.UTC}
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	if at.Weekday() != time.Sunday {
		t.Fatalf("%s is a %s", at.Format(review.Named), at.Weekday())
	}
	// Seven days away, so the window runs from five days off to nine: the
	// Friday, the Saturday, the Sunday it was sent to, the Monday and the
	// Tuesday.
	due := at.AddDate(0, 0, 7)
	p := review.Preset{
		Goal: review.GoalMinutes, MinutesADay: 20, EvenLoad: true,
		Load: map[time.Weekday]int{time.Friday: 50, time.Saturday: 0},
	}

	// The Tuesday carries the whole load and nothing yet, which is more than
	// the half day on the Friday, the three already on the Sunday and the one
	// on the Monday.
	on := counted(day, at, map[int]int{5: 0, 6: 0, 7: 3, 8: 1, 9: 0})
	if got, want := p.Places(on, at, due), at.AddDate(0, 0, 9); !got.Equal(want) {
		t.Errorf("the card was put on %s, want %s",
			got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	// And the day it landed on is counted against that day.
	if got := on.On(at.AddDate(0, 0, 9)); got != 1 {
		t.Errorf("the day the card landed on carries %d card faces, want 1", got)
	}

	// The day the scheduler named weighs as much as the Monday beside it, and
	// keeps the card.
	tied := counted(day, at, map[int]int{5: 3, 6: 0, 7: 0, 8: 0, 9: 1})
	if got := p.Places(tied, at, due); !got.Equal(due) {
		t.Errorf("a card tied between its own day and the next was put on %s, want %s",
			got.Format(time.RFC3339), due.Format(time.RFC3339))
	}
}

// A table of the days of review says how many card faces fall on the day
// holding an instant.
func TestHowLoadedADayOfReviewIs(t *testing.T) {
	day := review.Day{Starts: review.DayStarts, In: time.UTC}
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	on := counted(day, at, map[int]int{0: 2, 1: 1})

	if got := on.On(at.Add(6 * time.Hour)); got != 2 {
		t.Errorf("the day holding the two card faces carries %d, want 2", got)
	}
	// A day of review runs to the hour it opens at, so an instant before that
	// hour belongs to the day before it.
	if got := on.On(at.AddDate(0, 0, 1).Add(-8 * time.Hour)); got != 2 {
		t.Errorf("the small hours carry %d card faces, want the 2 of the day before", got)
	}
	if got := on.On(at.AddDate(0, 0, 2)); got != 0 {
		t.Errorf("a day nothing falls on carries %d card faces", got)
	}
}
