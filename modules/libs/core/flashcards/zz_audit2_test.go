package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

func TestAuditRetentionRule(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	for _, days := range []int{2, 7, 14, 21, 30} {
		p := history.Defaults()
		p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, days)
		p.Rule, p.Retention = history.RuleRetention, 0.9
		p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45
		run := history.Simulation{
			By: history.NewFSRSAt(p.Retention), Day: ahead, Cost: history.DefaultCost, Days: days + 1,
		}
		got := ran(t, run, now, p, nil, 40)
		t.Logf("by %2d: through=%.3f short=%d load=%v", days, got.Through[days], got.Short, got.Load)
		t.Logf("        through=%v", got.Through)
	}
}
