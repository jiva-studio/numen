package settings

import (
	"slices"
	"strings"
	"testing"
)

func TestUnsetThreadsAndJobsTakeTheResolvedProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	if got := cfg.GetThreads(8); got != 6 {
		t.Errorf("the recogniser takes %d threads, want 6", got)
	}
	if got := cfg.GetJobs(); got != 3 {
		t.Errorf("the recogniser runs %d jobs, want 3", got)
	}
}

func TestExplicitThreadsAndJobsWinOverTheResolvedProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	cfg.Recognize.Threads, cfg.Recognize.Jobs = 2, 5
	if got := cfg.GetThreads(8); got != 2 {
		t.Errorf("the recogniser takes %d threads, want 2", got)
	}
	if got := cfg.GetJobs(); got != 5 {
		t.Errorf("the recogniser runs %d jobs, want 5", got)
	}
}

func TestTheDefaultsReadAtThreeHundredDPIAndFetchWhatIsMissing(t *testing.T) {
	cfg := Defaults()
	if cfg.Recognize.DPI != 300 {
		t.Errorf("a page is rendered at %d dpi, want 300", cfg.Recognize.DPI)
	}
	if !cfg.ShouldDownload {
		t.Error("the defaults do not fetch a missing model")
	}
	if cfg.Layout.Minimum != 0 || cfg.Layout.Overlap != 0 || cfg.Layout.Margin != 0 {
		t.Errorf("unexpected layout numeric defaults: %+v", cfg.Layout)
	}
	if cfg.Detect.MaxSide != 0 || cfg.Detect.Expand != 0 || cfg.Detect.Minimum != 0 {
		t.Errorf("unexpected detect numeric defaults: %+v", cfg.Detect)
	}
	if cfg.Recognize.Classes != 0 || cfg.Recognize.Height != 0 || cfg.Recognize.Jobs != 0 || cfg.Recognize.Threads != 0 {
		t.Errorf("unexpected recognize numeric defaults: %+v", cfg.Recognize)
	}
	if cfg.Compute.Threads != 0 || cfg.Compute.Jobs != 0 {
		t.Errorf("unexpected compute defaults: %+v", cfg.Compute)
	}
	for name, got := range map[string]string{
		"layout":     cfg.Layout.Name,
		"detector":   cfg.Detect.Name,
		"recogniser": cfg.Recognize.Name,
	} {
		if !strings.HasPrefix(got, "https://") || !strings.HasSuffix(got, ".onnx") {
			t.Errorf("the %s model is %q, want an address of an onnx file", name, got)
		}
	}
}

func TestWhatASettingsFilePreservesOutIsTheDefaultOfTheModel(t *testing.T) {
	if got := (LayoutModel{}).GetMinimum(); got != 0.4 {
		t.Errorf("a part scores at least %v", got)
	}
	if got := (LayoutModel{}).GetOverlap(); got != 0.6 {
		t.Errorf("two parts overlap by at most %v", got)
	}
	if got := (LayoutModel{}).GetMargin(); got != 10 {
		t.Errorf("a part is widened by %d", got)
	}
	if got := (DetectModel{}).GetMaxSide(); got != 1600 {
		t.Errorf("a part is read at %d", got)
	}
	if got := (DetectModel{}).GetExpansion(); got != 18 {
		t.Errorf("a line is widened by %d", got)
	}
	if got := (DetectModel{}).GetMinimum(); got != 0.3 {
		t.Errorf("a pixel is hot from %v", got)
	}
	if got := (RecogniserModel{}).GetDPI(); got != 300 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecogniserModel{}).GetHeight(); got != 48 {
		t.Errorf("a line is scaled to %d", got)
	}
}

func TestWhatASettingsFileNamesExists(t *testing.T) {
	if got := (LayoutModel{Minimum: 0.5}).GetMinimum(); got != 0.5 {
		t.Errorf("a part scores at least %v", got)
	}
	if got := (LayoutModel{Overlap: 0.5}).GetOverlap(); got != 0.5 {
		t.Errorf("two parts overlap by at most %v", got)
	}
	if got := (LayoutModel{Margin: 1}).GetMargin(); got != 1 {
		t.Errorf("a part is widened by %d", got)
	}
	if got := (DetectModel{MaxSide: 1}).GetMaxSide(); got != 1 {
		t.Errorf("a part is read at %d", got)
	}
	if got := (DetectModel{Expand: 1}).GetExpansion(); got != 1 {
		t.Errorf("a line is widened by %d", got)
	}
	if got := (DetectModel{Minimum: 0.5}).GetMinimum(); got != 0.5 {
		t.Errorf("a pixel is hot from %v", got)
	}
	if got := (RecogniserModel{DPI: 1}).GetDPI(); got != 1 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecogniserModel{Height: 1}).GetHeight(); got != 1 {
		t.Errorf("a line is scaled to %d", got)
	}
}

func TestANegativeNumberIsNoNumberAtAll(t *testing.T) {
	if got := (LayoutModel{Minimum: -1}).GetMinimum(); got != 0.4 {
		t.Errorf("a part scores at least %v", got)
	}
	if got := (LayoutModel{Overlap: -1}).GetOverlap(); got != 0.6 {
		t.Errorf("two parts overlap by at most %v", got)
	}
	if got := (LayoutModel{Margin: -1}).GetMargin(); got != 10 {
		t.Errorf("a part is widened by %d", got)
	}
	if got := (DetectModel{MaxSide: -1}).GetMaxSide(); got != 1600 {
		t.Errorf("a part is read at %d", got)
	}
	if got := (DetectModel{Expand: -1}).GetExpansion(); got != 18 {
		t.Errorf("a line is widened by %d", got)
	}
	if got := (DetectModel{Minimum: -1}).GetMinimum(); got != 0.3 {
		t.Errorf("a pixel is hot from %v", got)
	}
	if got := (RecogniserModel{DPI: -1}).GetDPI(); got != 300 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecogniserModel{Height: -1}).GetHeight(); got != 48 {
		t.Errorf("a line is scaled to %d", got)
	}
}

func TestLayoutLabels_DefaultsUnlessFileOverrides(t *testing.T) {
	labels := (LayoutModel{}).GetLabels()
	if len(labels) != 25 || labels[0] != "abstract" || labels[24] != "vision_footnote" {
		t.Errorf("the model knows %v", labels)
	}
	if got := (LayoutModel{Labels: []string{"only"}}).GetLabels(); !slices.Equal(got, []string{"only"}) {
		t.Errorf("the file names %v", got)
	}

	body := (RegionKinds{}).GetBody()
	if !slices.Contains(body, "text") || !slices.Contains(body, "footnote") || len(body) != 9 {
		t.Errorf("a page carries what it says in %v", body)
	}
	if got := (RegionKinds{Body: []string{"only"}}).GetBody(); !slices.Equal(got, []string{"only"}) {
		t.Errorf("the file names %v", got)
	}

	if got := (RegionKinds{}).GetHead(); !slices.Equal(got, []string{"doc_title", "paragraph_title"}) {
		t.Errorf("a part is opened by %v", got)
	}
	if got := (RegionKinds{Head: []string{"only"}}).GetHead(); !slices.Equal(got, []string{"only"}) {
		t.Errorf("the file names %v", got)
	}
}
