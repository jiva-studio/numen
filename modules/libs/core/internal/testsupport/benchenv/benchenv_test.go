package benchenv_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/benchenv"
)

func TestTheDefaultsRunTheCPUOnTheDefaultThreads(t *testing.T) {
	for _, name := range []string{benchenv.ProviderVar, benchenv.ThreadsVar, benchenv.ComputeVar, benchenv.ModelsVar, benchenv.EmbeddingModelVar} {
		t.Setenv(name, "")
	}
	got := benchenv.Read(t)
	if got.Session.Provider != onnxruntime.ProviderCPU {
		t.Errorf("provider = %q, want cpu", got.Session.Provider)
	}
	if got.Session.Threads != onnxruntime.DefaultThreads() {
		t.Errorf("threads = %d, want %d", got.Session.Threads, onnxruntime.DefaultThreads())
	}
	if got.Session.ComputeUnits != onnxruntime.ComputeUnitsDefault {
		t.Errorf("compute units = %q, want the default", got.Session.ComputeUnits)
	}
	if got.Embedding != "" {
		t.Errorf("embedding = %q, want none", got.Embedding)
	}
}

func TestTheVariablesSetTheSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(benchenv.ProviderVar, "CoreML")
	t.Setenv(benchenv.ThreadsVar, "3")
	t.Setenv(benchenv.ComputeVar, "cpuandneuralengine")
	t.Setenv(benchenv.ModelsVar, dir)
	t.Setenv(benchenv.EmbeddingModelVar, dir)

	got := benchenv.Read(t)
	want := onnxruntime.SessionSettings{
		Provider:     onnxruntime.ProviderCoreML,
		Threads:      3,
		ComputeUnits: onnxruntime.ComputeUnitsCPUAndNeuralEngine,
	}
	if got.Session != want {
		t.Errorf("session = %+v, want %+v", got.Session, want)
	}
	if got.Models != dir || got.Embedding != dir {
		t.Errorf("directories = %q and %q, want %q", got.Models, got.Embedding, dir)
	}
}

func TestAModelDirectoryWithoutModelsSkipsTheBenchmark(t *testing.T) {
	t.Setenv(benchenv.ModelsVar, t.TempDir())
	t.Run("skipped", func(t *testing.T) {
		benchenv.Read(t).ListModels(t)
		t.Error("the listing did not skip")
	})
}

func TestAnEmbeddingDirectoryWithoutAModelSkipsTheBenchmark(t *testing.T) {
	t.Setenv(benchenv.EmbeddingModelVar, t.TempDir())
	t.Run("skipped", func(t *testing.T) {
		benchenv.Read(t).GetEmbeddingModel(t)
		t.Error("the lookup did not skip")
	})
}

func TestTheModelsOfADirectoryAreListedByName(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.onnx", "a.onnx", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(benchenv.ModelsVar, dir)
	got := benchenv.Read(t).ListModels(t)
	if len(got) != 2 || filepath.Base(got[0]) != "a.onnx" || filepath.Base(got[1]) != "b.onnx" {
		t.Errorf("listed %v, want a.onnx and b.onnx", got)
	}
}

func TestAModelIsNamedWithoutItsCacheHash(t *testing.T) {
	for path, want := range map[string]string{
		"/m/f034fa8b0a07-PP-OCRv6_rec_tiny.onnx": "PP-OCRv6_rec_tiny",
		"/m/model.onnx":                          "model",
		"/m/not-a-hash.onnx":                     "not-a-hash",
	} {
		if got := benchenv.GetModelName(path); got != want {
			t.Errorf("GetModelName(%q) = %q, want %q", path, got, want)
		}
	}
}
