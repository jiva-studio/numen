package review_test

import (
	"encoding/json"
	"maps"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// A date is a budget: past the day it names, the preset schedules nothing.
//
// The day it names is a whole review day, so the preset schedules through every
// hour of it and stops when the next one opens.
func TestADateIsABudget(t *testing.T) {
	p, _ := review.ReadPreset(front(t, "goal: by_date\nby_date: 2026-09-30\n"))
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}

	for _, one := range []struct {
		hour   time.Time
		isPast bool
	}{
		{time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC), false},
		// The small hours of the 30th are still the day before it.
		{time.Date(2026, 9, 30, 2, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC), false},
		// And the small hours of the 1st are still the day it names.
		{time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), true},
	} {
		if got := p.IsPast(day, one.hour); got != one.isPast {
			t.Errorf("at %v the day it aims at is past = %v, want %v", one.hour, got, one.isPast)
		}
		if got := p.IsPaused(day, one.hour); got != one.isPast {
			t.Errorf("at %v the preset is paused = %v, want %v", one.hour, got, one.isPast)
		}
	}
}

// A budget of zero is a pause where the goal names that budget, and every deck
// pointing at the preset stops. A budget the goal does not name is not read.
func TestZeroIsAPauseBelowTheGoalThatNamesIt(t *testing.T) {
	for _, one := range []struct {
		front    string
		isPaused bool
	}{
		{"goal: retention\nnew_a_day: 0\nreviews_a_day: 0\n", true},
		{"goal: retention\nminutes_a_day: 0\n", false},
		{"goal: minutes_a_day\nminutes_a_day: 0\n", true},
		{"goal: minutes_a_day\nnew_a_day: 0\nreviews_a_day: 0\n", false},
	} {
		p, problems := review.ReadPreset(front(t, one.front))

		if len(problems) != 0 {
			t.Fatalf("problems = %v", problems)
		}
		got := p.IsPaused(review.Day{Starts: review.DayStartTime}, time.Now())
		if got != one.isPaused {
			t.Errorf("%q is paused %v, want %v", one.front, got, one.isPaused)
		}
	}
}

// A review card is learned on one side of its rule's threshold and not on the
// other, and the rule the preset does not name takes no part.
func TestEachRuleOnEitherSideOfItsThreshold(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)
	schedule := func(since, away int, stability float64) review.Schedule {
		last := now.AddDate(0, 0, -since)
		return review.Schedule{
			Due: last.AddDate(0, 0, away), Last: last, Reps: 3, Stability: stability,
		}
	}

	p := review.Defaults()
	p.Rule, p.Interval, p.Retention = review.RuleInterval, 21, 0.9

	// The threshold is the interval the preset carries: a card sent away for
	// exactly it is learned and one sent away a day short of it is not,
	// whatever the chance of recalling either today.
	for _, days := range []int{7, 21, 60} {
		one := p
		one.Interval = days
		if !one.IsLearned(schedule(200, days, 10), now) {
			t.Errorf("a card sent away for %d days is not learned at an interval of %d",
				days, days)
		}
		if one.IsLearned(schedule(0, days-1, 90), now) {
			t.Errorf("a card sent away for %d days is learned at an interval of %d",
				days-1, days)
		}
	}
	if p.IsLearned(review.Schedule{}, now) {
		t.Error("unanswered review card marked as learned")
	}

	p.Rule = review.RuleRetention

	// The same two review cards, under the other rule: what is asked now is the
	// chance of recalling them today, and the intervals take no part.
	if p.IsLearned(schedule(200, 21, 10), now) {
		t.Error("a card 200 days past an answer at a stability of 10 is recalled nine times in ten")
	}
	if !p.IsLearned(schedule(0, 20, 90), now) {
		t.Error("a card answered today is not recalled nine times in ten")
	}
	if p.IsLearned(review.Schedule{}, now) {
		t.Error("unanswered review card marked as learned")
	}
}

