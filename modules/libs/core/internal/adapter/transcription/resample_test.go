package transcription

import (
	"context"
	"errors"
	"math"
	"testing"
)

// A recording at another rate is brought to the one the models take, and the
// tone it carries is still the tone it carried.
func TestResamplingKeepsTheTone(t *testing.T) {
	const from, hz = 48000, 440.0
	in := make([]float32, from/2)
	for n := range in {
		in[n] = float32(math.Sin(2 * math.Pi * hz * float64(n) / from))
	}

	out, err := resampled(t.Context(), in, from, sampleRate)
	if err != nil {
		t.Fatal(err)
	}
	if want := sampleRate / 2; out == nil || len(out) < want-8 || len(out) > want+8 {
		t.Fatalf("half a second at %d hertz came out as %d samples", from, len(out))
	}

	// Away from the ends, where the kernel reaches past the signal, the sample
	// is what the tone says at that moment.
	for n := 64; n < len(out)-64; n++ {
		want := math.Sin(2 * math.Pi * hz * float64(n) / sampleRate)
		if math.Abs(float64(out[n])-want) > 1e-2 {
			t.Fatalf("sample %d is %g, and the tone is %g there", n, out[n], want)
		}
	}
}

// A recording already at the rate the models take is left alone.
func TestResamplingWhatIsAlreadyRight(t *testing.T) {
	in := []float32{1, 2, 3}
	out, err := resampled(t.Context(), in, sampleRate, sampleRate)
	if err != nil {
		t.Fatal(err)
	}
	if &out[0] != &in[0] {
		t.Error("a recording at the right rate was resampled")
	}
}

// An hour of sound is minutes of arithmetic, and a person closing the window
// waits for none of it.
func TestResamplingStopsWhenAsked(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	stop()

	in := make([]float32, 44100*10)
	if _, err := resampled(ctx, in, 44100, sampleRate); !errors.Is(err, context.Canceled) {
		t.Errorf("stopping the resampling gave %v", err)
	}
}
