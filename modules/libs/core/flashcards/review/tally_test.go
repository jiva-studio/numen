package review_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// counting is a day as the rest of the application counts one, in a zone a test
// can say things about.
var counting = review.Day{
	Starts: review.DayStarts,
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
		if got := counting.GetName(moment(t, one.at)); got != one.want {
			t.Errorf("%s is named %q, want %q", one.at, got, one.want)
		}
	}
}

// What was answered on a day is a sum: an answer taken back is not in it, a
// line that stands twice is counted once, and nothing depends on the order the
// answers were read in.
func TestWhatWasAnsweredOnADayIsCounted(t *testing.T) {
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	one := review.Answer{ID: "01A", CardFace: on, At: moment(t, "2026-08-29T09:00:00"), Rating: review.Good}
	two := review.Answer{ID: "01B", CardFace: on, At: moment(t, "2026-08-29T21:00:00"), Rating: review.Again}
	night := review.Answer{ID: "01C", CardFace: on, At: moment(t, "2026-08-30T02:00:00"), Rating: review.Good}
	other := review.Answer{ID: "01D", CardFace: on, At: moment(t, "2026-08-31T09:00:00"), Rating: review.Good}
	back := review.Answer{ID: "01E", At: moment(t, "2026-08-31T09:01:00"), Undoes: other.ID}

	got := review.GetDayTallies(counting, []review.Answer{one, two, night, other, back, one})

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
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	said := func(id string, r review.Rating) review.Answer {
		return review.Answer{
			ID: id, CardFace: on, At: moment(t, "2026-08-29T09:00:00"), Rating: r,
		}
	}

	got := review.GetDayTallies(counting, []review.Answer{
		said("01A", review.Again),
		said("01B", review.Good),
		said("01C", review.Good),
		said("01D", review.Easy),
		said("01E", review.Hard),
	})

	want := review.Tally{Answered: 5, Again: 1, Hard: 1, Good: 2, Easy: 1}
	if got["2026-08-29"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-29"], want)
	}
}

// makeTallies is days a person answered on, as many as each says.
func makeTallies(days map[string]int) map[string]review.Tally {
	out := make(map[string]review.Tally, len(days))
	for day, answered := range days {
		out[day] = review.Tally{Answered: answered, Good: answered}
	}
	return out
}

// What a day came to is counted under the preset each card face is grouped
// under: the answers given, and the time they took.
func TestWhatADayCameToUnderEachPreset(t *testing.T) {
	root := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	mantra := review.CardFaceID{Card: "zpqrstvwxy", Face: "Recognise"}
	loose := review.CardFaceID{Card: "3f4g5h6j7k", Face: "Say it"}
	under := map[review.CardFaceID]string{root: "Sanskrit.md", mantra: "Sanskrit.md"}

	said := func(id string, face review.CardFaceID, at string, took time.Duration) review.Answer {
		return review.Answer{
			ID: id, CardFace: face, At: moment(t, at), Rating: review.Good, Took: took,
		}
	}
	answers := []review.Answer{
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

	got := review.GetSpentUnder(counting, "2026-08-29", answers, under, nil)

	// Each card face is new the first time it is answered, and counts once for
	// the day however many answers it took.
	want := review.Spent{Answered: 2, New: 2, Took: 20 * time.Second}
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
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	under := map[review.CardFaceID]string{on: "Steady.md"}

	answers := make([]review.Answer, 0, 9)
	for i := range 9 {
		answers = append(answers, review.Answer{
			ID:       fmt.Sprintf("01%d", i),
			CardFace: on,
			At:       moment(t, "2026-08-29T09:00:00").Add(time.Duration(i) * time.Minute),
			Rating:   review.Again,
			Took:     4 * time.Second,
		})
	}

	cards := review.GetSpentUnder(counting, "2026-08-29", answers, under,
		map[string]review.BudgetUnit{"Steady.md": review.BudgetUnitCards})
	want := review.Spent{Answered: 1, New: 1, Took: 36 * time.Second}
	if cards["Steady.md"] != want {
		t.Errorf("counting in cards the day came to %+v, want %+v", cards["Steady.md"], want)
	}

	shows := review.GetSpentUnder(counting, "2026-08-29", answers, under,
		map[string]review.BudgetUnit{"Steady.md": review.BudgetUnitShows})
	want = review.Spent{Answered: 9, New: 1, Reviews: 8, Took: 36 * time.Second}
	if shows["Steady.md"] != want {
		t.Errorf("counting in shows the day came to %+v, want %+v", shows["Steady.md"], want)
	}
}

// A card face is new on the day of its earliest answer.
//
// The files arrive in whatever order they were synchronised, and a run from
// another machine sorting last by name can carry the answer that came first.
func TestTheFirstAnswerOfACardFaceIsTheEarliestOne(t *testing.T) {
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	under := map[review.CardFaceID]string{on: "Steady.md"}

	answers := []review.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-31T09:00:00"),
			Rating: review.Good, Took: 5 * time.Second},
		{ID: "01Z", CardFace: on, At: moment(t, "2026-08-24T09:00:00"),
			Rating: review.Good, Took: 7 * time.Second},
	}

	got := review.GetSpentUnder(counting, "2026-08-31", answers, under, nil)

	want := review.Spent{Answered: 1, Reviews: 1, Took: 5 * time.Second}
	if got["Steady.md"] != want {
		t.Errorf("the day came to %+v, want %+v", got["Steady.md"], want)
	}
}