// A preset naming no rule counts by the default rule at its default value.
//
// A key nobody wrote leaves the default in force. It never leaves a threshold
// every review card passes: a preset that says nothing about what counts as
// learned counts nothing learned that three weeks away would not.
func TestAPresetNamingNoRuleCountsByTheDefault(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -1)
	// A review card first seen yesterday and put off by sixteen days, and one put
	// off by twenty-one.
	targetSchedule := review.Schedule{Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16}
	far := review.Schedule{Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21}

	// Every field left empty, which is a preset built from settings that name
	// no rule at all.
	var response review.Preset
	if response.IsLearned(targetSchedule, now) {
		t.Error("a review card sixteen days off is learned under a preset naming no rule")
	}
	if !response.IsLearned(far, now) {
		t.Error("a review card twenty-one days off is not learned at the default interval")
	}

	// A rule named with no value under it reads the default value too.
	named := review.Preset{Rule: review.RuleInterval}
	if named.IsLearned(targetSchedule, now) {
		t.Error("a review card sixteen days off is learned under an interval nobody wrote")
	}

	// And the other rule, whose value nobody wrote, holds cards to the default
	// chance of recall.
	faded := review.Schedule{
		Last: now.AddDate(0, 0, -200), Due: now, Reps: 3, Stability: 10,
	}
	chance := review.Preset{Rule: review.RuleRetention}
	if chance.IsLearned(faded, now) {
		t.Error("a review card two hundred days past its answer is recalled nine times in ten")
	}
	if !chance.IsLearned(targetSchedule, now) {
		t.Error("a review card answered yesterday is not recalled nine times in ten")
	}
}

// The mark a schedule cache is filed under carries everything that decides
// which day a card lands on.
//
// A cache worked out under one placing is thrown away whole when the placing
// changes, so a setting that moves a card and not the mark is a cache read back
// under settings it was never worked out for.
func TestThePlacingCarriesEverythingThatMovesACard(t *testing.T) {
	preset := review.Defaults()
	preset.Load = map[time.Weekday]int{time.Saturday: 50}
	was := preset.GetPlacing()

	for what, alter := range map[string]func(p *review.Preset){
		"an even load off":   func(p *review.Preset) { p.IsEvenLoad = false },
		"a goal of a date":   func(p *review.Preset) { p.Goal = review.GoalDate },
		"a Saturday freed":   func(p *review.Preset) { p.Load = nil },
		"a lighter Saturday": func(p *review.Preset) { p.Load[time.Saturday] = 20 },
	} {
		one := preset
		one.Load = maps.Clone(preset.Load)
		alter(&one)
		if got := one.GetPlacing(); got == was {
			t.Errorf("with %s a card is placed under %q, the same mark as without it",
				what, got)
		}
	}
	for day := time.Sunday; day <= time.Saturday; day++ {
		one := preset
		one.Load = maps.Clone(preset.Load)
		one.Load[day] = 30
		if got := one.GetPlacing(); got == was {
			t.Errorf("a %v at thirty is placed under %q, the same mark as one at the "+
				"whole of it", day, got)
		}
	}
}

