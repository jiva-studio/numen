package projection

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

func TestAliasesAndGridHelpers(t *testing.T) {
	t.Parallel()
	bounds := GetRetentionBounds()
	if bounds.Min != 0.7 || bounds.Max != 0.99 {
		t.Fatalf("GetRetentionBounds() = %+v, want [0.7, 0.99]", bounds)
	}
	if got := GetFullRecall(Schedule{}, time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)); got != 1 {
		t.Fatalf("GetFullRecall() = %v, want 1", got)
	}

	if got := abs(-5); got != 5 {
		t.Fatalf("abs(-5) = %d, want 5", got)
	}
	if got := abs(-1); got != 1 {
		t.Fatalf("abs(-1) = %d, want 1", got)
	}
	if got := abs(0); got != 0 {
		t.Fatalf("abs(0) = %d, want 0", got)
	}
	if got := abs(5); got != 5 {
		t.Fatalf("abs(5) = %d, want 5", got)
	}

	if got := pickEvenlySpacedDays(1, 1); !slices.Equal(got, []int{0}) {
		t.Fatalf("pickEvenlySpacedDays(1, 1) = %v, want [0]", got)
	}
	if got := pickEvenlySpacedDays(3, 5); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("pickEvenlySpacedDays(3, 5) = %v, want [0 1 2]", got)
	}
	if got := pickEvenlySpacedDays(5, 5); !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("pickEvenlySpacedDays(5, 5) = %v, want [0 1 2 3 4]", got)
	}
	if got := pickEvenlySpacedDays(10, 4); !slices.Equal(got, []int{0, 3, 6, 9}) {
		t.Fatalf("pickEvenlySpacedDays(10, 4) = %v, want [0 3 6 9]", got)
	}

	// setNearestStep boundary and interior cases.
	if got := setNearestStep([]int{-5, 2, 5}, -1); !slices.Equal(got, []int{-5, 2, 5}) {
		t.Fatalf("setNearestStep pos=-1 = %v, want [-5 2 5]", got)
	}
	if got := setNearestStep([]int{-5, 2, 5}, 0); !slices.Equal(got, []int{-5, 0, 5}) {
		t.Fatalf("setNearestStep pos=0 = %v, want [-5 0 5]", got)
	}
	if got := setNearestStep([]int{0, 10}, 5); !slices.Equal(got, []int{0, 10}) {
		t.Fatalf("setNearestStep len=2 = %v, want [0 10]", got)
	}
	if got := setNearestStep([]int{0, 4, 10}, 5); !slices.Equal(got, []int{0, 5, 10}) {
		t.Fatalf("setNearestStep len=3 = %v, want [0 5 10]", got)
	}
	if got := setNearestStep([]int{0, 2, 6, 10, 14}, 3); !slices.Equal(got, []int{0, 3, 6, 10, 14}) {
		t.Fatalf("setNearestStep index 1 = %v, want [0 3 6 10 14]", got)
	}
	if got := setNearestStep([]int{0, 2, 6, 10, 14}, 9); !slices.Equal(got, []int{0, 2, 6, 9, 14}) {
		t.Fatalf("setNearestStep index 3 = %v, want [0 2 6 9 14]", got)
	}
	if got := setNearestStep([]int{0, 2, 6, 10}, 4); !slices.Equal(got, []int{0, 4, 6, 10}) {
		t.Fatalf("setNearestStep tie = %v, want [0 4 6 10]", got)
	}
	if got := setNearestStep([]int{2, 5, 10}, 2); !slices.Equal(got, []int{2, 5, 10}) {
		t.Fatalf("setNearestStep pos==first = %v, want [2 5 10]", got)
	}
	if got := setNearestStep([]int{2, 5, 10}, 10); !slices.Equal(got, []int{2, 5, 10}) {
		t.Fatalf("setNearestStep pos==last = %v, want [2 5 10]", got)
	}

	// snap bounds and tie-breaking.
	grid := []float64{10, 20, 30}
	snap(grid, 99, -1, 2)
	snap(grid, 99, 0, 3)
	snap(grid, 99, 2, 1)
	if !slices.Equal(grid, []float64{10, 20, 30}) {
		t.Fatalf("snap out-of-bounds modified grid: %v", grid)
	}
	snap(grid, 25, 1, 1)
	if !slices.Equal(grid, []float64{10, 25, 30}) {
		t.Fatalf("snap first==last = %v, want [10 25 30]", grid)
	}
	grid = []float64{10, 20, 30}
	snap(grid, 15, 0, 2)
	if !slices.Equal(grid, []float64{15, 20, 30}) {
		t.Fatalf("snap tie = %v, want [15 20 30]", grid)
	}
	grid = []float64{10, 20, 30}
	snap(grid, 28, 0, 2)
	if !slices.Equal(grid, []float64{10, 20, 28}) {
		t.Fatalf("snap last index = %v, want [10 20 28]", grid)
	}
}

