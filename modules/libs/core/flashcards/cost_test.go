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

// A day carrying a share of the load answers that share of the cards, and the
// day it is answers no fewer than the same day carrying all of it would.
func TestADayCarriesTheShareOfTheLoadItIsGiven(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 600)
	run := history.Simulation{
		By: by, Day: ahead,
		Cost: history.Cost{New: 20 * time.Second, Review: 10 * time.Second}, Days: 7,
	}
	p := history.Preset{MinutesADay: 14, ReviewsADay: 9999}

	flat := ran(t, run, now, p, at, 0)
	light := p
	light.Load = map[time.Weekday]int{time.Wednesday: 50}
	cut := ran(t, run, now, light, at, 0)

	if cut.Answered >= flat.Answered {
		t.Errorf("a week with a half day answered %d, and the same week of whole days %d",
			cut.Answered, flat.Answered)
	}
	for i, day := range weekdays(now, len(cut.Load)) {
		if day == time.Wednesday && cut.Load[i] >= flat.Load[i] {
			t.Errorf("the half %s carried %d, and the same day whole %d",
				day, cut.Load[i], flat.Load[i])
		}
	}
}

// A day carrying none of the load takes no card, and the day after it picks up
// what stood over.
func TestADayCarryingNoneOfTheLoadTakesNoCard(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 600)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 14}
	p := history.Preset{
		Goal: history.GoalRetention, NewADay: 20, ReviewsADay: 40, EvenLoad: true,
		Load: map[time.Weekday]int{time.Wednesday: 0},
	}

	got := ran(t, run, now, p, at, 0)
	for i, day := range weekdays(now, len(got.Load)) {
		if day == time.Wednesday && got.Load[i] != 0 {
			t.Errorf("a %s carrying none of the load answered %d", day, got.Load[i])
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
	return p.Admits(history.Day{Starts: history.DayStarts}, at, history.Spent{}, history.Left{}).Keeps
}

// The budget a preset keeps on one day is that day of the week's share of it,
// whether or not the days are evened out. A day the preset does not name keeps
// the whole of it, and a day at nothing keeps none.
func TestTheBudgetOfOneDayIsItsShareOfTheLoad(t *testing.T) {
	p := history.Preset{
		MinutesADay: 20, NewADay: 10, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Wednesday: 50, time.Sunday: 0},
	}

	half := keeps(p, time.Wednesday)
	if want := (history.Budget{New: 5, Reviews: 20, Minutes: 10}); half != want {
		t.Errorf("a day at half the load holds %+v, want %+v", half, want)
	}
	whole := keeps(p, time.Tuesday)
	if want := (history.Budget{New: 10, Reviews: 40, Minutes: 20}); whole != want {
		t.Errorf("a day the preset does not name holds %+v, want %+v", whole, want)
	}
	if none := keeps(p, time.Sunday); none != (history.Budget{}) {
		t.Errorf("a day at none of the load holds %+v", none)
	}
}

// A day at none of the load schedules nothing, as a budget of zero does.
func TestADayAtNoneOfTheLoadIsAPause(t *testing.T) {
	p := history.Preset{
		Goal: history.GoalRetention, NewADay: 10, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Sunday: 0},
	}
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	if at.Weekday() != time.Sunday {
		t.Fatalf("%v is a %v", at, at.Weekday())
	}
	if !p.Admits(ahead, at, history.Spent{}, history.Left{}).Paused {
		t.Error("a day at none of the load is not a pause")
	}
	if p.Admits(ahead, at.AddDate(0, 0, 1), history.Spent{}, history.Left{}).Paused {
		t.Error("the day after it is a pause")
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

// A day the preset does not admit answers nothing, so nothing can leave the
// overdue pile on it: across such a day the pile stands where it was or grows.
func TestAPausedDayNeverDropsTheOverduePile(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now.AddDate(0, 0, -40), 300)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 28}

	for what, load := range map[string]map[time.Weekday]int{
		"a week of nothing": {
			time.Monday: 0, time.Tuesday: 0, time.Wednesday: 0, time.Thursday: 0,
			time.Friday: 0, time.Saturday: 0, time.Sunday: 0,
		},
		"a Wednesday and a Sunday of nothing": {time.Wednesday: 0, time.Sunday: 0},
	} {
		p := history.Preset{
			Goal: history.GoalRetention, NewADay: 20, ReviewsADay: 40,
			EvenLoad: true, Load: load,
		}
		got := ran(t, run, now, p, at, 60)
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

// sent is one card face last answered this many days before an instant and
// sent away for this many days from that answer.
func sent(
	name string, at time.Time, since, away int, stability float64,
) (history.CardFace, history.Schedule) {
	last := at.AddDate(0, 0, -since)
	return history.CardFace{Card: name, Face: "Recognise"}, history.Schedule{
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
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 1,
	}

	at := make(map[history.CardFace]history.Schedule)
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
		face, schedule := sent(one.name, now, one.since, one.away, one.stability)
		at[face] = schedule
	}

	for _, one := range []struct {
		rule      history.Rule
		interval  int
		retention float64
		learned   int
	}{
		// Sent away for 21 days or longer: the long one and the faded one.
		{history.RuleInterval, 21, 0.9, 2},
		// And for 45 or longer: the long one alone, which is what says the
		// threshold is read rather than assumed.
		{history.RuleInterval, 45, 0.9, 0},
		{history.RuleInterval, 40, 0.9, 1},
		// Recalled today with a chance of nine in ten: everything but the faded
		// one, whose answer is two hundred days behind a stability of ten.
		{history.RuleRetention, 21, 0.9, 3},
		// A harder target turns away the one sent furthest away, and a harder
		// one still turns away every one of them.
		{history.RuleRetention, 21, 0.95, 2},
		{history.RuleRetention, 21, 0.99, 0},
	} {
		p := history.Defaults()
		p.Goal, p.ReviewsADay, p.NewADay = history.GoalRetention, 9999, 0
		p.Rule, p.Interval, p.Retention = one.rule, one.interval, one.retention

		if got := ran(t, run, now, p, at, 0).Learned; got != one.learned {
			t.Errorf("under %s at %d days and %v, %d card faces stand learned, want %d",
				one.rule, one.interval, one.retention, got, one.learned)
		}
	}
}

