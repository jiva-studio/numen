package recognition

import (
	"errors"
	"image"
	"strings"
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// openLayoutOn opens the layout model held in body on the runtime of this
// machine.
func openLayoutOn(t *testing.T, body []byte, labels []string, minimum float32) (*Layout, error) {
	t.Helper()
	isolate(t)
	cfg := Defaults()
	cfg.ShouldDownload = false
	engine, _, err := onnxruntime.Open(t.Context(), cfg.settings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	options, _, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(options.Destroy)
	return OpenLayout(engine, writeModel(t, t.TempDir(), "layout.onnx", body), options, labels, minimum, 0.6)
}

// The model is asked at 800 by 800, and the scale it is given is how far the
// page is from that: first the height's, then the width's.
func TestTheLayoutModelIsAskedTheShapeAndTheScaleOfThePage(t *testing.T) {
	labels := make([]string, 801)
	labels[800] = "body"
	layout, err := openLayoutOn(t, echoModel([3]float32{150, 300, 5}), labels, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer layout.Close()

	got, err := layout.Regions(image.NewRGBA(image.Rect(0, 0, 200, 400)))
	if err != nil {
		t.Fatal(err)
	}
	want := []ocr.Region{{Label: "body", Score: 800, Rect: image.Rect(2, 4, 150, 300), Order: 5}}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("regions = %+v, want %+v", got, want)
	}
}

// A page is divided into the rows the model answers, and only the parts that
// are parts are kept, in the order the page is read.
func TestTheRegionsAreTheRowsTheModelAnswersThatAreParts(t *testing.T) {
	rows := [][7]float32{
		{0, 0.9, 10, 10, 100, 50, 2},    // kept
		{1, 0.9, 120, 10, 190, 50, 1},   // kept, and read first
		{0, 0.1, 10, 200, 100, 240, 9},  // too sure of nothing
		{2, 0.9, 10, 200, 100, 240, 9},  // a class with no name
		{-1, 0.9, 10, 200, 100, 240, 9}, // a class before the first
		{0, 0.9, 60, 60, 60, 90, 9},     // no area
		{1, 0.8, 190, 300, 120, 250, 3}, // corners given the wrong way round
		{0, 0.6, 11, 11, 100, 50, 9},    // the first part found again
		{0, 0.9, 300, 10, 400, 50, 9},   // off the page
		{0, 0.5, 10, 100, 100, 140, 6},  // exactly as sure as is asked
		{0, 0.7, 150, 350, 300, 500, 4}, // partly off the page
	}
	var values []float32
	for _, row := range rows {
		values = append(values, row[:]...)
	}
	layout, err := openLayoutOn(t, tableModel(outputBoxes, layoutSide, values), []string{"alpha", "beta"}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	defer layout.Close()

	got, err := layout.Regions(image.NewRGBA(image.Rect(0, 0, 200, 400)))
	if err != nil {
		t.Fatal(err)
	}
	want := []ocr.Region{
		{Label: "beta", Score: 0.9, Rect: image.Rect(120, 10, 190, 50), Order: 1},
		{Label: "alpha", Score: 0.9, Rect: image.Rect(10, 10, 100, 50), Order: 2},
		{Label: "beta", Score: 0.8, Rect: image.Rect(120, 250, 190, 300), Order: 3},
		{Label: "alpha", Score: 0.7, Rect: image.Rect(150, 350, 200, 400), Order: 4},
		{Label: "alpha", Score: 0.5, Rect: image.Rect(10, 100, 100, 140), Order: 6},
	}
	if len(got) != len(want) {
		t.Fatalf("regions = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("region %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// What the model answers beyond the last whole row is not a row.
func TestAnAnswerBeyondTheLastWholeRowIsNotARow(t *testing.T) {
	values := []float32{0, 0.9, 10, 10, 100, 50, 1, 0, 0.9, 120, 10}
	layout, err := openLayoutOn(t, tableModel(outputBoxes, layoutSide, values), []string{"alpha"}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	defer layout.Close()

	got, err := layout.Regions(image.NewRGBA(image.Rect(0, 0, 200, 400)))
	if err != nil || len(got) != 1 {
		t.Errorf("regions = %+v, %v", got, err)
	}
}

func TestAnEmptyPageHasNoRegions(t *testing.T) {
	layout, err := openLayoutOn(t, tableModel(outputBoxes, layoutSide, []float32{0, 0.9, 10, 10, 100, 50, 1}), []string{"alpha"}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	defer layout.Close()

	got, err := layout.Regions(image.NewRGBA(image.Rectangle{}))
	if err != nil || got != nil {
		t.Errorf("regions = %+v, %v", got, err)
	}
}

// A model that answers under another name has not answered.
func TestAModelThatAnswersUnderAnotherNameHasNotAnswered(t *testing.T) {
	layout, err := openLayoutOn(t, tableModel("elsewhere", layoutSide, []float32{0, 0.9, 10, 10, 100, 50, 1}), []string{"alpha"}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	defer layout.Close()

	_, err = layout.Regions(image.NewRGBA(image.Rect(0, 0, 200, 400)))
	if err == nil || !strings.Contains(err.Error(), "elsewhere") || !strings.Contains(err.Error(), outputBoxes) {
		t.Errorf("answered with %v", err)
	}
}

// A model that cannot be run on the page is the page's failure.
func TestAModelThatCannotBeRunIsThePagesFailure(t *testing.T) {
	layout, err := openLayoutOn(t, tableModel(outputBoxes, 10, []float32{0, 0.9, 10, 10, 100, 50, 1}), []string{"alpha"}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	defer layout.Close()

	got, err := layout.Regions(image.NewRGBA(image.Rect(0, 0, 200, 400)))
	if err == nil || got != nil || strings.Contains(err.Error(), "answered with") {
		t.Errorf("regions = %+v, %v", got, err)
	}
}

func TestALayoutNeedsNamesForItsRegions(t *testing.T) {
	layout, err := openLayoutOn(t, echoModel([3]float32{}), nil, 0.5)
	if err == nil || layout != nil {
		t.Errorf("opened %v, %v", layout, err)
	}
}

// A file that is not a model is refused, naming the file and keeping the
// cause.
func TestAFileThatIsNotAModelIsRefused(t *testing.T) {
	layout, err := openLayoutOn(t, []byte("not a model"), []string{"alpha"}, 0.5)
	if err == nil || layout != nil {
		t.Fatalf("opened %v, %v", layout, err)
	}
	if !strings.Contains(err.Error(), "layout.onnx") {
		t.Errorf("error = %v, want it to name the file", err)
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("error = %v keeps no cause", err)
	}
}

func TestALayoutThatIsOpenCanBeClosed(t *testing.T) {
	layout, err := openLayoutOn(t, echoModel([3]float32{}), []string{"alpha"}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}

// The answer is given back under the names the graph gives it.
func TestNamesAreTheKeysOfAnAnswer(t *testing.T) {
	got := names(map[string]*ort.Value{"a": nil, "b": nil})
	if len(got) != 2 || !(got[0] == "a" && got[1] == "b" || got[0] == "b" && got[1] == "a") {
		t.Errorf("names = %v", got)
	}
}

// The slice that holds a page is kept for the next one.
func TestPlanesAreWrittenIntoTheSameSliceEachTime(t *testing.T) {
	l := &Layout{}
	page := image.NewRGBA(image.Rect(0, 0, 40, 40))
	first := l.planes(page)
	second := l.planes(page)
	if &first[0] != &second[0] {
		t.Error("a second page was written into a slice of its own")
	}
}
