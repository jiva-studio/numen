package transcription

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// tone is half a second of audio written by a formula, so that the numbers this
// is measured against can be made from the same formula anywhere.
func tone() []float32 {
	out := make([]float32, sampleRate/2)
	for n := range out {
		t := float64(n) / sampleRate
		out[n] = float32(0.5*math.Sin(2*math.Pi*440*t) +
			0.3*math.Sin(2*math.Pi*1867*t+0.7) +
			0.15*math.Cos(2*math.Pi*57*t))
	}
	return out
}

// The spectrogram is what every word depends on, and a mistake in it moves
// every number a little. It is measured against a bank of values taken from the
// reference implementation.
func TestLogMelMatchesReference(t *testing.T) {
	want, bands, frames := reference(t, "testdata/features.txt")

	got, made := logMel(tone())
	if made != frames {
		t.Fatalf("%d frames, and the reference has %d", made, frames)
	}
	if len(got) != bands*frames {
		t.Fatalf("%d values, and the reference has %d", len(got), bands*frames)
	}
	for i := range want {
		if apart(float64(got[i]), float64(want[i])) > 1e-4 {
			t.Fatalf("band %d frame %d is %g, and the reference says %g",
				i/frames, i%frames, got[i], want[i])
		}
	}
}

// A band that is one value throughout has no spread, and the guard under the
// division is what keeps it a number.
func TestNormaliseFlatBand(t *testing.T) {
	row := []float32{3, 3, 3, 3}
	normalise(row)
	for i, v := range row {
		if v != 0 {
			t.Fatalf("frame %d of a flat band is %g", i, v)
		}
	}
}

// The transform is measured against the sum it is defined as, so that an error
// in the ordering shows.
func TestFFTAgainstDefinition(t *testing.T) {
	const n = fftSize
	a := make([]complex128, n)
	for i := range a {
		a[i] = complex(math.Sin(float64(i)), math.Cos(float64(2*i)))
	}
	want := make([]complex128, n)
	for k := 0; k < n; k++ {
		var sum complex128
		for j := 0; j < n; j++ {
			angle := -2 * math.Pi * float64(k*j) / n
			sum += a[j] * complex(math.Cos(angle), math.Sin(angle))
		}
		want[k] = sum
	}

	got := make([]complex128, n)
	copy(got, a)
	fft(got)
	for k := range want {
		if math.Abs(real(got[k])-real(want[k])) > 1e-9 || math.Abs(imag(got[k])-imag(want[k])) > 1e-9 {
			t.Fatalf("bin %d is %v, and the definition says %v", k, got[k], want[k])
		}
	}
}

// The signal is mirrored at each end, and the sample on the edge is not said
// twice.
func TestReflected(t *testing.T) {
	got := reflected([]float64{1, 2, 3, 4}, 2)
	want := []float64{3, 2, 1, 2, 3, 4, 3, 2}
	if len(got) != len(want) {
		t.Fatalf("%v, and it should be %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v, and it should be %v", got, want)
		}
	}
}

func TestEmphasised(t *testing.T) {
	got := emphasised([]float32{1, 1, 1})
	if got[0] != 1 {
		t.Fatalf("the first sample is %g", got[0])
	}
	if math.Abs(got[1]-(1-preemphasis)) > 1e-9 {
		t.Fatalf("the second sample is %g", got[1])
	}
}

// reference is a matrix written one band to a line, and its shape.
func reference(t *testing.T, path string) ([]float32, int, int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer file.Close()

	lines := bufio.NewScanner(file)
	lines.Buffer(make([]byte, 1<<20), 1<<20)
	if !lines.Scan() {
		t.Fatalf("%s says nothing", path)
	}
	shape := strings.Fields(lines.Text())
	bands, _ := strconv.Atoi(shape[0])
	frames, _ := strconv.Atoi(shape[1])

	out := make([]float32, 0, bands*frames)
	for lines.Scan() {
		for _, field := range strings.Fields(lines.Text()) {
			v, err := strconv.ParseFloat(field, 32)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			out = append(out, float32(v))
		}
	}
	if len(out) != bands*frames {
		t.Fatalf("%s holds %d values and says %d", path, len(out), bands*frames)
	}
	return out, bands, frames
}

// apart is how far two numbers are from each other, as a share of the larger.
func apart(got, want float64) float64 {
	scale := math.Max(math.Abs(want), 1)
	return math.Abs(got-want) / scale
}
