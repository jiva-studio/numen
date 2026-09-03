package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// A projection answers the returning share for the days it was asked for, and
// holds no number under any other day.
//
// The share is the one pass a day makes over the whole material, and a day
// nobody asked about is a day it is not worked out on. What the days that were
// asked for come to does not stand on which of them were.
func TestAProjectionAnswersTheReturningShareForTheDaysItIsAskedFor(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 40)

	// Both rules: one carries the count from day to day and walks the material
	// only where the share is wanted, and the other walks it every day.
	interval, recall := history.Defaults(), history.Defaults()
	recall.Rule, recall.Retention = history.RuleRetention, 0.9
	for name, p := range map[string]history.Preset{
		"an interval": interval, "a chance of recall": recall,
	} {
		run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 10}
		whole := run
		whole.Retains = everyDay(10)
		run.Retains = []int{2, 7}

		some, all := ran(t, run, now, p, at, 5), ran(t, whole, now, p, at, 5)
		if days := some.Retained.Days(); len(days) != 2 || days[0] != 2 || days[1] != 7 {
			t.Errorf("under %s a run asked for days 2 and 7 answers for %v", name, days)
		}
		for day := range 10 {
			share, answers := some.Retained.On(day)
			if answers != (day == 2 || day == 7) {
				t.Errorf("under %s day %d answers %v with %v", name, day, answers, share)
			}
			if !answers {
				continue
			}
			if want, _ := all.Retained.On(day); share != want {
				t.Errorf("under %s day %d leaves %v of the material in the head, and a run "+
					"asked for every day leaves %v", name, day, share, want)
			}
		}
		// A day outside the run is a day it never covered.
		if share, answers := some.Retained.On(10); answers {
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
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := owing(by, now, 12)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 5}
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, -1)

	got := ran(t, run, now, p, at, 0)
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
		if !got.Closed[day].Holds(history.ClosedPaused) {
			t.Errorf("day %d was closed by %v, want the pause", day, got.Closed[day].Names())
		}
	}
	if got.Admits() != 0 {
		t.Errorf("%d days of the run were admitted", got.Admits())
	}
}

// The load a run reports is over the days the preset admitted, and a day it did
// not admit takes no part in either figure.
func TestTheLoadOfARunIsOverTheDaysAdmitted(t *testing.T) {
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	at := learned(by, now, 600)
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 14}
	p := history.Preset{
		Goal: history.GoalRetention, ReviewsADay: 40,
		Load: map[time.Weekday]int{time.Sunday: 0},
	}

	got := ran(t, run, now, p, at, 0)
	// A fortnight from this day holds two Sundays, so twelve of its days are
	// days of review.
	if got.Admits() != 12 {
		t.Fatalf("a fortnight with the Sundays at nothing admitted %d days", got.Admits())
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
	day := history.Day{Starts: history.DayStarts, In: time.UTC}
	now := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	open := day.Ends(now).AddDate(0, 0, -1)
	answered := func(name string, since, due int) (history.CardFaceID, history.Schedule) {
		return history.CardFaceID{Card: name, Face: "Say it"}, history.Schedule{
			Last: open.AddDate(0, 0, -since), Due: open.AddDate(0, 0, due),
			Reps: 3, Stability: 20, Difficulty: 5, Phase: 2,
		}
	}
	late, lateAt := answered("late", 10, -1)
	soon, soonAt := answered("soon", 4, 0)
	soonAt.Due = now.Add(6 * time.Hour)
	unbegun := history.CardFaceID{Card: "unbegun", Face: "Say it"}

	at := map[history.CardFaceID]history.Schedule{
		late: lateAt, soon: soonAt, unbegun: {},
	}
	if got := history.Overdue(day, at, now); got != 1 {
		t.Errorf("%d card faces stand overdue, want the one due yesterday", got)
	}

	// A run opening with nothing overdue has nothing to clear, and one nobody
	// is answering never gets there.
	run := history.Simulation{By: history.NewFSRS(), Day: day, Cost: history.DefaultCost, Days: 3}
	p := history.Preset{Goal: history.GoalRetention}
	clear := map[history.CardFaceID]history.Schedule{soon: soonAt, unbegun: {}}
	if got := ran(t, run, now, p, clear, 0).Clears; got != 0 {
		t.Errorf("a run with nothing overdue clears in %d days, want none", got)
	}
	if got := ran(t, run, now, p, at, 0).Clears; got != history.NeverClears {
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
	day := history.Day{Starts: history.DayStarts, In: time.UTC}
	now := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	open := day.Ends(now).AddDate(0, 0, -1)
	standing := func(name string, since, due int) (history.CardFaceID, history.Schedule) {
		return history.CardFaceID{Card: name, Face: "Say it"}, history.Schedule{
			Last: open.AddDate(0, 0, -since), Due: open.AddDate(0, 0, due),
			Reps: 3, Stability: 20, Difficulty: 5, Phase: 2,
		}
	}
	// The two waiting longest were answered most recently, so the order the day
	// takes them in is the order of the days they came round on.
	first, firstAt := standing("first", 10, -9)
	second, secondAt := standing("second", 11, -8)
	third, thirdAt := standing("third", 100, -3)
	fourth, fourthAt := standing("fourth", 101, -2)
	fifth, fifthAt := standing("fifth", 102, -1)

	run := history.Simulation{
		By: history.NewFSRS(), Day: day, Cost: history.DefaultCost, Days: 1, Retains: []int{0},
	}
	p := history.Preset{Goal: history.GoalRetention, ReviewsADay: 2}

	got := ran(t, run, now, p, map[history.CardFaceID]history.Schedule{
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
	want := ran(t, run, now, p, map[history.CardFaceID]history.Schedule{
		first: firstAt, second: secondAt, third: thirdAt, fourth: fourthAt, fifth: fifthAt,
	}, 0)
	if want.Load[0] != 2 || want.Backlog[0] != 0 {
		t.Fatalf("the day owed the two oldest answered %d and left %d standing",
			want.Load[0], want.Backlog[0])
	}
	one, _ := got.Retained.On(0)
	other, _ := want.Retained.On(0)
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
	by := history.NewFSRS()
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	run := history.Simulation{By: by, Day: ahead, Cost: history.DefaultCost, Days: 10}
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 5)
	p.Rule, p.Interval = history.RuleInterval, 365

	got := ran(t, run, now, p, nil, 30)
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
	run := history.Simulation{
		By: history.NewFSRS(), Day: ahead, Cost: history.DefaultCost, Days: 5,
	}

	got := ran(t, run, now, history.Defaults(), nil, 0)
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
