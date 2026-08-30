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
		if p.By.Format(flashcards.DayFormat) != "2026-09-30" {
			t.Errorf("%q: by = %v", written, p.By)
		}
	}
}

// A date is a budget: past the day it names, the preset schedules nothing.
func TestADateIsABudget(t *testing.T) {
	p, _ := flashcards.ReadPreset(front(t, "goal: by_date\nby_date: 2026-09-30\n"))

	before := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	after := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	if p.Spent(before) || p.Paused(before) {
		t.Error("the day it aims at is still ahead")
	}
	if !p.Spent(after) || !p.Paused(after) {
		t.Error("the day it aims at has passed")
	}
}

// Limits of zero are a pause, and every deck pointing at the preset stops.
func TestZeroIsAPause(t *testing.T) {
	p, problems := flashcards.ReadPreset(front(t, "new_a_day: 0\nreviews_a_day: 0\n"))

	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if !p.Paused(time.Now()) {
		t.Error("a preset holding no cards a day schedules nothing")
	}
}
