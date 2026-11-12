package resample

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/signal"
)

// A recording at another rate is brought to the one the models take, and the
// tone it carries is still the tone it carried.
func TestResamplingKeepsTheTone(t *testing.T) {
	const from, hz = 48000, 440.0
	in := make([]float32, from/2)
	for n := range in {
		in[n] = float32(math.Sin(2 * math.Pi * hz * float64(n) / from))
	}

	out, err := Convert(t.Context(), in, from, signal.SampleRate, 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := signal.SampleRate / 2; out == nil || len(out) < want-8 || len(out) > want+8 {
		t.Fatalf("half a second at %d hertz came out as %d samples", from, len(out))
	}

	// Away from the ends, where the kernel extends past the signal, each sample
	// equals the tone value at that time.
	for n := 64; n < len(out)-64; n++ {
		want := math.Sin(2 * math.Pi * hz * float64(n) / signal.SampleRate)
		if math.Abs(float64(out[n])-want) > 1e-2 {
			t.Fatalf("sample %d is %g, and the tone is %g there", n, out[n], want)
		}
	}
}

// A recording already at the rate the models take is left alone.
func TestResamplingWhatIsAlreadyRight(t *testing.T) {
	in := []float32{1, 2, 3}
	out, err := Convert(t.Context(), in, signal.SampleRate, signal.SampleRate, 0)
	if err != nil {
		t.Fatal(err)
	}
	if &out[0] != &in[0] {
		t.Error("a recording at the right rate was resampled")
	}
}

// An hour of audio is minutes of computation, and a canceled context must stop
// the conversion, and closing the window does not wait for it.
func TestConvert_StopsOnCanceledContext(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	stop()

	in := make([]float32, 44100*10)
	if _, err := Convert(ctx, in, 44100, signal.SampleRate, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("stopping the resampling gave %v", err)
	}
}
func TestTheResampledSignalIsTheSameWhateverTheWorkerCount(t *testing.T) {
	in := make([]float32, 4410)
	for i := range in {
		in[i] = float32(i%97) / 97
	}
	want, err := Convert(t.Context(), in, 44100, signal.SampleRate, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, workers := range []int{0, 3} {
		got, err := Convert(t.Context(), in, 44100, signal.SampleRate, workers)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%d workers resampled to another signal than one worker did (%v)", workers, err)
		}
	}
}

func TestResampleEdgeCasesAndKernel(t *testing.T) {
	if sincWidth != 16 || kernelSteps != 512 {
		t.Fatalf("constants = (%d, %d), want (16, 512)", sincWidth, kernelSteps)
	}
	ctx := t.Context()
	sample := []float32{1, 2}
	if out, err := Convert(ctx, sample, 0, 16000, 1); err != nil || !slices.Equal(out, sample) {
		t.Errorf("Convert(from=0) = %v, %v", out, err)
	}
	if out, err := Convert(ctx, sample, -1, 16000, 1); err != nil || !slices.Equal(out, sample) {
		t.Errorf("Convert(from=-1) = %v, %v", out, err)
	}
	if out, err := Convert(ctx, sample, 1, 2, 1); err != nil || len(out) != 4 {
		t.Errorf("Convert(from=1, to=2) = len %d, %v, want 4", len(out), err)
	}
	if out, err := Convert(ctx, []float32{}, 48000, 16000, 1); err != nil || len(out) != 0 {
		t.Errorf("Convert(empty) = %v, %v", out, err)
	}
	if out, err := Convert(ctx, []float32{1}, 10, 20, 1); err != nil || len(out) != 2 {
		t.Errorf("Convert(len=1) = len %d, %v, want 2", len(out), err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if out, err := Convert(canceled, []float32{1}, 16000, 8000, 1); err != nil || len(out) != 0 {
		t.Errorf("Convert(out len=0, canceled) = %v, %v, want empty nil", out, err)
	}
	if out, err := Convert(ctx, []float32{5}, 8000, 8001, 1); err != nil || len(out) != 1 || out[0] != 5 {
		t.Errorf("Convert(out len=1) = %v, %v, want [5]", out, err)
	}

	// Upsampling uses cutoff = 1.0 (not ratio = 1.5).
	up, err := Convert(ctx, []float32{0, 1, 0, -1}, 100, 150, 1)
	if err != nil {
		t.Fatal(err)
	}
	k1 := buildKernel(1.0, 16.0)
	w0 := k1.getValueAt(2.0 / 3.0)
	w1 := k1.getValueAt(2.0/3.0 - 1.0)
	w2 := k1.getValueAt(2.0/3.0 - 2.0)
	w3 := k1.getValueAt(2.0/3.0 - 3.0)
	wantUp1 := float32((w1 - w3) / (w0 + w1 + w2 + w3))
	if math.Abs(float64(up[1]-wantUp1)) > 1e-6 {
		t.Errorf("up[1] = %g, want %g", up[1], wantUp1)
	}

	// Downsampling by 2 widens reach to 32, reaching index 31 from index 0.
	impulse := make([]float32, 40)
	impulse[31] = 100
	down, err := Convert(ctx, impulse, 200, 100, 1)
	if err != nil || down[0] == 0 {
		t.Errorf("down[0] = %g (%v), want non-zero from impulse at 31", down[0], err)
	}

	// Direct span.fill boundary and zero-weight checks.
	out := []float32{0, 0}
	span := resampleSpan{
		in:     []float32{10, 20},
		out:    out,
		ratio:  1,
		reach:  1.5,
		kernel: kernel{values: []float64{1, 1, 1}, step: 1},
	}
	span.fill(ctx, 0, 2)
	if out[0] != 15 {
		t.Errorf("span.fill out[0] = %g, want 15", out[0])
	}
	zeroOut := []float32{99}
	zeroSpan := resampleSpan{
		in:     []float32{10},
		out:    zeroOut,
		ratio:  1,
		reach:  1,
		kernel: kernel{values: []float64{0, 0}, step: 1},
	}
	zeroSpan.fill(ctx, 0, 1)
	if zeroOut[0] != 99 {
		t.Errorf("zero weight overwrote out[0] = %g, want 99", zeroOut[0])
	}

	// Kernel and window functions.
	kern := buildKernel(0.5, 4)
	if len(kern.values) != 4*kernelSteps+2 {
		t.Errorf("len(kern.values) = %d, want %d", len(kern.values), 4*kernelSteps+2)
	}
	wantMid := getSinc(0.5) * getBlackmanWindow(0.25)
	if math.Abs(kern.values[kernelSteps]-wantMid) > 1e-12 {
		t.Errorf("kern.values[512] = %g, want %g", kern.values[kernelSteps], wantMid)
	}
	interp := kernel{values: []float64{10, 20, 30}, step: 1}
	if got := interp.getValueAt(0.25); got != 12.5 {
		t.Errorf("interp.getValueAt(0.25) = %g, want 12.5", got)
	}
	if got := interp.getValueAt(1.5); got != 25 {
		t.Errorf("interp.getValueAt(1.5) = %g, want 25", got)
	}
	if got := interp.getValueAt(2.0); got != 0 {
		t.Errorf("interp.getValueAt(2.0) = %g, want 0", got)
	}
	if getSinc(0) != 1 || math.Abs(getSinc(0.5)-2/math.Pi) > 1e-12 {
		t.Errorf("getSinc = (%g, %g)", getSinc(0), getSinc(0.5))
	}
	if getBlackmanWindow(-1.5) != 0 || getBlackmanWindow(1.5) != 0 {
		t.Errorf("getBlackmanWindow outside [-1,1] = (%g, %g)", getBlackmanWindow(-1.5), getBlackmanWindow(1.5))
	}
	if math.Abs(getBlackmanWindow(0)-1.0) > 1e-12 || math.Abs(getBlackmanWindow(0.5)-0.34) > 1e-12 || math.Abs(getBlackmanWindow(-0.5)-0.34) > 1e-12 {
		t.Errorf("getBlackmanWindow inside [-1,1] = (%g, %g, %g)", getBlackmanWindow(0), getBlackmanWindow(0.5), getBlackmanWindow(-0.5))
	}
}
