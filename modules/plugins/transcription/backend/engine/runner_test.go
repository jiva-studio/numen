package engine

import (
	"testing"
	"unsafe"

	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/onnxruntime"
)

// A status of zero reports no error.
func TestCheckStatusZeroIsNil(t *testing.T) {
	if err := checkStatus(0); err != nil {
		t.Errorf("checkStatus(0) = %v, want nil", err)
	}
}

func TestOrtAPITableLayout(t *testing.T) {
	var table ortAPITable
	if got := unsafe.Offsetof(table.Run); got != 72 {
		t.Errorf("Offsetof(Run) = %d, want 72", got)
	}
	if got := unsafe.Offsetof(table.GetTensorMutableData); got != 408 {
		t.Errorf("Offsetof(GetTensorMutableData) = %d, want 408", got)
	}
	if got := unsafe.Offsetof(table.ReleaseStatus); got != 744 {
		t.Errorf("Offsetof(ReleaseStatus) = %d, want 744", got)
	}
	if got := unsafe.Offsetof(table.ReleaseValue); got != 768 {
		t.Errorf("Offsetof(ReleaseValue) = %d, want 768", got)
	}
	if got := unsafe.Sizeof(table); got != 776 {
		t.Errorf("Sizeof(table) = %d, want 776", got)
	}
}

func TestRunnerRegistersHook(t *testing.T) {
	if !onnxruntime.HasRegistration() {
		t.Fatal("expected onnxruntime hook to be registered by init")
	}
}