func TestProjectionMethodsAndDateSuggestions(t *testing.T) {
	t.Parallel()
	p := Projection{
		ReviewCards: 10,
		Short:       2,
		Spent:       []time.Duration{6 * time.Minute, 12 * time.Minute, 30 * time.Minute},
		Admitted:    []bool{false, true, true},
		Reviewed:    []int{9, 4, 5},
		Closed:      []BudgetNames{{ClosedPaused}, {}, {ClosedMinutes}},
		Backlog:     []int{3, 2, 1},
		Through:     []float64{0.79, 0.80, 0.85},
	}
	if got := p.minutesADay(0); got != 0 {
		t.Fatalf("minutesADay(0) = %v, want 0", got)
	}
	if got := p.minutesADay(1); got != 12 {
		t.Fatalf("minutesADay(1) = %v, want 12", got)
	}
	if got := p.minutesADay(2); got != 21 {
		t.Fatalf("minutesADay(2) = %v, want 21", got)
	}

	if !(Projection{ReviewCards: 0}).isAllLearned(0) {
		t.Fatal("expected ReviewCards==0 to be all learned")
	}
	if p.isAllLearned(0) {
		t.Fatal("expected day 0 (0.79 < 0.80) not to be all learned")
	}
	if !p.isAllLearned(1) {
		t.Fatal("expected day 1 (0.80 >= 0.80) to be all learned")
	}
	if !p.isAllLearned(2) {
		t.Fatal("expected day 2 (0.85 >= 0.80) to be all learned")
	}

	if day, ok := p.Session(); !ok || day != 1 {
		t.Fatalf("Session() = (%d, %v), want (1, true)", day, ok)
	}
	pausedProj := Projection{
		Days:        1,
		ReviewsADay: 7,
		MinutesADay: 15,
		Spent:       []time.Duration{5 * time.Minute},
		Reviewed:    []int{3},
		Admitted:    []bool{false},
		Closed:      []BudgetNames{{}},
		Backlog:     []int{4},
		Through:     []float64{0.5},
	}
	if day, ok := pausedProj.Session(); ok || day != 0 {
		t.Fatalf("paused Session() = (%d, %v), want (0, false)", day, ok)
	}
	if got := getSessionClosed(pausedProj); !slices.Equal(got, BudgetNames{ClosedPaused}) {
		t.Fatalf("getSessionClosed(paused) = %v, want [paused]", got)
	}
	if got := session(pausedProj); got.Reviews != 7 || got.Minutes != 15 {
		t.Fatalf("session(paused) = %+v, want Reviews=7 Minutes=15", got)
	}
	if got := getSessionMinutes(pausedProj); got != 0 {
		t.Fatalf("getSessionMinutes(paused) = %v, want 0", got)
	}
	if got := createDatePoint(pausedProj, 0); got.Reviews != 0 || !slices.Equal(got.Closed, BudgetNames{ClosedPaused}) {
		t.Fatalf("createDatePoint(paused) = %+v, want Reviews=0 Closed=[paused]", got)
	}

	// getNamedOffset boundaries.
	d := Day{Starts: review.DayStartTime, In: time.UTC}
	open := d.GetStart(time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC))
	if got := getNamedOffset(d, open, d.GetName(open)); got != 0 {
		t.Fatalf("getNamedOffset(0) = %d, want 0", got)
	}
	if got := getNamedOffset(d, open, d.GetName(open.AddDate(0, 0, 5))); got != 5 {
		t.Fatalf("getNamedOffset(5) = %d, want 5", got)
	}
	if got := getNamedOffset(d, open, d.GetName(open.AddDate(0, 0, MostAhead))); got != MostAhead {
		t.Fatalf("getNamedOffset(MostAhead) = %d, want %d", got, MostAhead)
	}
	if got := getNamedOffset(d, open, d.GetName(open.AddDate(0, 0, MostAhead+1))); got != GetOutsideGrid().Index {
		t.Fatalf("getNamedOffset(MostAhead+1) = %d, want %d", got, GetOutsideGrid().Index)
	}

	// findSuggestedDate cases.
	pts := []Point{
		{Short: 1, IsEnough: true, Minutes: 10},
		{Short: 0, IsEnough: false, Minutes: 10},
		{Short: 0, IsEnough: true, Minutes: 25},
		{Short: 0, IsEnough: true, Minutes: 20},
	}
	grid := []float64{1, 2, 3, 4}
	days := []string{"d1", "d2", "d3", "d4"}
	if got := findSuggestedDate(pts, grid, days, 20); got != (CurvePoint{Index: 3, Value: 4, Day: "d4"}) {
		t.Fatalf("findSuggestedDate(20) = %+v, want index 3", got)
	}
	if got := findSuggestedDate(pts, grid, days, 0); got != (CurvePoint{Index: 2, Value: 3, Day: "d3"}) {
		t.Fatalf("findSuggestedDate(0) = %+v, want index 2", got)
	}
	if got := findSuggestedDate(pts, grid, days, 5); got != (CurvePoint{Index: 2, Value: 3, Day: "d3"}) {
		t.Fatalf("findSuggestedDate(5 fallback) = %+v, want index 2", got)
	}
	if got := findSuggestedDate(pts[:2], grid[:2], days[:2], 20); got != GetOutsideGrid() {
		t.Fatalf("findSuggestedDate(none learned) = %+v, want OutsideGrid", got)
	}
	ptsMin1 := []Point{
		{Short: 0, IsEnough: true, Minutes: 5},
		{Short: 0, IsEnough: true, Minutes: 1},
	}
	if got := findSuggestedDate(ptsMin1, []float64{1, 2}, []string{"d1", "d2"}, 1); got.Index != 1 {
		t.Fatalf("findSuggestedDate(1) = %+v, want index 1", got)
	}
}

