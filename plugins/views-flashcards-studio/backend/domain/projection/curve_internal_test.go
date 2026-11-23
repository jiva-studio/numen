package projection

import (
	"slices"
	"testing"
	"time"
)

func TestGridHelpers(t *testing.T) {
	t.Parallel()
	if got := GetFullRecall(Card{}, time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)); got != 1 {
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
	setNearestCell(grid, 99, -1, 2)
	setNearestCell(grid, 99, 0, 3)
	setNearestCell(grid, 99, 2, 1)
	if !slices.Equal(grid, []float64{10, 20, 30}) {
		t.Fatalf("snap out-of-bounds modified grid: %v", grid)
	}
	setNearestCell(grid, 25, 1, 1)
	if !slices.Equal(grid, []float64{10, 25, 30}) {
		t.Fatalf("snap first==last = %v, want [10 25 30]", grid)
	}
	grid = []float64{10, 20, 30}
	setNearestCell(grid, 15, 0, 2)
	if !slices.Equal(grid, []float64{15, 20, 30}) {
		t.Fatalf("snap tie = %v, want [15 20 30]", grid)
	}
	grid = []float64{10, 20, 30}
	setNearestCell(grid, 28, 0, 2)
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
	if got := p.computeMinutesADay(0); got != 0 {
		t.Fatalf("minutesADay(0) = %v, want 0", got)
	}
	if got := p.computeMinutesADay(1); got != 12 {
		t.Fatalf("minutesADay(1) = %v, want 12", got)
	}
	if got := p.computeMinutesADay(2); got != 21 {
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

	if day, ok := p.FindSession(); !ok || day != 1 {
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
	if day, ok := pausedProj.FindSession(); ok || day != 0 {
		t.Fatalf("paused Session() = (%d, %v), want (0, false)", day, ok)
	}
	if got := getSessionClosed(pausedProj); !slices.Equal(got, BudgetNames{ClosedPaused}) {
		t.Fatalf("getSessionClosed(paused) = %v, want [paused]", got)
	}
	if got := buildSessionPoint(pausedProj); got.Reviews != 7 || got.Minutes != 15 {
		t.Fatalf("session(paused) = %+v, want Reviews=7 Minutes=15", got)
	}
	if got := getSessionMinutes(pausedProj); got != 0 {
		t.Fatalf("getSessionMinutes(paused) = %v, want 0", got)
	}
	if got := createDatePoint(pausedProj, 0); got.Reviews != 0 || !slices.Equal(got.Closed, BudgetNames{ClosedPaused}) {
		t.Fatalf("createDatePoint(paused) = %+v, want Reviews=0 Closed=[paused]", got)
	}

	// getNamedOffset boundaries.
	d := utcDays{}
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
