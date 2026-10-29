package projection

import "math"

// LeastCeiling is the shortest day a curve of minutes runs to.
const LeastCeiling = 60

// pickEvenlySpacedDays is up to as many places as are wanted, evenly over the days, and
// always both ends of them.
func pickEvenlySpacedDays(days, places int) []int {
	if days <= places {
		out := make([]int, days)
		for i := range out {
			out[i] = i
		}
		return out
	}
	out := make([]int, places)
	for i := range out {
		out[i] = int(math.Round(float64(i) * float64(days-1) / float64(places-1)))
	}
	return out
}

// setNearestStep replaces the place nearest pos with pos on the grid. The two end
// places remain fixed: a range begins tomorrow and extends to its natural end,
// whatever day the file names.
//
// The point under the place is worked out for the day the preset aims at.
func setNearestStep(steps []int, pos int) []int {
	if pos < 0 || len(steps) < 3 {
		return steps
	}
	step := 1
	for i := 2; i < len(steps)-1; i++ {
		if abs(steps[i]-pos) < abs(steps[step]-pos) {
			step = i
		}
	}
	if pos > steps[0] && pos < steps[len(steps)-1] {
		steps[step] = pos
	}
	return steps
}

// snap replaces the grid place nearest the value with the value, so the figures
// drawn at that place are those of the preset's current setting.
//
// First and last are the places a value may take: a range whose ends say what
// the setting may be at all keeps them.
func snap(grid []float64, value float64, first, last int) {
	if first < 0 || last >= len(grid) || first > last {
		return
	}
	pos := first
	for i := first; i <= last; i++ {
		if math.Abs(grid[i]-value) < math.Abs(grid[pos]-value) {
			pos = i
		}
	}
	grid[pos] = value
}

// abs returns the absolute value of a whole number.
func abs(one int) int {
	if one < 0 {
		return -one
	}
	return one
}

// getMinutesUpperBound is how far a curve of minutes runs: twice what carrying the whole
// load costs, and never less than a short day or more than a day holds.
func getMinutesUpperBound(load, keeping float64) float64 {
	top := math.Ceil(2 * math.Max(load, keeping))
	return math.Min(math.Max(top, LeastCeiling), MinutesADayBounds.Max)
}

// findNearest is the place of the grid a value falls at, and -1 for a value outside
// it.
func findNearest(grid []float64, value float64) int {
	if len(grid) == 0 || value < grid[0] || value > grid[len(grid)-1] {
		return -1
	}
	pos := 0
	for i, one := range grid {
		if math.Abs(one-value) < math.Abs(grid[pos]-value) {
			pos = i
		}
	}
	return pos
}
