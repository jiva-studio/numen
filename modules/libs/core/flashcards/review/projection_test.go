package review_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// A projection answers the returning share for the days it was asked for, and
// holds no number under any other day.
//
// The share is the one pass a day makes over the whole material, and a day
// nobody asked about is a day it is not worked out on. What the days that were
// asked for come to does not stand on which of them were.
func TestAProjectionAnswersTheReturningShareForTheDaysItIsAskedFor(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 40)

	// Both rules: one carries the count from day to day and walks the material
	// only where the share is wanted, and the other walks it every day.
	interval, recall := review.Defaults(), review.Defaults()
	recall.Rule, recall.Retention = review.RuleRetention, 0.9
	for name, p := range map[string]review.Preset{
		"an interval": interval, "a chance of recall": recall,
	} {
		run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 10}
		whole := run
		whole.Retains = everyDay(10)
		run.Retains = []int{2, 7}

		some, all := runProjection(t, run, now, p, at, 5), runProjection(t, whole, now, p, at, 5)
		if days := some.Retained.Days(); len(days) != 2 || days[0] != 2 || days[1] != 7 {
			t.Errorf("under %s a run asked for days 2 and 7 answers for %v", name, days)
		}
		for day := range 10 {
			share, answers := some.Retained.GetShare(day)
			if answers != (day == 2 || day == 7) {
				t.Errorf("under %s day %d answers %v with %v", name, day, answers, share)
			}
			if !answers {
				continue
			}
			if want, _ := all.Retained.GetShare(day); share != want {
				t.Errorf("under %s day %d leaves %v of the material in the head, and a run "+
					"asked for every day leaves %v", name, day, share, want)
			}
		}
		// A day outside the run is a day it never covered.
		if share, answers := some.Retained.GetShare(10); answers {
			t.Errorf("under %s a run of ten days answers for day 10 with %v", name, share)
		}
	}
}

// A preset scheduling nothing answers nothing inside a projection, whatever
// room the budgets it is not steered by stand at.
//
// A goal of a date past holds the day to a count of new cards alone, so nothing
// but the pause stands between the reviews and the debt.
func TestAProjectionAnswersNothingUnderAPresetThatSchedulesNothing(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeOverdueFaces(by, now, 12)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 5}
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, -1)

	got := runProjection(t, run, now, p, at, 0)
	if got.Answered != 0 {
		t.Errorf("a preset aiming at a day behind us answered %d card faces", got.Answered)
	}
	for day, load := range got.Load {
		if load != 0 {
			t.Errorf("day %d answered %d card faces", day, load)
		}
		if got.Admitted[day] {
			t.Errorf("day %d was admitted", day)
		}
		if !got.Closed[day].Has(review.ClosedPaused) {
			t.Errorf("day %d was closed by %v, want the pause", day, got.Closed[day])
		}
	}
	if got.CountAdmitted() != 0 {
		t.Errorf("%d days of the run were admitted", got.CountAdmitted())
	}
}

// The load a run reports is over the days the preset admitted, and a day it did
// not admit takes no part in either figure.
func TestTheLoadOfARunIsOverTheDaysAdmitted(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 600)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 14}
	p := review.Preset{
		Goal: review.GoalRetention, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Sunday: 0},
	}

	got := runProjection(t, run, now, p, at, 0)
	// A fortnight from this day holds two Sundays, so twelve of its days are
	// days of review.
	if got.CountAdmitted() != 12 {
		t.Fatalf("a fortnight with the Sundays at nothing admitted %d days", got.CountAdmitted())
	}
	answered := 0
	var spent time.Duration
	for day := range got.Load {
		answered += got.Load[day]
		spent += got.Spent[day]
	}
	if want := float64(answered) / 12; got.ReviewsADay != want {
		t.Errorf("%d answers over twelve days of review are %v a day, want %v",
			answered, got.ReviewsADay, want)
	}
	if want := spent.Minutes() / 12; got.MinutesADay != want {
		t.Errorf("%v over twelve days of review is %v minutes a day, want %v",
			spent, got.MinutesADay, want)
	}
}

// What is overdue is what has had its day and was not answered on it: a card
// falling due later in the day holding now has not had its day, and one nobody
// has answered has had none.
func TestWhatStandsOverdue(t *testing.T) {
	day := review.Day{Starts: review.DayStarts, In: time.UTC}
	now := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	open := day.GetEnd(now).AddDate(0, 0, -1)
	answered := func(name string, since, due int) (review.CardFaceID, review.Schedule) {
		return review.CardFaceID{Card: name, Face: "Say it"}, review.Schedule{
			Last: open.AddDate(0, 0, -since), Due: open.AddDate(0, 0, due),
			Reps: 3, Stability: 20, Difficulty: 5, Phase: 2,
		}
	}
	late, lateAt := answered("late", 10, -1)
	soon, soonAt := answered("soon", 4, 0)
	soonAt.Due = now.Add(6 * time.Hour)
	unbegun := review.CardFaceID{Card: "unbegun", Face: "Say it"}

	at := map[review.CardFaceID]review.Schedule{
		late: lateAt, soon: soonAt, unbegun: {},
	}
	if got := review.Overdue(day, at, now); got != 1 {
		t.Errorf("%d card faces stand overdue, want the one due yesterday", got)
	}

	// A run opening with nothing overdue has nothing to clear, and one nobody
	// is answering never gets there.
	run := review.Simulation{By: review.NewFSRS(), Day: day, Cost: review.DefaultCost, Days: 3}
	p := review.Preset{Goal: review.GoalRetention}
	clear := map[review.CardFaceID]review.Schedule{soon: soonAt, unbegun: {}}
	if got := runProjection(t, run, now, p, clear, 0).Clears; got != 0 {
		t.Errorf("a run with nothing overdue clears in %d days, want none", got)
	}
	if got := runProjection(t, run, now, p, at, 0).Clears; got != review.NeverClears {
		t.Errorf("a run nobody answers clears in %d days", got)
	}
}

