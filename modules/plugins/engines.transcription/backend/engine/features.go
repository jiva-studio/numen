package engine

import (
	"math"
	"math/cmplx"
	"runtime"
	"sync"
)

// The features the encoder is given are NeMo's: a log-mel spectrogram of
// pre-emphasised audio, each band brought to zero mean and unit deviation over
// the whole stretch. Every number here is part of the model's identity, and a
// model fed a spectrogram made differently writes other words.
const (
	sampleRate  = 16000
	fftSize     = 512
	hopSize     = 160
	windowSize  = 400
	melBands    = 128
	preemphasis = 0.97
	melFloor    = 0.0
	melCeiling  = sampleRate / 2
)

// logGuard is what is added under the logarithm, and deviation is what is added
// to a band's spread before it divides.
const (
	logGuard  = 1.0 / (1 << 24)
	deviation = 1e-5
)

// logMel is one stretch of 16 kHz mono audio as the encoder takes it: the bands
// one after another, each holding every frame, and how many frames there are.
// It is computed on up to workers goroutines; zero or less is one per CPU.
func logMel(samples []float32, workers int) ([]float32, int) {
	power, frames := spectrogram(applyPreemphasis(samples), workers)
	if frames == 0 {
		return nil, 0
	}

	bins := fftSize/2 + 1
	bank := filters()
	out := make([]float32, melBands*frames)

	workers = getWorkerCount(workers, melBands)

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := range workers {
		start := worker * melBands / workers
		end := (worker + 1) * melBands / workers
		go func(start, end int) {
			defer wg.Done()
			for band := start; band < end; band++ {
				filter := bank[band]
				row := out[band*frames : (band+1)*frames]
				for t := 0; t < frames; t++ {
					var sum float64
					offset := t*bins + filter.firstBin
					for bin, w := range filter.weights {
						sum += w * power[offset+bin]
					}
					row[t] = float32(math.Log(sum + logGuard))
				}
				normalise(row)
			}
		}(start, end)
	}
	wg.Wait()
	return out, frames
}

// normalise brings one band to zero mean and unit deviation over time. The
// spread is of the values themselves, not of an estimate drawn from them.
func normalise(row []float32) {
	var sum float64
	for _, v := range row {
		sum += float64(v)
	}
	mean := sum / float64(len(row))

	var square float64
	for _, v := range row {
		d := float64(v) - mean
		square += d * d
	}
	spread := math.Sqrt(square / float64(len(row)))

	for i, v := range row {
		row[i] = float32((float64(v) - mean) / (spread + deviation))
	}
}

// applyPreemphasis lifts the high end of one stretch. The first sample stands as it
// is, having nothing before it.
func applyPreemphasis(samples []float32) []float64 {
	out := make([]float64, len(samples))
	for i, v := range samples {
		if i == 0 {
			out[i] = float64(v)
			continue
		}
		out[i] = float64(v) - preemphasis*float64(samples[i-1])
	}
	return out
}

// spectrogram is the power in each frequency bin of each frame.
//
// The signal is reflected outward by half a window at each end, so that a frame
// is centred on every hop from the first sample to the last.
func spectrogram(x []float64, workers int) ([]float64, int) {
	if len(x) == 0 {
		return nil, 0
	}
	padded := padReflect(x, fftSize/2)
	frames := len(x)/hopSize + 1
	window := hann()
	bins := fftSize/2 + 1

	out := make([]float64, frames*bins)
	workers = getWorkerCount(workers, frames)

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := range workers {
		start := worker * frames / workers
		end := (worker + 1) * frames / workers
		go func(start, end int) {
			defer wg.Done()
			buf := make([]complex128, fftSize)
			for t := start; t < end; t++ {
				pos := t * hopSize
				for i := 0; i < fftSize; i++ {
					buf[i] = complex(padded[pos+i]*window[i], 0)
				}
				fft(buf)
				offset := t * bins
				for i := 0; i < bins; i++ {
					re, im := real(buf[i]), imag(buf[i])
					out[offset+i] = re*re + im*im
				}
			}
		}(start, end)
	}
	wg.Wait()
	return out, frames
}

