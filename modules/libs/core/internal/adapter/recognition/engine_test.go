package recognition

import (
	"reflect"
	"testing"

	"github.com/getcharzp/go-ocr/paddle"
	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// The binding stamps onto a tensor the engine that stood when it was made, and
// the reading library makes an engine of its own. A transcription that made a
// tensor and a reading opened after it are the two halves of that: this asks
// that the second does not move what the first was built through.
func TestATensorKeepsItsEngineWhenAReadingIsOpened(t *testing.T) {
	cfg := Defaults()
	cfg.ShouldDownload = false
	_, at, err := onnxruntime.Open(t.Context(), cfg.settings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}

	before, err := ort.NewTensor([]int64{1}, []float32{0})
	if err != nil {
		t.Fatal(err)
	}
	defer before.Destroy()

	// What opening a recogniser does with the runtime: the reading library is
	// asked for a reader, and makes its engine to answer.
	_, _ = paddle.NewEngine(paddle.Config{OnnxRuntimeLibPath: at})

	after, err := ort.NewTensor([]int64{1}, []float32{0})
	if err != nil {
		t.Fatal(err)
	}
	defer after.Destroy()

	if boundTo(before) != boundTo(after) {
		t.Errorf("a tensor made before the reading was opened runs on %x, and one made after on %x",
			boundTo(before), boundTo(after))
	}
}

// boundTo is the engine one tensor was made through. The binding keeps it to
// itself, and which one it is is the whole of what is being asked.
func boundTo(tensor *ort.Value) uintptr {
	return reflect.ValueOf(tensor).Elem().FieldByName("engine").Pointer()
}