// The day answers the card faces waiting longest, and leaves the rest of the
// pile standing.
//
// The same day is run again with only those two owed, and every figure read off
// the material is the figure of the first: what the day got through is the two
// oldest debts and not two others of the five.
func TestADayAnswersTheCardFacesWaitingLongest(t *testing.T) {
	day := review.Day{Starts: review.DayStarts, In: time.UTC}
	now := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	open := day.GetEnd(now).AddDate(0, 0, -1)
	face := func(name string, since, due int) (review.CardFaceID, review.Schedule) {
		return review.CardFaceID{Card: name, Face: "Say it"}, review.Schedule{
			Last: open.AddDate(0, 0, -since), Due: open.AddDate(0, 0, due),
			Reps: 3, Stability: 20, Difficulty: 5, Phase: 2,
		}
	}
	// The two waiting longest were answered most recently, so the order the day
	// takes them in is the order of the days they came round on.
	first, firstAt := face("first", 10, -9)
	second, secondAt := face("second", 11, -8)
	third, thirdAt := face("third", 100, -3)
	fourth, fourthAt := face("fourth", 101, -2)
	fifth, fifthAt := face("fifth", 102, -1)

	run := review.Simulation{
		By: review.NewFSRS(), Day: day, Cost: review.DefaultCost, Days: 1, Retains: []int{0},
	}
	p := review.Preset{Goal: review.GoalRetention, ReviewsADay: 2}

	got := runProjection(t, run, now, p, map[review.CardFaceID]review.Schedule{
		first: firstAt, second: secondAt, third: thirdAt, fourth: fourthAt, fifth: fifthAt,
	}, 0)
	if got.Load[0] != 2 {
		t.Fatalf("a day of two reviews answered %d card faces", got.Load[0])
	}
	if got.Backlog[0] != 3 {
		t.Errorf("the day left %d card faces standing, want the other three", got.Backlog[0])
	}

	// The same five card faces, with the three the day did not reach owed on
	// the days after it instead.
	thirdAt.Due, fourthAt.Due, fifthAt.Due =
		open.AddDate(0, 0, 1), open.AddDate(0, 0, 2), open.AddDate(0, 0, 3)
	want := runProjection(t, run, now, p, map[review.CardFaceID]review.Schedule{
		first: firstAt, second: secondAt, third: thirdAt, fourth: fourthAt, fifth: fifthAt,
	}, 0)
	if want.Load[0] != 2 || want.Backlog[0] != 0 {
		t.Fatalf("the day owed the two oldest answered %d and left %d standing",
			want.Load[0], want.Backlog[0])
	}
	one, _ := got.Retained.GetShare(0)
	other, _ := want.Retained.GetShare(0)
	if one != other {
		t.Errorf("the day left %v of the material in the head, and answering the two oldest leaves %v",
			one, other)
	}
}

// A rule no card face can reach by the day the preset aims at is handed the
// whole material at once, and every card face is counted as out of reach.
//
// A pace is what spreads a material over the days it has. Where no day leaves
// a card face time to be learned, there is nothing to spread and nothing to
// gain by trickling it out.
func TestAPaceForARuleNoCardCanReach(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 10}
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 5)
	p.Rule, p.Interval = review.RuleInterval, 365

	got := runProjection(t, run, now, p, nil, 30)
	if got.Faces != 30 || got.Short != 30 {
		t.Errorf("%d card faces of %d are out of reach of a year in five days",
			got.Short, got.Faces)
	}
	// The whole material is begun in the first day, so the day after it is
	// handed nothing.
	if got.Seen != 30 || got.Load[1] != 0 {
		t.Errorf("the first day began %d of the thirty card faces, and the day "+
			"after it was handed %d showings", got.Seen, got.Load[1])
	}
}

// A preset with nothing to schedule is through its material and has learned it,
// on every day of a run.
func TestAPresetWithNoMaterialIsThroughIt(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 5,
	}

	got := runProjection(t, run, now, review.Defaults(), nil, 0)
	if got.Faces != 0 || got.Answered != 0 {
		t.Fatalf("a preset of no card faces holds %d and answered %d", got.Faces, got.Answered)
	}
	if got.Learns != 0 || got.Clears != 0 {
		t.Errorf("a preset of no card faces learns its material in %d days and clears in %d",
			got.Learns, got.Clears)
	}
	for day, through := range got.Through {
		if through != 1 {
			t.Errorf("day %d stands %v through a material of nothing, want the whole of it",
				day, through)
		}
	}
}

// A projection walks day after day, and a caller that has given up on it is
// answered with what it gave up on.
func TestAProjectionAnswersTheCallersCancellation(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	stop()

	run := review.Simulation{By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost}
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	if _, err := run.Run(ctx, time.Now(), p, nil, 30); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled projection said %v", err)
	}
}

// ahead is the day a projection is counted in, on this machine.
var ahead = review.Day{Starts: review.DayStarts}

// opens is the hour a day begins at, which is where a projection puts its
// answers.
func opens(at time.Time) time.Time { return ahead.GetEnd(at).AddDate(0, 0, -1) }

// runProjection is one projection, over a context nothing gives up on.
func runProjection(
	t *testing.T, run review.Simulation, now time.Time, p review.Preset,
	at map[review.CardFaceID]review.Schedule, unseen int,
) review.Projection {
	t.Helper()
	out, err := run.Run(t.Context(), now, p, at, unseen)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// everyDay is every day of a run, as the days a run answers the returning share
// for.
func everyDay(days int) []int {
	out := make([]int, days)
	for i := range out {
		out[i] = i
	}
	return out
}

// makeLearnedFaces is a vault of card faces answered a few times each, which
// leaves them at stabilities of days and weeks.
func makeLearnedFaces(
	by review.Scheduler, at time.Time, faces int,
) map[review.CardFaceID]review.Schedule {
	out := make(map[review.CardFaceID]review.Schedule, faces)
	for i := range faces {
		c := review.Schedule{}
		when := at.AddDate(0, 0, -60)
		for step := 0; step <= i%5; step++ {
			c = by.Next(c, when, review.Good)
			when = c.Due
		}
		out[review.CardFaceID{Card: fmt.Sprintf("card%06d", i), Face: "Recognise"}] = c
	}
	return out
}

// The minutes a day a projection reports are the minutes its answers cost. The
// projection counts them day by day over the whole vault; this counts them one
// card face at a time, walking from one review of that card to the next, and
// the two meet.
func TestMinutesADayCountedOneCardAtATime(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	cost := review.AnswerCost{New: 20 * time.Second, Review: 9 * time.Second}
	days := 60

	at := makeLearnedFaces(by, now, 40)
	run := review.Simulation{By: by, Day: ahead, Cost: cost, Days: days}
	// Nothing is capped and no card is new, so every card face falls due on its
	// own and the two counts are of the same answers.
	p := review.Preset{
		Goal: review.GoalRetention, ReviewsADay: 100000, NewADay: 0, MinutesADay: 0,
	}
	got := runProjection(t, run, now, p, at, 0)

	answers := 0
	for _, c := range at {
		answers += countRounds(by, c, now, days)
	}
	want := (time.Duration(answers) * cost.Review).Minutes() / float64(days)

	if got.Answered != answers {
		t.Errorf("the projection gave %d answers, counted one card at a time they are %d",
			got.Answered, answers)
	}
	if math.Abs(got.MinutesADay-want) > 1e-9 {
		t.Errorf("minutes a day = %v, counted one card at a time %v", got.MinutesADay, want)
	}
}

// countRounds is how many times one card face comes round over these days,
// walked from one review to the next.
//
// It carries its own copy of what an answer does to a card face, so that what
// it says about the load is not what the projection says about it.
func countRounds(
	by review.Scheduler, c review.Schedule, from time.Time, days int,
) int {
	last := from
	for range days {
		last = ahead.GetEnd(last)
	}

	out, at := 0, from
	for {
		when := at
		if c.Due.After(at) {
			when = opens(c.Due)
		}
		if !when.Before(last) {
			return out
		}
		out++

		back := review.Recall(when.Sub(c.Last), c.Stability)
		good, again := by.Next(c, when, review.Good), by.Next(c, when, review.Again)
		away := back*good.Due.Sub(when).Seconds() + (1-back)*again.Due.Sub(when).Seconds()
		c = good
		c.Stability = good.Stability*back + again.Stability*(1-back)
		c.Difficulty = good.Difficulty*back + again.Difficulty*(1-back)
		c.Due = when.Add(time.Duration(away * float64(time.Second)))

		// One answer a day: a card sent minutes away is asked again tomorrow.
		at = ahead.GetEnd(when)
	}
}

// A longer day answers more cards and leaves less of the debt standing.
func TestALongerDayAnswersMore(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 60)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 45}

	var answered, owed []int
	for minutes := 2; minutes <= 40; minutes += 2 {
		got := runProjection(t, run, now, review.Preset{
			MinutesADay: minutes, ReviewsADay: 9999, NewADay: 5,
		}, at, 40)
		answered = append(answered, got.Answered)
		owed = append(owed, got.Owed)
	}

	for i := 1; i < len(answered); i++ {
		if answered[i] < answered[i-1] {
			t.Errorf("a longer day gave %d answers, a shorter one %d", answered[i], answered[i-1])
		}
		if owed[i] > owed[i-1] {
			t.Errorf("a longer day left %d owed, a shorter one %d", owed[i], owed[i-1])
		}
	}
}

