package engine

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// A run of encoder frames decoded through the Parakeet predictor and joiner.
func BenchmarkParakeetGreedyLoop(b *testing.B) {
	cfg := Defaults()
	cfg.ShouldDownload = false
	delta, err := Open(b.Context(), cfg)
	if err != nil {
		b.Skipf("parakeet models not available: %v", err)
	}
	defer cleanup.Close(b, delta)

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
		_, err := delta.decode(ctx, out)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// A run of audio cut by the segmenter, scored through Silero VAD.
func BenchmarkSileroVAD(b *testing.B) {
	cfg := Defaults()
	cfg.ShouldDownload = false
	delta, err := Open(b.Context(), cfg)
	if err != nil {
		b.Skipf("silero VAD model not available: %v", err)
	}
	defer cleanup.Close(b, delta)

	// 5 seconds of 16kHz audio = 80,000 samples (~156 windows of 512 samples)
	audio := make([]float32, 16000*5)
	for i := range audio {
		audio[i] = float32((i%2001)-1000) / 1000
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := delta.cutter.Split(b.Context(), audio)
		if err != nil {
			b.Fatal(err)
		}
	}
}
