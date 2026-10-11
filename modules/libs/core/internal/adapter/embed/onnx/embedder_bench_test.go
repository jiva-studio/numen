package onnx

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/benchenv"
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

// benchPassesVar is how many forward passes run at once in
// BenchmarkEmbedConcurrent; NUMEN_BENCH_THREADS is the model's thread count.
const benchPassesVar = "NUMEN_BENCH_PASSES"

// BenchmarkEmbedConcurrent embeds one full batch from each of several callers at
// once and reports the texts embedded a second.
func BenchmarkEmbedConcurrent(b *testing.B) {
	dir := benchModelDir(b)
	cfg, local := benchSetLocalModel(b, embed.Defaults(), dir)
	if raw := os.Getenv(benchPassesVar); raw != "" {
		passes, err := strconv.Atoi(raw)
		if err != nil || passes < 1 {
			b.Fatalf("%s: %q is not a positive integer", benchPassesVar, raw)
		}
		local.Sessions = passes
	}
	if raw := os.Getenv(benchenv.ThreadsVar); raw != "" {
		threads, err := strconv.Atoi(raw)
		if err != nil || threads < 1 {
			b.Fatalf("%s: %q is not a positive integer", benchenv.ThreadsVar, raw)
		}
		local.Threads = threads
	}
	e, err := Open(b.Context(), cfg.GetStoredModel(), local, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = e.Close() })

	callers := e.Concurrency()
	texts := make([]string, local.BatchTexts)
	for i := range texts {
		texts[i] = strings.Repeat("The quick brown fox jumps over the lazy dog. ", (i%20)+1)
	}

	b.ResetTimer()
	for range b.N {
		var wg sync.WaitGroup
		for range callers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := e.Embed(b.Context(), texts); err != nil {
					b.Error(err)
				}
			}()
		}
		wg.Wait()
	}
	b.ReportMetric(float64(b.N*callers*len(texts))/b.Elapsed().Seconds(), "texts/s")
}