type cancelAfterCallsScheduler struct {
	inner     Scheduler
	remaining int
	cancel    context.CancelFunc
}

func (s *cancelAfterCallsScheduler) GetName() string { return s.inner.GetName() }

func (s *cancelAfterCallsScheduler) IsSpaced(c Schedule) bool { return s.inner.IsSpaced(c) }

func (s *cancelAfterCallsScheduler) Next(c Schedule, at time.Time, r Rating) Schedule {
	s.remaining--
	if s.remaining == 0 {
		s.cancel()
	}
	return s.inner.Next(c, at, r)
}

func (s *cancelAfterCallsScheduler) Endings(c Schedule, at time.Time) (good, again Schedule) {
	s.remaining--
	if s.remaining == 0 {
		s.cancel()
	}
	return s.inner.Endings(c, at)
}

type multiDayNewScheduler struct {
	inner Scheduler
}

func (s multiDayNewScheduler) GetName() string { return "multi-day" }

func (s multiDayNewScheduler) IsSpaced(c Schedule) bool { return s.inner.IsSpaced(c) }

func (s multiDayNewScheduler) Next(c Schedule, at time.Time, r Rating) Schedule {
	if c.IsNew() {
		return Schedule{
			Last:       at,
			Due:        at.AddDate(0, 0, 5),
			Reps:       1,
			Stability:  10,
			Difficulty: 5,
			Phase:      2,
		}
	}
	return s.inner.Next(c, at, r)
}

func (s multiDayNewScheduler) Endings(c Schedule, at time.Time) (good, again Schedule) {
	return Schedule{
			Last:       at,
			Due:        at.AddDate(0, 0, 10),
			Reps:       c.Reps + 1,
			Stability:  20,
			Difficulty: 4,
			Phase:      2,
		}, Schedule{
			Last:       at,
			Due:        at.AddDate(0, 0, 4),
			Reps:       c.Reps + 1,
			Lapses:     c.Lapses + 1,
			Stability:  2,
			Difficulty: 7,
			Phase:      2,
		}
}

