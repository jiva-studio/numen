// Package benchenv reads the environment a device benchmark is configured by.
// Only _test.go files import it.
//
// Every variable is optional:
//
//	NUMEN_BENCH_PROVIDER         cpu, coreml, auto, cuda or directml; default cpu
//	NUMEN_BENCH_THREADS          intra-op threads; default onnxruntime.DefaultThreads
//	NUMEN_BENCH_COMPUTE          ALL, CPUAndNeuralEngine, CPUAndGPU or CPUOnly; default the provider's own
//	NUMEN_BENCH_MODELS           a directory of .onnx files; default the application's model cache
//	NUMEN_BENCH_EMBEDDING_MODEL  a directory holding an embedding model.onnx; default none
//	NUMEN_BENCH_RUNTIME          a libonnxruntime file; default the search order of onnxruntime.Open
package benchenv

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// The variables a benchmark reads.
const (
	ProviderVar       = "NUMEN_BENCH_PROVIDER"
	ThreadsVar        = "NUMEN_BENCH_THREADS"
	ComputeVar        = "NUMEN_BENCH_COMPUTE"
	ModelsVar         = "NUMEN_BENCH_MODELS"
	EmbeddingModelVar = "NUMEN_BENCH_EMBEDDING_MODEL"
	RuntimeVar        = "NUMEN_BENCH_RUNTIME"
)

// Config is what one benchmark run is configured with.
type Config struct {
	Session onnxruntime.SessionSettings
	// Models is the directory the model files are looked for in.
	Models string
	// Embedding is the directory holding an embedding model, or empty.
	Embedding string
	// Runtime is the library file to open, or empty.
	Runtime string
}

// Read returns the configuration the environment gives, and fails the test on
// a value that does not parse.
func Read(tb testing.TB) Config {
	tb.Helper()

	provider, err := onnxruntime.ParseProvider(lookup(ProviderVar, string(onnxruntime.ProviderCPU)))
	if err != nil {
		tb.Fatalf("%s: %v", ProviderVar, err)
	}

	threads := onnxruntime.DefaultThreads()
	if raw := lookup(ThreadsVar, ""); raw != "" {
		threads, err = strconv.Atoi(raw)
		if err != nil || threads < 1 {
			tb.Fatalf("%s: %q is not a positive integer", ThreadsVar, raw)
		}
	}

	units, err := onnxruntime.ParseComputeUnits(lookup(ComputeVar, ""))
	if err != nil {
		tb.Fatalf("%s: %v", ComputeVar, err)
	}

	models := lookup(ModelsVar, "")
	if models == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			tb.Skipf("no model directory: set %s (%v)", ModelsVar, err)
		}
		models = filepath.Join(cache, "numen", "models")
	}

	return Config{
		Session: onnxruntime.SessionSettings{
			Provider:     provider,
			Threads:      threads,
			ComputeUnits: units,
		},
		Models:    models,
		Embedding: lookup(EmbeddingModelVar, ""),
		Runtime:   lookup(RuntimeVar, ""),
	}
}

// ListModels returns the .onnx files in the models directory, sorted by name,
// and skips the test when there are none.
func (c Config) ListModels(tb testing.TB) []string {
	tb.Helper()
	paths, err := filepath.Glob(filepath.Join(c.Models, "*.onnx"))
	if err != nil || len(paths) == 0 {
		tb.Skipf("no .onnx files in %s: set %s to a directory that holds the models", c.Models, ModelsVar)
	}
	return paths
}

// GetEmbeddingModel returns the model.onnx in the embedding directory, and
// skips the test when none is given or it holds none.
func (c Config) GetEmbeddingModel(tb testing.TB) string {
	tb.Helper()
	if c.Embedding == "" {
		tb.Skipf("set %s to a directory that holds an embedding model.onnx", EmbeddingModelVar)
	}
	path := filepath.Join(c.Embedding, "model.onnx")
	if _, err := os.Stat(path); err != nil {
		tb.Skipf("no model.onnx in %s: %v", c.Embedding, err)
	}
	return path
}

// GetModelName returns the name a model file is reported under: its base name
// without the extension and without the hash prefix the model cache gives it.
func GetModelName(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	head, rest, found := strings.Cut(name, "-")
	if found && len(head) == 12 && isHex(head) {
		return rest
	}
	return name
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func lookup(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