// A higher target is shorter intervals, so it is more reviews a day.
func TestAHigherTargetIsMoreReviews(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(review.NewFSRS(), now, 40)

	var minutes []float64
	for share := review.RetentionBounds.Least; share <= review.RetentionBounds.Most; share += 0.01 {
		run := review.Simulation{
			By: review.NewFSRSAt(share), Day: ahead, Cost: review.DefaultCost, Days: 120,
		}
		got := runProjection(t, run, now, review.Preset{ReviewsADay: 9999, Retention: share}, at, 0)
		minutes = append(minutes, got.MinutesADay)
	}

	for i := 1; i < len(minutes); i++ {
		if minutes[i] < minutes[i-1]-1e-9 {
			t.Errorf("a target of one step higher costs %v minutes a day, the step below it %v",
				minutes[i], minutes[i-1])
		}
	}
}

// A day carrying a share of the load answers that share of the cards.
//
// A pile deep enough to fill every day is what says the count is the budget's
// and not the material's: each day hands over what it keeps, and the half day
// hands over half of it.
func TestADayCarriesTheShareOfTheLoadItIsGiven(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeOverdueFaces(by, now, 600)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 7}
	p := review.Preset{
		Goal: review.GoalRetention, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Wednesday: 50, time.Sunday: 20},
	}

	got := runProjection(t, run, now, p, at, 0)
	for i, day := range weekdays(now, len(got.Load)) {
		want := 40
		switch day {
		case time.Wednesday:
			want = 20
		case time.Sunday:
			want = 8
		}
		if got.Load[i] != want {
			t.Errorf("the %s answered %d card faces, want %d", day, got.Load[i], want)
		}
	}
}

// A day carrying none of the load takes no card, and the day after it picks up
// what stood over.
func TestADayCarryingNoneOfTheLoadTakesNoCard(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 600)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 14}
	p := review.Preset{
		Goal: review.GoalRetention, NewADay: 20, ReviewsADay: 40, EvenLoad: true,
		Load: map[time.Weekday]int{time.Wednesday: 0},
	}

	got := runProjection(t, run, now, p, at, 0)
	for i, day := range weekdays(now, len(got.Load)) {
		if day == time.Wednesday && got.Load[i] != 0 {
			t.Errorf("a %s carrying none of the load answered %d", day, got.Load[i])
		}
	}
}

// weekdays is the day of the week each day of a projection falls on.
func weekdays(now time.Time, days int) []time.Weekday {
	out := make([]time.Weekday, days)
	open := opens(now)
	for i := range out {
		out[i] = open.Weekday()
		open = ahead.GetEnd(open)
	}
	return out
}

// A preset scheduling nothing projects nothing: no answers, and every card face
// it holds still owed.
func TestAPresetSchedulingNothingProjectsNothing(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Now())
	at := makeLearnedFaces(by, now.AddDate(0, 0, -30), 12)

	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost}
	nothing := review.Preset{Goal: review.GoalRetention, MinutesADay: 60}
	got := runProjection(t, run, now, nothing, at, 8)

	if got.Answered != 0 || got.MinutesADay != 0 {
		t.Errorf("projection = %+v, want nothing answered", got)
	}
	if got.Seen != len(at) || got.Faces != len(at)+8 {
		t.Errorf("projection = %+v, want the eight nobody has answered still unseen", got)
	}
}

// A backlog is cleared the sooner the longer the day, and a pace that never
// gets through it answers NeverClears.
func TestHowLongABacklogTakesToClear(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	// Two hundred card faces last answered sixty days ago, which is a vault a
	// person has been away from.
	at := makeLearnedFaces(by, now, 200)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost}

	if got := review.Overdue(ahead, at, now); got == 0 {
		t.Fatal("nothing stands overdue, and there is no backlog to clear")
	}

	// A day of one review is a day that never gets through two hundred of them.
	starved := runProjection(t, run, now, review.Preset{
		Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 1,
	}, at, 0)
	if starved.Clears != review.NeverClears {
		t.Errorf("one review a day clears the backlog in %d days", starved.Clears)
	}

	// A day that carries the whole load clears it at once.
	freely := runProjection(t, run, now, review.Preset{
		Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 9999,
	}, at, 0)
	if freely.Clears != 1 {
		t.Errorf("a day carrying the whole load clears the backlog in %d days", freely.Clears)
	}

	// And a day between the two takes longer than the one above it.
	slower := runProjection(t, run, now, review.Preset{
		Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 20,
	}, at, 0)
	if slower.Clears <= freely.Clears || slower.Clears == review.NeverClears {
		t.Errorf("twenty reviews a day clears in %d days and the whole load in %d",
			slower.Clears, freely.Clears)
	}
}

// A vault with nothing overdue has nothing to clear, whatever the day runs to.
func TestNothingOverdueClearsInNoDays(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost}

	// A vault of cards nobody has answered: none of them has had a day.
	at := map[review.CardFaceID]review.Schedule{}
	if got := review.Overdue(ahead, at, now); got != 0 {
		t.Errorf("%d card faces stand overdue in a vault nobody has answered", got)
	}
	got := runProjection(t, run, now, review.Preset{
		Goal: review.GoalRetention, NewADay: 1, ReviewsADay: 1,
	}, at, 40)
	if got.Clears != 0 {
		t.Errorf("a vault with nothing overdue clears in %d days", got.Clears)
	}
}

