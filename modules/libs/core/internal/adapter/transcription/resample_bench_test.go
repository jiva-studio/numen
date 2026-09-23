package transcription

import "testing"

// A minute of 44.1 kHz sound, brought to what the model takes.
func BenchmarkResampledAMinute(b *testing.B) {
	in := make([]float32, 44100*60)
	for i := range in {
		in[i] = float32((i%2001)-1000) / 1000
	}
	b.ResetTimer()
	for b.Loop() {
		_, _ = resample(b.Context(), in, 44100, sampleRate)
	}
}
