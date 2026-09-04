package flashcards

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

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
		{load: 5000, keeping: 0, want: review.MinutesADayBounds.Most},
	} {
		if got := ceiling(one.load, one.keeping); got != one.want {
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
		value float64
		at    int
	}{
		{value: 9, at: Nowhere.Index},
		{value: 41, at: Nowhere.Index},
		{value: 10, at: 0},
		{value: 15, at: 0},
		{value: 24, at: 1},
		{value: 40, at: 3},
	} {
		if got := nearest(grid, one.value); got != one.at {
			t.Errorf("%v falls at place %d of %v, want %d", one.value, got, grid, one.at)
		}
	}
	if got := nearest(nil, 10); got != Nowhere.Index {
		t.Errorf("a value falls at place %d of a grid of no places", got)
	}
}
