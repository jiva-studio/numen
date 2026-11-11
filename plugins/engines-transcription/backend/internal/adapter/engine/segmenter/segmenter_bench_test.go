package segmenter

import "testing"

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
