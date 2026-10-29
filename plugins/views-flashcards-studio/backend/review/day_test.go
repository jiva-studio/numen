package review_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

var ahead = review.Day{Starts: review.DayStartTime}

func london(t *testing.T) *time.Location {
	t.Helper()
	in, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skipf("this machine carries no zone database: %v", err)
	}
	return in
}

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

func TestDayIsOwed_UnansweredCard(t *testing.T) {
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
				pos := from.Add(time.Duration(i) * time.Hour).In(in)
				y, m, d := pos.Date()
				want := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
				if pos.Hour() < hour {
					want = want.AddDate(0, 0, -1)
				}
				if got := day.GetName(pos); got != want.Format(review.DayLayout) {
					t.Fatalf("in %s, %v under a day beginning at %02d:00 is named %s, want %s",
						name, pos, hour, got, want.Format(review.DayLayout))
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
			for pos := time.Date(2025, 1, 1, 12, 0, 0, 0, in); len(days) < 800; pos = day.GetEnd(pos) {
				days = append(days, pos)
				if next := day.GetEnd(pos); !next.After(pos) {
					t.Fatalf("in %s, the day holding %v under a day beginning at %02d:00 "+
						"ends at %v", name, pos, hour, next)
				}
				on.Add(pos)
			}
			for _, item := range days {
				if got := on.CountOn(item); got != 1 {
					t.Fatalf("in %s, the day holding %v under a day beginning at %02d:00 "+
						"carries %d of the 800 cards, want 1", name, item, hour, got)
				}
			}
		}
	}
}

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
func TestAnHourTheDayDoesNotBeginAt(t *testing.T) {
	for _, content := range []string{"half past four", "18:00", "12:01", "4", "24:00"} {
		got, err := review.ParseDayStart(content)
		if !errors.Is(err, review.ErrNotAnHour) || got != review.DayStartTime {
			t.Errorf("%q: got %v (%v)", content, got, err)
		}
	}
}

// counting is a day as the rest of the application counts one, in a zone a test
// can say things about.
var counting = review.Day{
	Starts: review.DayStartTime,
	In:     time.FixedZone("test", 0),
}

