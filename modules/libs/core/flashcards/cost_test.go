package flashcards_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// A projection walks day after day, and a caller that has given up on it is
// answered with what it gave up on.
func TestAProjectionAnswersTheCallersCancellation(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	stop()

	run := history.Simulation{By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost}
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	if _, err := run.Run(ctx, time.Now(), p, nil, 30); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled projection said %v", err)
	}
}

// ahead is the day a projection is counted in, on this machine.
var ahead = history.Day{Starts: history.DayStarts}

// opens is the hour a day begins at, which is where a projection puts its
// answers.
func opens(at time.Time) time.Time { return ahead.Ends(at).AddDate(0, 0, -1) }

// ran is one projection, over a context nothing gives up on.
func ran(
	t *testing.T, run history.Simulation, now time.Time, p history.Preset,
	at map[history.CardFace]history.Schedule, unseen int,
) history.Projection {
	t.Helper()
	out, err := run.Run(t.Context(), now, p, at, unseen)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// learned is a vault of card faces answered a few times each, which leaves them
// at stabilities of days and weeks.
func learned(by history.Scheduler, at time.Time, faces int) map[history.CardFace]history.Schedule {
	out := make(map[history.CardFace]history.Schedule, faces)
	for i := range faces {
		c := history.Schedule{}
		when := at.AddDate(0, 0, -60)
		for step := 0; step <= i%5; step++ {
			c = by.Next(c, when, history.Good)
			when = c.Due
		}
		out[history.CardFace{Card: fmt.Sprintf("card%06d", i), Face: "Recognise"}] = c
	}
	return out
}

// The minutes a day a projection reports are the minutes its answers cost. The
// projection counts them day by day over the whole vault; this counts them one
// card face at a time, walking from one review of that card to the next, and
// the two meet.
func TestMinutesADayCountedOneCardAtATime(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	cost := history.Cost{New: 20 * time.Second, Review: 9 * time.Second}
	days := 60

	at := learned(by, now, 40)
	run := history.Simulation{By: by, Day: ahead, Cost: cost, Days: days}
	// Nothing is capped and no card is new, so every card face falls due on its
	// own and the two counts are of the same answers.
	p := history.Preset{
		Goal: history.GoalRetention, ReviewsADay: 100000, NewADay: 0, MinutesADay: 0,
	}
	got := ran(t, run, now, p, at, 0)

	answers := 0
	for _, c := range at {
		answers += comingRound(by, c, now, days)
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

// comingRound is how many times one card face comes round over these days,
// walked from one review to the next.
//
// It carries its own copy of what an answer does to a card face, so that what
// it says about the load is not what the projection says about it.
func comingRound(
	by history.Scheduler, c history.Schedule, from time.Time, days int,
) int {
	last := from
	for range days {
		last = ahead.Ends(last)
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

		back := history.Recall(when.Sub(c.Last), c.Stability)
		good, again := by.Next(c, when, history.Good), by.Next(c, when, history.Again)
		away := back*good.Due.Sub(when).Seconds() + (1-back)*again.Due.Sub(when).Seconds()
		c = good
		c.Stability = good.Stability*back + again.Stability*(1-back)
		c.Difficulty = good.Difficulty*back + again.Difficulty*(1-back)
		c.Due = when.Add(time.Duration(away * float64(time.Second)))

		// One answer a day: a card sent minutes away is asked again tomorrow.
		at = ahead.Ends(when)
	}
}

// A longer day answers more cards and leaves less of the debt standing.
func TestALongerDayAnswersMore(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 60)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 45}

	var answered, owed []int
	for minutes := 2; minutes <= 40; minutes += 2 {
		got := ran(t, run, now, history.Preset{
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
	at := learned(history.NewFSRS(), now, 40)

	var minutes []float64
	for share := history.RetentionBounds.Least; share <= history.RetentionBounds.Most; share += 0.01 {
		run := history.Simulation{
			By: history.NewFSRSAt(share), Day: ahead, Cost: history.DefaultCost, Days: 120,
		}
		got := ran(t, run, now, history.Preset{ReviewsADay: 9999, Retention: share}, at, 0)
		minutes = append(minutes, got.MinutesADay)
	}

	for i := 1; i < len(minutes); i++ {
		if minutes[i] < minutes[i-1]-1e-9 {
			t.Errorf("a target of one step higher costs %v minutes a day, the step below it %v",
				minutes[i], minutes[i-1])
		}
	}
}

// A light day sheds part of its load and the days either side of it take it up,
// so the week answers what it answered and the days that are not light are
// longer.
func TestALightDayMovesTheWeeksWorkAndKeepsIt(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 600)
	// A day of fourteen minutes at ten seconds an answer is whole cards at every
	// share a light day and its neighbours carry.
	run := history.Simulation{
		By: by, Day: ahead,
		Cost: history.Cost{New: 20 * time.Second, Review: 10 * time.Second}, Days: 7,
	}
	p := history.Preset{MinutesADay: 14, ReviewsADay: 9999}

	flat := ran(t, run, now, p, at, 0)
	light := p
	light.LightDays = []time.Weekday{time.Wednesday}
	cut := ran(t, run, now, light, at, 0)

	if cut.Answered != flat.Answered {
		t.Errorf("a week with a light day answered %d, and the same week without one %d",
			cut.Answered, flat.Answered)
	}
	days := weekdays(now, len(cut.Load))
	for i, day := range days {
		switch day {
		case time.Wednesday:
			if cut.Load[i] >= flat.Load[i] {
				t.Errorf("the light %s carried %d, and the same day not light %d",
					day, cut.Load[i], flat.Load[i])
			}
		case time.Tuesday, time.Thursday:
			if cut.Load[i] <= flat.Load[i] {
				t.Errorf("%s, beside a light day, carried %d, and away from one %d",
					day, cut.Load[i], flat.Load[i])
			}
		}
	}
}

// keeps is what a preset keeps for a day of the week, read off the day that
// admits it.
func keeps(p history.Preset, day time.Weekday) history.Budget {
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	for at.Weekday() != day {
		at = at.AddDate(0, 0, 1)
	}
	return p.Admits(history.Day{Starts: history.DayStarts}, at, history.Spent{}, 0).Keeps
}

// The budget a preset keeps on one day is the day of the week's share of it: a
// light day holds less, the days either side of it hold more, and a week holds
// what it held.
func TestTheBudgetOfOneDayIsItsShareOfTheLoad(t *testing.T) {
	p := history.Preset{MinutesADay: 20, NewADay: 10, ReviewsADay: 40}
	p.LightDays = []time.Weekday{time.Wednesday}

	light := keeps(p, time.Wednesday)
	if want := (history.Budget{New: 5, Reviews: 20, Minutes: 10}); light != want {
		t.Errorf("a light day holds %+v, want %+v", light, want)
	}
	beside := keeps(p, time.Tuesday)
	if want := (history.Budget{New: 13, Reviews: 50, Minutes: 25}); beside != want {
		t.Errorf("the day beside it holds %+v, want %+v", beside, want)
	}
	if away := keeps(p, time.Sunday); away.Reviews != p.ReviewsADay {
		t.Errorf("a day away from the light one holds %+v", away)
	}

	var week history.Budget
	for day := time.Sunday; day <= time.Saturday; day++ {
		one := keeps(p, day)
		week.New += one.New
		week.Reviews += one.Reviews
		week.Minutes += one.Minutes
	}
	if week.Reviews != 7*p.ReviewsADay {
		t.Errorf("the week holds %d reviews, want %d", week.Reviews, 7*p.ReviewsADay)
	}
}

// A week of nothing but light days is a week of ordinary ones: there is no
// neighbour for the load to move to.
func TestAWeekOfNothingButLightDaysIsAnOrdinaryWeek(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 200)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 14}
	p := history.Preset{MinutesADay: 15, ReviewsADay: 9999}

	all := p
	for day := time.Sunday; day <= time.Saturday; day++ {
		all.LightDays = append(all.LightDays, day)
	}
	got, want := ran(t, run, now, all, at, 0).Load, ran(t, run, now, p, at, 0).Load
	if !slices.Equal(got, want) {
		t.Errorf("a week of light days carried %v, and a week of none %v", got, want)
	}
}

// An even load moves reviews to the quieter days around them, so the busiest
// day of a week stands nearer its quietest.
func TestAnEvenLoadEvensTheDaysOut(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 600)
	// No budget binds, so a day carries what falls on it and what is compared is
	// where the reviews fall.
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 60}
	p := history.Preset{Goal: history.GoalRetention, ReviewsADay: 9999}

	lumpy := ran(t, run, now, p, at, 0)
	p.EvenLoad = true
	even := ran(t, run, now, p, at, 0)

	// The first week pays the backlog, which stands where the answers already
	// given left it.
	was, is := widest(lumpy.Load[7:]), widest(even.Load[7:])
	if is >= was {
		t.Errorf("an even load spread a week over %d answers, and a load left alone over %d", is, was)
	}
}

