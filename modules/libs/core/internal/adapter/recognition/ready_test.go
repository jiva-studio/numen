package recognition

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile writes an empty file under dir and answers with where it is.
func writeFile(t *testing.T, dir, name string) string {
	t.Helper()
	at := filepath.Join(dir, name)
	if err := os.WriteFile(at, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return at
}

// Ready is answered by what is on this machine: a path written down is the
// file it names, a name is the file of that name in the models folder, and a
// name that is neither is not fetched.
func TestReadyIsAnsweredByTheModelsOnThisMachine(t *testing.T) {
	dir := t.TempDir()
	runtime := writeFile(t, dir, "libonnxruntime.so")
	layout := writeFile(t, dir, "layout.onnx")
	detect := writeFile(t, dir, "detect.onnx")
	writeFile(t, dir, "inference.onnx")

	for name, c := range map[string]struct {
		change  func(*Config)
		isReady bool
	}{
		"every model is here": {func(*Config) {}, true},
		"the download switch changes nothing": {
			func(cfg *Config) { cfg.ShouldDownload = true }, true,
		},
		"a path written down names nothing": {
			func(cfg *Config) { cfg.Layout.Path = filepath.Join(dir, "gone.onnx") }, false,
		},
		"a name is not in the models folder": {
			func(cfg *Config) { cfg.Recognise.Name = "https://example.invalid/models/elsewhere.onnx" }, false,
		},
		"a name is nowhere to fetch from": {
			func(cfg *Config) { cfg.Detect.Path, cfg.Detect.Name = "", "elsewhere.onnx" }, false,
		},
		"a model is not named at all": {
			func(cfg *Config) { cfg.Detect.Path = "" }, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Config{
				Runtime:   runtime,
				Dir:       dir,
				Layout:    LayoutModel{Path: layout},
				Detect:    DetectModel{Path: detect},
				Recognise: RecogniserModel{Name: "https://example.invalid/models/inference.onnx"},
			}
			c.change(&cfg)
			if got := Ready(cfg); got != c.isReady {
				t.Errorf("ready: %v", got)
			}
		})
	}
}

// Ready fetches nothing, whatever the configuration allows.
func TestReadyFetchesNothing(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	runtime := writeFile(t, dir, "libonnxruntime.so")
	layout := writeFile(t, dir, "layout.onnx")
	detect := writeFile(t, dir, "detect.onnx")

	var told []string
	cfg := Config{
		Runtime:        runtime,
		Dir:            dir,
		ShouldDownload: true,
		Layout:         LayoutModel{Path: layout},
		Detect:         DetectModel{Path: detect},
		Recognise:      RecogniserModel{Name: "https://127.0.0.1:1/models/inference.onnx"},
		Progress:       func(what string, _, _ int64) { told = append(told, what) },
	}

	if Ready(cfg) {
		t.Error("ready with a model that is not on this machine")
	}
	if len(told) != 0 {
		t.Errorf("a fetch of %v was begun", told)
	}
}

// Without a runtime nothing is ready, models or not.
func TestReadyWithoutARuntimeIsNotReady(t *testing.T) {
	if !isFreshProcess() {
		runInFreshProcess(t, "TestReadyWithoutARuntimeIsNotReady")
		return
	}
	dir := t.TempDir()
	cfg := Config{
		Runtime:   filepath.Join(dir, "missing-runtime.so"),
		Dir:       dir,
		Layout:    LayoutModel{Path: writeFile(t, dir, "layout.onnx")},
		Detect:    DetectModel{Path: writeFile(t, dir, "detect.onnx")},
		Recognise: RecogniserModel{Path: writeFile(t, dir, "inference.onnx")},
	}
	if Ready(cfg) {
		t.Error("ready with no runtime")
	}
}
