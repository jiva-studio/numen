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

	if got["2026-08-29"].Answered != 3 {
		t.Errorf("the evening and the night after it come to %+v, want 3", got["2026-08-29"])
	}
	if _, held := got["2026-08-31"]; held {
		t.Errorf("a day whose only answer was taken back is counted: %v", got)
	}
	if len(got) != 1 {
		t.Errorf("counted %v", got)
	}
}

// A day says how each of the four was answered on it, because fifty cards a
// person could not recall is a different day from fifty they could.
func TestADaySaysHowEachOfTheFourWasAnswered(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	said := func(id string, r flashcards.Rating) flashcards.Answer {
		return flashcards.Answer{
			ID: id, CardFace: on, At: moment(t, "2026-08-29T09:00:00"), Rating: r,
		}
	}

	got := flashcards.Counted(counting, []flashcards.Answer{
		said("01A", flashcards.Again),
		said("01B", flashcards.Good),
		said("01C", flashcards.Good),
		said("01D", flashcards.Easy),
	})

	want := flashcards.Tally{Answered: 4, Again: 1, Good: 2, Easy: 1}
	if got["2026-08-29"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-29"], want)
	}
}

// answeredOn is days a person answered on, as many as each says.
func answeredOn(days map[string]int) map[string]flashcards.Tally {
	out := make(map[string]flashcards.Tally, len(days))
	for day, answered := range days {
		out[day] = flashcards.Tally{Answered: answered, Good: answered}
	}
	return out
}

// A streak is the days up to now with no gap in them.
func TestAStreakIsTheDaysUpToNowWithNoGap(t *testing.T) {
	days := answeredOn(map[string]int{
		"2026-08-25": 4,
		// the 26th is a day nobody answered on
		"2026-08-27": 2,
		"2026-08-28": 9,
		"2026-08-29": 1,
	})
	if got := flashcards.Streak(counting, days, moment(t, "2026-08-29T20:00:00")); got != 3 {
		t.Errorf("the streak is %d, want the three days since the gap", got)
	}
}

// A day nobody has answered on yet does not end a streak. A person who has not
// sat down this morning has not broken anything.
func TestADayNotAnsweredOnYetDoesNotEndAStreak(t *testing.T) {
	days := answeredOn(map[string]int{"2026-08-27": 2, "2026-08-28": 9})

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
	days := answeredOn(map[string]int{"2026-08-28": 3, "2026-08-29": 5})
	if got := flashcards.Streak(counting, days, moment(t, "2026-08-30T02:00:00")); got != 2 {
		t.Errorf("the streak is %d at two in the morning, want 2", got)
	}
}

func TestNothingAnsweredIsNoStreak(t *testing.T) {
	if got := flashcards.Streak(counting, nil, moment(t, "2026-08-29T09:00:00")); got != 0 {
		t.Errorf("the streak is %d", got)
	}
}

// What came back is counted over the cards a person had learned, and only
// those: a card still being learned is asked whether it comes back after ten
// minutes, which says nothing about how well anything is remembered.
func TestWhatCameBackIsCountedOverWhatWasLearned(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	other := flashcards.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}
	by := flashcards.NewFSRS()

	var history []flashcards.Answer
	said := func(id string, face flashcards.CardFace, at string, r flashcards.Rating) {
		history = append(history, flashcards.Answer{
			ID: id, CardFace: face, At: moment(t, at), Rating: r,
		})
	}

	// One card learned on the first day, and asked again a week later.
	said("01A", on, "2026-08-01T09:00:00", flashcards.Easy)
	said("01B", on, "2026-08-08T09:00:00", flashcards.Good)
	// Another card met for the first time on that second day.
	said("01C", other, "2026-08-08T09:05:00", flashcards.Again)

	got := flashcards.Retained(by, counting, history)

	// The first day is a card being learned, so nothing was tested on it.
	if _, held := got["2026-08-01"]; held {
		t.Errorf("a card being learned was counted: %+v", got["2026-08-01"])
	}
	// The second is one learned card asked and recalled, and one being met.
	if want := (flashcards.Retention{Asked: 1, Recalled: 1}); got["2026-08-08"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-08"], want)
	}
}

// A card a person had learned and could not recall is asked and not recalled,
// which is what the number is for.
func TestACardForgottenIsAskedAndNotRecalled(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	by := flashcards.NewFSRS()

	got := flashcards.Retained(by, counting, []flashcards.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-01T09:00:00"), Rating: flashcards.Easy},
		{ID: "01B", CardFace: on, At: moment(t, "2026-08-08T09:00:00"), Rating: flashcards.Again},
	})

	if want := (flashcards.Retention{Asked: 1, Recalled: 0}); got["2026-08-08"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-08"], want)
	}
}

// An answer taken back is not in it, as it is in nothing else.
func TestAnAnswerTakenBackIsNotCountedAsRecall(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	by := flashcards.NewFSRS()

	got := flashcards.Retained(by, counting, []flashcards.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-01T09:00:00"), Rating: flashcards.Easy},
		{ID: "01B", CardFace: on, At: moment(t, "2026-08-08T09:00:00"), Rating: flashcards.Again},
		{ID: "01C", At: moment(t, "2026-08-08T09:00:30"), Undoes: "01B"},
	})

	if _, held := got["2026-08-08"]; held {
		t.Errorf("an answer taken back was counted: %+v", got["2026-08-08"])
	}
}
