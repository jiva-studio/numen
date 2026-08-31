package flashcards_test

import (
	"fmt"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

func TestAuditTable(t *testing.T) {
	s := opened(t, studied(40))
	fmt.Printf("%-10s %-6s %-5s | %-8s %-6s %-6s %-7s %-5s %s\n",
		"rule", "even", "days", "through", "short", "met", "enough", "owed", "day")
	for _, rule := range []history.Rule{history.RuleInterval, history.RuleRetention} {
		for _, even := range []bool{true, false} {
			for _, days := range []int{1, 2, 7, 14, 21, 30, 90, 365} {
				p := history.Defaults()
				p.Goal = history.GoalDate
				p.By = noon.AddDate(0, 0, days).Truncate(24 * time.Hour)
				p.Rule, p.Interval, p.Retention = rule, 21, 0.9
				p.EvenLoad = even
				p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45
				got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
				if err != nil {
					t.Fatal(err)
				}
				if len(got.At) == 0 {
					fmt.Printf("%-10s %-6t %-5d | no curve\n", rule, even, days)
					continue
				}
				at := -1
				for i, d := range got.Days {
					if d == got.Now.Day {
						at = i
					}
				}
				if at < 0 {
					fmt.Printf("%-10s %-6t %-5d | the day named is not on the grid (mark at %s)\n",
						rule, even, days, got.Days[got.Now.At])
					continue
				}
				one := got.At[at]
				fmt.Printf("%-10s %-6t %-5d | %-8.3f %-6d %-6t %-7t %-5d %s\n",
					rule, even, days, one.Through, one.Short, one.Met, one.Enough, one.Owed,
					got.Days[at])
			}
		}
	}
}