// What a preset schedules is the core's answer, over every goal and every
// budget at zero and not.
//
// The budget the goal names decides it, and a budget the goal does not name
// takes no part whatever it holds.
func TestWhatAPresetSchedules(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	// A Thursday, and the two days a preset may aim at from it.
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	ahead := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	pastDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	for _, one := range []struct {
		what string
		p    review.Preset
		want review.StopReason
	}{
		{
			what: "minutes, and the minutes it names",
			p:    review.Preset{Goal: review.GoalMinutes, MinutesADay: 20},
			want: review.StoppedNothing,
		},
		{
			what: "minutes at zero",
			p: review.Preset{
				Goal: review.GoalMinutes, MinutesADay: 0, NewADay: 8, ReviewsADay: 45,
			},
			want: review.StoppedNoMinutes,
		},
		{
			what: "minutes, with both card counts at zero",
			p: review.Preset{
				Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 0, ReviewsADay: 0,
			},
			want: review.StoppedNothing,
		},
		{
			what: "minutes, past a day it does not aim at",
			p: review.Preset{
				Goal: review.GoalMinutes, MinutesADay: 20, By: pastDate,
			},
			want: review.StoppedNothing,
		},
		{
			what: "retention, and both counts",
			p: review.Preset{
				Goal: review.GoalRetention, NewADay: 8, ReviewsADay: 45,
			},
			want: review.StoppedNothing,
		},
		{
			what: "retention, with no new cards a day",
			p: review.Preset{
				Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 45,
			},
			want: review.StoppedNothing,
		},
		{
			what: "retention, with no reviews a day",
			p: review.Preset{
				Goal: review.GoalRetention, NewADay: 8, ReviewsADay: 0,
			},
			want: review.StoppedNothing,
		},
		{
			what: "retention, with both counts at zero",
			p: review.Preset{
				Goal: review.GoalRetention, NewADay: 0, ReviewsADay: 0, MinutesADay: 20,
			},
			want: review.StoppedNoCards,
		},
		{
			what: "retention, with the minutes at zero",
			p: review.Preset{
				Goal: review.GoalRetention, NewADay: 8, ReviewsADay: 45, MinutesADay: 0,
			},
			want: review.StoppedNothing,
		},
		{
			what: "a day ahead, with every other budget at zero",
			p:    review.Preset{Goal: review.GoalDate, By: ahead},
			want: review.StoppedNothing,
		},
		{
			what: "the day it aims at",
			p: review.Preset{
				Goal: review.GoalDate, By: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
			},
			want: review.StoppedNothing,
		},
		{
			what: "a day behind us",
			p: review.Preset{
				Goal: review.GoalDate, By: pastDate,
				MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			},
			want: review.StoppedPastDay,
		},
		{
			what: "a date and no day",
			p: review.Preset{
				Goal: review.GoalDate, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			},
			want: review.StoppedNoDay,
		},
	} {
		if got := one.p.GetOverallStopReason(day, now); got != one.want {
			t.Errorf("%s stops on %q, want %q", one.what, got, one.want)
		}
		if got, want := one.p.IsPaused(day, now), one.want != review.StoppedNothing; got != want {
			t.Errorf("%s is paused %v, want %v", one.what, got, want)
		}
		// A day of the week at the whole of the load stops nothing of its own.
		if got := one.p.GetStopReason(day, now); got != one.want {
			t.Errorf("%s stops today on %q, want %q", one.what, got, one.want)
		}
	}
}

// A day of the week carrying none of the load is a fact about that one day. The
// preset schedules, and it schedules again on the next day that carries some.
func TestADayAtNoLoadStopsTheDayAndNotThePreset(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	thursday := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	if thursday.Weekday() != time.Thursday {
		t.Fatalf("%v is a %v", thursday, thursday.Weekday())
	}

	p := review.Preset{
		Goal: review.GoalMinutes, MinutesADay: 20,
		Load: map[time.Weekday]int{time.Thursday: 0},
	}

	if got := p.GetOverallStopReason(day, thursday); got != review.StoppedNothing {
		t.Errorf("a preset with a light Thursday stops on %q", got)
	}
	if got := p.GetStopReason(day, thursday); got != review.StoppedNoLoad {
		t.Errorf("its Thursday stops on %q, want %q", got, review.StoppedNoLoad)
	}
	if got := p.GetStopReason(day, thursday.AddDate(0, 0, 1)); got != review.StoppedNothing {
		t.Errorf("its Friday stops on %q", got)
	}

	// A preset that schedules nothing at all says so on a light day too, and the
	// day of the week is not what to fix.
	p.MinutesADay = 0
	if got := p.GetStopReason(day, thursday); got != review.StoppedNoMinutes {
		t.Errorf("a paused preset stops today on %q, want %q", got, review.StoppedNoMinutes)
	}
}

