package wire

import (
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// Settings that name no rule count by the default rule at its default value.
//
// They are what a window written before the key existed sends, and what a
// preset written before it holds. A key nobody wrote leaves the default in
// force: it never leaves a threshold every card face passes.
func TestSettingsThatNameNoRuleCountByTheDefault(t *testing.T) {
	old := &v1.Settings{
		Goal:        v1.Goal_GOAL_RETENTION,
		Counts:      v1.Counts_COUNTS_CARDS,
		MinutesADay: 10,
		NewADay:     12,
		ReviewsADay: 5,
		Retention:   0.8,
		Backlog:     68,
		EvenLoad:    true,
	}
	p, err := SettingsIn(old)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	last := now.AddDate(0, 0, -1)
	// A card face first seen yesterday and put off by sixteen days, which is
	// under the default interval of twenty-one.
	near := history.Schedule{Last: last, Due: last.AddDate(0, 0, 16), Reps: 1, Stability: 16}
	if p.Learned(near, now) {
		t.Error("a card face sixteen days off is learned under settings naming no rule")
	}
	far := history.Schedule{Last: last, Due: last.AddDate(0, 0, 21), Reps: 1, Stability: 21}
	if !p.Learned(far, now) {
		t.Error("a card face twenty-one days off is not learned at the default interval")
	}
}

// A place with no day to name carries none, and a place whose horizon ended
// first carries the day it never reached.
func TestAPlaceWithNoDayToNameCarriesNone(t *testing.T) {
	got := CurveOf(flashcards.Curve{
		Goal: history.GoalRetention,
		Grid: []float64{0.9},
		At: []flashcards.Point{
			{Learns: history.LearnsUnasked},
			{Learns: history.NeverLearns},
			{Learns: 12},
		},
		Now: flashcards.Nowhere, Suggested: flashcards.Nowhere,
	})

	if at := got.GetAt()[0]; at.Learns != nil {
		t.Errorf("a place with no day to name carries %d", at.GetLearns())
	}
	if at := got.GetAt()[1]; at.Learns == nil || at.GetLearns() != history.NeverLearns {
		t.Errorf("a place whose horizon ended first carries %+v", at.Learns)
	}
	if at := got.GetAt()[2]; at.Learns == nil || at.GetLearns() != 12 {
		t.Errorf("a place learned on day 12 carries %+v", at.Learns)
	}
}
