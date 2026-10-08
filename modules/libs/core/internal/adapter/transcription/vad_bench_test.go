package transcription

import "testing"

// A run of audio scored through Silero VAD.
func BenchmarkSileroVAD(b *testing.B) {
	cfg := Defaults()
	cfg.ShouldDownload = false
	by, err := Open(b.Context(), cfg)
	if err != nil {
		b.Skipf("silero VAD model not available: %v", err)
	}
	defer by.Close()

	// 5 seconds of 16kHz audio = 80,000 samples (~156 windows of 512 samples)
	audio := make([]float32, 16000*5)
	for i := range audio {
		audio[i] = float32((i%2001)-1000) / 1000
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		scores, err := by.scoreSpeech(b.Context(), audio)
		if err != nil {
			b.Fatal(err)
		}
		if len(scores) == 0 {
			b.Fatal("no scores returned")
		}
	}
}

// A run of scores cut into spans.
func BenchmarkGetSpans(b *testing.B) {
	scores := make([]float32, 10000)
	for i := range scores {
		if (i/50)%2 == 0 {
			scores[i] = 0.8
		} else {
			scores[i] = 0.1
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		spans := getSpans(scores, 0.5, 10, 5, 500, 2)
		if len(spans) == 0 {
			b.Fatal("no spans found")
		}
	}
}
