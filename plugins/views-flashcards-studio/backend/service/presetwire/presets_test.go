package presetwire

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review/projection"
)

// Settings that name no rule count by the default rule at its default value.
//
// They are what a client written before the key existed sends, and what a
// preset written before it holds. A missing key leaves the default in force: it
// never produces a threshold that every review card passes.
func TestSettingsThatNameNoRuleCountByTheDefault(t *testing.T) {
	old := &api.Settings{
		Goal:        api.GoalRetention,
		Counts:      api.BudgetUnitCards,
		MinutesADay: 10,
		NewADay:     12,
		ReviewsADay: 5,
		Retention:   0.8,
		Backlog:     68,
		HasEvenLoad: true,
	}
	p, err := ConvertSettingsFromWire(old)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -1)
	// A review card first seen yesterday and put off by sixteen days, which is
	// under the default interval of twenty-one.
	targetSchedule := review.Schedule{Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16}
	if p.IsLearned(targetSchedule, now) {
		t.Error("a review card sixteen days off is learned under settings naming no rule")
	}
	far := review.Schedule{Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21}
	if !p.IsLearned(far, now) {
		t.Error("a review card twenty-one days off is not learned at the default interval")
	}
}

// A place with no day to name carries none, and a place whose horizon ended
// first carries the day it never reached.
func TestAPlaceWithNoDayToNameCarriesNone(t *testing.T) {
	got := ConvertCurveToWire(projection.Curve{
		Goal: review.GoalRetention,
		Grid: []float64{0.9},
		Points: []projection.Point{
			{Learns: review.LearnsUnasked},
			{Learns: review.NeverLearns},
			{Learns: 12},
		},
		Now: projection.OutsideGrid, Suggested: projection.OutsideGrid,
	})

	if pos := got.At[0]; pos.Learns != nil {
		t.Errorf("a place with no day to name carries %d", *pos.Learns)
	}
	if offset := got.At[1]; offset.Learns == nil || *offset.Learns != review.NeverLearns {
		t.Errorf("a place whose horizon ended first carries %+v", offset.Learns)
	}
	if index := got.At[2]; index.Learns == nil || *index.Learns != 12 {
		t.Errorf("a place learned on day 12 carries %+v", index.Learns)
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
	for _, counts := range []api.BudgetUnit{
		api.BudgetUnitCards, api.BudgetUnitShows,
	} {
		for _, rule := range []api.Rule{api.RuleInterval, api.RuleRetention} {
			p, err := ConvertSettingsFromWire(&api.Settings{
				Goal: api.GoalMinutesADay, Counts: counts, Learned: rule,
				MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
				Retention: 0.9, Interval: 21,
			})
			if err != nil {
				t.Fatal(err)
			}
			if p.Counts != ConvertBudgetUnitFromWire(counts) {
				t.Errorf("%v beside %v was read as a budget spent on %q", counts, rule, p.Counts)
			}
			if p.Rule != ConvertRuleFromWire(rule) {
				t.Errorf("%v beside %v was read as a rule of %q", rule, counts, p.Rule)
			}

			back := ConvertSettingsToWire(p)
			if back.Counts != counts {
				t.Errorf("%v beside %v came back as %v", counts, rule, back.Counts)
			}
			if back.Learned != rule {
				t.Errorf("%v beside %v came back as %v", rule, counts, back.Learned)
			}
		}
	}
}

// Every setting of a preset is sent on the schema and returned unchanged.
//
// A setting dropped on either side means a control change that never leaves the
// window and a note that is written without it.
func TestEverySettingReachesBackAsItWentOut(t *testing.T) {
	was := review.Preset{
		Goal:        review.GoalDate,
		By:          time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		MinutesADay: 35,
		NewADay:     8,
		ReviewsADay: 45,
		Retention:   0.87,
		Rule:        review.RuleRetention,
		Interval:    14,
		Counts:      review.BudgetUnitShows,
		Backlog:     40,
		Load:        map[time.Weekday]int{time.Monday: 80, time.Saturday: 50, time.Sunday: 0},
		IsEvenLoad:  true,
	}

	got, err := ConvertSettingsFromWire(ConvertSettingsToWire(was))
	if err != nil {
		t.Fatal(err)
	}

	if got.Goal != was.Goal {
		t.Errorf("goal came back as %q", got.Goal)
	}
	if !got.By.Equal(was.By) {
		t.Errorf("by_date came back as %s", got.By.Format(review.DayLayout))
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
	if got.IsEvenLoad != was.IsEvenLoad {
		t.Errorf("even_load came back as %t", got.IsEvenLoad)
	}
	if !reflect.DeepEqual(got.Load, was.Load) {
		t.Errorf("load came back as %v", got.Load)
	}
	// A setting neither this test nor the schema names yet.
	if !reflect.DeepEqual(got, was) {
		t.Errorf("the preset came back as %+v, and went out as %+v", got, was)
	}
}

// A goal, a rule and a budget unit received as unspecified are each not a
// value.
//
// A client that names none of them has not set them, and the settings are
// validated against what a preset may hold before anything is written.
func TestAnUnspecifiedSettingIsNotAValue(t *testing.T) {
	p, err := ConvertSettingsFromWire(&api.Settings{})
	if err != nil {
		t.Fatal(err)
	}

	if review.IsKnownGoal(p.Goal) {
		t.Errorf("an unspecified goal was read as %q", p.Goal)
	}
	if review.IsKnownRule(p.Rule) {
		t.Errorf("an unspecified rule was read as %q", p.Rule)
	}
	if review.IsKnownBudgetUnit(p.Counts) {
		t.Errorf("an unspecified counts was read as %q", p.Counts)
	}
}

// Every verdict is the schema value of the same name.
//
// Both windows hold a sentence per verdict, and each names the schema value. A
// verdict converted to another one shows the wrong sentence to the user.
func TestEveryVerdictCrossesAsItself(t *testing.T) {
	for _, one := range []struct {
		why      review.StopReason
		expected api.StopReason
	}{
		{review.StoppedNothing, api.StopReasonNothing},
		{review.StoppedNoMinutes, api.StopReasonNoMinutes},
		{review.StoppedNoCards, api.StopReasonNoCards},
		{review.StoppedNoDay, api.StopReasonNoDay},
		{review.StoppedPastDay, api.StopReasonPastDay},
		{review.StoppedNoLoad, api.StopReasonNoLoad},
		{review.StoppedNoWeek, api.StopReasonNoWeek},
	} {
		if got := ConvertStopReasonToWire(one.why); got != one.expected {
			t.Errorf("%q crosses as %v, and it is %v", one.why, got, one.expected)
		}
	}

	// A verdict missing from this table is not "the preset schedules". A preset
	// stopped for a reason the schema has no value for would be displayed
	// as one that offers a day of review.
	if got := ConvertStopReasonToWire(review.StopReason("sideways")); got != api.StopReasonUnspecified {
		t.Errorf("a verdict the schema does not name crosses as %v", got)
	}
}

// A goal, a rule and a count that the schema does not name convert as
// unspecified.
//
// A note names them in its own words, and a word the reader does not recognise
// leaves the field at its default.
func TestAWordTheSchemaDoesNotNameCrossesAsUnspecified(t *testing.T) {
	if got := ConvertGoalToWire(review.Goal("sideways")); got != api.GoalUnspecified {
		t.Errorf("a goal the schema does not name crosses as %v", got)
	}
	if got := ConvertRuleToWire(review.LearnedRule("sideways")); got != api.RuleUnspecified {
		t.Errorf("a rule the schema does not name crosses as %v", got)
	}
	if got := ConvertBudgetUnitToWire(review.BudgetUnit("sideways")); got != api.BudgetUnitUnspecified {
		t.Errorf("a count the schema does not name crosses as %v", got)
	}
}

// A day that is not one, and a load kept on something that is not a day of the
// week, are the client's to correct.
func TestSettingsTheCoreCannotMakeOutAreRefused(t *testing.T) {
	for _, one := range []struct {
		named    string
		settings *api.Settings
	}{
		{"a day written in no form the core reads", &api.Settings{ByDate: "30 September"}},
		{"a load on a name no week carries", &api.Settings{Load: map[string]int32{"caturday": 50}}},
	} {
		p, err := ConvertSettingsFromWire(one.settings)
		if err == nil {
			t.Errorf("%s is taken", one.named)
		}
		if !reflect.DeepEqual(p, review.Preset{}) {
			t.Errorf("%s comes back as a preset", one.named)
		}
	}
}