// A day that begins new cards still clears the backlog it answered.
//
// A card begun this morning is asked for again ten minutes later, so it stands
// past its hour for the rest of the day. It is no part of what the day left
// behind.
func TestBeginningNewCardsDoesNotHoldTheBacklogOpen(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	// Sixty card faces overdue, and material enough to be starting new ones on
	// every day of the projection.
	at := makeLearnedFaces(by, now, 60)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost}
	p := review.Preset{
		Goal: review.GoalRetention, NewADay: 5, ReviewsADay: 9999,
	}

	if got := review.Overdue(ahead, at, now); got == 0 {
		t.Fatal("nothing stands overdue, and there is no backlog to clear")
	}
	got := runProjection(t, run, now, p, at, 500)
	if got.Clears != 1 {
		t.Errorf("a day carrying the whole backlog clears it in %d days", got.Clears)
	}
	if got.Seen == len(at) {
		t.Fatal("no new card was begun, and the day that begins them is not under test")
	}
}

// The backlog day by day is what a band under a curve is drawn from: it begins
// where the pile stands now, and the day it reaches nothing is the day the
// clearing names.
func TestTheBacklogDayByDayAgreesWithTheDayItClears(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 200)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost}

	for _, one := range []struct {
		what string
		p    review.Preset
	}{
		{"a starved day", review.Preset{Goal: review.GoalRetention, ReviewsADay: 3}},
		{"a day of twenty", review.Preset{Goal: review.GoalRetention, ReviewsADay: 20}},
		{"a day of all of it", review.Preset{Goal: review.GoalRetention, ReviewsADay: 9999}},
	} {
		got := runProjection(t, run, now, one.p, at, 0)
		if len(got.Backlog) != got.Days {
			t.Errorf("%s projected %d days and left a backlog of %d",
				one.what, got.Days, len(got.Backlog))
		}
		for day, backlog := range got.Backlog {
			if backlog < 0 {
				t.Errorf("%s stands %d behind on day %d", one.what, backlog, day)
			}
		}

		first := review.NeverClears
		for day, backlog := range got.Backlog {
			if backlog == 0 {
				first = day + 1
				break
			}
		}
		if got.Clears != first {
			t.Errorf("%s clears on day %d and the backlog reaches nothing on day %d",
				one.what, got.Clears, first)
		}
	}
}

// A day the preset does not admit answers nothing, so nothing can leave the
// overdue pile on it: across such a day the pile stands where it was or grows.
func TestAPausedDayNeverDropsTheOverduePile(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now.AddDate(0, 0, -40), 300)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 28}

	for what, load := range map[string]map[time.Weekday]int{
		"a week of nothing": {
			time.Monday: 0, time.Tuesday: 0, time.Wednesday: 0, time.Thursday: 0,
			time.Friday: 0, time.Saturday: 0, time.Sunday: 0,
		},
		"a Wednesday and a Sunday of nothing": {time.Wednesday: 0, time.Sunday: 0},
	} {
		p := review.Preset{
			Goal: review.GoalRetention, NewADay: 20, ReviewsADay: 40,
			EvenLoad: true, Load: load,
		}
		got := runProjection(t, run, now, p, at, 60)
		if len(got.Admitted) != len(got.Backlog) {
			t.Fatalf("%s: %d days admitted against %d of backlog",
				what, len(got.Admitted), len(got.Backlog))
		}
		for day := 1; day < len(got.Backlog); day++ {
			if got.Admitted[day] {
				continue
			}
			if got.Backlog[day] < got.Backlog[day-1] {
				t.Errorf("%s: day %d admits nothing and the pile fell from %d to %d",
					what, day, got.Backlog[day-1], got.Backlog[day])
			}
			if got.Load[day] != 0 {
				t.Errorf("%s: day %d admits nothing and answered %d",
					what, day, got.Load[day])
			}
		}
	}
}

// makeSentAway is one card face last answered this many days before an instant
// and sent away for this many days from that answer.
func makeSentAway(
	name string, at time.Time, since, away int, stability float64,
) (review.CardFaceID, review.Schedule) {
	last := at.AddDate(0, 0, -since)
	return review.CardFaceID{Card: name, Face: "Recognise"}, review.Schedule{
		Due: last.AddDate(0, 0, away), Last: last, Reps: 3,
		Stability: stability, Difficulty: 5, Phase: 2,
	}
}

// What stands learned today is counted under the rule the preset names, and
// each rule reads its own value: an interval of 21 days learns the card sent
// away for 40 and passes over the one sent away for 5, and a chance of recall
// of nine in ten learns the card answered yesterday and passes over the one
// answered two hundred days ago.
func TestWhatStandsLearnedToday(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 1,
	}

	at := make(map[review.CardFaceID]review.Schedule)
	for _, one := range []struct {
		name        string
		since, away int
		stability   float64
	}{
		{"long", 30, 40, 60},
		{"faded", 200, 30, 10},
		{"short", 2, 5, 5},
		{"brief", 1, 3, 3},
	} {
		face, schedule := makeSentAway(one.name, now, one.since, one.away, one.stability)
		at[face] = schedule
	}

	for _, one := range []struct {
		rule      review.LearnedRule
		interval  int
		retention float64
		learned   int
	}{
		// Sent away for 21 days or longer: the long one and the faded one.
		{review.RuleInterval, 21, 0.9, 2},
		// And for 45 or longer: none of them, where 40 still learns the long
		// one.
		{review.RuleInterval, 45, 0.9, 0},
		{review.RuleInterval, 40, 0.9, 1},
		// Recalled today with a chance of nine in ten: everything but the faded
		// one, whose answer is two hundred days behind a stability of ten.
		{review.RuleRetention, 21, 0.9, 3},
		// A harder target turns away the one sent furthest away, and a harder
		// one still turns away every one of them.
		{review.RuleRetention, 21, 0.95, 2},
		{review.RuleRetention, 21, 0.99, 0},
	} {
		p := review.Defaults()
		p.Goal, p.ReviewsADay, p.NewADay = review.GoalRetention, 9999, 0
		p.Rule, p.Interval, p.Retention = one.rule, one.interval, one.retention

		if got := runProjection(t, run, now, p, at, 0).Learned; got != one.learned {
			t.Errorf("under %s at %d days and %v, %d card faces stand learned, want %d",
				one.rule, one.interval, one.retention, got, one.learned)
		}
	}
}

