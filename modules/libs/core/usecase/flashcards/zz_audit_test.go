package flashcards_test

import (
	"fmt"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// The table asked for: Through on the named day and Short, both rules, both
// even loads, at a spread of dates.
func TestAuditTable(t *testing.T) {
	s := opened(t, studied(40))
	fmt.Printf("%-10s %-6s %-5s | %-8s %-6s | %-8s %-6s\n",
		"rule", "even", "days", "through", "short", "curveDay", "atDay")
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
					fmt.Printf("%-10s %-6t %-5d | %-8s %-6s |\n", rule, even, days, "-", "-")
					continue
				}
				at := -1
				for i, g := range got.Grid {
					if int(g) == days {
						at = i
					}
				}
				if at < 0 {
					at = got.Now.At
				}
				one := got.At[at]
				fmt.Printf("%-10s %-6t %-5d | %-8.3f %-6d | %-8s %-8s met=%t enough=%t owed=%d\n",
					rule, even, days, one.Through, one.Short,
					got.Days[at], got.Now.Day, one.Met, one.Enough, one.Owed)
			}
		}
	}
	_ = flashcards.Nowhere
}