// A week no day of which carries any of the load stops the preset itself: there
// is no next day for the cards to be picked up on.
func TestAWeekAtNoLoadStopsThePresetAndNotOneDay(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	pos := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)

	dead := map[time.Weekday]int{}
	for one := time.Sunday; one <= time.Saturday; one++ {
		dead[one] = 0
	}

	for what, p := range map[string]review.Preset{
		"minutes": {Goal: review.GoalMinutes, MinutesADay: 20, Load: dead},
		"retention": {
			Goal: review.GoalRetention, NewADay: 8, ReviewsADay: 45, Load: dead,
		},
		"a date": {
			Goal: review.GoalDate, By: pos.AddDate(0, 0, 30),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45, Load: dead,
		},
	} {
		t.Run(what, func(t *testing.T) {
			if got := p.GetOverallStopReason(day, pos); got != review.StoppedNoWeek {
				t.Errorf("a dead week stops on %q, want %q", got, review.StoppedNoWeek)
			}
			if got := p.GetStopReason(day, pos); got != review.StoppedNoWeek {
				t.Errorf("its day stops on %q, want %q", got, review.StoppedNoWeek)
			}
			for i := range 7 {
				on := pos.AddDate(0, 0, i)
				if got := p.GetStopReason(day, on); got == review.StoppedNoLoad {
					t.Errorf("its %v promises a next day that carries some load", on.Weekday())
				}
			}
		})
	}

	// A week with one loud day is a week of quiet days and not a dead one.
	alive := maps.Clone(dead)
	alive[time.Friday] = review.FullLoad
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, Load: alive}
	if got := p.GetOverallStopReason(day, pos); got != review.StoppedNothing {
		t.Errorf("a week with one loud day stops on %q", got)
	}
	if got := p.GetStopReason(day, pos); got != review.StoppedNoLoad {
		t.Errorf("its Thursday stops on %q, want %q", got, review.StoppedNoLoad)
	}
}

// front is a preset's frontmatter, as the parser hands it over.
func front(t *testing.T, output string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := yaml.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("frontmatter: %v", err)
	}
	return out
}

// A preset says how the decks pointing at it are scheduled.
func TestWhatAPresetReturns(t *testing.T) {
	p, problems := review.ReadPreset(front(t, `
goal: minutes_a_day
minutes_a_day: 20
new_a_day: 8
reviews_a_day: 45
retention: 0.87
load: {sat: 50, sun: 0}
even_load: true
`))

	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.Goal != review.GoalMinutes {
		t.Errorf("goal = %q", p.Goal)
	}
	if p.MinutesADay != 20 || p.NewADay != 8 || p.ReviewsADay != 45 {
		t.Errorf("limits = %d minutes, %d new, %d reviews", p.MinutesADay, p.NewADay, p.ReviewsADay)
	}
	if p.Retention != 0.87 {
		t.Errorf("retention = %g", p.Retention)
	}
	if p.GetShare(time.Saturday) != 0.5 || p.GetShare(time.Sunday) != 0 ||
		p.GetShare(time.Monday) != 1 {
		t.Errorf("load = %v", p.Load)
	}
	if !p.IsEvenLoad {
		t.Error("even load was asked for")
	}
}

// What a day's budget is spent on is read from the file, and a preset saying
// nothing spends it on cards.
func TestWhatABudgetIsSpentOn(t *testing.T) {
	for content, want := range map[string]review.BudgetUnit{
		"":                review.BudgetUnitCards,
		"counts: cards\n": review.BudgetUnitCards,
		"counts: shows\n": review.BudgetUnitShows,
	} {
		p, problems := review.ReadPreset(front(t, content))
		if len(problems) != 0 {
			t.Fatalf("%q: problems = %v", content, problems)
		}
		if p.Counts != want {
			t.Errorf("%q: counts = %q, want %q", content, p.Counts, want)
		}
	}
}

func TestWhatAPresetPreservesUnsaid(t *testing.T) {
	p, problems := review.ReadPreset(front(t, "goal: retention\nretention: 0.95\n"))

	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.NewADay != review.Defaults().NewADay || p.ReviewsADay != review.Defaults().ReviewsADay {
		t.Errorf("limits = %d new, %d reviews", p.NewADay, p.ReviewsADay)
	}
	if p.Retention != 0.95 {
		t.Errorf("retention = %g", p.Retention)
	}
}