// A card face whose chance of recall stands exactly at the target is learned.
func TestACardAtTheTargetExactlyIsLearned(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	p := review.Defaults()
	p.Rule, p.Retention = review.RuleRetention, 0.9
	// A stability at which the chance of recall a day on is the target itself.
	away := 24 * time.Hour
	stability := 1.0
	for range 200 {
		if review.Recall(away, stability) >= p.Retention {
			break
		}
		stability *= 1.1
	}
	c := review.Schedule{
		Last: now.Add(-away), Due: now, Reps: 3, Stability: stability, Difficulty: 5, Phase: 2,
	}
	if !p.IsLearned(c, now) {
		t.Errorf("a card face recalled with a chance of %v stands short of a target of %v",
			review.Recall(away, stability), p.Retention)
	}
	// And a target above where it stands does not learn it.
	tighter := p
	tighter.Retention = 0.99
	if tighter.IsLearned(c, now) {
		t.Errorf("a card face recalled with a chance of %v is learned at a target of %v",
			review.Recall(away, stability), tighter.Retention)
	}
}

// The day every card face is learned is a day of the projection, and a horizon
// that ends with one of them still to learn names no day at all.
func TestTheDayEveryCardIsLearned(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 40)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 365}
	p := review.Defaults()
	p.Goal, p.ReviewsADay, p.NewADay = review.GoalRetention, 9999, 50
	p.Rule, p.Interval = review.RuleInterval, 21

	// Forty card faces answered a few times each, and a year to carry them all
	// past an interval of 21 days.
	got := runProjection(t, run, now, p, at, 0)
	if got.Learns == review.NeverLearns || got.Learns > got.Days {
		t.Errorf("a year of review learns the whole material in %d days", got.Learns)
	}
	if got.Learned == got.Faces {
		t.Fatalf("the vault opens with all %d card faces learned", got.Faces)
	}
	if got.Learns == 0 {
		t.Error("a vault with something still to learn was learned in no days")
	}

	// The same cards, and five thousand nobody has begun at fifty a day: the
	// material outruns a horizon of sixty days, and no day is named.
	shorter := run
	shorter.Days = 60
	crowded := runProjection(t, shorter, now, p, at, 5000)
	if crowded.Learns != review.NeverLearns {
		t.Errorf("a material of %d card faces was learned in %d days",
			crowded.Faces, crowded.Learns)
	}
}

// A projection over a material already learned is learned in no days.
func TestAMaterialAlreadyLearnedIsLearnedInNoDays(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 30,
	}
	face, schedule := makeSentAway("long", now, 30, 40, 60)
	p := review.Defaults()
	p.Rule, p.Interval = review.RuleInterval, 21

	got := runProjection(t, run, now, p, map[review.CardFaceID]review.Schedule{face: schedule}, 0)
	if got.Learned != 1 || got.Learns != 0 {
		t.Errorf("a material of one learned card face stands %d learned, learned in %d days",
			got.Learned, got.Learns)
	}
}

// A date is paced by the rule the preset counts by.
//
// A card face that has to be sent away for three weeks is begun three weeks
// before the day it is wanted for; one that has only to be recalled on that day
// is begun on it. The same material and the same date are two paces.
func TestADateIsPacedByTheRuleItCountsBy(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 40,
	}

	// A hundred card faces nobody has begun, and a month to learn them in.
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 30)
	p.Rule, p.Interval = review.RuleInterval, 21
	loose := p
	loose.Rule = review.RuleRetention

	tight := runProjection(t, run, now, p, nil, 100)
	soft := runProjection(t, run, now, loose, nil, 100)
	if tight.Load[0] <= soft.Load[0] {
		t.Errorf("an interval of 21 days begins %d card faces today and a chance of recall %d",
			tight.Load[0], soft.Load[0])
	}
	// Both reach the day: what separates them is when the material is begun.
	if tight.Short != 0 || soft.Short != 0 {
		t.Errorf("a month leaves %d card faces short under an interval and %d under a chance",
			tight.Short, soft.Short)
	}
}

// How many card faces cannot be learned by the day the preset aims at, whatever
// the pace, is counted and said.
//
// A card face begun today needs a fortnight to be sent away for three weeks, so
// ten days leaves every unbegun one of them short. Nothing is moved to hide it:
// the day stands, the rule stands, and the pace gets there every card face that
// can.
func TestWhatNoPaceCanReachIsCounted(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 30,
	}
	// One card face already sent away for forty days, and five nobody has begun.
	face, schedule := makeSentAway("long", now, 30, 40, 60)
	at := map[review.CardFaceID]review.Schedule{face: schedule}

	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 10)
	p.Rule, p.Interval = review.RuleInterval, 21

	got := runProjection(t, run, now, p, at, 5)
	if got.Short != 5 {
		t.Errorf("ten days leave %d of the six card faces short, want the five unbegun",
			got.Short)
	}
	// The pace is the one that gets every card face that can there, which is
	// all of them at once.
	if got.Load[0] < 5 {
		t.Errorf("the day began %d of the five it cannot learn in time", got.Load[0])
	}

	// The same vault under a rule ten days can meet.
	loose := p
	loose.Rule = review.RuleRetention
	if short := runProjection(t, run, now, loose, at, 5).Short; short != 0 {
		t.Errorf("a card face is learned the day it is answered, and %d stand short", short)
	}

	// And under the same rule with a year to do it in. What can be reached is
	// worked out over the days to the day named and not over the horizon.
	far := p
	far.By = now.AddDate(0, 0, 365)
	if short := runProjection(t, run, now, far, at, 5).Short; short != 0 {
		t.Errorf("a year leaves %d card faces short of a three-week interval", short)
	}
}

// A preset aiming at no day has nothing it cannot reach.
func TestAPresetAimingAtNoDayIsNeverShort(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 7,
	}
	p := review.Defaults()
	p.Rule, p.Interval = review.RuleInterval, 21

	if short := runProjection(t, run, now, p, nil, 20).Short; short != 0 {
		t.Errorf("a preset steered by its minutes left %d card faces short", short)
	}
}

// The day the whole material stands learned is answered where it is a day.
//
// An interval is passed once and stays passed, so the last card face to pass it
// passes it on a day. A chance of recall is a level: a card falls under the
// target as it fades and rises over it when it is answered, so no day holds all
// of them at once. And a preset aiming at a date is answered by its date.
func TestTheDayTheMaterialIsLearnedIsAskedWhereItIsADay(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 40)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 120}

	counting := review.Defaults()
	counting.Goal, counting.ReviewsADay, counting.NewADay = review.GoalRetention, 9999, 50
	counting.Rule, counting.Interval = review.RuleInterval, 21

	if got := runProjection(t, run, now, counting, at, 0).Learns; got < 0 {
		t.Errorf("an interval of 21 days is reached on day %d", got)
	}

	// The same material counted by a chance of recall.
	level := counting
	level.Rule = review.RuleRetention
	if got := runProjection(t, run, now, level, at, 0).Learns; got != review.LearnsUnasked {
		t.Errorf("a material counted by a chance of recall is all learned on day %d", got)
	}

	// And the same material under a date, which is its own answer.
	dated := counting
	dated.Goal, dated.By = review.GoalDate, now.AddDate(0, 0, 60)
	if got := runProjection(t, run, now, dated, at, 0).Learns; got != review.LearnsUnasked {
		t.Errorf("a preset aiming at a day answers with day %d beside it", got)
	}
	// What a date is qualified by is the count no pace reaches.
	if got := runProjection(t, run, now, dated, at, 0).Short; got < 0 {
		t.Errorf("a date says %d card faces cannot get there", got)
	}
}