// A card face whose chance of recall stands exactly at the target is learned.
func TestACardAtTheTargetExactlyIsLearned(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	p := history.Defaults()
	p.Rule, p.Retention = history.RuleRetention, 0.9
	// A stability at which the chance of recall a day on is the target itself.
	away := 24 * time.Hour
	stability := 1.0
	for range 200 {
		if history.Recall(away, stability) >= p.Retention {
			break
		}
		stability *= 1.1
	}
	c := history.Schedule{
		Last: now.Add(-away), Due: now, Reps: 3, Stability: stability, Difficulty: 5, Phase: 2,
	}
	if !p.Learned(c, now) {
		t.Errorf("a card face recalled with a chance of %v stands short of a target of %v",
			history.Recall(away, stability), p.Retention)
	}
	// And a target above where it stands does not learn it.
	tighter := p
	tighter.Retention = 0.99
	if tighter.Learned(c, now) {
		t.Errorf("a card face recalled with a chance of %v is learned at a target of %v",
			history.Recall(away, stability), tighter.Retention)
	}
}

// The day every card face is learned is a day of the projection, and a horizon
// that ends with one of them still to learn names no day at all.
func TestTheDayEveryCardIsLearned(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 40)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 365}
	p := history.Defaults()
	p.Goal, p.ReviewsADay, p.NewADay = history.GoalRetention, 9999, 50
	p.Rule, p.Interval = history.RuleInterval, 21

	// Forty card faces answered a few times each, and a year to carry them all
	// past an interval of 21 days.
	got := ran(t, run, now, p, at, 0)
	if got.Learns == history.NeverLearns || got.Learns > got.Days {
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
	crowded := ran(t, shorter, now, p, at, 5000)
	if crowded.Learns != history.NeverLearns {
		t.Errorf("a material of %d card faces was learned in %d days",
			crowded.Faces, crowded.Learns)
	}
}

// A projection over a material already learned is learned in no days.
func TestAMaterialAlreadyLearnedIsLearnedInNoDays(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 30,
	}
	face, schedule := sent("long", now, 30, 40, 60)
	p := history.Defaults()
	p.Rule, p.Interval = history.RuleInterval, 21

	got := ran(t, run, now, p, map[history.CardFace]history.Schedule{face: schedule}, 0)
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
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 40,
	}

	// A hundred card faces nobody has begun, and a month to learn them in.
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 30)
	p.Rule, p.Interval = history.RuleInterval, 21
	loose := p
	loose.Rule = history.RuleRetention

	tight := ran(t, run, now, p, nil, 100)
	soft := ran(t, run, now, loose, nil, 100)
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
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 30,
	}
	// One card face already sent away for forty days, and five nobody has begun.
	face, schedule := sent("long", now, 30, 40, 60)
	at := map[history.CardFace]history.Schedule{face: schedule}

	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 10)
	p.Rule, p.Interval = history.RuleInterval, 21

	got := ran(t, run, now, p, at, 5)
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
	loose.Rule = history.RuleRetention
	if short := ran(t, run, now, loose, at, 5).Short; short != 0 {
		t.Errorf("a card face is learned the day it is answered, and %d stand short", short)
	}

	// And under the same rule with a year to do it in. What can be reached is
	// worked out over the days to the day named and not over the horizon.
	far := p
	far.By = now.AddDate(0, 0, 365)
	if short := ran(t, run, now, far, at, 5).Short; short != 0 {
		t.Errorf("a year leaves %d card faces short of a three-week interval", short)
	}
}

