package projection

import "math"

// LeastCeiling is the shortest day a curve of minutes runs to.
const LeastCeiling = 60

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

func abs(one int) int {
	if one < 0 {
		return -one
	}
	return one
}

func getMinutesUpperBound(load, keeping float64) float64 {
	top := math.Ceil(2 * math.Max(load, keeping))
	return math.Min(math.Max(top, LeastCeiling), GetMinutesADayBounds().Max)
}

func findNearestGridIndex(grid []float64, value float64) int {
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
