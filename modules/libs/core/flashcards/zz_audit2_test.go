package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

func TestAuditPlacement(t *testing.T) {
	now := opens(time.Date(2026, 3, 2, 9, 41, 0, 0, time.Local))
	p := history.Defaults()
	p.Goal, p.By = history.GoalDate, now.AddDate(0, 0, 21)
	p.Rule, p.Interval = history.RuleInterval, 21
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 8, 45
	for _, even := range []bool{true, false} {
		q := p
		q.EvenLoad = even
		run := history.Simulation{By: history.NewFSRSAt(q.Retention), Day: ahead, Cost: history.DefaultCost, Days: 22}
		got := ran(t, run, now, q, nil, 40)
		t.Logf("even=%t through21=%.3f short=%d load=%v", even, got.Through[21], got.Short, got.Load)
	}
}
