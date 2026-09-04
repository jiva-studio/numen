package flashcards_test

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// A curve works every place of its grid out alongside the others, and the curve
// it comes to is the same curve every time it is asked for.
//
// Run under the race detector this is what holds the places to sharing nothing:
// each run reads the schedules it is given and writes only what it hands back.
// Read without it, what it holds is that the answer does not depend on the order
// the places happen to finish in, which is the order of the cores the machine
// has.
func TestACurveIsTheSameCurveEveryTimeItIsAsked(t *testing.T) {
	t.Parallel()
	s := answering(t, 200)
	curves := s.curves(noon)
	for _, one := range atOnceGoals() {
		t.Run(one.name, func(t *testing.T) {
			first, err := curves.Execute(t.Context(), s.vault, "Sanskrit.md", one.preset)
			if err != nil {
				t.Fatal(err)
			}
			want := drawn(first)
			for again := range 8 {
				got, err := curves.Execute(t.Context(), s.vault, "Sanskrit.md", one.preset)
				if err != nil {
					t.Fatal(err)
				}
				if drawn(got) != want {
					t.Fatalf("asking %d times over gives\n%s\nand the first asking gave\n%s",
						again+2, drawn(got), want)
				}
			}
		})
	}
}

// atOnceGoals is one preset under each goal, so that all three of the loops
// that work a grid out are asked the question.
func atOnceGoals() []struct {
	name   string
	preset review.Preset
} {
	return []struct {
		name   string
		preset review.Preset
	}{
		{"minutes", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.87, Rule: review.RuleInterval, Interval: 21,
			Counts: review.CountsCards, Backlog: 100, EvenLoad: true,
		}},
		{"retention", review.Preset{
			Goal: review.GoalRetention, MinutesADay: 20, NewADay: 6, ReviewsADay: 30,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 21,
			Counts: review.CountsCards, Backlog: 60, EvenLoad: true,
		}},
		{"a date", review.Preset{
			Goal: review.GoalDate, By: noon.AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.CountsCards, EvenLoad: true,
		}},
	}
}

// drawn is a whole curve written out, every scalar and every series of it, so
// that two of them are compared by what they say and not by what they point at.
func drawn(c review.Curve) string {
	return fmt.Sprintf("%v", c)
}