// One answer counts an hour of it at most, whatever the card stood on the
// screen for.
func TestALongAnswerIsCountedAtItsBound(t *testing.T) {
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	under := map[review.CardFaceID]string{on: ""}

	got := review.GetSpentUnder(counting, "2026-08-29", []review.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-29T09:00:00"),
			Rating: review.Good, Took: time.Hour},
	}, under, nil)

	want := review.Spent{Answered: 1, New: 1, Took: review.LongestAnswer}
	if got[""] != want {
		t.Errorf("the day came to %+v, want %+v", got[""], want)
	}
}

// The card faces a day has answered are the day's however late it ran. A card
// answered at one in the morning was answered that evening, and a budget
// counting in cards has already charged it.
func TestTheFacesADayAnsweredAreCountedByTheDayTheyFallIn(t *testing.T) {
	evening := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	night := review.CardFaceID{Card: "zpqrstvwxy", Face: "Recognise"}
	morning := review.CardFaceID{Card: "3f4g5h6j7k", Face: "Recognise"}
	back := review.CardFaceID{Card: "m9n8b7v6c5", Face: "Recognise"}

	got := review.GetFaced(counting, "2026-08-29", []review.Answer{
		{ID: "01A", CardFace: evening, At: moment(t, "2026-08-29T21:00:00"), Rating: review.Good},
		{ID: "01B", CardFace: night, At: moment(t, "2026-08-30T02:00:00"), Rating: review.Good},
		{ID: "01C", CardFace: morning, At: moment(t, "2026-08-30T09:00:00"), Rating: review.Good},
		{ID: "01D", CardFace: back, At: moment(t, "2026-08-29T22:00:00"), Rating: review.Good},
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
	days := makeTallies(map[string]int{
		"2026-08-25": 4,
		// the 26th is a day nobody answered on
		"2026-08-27": 2,
		"2026-08-28": 9,
		"2026-08-29": 1,
	})
	if got := review.Streak(counting, days, moment(t, "2026-08-29T20:00:00")); got != 3 {
		t.Errorf("the streak is %d, want the three days since the gap", got)
	}
}

// A day nobody has answered on yet does not end a streak. A person who has not
// sat down this morning has not broken anything.
func TestADayNotAnsweredOnYetDoesNotEndAStreak(t *testing.T) {
	days := makeTallies(map[string]int{"2026-08-27": 2, "2026-08-28": 9})

	if got := review.Streak(counting, days, moment(t, "2026-08-29T09:00:00")); got != 2 {
		t.Errorf("the streak is %d in the morning, want the two days behind it", got)
	}
	// And the day after that, the streak is over.
	if got := review.Streak(counting, days, moment(t, "2026-08-30T09:00:00")); got != 0 {
		t.Errorf("the streak is %d a day later, want nothing", got)
	}
}

// A streak counted in the small hours is the evening's, because the night
// belongs to the day it began in.
func TestAStreakInTheSmallHoursIsTheEveningsStill(t *testing.T) {
	days := makeTallies(map[string]int{"2026-08-28": 3, "2026-08-29": 5})
	if got := review.Streak(counting, days, moment(t, "2026-08-30T02:00:00")); got != 2 {
		t.Errorf("the streak is %d at two in the morning, want 2", got)
	}
}

// A day counted in no zone in particular is counted in the machine's own,
// which is where a person's evening is.
func TestADayWithNoZoneIsTheMachinesOwn(t *testing.T) {
	here := review.Day{Starts: review.DayStarts}
	now := time.Now()
	days := makeTallies(map[string]int{
		here.GetName(now):                   1,
		here.GetName(now.AddDate(0, 0, -1)): 1,
	})

	if got := review.Streak(here, days, now); got != 2 {
		t.Errorf("the streak is %d, want the two days answered", got)
	}
}

func TestNothingAnsweredIsNoStreak(t *testing.T) {
	if got := review.Streak(counting, nil, moment(t, "2026-08-29T09:00:00")); got != 0 {
		t.Errorf("the streak is %d", got)
	}
}

// What came back is counted over the cards a person had learned, and only
// those: a card still being learned is asked whether it comes back after ten
// minutes, which says nothing about how well anything is remembered.
func TestWhatCameBackIsCountedOverWhatWasLearned(t *testing.T) {
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	other := review.CardFaceID{Card: "zpqrstvwxy", Face: "Recognise"}
	by := review.NewFSRS()

	var history []review.Answer
	said := func(id string, face review.CardFaceID, at string, r review.Rating) {
		history = append(history, review.Answer{
			ID: id, CardFace: face, At: moment(t, at), Rating: r,
		})
	}

	// One card learned on the first day, and asked again a week later.
	said("01A", on, "2026-08-01T09:00:00", review.Easy)
	said("01B", on, "2026-08-08T09:00:00", review.Good)
	// Another card met for the first time on that second day.
	said("01C", other, "2026-08-08T09:05:00", review.Again)

	got := review.GetRetained(by, counting, history)

	// The first day is a card being learned, so nothing was tested on it.
	if _, held := got["2026-08-01"]; held {
		t.Errorf("a card being learned was counted: %+v", got["2026-08-01"])
	}
	// The second is one learned card asked and recalled, and one being met.
	if want := (review.RecallTally{Asked: 1, Recalled: 1}); got["2026-08-08"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-08"], want)
	}
}

// A card a person had learned and could not recall is asked and not recalled,
// which is what the number is for.
func TestACardForgottenIsAskedAndNotRecalled(t *testing.T) {
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	by := review.NewFSRS()

	got := review.GetRetained(by, counting, []review.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-01T09:00:00"), Rating: review.Easy},
		{ID: "01B", CardFace: on, At: moment(t, "2026-08-08T09:00:00"), Rating: review.Again},
	})

	if want := (review.RecallTally{Asked: 1, Recalled: 0}); got["2026-08-08"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-08"], want)
	}
}

// An answer taken back is not in it, as it is in nothing else.
func TestAnAnswerTakenBackIsNotCountedAsRecall(t *testing.T) {
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	by := review.NewFSRS()

	got := review.GetRetained(by, counting, []review.Answer{
		{ID: "01A", CardFace: on, At: moment(t, "2026-08-01T09:00:00"), Rating: review.Easy},
		{ID: "01B", CardFace: on, At: moment(t, "2026-08-08T09:00:00"), Rating: review.Again},
		{ID: "01C", At: moment(t, "2026-08-08T09:00:30"), Undoes: "01B"},
	})

	if _, held := got["2026-08-08"]; held {
		t.Errorf("an answer taken back was counted: %+v", got["2026-08-08"])
	}
}
