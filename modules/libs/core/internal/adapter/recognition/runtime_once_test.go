package recognition

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// A recogniser that could not be built leaves the runtime where every other one
// finds it.
func TestLocatingLetsGoOfNoRuntime(t *testing.T) {
	cfg := Defaults()
	cfg.Download = false

	held, _, err := onnxruntime.Open(t.Context(), cfg.settings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	// A recogniser that could not be built gives back what it opened and
	// reaches for nothing else.
	if _, err := Open(t.Context(), Config{Runtime: cfg.Runtime, Download: false}); err == nil {
		t.Fatal("a recogniser was built with no models named")
	}

	again, _, err := onnxruntime.Open(t.Context(), cfg.settings())
	if err != nil {
		t.Fatal(err)
	}
	if again != held {
		t.Fatal("the runtime was let go of")
	}
	// The runtime that is there is one a session can be opened on.
	if _, err := again.NewSessionOptions(); err != nil {
		t.Errorf("the runtime that is left does not work: %v", err)
	}
}
