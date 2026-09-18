package review_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// makeDueByDay is a table of the days of review, loaded with as many card faces
// on each of these many days past an instant.
func makeDueByDay(d review.Day, at time.Time, on map[int]int) *review.DueByDay {
	out := review.NewDueByDay(d)
	for day, cards := range on {
		for range cards {
			out.Add(at.AddDate(0, 0, day))
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
		Goal: review.GoalMinutes, MinutesADay: 20, IsEvenLoad: true,
		Load: map[time.Weekday]int{time.Friday: 50, time.Saturday: 0},
	}

	// The Tuesday carries the whole load and nothing yet, which is more than
	// the half day on the Friday, the three already on the Sunday and the one
	// on the Monday.
	on := makeDueByDay(day, at, map[int]int{5: 0, 6: 0, 7: 3, 8: 1, 9: 0})
	if got, want := p.ScheduleDay(on, at, due), at.AddDate(0, 0, 9); !got.Equal(want) {
		t.Errorf("the card was put on %s, want %s",
			got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	// And the day it landed on is counted against that day.
	if got := on.CountOn(at.AddDate(0, 0, 9)); got != 1 {
		t.Errorf("the day the card landed on carries %d card faces, want 1", got)
	}

	// The day the scheduler named weighs as much as the Monday beside it, and
	// keeps the card.
	tied := makeDueByDay(day, at, map[int]int{5: 3, 6: 0, 7: 0, 8: 0, 9: 1})
	if got := p.ScheduleDay(tied, at, due); !got.Equal(due) {
		t.Errorf("a card tied between its own day and the next was put on %s, want %s",
			got.Format(time.RFC3339), due.Format(time.RFC3339))
	}
}

// A table of the days of review says how many card faces fall on the day
// holding an instant.
func TestHowLoadedADayOfReviewIs(t *testing.T) {
	day := review.Day{Starts: review.DayStarts, In: time.UTC}
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	on := makeDueByDay(day, at, map[int]int{0: 2, 1: 1})

	if got := on.CountOn(at.Add(6 * time.Hour)); got != 2 {
		t.Errorf("the day holding the two card faces carries %d, want 2", got)
	}
	// A day of review runs to the hour it opens at, so an instant before that
	// hour belongs to the day before it.
	if got := on.CountOn(at.AddDate(0, 0, 1).Add(-8 * time.Hour)); got != 2 {
		t.Errorf("the small hours carry %d card faces, want the 2 of the day before", got)
	}
	if got := on.CountOn(at.AddDate(0, 0, 2)); got != 0 {
		t.Errorf("a day nothing falls on carries %d card faces", got)
	}
}

// An even load moves reviews to the quieter days around them, so the busiest
// day of a week stands nearer its quietest.
func TestAnEvenLoadEvensTheDaysOut(t *testing.T) {
	by := review.NewFSRS()
	now := getDayStart(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 600)
	// No budget binds, so a day carries what falls on it and what is compared is
	// where the reviews fall.
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 60}
	p := review.Preset{Goal: review.GoalRetention, ReviewsADay: 9999}

	lumpy := runProjection(t, run, now, p, at, 0)
	p.IsEvenLoad = true
	even := runProjection(t, run, now, p, at, 0)

	// The first week pays the backlog, which stands where the answers already
	// given left it.
	if got := getWidestWeek(lumpy.Load[7:]); got != 120 {
		t.Errorf("a load left alone spread its widest week over %d answers, want 120", got)
	}
	if got := getWidestWeek(even.Load[7:]); got != 30 {
		t.Errorf("an even load spread its widest week over %d answers, want 30", got)
	}
}

// getWidestWeek is the most a week's busiest day stands above its quietest, over every
// week of a projection.
func getWidestWeek(load []int) int {
	out := 0
	for i := 0; i+7 <= len(load); i++ {
		week := load[i : i+7]
		out = max(out, slices.Max(week)-slices.Min(week))
	}
	return out
}

// A session works out the day from a local now, and a replay from the stamp its
// log carries, which is read back in UTC. One card and one answer, over the
// night the clock goes back: both land on one moment, and not merely on one
// date.
func TestTheSessionAndTheReplayLandOnOneMomentAcrossAClockChange(t *testing.T) {
	in, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Skipf("this machine holds no zone whose clock changes: %v", err)
	}
	day := review.Day{Starts: review.DayStarts, In: in}
	by := review.NewFSRS()
	p := review.Defaults()

	face := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	stood := review.Give([]review.Answer{
		makeAnswer("01A", face.Card, face.Face, "2026-10-10T08:00:00Z", review.Good),
		makeAnswer("01B", face.Card, face.Face, "2026-10-13T08:00:00Z", review.Good),
	}).Replay(day, review.ScheduleBy(by))[face]

	// The answer is given ten days before the night the clock goes back, so the
	// day the scheduler names falls the far side of it and the window the card
	// may be moved within reaches back over it.
	local := time.Date(2026, 10, 15, 10, 0, 0, 0, in)
	changes := time.Date(2026, 10, 25, 3, 0, 0, 0, in)
	due := by.Next(stood, local, review.Good).Due
	if !due.After(changes) {
		t.Fatalf("the card comes back on %v, which is not past the clock change", due.In(in))
	}

	// The stamp that answer stands as in the log, read back as a replay reads it.
	stamp, err := review.Moment(local.UTC().Format(review.Stamp))
	if err != nil {
		t.Fatal(err)
	}
	if !stamp.Equal(local) {
		t.Fatalf("the stamp reads back as %v, and the answer was given at %v", stamp, local)
	}

	// The day the scheduler named already carries cards, so the placement moves
	// the card and the arithmetic that adds days is reached.
	loaded := func(on time.Time) *review.DueByDay {
		s := review.NewDueByDay(day)
		for range 9 {
			s.Add(on)
		}
		return s
	}

	button := p.GetDueDay(loaded(due), local, due)
	replayed := p.ScheduleDay(loaded(due.UTC()), stamp, by.Next(stood, stamp, review.Good).Due)
	if button.Equal(due) {
		t.Fatalf("the placement left the card on %v, where the scheduler put it", due.In(in))
	}
	if !button.Equal(replayed) {
		t.Errorf("the button said %v and the replay put the card on %v",
			button.In(in), replayed.In(in))
	}
}