// padReflect is one signal with by samples of itself mirrored onto each end, the
// edge sample not repeated.
func padReflect(x []float64, delta int) []float64 {
	out := make([]float64, 0, len(x)+2*delta)
	for i := delta; i > 0; i-- {
		out = append(out, x[getMirroredIndex(i, len(x))])
	}
	out = append(out, x...)
	for i := 1; i <= delta; i++ {
		out = append(out, x[getMirroredIndex(len(x)-1-i, len(x))])
	}
	return out
}

// getMirroredIndex folds an index back into a signal of n samples, mirroring at each
// end as often as it takes.
func getMirroredIndex(i, n int) int {
	if n == 1 {
		return 0
	}
	period := 2 * (n - 1)
	i = ((i % period) + period) % period
	if i >= n {
		return period - i
	}
	return i
}

// hann is the analysis window: a periodic raised cosine over the window's own
// length, sitting in the middle of the transform.
var hann = sync.OnceValue(func() []float64 {
	out := make([]float64, fftSize)
	offset := (fftSize - windowSize) / 2
	for i := 0; i < windowSize; i++ {
		out[offset+i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(windowSize))
	}
	return out
})

// melFilter is one triangular frequency filter in the mel bank, holding its
// non-zero weights from firstBin upward.
type melFilter struct {
	firstBin int
	weights  []float64
}

// filters is the mel bank: for each band, what share of each frequency bin it
// carries. The triangles are laid out on the Slaney mel scale and each is
// scaled by the width of the frequencies it spans.
var filters = sync.OnceValue(func() []melFilter {
	bins := fftSize/2 + 1
	edges := make([]float64, melBands+2)
	low, high := hzToMel(melFloor), hzToMel(melCeiling)
	for i := range edges {
		edges[i] = melToHz(low + (high-low)*float64(i)/float64(melBands+1))
	}

	freq := make([]float64, bins)
	for i := range freq {
		freq[i] = float64(i) * sampleRate / fftSize
	}

	out := make([]melFilter, melBands)
	for band := 0; band < melBands; band++ {
		lower, centre, upper := edges[band], edges[band+1], edges[band+2]
		scale := 2.0 / (upper - lower)
		first := -1
		var weights []float64
		for i, f := range freq {
			rising := (f - lower) / (centre - lower)
			falling := (upper - f) / (upper - centre)
			w := math.Min(rising, falling)
			if w > 0 {
				if first == -1 {
					first = i
				}
				weights = append(weights, w*scale)
			}
		}
		if first == -1 {
			first = 0
		}
		out[band] = melFilter{
			firstBin: first,
			weights:  weights,
		}
	}
	return out
})

// The Slaney mel scale: linear below a thousand hertz and logarithmic above it.
const (
	melLinear = 200.0 / 3
	melKnee   = 1000.0
)

var melStep = math.Log(6.4) / 27.0

func hzToMel(hz float64) float64 {
	if hz >= melKnee {
		return melKnee/melLinear + math.Log(hz/melKnee)/melStep
	}
	return hz / melLinear
}

func melToHz(mel float64) float64 {
	if knee := melKnee / melLinear; mel >= knee {
		return melKnee * math.Exp(melStep*(mel-knee))
	}
	return mel * melLinear
}

// fft transforms one frame in place. The size is a power of two, which is the
// only size this is ever given.
func fft(a []complex128) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j |= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	w := twiddles()
	for length := 2; length <= n; length <<= 1 {
		stride := n / length
		for i := 0; i < n; i += length {
			for j := 0; j < length/2; j++ {
				u, v := a[i+j], a[i+j+length/2]*w[j*stride]
				a[i+j], a[i+j+length/2] = u+v, u-v
			}
		}
	}
}

// twiddles are the roots of unity one transform turns on, each computed from
// its own angle.
var twiddles = sync.OnceValue(func() []complex128 {
	out := make([]complex128, fftSize/2)
	for k := range out {
		out[k] = cmplx.Rect(1, -2*math.Pi*float64(k)/float64(fftSize))
	}
	return out
})

// getWorkerCount is how many goroutines share jobs units of work: the count
// asked for, or one per CPU when none is, and never more than there are jobs.
func getWorkerCount(requested, jobs int) int {
	if requested <= 0 {
		requested = runtime.GOMAXPROCS(0)
	}
	return max(1, min(requested, jobs))
}