// The day a card face ripens is the day the projection learns it.
//
// The pace of a date is worked out from the first and the picture beside it
// from the second, so a day is one day in both: a card face begun today is
// learned on the day the pace was told it would be.
func TestRipeningAndTheProjectionAreOneDay(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	by := review.NewFSRS()
	// A day long enough and a budget large enough that nothing but the rule
	// decides when the one card face is learned.
	p := review.Defaults()
	p.Goal, p.MinutesADay = review.GoalMinutes, int(review.MinutesADayBounds.Most)
	p.NewADay, p.ReviewsADay, p.EvenLoad = 1, 1000, false
	p.Rule = review.RuleInterval

	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 400}
	for _, interval := range []int{1, 2, 5, 7, 14, 21, 30, 60} {
		one := p
		one.Interval = interval
		learns := -1
		for day, through := range runProjection(t, run, now, one, nil, 1).Through {
			if through >= 1 {
				learns = day
				break
			}
		}
		if got := review.Ripens(by, ahead, one, now); got != learns {
			t.Errorf("an interval of %d days ripens in %d days of review, and the "+
				"projection learns the card face on day %d", interval, got, learns)
		}
	}
}

// A date reaches the same material whether or not the days are evened out.
//
// The pace of a date walks a card face with no day chosen for it, and the run
// beside it chooses one. The days past the front of a run carry nothing, so the
// lightest day of a window is its last and every card face is asked for later
// than the pace was told it would be.
func TestADateReachesTheSameMaterialWithTheDaysEvenedOut(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 21)
	p.Rule, p.Interval = review.RuleInterval, 21
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45

	run := review.Simulation{
		By: review.NewFSRSAt(p.Retention), Day: ahead, Cost: review.DefaultCost, Days: 22,
	}
	even, flat := p, p
	even.EvenLoad, flat.EvenLoad = true, false
	got, was := runProjection(t, run, now, even, nil, 40), runProjection(t, run, now, flat, nil, 40)

	if got.Through[21] != was.Through[21] {
		t.Errorf("the day it aims at gets through %v of the material with the days evened "+
			"out and %v without", got.Through[21], was.Through[21])
	}
	// And the pace is one that gets there: what it says no pace reaches is what
	// the run leaves unlearned.
	if got.Short != 0 || got.Through[21] < 0.5 {
		t.Errorf("the pace leaves %d card faces short and gets through %v of the material",
			got.Short, got.Through[21])
	}
}

// The pace of a date counts the days the preset admits, and no others.
//
// A day of the week at none of the load schedules nothing. Dividing the
// material by the calendar puts on the surviving days a load they were never
// paced for, and the card faces it begins too late stand unlearned on the day
// with nothing saying so.
func TestADateIsPacedOverTheDaysThePresetAdmits(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 21)
	p.Rule, p.Interval = review.RuleInterval, 21
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45
	p.Load = map[time.Weekday]int{time.Saturday: 0, time.Sunday: 0}

	run := review.Simulation{
		By: review.NewFSRSAt(p.Retention), Day: ahead, Cost: review.DefaultCost, Days: 22,
	}
	got := runProjection(t, run, now, p, nil, 40)
	if got.Through[21] < 1 || got.Short != 0 {
		t.Errorf("a week of five days gets through %v of the material by the day it aims "+
			"at, and %d card faces stand short", got.Through[21], got.Short)
	}
	// And the week the preset admits is fewer days, so each of them begins more.
	whole := p
	whole.Load = nil
	if flat := runProjection(t, run, now, whole, nil, 40); flat.Load[0] >= got.Load[0] {
		t.Errorf("a week of five days begins %d card faces today and a week of seven %d",
			got.Load[0], flat.Load[0])
	}
}

// No figure a projection reports is a number nobody can read.
//
// The scheduler divides by stability, so a card face a long run of lapses has
// worn down to none of it answers with one, and the share of the material left
// in the head is the mean over every card face beside it.
func TestNoFigureOfAProjectionIsUnreadable(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	// One card face worn down to no stability at all, among forty nobody has
	// begun.
	worn := review.Schedule{
		Last: now.AddDate(0, 0, -1), Due: now.AddDate(0, 0, -1),
		Difficulty: 9, Reps: 300, Lapses: 300, Phase: 2,
	}
	at := map[review.CardFaceID]review.Schedule{{Card: "worn", Face: "Say it"}: worn}

	p := review.Defaults()
	p.Goal, p.MinutesADay = review.GoalMinutes, 1440
	p.NewADay, p.ReviewsADay = 40, 4000
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 30,
		Retains: everyDay(30),
	}
	got := runProjection(t, run, now, p, at, 40)

	for _, day := range got.Retained.Days() {
		one, _ := got.Retained.GetShare(day)
		if math.IsNaN(one) || one < 0 || one > 1 {
			t.Fatalf("day %d leaves %v of the material in the head", day, one)
		}
	}
	for day, one := range got.Through {
		if math.IsNaN(one) || one < 0 || one > 1 {
			t.Fatalf("day %d gets through %v of the material", day, one)
		}
	}
	if math.IsNaN(got.ReviewsADay) || math.IsNaN(got.MinutesADay) {
		t.Errorf("the run answers %v cards a day in %v minutes",
			got.ReviewsADay, got.MinutesADay)
	}
}

// The pace of a date carries the share of the load its day of the week carries.
//
// The pace is a whole day's share of the material, and the day it falls on
// keeps as much of it as it keeps of everything else. A Saturday at half was
// handed a whole day of new material, and the days around it were paced as
// though it had carried its own.
func TestThePaceOfADateCarriesTheDaysShareOfTheLoad(t *testing.T) {
	// A Saturday, so the light day is the day being asked about.
	now := opens(time.Date(2026, 3, 7, 9, 41, 0, 0, time.Local))
	if now.Weekday() != time.Saturday {
		t.Fatalf("the day the pace is asked on is a %v", now.Weekday())
	}
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 9)
	p.Rule, p.Retention = review.RuleRetention, 0.9

	whole := p.GetAllowance(ahead, now, review.Spent{}, 40, 0)
	p.Load = map[time.Weekday]int{time.Saturday: 50}
	half := p.GetAllowance(ahead, now, review.Spent{}, 40, 0)

	if half.Keeps.New >= whole.Keeps.New {
		t.Errorf("a Saturday at half the load is paced %d card faces and a whole Saturday %d",
			half.Keeps.New, whole.Keeps.New)
	}
}