// A key that cannot be read is a problem against the note and keeps its
// default.
func TestAKeyThatCannotBeRead(t *testing.T) {
	for content, says := range map[string]string{
		"goal: sideways\n":        "goal",
		"new_a_day: many\n":       "new_a_day",
		"new_a_day: 2.5\n":        "whole numbers",
		"reviews_a_day: 100000\n": "outside",
		"retention: 0.2\n":        "outside",
		"even_load: perhaps\n":    "even_load",
		"load: sat\n":             "load is a day of the week",
		"load: {caturday: 50}\n":  "day of the week",
		"load: {sat: 120}\n":      "outside",
		"load: {sat: 12.5}\n":     "whole per cent",
		"load: {sat: half}\n":     "not a number",
		"goal: by_date\n":         "requires the day",
		"by_date: 30 September\n": "by_date",
		"counts: minutes\n":       "counts",
		"learned: sideways\n":     "learned",
		"interval: 400\n":         "outside",
	} {
		p, problems := review.ReadPreset(front(t, content))
		if len(problems) != 1 || !strings.Contains(problems[0], says) {
			t.Errorf("%q: problems = %v", content, problems)
		}
		if content == "new_a_day: many\n" && p.NewADay != review.Defaults().NewADay {
			t.Errorf("%q: new a day = %d", content, p.NewADay)
		}
	}
}

// A day it aims at is read whether the file quotes it or not.
func TestTheDayItAimsAt(t *testing.T) {
	for _, content := range []string{
		"goal: by_date\nby_date: 2026-09-30\n",
		"goal: by_date\nby_date: \"2026-09-30\"\n",
	} {
		p, problems := review.ReadPreset(front(t, content))
		if len(problems) != 0 {
			t.Fatalf("%q: problems = %v", content, problems)
		}
		if p.By.Format(review.DayLayout) != "2026-09-30" {
			t.Errorf("%q: by = %v", content, p.By)
		}
	}
}

// The share of a day that goes to the debt is read like the other whole
// outside its bounds.
func TestTheBacklogShareIsReadFromTheFile(t *testing.T) {
	if got := review.Defaults().Backlog; got != review.AllBacklog {
		t.Errorf("a preset naming nothing gives the debt %d of its day", got)
	}

	p, problems := review.ReadPreset(front(t, "backlog: 40\n"))
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.Backlog != 40 {
		t.Errorf("backlog = %d, want 40", p.Backlog)
	}

	p, problems = review.ReadPreset(front(t, "backlog: 140\n"))
	if len(problems) != 1 {
		t.Fatalf("a share outside its bounds turned up %v", problems)
	}
	if p.Backlog != review.AllBacklog {
		t.Errorf("a share outside its bounds left %d standing", p.Backlog)
	}
}

// What counts as learned is read from the file, and a preset saying nothing
// learns a review card by the interval it is sent away for.
func TestWhatCountsAsLearnedIsReadFromTheFile(t *testing.T) {
	for content, want := range map[string]review.LearnedRule{
		"":                     review.RuleInterval,
		"learned: interval\n":  review.RuleInterval,
		"learned: retention\n": review.RuleRetention,
	} {
		p, problems := review.ReadPreset(front(t, content))
		if len(problems) != 0 {
			t.Fatalf("%q: problems = %v", content, problems)
		}
		if p.Rule != want {
			t.Errorf("%q: learned = %q, want %q", content, p.Rule, want)
		}
	}

	p, problems := review.ReadPreset(front(t, "interval: 45\n"))
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.Interval != 45 {
		t.Errorf("interval = %d, want 45", p.Interval)
	}
}

// The defaults are written once and read twice: here, and in the window, where
// a person making a preset is shown the numbers it is scheduled by. Both read
// this one corpus, and neither owns it.
const presetCorpus = "../../frontend/testdata/presets.json"

// defaultPreset is a preset naming nothing, in the words a preset file writes
// its values in.
type defaultPreset struct {
	Goal        string         `json:"goal"`
	ByDate      string         `json:"byDate"`
	MinutesADay int            `json:"minutesADay"`
	NewADay     int            `json:"newADay"`
	ReviewsADay int            `json:"reviewsADay"`
	Retention   float64        `json:"retention"`
	Counts      string         `json:"counts"`
	Backlog     int            `json:"backlog"`
	Load        map[string]int `json:"load"`
	IsEvenLoad  bool           `json:"hasEvenLoad"`
	Learned     string         `json:"learned"`
	Interval    int            `json:"interval"`
}

