package wire

import (
	"reflect"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// Settings that name no rule count by the default rule at its default value.
//
// They are what a window written before the key existed sends, and what a
// preset written before it holds. A key nobody wrote leaves the default in
// force: it never leaves a threshold every card face passes.
func TestSettingsThatNameNoRuleCountByTheDefault(t *testing.T) {
	old := &v1.Settings{
		Goal:        v1.Goal_GOAL_RETENTION,
		Counts:      v1.Counts_COUNTS_CARDS,
		MinutesADay: 10,
		NewADay:     12,
		ReviewsADay: 5,
		Retention:   0.8,
		Backlog:     68,
		EvenLoad:    true,
	}
	p, err := SettingsIn(old)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -1)
	// A card face first seen yesterday and put off by sixteen days, which is
	// under the default interval of twenty-one.
	near := history.Schedule{Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16}
	if p.Learned(near, now) {
		t.Error("a card face sixteen days off is learned under settings naming no rule")
	}
	far := history.Schedule{Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21}
	if !p.Learned(far, now) {
		t.Error("a card face twenty-one days off is not learned at the default interval")
	}
}

// A place with no day to name carries none, and a place whose horizon ended
// first carries the day it never reached.
func TestAPlaceWithNoDayToNameCarriesNone(t *testing.T) {
	got := CurveOf(flashcards.Curve{
		Goal: history.GoalRetention,
		Grid: []float64{0.9},
		At: []flashcards.Point{
			{Learns: history.LearnsUnasked},
			{Learns: history.NeverLearns},
			{Learns: 12},
		},
		Now: flashcards.Nowhere, Suggested: flashcards.Nowhere,
	})

	if at := got.GetAt()[0]; at.Learns != nil {
		t.Errorf("a place with no day to name carries %d", at.GetLearns())
	}
	if at := got.GetAt()[1]; at.Learns == nil || at.GetLearns() != history.NeverLearns {
		t.Errorf("a place whose horizon ended first carries %+v", at.Learns)
	}
	if at := got.GetAt()[2]; at.Learns == nil || at.GetLearns() != 12 {
		t.Errorf("a place learned on day 12 carries %+v", at.Learns)
	}
}

// What a day's budget is spent on and what counts as learned are two settings,
// and they are carried apart.
//
// The two enums number their values alike — cards and an interval are both one,
// shows and a chance of recall are both two — so a message that carried one in
// the other's field would be read without complaint. Each value of each is put
// through and read back beside every value of the other.
func TestWhatABudgetCountsIsNotWhatCountsAsLearned(t *testing.T) {
	for _, counts := range []v1.Counts{v1.Counts_COUNTS_CARDS, v1.Counts_COUNTS_SHOWS} {
		for _, rule := range []v1.Rule{v1.Rule_RULE_INTERVAL, v1.Rule_RULE_RETENTION} {
			p, err := SettingsIn(&v1.Settings{
				Goal: v1.Goal_GOAL_MINUTES_A_DAY, Counts: counts, Learned: rule,
				MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
				Retention: 0.9, Interval: 21,
			})
			if err != nil {
				t.Fatal(err)
			}
			if p.Counts != CountsIn(counts) {
				t.Errorf("%v beside %v was read as a budget spent on %q", counts, rule, p.Counts)
			}
			if p.Rule != RuleIn(rule) {
				t.Errorf("%v beside %v was read as a rule of %q", rule, counts, p.Rule)
			}

			back := SettingsOf(p)
			if back.GetCounts() != counts {
				t.Errorf("%v beside %v came back as %v", counts, rule, back.GetCounts())
			}
			if back.GetLearned() != rule {
				t.Errorf("%v beside %v came back as %v", rule, counts, back.GetLearned())
			}
		}
	}
}

// Every setting of a preset goes out on the schema and comes back as it was.
//
// A setting dropped on either side is a person moving a control, watching the
// value never leave the window, and finding the note written without it.
func TestEverySettingComesBackAsItWentOut(t *testing.T) {
	was := history.Preset{
		Goal:        history.GoalDate,
		By:          time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		MinutesADay: 35,
		NewADay:     8,
		ReviewsADay: 45,
		Retention:   0.87,
		Rule:        history.RuleRetention,
		Interval:    14,
		Counts:      history.CountsShows,
		Backlog:     40,
		Load:        map[time.Weekday]int{time.Monday: 80, time.Saturday: 50, time.Sunday: 0},
		EvenLoad:    true,
	}

	got, err := SettingsIn(SettingsOf(was))
	if err != nil {
		t.Fatal(err)
	}

	if got.Goal != was.Goal {
		t.Errorf("goal came back as %q", got.Goal)
	}
	if !got.By.Equal(was.By) {
		t.Errorf("by_date came back as %s", got.By.Format(history.Named))
	}
	if got.MinutesADay != was.MinutesADay {
		t.Errorf("minutes_a_day came back as %d", got.MinutesADay)
	}
	if got.NewADay != was.NewADay {
		t.Errorf("new_a_day came back as %d", got.NewADay)
	}
	if got.ReviewsADay != was.ReviewsADay {
		t.Errorf("reviews_a_day came back as %d", got.ReviewsADay)
	}
	if got.Retention != was.Retention {
		t.Errorf("retention came back as %g", got.Retention)
	}
	if got.Rule != was.Rule {
		t.Errorf("learned came back as %q", got.Rule)
	}
	if got.Interval != was.Interval {
		t.Errorf("interval came back as %d", got.Interval)
	}
	if got.Counts != was.Counts {
		t.Errorf("counts came back as %q", got.Counts)
	}
	if got.Backlog != was.Backlog {
		t.Errorf("backlog came back as %d", got.Backlog)
	}
	if got.EvenLoad != was.EvenLoad {
		t.Errorf("even_load came back as %t", got.EvenLoad)
	}
	if !reflect.DeepEqual(got.Load, was.Load) {
		t.Errorf("load came back as %v", got.Load)
	}
	// A setting neither this test nor the schema names yet.
	if !reflect.DeepEqual(got, was) {
		t.Errorf("the preset came back as %+v, and went out as %+v", got, was)
	}
}

// A goal, a rule and what a budget counts arriving unspecified are each not a
// value.
//
// A client that names none of them is one that has not said, and the settings
// are weighed against what a preset may hold before anything is written.
func TestAnUnspecifiedSettingIsNotAValue(t *testing.T) {
	p, err := SettingsIn(&v1.Settings{})
	if err != nil {
		t.Fatal(err)
	}

	if history.KnownGoal(p.Goal) {
		t.Errorf("an unspecified goal was read as %q", p.Goal)
	}
	if history.KnownRule(p.Rule) {
		t.Errorf("an unspecified rule was read as %q", p.Rule)
	}
	if history.KnownCounts(p.Counts) {
		t.Errorf("an unspecified counts was read as %q", p.Counts)
	}
}