// What a projection assumes about coming back is an input to the run.
//
// A run told that nothing is forgotten leaves a card face exactly where the
// scheduler leaves one answered well, and a run told nothing follows it down
// the middle of what it may do, which is somewhere short of that.
func TestWhatAProjectionAssumesAboutRecallIsAnInputToTheRun(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 40)

	// One day of review, so every card face falling due in it is answered once
	// and left where that answer leaves it.
	p := review.Preset{
		Goal: review.GoalRetention, ReviewsADay: 9999, NewADay: 0, MinutesADay: 0,
	}
	run := review.Simulation{
		By: by, Day: ahead, Cost: review.DefaultCost, Days: 1, Retains: []int{0},
	}

	kept := run
	kept.Recalls = review.GetFullRecall
	modelled := runProjection(t, run, now, p, at, 0)
	nothing := runProjection(t, kept, now, p, at, 0)

	// The two runs answer the same cards, and leave them in different places.
	if modelled.Answered != nothing.Answered || modelled.Answered == 0 {
		t.Fatalf("the two runs answered %d and %d", modelled.Answered, nothing.Answered)
	}
	forgot, _ := nothing.Retained.GetShare(0)
	middle, _ := modelled.Retained.GetShare(0)
	if forgot <= middle {
		t.Errorf("assuming nothing is forgotten leaves %v of the material in the head, "+
			"and following the middle of what a card may do leaves %v", forgot, middle)
	}

	// A run told what the model says is the run told nothing.
	said := run
	said.Recalls = review.AsModelled
	if got, _ := runProjection(t, said, now, p, at, 0).Retained.GetShare(0); got != middle {
		t.Errorf("told what the model says the run retained %v, and told nothing %v",
			got, middle)
	}
}

// makeOverdueFaces is this many card faces answered once a long while ago, so
// every one of them stands overdue.
func makeOverdueFaces(
	by review.Scheduler, at time.Time, faces int,
) map[review.CardFaceID]review.Schedule {
	out := make(map[review.CardFaceID]review.Schedule, faces)
	for i := range faces {
		out[review.CardFaceID{Card: fmt.Sprintf("owed%06d", i), Face: "Recognise"}] =
			by.Next(review.Schedule{}, at.AddDate(0, 0, -60), review.Good)
	}
	return out
}

// A day two budgets closed names both.
//
// Under a goal of retention the day is held to its new cards and to its reviews
// at once. A day that hands over the whole of each has been closed by each, and
// a person told only one of them raises that one and finds nothing changed.
func TestADayTwoBudgetsClosedNamesBoth(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 1}
	at := makeOverdueFaces(by, now, 30)

	for _, one := range []struct {
		what        string
		new, review int
		want        review.BudgetNames
	}{
		{
			what: "both counts spent", new: 12, review: 5,
			want: review.BudgetNames{review.ClosedNew, review.ClosedReviews},
		},
		{
			what: "the reviews over", new: 12, review: 500,
			want: review.BudgetNames{review.ClosedNew},
		},
		{
			what: "the new cards over", new: 500, review: 5,
			want: review.BudgetNames{review.ClosedReviews},
		},
		{
			what: "both over", new: 500, review: 500,
			want: nil,
		},
	} {
		p := review.Preset{
			Goal: review.GoalRetention, NewADay: one.new, ReviewsADay: one.review,
		}
		got := runProjection(t, run, now, p, at, 20)

		if !slices.Equal(got.Closed[0], one.want) {
			t.Errorf("%s: the day closed on %v, want %v", one.what, got.Closed[0], one.want)
		}
	}
}

// The pace of a date divides the material by days of review, in the one unit.
//
// How many days there are counts each day for the share of the load it carries,
// and how many a card face needs counts days of review. A week at half the load
// has half a week of room and needs as many days of review as any other, so the
// second is weighed at the shares of the days it takes.
func TestAPaceUnderALightWeekDividesByTheRoomThatIsLeft(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	by := review.NewFSRS()
	p := review.Defaults()
	p.Goal, p.By = review.GoalDate, now.AddDate(0, 0, 29).Truncate(24*time.Hour)
	p.Rule, p.Interval, p.EvenLoad = review.RuleInterval, 21, false
	p.Load = map[time.Weekday]int{}
	for day := time.Sunday; day <= time.Saturday; day++ {
		p.Load[day] = 50
	}

	full := review.Defaults()
	full.Goal, full.By = p.Goal, p.By
	full.Rule, full.Interval, full.EvenLoad = p.Rule, p.Interval, p.EvenLoad

	for _, out := range []int{29, 45, 60} {
		p.By = now.AddDate(0, 0, out).Truncate(24 * time.Hour)
		full.By = p.By
		light := review.Ripens(by, ahead, p, now)
		whole := review.Ripens(by, ahead, full, now)

		at := p.GetAllowance(ahead, now, review.Spent{}, 500, light).Keeps.New
		was := full.GetAllowance(ahead, now, review.Spent{}, 500, whole).Keeps.New
		if at > was {
			t.Errorf("%d days out, a week at half the load begins %d card faces "+
				"a day and a whole week begins %d", out, at, was)
		}
	}
}

// allForgotten is a run in which no card face asked comes back. It is what puts
// a lapse inside the day it was answered in, on every card.
func allForgotten(review.Schedule, time.Time) float64 { return 0 }

// A day counting showings spends a slot on every one of them, and a day
// counting cards charges a face once and asks it again for nothing.
//
// Nothing comes back here, so every card the day asks falls back into it. A day
// counting showings hands over as many showings as its budget holds, and a day
// counting cards hands over that many faces, each asked until the day puts it
// down.
func TestADayCountingShowingsSpendsASlotOnEveryShowing(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: by, Day: ahead, Cost: review.DefaultCost, Days: 1, Recalls: allForgotten,
	}
	at := makeOverdueFaces(by, now, 40)
	reviews := 6

	for _, one := range []struct {
		counts review.BudgetUnit
		want   int
	}{
		{review.BudgetUnitShows, reviews},
		{review.BudgetUnitCards, reviews * review.MostShowings},
	} {
		p := review.Preset{
			Goal: review.GoalRetention, NewADay: 0, ReviewsADay: reviews, Counts: one.counts,
		}
		if got := runProjection(t, run, now, p, at, 0).Load[0]; got != one.want {
			t.Errorf("counting in %s the day gave %d showings, want %d", one.counts, got, one.want)
		}
	}
}