func moment(t *testing.T, response string) time.Time {
	t.Helper()
	pos, err := time.ParseInLocation("2006-01-02T15:04:05", response, counting.In)
	if err != nil {
		t.Fatal(err)
	}
	return pos
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
func TestGetDayTallies_CountsAnswersPerDay(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	one := review.Answer{ID: "01A", ReviewCard: on, At: moment(t, "2026-08-29T09:00:00"), Rating: review.Good}
	two := review.Answer{ID: "01B", ReviewCard: on, At: moment(t, "2026-08-29T21:00:00"), Rating: review.Again}
	night := review.Answer{ID: "01C", ReviewCard: on, At: moment(t, "2026-08-30T02:00:00"), Rating: review.Good}
	other := review.Answer{ID: "01D", ReviewCard: on, At: moment(t, "2026-08-31T09:00:00"), Rating: review.Good}
	back := review.Answer{ID: "01E", At: moment(t, "2026-08-31T09:01:00"), Undoes: other.ID}

	got := review.GetDayTallies(counting, []review.Answer{one, two, night, other, back, one})

	if got["2026-08-29"].Answered != 3 {
		t.Errorf("the evening and the night after it come to %+v, want 3", got["2026-08-29"])
	}
	if _, target := got["2026-08-31"]; target {
		t.Errorf("a day whose only answer was taken back is counted: %v", got)
	}
	if len(got) != 1 {
		t.Errorf("counted %v", got)
	}
}

// A day says how each of the four was answered on it, because fifty cards a
// person could not recall is a different day from fifty they could.
func TestGetDayTallies_CountsPerRating(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	response := func(id string, r review.Rating) review.Answer {
		return review.Answer{
			ID: id, ReviewCard: on, At: moment(t, "2026-08-29T09:00:00"), Rating: r,
		}
	}

	got := review.GetDayTallies(counting, []review.Answer{
		response("01A", review.Again),
		response("01B", review.Good),
		response("01C", review.Good),
		response("01D", review.Easy),
		response("01E", review.Hard),
	})

	want := review.Tally{Answered: 5, Again: 1, Hard: 1, Good: 2, Easy: 1}
	if got["2026-08-29"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-29"], want)
	}
}

// makeTallies is days a person answered on, as many as each says.
func makeTallies(days map[string]int) map[string]review.Tally {
	out := make(map[string]review.Tally, len(days))
	for day, reply := range days {
		out[day] = review.Tally{Answered: reply, Good: reply}
	}
	return out
}

// What a day came to is counted under the preset each review card is grouped
// under: the answers given, and the time they took.
func TestWhatADayCameToBelowEachPreset(t *testing.T) {
	root := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	mantra := review.ReviewCardID{Card: "zpqrstvwxy", Template: "Recognise"}
	ungrouped := review.ReviewCardID{Card: "3f4g5h6j7k", Template: "Say it"}
	underMap := map[review.ReviewCardID]string{root: "Sanskrit.md", mantra: "Sanskrit.md"}

	response := func(id string, reviewCard review.ReviewCardID, pos string, took time.Duration) review.Answer {
		return review.Answer{
			ID: id, ReviewCard: reviewCard, At: moment(t, pos), Rating: review.Good, Took: took,
		}
	}
	answers := []review.Answer{
		response("01A", root, "2026-08-29T09:00:00", 6*time.Second),
		response("01B", mantra, "2026-08-29T21:00:00", 9*time.Second),
		// The night belongs to the evening it began in.
		response("01C", root, "2026-08-30T02:00:00", 5*time.Second),
		// A review card nothing groups, an answer taken back, and another day.
		response("01D", ungrouped, "2026-08-29T09:30:00", 8*time.Second),
		response("01E", root, "2026-08-29T10:00:00", 7*time.Second),
		{ID: "01F", At: moment(t, "2026-08-29T10:01:00"), Undoes: "01E"},
		response("01G", root, "2026-08-31T09:00:00", 4*time.Second),
	}

	got := review.GetSpentUnder(counting, "2026-08-29", answers, underMap, nil)

	// Each review card is new the first time it is answered, and counts once for
	// the day however many answers it took.
	want := review.Spent{Answered: 2, New: 2, Took: 20 * time.Second}
	if got["Sanskrit.md"] != want {
		t.Errorf("the day came to %+v, want %+v", got["Sanskrit.md"], want)
	}
	if len(got) != 1 {
		t.Errorf("a review card nothing groups was counted: %+v", got)
	}
}

// A preset counting in cards counts a card rated again as the one card, and a
// preset counting in shows counts each time it was put to the person. The time
// spent is the same time either way.
func TestGetSpentTime_CountsCardsAndShows(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	underMap := map[review.ReviewCardID]string{on: "Steady.md"}

	answers := make([]review.Answer, 0, 9)
	for i := range 9 {
		answers = append(answers, review.Answer{
			ID:         fmt.Sprintf("01%d", i),
			ReviewCard: on,
			At:         moment(t, "2026-08-29T09:00:00").Add(time.Duration(i) * time.Minute),
			Rating:     review.Again,
			Took:       4 * time.Second,
		})
	}

	cards := review.GetSpentUnder(counting, "2026-08-29", answers, underMap,
		map[string]review.BudgetUnit{"Steady.md": review.BudgetUnitCards})
	want := review.Spent{Answered: 1, New: 1, Took: 36 * time.Second}
	if cards["Steady.md"] != want {
		t.Errorf("counting in cards the day came to %+v, want %+v", cards["Steady.md"], want)
	}

	shows := review.GetSpentUnder(counting, "2026-08-29", answers, underMap,
		map[string]review.BudgetUnit{"Steady.md": review.BudgetUnitShows})
	want = review.Spent{Answered: 9, New: 1, Reviews: 8, Took: 36 * time.Second}
	if shows["Steady.md"] != want {
		t.Errorf("counting in shows the day came to %+v, want %+v", shows["Steady.md"], want)
	}
}

// A review card is new on the day of its earliest answer.
//
// The files arrive in whatever order they were synchronised, and a run from
// another machine sorting last by name can carry the answer that came first.
func TestTheFirstAnswerOfAReviewCardIsTheEarliestOne(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	underMap := map[review.ReviewCardID]string{on: "Steady.md"}

	answers := []review.Answer{
		{ID: "01A", ReviewCard: on, At: moment(t, "2026-08-31T09:00:00"),
			Rating: review.Good, Took: 5 * time.Second},
		{ID: "01Z", ReviewCard: on, At: moment(t, "2026-08-24T09:00:00"),
			Rating: review.Good, Took: 7 * time.Second},
	}

	got := review.GetSpentUnder(counting, "2026-08-31", answers, underMap, nil)

	want := review.Spent{Answered: 1, Reviews: 1, Took: 5 * time.Second}
	if got["Steady.md"] != want {
		t.Errorf("the day came to %+v, want %+v", got["Steady.md"], want)
	}
}

// One answer counts an hour of it at most, whatever the card stood on the
// screen for.
func TestGetSpentTime_CapsLongAnswer(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	underMap := map[review.ReviewCardID]string{on: ""}

	got := review.GetSpentUnder(counting, "2026-08-29", []review.Answer{
		{ID: "01A", ReviewCard: on, At: moment(t, "2026-08-29T09:00:00"),
			Rating: review.Good, Took: time.Hour},
	}, underMap, nil)

	want := review.Spent{Answered: 1, New: 1, Took: review.LongestAnswer}
	if got[""] != want {
		t.Errorf("the day came to %+v, want %+v", got[""], want)
	}
}

// The review cards a day has answered are the day's however late it ran. A card
// answered at one in the morning was answered that evening, and a budget
// counting in cards has already charged it.
func TestGetReviewed_GroupsByDayOfAnswer(t *testing.T) {
	evening := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	night := review.ReviewCardID{Card: "zpqrstvwxy", Template: "Recognise"}
	morning := review.ReviewCardID{Card: "3f4g5h6j7k", Template: "Recognise"}
	back := review.ReviewCardID{Card: "m9n8b7v6c5", Template: "Recognise"}

	got := review.GetReviewed(counting, "2026-08-29", []review.Answer{
		{ID: "01A", ReviewCard: evening, At: moment(t, "2026-08-29T21:00:00"), Rating: review.Good},
		{ID: "01B", ReviewCard: night, At: moment(t, "2026-08-30T02:00:00"), Rating: review.Good},
		{ID: "01C", ReviewCard: morning, At: moment(t, "2026-08-30T09:00:00"), Rating: review.Good},
		{ID: "01D", ReviewCard: back, At: moment(t, "2026-08-29T22:00:00"), Rating: review.Good},
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

// sat down this morning has not broken anything.
func TestStreak_OpenDayDoesNotEndStreak(t *testing.T) {
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
	here := review.Day{Starts: review.DayStartTime}
	now := time.Now()
	days := makeTallies(map[string]int{
		here.GetName(now):                   1,
		here.GetName(now.AddDate(0, 0, -1)): 1,
	})

	if got := review.Streak(here, days, now); got != 2 {
		t.Errorf("the streak is %d, want the two days answered", got)
	}
}

func TestStreak_ZeroWithoutAnswers(t *testing.T) {
	if got := review.Streak(counting, nil, moment(t, "2026-08-29T09:00:00")); got != 0 {
		t.Errorf("the streak is %d", got)
	}
}

// What came back is counted over the cards a person had learned, and only
// those: a card still being learned is asked whether it comes back after ten
// minutes, which says nothing about how well anything is remembered.
func TestWhatCameBackIsCountedOverWhatWasLearned(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	other := review.ReviewCardID{Card: "zpqrstvwxy", Template: "Recognise"}
	delta := review.NewFSRS()

	var history []review.Answer
	response := func(id string, reviewCard review.ReviewCardID, pos string, r review.Rating) {
		history = append(history, review.Answer{
			ID: id, ReviewCard: reviewCard, At: moment(t, pos), Rating: r,
		})
	}

	// One card learned on the first day, and asked again a week later.
	response("01A", on, "2026-08-01T09:00:00", review.Easy)
	response("01B", on, "2026-08-08T09:00:00", review.Good)
	// Another card met for the first time on that second day.
	response("01C", other, "2026-08-08T09:05:00", review.Again)

	got := review.OrderAnswers(history).GetRetained(delta, counting)

	// The first day is a card being learned, so nothing was tested on it.
	if _, target := got["2026-08-01"]; target {
		t.Errorf("a card being learned was counted: %+v", got["2026-08-01"])
	}
	// The second is one learned card asked and recalled, and one being met.
	if want := (review.RecallTally{Asked: 1, Recalled: 1}); got["2026-08-08"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-08"], want)
	}
}

// A card a person had learned and could not recall is asked and not recalled,
// which is what the number is for.
func TestGetRetained_LapsedCardCountsAsNotRecalled(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	delta := review.NewFSRS()

	got := review.OrderAnswers([]review.Answer{
		{ID: "01A", ReviewCard: on, At: moment(t, "2026-08-01T09:00:00"), Rating: review.Easy},
		{ID: "01B", ReviewCard: on, At: moment(t, "2026-08-08T09:00:00"), Rating: review.Again},
	}).GetRetained(delta, counting)

	if want := (review.RecallTally{Asked: 1, Recalled: 0}); got["2026-08-08"] != want {
		t.Errorf("the day came to %+v, want %+v", got["2026-08-08"], want)
	}
}

// An answer taken back is not in it, as it is in nothing else.
func TestAnAnswerTakenBackIsNotCountedAsRecall(t *testing.T) {
	on := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	delta := review.NewFSRS()

	got := review.OrderAnswers([]review.Answer{
		{ID: "01A", ReviewCard: on, At: moment(t, "2026-08-01T09:00:00"), Rating: review.Easy},
		{ID: "01B", ReviewCard: on, At: moment(t, "2026-08-08T09:00:00"), Rating: review.Again},
		{ID: "01C", At: moment(t, "2026-08-08T09:00:30"), Undoes: "01B"},
	}).GetRetained(delta, counting)

	if _, target := got["2026-08-08"]; target {
		t.Errorf("an answer taken back was counted: %+v", got["2026-08-08"])
	}
}
