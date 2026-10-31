package projection

import (
	"testing"
)

// The upper bound of a minutes curve is twice the larger of the cost of the
// whole load and the minutes the preset keeps, so the current day and the day
// the load requires are both inside it. It is never below the least ceiling
// and never above the minutes in a day.
func TestHowFarACurveOfMinutesRuns(t *testing.T) {
	t.Parallel()
	for _, one := range []struct{ load, keeping, want float64 }{
		{load: 0, keeping: 0, want: LeastCeiling},
		{load: 4, keeping: 1, want: LeastCeiling},
		{load: 100, keeping: 0, want: 200},
		{load: 0, keeping: 100, want: 200},
		{load: 30.5, keeping: 0, want: 61},
		{load: 5000, keeping: 0, want: MinutesADayBounds.Max},
	} {
		if got := getMinutesUpperBound(one.load, one.keeping); got != one.want {
			t.Errorf("a load of %v minutes under a day of %v runs to %v, want %v",
				one.load, one.keeping, got, one.want)
		}
	}
}

// The place of the grid for a value is the nearest grid value, the first one
// when two are equally near. A value beyond either end has no place.
func TestThePlaceOfTheGridAValueFallsAt(t *testing.T) {
	t.Parallel()
	grid := []float64{10, 20, 30, 40}
	for _, one := range []struct {
		value         float64
		expectedIndex int
	}{
		{value: 9, expectedIndex: OutsideGrid.Index},
		{value: 41, expectedIndex: OutsideGrid.Index},
		{value: 10, expectedIndex: 0},
		{value: 15, expectedIndex: 0},
		{value: 24, expectedIndex: 1},
		{value: 40, expectedIndex: 3},
	} {
		if got := findNearest(grid, one.value); got != one.expectedIndex {
			t.Errorf("%v falls at place %d of %v, want %d", one.value, got, grid, one.expectedIndex)
		}
	}
	if got := findNearest(nil, 10); got != OutsideGrid.Index {
		t.Errorf("a value falls at place %d of a grid of no places", got)
	}
}