// What a day's budget is spent on is read where a count closes the day, and
// only a goal of retention keeps one on each side.
//
// A goal of minutes is closed by the clock, and the minutes go on every showing
// at either counting. A goal of a date is closed by the count of cards it has to
// begin, and every showing after the first is a review, which that goal holds to
// nothing. So the counting shortens the day under retention and decides nothing
// under the other two.
func TestWhatADaysBudgetIsSpentOnIsReadUnderEachGoal(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 60)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 30}
	newADay, reviewsADay := 10, 40

	for _, one := range []struct {
		goal  review.Goal
		binds bool
	}{
		{review.GoalMinutes, false},
		{review.GoalRetention, true},
		{review.GoalDate, false},
	} {
		p := review.Preset{
			Goal: one.goal, MinutesADay: 20, NewADay: newADay, ReviewsADay: reviewsADay,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 21,
		}
		if one.goal == review.GoalDate {
			p.By = now.AddDate(0, 0, 40).Truncate(24 * time.Hour)
		}

		p.Counts = review.BudgetUnitCards
		cards := runProjection(t, run, now, p, at, 40).Load[0]
		p.Counts = review.BudgetUnitShows
		shows := runProjection(t, run, now, p, at, 40).Load[0]

		if !one.binds {
			if shows != cards {
				t.Errorf("under %s the day gave %d showings counting showings and %d "+
					"counting cards, and no count closes that day", one.goal, shows, cards)
			}
			continue
		}
		// Every showing is charged, so the day is over when both counts are, and
		// a day counting cards asks for more on the same budget.
		if want := newADay + reviewsADay; shows != want {
			t.Errorf("under %s the day gave %d showings counting showings, want %d",
				one.goal, shows, want)
		}
		if shows >= cards {
			t.Errorf("under %s the day gave %d showings counting showings and %d "+
				"counting cards", one.goal, shows, cards)
		}
	}
}

// A card begun today is put into memory over minutes, so the step that sends it
// away in days falls in the day it was begun and not the day after.
func TestANewCardsLearningStepFallsInTheDayItWasBegun(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 2}
	p := review.Preset{
		Goal: review.GoalRetention, NewADay: 1, ReviewsADay: 9999,
	}

	got := runProjection(t, run, now, p, nil, 1)
	if got.Load[0] != 2 {
		t.Errorf("the day the card was begun gave %d showings, want 2", got.Load[0])
	}
	if got.Load[1] != 0 {
		t.Errorf("the day after gave %d showings, and the card was settled in the "+
			"day it was begun", got.Load[1])
	}
}

// A card the day answers into minutes falls due again before that day closes,
// and the day asks it again.
func TestAnAnswerThatLandsInsideTheDayIsAskedAgainInIt(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 1,
		Recalls: allForgotten,
	}
	p := review.Preset{Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 9999}

	face, one := makeSentAway("lapsing", now, 30, 1, 2)
	got := runProjection(t, run, now, p, map[review.CardFaceID]review.Schedule{face: one}, 0)
	if got.Load[0] <= 1 {
		t.Errorf("a day that forgot its one card gave it %d showings, and a card "+
			"answered into minutes comes round again in the day", got.Load[0])
	}
}

// A day asks one card face MostShowings times and no more. A card that keeps
// landing back in the day it was answered in is put down, and the day after it
// picks it up.
func TestADayAsksOneCardFaceNoMoreThanMostShowings(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 2,
		Recalls: allForgotten,
	}
	// A budget that never binds, so nothing but the cap stops the day.
	p := review.Preset{Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 9999}

	face, one := makeSentAway("looping", now, 30, 1, 2)
	got := runProjection(t, run, now, p, map[review.CardFaceID]review.Schedule{face: one}, 0)
	if got.Load[0] != review.MostShowings {
		t.Errorf("the day gave one card face %d showings, want %d",
			got.Load[0], review.MostShowings)
	}
	if got.Load[1] != review.MostShowings {
		t.Errorf("the day after gave the card it picked up %d showings, want %d",
			got.Load[1], review.MostShowings)
	}
}

// A budget that never binds gets through the same material at either counting:
// what a day is spent on decides nothing where nothing closes the day.
func TestABudgetThatNeverBindsGetsThroughTheSameAtEitherCounting(t *testing.T) {
	by := review.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := makeLearnedFaces(by, now, 40)
	run := review.Simulation{By: by, Day: ahead, Cost: review.DefaultCost, Days: 20}

	p := review.Preset{Goal: review.GoalRetention, NewADay: 9999, ReviewsADay: 9999}
	p.Counts = review.BudgetUnitCards
	cards := runProjection(t, run, now, p, at, 20)
	p.Counts = review.BudgetUnitShows
	shows := runProjection(t, run, now, p, at, 20)

	if !slices.Equal(cards.Load, shows.Load) {
		t.Errorf("counting cards the days carried %v, and counting showings %v",
			cards.Load, shows.Load)
	}
}

// A deck of nothing projects nothing at either counting.
func TestADeckOfNothingProjectsNothingAtEitherCounting(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 10,
	}

	for _, counts := range []review.BudgetUnit{review.BudgetUnitCards, review.BudgetUnitShows} {
		p := review.Preset{
			Goal: review.GoalRetention, NewADay: 10, ReviewsADay: 40, Counts: counts,
		}
		got := runProjection(t, run, now, p, nil, 0)
		if got.Answered != 0 || got.Faces != 0 {
			t.Errorf("counting in %s a deck of nothing answered %d over %d faces",
				counts, got.Answered, got.Faces)
		}
	}
}

// A day's showings and the card faces they are of are two counts, and what
// separates them is the card the day comes back to.
//
// The load a caller draws is counted in faces: a card asked twice in a day is
// the one card, and the second showing is time and not another card.
func TestADaysShowingsAndItsCardFacesAreTwoCounts(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := review.Simulation{
		By: review.NewFSRS(), Day: ahead, Cost: review.DefaultCost, Days: 3,
	}
	// A budget that never binds, so the day asks everything it has.
	p := review.Preset{Goal: review.GoalRetention, NewADay: 1, ReviewsADay: 9999}

	got := runProjection(t, run, now, p, nil, 1)
	if got.Load[0] != 2 || got.Faced[0] != 1 {
		t.Errorf("the day gave %d showings of %d card faces, want 2 of 1",
			got.Load[0], got.Faced[0])
	}
	if got.Answered != 2 {
		t.Errorf("the run gave %d showings, want 2", got.Answered)
	}
	// One face over three days of review, two of which the preset admitted
	// nothing to ask on.
	if want := 1.0 / 3.0; math.Abs(got.ReviewsADay-want) > 1e-9 {
		t.Errorf("the daily load is %v card faces, want %v", got.ReviewsADay, want)
	}
}

// A card face begun on any day of the week is learned in the days of review the
// preset is paced by, so a week carrying a quiet day ripens as slowly as its
// slowest day and answers the same figure whichever day it is asked on.
func TestAWeekRipensAsSlowlyAsItsSlowestDay(t *testing.T) {
	by := review.NewFSRS()
	p := review.Defaults()
	p.Rule, p.Interval = review.RuleInterval, 21
	p.Load = map[time.Weekday]int{time.Saturday: 0, time.Sunday: 0}

	from := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	want := review.Ripens(by, ahead, p, from)
	for i := 1; i < 7; i++ {
		on := from.AddDate(0, 0, i)
		if got := review.Ripens(by, ahead, p, on); got != want {
			t.Errorf("a week ripens in %d days of review asked on a %s and %d asked on a %s",
				got, on.Weekday(), want, from.Weekday())
		}
	}
}
