package review_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/review"
)

func london(t *testing.T) *time.Location {
	t.Helper()
	in, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skipf("this machine carries no zone database: %v", err)
	}
	return in
}

// An hour after midnight belongs to the day before: a person answering cards at
// one in the morning is finishing that day, not starting the next.
func TestAnHourAfterMidnightBelongsToTheDayBefore(t *testing.T) {
	in := london(t)
	day := review.Day{Starts: review.DayStarts, In: in}

	late := time.Date(2026, 3, 10, 1, 30, 0, 0, in)
	if got, want := day.Ends(late), time.Date(2026, 3, 10, 4, 0, 0, 0, in); !got.Equal(want) {
		t.Errorf("the day holding %v ends at %v, want %v", late, got, want)
	}

	morning := time.Date(2026, 3, 10, 9, 0, 0, 0, in)
	if got, want := day.Ends(morning), time.Date(2026, 3, 11, 4, 0, 0, 0, in); !got.Equal(want) {
		t.Errorf("the day holding %v ends at %v, want %v", morning, got, want)
	}
}

// The boundary is an hour of the clock on the wall, so the day an hour is taken
// out of ends at the hour a person reads.
func TestTheDayAnHourIsTakenOutOfEndsAtTheHourOnTheWall(t *testing.T) {
	in := london(t)
	day := review.Day{Starts: review.DayStarts, In: in}

	// The clocks go forward at one in the morning on 29 March 2026.
	before := time.Date(2026, 3, 28, 9, 0, 0, 0, in)
	if got, want := day.Ends(before), time.Date(2026, 3, 29, 4, 0, 0, 0, in); !got.Equal(want) {
		t.Errorf("ends at %v, want %v", got, want)
	}
}

// A card due later today is owed now: a person sits down when they sit down,
// and a card owed today is owed for the whole of it.
func TestACardDueLaterTodayIsOwedNow(t *testing.T) {
	in := london(t)
	day := review.Day{Starts: review.DayStarts, In: in}
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, in)

	for name, one := range map[string]struct {
		due  time.Time
		owed bool
	}{
		"days ago":             {now.Add(-72 * time.Hour), true},
		"this morning":         {now.Add(-time.Hour), true},
		"this evening":         {now.Add(8 * time.Hour), true},
		"after the day is out": {time.Date(2026, 3, 11, 5, 0, 0, 0, in), false},
		"next week":            {now.Add(7 * 24 * time.Hour), false},
	} {
		t.Run(name, func(t *testing.T) {
			s := review.Schedule{Due: one.due, Last: now.Add(-24 * time.Hour)}
			if got := day.Owed(s, now); got != one.owed {
				t.Errorf("a card due %v is owed = %v, want %v", one.due, got, one.owed)
			}
		})
	}
}

// A card nobody has answered is owed the first time it is asked about.
func TestACardNobodyAnsweredIsOwed(t *testing.T) {
	day := review.Day{Starts: review.DayStarts}
	if !day.Owed(review.Schedule{}, time.Now()) {
		t.Error("a card with no answers behind it is not owed")
	}
}

// A build holding no zone counts the day in the machine's own.
func TestADayWithNoZoneIsCountedInTheMachinesOwn(t *testing.T) {
	day := review.Day{Starts: review.DayStarts}
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.Local)
	if got := day.Ends(now); got.Location() != time.Local {
		t.Errorf("counted in %v, want the machine's own", got.Location())
	}
}
