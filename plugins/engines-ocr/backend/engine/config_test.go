package engine

import (
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

func TestAnUnresolvedProfileConsumesTheFallbackCounts(t *testing.T) {
	cfg := settings.Defaults()
	if got, want := getThreads(cfg), min(settings.FallbackThreads, onnx.GetDefaultThreads()); got != want {
		t.Errorf("the recogniser takes %d threads, want %d", got, want)
	}
	if got := cfg.GetJobs(); got != settings.FallbackJobs {
		t.Errorf("the recogniser runs %d jobs, want %d", got, settings.FallbackJobs)
	}
}

func TestAnUnresolvedProfileConsumesFourOfEach(t *testing.T) {
	previous := runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	cfg := settings.Defaults()
	if got := getThreads(cfg); got != 4 {
		t.Errorf("the recogniser takes %d threads, want 4", got)
	}
	if got := cfg.GetJobs(); got != 4 {
		t.Errorf("the recogniser runs %d jobs, want 4", got)
	}
}

func TestAFallbackThreadCountStaysWithinTheCoresThereAre(t *testing.T) {
	previous := runtime.GOMAXPROCS(2)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	if got := getThreads(settings.Defaults()); got != 2 {
		t.Errorf("the recogniser takes %d threads on two cores, want 2", got)
	}
}

func TestOneThreadAndOneJobAreCountsToo(t *testing.T) {
	previous := runtime.GOMAXPROCS(4)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	cfg := settings.Defaults()
	cfg.Compute = settings.Compute{Threads: 6, Jobs: 3}
	cfg.Recognize.Threads, cfg.Recognize.Jobs = 1, 1
	if got := getThreads(cfg); got != 1 {
		t.Errorf("an explicit one thread gave %d", got)
	}
	if got := cfg.GetJobs(); got != 1 {
		t.Errorf("an explicit one job gave %d", got)
	}

	cfg.Recognize = settings.RecogniserModel{}
	cfg.Compute = settings.Compute{Threads: 1, Jobs: 1}
	if got := getThreads(cfg); got != 1 {
		t.Errorf("a profile of one thread gave %d", got)
	}
	if got := cfg.GetJobs(); got != 1 {
		t.Errorf("a profile of one job gave %d", got)
	}
}

func TestTheModelsAreFoundWhereTheConfigurationReturns(t *testing.T) {
	cfg := settings.Config{
		Runtime:        "lib.so",
		Dir:            "models",
		ShouldDownload: true,
	}
	if got := onnx.ResolveLibrary(cfg.Runtime, cfg.Dir); got != "lib.so" {
		t.Errorf("runtime = %q, want lib.so", got)
	}
}

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for name, want := range map[string]onnx.Provider{"": onnx.ProviderCPU, "auto": onnx.ProviderCPU, "cuda": onnx.ProviderCUDA} {
		t.Setenv("NUMEN_ONNX_PROVIDER", name)
		if got := onnx.GetSessionProvider(); got != want {
			t.Errorf("with %q, GetSessionProvider() = %q, want %q", name, got, want)
		}
	}
}