func TestCurveAndSimulationEdgeCases(t *testing.T) {
	t.Parallel()
	d := Day{Starts: review.DayStartTime, In: time.UTC}
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	open := d.GetStart(now)
	fsrs := NewFSRSAt(0.9)

	// createCurve initializes Goal, Now, Suggested, and zero counts.
	emptyCurve := createCurve(GoalMinutes)
	if emptyCurve.Goal != GoalMinutes || emptyCurve.Now != GetOutsideGrid() || emptyCurve.Suggested != GetOutsideGrid() ||
		emptyCurve.Decks != 0 || emptyCurve.Cards != 0 || emptyCurve.Overdue != 0 || emptyCurve.Unbegun != 0 {
		t.Fatalf("createCurve = %+v", emptyCurve)
	}

	schedules := make(map[ReviewCardID]Schedule, 30)
	for i := range 30 {
		id := ReviewCardID{Card: string(rune('a' + i)), Template: "T"}
		schedules[id] = Schedule{
			Last:       open.AddDate(0, 0, -20),
			Due:        open.AddDate(0, 0, -5),
			Reps:       2,
			Stability:  10,
			Difficulty: 5,
			Phase:      2,
		}
	}

	sim := Simulation{
		By:   fsrs,
		Day:  d,
		Cost: AnswerCost{New: 30 * time.Second, Review: 2 * time.Minute, ShouldReadNew: true, ShouldReadReview: true},
		Days: 2,
	}

	// 1. Minutes curve: snap at index 0 (MinutesADay=1), snap at index 24 (MinutesADay=475), and all points closed vs some open.
	pMinLow := Preset{
		Goal:        GoalMinutes,
		MinutesADay: 1,
		NewADay:     10,
		ReviewsADay: 50,
		Retention:   0.9,
		Rule:        RuleInterval,
		Interval:    21,
	}
	gotLow, err := sim.Curve(t.Context(), now, pMinLow, CurveInputs{Schedules: schedules, UnbegunCount: 5})
	if err != nil {
		t.Fatalf("Curve(GoalMinutes low) err = %v", err)
	}
	if gotLow.Grid[0] != 1 || gotLow.Now != (CurvePoint{Index: 0, Value: 1, Day: ""}) {
		t.Fatalf("Curve(GoalMinutes low) Grid[0]=%v, Now=%+v", gotLow.Grid[0], gotLow.Now)
	}
	if gotLow.Grid[len(gotLow.Grid)-1] <= LeastCeiling {
		t.Fatalf("expected free load to raise upper bound above LeastCeiling, got %v", gotLow.Grid[len(gotLow.Grid)-1])
	}
	if gotLow.Suggested.Index <= 0 || gotLow.Suggested.Index >= len(gotLow.Grid)-1 {
		t.Fatalf("expected interior Suggested on low curve, got %+v", gotLow.Suggested)
	}
	if gotLow.Points[0].Retained <= 0 || gotLow.Points[0].Learns == 0 {
		t.Fatalf("expected Retained>0 and non-zero Learns on Points[0], got %+v", gotLow.Points[0])
	}

	// Default goal falls back to minutes; high MinutesADay=1435 snaps last grid index (24); 3000 unbegun cards close every point by minutes so Suggested stays at index 24.
	pMinHigh := Preset{
		Goal:        "",
		MinutesADay: 1435,
		NewADay:     10,
		ReviewsADay: 50,
		Retention:   0.9,
		Rule:        RuleInterval,
		Interval:    21,
	}
	gotHigh, err := sim.Curve(t.Context(), now, pMinHigh, CurveInputs{Schedules: schedules, UnbegunCount: 3000})
	if err != nil {
		t.Fatalf("Curve(GoalMinutes high) err = %v", err)
	}
	if gotHigh.Grid[len(gotHigh.Grid)-1] != 1435 || gotHigh.Now.Index != len(gotHigh.Grid)-1 {
		t.Fatalf("expected Grid[24]=1435 and Now.Index=24, got Grid[24]=%v Now=%+v", gotHigh.Grid[len(gotHigh.Grid)-1], gotHigh.Now)
	}
	if gotHigh.Suggested != (CurvePoint{Index: len(gotHigh.Grid) - 1, Value: 1435, Day: ""}) {
		t.Fatalf("expected Suggested at last index when all points closed, got %+v", gotHigh.Suggested)
	}

	// 2. Retention curve: snap bounds 1..len(Grid)-2 (0.701, 0.712, 0.978, 0.988).
	for _, tc := range []struct {
		ret       float64
		wantIndex int
	}{
		{0.701, 1},
		{0.712, 1},
		{0.978, Points - 2},
		{0.988, Points - 2},
	} {
		pRet := Preset{
			Goal:        GoalRetention,
			Retention:   tc.ret,
			MinutesADay: 60,
			NewADay:     10,
			ReviewsADay: 50,
			Rule:        RuleInterval,
			Interval:    21,
		}
		gotRet, err := sim.Curve(t.Context(), now, pRet, CurveInputs{Schedules: schedules, UnbegunCount: 2})
		if err != nil {
			t.Fatalf("Curve(GoalRetention %v) err = %v", tc.ret, err)
		}
		if gotRet.Grid[0] != 0.7 || gotRet.Grid[Points-1] != 0.99 {
			t.Fatalf("retention endpoints displaced for %v: [%v, %v]", tc.ret, gotRet.Grid[0], gotRet.Grid[Points-1])
		}
		if gotRet.Grid[tc.wantIndex] != tc.ret || gotRet.Now != (CurvePoint{Index: tc.wantIndex, Value: tc.ret, Day: ""}) {
			t.Fatalf("retention %v: Grid[%d]=%v, Now=%+v", tc.ret, tc.wantIndex, gotRet.Grid[tc.wantIndex], gotRet.Now)
		}
		if gotRet.Points[0].Retained <= 0 || gotRet.Points[Points-1].Retained <= gotRet.Points[0].Retained {
			t.Fatalf("retention points invalid: first=%+v last=%+v", gotRet.Points[0], gotRet.Points[Points-1])
		}
	}

	// 3. Date curve: zero By, past By, today By (named==0), normal By (named==4), and beyond MostAhead By (named<0).
	if gotZero, err := sim.Curve(t.Context(), now, Preset{Goal: GoalDate}, CurveInputs{}); err != nil || len(gotZero.Grid) != 0 {
		t.Fatalf("Curve(GoalDate zero) = %+v, %v", gotZero, err)
	}
	if gotPast, err := sim.Curve(t.Context(), now, Preset{Goal: GoalDate, By: open.AddDate(0, 0, -1)}, CurveInputs{}); err != nil || len(gotPast.Grid) != 0 {
		t.Fatalf("Curve(GoalDate past) = %+v, %v", gotPast, err)
	}
	gotToday, err := sim.Curve(t.Context(), now, Preset{
		Goal: GoalDate, By: open, MinutesADay: 30, Retention: 0.9, Rule: RuleInterval, Interval: 7,
	}, CurveInputs{})
	if err != nil {
		t.Fatalf("Curve(GoalDate today) err = %v", err)
	}
	if len(gotToday.Grid) != 1 || gotToday.Grid[0] != 1 || gotToday.Now.Day != d.GetName(open) || gotToday.Now.Value != 0 {
		t.Fatalf("Curve(GoalDate today) = %+v", gotToday)
	}

	target4 := open.AddDate(0, 0, 4)
	gotDate, err := sim.Curve(t.Context(), now, Preset{
		Goal: GoalDate, By: target4, MinutesADay: 30, Retention: 0.9, Rule: RuleInterval, Interval: 3,
	}, CurveInputs{
		Schedules:    map[ReviewCardID]Schedule{{Card: "a", Template: "T"}: schedules[ReviewCardID{Card: "a", Template: "T"}]},
		UnbegunCount: 2,
	})
	if err != nil {
		t.Fatalf("Curve(GoalDate +4d) err = %v", err)
	}
	if gotDate.Grid[len(gotDate.Grid)-1] != 8 || gotDate.Now != (CurvePoint{Index: 3, Value: 4, Day: d.GetName(target4)}) {
		t.Fatalf("Curve(GoalDate +4d) Grid=%v Now=%+v", gotDate.Grid, gotDate.Now)
	}
	if gotDate.Suggested.Index < 0 {
		t.Fatalf("expected Suggested on +4d curve, got %+v", gotDate.Suggested)
	}

	// Beyond MostAhead: use custom Places that only runs index 0 and len-1 so test stays fast.
	farBy := open.AddDate(0, 0, MostAhead+5)
	gotFar, err := sim.Curve(t.Context(), now, Preset{
		Goal: GoalDate, By: farBy, MinutesADay: 30, Retention: 0.9, Rule: RuleInterval, Interval: 7,
	}, CurveInputs{
		Places: func(count int, each func(index int) error) error {
			if err := each(0); err != nil {
				return err
			}
			return each(count - 1)
		},
	})
	if err != nil {
		t.Fatalf("Curve(GoalDate far) err = %v", err)
	}
	if gotFar.Grid[len(gotFar.Grid)-1] != MostAhead || gotFar.Now != GetOutsideGrid() {
		t.Fatalf("Curve(GoalDate far) last=%v Now=%+v", gotFar.Grid[len(gotFar.Grid)-1], gotFar.Now)
	}

	// 4. Error propagation in minutes, retention, date, and getDaysToLearnWithoutForgetting.
	canceledCtx, cancelNow := context.WithCancel(t.Context())
	cancelNow()
	if _, err := sim.Curve(canceledCtx, now, pMinLow, CurveInputs{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled on minutes initial Run, got %v", err)
	}
	if _, err := sim.Curve(canceledCtx, now, Preset{Goal: GoalRetention, Retention: 0.9}, CurveInputs{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled on retention Run, got %v", err)
	}
	if _, err := sim.Curve(canceledCtx, now, Preset{Goal: GoalDate, By: target4}, CurveInputs{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled on date Run, got %v", err)
	}

	// Cancel during free load Run so first grid point's Run fails (line 142).
	oneCard := map[ReviewCardID]Schedule{
		{Card: "c1", Template: "T"}: schedules[ReviewCardID{Card: "a", Template: "T"}],
	}
	sim1Day := Simulation{By: fsrs, Day: d, Cost: review.GetDefaultCost(), Days: 1}
	ctxMinPlace, cancelMinPlace := context.WithCancel(t.Context())
	sim1Day.By = &cancelAfterCallsScheduler{inner: fsrs, remaining: 1, cancel: cancelMinPlace}
	if _, err := sim1Day.Curve(ctxMinPlace, now, pMinLow, CurveInputs{Schedules: oneCard}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled on minutes place Run, got %v", err)
	}

	// Cancel during first grid point's Run so getDaysToLearnWithoutForgetting fails in minutes (line 146 & 328).
	ctxMinLearn, cancelMinLearn := context.WithCancel(t.Context())
	sim1Day.By = &cancelAfterCallsScheduler{inner: fsrs, remaining: 2, cancel: cancelMinLearn}
	if _, err := sim1Day.Curve(ctxMinLearn, now, pMinLow, CurveInputs{Schedules: oneCard}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled on minutes getDaysToLearnWithoutForgetting, got %v", err)
	}

	// Cancel during first grid point's Run in retention so getDaysToLearnWithoutForgetting fails in retention (line 194).
	ctxRetLearn, cancelRetLearn := context.WithCancel(t.Context())
	if _, err := sim1Day.Curve(ctxRetLearn, now, Preset{
		Goal: GoalRetention, Retention: 0.9, MinutesADay: 60, ReviewsADay: 10,
	}, CurveInputs{
		Schedules: oneCard,
		Scheduler: func(float64) Scheduler {
			return &cancelAfterCallsScheduler{inner: fsrs, remaining: 1, cancel: cancelRetLearn}
		},
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled on retention getDaysToLearnWithoutForgetting, got %v", err)
	}

	// 5. Simulation.go edge cases: GetDaysToLearn > 0, short() with future due card and unreviewed==0 vs >0,
	// Stability==0 floor, AssignDueDay on new card with EvenLoad, Endings weighted average, and 1-day ReviewsADay/MinutesADay.
	if daysToLearn := GetDaysToLearn(fsrs, d, Preset{Rule: RuleInterval, Interval: 21, Retention: 0.9}, now); daysToLearn <= 0 || daysToLearn == NeverLearns {
		t.Fatalf("GetDaysToLearn = %d, want > 0", daysToLearn)
	}

	// short() when Goal != GoalDate or By.IsZero() is 0.
	if got := sim.short(Preset{Goal: GoalMinutes, By: target4}, []Schedule{{Due: open}}, 5, open); got != 0 {
		t.Fatalf("short(GoalMinutes) = %d, want 0", got)
	}
	if got := sim.short(Preset{Goal: GoalDate}, []Schedule{{Due: open}}, 5, open); got != 0 {
		t.Fatalf("short(By.IsZero) = %d, want 0", got)
	}
	// short() with a card due in 2 days that cannot reach a 21-day interval by day 4, and unreviewed=0 vs unreviewed=3.
	futureCard := Schedule{
		Last: open.AddDate(0, 0, -1), Due: open.AddDate(0, 0, 2), Reps: 1, Stability: 2, Difficulty: 5, Phase: 2,
	}
	pShort := Preset{Goal: GoalDate, By: open.AddDate(0, 0, 3), Rule: RuleInterval, Interval: 30, Retention: 0.9}
	if got := sim.short(pShort, []Schedule{futureCard}, 0, open); got != 1 {
		t.Fatalf("short(unreviewed=0) = %d, want 1", got)
	}
	if got := sim.short(pShort, []Schedule{futureCard}, 3, open); got != 4 {
		t.Fatalf("short(unreviewed=3) = %d, want 4", got)
	}
	// Card due after delta that is already learned is not short; card due on day 2 that learns by day 10 is not short.
	pReach := Preset{Goal: GoalDate, By: open.AddDate(0, 0, 10), Rule: RuleInterval, Interval: 5, Retention: 0.9}
	if got := sim.short(pReach, []Schedule{futureCard}, 0, open); got != 0 {
		t.Fatalf("short(reachable future card) = %d, want 0", got)
	}

	// Stability==0 floor prevents NaN recall; 1-day admitted run sets ReviewsADay and MinutesADay.
	zeroStab := map[ReviewCardID]Schedule{
		{Card: "z", Template: "T"}: {
			Last: open.AddDate(0, 0, -2), Due: open.AddDate(0, 0, -1), Reps: 1, Stability: 0, Difficulty: 5, Phase: 2,
		},
	}
	ran1Day, err := Simulation{
		By: fsrs, Day: d, Cost: review.GetDefaultCost(), Days: 1, Retains: []int{0},
	}.Run(t.Context(), now, Preset{Goal: GoalRetention, ReviewsADay: 10, MinutesADay: 30, Retention: 0.9}, zeroStab, 0)
	if err != nil {
		t.Fatalf("Run(zeroStab) err = %v", err)
	}
	if share, ok := ran1Day.Retained.GetShare(0); !ok || math.IsNaN(share) || share <= 0 {
		t.Fatalf("Retained share with zero stability = (%v, %v)", share, ok)
	}
	if ran1Day.ReviewsADay <= 0 || ran1Day.MinutesADay <= 0 {
		t.Fatalf("1-day run ReviewsADay=%v MinutesADay=%v, want > 0", ran1Day.ReviewsADay, ran1Day.MinutesADay)
	}

	// step(): new card AssignDueDay under IsEvenLoad moves due date from crowded day 5 to empty day 4;
	// review card Endings weighted average uses (1-back)*again.Due (0.5*10d + 0.5*4d = 7d).
	simMulti := Simulation{
		By:      multiDayNewScheduler{inner: fsrs},
		Day:     d,
		Cost:    review.GetDefaultCost(),
		Days:    10,
		Recalls: func(Schedule, time.Time) float64 { return 0.5 },
	}
	on := NewDueByDay(d)
	on.Add(open.AddDate(0, 0, 5))
	on.Add(open.AddDate(0, 0, 5))
	on.Add(open.AddDate(0, 0, 6))
	evenPreset := Preset{IsEvenLoad: true}
	steppedNew := simMulti.step(Schedule{}, open, evenPreset, on)
	if gotDay := d.GetName(steppedNew.Due); gotDay != d.GetName(open.AddDate(0, 0, 4)) {
		t.Fatalf("steppedNew.Due = %s, want %s (moved from crowded day 5 to day 4)", gotDay, d.GetName(open.AddDate(0, 0, 4)))
	}
	steppedRev := simMulti.step(Schedule{Last: open.AddDate(0, 0, -5), Due: open, Reps: 1, Stability: 5}, open, Preset{}, nil)
	if gotDay := d.GetName(steppedRev.Due); gotDay != d.GetName(open.AddDate(0, 0, 7)) {
		t.Fatalf("steppedRev.Due = %s, want %s (0.5*10d + 0.5*4d = 7d)", gotDay, d.GetName(open.AddDate(0, 0, 7)))
	}
}
