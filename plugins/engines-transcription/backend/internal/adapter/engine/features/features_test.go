package features

import (
	"bufio"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

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

// The spectrogram is the input to every word, and an error in it shifts every
// value. It is compared against values from the reference implementation.
func TestLogMelMatchesReference(t *testing.T) {
	want, bands, frames := reference(t, "testdata/features.txt")

	got, made := GetLogMel(tone(), 0)
	if made != frames {
		t.Fatalf("%d frames, and the reference has %d", made, frames)
	}
	if len(got) != bands*frames {
		t.Fatalf("%d values, and the reference has %d", len(got), bands*frames)
	}
	for i := range want {
		if getRelativeError(float64(got[i]), float64(want[i])) > 1e-4 {
			t.Fatalf("band %d frame %d is %g, and the reference says %g",
				i/frames, i%frames, got[i], want[i])
		}
	}
}

// A band that is one value throughout has no spread, and the guard under the
// division is what keeps it a number.
func TestNormalizeFlatBand(t *testing.T) {
	row := []float32{3, 3, 3, 3}
	normalize(row)
	for i, v := range row {
		if v != 0 {
			t.Fatalf("frame %d of a flat band is %g", i, v)
		}
	}
}

// The transform is compared against the sum it is defined as, so an error in
// the ordering is detected.
func TestFFTAgainstDefinition(t *testing.T) {
	const n = fftSize
	a := make([]complex128, n)
	for i := range a {
		a[i] = complex(math.Sin(float64(i)), math.Cos(float64(2*i)))
	}
	want := make([]complex128, n)
	for k := range n {
		var sum complex128
		for j := range n {
			angle := -2 * math.Pi * float64(k*j) / n
			sum += a[j] * complex(math.Cos(angle), math.Sin(angle))
		}
		want[k] = sum
	}

	got := make([]complex128, n)
	copy(got, a)
	fft(got, getTwiddles())
	for k := range want {
		if math.Abs(real(got[k])-real(want[k])) > 1e-9 || math.Abs(imag(got[k])-imag(want[k])) > 1e-9 {
			t.Fatalf("bin %d is %v, and the definition says %v", k, got[k], want[k])
		}
	}
}

// The signal is mirrored at each end, and the edge sample is not duplicated.
func TestReflected(t *testing.T) {
	got := padReflect([]float64{1, 2, 3, 4}, 2)
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

func TestEmphasized(t *testing.T) {
	got := applyPreemphasis([]float32{1, 1, 1})
	if got[0] != 1 {
		t.Fatalf("the first sample is %g", got[0])
	}
	if math.Abs(got[1]-(1-preemphasis)) > 1e-9 {
		t.Fatalf("the second sample is %g", got[1])
	}
}

func reference(t *testing.T, path string) ([]float32, int, int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer closeAndCheck(t, file)

	lines := bufio.NewScanner(file)
	lines.Buffer(make([]byte, 1<<20), 1<<20)
	if !lines.Scan() {
		t.Fatalf("%s says nothing", path)
	}
	shape := strings.Fields(lines.Text())
	bands, err := strconv.Atoi(shape[0])
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	frames, err := strconv.Atoi(shape[1])
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	out := make([]float32, 0, bands*frames)
	for lines.Scan() {
		for field := range strings.FieldsSeq(lines.Text()) {
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

func getRelativeError(got, want float64) float64 {
	scale := math.Max(math.Abs(want), 1)
	return math.Abs(got-want) / scale
}

func BenchmarkLogMel(b *testing.B) {
	audio := make([]float32, 10*sampleRate)
	for i := range audio {
		audio[i] = float32(math.Sin(float64(i) * 0.05))
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _ = GetLogMel(audio, 0)
	}
}
func TestTheFeaturesAreTheSameWhateverTheWorkerCount(t *testing.T) {
	sound := tone()
	want, frames := GetLogMel(sound, 1)
	for _, workers := range []int{0, 2, 7} {
		got, made := GetLogMel(sound, workers)
		if made != frames || !slices.Equal(got, want) {
			t.Errorf("%d workers made other features than one worker did", workers)
		}
	}
}

func TestFeaturesEdgeCases(t *testing.T) {
	if got, frames := GetLogMel(nil, 1); got != nil || frames != 0 {
		t.Errorf("GetLogMel(nil) = (%v, %d), want (nil, 0)", got, frames)
	}
	if got, frames := GetLogMel([]float32{1}, 1); len(got) != MelBands || frames != 1 {
		t.Errorf("GetLogMel(1 sample) = (len %d, %d), want (%d, 1)", len(got), frames, MelBands)
	}
	if power, frames := spectrogram(nil, 1); power != nil || frames != 0 {
		t.Errorf("spectrogram(nil) = (%v, %d), want (nil, 0)", power, frames)
	}
	power, frames := spectrogram([]float64{1}, 1)
	if frames != 1 || math.Abs(power[0]-40000) > 1e-6 {
		t.Errorf("spectrogram([1]) = (frames %d, power[0] %g), want (1, 40000)", frames, power[0])
	}

	for _, tc := range []struct {
		i, n, want int
	}{
		{5, 1, 0},
		{1, 2, 1},
		{-1, 4, 1},
		{-2, 4, 2},
		{4, 4, 2},
		{5, 4, 1},
		{6, 4, 0},
		{7, 4, 1},
	} {
		if got := getMirroredIndex(tc.i, tc.n); got != tc.want {
			t.Errorf("getMirroredIndex(%d, %d) = %d, want %d", tc.i, tc.n, got, tc.want)
		}
	}

	bank := getMelBank()
	if bank[0].firstBin != 1 {
		t.Errorf("bank[0].firstBin = %d, want 1", bank[0].firstBin)
	}
	if got := hzToMel(500); math.Abs(got-7.5) > 1e-9 {
		t.Errorf("hzToMel(500) = %g, want 7.5", got)
	}
	if got := len(getTwiddles()); got != fftSize/2 {
		t.Errorf("len(getTwiddles()) = %d, want %d", got, fftSize/2)
	}
}

func closeAndCheck(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}
