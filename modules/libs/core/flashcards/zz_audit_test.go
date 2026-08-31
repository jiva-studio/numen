package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

var auditNoon = time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)
var auditDay = history.Day{Starts: history.DayStarts}

// One card face, begun today, walked by Run; the day it is first learned.
func learnsOn(t *testing.T, p history.Preset, days int) int {
	t.Helper()
	run := history.Simulation{By: history.NewFSRSAt(p.Retention), Day: auditDay, Cost: history.DefaultCost, Days: days}
	got, err := run.Run(t.Context(), auditNoon, p, map[history.CardFace]history.Schedule{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.Through {
		if one >= 1 {
			return i
		}
	}
	return -1
}

func TestAuditRipensAgainstRun(t *testing.T) {
	p := history.Defaults()
	p.Goal = history.GoalMinutes
	p.MinutesADay = 1440
	p.NewADay = 100
	p.ReviewsADay = 1000
	p.Rule, p.Interval = history.RuleInterval, 21
	p.EvenLoad = false
	ripens := history.Ripens(history.NewFSRSAt(p.Retention), auditDay, p, auditNoon)
	t.Logf("interval 21: Ripens=%d Run learns on day %d", ripens, learnsOn(t, p, 60))
	for _, iv := range []int{1, 2, 5, 7, 14, 21, 30, 60} {
		q := p
		q.Interval = iv
		t.Logf("interval %3d: Ripens=%3d Run=%3d", iv,
			history.Ripens(history.NewFSRSAt(q.Retention), auditDay, q, auditNoon), learnsOn(t, q, 400))
	}
}
