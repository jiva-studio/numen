package recognition

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// Preparing makes the runtime out of nothing but the library. A machine holding
// every model and a machine holding none prepare alike, so the window is not
// held open by a model being read.
func TestPreparingMakesTheRuntimeWithoutAModel(t *testing.T) {
	cfg := Defaults()
	cfg.Download = false
	cfg.Layout.Name, cfg.Layout.Path = "", ""
	cfg.Detect.Name, cfg.Detect.Path = "", ""
	cfg.Recognise.Name, cfg.Recognise.Path = "", ""

	if _, _, err := onnxruntime.Open(t.Context(), cfg.settings()); err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	prepared.Store(false)
	t.Cleanup(func() { prepared.Store(false) })

	if err := Prepare(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	if !Prepared() {
		t.Error("the runtime a page is read through was not made")
	}
}
