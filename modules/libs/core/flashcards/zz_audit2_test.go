package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

func TestAuditDeadDays(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	for _, dead := range []bool{false, true} {
		for _, days := range []int{7, 14, 21, 30} {
			p := history.Defaults()
			p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, days)
			p.Rule, p.Interval = history.RuleInterval, 21
			p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45
			if dead {
				p.Load = map[time.Weekday]int{time.Saturday: 0, time.Sunday: 0}
			}
			run := history.Simulation{
				By: history.NewFSRSAt(p.Retention), Day: ahead,
				Cost: history.DefaultCost, Days: days + 1,
			}
			got := ran(t, run, now, p, nil, 40)
			t.Logf("dead=%t by %2d: through=%.3f short=%d load=%v",
				dead, days, got.Through[days], got.Short, got.Load)
		}
	}
}
