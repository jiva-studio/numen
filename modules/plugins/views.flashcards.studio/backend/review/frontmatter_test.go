package review_test

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

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
		"goal: by_date\n":         "which day",
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
// learns a card face by the interval it is sent away for.
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
