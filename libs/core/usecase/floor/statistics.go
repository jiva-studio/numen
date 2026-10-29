package floor

import (
	"math"
	"slices"
)

const (
	// minFloor and maxFloor bound a calibrated floor.
	minFloor = 0.10
	maxFloor = 0.95

	// minSpread is the lower bound of the spread a calibration uses.
	minSpread = 0.01

	// iqrPerSigma is the width of the interquartile range of a normal
	// distribution in standard deviations.
	iqrPerSigma = 1.349

	// spreadsInFloor is how many spreads above the median the floor lies.
	spreadsInFloor = 2

	// histogramBins is the number of equal bins from zero to one.
	histogramBins = 20
)

// Statistics summarises the similarities between queries and passages that are
// not about them.
type Statistics struct {
	Median    float64
	Spread    float64
	Floor     float64
	Histogram []int
}

// Summarize computes the statistics of a sample. The sample is left as it was
// given. A sample with no value has the minimum floor, a zero median and spread
// and an empty histogram.
func Summarize(similarities []float64) Statistics {
	histogram := make([]int, histogramBins)
	if len(similarities) == 0 {
		return Statistics{Spread: minSpread, Floor: minFloor, Histogram: histogram}
	}
	sorted := slices.Clone(similarities)
	slices.Sort(sorted)

	median := getQuantile(sorted, 0.5)
	spread := max((getQuantile(sorted, 0.75)-getQuantile(sorted, 0.25))/iqrPerSigma, minSpread)
	for _, one := range sorted {
		histogram[getBin(one)]++
	}
	return Statistics{
		Median:    median,
		Spread:    spread,
		Floor:     min(max(median+spreadsInFloor*spread, minFloor), maxFloor),
		Histogram: histogram,
	}
}

// getQuantile is the value at fraction p of a sorted sample, interpolating
// linearly between the two values around it.
func getQuantile(sorted []float64, p float64) float64 {
	position := p * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := min(lower+1, len(sorted)-1)
	return sorted[lower] + (sorted[upper]-sorted[lower])*(position-float64(lower))
}

// getBin is the histogram bin of a similarity. Values outside zero to one fall in
// the first or last bin.
func getBin(similarity float64) int {
	if math.IsNaN(similarity) {
		return 0
	}
	return min(max(int(math.Floor(similarity*histogramBins)), 0), histogramBins-1)
}