// A preset aiming at no day has nothing it cannot reach.
func TestAPresetAimingAtNoDayIsNeverShort(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 7,
	}
	p := history.Defaults()
	p.Rule, p.Interval = history.RuleInterval, 21

	if short := ran(t, run, now, p, nil, 20).Short; short != 0 {
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
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 40)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 120}

	counting := history.Defaults()
	counting.Goal, counting.ReviewsADay, counting.NewADay = history.GoalRetention, 9999, 50
	counting.Rule, counting.Interval = history.RuleInterval, 21

	if got := ran(t, run, now, counting, at, 0).Learns; got < 0 {
		t.Errorf("an interval of 21 days is reached on day %d", got)
	}

	// The same material counted by a chance of recall.
	level := counting
	level.Rule = history.RuleRetention
	if got := ran(t, run, now, level, at, 0).Learns; got != history.LearnsUnasked {
		t.Errorf("a material counted by a chance of recall is all learned on day %d", got)
	}

	// And the same material under a date, which is its own answer.
	dated := counting
	dated.Goal, dated.By = history.GoalDate, now.AddDate(0, 0, 60)
	if got := ran(t, run, now, dated, at, 0).Learns; got != history.LearnsUnasked {
		t.Errorf("a preset aiming at a day answers with day %d beside it", got)
	}
	// What a date is qualified by is the count no pace reaches.
	if got := ran(t, run, now, dated, at, 0).Short; got < 0 {
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
	by := history.NewFSRS()
	// A day long enough and a budget large enough that nothing but the rule
	// decides when the one card face is learned.
	p := history.Defaults()
	p.Goal, p.MinutesADay = history.GoalMinutes, int(history.MinutesADayBounds.Most)
	p.NewADay, p.ReviewsADay, p.EvenLoad = 1, 1000, false
	p.Rule = history.RuleInterval

	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 400}
	for _, interval := range []int{1, 2, 5, 7, 14, 21, 30, 60} {
		one := p
		one.Interval = interval
		learns := -1
		for day, through := range ran(t, run, now, one, nil, 1).Through {
			if through >= 1 {
				learns = day
				break
			}
		}
		if got := history.Ripens(by, ahead, one, now); got != learns {
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
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 21)
	p.Rule, p.Interval = history.RuleInterval, 21
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45

	run := history.Simulation{
		By: history.NewFSRSAt(p.Retention), Day: ahead, Cost: history.DefaultCost, Days: 22,
	}
	even, flat := p, p
	even.EvenLoad, flat.EvenLoad = true, false
	got, was := ran(t, run, now, even, nil, 40), ran(t, run, now, flat, nil, 40)

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
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 21)
	p.Rule, p.Interval = history.RuleInterval, 21
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45
	p.Load = map[time.Weekday]int{time.Saturday: 0, time.Sunday: 0}

	run := history.Simulation{
		By: history.NewFSRSAt(p.Retention), Day: ahead, Cost: history.DefaultCost, Days: 22,
	}
	got := ran(t, run, now, p, nil, 40)
	if got.Through[21] < 1 || got.Short != 0 {
		t.Errorf("a week of five days gets through %v of the material by the day it aims "+
			"at, and %d card faces stand short", got.Through[21], got.Short)
	}
	// And the week the preset admits is fewer days, so each of them begins more.
	whole := p
	whole.Load = nil
	if flat := ran(t, run, now, whole, nil, 40); flat.Load[0] >= got.Load[0] {
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
	worn := history.Schedule{
		Last: now.AddDate(0, 0, -1), Due: now.AddDate(0, 0, -1),
		Difficulty: 9, Reps: 300, Lapses: 300, Phase: 2,
	}
	at := map[history.CardFace]history.Schedule{{Card: "worn", Face: "Say it"}: worn}

	p := history.Defaults()
	p.Goal, p.MinutesADay = history.GoalMinutes, 1440
	p.NewADay, p.ReviewsADay = 40, 4000
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 30,
	}
	got := ran(t, run, now, p, at, 40)

	for day, one := range got.Retained {
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
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 9)
	p.Rule, p.Retention = history.RuleRetention, 0.9

	whole := p.Admits(ahead, now, history.Spent{}, history.Left{New: 40})
	p.Load = map[time.Weekday]int{time.Saturday: 50}
	half := p.Admits(ahead, now, history.Spent{}, history.Left{New: 40})

	if half.Keeps.New >= whole.Keeps.New {
		t.Errorf("a Saturday at half the load is paced %d card faces and a whole Saturday %d",
			half.Keeps.New, whole.Keeps.New)
	}
}
