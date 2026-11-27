package settings

import (
	"slices"
	"testing"
)

func TestUnsetThreadsAndJobsTakeTheResolvedProfile(t *testing.T) {
	cfg := NewDefaultConfig()
	if got := cfg.GetThreads(8); got != 4 {
		t.Errorf("default fallback threads with 8 cores = %d, want 4", got)
	}
	if got := cfg.GetThreads(2); got != 2 {
		t.Errorf("default fallback threads with 2 cores = %d, want 2", got)
	}
	if got := cfg.GetJobs(); got != 4 {
		t.Errorf("default fallback jobs = %d, want 4", got)
	}

	cfg.Compute = Compute{Threads: 1, Jobs: 1}
	if got := cfg.GetThreads(8); got != 1 {
		t.Errorf("the recognizer takes %d threads, want 1", got)
	}
	if got := cfg.GetJobs(); got != 1 {
		t.Errorf("the recognizer runs %d jobs, want 1", got)
	}
}

func TestExplicitThreadsAndJobsWinOverTheResolvedProfile(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Compute = Compute{Threads: 6, Jobs: 3}
	cfg.Recognize.Threads, cfg.Recognize.Jobs = 1, 1
	if got := cfg.GetThreads(8); got != 1 {
		t.Errorf("the recognizer takes %d threads, want 1", got)
	}
	if got := cfg.GetJobs(); got != 1 {
		t.Errorf("the recognizer runs %d jobs, want 1", got)
	}
}

func TestTheDefaultsReadAtThreeHundredDPIAndFetchWhatIsMissing(t *testing.T) {
	cfg := NewDefaultConfig()
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
	for name, ids := range map[string][2]string{
		"layout":     {cfg.Layout.Name, "pp-doclayout"},
		"detector":   {cfg.Detect.Name, "pp-ocr-det"},
		"recognizer": {cfg.Recognize.Name, "rec-pp-ocrv6-tiny"},
	} {
		if ids[0] != ids[1] {
			t.Errorf("the %s model is %q, want the dependency %s of the manifest", name, ids[0], ids[1])
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
	if got := (RecognizerModel{}).GetDPI(); got != 300 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecognizerModel{}).GetHeight(); got != 48 {
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
	if got := (RecognizerModel{DPI: 1}).GetDPI(); got != 1 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecognizerModel{Height: 1}).GetHeight(); got != 1 {
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
	if got := (RecognizerModel{DPI: -1}).GetDPI(); got != 300 {
		t.Errorf("a page is rendered at %d", got)
	}
	if got := (RecognizerModel{Height: -1}).GetHeight(); got != 48 {
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
