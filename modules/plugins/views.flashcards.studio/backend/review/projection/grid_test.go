package projection

import "testing"

// How far a curve of minutes runs is twice the longer of what carrying the
// whole load costs and what the preset keeps, so the day the person is on and
// the day the load asks for both stand inside it. It is never shorter than a
// short day and never longer than a day holds.
func TestHowFarACurveOfMinutesRuns(t *testing.T) {
	t.Parallel()
	for _, one := range []struct{ load, keeping, want float64 }{
		{load: 0, keeping: 0, want: LeastCeiling},
		{load: 4, keeping: 1, want: LeastCeiling},
		{load: 100, keeping: 0, want: 200},
		{load: 0, keeping: 100, want: 200},
		{load: 30.5, keeping: 0, want: 61},
		{load: 5000, keeping: 0, want: MinutesADayBounds.Most},
	} {
		if got := getMinutesUpperBound(one.load, one.keeping); got != one.want {
			t.Errorf("a load of %v minutes under a day of %v runs to %v, want %v",
				one.load, one.keeping, got, one.want)
		}
	}
}

// The place of the grid a value falls at is the nearest of them, the first of
// two it stands equally far from. A value beyond either end stands nowhere.
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
