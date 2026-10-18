package engine

import (
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/onnxruntime"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for name, want := range map[string]onnxruntime.Provider{"": onnxruntime.ProviderCPU, "auto": onnxruntime.ProviderCPU, "cuda": onnxruntime.ProviderCUDA} {
		t.Setenv("NUMEN_ONNX_PROVIDER", name)
		if got := getSessionProvider(); got != want {
			t.Errorf("with %q, getSessionProvider() = %q, want %q", name, got, want)
		}
	}
}