// presetBounds is how far one setting goes, at each end.
type presetBounds struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// presetCorpusFile is what the corpus holds.
type presetCorpusFile struct {
	Defaults        defaultPreset `json:"defaults"`
	FullLoad        int           `json:"fullLoad"`
	RetentionBounds presetBounds  `json:"retentionBounds"`
}

func readPresetCorpus(t *testing.T) presetCorpusFile {
	t.Helper()
	raw, err := os.ReadFile(presetCorpus)
	if err != nil {
		t.Fatal(err)
	}
	var out presetCorpusFile
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAPresetNamingNothingIsScheduledByWhatTheCorpusReturns(t *testing.T) {
	want := readPresetCorpus(t).Defaults

	d := review.Defaults()
	byDate := ""
	if !d.By.IsZero() {
		byDate = d.By.Format(review.DayLayout)
	}
	load := map[string]int{}
	for day, share := range d.Load {
		load[review.DayName(day)] = share
	}
	got := defaultPreset{
		Goal:        string(d.Goal),
		ByDate:      byDate,
		MinutesADay: d.MinutesADay,
		NewADay:     d.NewADay,
		ReviewsADay: d.ReviewsADay,
		Retention:   d.Retention,
		Counts:      string(d.Counts),
		Backlog:     d.Backlog,
		Load:        load,
		IsEvenLoad:  d.IsEvenLoad,
		Learned:     string(d.Rule),
		Interval:    d.Interval,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("the defaults are %+v, want %+v", got, want)
	}
}

func TestAWholeDaysLoadIsWhatTheCorpusReturns(t *testing.T) {
	want := readPresetCorpus(t).FullLoad

	if review.FullLoad != want {
		t.Errorf("a whole day's load is %d, want %d", review.FullLoad, want)
	}
	if review.LoadBounds.Max != float64(want) {
		t.Errorf("a day is loaded up to %g, want %d", review.LoadBounds.Max, want)
	}
}

func TestRetentionGoesAsFarAsTheCorpusReturns(t *testing.T) {
	want := readPresetCorpus(t).RetentionBounds

	if review.RetentionBounds.Min != want.Min || review.RetentionBounds.Max != want.Max {
		t.Errorf("retention runs %g to %g, want %g to %g",
			review.RetentionBounds.Min, review.RetentionBounds.Max, want.Min, want.Max)
	}
}

// lastAnswer is the instant the schedules below were last answered at.
var lastAnswer = time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)

// genSchedule generates where the answers so far have left one review card: how
// long it is sent away for, and how long the scheduler expects it to stay
// and carries a day it is wanted on like any other: what says nobody has
// answered it is that it was never answered, and not that nothing is set on it.
func genSchedule(t *rapid.T) review.Schedule {
	// Whole days as often as the minutes between them: an interval is counted in
	// days, so the day it falls exactly on is the case that decides the rule.
	away := time.Duration(rapid.IntRange(0, 400).Draw(t, "away"))*24*time.Hour +
		time.Duration(rapid.SampledFrom([]int{0, 0, 0, 1, 59, 1439}).Draw(t, "minutes"))*time.Minute
	out := review.Schedule{
		Due:       lastAnswer.Add(away),
		Stability: float64(rapid.IntRange(0, 500).Draw(t, "stability")),
	}
	if rapid.Bool().Draw(t, "seen") {
		out.Last = lastAnswer
	}
	return out
}

// ruleNamed generates a preset's three settings for the rule, over the whole of
// what each may be and past both ends of it: a value a rule cannot hold is what
// a preset written by hand carries.
func ruleNamed(t *rapid.T) review.Preset {
	return review.Preset{
		Rule: review.LearnedRule(rapid.SampledFrom([]string{
			string(review.RuleInterval), string(review.RuleRetention), "", "sometimes",
		}).Draw(t, "rule")),
		Interval:  rapid.IntRange(-5, 400).Draw(t, "interval"),
		Retention: float64(rapid.IntRange(0, 100).Draw(t, "retention")) / 100,
	}
}

