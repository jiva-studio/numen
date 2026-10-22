package review_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
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
	day := review.Day{Starts: review.DayStartTime, In: in}

	late := time.Date(2026, 3, 10, 1, 30, 0, 0, in)
	if got, want := day.GetEnd(late), time.Date(2026, 3, 10, 4, 0, 0, 0, in); !got.Equal(want) {
		t.Errorf("the day holding %v ends at %v, want %v", late, got, want)
	}

	morning := time.Date(2026, 3, 10, 9, 0, 0, 0, in)
	if got, want := day.GetEnd(morning), time.Date(2026, 3, 11, 4, 0, 0, 0, in); !got.Equal(want) {
		t.Errorf("the day holding %v ends at %v, want %v", morning, got, want)
	}
}

// The boundary is an hour of the clock on the wall, so the day an hour is taken
// out of ends at the hour a person reads.
func TestTheDayAnHourIsTakenOutOfEndsAtTheHourOnTheWall(t *testing.T) {
	in := london(t)
	day := review.Day{Starts: review.DayStartTime, In: in}

	// The clocks go forward at one in the morning on 29 March 2026.
	before := time.Date(2026, 3, 28, 9, 0, 0, 0, in)
	if got, want := day.GetEnd(before), time.Date(2026, 3, 29, 4, 0, 0, 0, in); !got.Equal(want) {
		t.Errorf("ends at %v, want %v", got, want)
	}
}

// A card due later today is owed now: a person sits down when they sit down,
// and a card owed today is owed for the whole of it.
func TestACardDueLaterTodayIsOwedNow(t *testing.T) {
	in := london(t)
	day := review.Day{Starts: review.DayStartTime, In: in}
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, in)

	for name, one := range map[string]struct {
		due    time.Time
		isOwed bool
	}{
		"days ago":             {now.Add(-72 * time.Hour), true},
		"this morning":         {now.Add(-time.Hour), true},
		"this evening":         {now.Add(8 * time.Hour), true},
		"after the day is out": {time.Date(2026, 3, 11, 5, 0, 0, 0, in), false},
		"next week":            {now.Add(7 * 24 * time.Hour), false},
	} {
		t.Run(name, func(t *testing.T) {
			s := review.Schedule{Due: one.due, Last: now.Add(-24 * time.Hour)}
			if got := day.IsOwed(s, now); got != one.isOwed {
				t.Errorf("a card due %v is owed = %v, want %v", one.due, got, one.isOwed)
			}
		})
	}
}

// A card nobody has answered is owed the first time it is asked about.
func TestACardNobodyAnsweredIsOwed(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime}
	if !day.IsOwed(review.Schedule{}, time.Now()) {
		t.Error("a card with no answers behind it is not owed")
	}
}

// A build holding no zone counts the day in the machine's own.
func TestADayWithNoZoneIsCountedInTheMachinesOwn(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime}
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.Local)
	if got := day.GetEnd(now); got.Location() != time.Local {
		t.Errorf("counted in %v, want the machine's own", got.Location())
	}
}

// zones is the spread of zones the days are counted over, taking in a clock
// that goes forward at midnight, one that moves by half an hour, and one that
// crossed the date line.
var zones = []string{
	"America/Santiago", "America/Havana", "Australia/Lord_Howe", "Pacific/Chatham",
	"Pacific/Apia", "America/Sao_Paulo", "America/New_York", "Europe/London",
	"Europe/Moscow", "Asia/Tehran", "Australia/Sydney",
}

// A day of review is named for its own date. A day beginning at midnight where
// the clock goes forward at midnight has no boundary to count back from.
func TestADayOfReviewIsNamedForItsOwnDate(t *testing.T) {
	for _, name := range zones {
		in, err := time.LoadLocation(name)
		if err != nil {
			t.Skipf("this machine carries no zone database: %v", err)
		}
		for hour := range 24 {
			day := review.Day{Starts: time.Duration(hour) * time.Hour, In: in}
			from := time.Date(2025, 1, 1, 0, 0, 0, 0, in)
			for i := range 800 * 24 {
				at := from.Add(time.Duration(i) * time.Hour).In(in)
				y, m, d := at.Date()
				want := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
				if at.Hour() < hour {
					want = want.AddDate(0, 0, -1)
				}
				if got := day.GetName(at); got != want.Format(review.DayLayout) {
					t.Fatalf("in %s, %v under a day beginning at %02d:00 is named %s, want %s",
						name, at, hour, got, want.Format(review.DayLayout))
				}
			}
		}
	}
}

// Each day of review carries away the card held on it: no two of them share a
// number, and none of them is passed over.
func TestEachDayOfReviewIsNumberedApartFromTheNext(t *testing.T) {
	for _, name := range zones {
		in, err := time.LoadLocation(name)
		if err != nil {
			t.Skipf("this machine carries no zone database: %v", err)
		}
		for hour := range 24 {
			day := review.Day{Starts: time.Duration(hour) * time.Hour, In: in}
			on := review.NewDueByDay(day)
			days := make([]time.Time, 0, 800)
			for at := time.Date(2025, 1, 1, 12, 0, 0, 0, in); len(days) < 800; at = day.GetEnd(at) {
				days = append(days, at)
				if next := day.GetEnd(at); !next.After(at) {
					t.Fatalf("in %s, the day holding %v under a day beginning at %02d:00 "+
						"ends at %v", name, at, hour, next)
				}
				on.Add(at)
			}
			for _, at := range days {
				if got := on.CountOn(at); got != 1 {
					t.Fatalf("in %s, the day holding %v under a day beginning at %02d:00 "+
						"carries %d of the 800 cards, want 1", name, at, hour, got)
				}
			}
		}
	}
}

// The hour a day of review begins at is read as hours and minutes of the clock on the wall.
func TestTheHourADayBeginsAtIsReadAsTheClockOnTheWall(t *testing.T) {
	for content, want := range map[string]time.Duration{
		"":      review.DayStartTime,
		"  ":    review.DayStartTime,
		"03:30": 3*time.Hour + 30*time.Minute,
		"00:00": 0,
		"12:00": 12 * time.Hour,
	} {
		got, err := review.ParseDayStart(content)
		if err != nil || got != want {
			t.Errorf("%q: got %v (%v), want %v", content, got, err, want)
		}
	}
}

// Something that is not an hour of the day, or lies past the latest hour a day may begin at, is
// refused, and the default stands with the refusal.
func TestAnHourTheDayDoesNotBeginAt(t *testing.T) {
	for _, content := range []string{"half past four", "18:00", "12:01", "4", "24:00"} {
		got, err := review.ParseDayStart(content)
		if !errors.Is(err, review.ErrNotAnHour) || got != review.DayStartTime {
			t.Errorf("%q: got %v (%v)", content, got, err)
		}
	}
}
