package flashcards_test

import (
	"maps"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// front is a preset's frontmatter, as the parser hands it over.
func front(t *testing.T, written string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := yaml.Unmarshal([]byte(written), &out); err != nil {
		t.Fatalf("frontmatter: %v", err)
	}
	return out
}

// A preset says how the decks pointing at it are scheduled.
func TestWhatAPresetSays(t *testing.T) {
	p, problems := flashcards.ReadPreset(front(t, `
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
	if p.Goal != flashcards.GoalMinutes {
		t.Errorf("goal = %q", p.Goal)
	}
	if p.MinutesADay != 20 || p.NewADay != 8 || p.ReviewsADay != 45 {
		t.Errorf("limits = %d minutes, %d new, %d reviews", p.MinutesADay, p.NewADay, p.ReviewsADay)
	}
	if p.Retention != 0.87 {
		t.Errorf("retention = %g", p.Retention)
	}
	if p.Share(time.Saturday) != 0.5 || p.Share(time.Sunday) != 0 || p.Share(time.Monday) != 1 {
		t.Errorf("load = %v", p.Load)
	}
	if !p.EvenLoad {
		t.Error("even load was asked for")
	}
}

// What a day's budget is spent on is read from the file, and a preset saying
// nothing spends it on cards.
func TestWhatABudgetIsSpentOn(t *testing.T) {
	for written, want := range map[string]flashcards.Counts{
		"":                flashcards.CountsCards,
		"counts: cards\n": flashcards.CountsCards,
		"counts: shows\n": flashcards.CountsShows,
	} {
		p, problems := flashcards.ReadPreset(front(t, written))
		if len(problems) != 0 {
			t.Fatalf("%q: problems = %v", written, problems)
		}
		if p.Counts != want {
			t.Errorf("%q: counts = %q, want %q", written, p.Counts, want)
		}
	}
}

// A key the file does not carry stands at the default.
func TestWhatAPresetLeavesUnsaid(t *testing.T) {
	p, problems := flashcards.ReadPreset(front(t, "goal: retention\nretention: 0.95\n"))

	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.NewADay != flashcards.Defaults().NewADay || p.ReviewsADay != flashcards.Defaults().ReviewsADay {
		t.Errorf("limits = %d new, %d reviews", p.NewADay, p.ReviewsADay)
	}
	if p.Retention != 0.95 {
		t.Errorf("retention = %g", p.Retention)
	}
}

// A key that cannot be read is a problem against the note and keeps its
// default.
func TestAKeyThatCannotBeRead(t *testing.T) {
	for written, says := range map[string]string{
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
		"goal: by_date\n":         "which day",
		"by_date: 30 September\n": "by_date",
		"counts: minutes\n":       "counts",
		"learned: sideways\n":     "learned",
		"interval: 400\n":         "outside",
	} {
		p, problems := flashcards.ReadPreset(front(t, written))
		if len(problems) != 1 || !strings.Contains(problems[0], says) {
			t.Errorf("%q: problems = %v", written, problems)
		}
		if written == "new_a_day: many\n" && p.NewADay != flashcards.Defaults().NewADay {
			t.Errorf("%q: new a day = %d", written, p.NewADay)
		}
	}
}

// A day it aims at is read whether the file quotes it or not.
func TestTheDayItAimsAt(t *testing.T) {
	for _, written := range []string{
		"goal: by_date\nby_date: 2026-09-30\n",
		"goal: by_date\nby_date: \"2026-09-30\"\n",
	} {
		p, problems := flashcards.ReadPreset(front(t, written))
		if len(problems) != 0 {
			t.Fatalf("%q: problems = %v", written, problems)
		}
		if p.By.Format(flashcards.Named) != "2026-09-30" {
			t.Errorf("%q: by = %v", written, p.By)
		}
	}
}

// A date is a budget: past the day it names, the preset schedules nothing.
//
// The day it names is a whole review day, so the preset schedules through every
// hour of it and stops when the next one opens.
func TestADateIsABudget(t *testing.T) {
	p, _ := flashcards.ReadPreset(front(t, "goal: by_date\nby_date: 2026-09-30\n"))
	day := flashcards.Day{Starts: flashcards.DayStarts, In: time.UTC}

	for _, one := range []struct {
		hour time.Time
		past bool
	}{
		{time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC), false},
		// The small hours of the 30th are still the day before it.
		{time.Date(2026, 9, 30, 2, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC), false},
		// And the small hours of the 1st are still the day it names.
		{time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), true},
	} {
		if got := p.Past(day, one.hour); got != one.past {
			t.Errorf("at %v the day it aims at is past = %v, want %v", one.hour, got, one.past)
		}
		if got := p.Paused(day, one.hour); got != one.past {
			t.Errorf("at %v the preset is paused = %v, want %v", one.hour, got, one.past)
		}
	}
}

// A budget of zero is a pause where the goal names that budget, and every deck
// pointing at the preset stops. A budget the goal does not name is not read.
func TestZeroIsAPauseUnderTheGoalThatNamesIt(t *testing.T) {
	for _, one := range []struct {
		front  string
		paused bool
	}{
		{"goal: retention\nnew_a_day: 0\nreviews_a_day: 0\n", true},
		{"goal: retention\nminutes_a_day: 0\n", false},
		{"goal: minutes_a_day\nminutes_a_day: 0\n", true},
		{"goal: minutes_a_day\nnew_a_day: 0\nreviews_a_day: 0\n", false},
	} {
		p, problems := flashcards.ReadPreset(front(t, one.front))

		if len(problems) != 0 {
			t.Fatalf("problems = %v", problems)
		}
		got := p.Paused(flashcards.Day{Starts: flashcards.DayStarts}, time.Now())
		if got != one.paused {
			t.Errorf("%q is paused %v, want %v", one.front, got, one.paused)
		}
	}
}

// The share of a day that goes to the debt is read like the other whole
// numbers, stands at all of it where the file names none, and is refused
// outside its bounds.
func TestTheBacklogShareIsReadFromTheFile(t *testing.T) {
	if got := flashcards.Defaults().Backlog; got != flashcards.AllBacklog {
		t.Errorf("a preset naming nothing gives the debt %d of its day", got)
	}

	p, problems := flashcards.ReadPreset(front(t, "backlog: 40\n"))
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.Backlog != 40 {
		t.Errorf("backlog = %d, want 40", p.Backlog)
	}

	p, problems = flashcards.ReadPreset(front(t, "backlog: 140\n"))
	if len(problems) != 1 {
		t.Fatalf("a share outside its bounds turned up %v", problems)
	}
	if p.Backlog != flashcards.AllBacklog {
		t.Errorf("a share outside its bounds left %d standing", p.Backlog)
	}
}

// The share of the day that goes to the debt is read where one pot is spent
// between the two, and the goal says so where it says which budget closes its
// day.
//
// A goal of retention keeps a count for each side, so each is held to its own
// and the share decides nothing. A goal of a date carries the whole material by
// its own reckoning.
func TestWhichSettingsAGoalReads(t *testing.T) {
	day := flashcards.Day{Starts: flashcards.DayStarts}
	for _, one := range []struct {
		goal  flashcards.Goal
		reads bool
	}{
		{flashcards.GoalMinutes, true},
		{flashcards.GoalRetention, false},
		{flashcards.GoalDate, false},
	} {
		p := flashcards.Defaults()
		p.Goal, p.By, p.Backlog = one.goal, time.Now().AddDate(0, 0, 30), 40
		admits := p.Admits(day, time.Now(), flashcards.Spent{}, flashcards.Left{})

		if got := admits.Closes.Backlog != ""; got != one.reads {
			t.Errorf("under %s the share is read %v, want %v", one.goal, got, one.reads)
		}
		want := 40
		if !one.reads {
			want = flashcards.AllBacklog
		}
		if admits.Backlog != want {
			t.Errorf("under %s the day gives the debt %d, want %d",
				one.goal, admits.Backlog, want)
		}
	}
}

// What counts as learned is read from the file, and a preset saying nothing
// learns a card face by the interval it is sent away for.
func TestWhatCountsAsLearnedIsReadFromTheFile(t *testing.T) {
	for written, want := range map[string]flashcards.Rule{
		"":                     flashcards.RuleInterval,
		"learned: interval\n":  flashcards.RuleInterval,
		"learned: retention\n": flashcards.RuleRetention,
	} {
		p, problems := flashcards.ReadPreset(front(t, written))
		if len(problems) != 0 {
			t.Fatalf("%q: problems = %v", written, problems)
		}
		if p.Rule != want {
			t.Errorf("%q: learned = %q, want %q", written, p.Rule, want)
		}
	}

	p, problems := flashcards.ReadPreset(front(t, "interval: 45\n"))
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if p.Interval != 45 {
		t.Errorf("interval = %d, want 45", p.Interval)
	}
}

// A card face is learned on one side of its rule's threshold and not on the
// other, and the rule the preset does not name takes no part.
func TestEachRuleOnEitherSideOfItsThreshold(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)
	standing := func(since, away int, stability float64) flashcards.Schedule {
		last := now.AddDate(0, 0, -since)
		return flashcards.Schedule{
			Due: last.AddDate(0, 0, away), Last: last, Reps: 3, Stability: stability,
		}
	}

	p := flashcards.Defaults()
	p.Rule, p.Interval, p.Retention = flashcards.RuleInterval, 21, 0.9

	// An interval of 21 days learns the card sent away for 21 and not the one
	// sent away for 20, whatever the chance of recalling either today.
	if !p.Learned(standing(200, 21, 10), now) {
		t.Error("a card sent away for 21 days is not learned at an interval of 21")
	}
	if p.Learned(standing(0, 20, 90), now) {
		t.Error("a card sent away for 20 days is learned at an interval of 21")
	}
	// Nobody has answered it, so neither rule has anything to read.
	if p.Learned(flashcards.Schedule{}, now) {
		t.Error("a card face nobody has answered is learned")
	}

	p.Rule = flashcards.RuleRetention

	// The same two card faces, under the other rule: what is asked now is the
	// chance of recalling them today, and the intervals take no part.
	if p.Learned(standing(200, 21, 10), now) {
		t.Error("a card 200 days past an answer at a stability of 10 is recalled nine times in ten")
	}
	if !p.Learned(standing(0, 20, 90), now) {
		t.Error("a card answered today is not recalled nine times in ten")
	}
	if p.Learned(flashcards.Schedule{}, now) {
		t.Error("a card face nobody has answered is learned")
	}
}

// A preset naming no rule counts by the default rule at its default value.
//
// A key nobody wrote leaves the default in force. It never leaves a threshold
// every card face passes: a preset that says nothing about what counts as
// learned counts nothing learned that three weeks away would not.
func TestAPresetNamingNoRuleCountsByTheDefault(t *testing.T) {
	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -1)
	// A card face first seen yesterday and put off by sixteen days, and one put
	// off by twenty-one.
	near := flashcards.Schedule{Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16}
	far := flashcards.Schedule{Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21}

	// Every field left empty, which is a preset built from settings that name
	// no rule at all.
	var said flashcards.Preset
	if said.Learned(near, now) {
		t.Error("a card face sixteen days off is learned under a preset naming no rule")
	}
	if !said.Learned(far, now) {
		t.Error("a card face twenty-one days off is not learned at the default interval")
	}

	// A rule named with no value under it reads the default value too.
	named := flashcards.Preset{Rule: flashcards.RuleInterval}
	if named.Learned(near, now) {
		t.Error("a card face sixteen days off is learned under an interval nobody wrote")
	}

	// And the other rule, whose value nobody wrote, holds cards to the default
	// chance of recall.
	faded := flashcards.Schedule{
		Last: now.AddDate(0, 0, -200), Due: now, Reps: 3, Stability: 10,
	}
	chance := flashcards.Preset{Rule: flashcards.RuleRetention}
	if chance.Learned(faded, now) {
		t.Error("a card face two hundred days past its answer is recalled nine times in ten")
	}
	if !chance.Learned(near, now) {
		t.Error("a card face answered yesterday is not recalled nine times in ten")
	}
}

// The mark a schedule cache is filed under carries everything that decides
// which day a card lands on.
//
// A cache worked out under one placing is thrown away whole when the placing
// changes, so a setting that moves a card and not the mark is a cache read back
// under settings it was never worked out for.
func TestThePlacingCarriesEverythingThatMovesACard(t *testing.T) {
	stands := flashcards.Defaults()
	stands.Load = map[time.Weekday]int{time.Saturday: 50}
	was := stands.Placing()

	for what, alter := range map[string]func(p *flashcards.Preset){
		"an even load off":   func(p *flashcards.Preset) { p.EvenLoad = false },
		"a goal of a date":   func(p *flashcards.Preset) { p.Goal = flashcards.GoalDate },
		"a Saturday freed":   func(p *flashcards.Preset) { p.Load = nil },
		"a lighter Saturday": func(p *flashcards.Preset) { p.Load[time.Saturday] = 20 },
	} {
		one := stands
		one.Load = maps.Clone(stands.Load)
		alter(&one)
		if got := one.Placing(); got == was {
			t.Errorf("with %s a card is placed under %q, the same mark as without it",
				what, got)
		}
	}
	// And every day of the week stands in it on its own.
	for day := time.Sunday; day <= time.Saturday; day++ {
		one := stands
		one.Load = maps.Clone(stands.Load)
		one.Load[day] = 30
		if got := one.Placing(); got == was {
			t.Errorf("a %v at thirty is placed under %q, the same mark as one at the "+
				"whole of it", day, got)
		}
	}
}

// What the day has already gone through is off what it still admits, so a
// second sitting takes up where the first left off.
func TestADaysSpendIsOffWhatItStillAdmits(t *testing.T) {
	day := flashcards.Day{Starts: flashcards.DayStarts}
	p := flashcards.Defaults()
	p.Goal, p.MinutesADay = flashcards.GoalMinutes, 1
	p.NewADay, p.ReviewsADay = 20, 20

	fresh := p.Admits(day, time.Now(), flashcards.Spent{}, flashcards.Left{})
	after := p.Admits(day, time.Now(), flashcards.Spent{
		Answered: 3, New: 3, Reviews: 3, Took: 18 * time.Second,
	}, flashcards.Left{})

	if after.Minutes != fresh.Minutes-18*time.Second {
		t.Errorf("a day of %v with 18s gone still admits %v", fresh.Minutes, after.Minutes)
	}
	if after.New != fresh.New-3 || after.Reviews != fresh.Reviews-3 {
		t.Errorf("a day that has begun 3 and reviewed 3 still admits %d new and %d reviews",
			after.New, after.Reviews)
	}
}

// What a preset schedules is the core's answer, over every goal and every
// budget at zero and not.
//
// The budget the goal names decides it, and a budget the goal does not name
// takes no part whatever it holds.
func TestWhatAPresetSchedules(t *testing.T) {
	day := flashcards.Day{Starts: flashcards.DayStarts, In: time.UTC}
	// A Thursday, and the two days a preset may aim at from it.
	now := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	ahead := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	behind := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	for _, one := range []struct {
		what string
		p    flashcards.Preset
		want flashcards.Stopped
	}{
		{
			what: "minutes, and the minutes it names",
			p:    flashcards.Preset{Goal: flashcards.GoalMinutes, MinutesADay: 20},
			want: flashcards.StoppedNothing,
		},
		{
			what: "minutes at zero",
			p: flashcards.Preset{
				Goal: flashcards.GoalMinutes, MinutesADay: 0, NewADay: 8, ReviewsADay: 45,
			},
			want: flashcards.StoppedNoMinutes,
		},
		{
			what: "minutes, with both card counts at zero",
			p: flashcards.Preset{
				Goal: flashcards.GoalMinutes, MinutesADay: 20, NewADay: 0, ReviewsADay: 0,
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "minutes, past a day it does not aim at",
			p: flashcards.Preset{
				Goal: flashcards.GoalMinutes, MinutesADay: 20, By: behind,
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "retention, and both counts",
			p: flashcards.Preset{
				Goal: flashcards.GoalRetention, NewADay: 8, ReviewsADay: 45,
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "retention, with no new cards a day",
			p: flashcards.Preset{
				Goal: flashcards.GoalRetention, NewADay: 0, ReviewsADay: 45,
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "retention, with no reviews a day",
			p: flashcards.Preset{
				Goal: flashcards.GoalRetention, NewADay: 8, ReviewsADay: 0,
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "retention, with both counts at zero",
			p: flashcards.Preset{
				Goal: flashcards.GoalRetention, NewADay: 0, ReviewsADay: 0, MinutesADay: 20,
			},
			want: flashcards.StoppedNoCards,
		},
		{
			what: "retention, with the minutes at zero",
			p: flashcards.Preset{
				Goal: flashcards.GoalRetention, NewADay: 8, ReviewsADay: 45, MinutesADay: 0,
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "a day ahead, with every other budget at zero",
			p:    flashcards.Preset{Goal: flashcards.GoalDate, By: ahead},
			want: flashcards.StoppedNothing,
		},
		{
			what: "the day it aims at",
			p: flashcards.Preset{
				Goal: flashcards.GoalDate, By: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
			},
			want: flashcards.StoppedNothing,
		},
		{
			what: "a day behind us",
			p: flashcards.Preset{
				Goal: flashcards.GoalDate, By: behind,
				MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			},
			want: flashcards.StoppedPastDay,
		},
		{
			what: "a date and no day",
			p: flashcards.Preset{
				Goal: flashcards.GoalDate, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			},
			want: flashcards.StoppedNoDay,
		},
	} {
		if got := one.p.Stops(day, now); got != one.want {
			t.Errorf("%s stops on %q, want %q", one.what, got, one.want)
		}
		if got, want := one.p.Paused(day, now), one.want != flashcards.StoppedNothing; got != want {
			t.Errorf("%s is paused %v, want %v", one.what, got, want)
		}
		// A day of the week at the whole of the load stops nothing of its own.
		if got := one.p.StopsOn(day, now); got != one.want {
			t.Errorf("%s stops today on %q, want %q", one.what, got, one.want)
		}
	}
}

// A day of the week carrying none of the load is a fact about that one day. The
// preset schedules, and it schedules again on the next day that carries some.
func TestADayAtNoLoadStopsTheDayAndNotThePreset(t *testing.T) {
	day := flashcards.Day{Starts: flashcards.DayStarts, In: time.UTC}
	thursday := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	if thursday.Weekday() != time.Thursday {
		t.Fatalf("%v is a %v", thursday, thursday.Weekday())
	}

	p := flashcards.Preset{
		Goal: flashcards.GoalMinutes, MinutesADay: 20,
		Load: map[time.Weekday]int{time.Thursday: 0},
	}

	if got := p.Stops(day, thursday); got != flashcards.StoppedNothing {
		t.Errorf("a preset with a light Thursday stops on %q", got)
	}
	if got := p.StopsOn(day, thursday); got != flashcards.StoppedNoLoad {
		t.Errorf("its Thursday stops on %q, want %q", got, flashcards.StoppedNoLoad)
	}
	if got := p.StopsOn(day, thursday.AddDate(0, 0, 1)); got != flashcards.StoppedNothing {
		t.Errorf("its Friday stops on %q", got)
	}

	// A preset that schedules nothing at all says so on a light day too, and the
	// day of the week is not what to fix.
	p.MinutesADay = 0
	if got := p.StopsOn(day, thursday); got != flashcards.StoppedNoMinutes {
		t.Errorf("a paused preset stops today on %q, want %q", got, flashcards.StoppedNoMinutes)
	}
}
