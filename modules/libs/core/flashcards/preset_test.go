package flashcards_test

import (
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
light_days: [sat]
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
	if len(p.LightDays) != 1 || p.LightDays[0] != time.Saturday {
		t.Errorf("light days = %v", p.LightDays)
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
		"goal: sideways\n":         "goal",
		"new_a_day: many\n":        "new_a_day",
		"new_a_day: 2.5\n":         "whole numbers",
		"reviews_a_day: 100000\n":  "outside",
		"retention: 0.2\n":         "outside",
		"even_load: perhaps\n":     "even_load",
		"light_days: sat\n":        "light_days",
		"light_days: [caturday]\n": "day of the week",
		"goal: by_date\n":          "which day",
		"by_date: 30 September\n":  "by_date",
		"counts: minutes\n":        "counts",
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

// A goal of a date reads no share, and says so where it says which of the
// budgets closes its day.
func TestWhichSettingsAGoalReads(t *testing.T) {
	day := flashcards.Day{Starts: flashcards.DayStarts}
	for _, one := range []struct {
		goal  flashcards.Goal
		reads bool
	}{
		{flashcards.GoalMinutes, true},
		{flashcards.GoalRetention, true},
		{flashcards.GoalDate, false},
	} {
		p := flashcards.Defaults()
		p.Goal, p.By, p.Backlog = one.goal, time.Now().AddDate(0, 0, 30), 40
		admits := p.Admits(day, time.Now(), flashcards.Spent{}, 0)

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
