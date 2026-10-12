package recognition

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// openConfig names three model files in dir and leaves the runtime to the
// machine.
func openConfig(t *testing.T, dir string, layout, detect, recognise []byte) Config {
	t.Helper()
	isolate(t)
	t.Setenv("TMPDIR", t.TempDir())
	cfg := Defaults()
	cfg.ShouldDownload = false
	if _, _, err := onnxruntime.Open(t.Context(), cfg.settings()); err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	cfg.Layout = LayoutModel{Path: writeModel(t, dir, "layout.onnx", layout)}
	cfg.Detect = DetectModel{Path: writeModel(t, dir, "detect.onnx", detect)}
	cfg.Recognise = RecogniserModel{Path: writeModel(t, dir, "rec.onnx", recognise)}
	return cfg
}

func TestOpeningWithoutALayoutModelThatLoadsFails(t *testing.T) {
	cfg := openConfig(t, t.TempDir(), []byte("not a model"), nil, nil)

	r, err := Open(t.Context(), cfg)
	if err == nil || r != nil {
		t.Fatalf("opened %v, %v", r, err)
	}
	if !strings.Contains(err.Error(), "layout.onnx") {
		t.Errorf("error = %v, want it to name the layout model", err)
	}
}

func TestOpeningWithoutAnAlphabetFails(t *testing.T) {
	cfg := openConfig(t, t.TempDir(), echoModel([3]float32{}), nil, echoModel([3]float32{}))
	cfg.Layout.Labels = []string{"alpha"}

	r, err := Open(t.Context(), cfg)
	if err == nil || r != nil {
		t.Fatalf("opened %v, %v", r, err)
	}
	if !strings.Contains(err.Error(), "carries no alphabet") {
		t.Errorf("error = %v", err)
	}
}

// The recognition library that cannot load the detector is the recogniser's
// failure, and its cause is kept.
func TestOpeningWithoutADetectorThatLoadsFails(t *testing.T) {
	recognise := onnxModel(pbString(2, "rec"), map[string]string{alphabetKey: "a\nb"})
	cfg := openConfig(t, t.TempDir(), echoModel([3]float32{}), []byte("not a model"), recognise)

	r, err := Open(t.Context(), cfg)
	if err == nil || r != nil {
		t.Fatalf("opened %v, %v", r, err)
	}
	if !strings.HasPrefix(err.Error(), "the recogniser: ") || errors.Unwrap(err) == nil {
		t.Errorf("error = %v", err)
	}
}

// Locating finds the runtime and says where each model is and where they came
// from.
func TestLocatingNamesTheModelsAndWhereTheyCameFrom(t *testing.T) {
	dir := t.TempDir()
	cfg := openConfig(t, dir, nil, nil, nil)

	opened, found, err := locateModels(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if opened.engine == nil || opened.at == "" {
		t.Errorf("runtime = %+v", opened)
	}
	want := paths{layout: cfg.Layout.Path, detect: cfg.Detect.Path, recognise: cfg.Recognise.Path, from: "settings"}
	if found != want {
		t.Errorf("paths = %+v, want %+v", found, want)
	}

	cfg.Dir = dir
	if _, found, err = locateModels(t.Context(), cfg); err != nil || found.from != dir {
		t.Errorf("from = %q, %v; want %q", found.from, err, dir)
	}
}

// A model that is not beside the application and is already fetched is the
// one that was fetched.
func TestAModelAlreadyFetchedIsTheOneFetched(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	address := "https://example.invalid/models/fetched.onnx"
	at := writeFile(t, dir, onnxruntime.GetCacheName(address))

	got, err := model(t.Context(), Config{Dir: dir}, "", address, "layout")
	if err != nil || got != at {
		t.Errorf("model = %q, %v; want %q", got, err, at)
	}
}

func TestAModelIsNamedByItsFileWithoutItsExtension(t *testing.T) {
	if got := name(filepath.Join("models", "PP-DocLayout.v3.onnx")); got != "PP-DocLayout.v3" {
		t.Errorf("name = %q", got)
	}
}

func TestARecogniserSaysWhichModelsItReadsWith(t *testing.T) {
	want := port.RecognitionModel{Layout: "layout", Recogniser: "rec", DPI: 300, From: "settings"}
	r := &Recogniser{model: want}
	if got := r.GetRecognitionModel(); got != want {
		t.Errorf("recognition = %+v, want %+v", got, want)
	}
}

func TestDepthsAreWhereANameStandsInTheList(t *testing.T) {
	got := getDepths([]string{"title", "section", "subsection"})
	want := map[string]int{"title": 0, "section": 1, "subsection": 2}
	if len(got) != len(want) {
		t.Fatalf("depths = %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("depth of %s = %d, want %d", k, got[k], v)
		}
	}
}

func TestASetHoldsEveryName(t *testing.T) {
	got := toSet([]string{"text", "title"})
	if len(got) != 2 || !got["text"] || !got["title"] || got["other"] {
		t.Errorf("set = %v", got)
	}
}
