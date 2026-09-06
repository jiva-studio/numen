package transcription

import (
	"context"
	"math"
)

// The resampler reads through a windowed sinc reaching this many of its own
// zeroes on each side.
const sincWidth = 16

// resampled is one signal at another rate.
//
// Each output sample is what the input says at that moment, band-limited to
// whichever of the two rates is the lower.
func resampled(ctx context.Context, in []float32, from, to int) ([]float32, error) {
	if from == to || from <= 0 || len(in) == 0 {
		return in, nil
	}
	ratio := float64(to) / float64(from)
	cutoff := math.Min(ratio, 1)
	reach := sincWidth / cutoff
	kernel := weighing(cutoff, reach)

	out := make([]float32, int(float64(len(in))*ratio))
	for i := range out {
		// An hour of sound is tens of millions of these, and a person closing
		// the window waits for whichever one it is on.
		if i%(1<<16) == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		centre := float64(i) / ratio
		first := max(int(math.Ceil(centre-reach)), 0)
		last := min(int(math.Floor(centre+reach)), len(in)-1)

		var sum, weight float64
		for j := first; j <= last; j++ {
			w := kernel.at(centre - float64(j))
			sum += w * float64(in[j])
			weight += w
		}
		if weight != 0 {
			out[i] = float32(sum / weight)
		}
	}
	return out, nil
}

// The kernel is read off a table this many steps to the input sample. An hour
// of sound is tens of millions of output samples of a hundred taps each, and
// the shape they are read from does not change between any two of them.
const kernelSteps = 512

// A kernel is the windowed sinc, worked out once and read from.
type kernel struct {
	held []float64
	step float64
}

// weighing works the kernel out over its whole reach.
func weighing(cutoff, reach float64) kernel {
	step := float64(kernelSteps)
	held := make([]float64, int(reach*step)+2)
	for i := range held {
		d := float64(i) / step
		held[i] = sinc(cutoff*d) * blackman(d/reach)
	}
	return kernel{held: held, step: step}
}

// at is the kernel at a distance, between the two steps it falls between. The
// kernel is even, so a distance either side of nothing reads the same.
func (w kernel) at(d float64) float64 {
	at := math.Abs(d) * w.step
	i := int(at)
	if i+1 >= len(w.held) {
		return 0
	}
	part := at - float64(i)
	return w.held[i]*(1-part) + w.held[i+1]*part
}

// sinc is the interpolating kernel, one at nothing.
func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(math.Pi*x) / (math.Pi * x)
}

// blackman brings the kernel to nothing at the edge of its reach.
func blackman(t float64) float64 {
	if t < -1 || t > 1 {
		return 0
	}
	return 0.42 + 0.5*math.Cos(math.Pi*t) + 0.08*math.Cos(2*math.Pi*t)
}
