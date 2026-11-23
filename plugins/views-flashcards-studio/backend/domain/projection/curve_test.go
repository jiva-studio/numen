package projection

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

func getRetentionBounds() Bounds { return Bounds{Min: 0.7, Max: 0.99} }

func TestACurveOfMinutesContainsThePresetMinutes(t *testing.T) {
	t.Parallel()
	p := Settings{Goal: GoalMinutes, MinutesADay: 7, Rule: RuleInterval, Interval: 4, Backlog: wholeBacklog}
	got, err := newTestSimulation(5).ComputeCurve(t.Context(), getTestNow(), p, CurveInputs{
		Cards: getOverdueCards(20), UnbegunCount: 5, GetScheduler: nil, VisitPlaces: nil, RetentionBounds: getRetentionBounds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Goal != GoalMinutes || len(got.Grid) != Points || len(got.Points) != Points || !slices.IsSorted(got.Grid) {
		t.Fatalf("the curve of minutes runs over %v", got.Grid)
	}
	if got.Now.Value != 7 || got.Grid[got.Now.Index] != 7 {
		t.Errorf("the preset minutes are at %+v on %v", got.Now, got.Grid)
	}
	for i, one := range got.Points[:got.Suggested.Index] {
		if len(one.Closed) == 0 {
			t.Errorf("place %d closes on nothing, and the suggestion is place %d", i, got.Suggested.Index)
		}
	}
	if got.Suggested.Value != got.Grid[got.Suggested.Index] {
		t.Errorf("the suggestion %+v is off the grid %v", got.Suggested, got.Grid)
	}
}

func TestACurveOfRetentionRunsTheSchedulerOfEachPlace(t *testing.T) {
	t.Parallel()
	var asked []float64
	p := getRetentionSettings(5, 50)
	p.Retention = 0.85
	got, err := newTestSimulation(5).ComputeCurve(t.Context(), getTestNow(), p, CurveInputs{
		Cards: getOverdueCards(10), UnbegunCount: 2, VisitPlaces: nil, RetentionBounds: getRetentionBounds(),
		GetScheduler: func(retention float64) Scheduler {
			asked = append(asked, retention)
			return fakeScheduler{placed: nil}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Grid[0] != 0.7 || got.Grid[Points-1] != 0.99 {
		t.Errorf("the curve of retention runs from %v to %v", got.Grid[0], got.Grid[Points-1])
	}
	if got.Now.Value != 0.85 || got.Grid[got.Now.Index] != 0.85 {
		t.Errorf("the preset retention is at %+v on %v", got.Now, got.Grid)
	}
	if !slices.Equal(asked, got.Grid) {
		t.Errorf("the scheduler was asked for at %v, and the places are %v", asked, got.Grid)
	}
}

func TestACurveOfADateRunsToTwiceItsDay(t *testing.T) {
	t.Parallel()
	open := getTestOpen()
	by := open.AddDate(0, 0, 4)
	p := Settings{Goal: GoalDate, By: by, MinutesADay: 30, Retention: 0.9, Rule: RuleInterval, Interval: 3}
	got, err := newTestSimulation(5).ComputeCurve(t.Context(), getTestNow(), p, CurveInputs{
		Cards: getOverdueCards(1), UnbegunCount: 2, GetScheduler: nil, VisitPlaces: nil, RetentionBounds: getRetentionBounds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Grid, []float64{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("the curve of a date four days off runs over %v", got.Grid)
	}
	if got.Now != (CurvePoint{Index: 3, Value: 4, Day: utcDays{}.GetName(by)}) {
		t.Errorf("the preset date is at %+v", got.Now)
	}
	if first := (utcDays{}).GetName(open.AddDate(0, 0, 1)); got.Days[0] != first {
		t.Errorf("the first place names %s", got.Days[0])
	}

	for _, past := range []time.Time{{}, open.AddDate(0, 0, -1)} {
		p.By = past
		empty, err := newTestSimulation(5).ComputeCurve(t.Context(), getTestNow(), p, CurveInputs{RetentionBounds: getRetentionBounds()})
		if err != nil || len(empty.Grid) != 0 || empty.Now != GetOutsideGrid() {
			t.Errorf("a date of %v draws %+v, %v", past, empty, err)
		}
	}
}

func TestACurveOfADateBeyondTheRangeEndsAtTheRange(t *testing.T) {
	t.Parallel()
	p := Settings{Goal: GoalDate, By: getTestOpen().AddDate(0, 0, MostAhead+5), Rule: RuleInterval, Interval: 3}
	got, err := newTestSimulation(5).ComputeCurve(t.Context(), getTestNow(), p, CurveInputs{
		RetentionBounds: getRetentionBounds(),
		VisitPlaces: func(count int, each func(index int) error) error {
			if err := each(0); err != nil {
				return err
			}
			return each(count - 1)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Grid[len(got.Grid)-1] != MostAhead || got.Now != GetOutsideGrid() {
		t.Errorf("a date beyond the range ends at %v and the preset is at %+v", got.Grid[len(got.Grid)-1], got.Now)
	}
}

func TestACurveReturnsTheErrorOfAPlace(t *testing.T) {
	t.Parallel()
	errPlace := errors.New("place")
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	for _, goal := range []Goal{GoalMinutes, GoalRetention, GoalDate} {
		p := getRetentionSettings(5, 5)
		p.Goal, p.MinutesADay, p.By = goal, 10, getTestOpen().AddDate(0, 0, 4)
		run := newTestSimulation(2)
		if _, err := run.ComputeCurve(t.Context(), getTestNow(), p, CurveInputs{
			RetentionBounds: getRetentionBounds(),
			VisitPlaces:     func(int, func(int) error) error { return errPlace },
		}); !errors.Is(err, errPlace) {
			t.Errorf("%s: a failing place returns %v", goal, err)
		}
		if _, err := run.ComputeCurve(canceled, getTestNow(), p, CurveInputs{RetentionBounds: getRetentionBounds()}); !errors.Is(err, context.Canceled) {
			t.Errorf("%s: a canceled curve returns %v", goal, err)
		}
	}
}

func TestACurveOfADateNamesTheDaysAcrossAClockChange(t *testing.T) {
	t.Parallel()
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	// 02:30 does not exist on 2026-03-08 in this zone: the clock jumps from 02:00 to 03:00.
	sim := newTestSimulation(2)
	sim.Day = zonedDays{starts: 2*time.Hour + 30*time.Minute, in: zone}
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, zone)
	by := time.Date(2026, 3, 9, 12, 0, 0, 0, zone)
	got, err := sim.ComputeCurve(t.Context(), now, Settings{
		Goal: GoalDate, By: by, MinutesADay: 30, Retention: 0.9, Rule: RuleInterval, Interval: 7,
	}, CurveInputs{UnbegunCount: 2, RetentionBounds: getRetentionBounds()})
	if err != nil {
		t.Fatalf("Curve err = %v", err)
	}
	if got.Now.Value != 3 || got.Now.Day != "2026-03-09" {
		t.Fatalf("Now = %+v, want Value 3 and Day 2026-03-09", got.Now)
	}
	for i, grid := range got.Grid {
		want := time.Date(2026, 3, 6+int(grid), 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
		if got.Days[i] != want {
			t.Fatalf("Days[%d] = %s for day %v, want %s", i, got.Days[i], grid, want)
		}
	}
}