// widest is the most a week's busiest day stands above its quietest, over every
// week of a projection.
func widest(load []int) int {
	out := 0
	for i := 0; i+7 <= len(load); i++ {
		week := load[i : i+7]
		out = max(out, slices.Max(week)-slices.Min(week))
	}
	return out
}

// weekdays is the day of the week each day of a projection falls on.
func weekdays(now time.Time, days int) []time.Weekday {
	out := make([]time.Weekday, days)
	open := opens(now)
	for i := range out {
		out[i] = open.Weekday()
		open = ahead.Ends(open)
	}
	return out
}

// A preset scheduling nothing projects nothing: no answers, and every card face
// it holds still owed.
func TestAPresetSchedulingNothingProjectsNothing(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Now())
	at := learned(by, now.AddDate(0, 0, -30), 12)

	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost}
	nothing := history.Preset{Goal: history.GoalRetention, MinutesADay: 60}
	got := ran(t, run, now, nothing, at, 8)

	if got.Answered != 0 || got.MinutesADay != 0 {
		t.Errorf("projection = %+v, want nothing answered", got)
	}
	if got.Seen != len(at) || got.Faces != len(at)+8 {
		t.Errorf("projection = %+v, want the eight nobody has answered still unseen", got)
	}
}

// The answer times already recorded are what a projection spends. An answer
// nobody sat through for an hour is counted at what a card is worth.
func TestWhatAnAnswerCostsIsReadFromTheAnswers(t *testing.T) {
	by := history.NewFSRS()
	at := time.Now().Add(-72 * time.Hour)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	var answers []history.Answer
	for i, took := range []time.Duration{
		6 * time.Second, 10 * time.Second, 8 * time.Second, 12 * time.Second, time.Hour,
	} {
		answers = append(answers, history.Answer{
			ID: string(rune('a' + i)), CardFace: on, At: at.Add(time.Duration(i) * time.Hour),
			Rating: history.Good, Took: took,
		})
	}

	// The card is still being learned for the first two answers and learned for
	// the three after them, and the hour among those counts as a minute.
	cost := history.Costed(by, answers)
	if want := (6*time.Second + 10*time.Second) / 2; cost.New != want {
		t.Errorf("a card being learned costs %v, want %v", cost.New, want)
	}
	if want := (8*time.Second + 12*time.Second + history.LongestAnswer) / 3; cost.Review != want {
		t.Errorf("a review costs %v, want %v", cost.Review, want)
	}
	if cost == history.DefaultCost {
		t.Error("the answers were never read")
	}
}

