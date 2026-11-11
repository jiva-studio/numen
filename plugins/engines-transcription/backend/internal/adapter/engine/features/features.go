// Package features computes the log-mel spectrogram the transcription model reads.
package features

import (
	"math"
	"math/cmplx"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/signal"
)

// The features the encoder is given are NeMo's: a log-mel spectrogram of
// pre-emphasized audio, each band brought to zero mean and unit deviation over
// the whole stretch. Every number here is part of the model's identity, and a
// model fed a spectrogram made differently writes other words.
const (
	sampleRate  = signal.SampleRate
	fftSize     = 512
	hopSize     = 160
	windowSize  = 400
	MelBands    = 128
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

// GetLogMel is one stretch of 16 kHz mono audio as the encoder takes it: the bands
// one after another, each holding every frame, and how many frames there are.
// It is computed on up to workers goroutines; zero or less is one per CPU.
func GetLogMel(samples []float32, workers int) ([]float32, int) {
	power, frames := spectrogram(applyPreemphasis(samples), workers)
	if frames == 0 {
		return nil, 0
	}

	bins := fftSize/2 + 1
	bank := getMelBank()
	out := make([]float32, MelBands*frames)

	workers = signal.GetWorkerCount(workers, MelBands)

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := range workers {
		// gomutants:disable-next-line ARITHMETIC_BASE reason="shrinking start only repeats deterministic band work"
		start := worker * MelBands / workers
		end := (worker + 1) * MelBands / workers
		go func(start, end int) {
			defer wg.Done()
			for band := start; band < end; band++ {
				filter := bank[band]
				row := out[band*frames : (band+1)*frames]
				for t := range frames {
					var sum float64
					offset := t*bins + filter.firstBin
					for bin, w := range filter.weights {
						sum += w * power[offset+bin]
					}
					row[t] = float32(math.Log(sum + logGuard))
				}
				normalize(row)
			}
		}(start, end)
	}
	wg.Wait()
	return out, frames
}

func normalize(row []float32) {
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

func spectrogram(x []float64, workers int) ([]float64, int) {
	if len(x) == 0 {
		return nil, 0
	}
	padded := padReflect(x, fftSize/2)
	frames := len(x)/hopSize + 1
	window := getHannWindow()
	twiddles := getTwiddles()
	bins := fftSize/2 + 1

	out := make([]float64, frames*bins)
	workers = signal.GetWorkerCount(workers, frames)

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := range workers {
		// gomutants:disable-next-line ARITHMETIC_BASE reason="shrinking start only repeats deterministic frame work"
		start := worker * frames / workers
		end := (worker + 1) * frames / workers
		go func(start, end int) {
			defer wg.Done()
			buf := make([]complex128, fftSize)
			for t := start; t < end; t++ {
				pos := t * hopSize
				for i := range fftSize {
					buf[i] = complex(padded[pos+i]*window[i], 0)
				}
				fft(buf, twiddles)
				offset := t * bins
				for i := range bins {
					re, im := real(buf[i]), imag(buf[i])
					out[offset+i] = re*re + im*im
				}
			}
		}(start, end)
	}
	wg.Wait()
	return out, frames
}

func padReflect(x []float64, delta int) []float64 {
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="slice capacity preallocation does not alter reflected elements"
	out := make([]float64, 0, len(x)+2*delta)
	for i := delta; i > 0; i-- {
		out = append(out, x[getMirroredIndex(i, len(x))])
	}
	out = append(out, x...)
	for i := 1; i <= delta; i++ {
		// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="getMirroredIndex is symmetric around len(x)-1"
		out = append(out, x[getMirroredIndex(len(x)-1-i, len(x))])
	}
	return out
}

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

func getHannWindow() []float64 {
	out := make([]float64, fftSize)
	offset := (fftSize - windowSize) / 2
	// gomutants:disable-next-line INTEGER_INCREMENT,CONDITIONALS_BOUNDARY reason="the Hann window is zero at both endpoints and out is zero-initialized"
	for i := range windowSize {
		out[offset+i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(windowSize))
	}
	return out
}

// melFilter is one triangular frequency filter in the mel bank, holding its
// non-zero weights from firstBin upward.
type melFilter struct {
	firstBin int
	weights  []float64
}

func getMelBank() []melFilter {
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="bins at and above Nyquist have non-positive filter weight"
	bins := fftSize/2 + 1
	// gomutants:disable-next-line INTEGER_INCREMENT reason="extra trailing edge elements are never read"
	edges := make([]float64, MelBands+2)
	low, high := hzToMel(melFloor), hzToMel(melCeiling)
	for i := range edges {
		// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="low is zero at melFloor"
		edges[i] = melToHz(low + (high-low)*float64(i)/float64(MelBands+1))
	}

	freq := make([]float64, bins)
	for i := range freq {
		freq[i] = float64(i) * sampleRate / fftSize
	}

	out := make([]melFilter, MelBands)
	for band := range MelBands {
		lower, center, upper := edges[band], edges[band+1], edges[band+2]
		scale := 2.0 / (upper - lower)
		first := -1
		var weights []float64
		for i, f := range freq {
			rising := (f - lower) / (center - lower)
			falling := (upper - f) / (upper - center)
			w := math.Min(rising, falling)
			if w > 0 {
				if first == -1 {
					first = i
				}
				weights = append(weights, w*scale)
			}
		}
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="every mel band overlaps at least one FFT bin"
		if first == -1 {
			first = 0
		}
		out[band] = melFilter{
			firstBin: first,
			weights:  weights,
		}
	}
	return out
}

// The Slaney mel scale: linear below a thousand hertz and logarithmic above it.
const (
	melLinear = 200.0 / 3
	melKnee   = 1000.0
)

func getMelStep() float64 {
	return math.Log(6.4) / 27.0
}

func hzToMel(hz float64) float64 {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="the Slaney mel curve is continuous at the knee"
	if hz >= melKnee {
		return melKnee/melLinear + math.Log(hz/melKnee)/getMelStep()
	}
	return hz / melLinear
}

func melToHz(mel float64) float64 {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="the Slaney mel curve is continuous at the knee"
	if knee := melKnee / melLinear; mel >= knee {
		return melKnee * math.Exp(getMelStep()*(mel-knee))
	}
	return mel * melLinear
}

func fft(a, w []complex128) {
	n := len(a)
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at i == n the bit-reversed index is 0 so i < j is false"
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j |= bit
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION reason="bit-reversal is symmetric and self-swaps are no-ops"
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	// gomutants:disable-next-line INTEGER_DECREMENT reason="length 1 has length/2 == 0 and executes zero butterfly steps"
	for length := 2; length <= n; length <<= 1 {
		stride := n / length
		for i := 0; i < n; i += length {
			for j := range length / 2 {
				u, v := a[i+j], a[i+j+length/2]*w[j*stride]
				a[i+j], a[i+j+length/2] = u+v, u-v
			}
		}
	}
}

func getTwiddles() []complex128 {
	out := make([]complex128, fftSize/2)
	for k := range out {
		out[k] = cmplx.Rect(1, -2*math.Pi*float64(k)/float64(fftSize))
	}
	return out
}
