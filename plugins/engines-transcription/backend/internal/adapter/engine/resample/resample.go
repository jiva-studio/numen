// Package resample converts audio samples between sampling rates.
package resample

import (
	"context"
	"math"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/signal"
)

// The resampler reads through a windowed sinc reaching this many of its own
// zeroes on each side.
const sincWidth = 16

// Convert resamples one signal to another rate.
//
// Each output sample is the input value at that moment, band-limited to the
// lower of the two rates. It runs on up to workers goroutines; zero or less is
// one per CPU.
func Convert(ctx context.Context, in []float32, from, to, workers int) ([]float32, error) {
	if from == to || from <= 0 || len(in) == 0 {
		return in, nil
	}
	ratio := float64(to) / float64(from)
	cutoff := math.Min(ratio, 1)
	reach := sincWidth / cutoff
	kernel := buildKernel(cutoff, reach)

	out := make([]float32, int(float64(len(in))*ratio))
	if len(out) == 0 {
		return out, nil
	}

	workers = signal.GetWorkerCount(workers, len(out))

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := range workers {
		// gomutants:disable-next-line ARITHMETIC_BASE reason="shrinking start only repeats deterministic sample work"
		start := worker * len(out) / workers
		end := (worker + 1) * len(out) / workers
		go func(start, end int) {
			defer wg.Done()
			span := resampleSpan{in: in, out: out, ratio: ratio, reach: reach, kernel: kernel}
			span.fill(ctx, start, end)
		}(start, end)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// resampleSpan is the signal read at one rate and written at another.
type resampleSpan struct {
	in, out []float32
	ratio   float64
	reach   float64
	kernel  kernel
}

func (s resampleSpan) fill(ctx context.Context, start, end int) {
	for i := start; i < end; i++ {
		// gomutants:disable-next-line EXPRESSION_REMOVE,ARITHMETIC_BASE,INVERT_NEGATIVES,INTEGER_INCREMENT,INTEGER_DECREMENT,CONDITIONALS_NEGATION,BRANCH_IF reason="periodic cancellation check is a performance optimization backed by the post-Wait check"
		if (i-start)%(1<<16) == 0 && ctx.Err() != nil {
			return
		}
		center := float64(i) / s.ratio
		first := max(int(math.Ceil(center-s.reach)), 0)
		last := min(int(math.Floor(center+s.reach)), len(s.in)-1)

		var sum, weight float64
		for j := first; j <= last; j++ {
			w := s.kernel.getValueAt(center - float64(j))
			sum += w * float64(s.in[j])
			weight += w
		}
		if weight != 0 {
			s.out[i] = float32(sum / weight)
		}
	}
}

// The kernel is read off a table this many steps to the input sample. An hour
// of sound is tens of millions of output samples of a hundred taps each, and
// the shape they are read from does not change between any two of them.
const kernelSteps = 512

// A kernel is the windowed sinc, worked out once and read from.
type kernel struct {
	values []float64
	step   float64
}

func buildKernel(cutoff, reach float64) kernel {
	step := float64(kernelSteps)
	target := make([]float64, int(reach*step)+2)
	for i := range target {
		d := float64(i) / step
		target[i] = getSinc(cutoff*d) * getBlackmanWindow(d/reach)
	}
	return kernel{values: target, step: step}
}

func (w kernel) getValueAt(d float64) float64 {
	pos := math.Abs(d) * w.step
	i := int(pos)
	if i+1 >= len(w.values) {
		return 0
	}
	part := pos - float64(i)
	return w.values[i]*(1-part) + w.values[i+1]*part
}

func getSinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(math.Pi*x) / (math.Pi * x)
}

func getBlackmanWindow(t float64) float64 {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="the Blackman window is zero at t == -1 and t == 1"
	if t < -1 || t > 1 {
		return 0
	}
	return 0.42 + 0.5*math.Cos(math.Pi*t) + 0.08*math.Cos(2*math.Pi*t)
}
