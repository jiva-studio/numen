package onnx

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
)

const benchModelDirEnvVar = "NUMEN_TEST_MODEL_DIR"

func benchModelDir(t testing.TB) string {
	t.Helper()
	dir := os.Getenv(benchModelDirEnvVar)
	if dir == "" {
		t.Skipf("set %s to a directory with model.onnx and tokenizer.json", benchModelDirEnvVar)
	}
	return dir
}

func benchSetLocalModel(t testing.TB, cfg embed.Config, dir string) (embed.Config, embed.LocalModel) {
	t.Helper()
	local, ok := cfg.Indexing.Local()
	if !ok {
		t.Fatal("the settings run no model on this machine")
	}
	local.Dir = dir
	cfg.Indexing = cfg.Indexing.SetLocal(local)
	return cfg, local
}

func benchOpen(t testing.TB, dir string) *Embedder {
	t.Helper()
	cfg, local := benchSetLocalModel(t, embed.Defaults(), dir)
	e, err := Open(t.Context(), cfg.GetStoredModel(), local, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func BenchmarkTokenizeBatch(b *testing.B) {
	e := benchOpen(b, benchModelDir(b))
	sample := strings.Repeat("The quick brown fox jumps over the lazy dog. A paragraph with multiple sentences for embedding. ", 10)

	for _, count := range []int{1, 8, 16, 32, 64, 128, 256, 512} {
		texts := make([]string, count)
		for i := range texts {
			texts[i] = fmt.Sprintf("%s chunk %d", sample, i)
		}

		b.Run(fmt.Sprintf("texts=%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = e.tokenize(texts)
			}
		})
	}
}

func BenchmarkEmbedBatch(b *testing.B) {
	e := benchOpen(b, benchModelDir(b))
	sample := strings.Repeat("The quick brown fox jumps over the lazy dog. A paragraph with multiple sentences for embedding. ", 10)

	for _, count := range []int{1, 8, 16, 32} {
		texts := make([]string, count)
		for i := range texts {
			texts[i] = fmt.Sprintf("%s chunk %d", sample, i)
		}

		b.Run(fmt.Sprintf("texts=%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := e.Embed(b.Context(), texts); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEmbedVariableLengthBatch(b *testing.B) {
	e := benchOpen(b, benchModelDir(b))
	sample := "The quick brown fox jumps over the lazy dog."

	for _, count := range []int{32, 64, 128} {
		texts := make([]string, count)
		for i := range texts {
			repeats := (i % 20) + 1
			texts[i] = strings.Repeat(sample+" ", repeats)
		}

		b.Run(fmt.Sprintf("texts=%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := e.Embed(b.Context(), texts); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