// The two kinds of answer are counted apart: a history of one kind leaves the
// other at the default.
func TestACostOfOneKindLeavesTheOtherAtTheDefault(t *testing.T) {
	by := history.NewFSRS()
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	first := history.Answer{
		ID: "a", CardFace: on, At: time.Now().Add(-72 * time.Hour),
		Rating: history.Good, Took: 30 * time.Second,
	}

	cost := history.Costed(by, []history.Answer{first})
	if cost.New != 30*time.Second {
		t.Errorf("a card being learned costs %v, want 30s", cost.New)
	}
	if cost.Review != history.DefaultCost.Review {
		t.Errorf("a review costs %v, want the default %v", cost.Review, history.DefaultCost.Review)
	}
}

// A vault holding no answer times is projected at the default, and not at
// nothing a minute.
func TestAVaultHoldingNoAnswerTimesIsProjectedAtTheDefault(t *testing.T) {
	if got := history.Costed(history.NewFSRS(), nil); got != history.DefaultCost {
		t.Errorf("cost = %+v, want the default %+v", got, history.DefaultCost)
	}
}

// A backlog is cleared the sooner the longer the day, and a pace that never
// gets through it says so rather than naming a day.
func TestHowLongABacklogTakesToClear(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	// Two hundred card faces last answered sixty days ago, which is a vault a
	// person has been away from.
	at := learned(by, now, 200)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost}

	if got := history.Overdue(ahead, at, now); got == 0 {
		t.Fatal("nothing stands overdue, and there is no backlog to clear")
	}

	// A day of one review is a day that never gets through two hundred of them.
	starved := ran(t, run, now, history.Preset{
		Goal: history.GoalRetention, NewADay: 0, ReviewsADay: 1,
	}, at, 0)
	if starved.Clears != history.NeverClears {
		t.Errorf("one review a day clears the backlog in %d days", starved.Clears)
	}

	// A day that carries the whole load clears it at once.
	freely := ran(t, run, now, history.Preset{
		Goal: history.GoalRetention, NewADay: 0, ReviewsADay: 9999,
	}, at, 0)
	if freely.Clears != 1 {
		t.Errorf("a day carrying the whole load clears the backlog in %d days", freely.Clears)
	}

	// And a day between the two takes longer than the one above it.
	slower := ran(t, run, now, history.Preset{
		Goal: history.GoalRetention, NewADay: 0, ReviewsADay: 20,
	}, at, 0)
	if slower.Clears <= freely.Clears || slower.Clears == history.NeverClears {
		t.Errorf("twenty reviews a day clears in %d days and the whole load in %d",
			slower.Clears, freely.Clears)
	}
}

