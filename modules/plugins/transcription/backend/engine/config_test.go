package engine

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for name, want := range map[string]onnx.Provider{"": onnx.ProviderCPU, "auto": onnx.ProviderCPU, "cuda": onnx.ProviderCUDA} {
		t.Setenv("NUMEN_ONNX_PROVIDER", name)
		if got := getSessionProvider(); got != want {
			t.Errorf("with %q, getSessionProvider() = %q, want %q", name, got, want)
		}
	}
}
