package transcription

import (
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// A run of encoder frames decoded through the Parakeet predictor and joiner.
func BenchmarkParakeetGreedyLoop(b *testing.B) {
	cfg := Defaults()
	cfg.ShouldDownload = false
	by, err := Open(b.Context(), cfg)
	if err != nil {
		b.Skipf("parakeet models not available: %v", err)
	}
	defer by.Close()

	const width = 100
	data := make([]float32, encoded*width)
	framesTensor, err := ort.NewTensor([]int64{1, encoded, width}, data)
	if err != nil {
		b.Fatal(err)
	}
	defer framesTensor.Destroy()

	out := map[string]*ort.Value{
		outputFrames: framesTensor,
	}
	ctx := b.Context()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := by.decode(ctx, out)
		if err != nil {
			b.Fatal(err)
		}
	}
}
