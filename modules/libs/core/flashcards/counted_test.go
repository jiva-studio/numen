package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// counting is a day as the rest of the application counts one, in a zone a test
// can say things about.
var counting = flashcards.Day{
	Starts: flashcards.DayStarts,
	In:     time.FixedZone("test", 0),
}

func moment(t *testing.T, said string) time.Time {
	t.Helper()
	at, err := time.ParseInLocation("2006-01-02T15:04:05", said, counting.In)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

// A day is named for the day a person would say they answered on. The night
// belongs to the day it began in: cards answered at one in the morning are that
// evening's work.
func TestADayIsNamedForTheDayAPersonWouldSayItWas(t *testing.T) {
	for _, one := range []struct{ at, want string }{
		{"2026-08-29T09:00:00", "2026-08-29"},
		{"2026-08-29T23:59:00", "2026-08-29"},
		{"2026-08-30T01:00:00", "2026-08-29"},
		{"2026-08-30T03:59:00", "2026-08-29"},
		{"2026-08-30T04:00:00", "2026-08-30"},
	} {
		if got := counting.Names(moment(t, one.at)); got != one.want {
			t.Errorf("%s is named %q, want %q", one.at, got, one.want)
		}
	}
}

// What was answered on a day is a sum: an answer taken back is not in it, a
// line that stands twice is counted once, and nothing depends on the order the
// answers were read in.
func TestWhatWasAnsweredOnADayIsCounted(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	one := flashcards.Answer{ID: "01A", CardFace: on, At: moment(t, "2026-08-29T09:00:00"), Rating: flashcards.Good}
	two := flashcards.Answer{ID: "01B", CardFace: on, At: moment(t, "2026-08-29T21:00:00"), Rating: flashcards.Again}
	night := flashcards.Answer{ID: "01C", CardFace: on, At: moment(t, "2026-08-30T02:00:00"), Rating: flashcards.Good}
	other := flashcards.Answer{ID: "01D", CardFace: on, At: moment(t, "2026-08-31T09:00:00"), Rating: flashcards.Good}
	back := flashcards.Answer{ID: "01E", At: moment(t, "2026-08-31T09:01:00"), Undoes: other.ID}

	got := flashcards.Counted(counting, []flashcards.Answer{one, two, night, other, back, one})

	if got["2026-08-29"] != 3 {
		t.Errorf("the evening and the night after it come to %d, want 3", got["2026-08-29"])
	}
	if _, held := got["2026-08-31"]; held {
		t.Errorf("a day whose only answer was taken back is counted: %v", got)
	}
	if len(got) != 1 {
		t.Errorf("counted %v", got)
	}
}

// A streak is the days up to now with no gap in them.
func TestAStreakIsTheDaysUpToNowWithNoGap(t *testing.T) {
	days := map[string]int{
		"2026-08-25": 4,
		// the 26th is a day nobody answered on
		"2026-08-27": 2,
		"2026-08-28": 9,
		"2026-08-29": 1,
	}
	if got := flashcards.Streak(counting, days, moment(t, "2026-08-29T20:00:00")); got != 3 {
		t.Errorf("the streak is %d, want the three days since the gap", got)
	}
}

// A day nobody has answered on yet does not end a streak. A person who has not
// sat down this morning has not broken anything.
func TestADayNotAnsweredOnYetDoesNotEndAStreak(t *testing.T) {
	days := map[string]int{"2026-08-27": 2, "2026-08-28": 9}

	if got := flashcards.Streak(counting, days, moment(t, "2026-08-29T09:00:00")); got != 2 {
		t.Errorf("the streak is %d in the morning, want the two days behind it", got)
	}
	// And the day after that, the streak is over.
	if got := flashcards.Streak(counting, days, moment(t, "2026-08-30T09:00:00")); got != 0 {
		t.Errorf("the streak is %d a day later, want nothing", got)
	}
}

// A streak counted in the small hours is the evening's, because the night
// belongs to the day it began in.
func TestAStreakInTheSmallHoursIsTheEveningsStill(t *testing.T) {
	days := map[string]int{"2026-08-28": 3, "2026-08-29": 5}
	if got := flashcards.Streak(counting, days, moment(t, "2026-08-30T02:00:00")); got != 2 {
		t.Errorf("the streak is %d at two in the morning, want 2", got)
	}
}

func TestNothingAnsweredIsNoStreak(t *testing.T) {
	if got := flashcards.Streak(counting, map[string]int{}, moment(t, "2026-08-29T09:00:00")); got != 0 {
		t.Errorf("the streak is %d", got)
	}
}
