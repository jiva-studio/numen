package transcription

import (
	"math"
	"math/cmplx"
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
func logMel(samples []float32) ([]float32, int) {
	power := spectrogram(emphasised(samples))
	frames := len(power)
	if frames == 0 {
		return nil, 0
	}

	bank := filters()
	out := make([]float32, melBands*frames)
	for band := 0; band < melBands; band++ {
		weights := bank[band]
		row := out[band*frames : (band+1)*frames]
		for t := 0; t < frames; t++ {
			var sum float64
			for bin, w := range weights {
				sum += w * power[t][bin]
			}
			row[t] = float32(math.Log(sum + logGuard))
		}
		normalise(row)
	}
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

// emphasised lifts the high end of one stretch. The first sample stands as it
// is, having nothing before it.
func emphasised(samples []float32) []float64 {
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
func spectrogram(x []float64) [][]float64 {
	if len(x) == 0 {
		return nil
	}
	padded := reflected(x, fftSize/2)
	frames := len(x)/hopSize + 1
	window := hann()
	bins := fftSize/2 + 1

	out := make([][]float64, frames)
	buf := make([]complex128, fftSize)
	for t := 0; t < frames; t++ {
		at := t * hopSize
		for i := 0; i < fftSize; i++ {
			buf[i] = complex(padded[at+i]*window[i], 0)
		}
		fft(buf)
		row := make([]float64, bins)
		for i := 0; i < bins; i++ {
			re, im := real(buf[i]), imag(buf[i])
			row[i] = re*re + im*im
		}
		out[t] = row
	}
	return out
}

// reflected is one signal with by samples of itself mirrored onto each end, the
// edge sample not repeated.
func reflected(x []float64, by int) []float64 {
	out := make([]float64, 0, len(x)+2*by)
	for i := by; i > 0; i-- {
		out = append(out, x[at(i, len(x))])
	}
	out = append(out, x...)
	for i := 1; i <= by; i++ {
		out = append(out, x[at(len(x)-1-i, len(x))])
	}
	return out
}

// at folds an index back into a signal of n samples, mirroring at each end as
// often as it takes.
func at(i, n int) int {
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

// filters is the mel bank: for each band, what share of each frequency bin it
// carries. The triangles are laid out on the Slaney mel scale and each is
// scaled by the width of the frequencies it spans.
var filters = sync.OnceValue(func() [][]float64 {
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

	out := make([][]float64, melBands)
	for band := 0; band < melBands; band++ {
		row := make([]float64, bins)
		lower, centre, upper := edges[band], edges[band+1], edges[band+2]
		scale := 2.0 / (upper - lower)
		for i, f := range freq {
			rising := (f - lower) / (centre - lower)
			falling := (upper - f) / (upper - centre)
			w := math.Min(rising, falling)
			if w > 0 {
				row[i] = w * scale
			}
		}
		out[band] = row
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
