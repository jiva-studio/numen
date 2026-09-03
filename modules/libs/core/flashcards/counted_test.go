package flashcards_test

import (
	"fmt"
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
		said("01E", flashcards.Hard),
	})

	want := flashcards.Tally{Answered: 5, Again: 1, Hard: 1, Good: 2, Easy: 1}
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

// What a day came to is counted under the preset each card face is grouped
// under: the answers given, and the time they took.
func TestWhatADayCameToUnderEachPreset(t *testing.T) {
	root := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	mantra := flashcards.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}
	loose := flashcards.CardFace{Card: "3f4g5h6j7k", Face: "Say it"}
	under := map[flashcards.CardFace]string{root: "Sanskrit.md", mantra: "Sanskrit.md"}

	said := func(id string, face flashcards.CardFace, at string, took time.Duration) flashcards.Answer {
		return flashcards.Answer{
			ID: id, CardFace: face, At: moment(t, at), Rating: flashcards.Good, Took: took,
		}
	}
	answers := []flashcards.Answer{
		said("01A", root, "2026-08-29T09:00:00", 6*time.Second),
		said("01B", mantra, "2026-08-29T21:00:00", 9*time.Second),
		// The night belongs to the evening it began in.
		said("01C", root, "2026-08-30T02:00:00", 5*time.Second),
		// A card face nothing groups, an answer taken back, and another day.
		said("01D", loose, "2026-08-29T09:30:00", 8*time.Second),
		said("01E", root, "2026-08-29T10:00:00", 7*time.Second),
		{ID: "01F", At: moment(t, "2026-08-29T10:01:00"), Undoes: "01E"},
		said("01G", root, "2026-08-31T09:00:00", 4*time.Second),
	}

	got := flashcards.Sat(counting, "2026-08-29", answers, under, nil)

	// Each card face is new the first time it is answered, and counts once for
	// the day however many answers it took.
	want := flashcards.Spent{Answered: 2, New: 2, Took: 20 * time.Second}
	if got["Sanskrit.md"] != want {
		t.Errorf("the day came to %+v, want %+v", got["Sanskrit.md"], want)
	}
	if len(got) != 1 {
		t.Errorf("a card face nothing groups was counted: %+v", got)
	}
}

// A preset counting in cards counts a card rated again as the one card, and a
// preset counting in shows counts each time it was put to the person. The time
// spent is the same time either way.
func TestACardAnsweredAgainInTheDayIsCountedBothWays(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	under := map[flashcards.CardFace]string{on: "Steady.md"}

	answers := make([]flashcards.Answer, 0, 9)
	for i := range 9 {
		answers = append(answers, flashcards.Answer{
			ID:       fmt.Sprintf("01%d", i),
			CardFace: on,
			At:       moment(t, "2026-08-29T09:00:00").Add(time.Duration(i) * time.Minute),
			Rating:   flashcards.Again,
			Took:     4 * time.Second,
		})
	}

	cards := flashcards.Sat(counting, "2026-08-29", answers, under,
		map[string]flashcards.Counts{"Steady.md": flashcards.CountsCards})
	want := flashcards.Spent{Answered: 1, New: 1, Took: 36 * time.Second}
	if cards["Steady.md"] != want {
		t.Errorf("counting in cards the day came to %+v, want %+v", cards["Steady.md"], want)
	}

	shows := flashcards.Sat(counting, "2026-08-29", answers, under,
		map[string]flashcards.Counts{"Steady.md": flashcards.CountsShows})
	want = flashcards.Spent{Answered: 9, New: 1, Reviews: 8, Took: 36 * time.Second}
	if shows["Steady.md"] != want {
		t.Errorf("counting in shows the day came to %+v, want %+v", shows["Steady.md"], want)
	}
}

// A card face is new on the day of its earliest answer.
//
// The files arrive in whatever order they were synchronised, and a run from
// another machine sorting last by name can carry the answer that came first.
func TestTheFirstAnswerOfACardFaceIsTheEarliestOne(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	under := map[flashcards.CardFace]string{on: "Steady.md"}

	answers := []flashcards.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-31T09:00:00"),
			Rating: flashcards.Good, Took: 5 * time.Second},
		{ID: "01Z", CardFace: on, At: moment(t, "2026-08-24T09:00:00"),
			Rating: flashcards.Good, Took: 7 * time.Second},
	}

	got := flashcards.Sat(counting, "2026-08-31", answers, under, nil)

	want := flashcards.Spent{Answered: 1, Reviews: 1, Took: 5 * time.Second}
	if got["Steady.md"] != want {
		t.Errorf("the day came to %+v, want %+v", got["Steady.md"], want)
	}
}

// One answer counts an hour of it at most, whatever the card stood on the
// screen for.
func TestALongAnswerIsCountedAtItsBound(t *testing.T) {
	on := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	under := map[flashcards.CardFace]string{on: ""}

	got := flashcards.Sat(counting, "2026-08-29", []flashcards.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-29T09:00:00"),
			Rating: flashcards.Good, Took: time.Hour},
	}, under, nil)

	want := flashcards.Spent{Answered: 1, New: 1, Took: flashcards.LongestAnswer}
	if got[""] != want {
		t.Errorf("the day came to %+v, want %+v", got[""], want)
	}
}

// The card faces a day has answered are the day's however late it ran. A card
// answered at one in the morning was answered that evening, and a budget
// counting in cards has already charged it.
func TestTheFacesADayAnsweredAreCountedByTheDayTheyFallIn(t *testing.T) {
	evening := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	night := flashcards.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}
	morning := flashcards.CardFace{Card: "3f4g5h6j7k", Face: "Recognise"}
	back := flashcards.CardFace{Card: "m9n8b7v6c5", Face: "Recognise"}

	got := flashcards.Faced(counting, "2026-08-29", []flashcards.Answer{
		{ID: "01A", CardFace: evening, At: moment(t, "2026-08-29T21:00:00"), Rating: flashcards.Good},
		{ID: "01B", CardFace: night, At: moment(t, "2026-08-30T02:00:00"), Rating: flashcards.Good},
		{ID: "01C", CardFace: morning, At: moment(t, "2026-08-30T09:00:00"), Rating: flashcards.Good},
		{ID: "01D", CardFace: back, At: moment(t, "2026-08-29T22:00:00"), Rating: flashcards.Good},
		{ID: "01E", At: moment(t, "2026-08-29T22:01:00"), Undoes: "01D"},
	})

	if !got[evening] {
		t.Error("a card answered in the evening is not one the day answered")
	}
	if !got[night] {
		t.Error("a card answered at two in the morning is not that evening's")
	}
	if got[morning] {
		t.Error("a card answered the next morning is counted in the day before")
	}
	if got[back] {
		t.Error("an answer taken back left the card among the day's")
	}
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

// A day counted in no zone in particular is counted in the machine's own,
// which is where a person's evening is.
func TestADayWithNoZoneIsTheMachinesOwn(t *testing.T) {
	here := flashcards.Day{Starts: flashcards.DayStarts}
	now := time.Now()
	days := answeredOn(map[string]int{
		here.Names(now):                   1,
		here.Names(now.AddDate(0, 0, -1)): 1,
	})

	if got := flashcards.Streak(here, days, now); got != 2 {
		t.Errorf("the streak is %d, want the two days answered", got)
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
	if want := (flashcards.RecallTally{Asked: 1, Recalled: 1}); got["2026-08-08"] != want {
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

	if want := (flashcards.RecallTally{Asked: 1, Recalled: 0}); got["2026-08-08"] != want {
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
