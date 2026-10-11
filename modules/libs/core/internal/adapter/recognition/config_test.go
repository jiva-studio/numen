package recognition

import (
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

func TestUnsetThreadsAndJobsTakeTheResolvedProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	if got := cfg.threads(); got != 6 {
		t.Errorf("the recogniser takes %d threads, want 6", got)
	}
	if got := cfg.jobs(); got != 3 {
		t.Errorf("the recogniser runs %d jobs, want 3", got)
	}
}

func TestExplicitThreadsAndJobsWinOverTheResolvedProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	cfg.Recognise.Threads, cfg.Recognise.Jobs = 2, 5
	if got := cfg.threads(); got != 2 {
		t.Errorf("the recogniser takes %d threads, want 2", got)
	}
	if got := cfg.jobs(); got != 5 {
		t.Errorf("the recogniser runs %d jobs, want 5", got)
	}
}

func TestAnUnresolvedProfileTakesTheFallbackCounts(t *testing.T) {
	cfg := Defaults()
	if got, want := cfg.threads(), min(fallbackThreads, onnxruntime.DefaultThreads()); got != want {
		t.Errorf("the recogniser takes %d threads, want %d", got, want)
	}
	if got := cfg.jobs(); got != fallbackJobs {
		t.Errorf("the recogniser runs %d jobs, want %d", got, fallbackJobs)
	}
}

func TestTheDefaultsReadAtThreeHundredDPIAndFetchWhatIsMissing(t *testing.T) {
	cfg := Defaults()
	if cfg.Recognise.DPI != 300 {
		t.Errorf("a page is rendered at %d dpi, want 300", cfg.Recognise.DPI)
	}
	if !cfg.ShouldDownload {
		t.Error("the defaults do not fetch a missing model")
	}
	for name, got := range map[string]string{
		"layout":     cfg.Layout.Name,
		"detector":   cfg.Detect.Name,
		"recogniser": cfg.Recognise.Name,
	} {
		if !strings.HasPrefix(got, "https://") || !strings.HasSuffix(got, ".onnx") {
			t.Errorf("the %s model is %q, want an address of an onnx file", name, got)
		}
	}
}

func TestAnUnresolvedProfileTakesFourOfEach(t *testing.T) {
	previous := runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	cfg := Defaults()
	if got := cfg.threads(); got != 4 {
		t.Errorf("the recogniser takes %d threads, want 4", got)
	}
	if got := cfg.jobs(); got != 4 {
		t.Errorf("the recogniser runs %d jobs, want 4", got)
	}
}

func TestAFallbackThreadCountStaysWithinTheCoresThereAre(t *testing.T) {
	previous := runtime.GOMAXPROCS(2)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

	if got := Defaults().threads(); got != 2 {
		t.Errorf("the recogniser takes %d threads on two cores, want 2", got)
	}
}

func TestOneThreadAndOneJobAreCountsToo(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	cfg.Recognise.Threads, cfg.Recognise.Jobs = 1, 1
	if got := cfg.threads(); got != 1 {
		t.Errorf("an explicit one thread gave %d", got)
	}
	if got := cfg.jobs(); got != 1 {
		t.Errorf("an explicit one job gave %d", got)
	}

	cfg.Recognise = RecogniserModel{}
	cfg.Compute = Compute{Threads: 1, Jobs: 1}
	if got := cfg.threads(); got != 1 {
		t.Errorf("a profile of one thread gave %d", got)
	}
	if got := cfg.jobs(); got != 1 {
		t.Errorf("a profile of one job gave %d", got)
	}
}