// A vault with nothing overdue has nothing to clear, whatever the day runs to.
func TestNothingOverdueClearsInNoDays(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost}

	// A vault of cards nobody has answered: none of them has had a day.
	at := map[history.CardFace]history.Schedule{}
	if got := history.Overdue(ahead, at, now); got != 0 {
		t.Errorf("%d card faces stand overdue in a vault nobody has answered", got)
	}
	got := ran(t, run, now, history.Preset{
		Goal: history.GoalRetention, NewADay: 1, ReviewsADay: 1,
	}, at, 40)
	if got.Clears != 0 {
		t.Errorf("a vault with nothing overdue clears in %d days", got.Clears)
	}
}

// A day that begins new cards still clears the backlog it answered.
//
// A card begun this morning is asked for again ten minutes later, so it stands
// past its hour for the rest of the day. It is not a card the day left behind,
// and a preset with new material to begin every day would otherwise never be
// through its backlog however much of it a day carries.
func TestBeginningNewCardsDoesNotHoldTheBacklogOpen(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	// Sixty card faces overdue, and material enough to be starting new ones on
	// every day of the projection.
	at := learned(by, now, 60)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost}
	p := history.Preset{
		Goal: history.GoalRetention, NewADay: 5, ReviewsADay: 9999,
	}

	if got := history.Overdue(ahead, at, now); got == 0 {
		t.Fatal("nothing stands overdue, and there is no backlog to clear")
	}
	got := ran(t, run, now, p, at, 500)
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
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 200)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost}

	for _, one := range []struct {
		what string
		p    history.Preset
	}{
		{"a starved day", history.Preset{Goal: history.GoalRetention, ReviewsADay: 3}},
		{"a day of twenty", history.Preset{Goal: history.GoalRetention, ReviewsADay: 20}},
		{"a day of all of it", history.Preset{Goal: history.GoalRetention, ReviewsADay: 9999}},
	} {
		got := ran(t, run, now, one.p, at, 0)
		if len(got.Backlog) != got.Days {
			t.Errorf("%s projected %d days and left a backlog of %d",
				one.what, got.Days, len(got.Backlog))
		}
		for day, standing := range got.Backlog {
			if standing < 0 {
				t.Errorf("%s stands %d behind on day %d", one.what, standing, day)
			}
		}

		first := history.NeverClears
		for day, standing := range got.Backlog {
			if standing == 0 {
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
