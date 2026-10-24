package engine

import (
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
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

func TestAnUnresolvedProfileConsumesTheFallbackCounts(t *testing.T) {
	cfg := Defaults()
	if got, want := cfg.threads(), min(fallbackThreads, onnx.GetDefaultThreads()); got != want {
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

func TestAnUnresolvedProfileConsumesFourOfEach(t *testing.T) {
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
	previous := runtime.GOMAXPROCS(4)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })

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

func TestWhatASettingsFilePreservesOutIsTheDefaultOfTheModel(t *testing.T) {
	if got := (LayoutModel{}).minimum(); got != 0.4 {
		t.Errorf("a part scores at least %v", got)
	}
	if got := (LayoutModel{}).overlap(); got != 0.6 {
		t.Errorf("two parts overlap by at most %v", got)
	}
	if got := (LayoutModel{}).margin(); got != 10 {
		t.Errorf("a part is widened by %d", got)
	}
	if got := (DetectModel{}).maxSide(); got != 1600 {
		t.Errorf("a part is read at %d", got)
	}
	if got := (DetectModel{}).getExpansion(); got != 18 {
		t.Errorf("a line is widened by %d", got)
	}
	if got := (DetectModel{}).minimum(); got != 0.3 {
		t.Errorf("a pixel is hot from %v", got)
	}
	if got := (RecogniserModel{}).dpi(); got != 300 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecogniserModel{}).height(); got != 48 {
		t.Errorf("a line is scaled to %d", got)
	}
}

func TestWhatASettingsFileNamesExists(t *testing.T) {
	if got := (LayoutModel{Minimum: 0.5}).minimum(); got != 0.5 {
		t.Errorf("a part scores at least %v", got)
	}
	if got := (LayoutModel{Overlap: 0.5}).overlap(); got != 0.5 {
		t.Errorf("two parts overlap by at most %v", got)
	}
	if got := (LayoutModel{Margin: 1}).margin(); got != 1 {
		t.Errorf("a part is widened by %d", got)
	}
	if got := (DetectModel{MaxSide: 1}).maxSide(); got != 1 {
		t.Errorf("a part is read at %d", got)
	}
	if got := (DetectModel{Expand: 1}).getExpansion(); got != 1 {
		t.Errorf("a line is widened by %d", got)
	}
	if got := (DetectModel{Minimum: 0.5}).minimum(); got != 0.5 {
		t.Errorf("a pixel is hot from %v", got)
	}
	if got := (RecogniserModel{DPI: 1}).dpi(); got != 1 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecogniserModel{Height: 1}).height(); got != 1 {
		t.Errorf("a line is scaled to %d", got)
	}
}

func TestANegativeNumberIsNoNumberAtAll(t *testing.T) {
	if got := (LayoutModel{Minimum: -1}).minimum(); got != 0.4 {
		t.Errorf("a part scores at least %v", got)
	}
	if got := (LayoutModel{Overlap: -1}).overlap(); got != 0.6 {
		t.Errorf("two parts overlap by at most %v", got)
	}
	if got := (LayoutModel{Margin: -1}).margin(); got != 10 {
		t.Errorf("a part is widened by %d", got)
	}
	if got := (DetectModel{MaxSide: -1}).maxSide(); got != 1600 {
		t.Errorf("a part is read at %d", got)
	}
	if got := (DetectModel{Expand: -1}).getExpansion(); got != 18 {
		t.Errorf("a line is widened by %d", got)
	}
	if got := (DetectModel{Minimum: -1}).minimum(); got != 0.3 {
		t.Errorf("a pixel is hot from %v", got)
	}
	if got := (RecogniserModel{DPI: -1}).dpi(); got != 300 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecogniserModel{Height: -1}).height(); got != 48 {
		t.Errorf("a line is scaled to %d", got)
	}
}

func TestThePartsOfAPageAreTheOnesTheModelKnowsUnlessTheFileReturnsOtherwise(t *testing.T) {
	labels := (LayoutModel{}).labels()
	if len(labels) != 25 || labels[0] != "abstract" || labels[24] != "vision_footnote" {
		t.Errorf("the model knows %v", labels)
	}
	if got := (LayoutModel{Labels: []string{"only"}}).labels(); !slices.Equal(got, []string{"only"}) {
		t.Errorf("the file names %v", got)
	}

	body := (RegionKinds{}).body()
	if !slices.Contains(body, "text") || !slices.Contains(body, "footnote") || len(body) != 9 {
		t.Errorf("a page carries what it says in %v", body)
	}
	if got := (RegionKinds{Body: []string{"only"}}).body(); !slices.Equal(got, []string{"only"}) {
		t.Errorf("the file names %v", got)
	}

	if got := (RegionKinds{}).head(); !slices.Equal(got, []string{"doc_title", "paragraph_title"}) {
		t.Errorf("a part is opened by %v", got)
	}
	if got := (RegionKinds{Head: []string{"only"}}).head(); !slices.Equal(got, []string{"only"}) {
		t.Errorf("the file names %v", got)
	}
}

func TestTheModelsAreFoundWhereTheConfigurationReturns(t *testing.T) {
	cfg := Config{
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