// preset says and whenever it is asked.
func TestIsLearned_UnansweredCardIsNeverLearned(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		p := ruleNamed(t)
		s := genSchedule(t)
		s.Last = time.Time{}
		pos := lastAnswer.Add(time.Duration(rapid.IntRange(-1000, 1000).Draw(t, "days")) * 24 * time.Hour)
		if p.IsLearned(s, pos) {
			t.Fatalf("a review card unanswered counts learned under %+v "+
				"at %v, standing at %+v", p, pos, s)
		}
	})
}

// A preset naming a rule it does not have, or a value the rule cannot hold,
// counts by the default rule at its default value. A key nobody wrote leaves
// the default in force, and never a threshold every card passes.
//
// A preset that names no rule counts by the default rule at its default value.
func TestAValueTheRuleCannotHoldCountsByTheDefault(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		p := ruleNamed(t)
		s := genSchedule(t)
		pos := lastAnswer.Add(time.Duration(rapid.IntRange(0, 1000).Draw(t, "days")) * 24 * time.Hour)

		standard := review.Defaults()
		if review.IsKnownRule(p.Rule) {
			standard.Rule = p.Rule
		}
		if review.IntervalBounds.Contains(float64(p.Interval)) {
			standard.Interval = p.Interval
		}
		if review.RetentionBounds.Contains(p.Retention) {
			standard.Retention = p.Retention
		}
		if got, want := p.IsLearned(s, pos), standard.IsLearned(s, pos); got != want {
			t.Fatalf("%+v counts %+v learned as %v at %v, where the rule in "+
				"force is %+v and counts it %v", p, s, got, pos, standard, want)
		}
	})
}

// Under a rule of an interval a review card is learned once it is sent away for
// the preset's interval or longer, so the further away it is sent the more
// presets count it learned: a card learned under one interval is learned under
// every shorter one.
//
// Under learned: interval a card is learned once the interval it is sent away
// for reaches the preset's interval.
func TestALongerIntervalIsLearnedByEveryShorterOne(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		s := genSchedule(t)
		short := rapid.IntRange(1, 365).Draw(t, "shorter")
		long := rapid.IntRange(short, 365).Draw(t, "longer")
		pos := lastAnswer.Add(time.Duration(rapid.IntRange(0, 1000).Draw(t, "days")) * 24 * time.Hour)

		delta := func(days int) review.Preset {
			return review.Preset{Rule: review.RuleInterval, Interval: days}
		}
		if delta(long).IsLearned(s, pos) && !delta(short).IsLearned(s, pos) {
			t.Fatalf("a review card sent away %v counts learned at an interval of "+
				"%d days and not at one of %d", s.Due.Sub(s.Last), long, short)
		}
	})
}

// Under a rule of retention, learned is where the review card stands now and not
// a milestone it passed: the chance of recalling it only falls as the days go
// by, so a card counted learned on a later day was counted learned on every
// earlier one, and a lapse that collapses its stability takes it out of that
// standing.
//
// Learned is a state and not a milestone, asked of where the card stands now.
func TestRecallIsAStateAndNotAMilestone(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		s := genSchedule(t)
		if s.IsNew() {
			return
		}
		p := review.Preset{
			Rule:      review.RuleRetention,
			Retention: float64(rapid.IntRange(70, 99).Draw(t, "retention")) / 100,
		}
		soon := rapid.IntRange(0, 1000).Draw(t, "sooner")
		late := rapid.IntRange(soon, 1000).Draw(t, "later")
		day := func(days int) time.Time {
			return lastAnswer.Add(time.Duration(days) * 24 * time.Hour)
		}
		if p.IsLearned(s, day(late)) && !p.IsLearned(s, day(soon)) {
			t.Fatalf("a review card at a stability of %v counts learned %d days "+
				"out and not %d days out", s.Stability, late, soon)
		}

		lapsed := s
		lapsed.Stability = 0
		if p.IsLearned(lapsed, day(soon)) {
			t.Fatalf("a review card nothing is known about counts learned at %v", p.Retention)
		}
	})
}
